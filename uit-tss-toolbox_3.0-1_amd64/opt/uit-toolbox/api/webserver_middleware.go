package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
	"golang.org/x/time/rate"
)

type ctxClientIP struct{}
type ctxURLRequest struct{}
type ctxFileRequest struct {
	FullPath     string
	ResolvedPath string
	FileName     string
}

func limitRequestSizeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		maxSize := int64(64 << 20)
		if req.ContentLength > maxSize {
			//req.RemoteAddr used here because the ip has not been assigned to the context yet
			log.Warning("Request content length exceeds limit: " + fmt.Sprintf("%.2fMB", float64(req.ContentLength)/1e6) + " " + req.RemoteAddr)
			http.Error(w, formatHttpError("Request too large"), http.StatusRequestEntityTooLarge)
			return
		}
		req.Body = http.MaxBytesReader(w, req.Body, maxSize)
		next.ServeHTTP(w, req)
	})
}

func timeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
		defer cancel()
		req = req.WithContext(ctx)
		next.ServeHTTP(w, req)
	})
}

func storeClientIPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// xffHeader := strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]
		// xripHeader := strings.TrimSpace(r.Header.Get("X-Real-IP"))
		// xffExists := xffHeader != ""
		// xripExists := xripHeader != ""

		ip, port, err := net.SplitHostPort(req.RemoteAddr)
		if err != nil {
			log.Warning("Could not parse IP address: " + err.Error())
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(port) == "" {
			log.Warning("Empty port in request")
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		ipValid, _, _ := checkValidIP(ip)
		if !ipValid {
			log.Warning("Invalid IP address, terminating connection")
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(req.Context(), ctxClientIP{}, ip)
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func allowIPRangeMiddleware(acceptedCIDRs []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			requestIP, ok := GetRequestIP(req)
			if !ok {
				log.Warning("no IP address stored in context")
				http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
				return
			}
			parsedRequestIP := net.ParseIP(requestIP)
			if parsedRequestIP == nil {
				log.Warning("cannot parse request IP")
				http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
				return
			}
			allowed := false
			for _, cidr := range acceptedCIDRs {
				_, ipNet, err := net.ParseCIDR(cidr)
				if err == nil && ipNet.Contains(parsedRequestIP) {
					allowed = true
					break
				}
			}
			if !allowed {
				log.Warning("IP address not in allowed range: " + requestIP)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}

func checkValidURLMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("No IP address stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		// URL length
		if len(req.URL.RequestURI()) > 2048 {
			log.Warning("Request URL length exceeds limit: " + fmt.Sprintf("%d", len(req.URL.RequestURI())) + " (" + requestIP + ": " + req.Method + " " + req.URL.RequestURI() + ")")
			http.Error(w, formatHttpError("Request URI too long"), http.StatusRequestURITooLong)
			return
		}

		// URL query
		if strings.ContainsAny(req.URL.RawQuery, "<>\"'%;()+") {
			log.Warning("Invalid characters in query parameters: " + "( " + requestIP + ": " + req.Method + " " + req.URL.RequestURI() + ")")
			http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
			return
		}

		// Check URL path
		var disallowedPathChars = "~%$#\\<>:\"'`|?*\x00\r\n"
		// Get unescaped path (decode URL) & normalize UTF-8
		rawPath := strings.TrimSpace(req.URL.Path)
		if rawPath == "" || len(rawPath) > 255 {
			log.Warning("Empty URL requested from: " + requestIP)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		unescapedPath, err := url.PathUnescape(rawPath)
		if err != nil {
			log.Warning("Cannot unescape URL path: " + err.Error())
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		normalizedPath := norm.NFC.String(unescapedPath)
		if !utf8.ValidString(normalizedPath) ||
			!path.IsAbs(normalizedPath) ||
			strings.Contains(normalizedPath, "..") ||
			strings.Contains(normalizedPath, "//") ||
			strings.HasPrefix(normalizedPath, ".") ||
			strings.HasSuffix(normalizedPath, ".") ||
			strings.ContainsAny(normalizedPath, disallowedPathChars) {
			log.Warning("Normalized URL path is invalid: " + requestIP)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Clean entire path and format the URL path
		fullPath := path.Clean(normalizedPath)
		if !path.IsAbs(fullPath) ||
			strings.TrimSpace(fullPath) == "" ||
			strings.Contains(fullPath, "..") ||
			strings.Contains(fullPath, "../") ||
			fullPath == "/" ||
			fullPath == "." ||
			fullPath == "" {

			log.Warning("Empty file path requested: " + requestIP)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Split URL path into path + file name
		_, fileRequested := path.Split(fullPath)
		if strings.HasPrefix(fileRequested, ".") ||
			strings.HasPrefix(fileRequested, "~") ||
			strings.HasSuffix(fileRequested, ".tmp") ||
			strings.HasSuffix(fileRequested, ".bak") ||
			strings.HasSuffix(fileRequested, ".swp") {

			log.Warning("Invalid characters in file requested")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		pathSegments := strings.Split(strings.Trim(fullPath, "/"), "/")
		for _, segment := range pathSegments {
			if segment == "" {
				continue
			}

			// Check valid ASCII & UTF-8
			for _, char := range fullPath {
				if char < 32 || char == 127 {
					log.Warning("Control/non-printable character in filename: " + requestIP)
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
				if char > 127 || char > unicode.MaxASCII || char > unicode.MaxLatin1 {
					log.Warning("Non-ASCII character in filename: " + requestIP)
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
				// if !(unicode.IsPrint(char) ||
				// 	unicode.Isletter(char) ||
				// 	unicode.isNumber(char) ||
				// 	unicode.IsDigit(char)) ||
				// 	!unicode.isSpace(char)
				// 	!unicode.IsControl(char)

				if !unicode.In(char, unicode.Digit, unicode.Letter, unicode.Mark, unicode.Number, unicode.Punct, unicode.Space) {
					log.Warning("Invalid Unicode Char")
					http.Error(w, formatHttpError("Forbidden"), http.StatusForbidden)
					return
				}

				if strings.ContainsRune(disallowedPathChars, char) {
					log.Warning("Disallowed character in filename: " + requestIP)
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
			}
		}

		var ctx context.Context
		if strings.TrimSpace(req.URL.RawQuery) == "" {
			ctx = context.WithValue(req.Context(), ctxURLRequest{}, fullPath)
		} else {
			ctx = context.WithValue(req.Context(), ctxURLRequest{}, fullPath+"?"+req.URL.RawQuery)
		}
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func rateLimitMiddleware(appState *AppState, rateType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			requestIP, ok := GetRequestIP(req)
			if !ok {
				log.Warning("no IP address stored in context")
				http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
				return
			}

			if appState.blockedIPs.IsBlocked(requestIP) {
				log.Debug("Blocked IP attempted request: " + requestIP)
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			var limiter *rate.Limiter
			switch rateType {
			case "webserver":
				limiter = appState.webServerLimiter.Get(requestIP)
			case "file":
				limiter = appState.fileLimiter.Get(requestIP)
			case "api":
				limiter = appState.apiLimiter.Get(requestIP)
			case "auth":
				limiter = appState.authLimiter.Get(requestIP)
			default:
				limiter = appState.webServerLimiter.Get(requestIP)
			}
			if !limiter.Allow() {
				appState.blockedIPs.Block(requestIP)
				log.Debug("Client has exceeded rate limit: " + requestIP)
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, req)
		})
	}
}

func allowedFilesMiddleware(appState *AppState) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			requestIP, ok := GetRequestIP(req)
			if !ok {
				log.Warning("no IP address stored in context")
				http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
				return
			}
			requestURL, ok := GetRequestURL(req)
			if !ok {
				log.Warning("no URL stored in context")
				http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
				return
			}

			var basePath string
			if strings.HasPrefix(requestURL, "/client/") {
				basePath = "/srv/uit-toolbox/"
			} else {
				basePath = "/var/www/html/uit-web/"
			}

			fullPath := path.Join(basePath, requestURL)
			_, fileRequested := path.Split(fullPath)

			if len(appState.allowedFiles) > 0 && !appState.allowedFiles[fileRequested] {
				log.Warning("File not in whitelist: " + fileRequested)
				http.Error(w, formatHttpError("Forbidden"), http.StatusForbidden)
				return
			}

			resolvedPath, err := filepath.EvalSymlinks(fullPath)
			if err != nil || !strings.HasPrefix(resolvedPath, basePath) {
				log.Warning("File request error from " + requestIP + " (" + resolvedPath + "): Error resolving symlink: " + err.Error())
				http.Error(w, formatHttpError("Forbidden"), http.StatusForbidden)
				return
			}

			if resolvedPath != fullPath {
				log.Warning("Resolved path does not match full path (" + requestIP + "): " + resolvedPath + " -> " + fullPath)
				http.Error(w, formatHttpError("Forbidden"), http.StatusForbidden)
				return
			}

			metadata, err := os.Lstat(fullPath)
			if err != nil {
				log.Error("Cannot get metadata from file: " + fullPath + " (" + err.Error() + ")")
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			if metadata == nil {
				log.Error("Metadata is nil for file: " + fullPath)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			if metadata.Name() != fileRequested || !appState.allowedFiles[metadata.Name()] {
				log.Warning("Filename mismatch: " + metadata.Name() + " != " + fileRequested)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			if metadata.Size() <= 0 {
				log.Warning("Attempt to access empty file: " + fileRequested)
				http.Error(w, "Empty file", http.StatusNoContent)
				return
			}
			if metadata.IsDir() {
				log.Warning("Attempt to access directory as file: " + fileRequested)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			fileMode := metadata.Mode()
			if fileMode&os.ModeSymlink != 0 {
				log.Warning("Attempt to access symbolic link: " + fileRequested)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			if fileMode&os.ModeDevice != 0 {
				log.Warning("Attempt to access device file: " + fileRequested)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			if fileMode&os.ModeNamedPipe != 0 {
				log.Warning("Attempt to access named pipe: " + fileRequested)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			if fileMode&os.ModeSocket != 0 {
				log.Warning("Attempt to access socket file: " + fileRequested)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			if fileMode&os.ModeCharDevice != 0 {
				log.Warning("Attempt to access character device file: " + fileRequested)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			if fileMode&os.ModeIrregular != 0 {
				log.Warning("Attempt to access irregular file: " + fileRequested)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			if !fileMode.IsRegular() {
				log.Warning("Attempt to access non-regular file: " + fileRequested)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			fileRequest := ctxFileRequest{
				FullPath:     fullPath,
				ResolvedPath: resolvedPath,
				FileName:     fileRequested,
			}
			ctxWithFile := context.WithValue(req.Context(), ctxFileRequest{}, fileRequest)
			next.ServeHTTP(w, req.WithContext(ctxWithFile))
		})
	}
}

func tlsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("no IP address stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		if req.TLS == nil || !req.TLS.HandshakeComplete {
			log.Warning("TLS handshake failed for client " + requestIP)
			http.Error(w, formatHttpError("TLS required"), http.StatusUpgradeRequired)
			return
		}

		if req.TLS.Version < tls.VersionTLS13 {
			log.Warning("Rejected connection with weak TLS version from " + requestIP)
			http.Error(w, formatHttpError("TLS version too low"), http.StatusUpgradeRequired)
			return
		}

		weakCiphers := map[uint16]bool{
			tls.TLS_RSA_WITH_RC4_128_SHA:                true,
			tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA:           true,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA256:         true,
			tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA:        true,
			tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA:          true,
			tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA:     true,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256: true,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256:   true,
		}
		if weakCiphers[req.TLS.CipherSuite] {
			log.Warning("Rejected connection with weak cipher suite from " + requestIP)
			http.Error(w, formatHttpError("Weak cipher suite not allowed"), http.StatusUpgradeRequired)
			return
		}

		next.ServeHTTP(w, req)
	})
}

func httpMethodMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Get IP address
		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("no IP address stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		requestURL, ok := GetRequestURL(req)
		if !ok {
			log.Warning("no URL stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		// Check method
		validMethods := map[string]bool{
			http.MethodOptions: true,
			http.MethodGet:     true,
			http.MethodPost:    true,
			http.MethodPut:     true,
			http.MethodDelete:  true,
		}
		if !validMethods[req.Method] {
			log.Warning("Invalid request method (" + requestIP + "): " + req.Method)
			http.Error(w, formatHttpError("Method not allowed"), http.StatusMethodNotAllowed)
			return
		}

		// Check Content-Type for POST/PUT
		if req.Method == http.MethodPost || req.Method == http.MethodPut {
			contentType := req.Header.Get("Content-Type")
			if contentType != "application/x-www-form-urlencoded" && contentType != "application/json" {
				log.Warning("Invalid Content-Type header: " + contentType + " (" + requestIP + ": " + req.Method + " " + requestURL + ")")
				http.Error(w, formatHttpError("Invalid content type"), http.StatusUnsupportedMediaType)
				return
			}
		}
		next.ServeHTTP(w, req)
	})
}

func checkHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("no IP address stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		requestURL, ok := GetRequestURL(req)
		if !ok {
			log.Warning("no URL stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		// Content length
		if req.ContentLength > 64<<20 {
			log.Warning("Request content length exceeds limit: " + fmt.Sprintf("%.2fMB", float64(req.ContentLength)/1e6))
			http.Error(w, formatHttpError("Request too large"), http.StatusRequestEntityTooLarge)
			return
		}

		// Origin header
		origin := req.Header.Get("Origin")
		if origin != "" && len(origin) > 2048 {
			log.Warning("Invalid Origin header: " + origin + " (" + requestIP + ": " + req.Method + " " + requestURL + ")")
			http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
			return
		}

		// Host header
		host := strings.TrimSpace(req.Host)
		if host == "" || strings.ContainsAny(host, " <>\"'%;()&+") || len(host) > 255 {
			log.Warning("Invalid Host header: " + host + " (" + requestIP + ": " + req.Method + " " + requestURL + ")")
			http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
			return
		}

		// User-Agent header
		userAgent := req.Header.Get("User-Agent")
		if userAgent == "" || len(userAgent) > 256 {
			log.Warning("Invalid User-Agent header: " + userAgent + " (" + requestIP + ": " + req.Method + " " + requestURL + ")")
			http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
			return
		}

		// Referer header
		referer := req.Header.Get("Referer")
		if referer != "" && len(referer) > 2048 {
			log.Warning("Invalid Referer header: " + referer + " (" + requestIP + ": " + req.Method + " " + requestURL + ")")
			http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
			return
		}

		// Other headers
		// for key, value := range req.Header {
		//   if strings.ContainsAny(key, "<>\"'%;()&+") || strings.ContainsAny(value[0], "<>\"'%;()&+") {
		//     log.Warning("Invalid characters in header '" + key + "': " + value[0] + " (" + requestIP + ": " + req.Method + " " + requestURL + ")")
		//     http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
		//     return
		//   }
		// }

		next.ServeHTTP(w, req)
	})
}

func setHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("no IP address stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		requestURL, ok := GetRequestURL(req)
		if !ok {
			log.Warning("no URL stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		// Get env vars
		env := configureEnvironment()
		webServerIP := env.UIT_WAN_IP_ADDRESS
		// Check CORS policy
		cors := http.NewCrossOriginProtection()
		cors.AddTrustedOrigin("https://" + webServerIP + ":1411")
		if err := cors.Check(req); err != nil {
			log.Warning("Request to " + requestURL + " blocked from " + requestIP)
			http.Error(w, formatHttpError("CORS policy violation"), http.StatusForbidden)
			return
		}

		// Handle OPTIONS early
		if req.Method == http.MethodOptions {
			// Headers for OPTIONS request
			w.Header().Set("Access-Control-Allow-Origin", "https://"+webServerIP+":1411")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Set-Cookie, credentials")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "https://"+webServerIP+":1411")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Strict-Transport-Security", "max-age=86400; includeSubDomains")
		w.Header().Set("X-Accel-Buffering", "no")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Server", "")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Deprecated headers
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("X-Download-Options", "noopen")
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")

		next.ServeHTTP(w, req)
	})
}

func apiAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var requestBasicToken string
		var requestBearerToken string
		var sessionCount int = 0

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("no IP address stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		requestURL, ok := GetRequestURL(req)
		if !ok {
			log.Warning("no URL stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		// Delete expired tokens & malformed entries out of authMap
		authMap.Range(func(k, v any) bool {
			sessionID := k.(string)
			authSession := v.(AuthSession)
			sessionIP := strings.SplitN(sessionID, ":", 2)[0]

			// basicExpiry := authSession.Basic.Expiry.Sub(time.Now())
			bearerExpiry := time.Until(authSession.Bearer.Expiry)

			// Auth cache entry expires once countdown reaches zero
			if bearerExpiry.Seconds() <= 0 {
				authMap.Delete(sessionID)
				sessionCount = countAuthSessions(&authMap)
				log.Info("Auth session expired: " + sessionIP + " (TTL: " + fmt.Sprintf("%.2f", bearerExpiry.Seconds()) + ", " + strconv.Itoa(int(sessionCount)) + " session(s))")
			}
			return true
		})

		queryType := strings.TrimSpace(req.URL.Query().Get("type"))
		// if strings.TrimSpace(queryType) == "" {
		// 	log.Warning("No query type defined for request: " + requestIP)
		// 	http.Error(w, formatHttpError("Bad request"), http.StatusBadRequest)
		// 	return
		// }

		headers := ParseHeaders(req.Header)
		if headers.Authorization.Basic != nil {
			requestBasicToken = *headers.Authorization.Basic
		}
		if headers.Authorization.Bearer != nil {
			requestBearerToken = *headers.Authorization.Bearer
		}

		if strings.TrimSpace(requestBearerToken) == "" && strings.TrimSpace(requestBasicToken) == "" {
			log.Warning("Empty value for Authorization header: " + requestIP + " ( " + requestURL + ")")
			http.Error(w, formatHttpError("Unauthorized"), http.StatusUnauthorized)
			return
		}

		basicValid, bearerValid, _, bearerTTL, matchedSession := checkAuthSession(&authMap, requestIP, requestBasicToken, requestBearerToken)

		if (basicValid && bearerValid) || bearerValid {
			if strings.TrimSpace(queryType) == "check-token" {
				jsonData, err := json.Marshal(returnedJsonToken{
					Token: matchedSession.Bearer.Token,
					TTL:   bearerTTL,
					Valid: true,
				})
				if err != nil {
					log.Error("Cannot marshal Token to JSON: " + err.Error())
					return
				}
				w.Write(jsonData)
				return
			} else if strings.TrimSpace(queryType) != "" {
				next.ServeHTTP(w, req)
			}
		} else if (basicValid && !bearerValid) || (!basicValid && !bearerValid) {
			sessionCount = countAuthSessions(&authMap)
			log.Debug("Auth cache miss: " + requestIP + " (Sessions: " + strconv.Itoa(int(sessionCount)) + ") " + requestURL)
			if queryType == "new-token" && strings.TrimSpace(requestBasicToken) != "" {
				next.ServeHTTP(w, req)
			} else {
				http.Error(w, formatHttpError("Unauthorized"), http.StatusUnauthorized)
				return
			}
		} else {
			log.Warning("No valid authentication found for request: " + requestIP + " ( " + requestURL + ")")
			http.Error(w, formatHttpError("Unauthorized"), http.StatusUnauthorized)
			return
		}
	})
}

func httpCookieAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("no IP address stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		requestURL, ok := GetRequestURL(req)
		if !ok {
			log.Warning("no URL stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		basicCookie, errBasic := req.Cookie("uit_basic_token")

		if errBasic != nil || strings.TrimSpace(basicCookie.Value) == "" {
			log.Warning("Missing basic auth cookie")
			http.Error(w, formatHttpError("Unauthorized"), http.StatusUnauthorized)
			return
		}

		requestBasicToken := basicCookie.Value

		// Clean up expired tokens
		var sessionCount int
		authMap.Range(func(k, v any) bool {
			sessionID := k.(string)
			authSession := v.(AuthSession)
			sessionIP := strings.SplitN(sessionID, ":", 2)[0]
			bearerExpiry := time.Until(authSession.Bearer.Expiry)
			if bearerExpiry.Seconds() <= 0 {
				authMap.Delete(sessionID)
				sessionCount = countAuthSessions(&authMap)
				log.Info("Auth session expired: " + sessionIP + " (TTL: " + fmt.Sprintf("%.2f", bearerExpiry.Seconds()) + ", " + strconv.Itoa(int(sessionCount)) + " session(s))")
			}
			return true
		})

		// Check session using the cookie value as bearer token
		basicValid, _, _, _, _ := checkAuthSession(&authMap, requestIP, requestBasicToken, "")

		if basicValid && !strings.HasSuffix(requestURL, "/logout") {
			http.SetCookie(w, &http.Cookie{
				Name:     "uit_basic_token",
				Value:    requestBasicToken,
				Path:     "/",
				Expires:  time.Now().Add(20 * time.Minute),
				MaxAge:   20 * 60,
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})
			basicExpiry := time.Now().Add(20 * time.Minute)
			bearerExpiry := time.Now().Add(10 * time.Minute)
			basic := BasicToken{Token: requestBasicToken, Expiry: basicExpiry, NotBefore: time.Now(), TTL: time.Until(basicExpiry).Seconds(), IP: requestIP, Valid: true}
			bearer := BearerToken{Token: requestBasicToken, Expiry: bearerExpiry, NotBefore: time.Now(), TTL: time.Until(bearerExpiry).Seconds(), IP: requestIP, Valid: true}

			authSession := AuthSession{
				Basic:  basic,
				Bearer: bearer,
			}
			sessionID := requestIP + ":" + requestBasicToken
			newSessionHash, err := generateSecureSessionID()
			if err != nil {
				log.Error("Failed to generate secure session ID: " + err.Error())
				http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
				return
			}
			newSessionID := requestIP + ":" + newSessionHash
			if sessionID != newSessionID {
				authMap.Delete(sessionID)
			}
			authMap.Store(newSessionID, authSession)
			log.Debug("Auth session extended: " + requestIP)
			next.ServeHTTP(w, req)
			return
		} else {
			sessionCount = countAuthSessions(&authMap)
			log.Debug("Auth cookie cache miss: " + requestIP + " (Sessions: " + strconv.Itoa(int(sessionCount)) + ") " + requestURL)
			http.Error(w, formatHttpError("Unauthorized"), http.StatusUnauthorized)
			return
		}
	})
}
