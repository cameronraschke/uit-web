package middleware

import (
	"context"
	"crypto/tls"
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

	config "uit-toolbox/config"

	"golang.org/x/text/unicode/norm"
)

func LimitRequestSizeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		maxSize := int64(64 << 20)
		if req.ContentLength > maxSize {
			//req.RemoteAddr used here because the ip has not been assigned to the context yet
			log.Warning("Request content length exceeds limit: " + fmt.Sprintf("%.2fMB", float64(req.ContentLength)/1e6) + " " + req.RemoteAddr)
			http.Error(w, FormatHttpError("Request too large"), http.StatusRequestEntityTooLarge)
			return
		}
		req.Body = http.MaxBytesReader(w, req.Body, maxSize)
		next.ServeHTTP(w, req)
	})
}

func TimeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
		defer cancel()
		req = req.WithContext(ctx)
		next.ServeHTTP(w, req)
	})
}

func StoreClientIPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()

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

		ctx := context.WithValue(req.Context(), CTXClientIP{}, ip)
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func AllowIPRangeMiddleware(source string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			log := config.GetLogger()
			if strings.TrimSpace(source) == "" {
				log.Warning("No source specified for AllowIPRangeMiddleware")
				http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
				return
			}
			requestIP, ok := GetRequestIP(req)
			if !ok {
				log.Warning("no IP address stored in context")
				http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
				return
			}
			allowed := config.IsIPAllowed(source, requestIP)
			if !allowed {
				log.Warning("IP address not in allowed range: " + requestIP)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}

func CheckValidURLMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("No IP address stored in context")
			http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		// URL length
		if len(req.URL.RequestURI()) > 2048 {
			log.Warning("Request URL length exceeds limit: " + fmt.Sprintf("%d", len(req.URL.RequestURI())) + " (" + requestIP + ": " + req.Method + " " + req.URL.RequestURI() + ")")
			http.Error(w, FormatHttpError("Request URI too long"), http.StatusRequestURITooLong)
			return
		}

		// URL query
		queryValues := req.URL.Query()
		disallowedQueryChars := "~%$#\\<>:\"'`|?*\x00\r\n"
		for k, v := range queryValues {
			if len(k) > 128 || len(v) > 512 {
				log.Warning("Request URL query key length exceeds limit: " + fmt.Sprintf("%d", len(k)) + " (" + requestIP + ": " + req.Method + " " + req.URL.RequestURI() + ")")
				http.Error(w, FormatHttpError("Request URI too long"), http.StatusRequestURITooLong)
				return
			}
			if strings.ContainsAny(k, disallowedQueryChars) {
				log.Warning("Invalid characters in query key: " + "( " + requestIP + ": " + req.Method + " " + req.URL.RequestURI() + ")")
				http.Error(w, FormatHttpError("Bad request"), http.StatusBadRequest)
				return
			}
			for _, value := range v {
				if strings.ContainsAny(value, disallowedQueryChars) {
					log.Warning("Invalid characters in query value: " + "( " + requestIP + ": " + req.Method + " " + req.URL.RequestURI() + ")")
					http.Error(w, FormatHttpError("Bad request"), http.StatusBadRequest)
					return
				}
			}

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
					http.Error(w, FormatHttpError("Forbidden"), http.StatusForbidden)
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
			ctx = context.WithValue(req.Context(), CTXURLRequest{}, fullPath)
		} else {
			parsedQuery, err := url.ParseQuery(req.URL.RawQuery)
			if err != nil {
				log.Warning("Failed to parse URL query: " + requestIP)
				http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
				return
			}
			newURL := url.URL{
				Path:     fullPath,
				RawQuery: parsedQuery.Encode(),
			}
			ctx = context.WithValue(req.Context(), CTXURLRequest{}, newURL.RequestURI())
		}
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func RateLimitMiddleware(appState *config.AppState, rateType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			log := config.GetLogger()
			requestIP, ok := GetRequestIP(req)
			if !ok {
				log.Warning("no IP address stored in context")
				http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
				return
			}

			if config.IsIPBlocked(requestIP) {
				log.Debug("Blocked IP attempted request: " + requestIP)
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			limited, retryAfter := config.IsClientRateLimited(rateType, requestIP)
			if limited {
				log.Debug("Client is rate limited: " + requestIP + " (retry after " + retryAfter.String() + ")")
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, req)
		})
	}
}

func AllowedFilesMiddleware(appState *config.AppState) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			log := config.GetLogger()
			requestIP, ok := GetRequestIP(req)
			if !ok {
				log.Warning("no IP address stored in context")
				http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
				return
			}
			requestURL, ok := GetRequestURL(req)
			if !ok {
				log.Warning("no URL stored in context")
				http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
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

			if !config.IsFileAllowed(fileRequested) {
				log.Warning("File not in whitelist: " + fileRequested)
				http.Error(w, FormatHttpError("Forbidden"), http.StatusForbidden)
				return
			}

			resolvedPath, err := filepath.EvalSymlinks(fullPath)
			if err != nil || !strings.HasPrefix(resolvedPath, basePath) {
				log.Warning("File request error from " + requestIP + " (" + resolvedPath + "): Error resolving symlink: " + err.Error())
				http.Error(w, FormatHttpError("Forbidden"), http.StatusForbidden)
				return
			}

			if resolvedPath != fullPath {
				log.Warning("Resolved path does not match full path (" + requestIP + "): " + resolvedPath + " -> " + fullPath)
				http.Error(w, FormatHttpError("Forbidden"), http.StatusForbidden)
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
			if metadata.Name() != fileRequested || !config.IsFileAllowed(metadata.Name()) {
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

			fileRequest := CTXFileRequest{
				FullPath:     fullPath,
				ResolvedPath: resolvedPath,
				FileName:     fileRequested,
			}
			ctxWithFile := context.WithValue(req.Context(), CTXFileRequest{}, fileRequest)
			next.ServeHTTP(w, req.WithContext(ctxWithFile))
		})
	}
}

func TLSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("no IP address stored in context")
			http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		if req.TLS == nil || !req.TLS.HandshakeComplete {
			log.Warning("TLS handshake failed for client " + requestIP)
			http.Error(w, FormatHttpError("TLS required"), http.StatusUpgradeRequired)
			return
		}

		if req.TLS.Version < tls.VersionTLS13 {
			log.Warning("Rejected connection with weak TLS version from " + requestIP)
			http.Error(w, FormatHttpError("TLS version too low"), http.StatusUpgradeRequired)
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
			http.Error(w, FormatHttpError("Weak cipher suite not allowed"), http.StatusUpgradeRequired)
			return
		}

		next.ServeHTTP(w, req)
	})
}

func HTTPMethodMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		// Get IP address
		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("no IP address stored in context")
			http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		requestURL, ok := GetRequestURL(req)
		if !ok {
			log.Warning("no URL stored in context")
			http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
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
			http.Error(w, FormatHttpError("Method not allowed"), http.StatusMethodNotAllowed)
			return
		}

		// Check Content-Type for POST/PUT
		if req.Method == http.MethodPost || req.Method == http.MethodPut {
			contentType := req.Header.Get("Content-Type")
			if contentType != "application/x-www-form-urlencoded" && contentType != "application/json" {
				log.Warning("Invalid Content-Type header: " + contentType + " (" + requestIP + ": " + req.Method + " " + requestURL + ")")
				http.Error(w, FormatHttpError("Invalid content type"), http.StatusUnsupportedMediaType)
				return
			}
		}
		next.ServeHTTP(w, req)
	})
}

func CheckHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("no IP address stored in context")
			http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		requestURL, ok := GetRequestURL(req)
		if !ok {
			log.Warning("no URL stored in context")
			http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		// Content length
		if req.ContentLength > 64<<20 {
			log.Warning("Request content length exceeds limit: " + fmt.Sprintf("%.2fMB", float64(req.ContentLength)/1e6))
			http.Error(w, FormatHttpError("Request too large"), http.StatusRequestEntityTooLarge)
			return
		}

		// Origin header
		origin := req.Header.Get("Origin")
		if origin != "" && len(origin) > 2048 {
			log.Warning("Invalid Origin header: " + origin + " (" + requestIP + ": " + req.Method + " " + requestURL + ")")
			http.Error(w, FormatHttpError("Bad request"), http.StatusBadRequest)
			return
		}

		// Host header
		host := strings.TrimSpace(req.Host)
		if host == "" || strings.ContainsAny(host, " <>\"'%;()&+") || len(host) > 255 {
			log.Warning("Invalid Host header: " + host + " (" + requestIP + ": " + req.Method + " " + requestURL + ")")
			http.Error(w, FormatHttpError("Bad request"), http.StatusBadRequest)
			return
		}

		// User-Agent header
		userAgent := req.Header.Get("User-Agent")
		if userAgent == "" || len(userAgent) > 256 {
			log.Warning("Invalid User-Agent header: " + userAgent + " (" + requestIP + ": " + req.Method + " " + requestURL + ")")
			http.Error(w, FormatHttpError("Bad request"), http.StatusBadRequest)
			return
		}

		// Referer header
		referer := req.Header.Get("Referer")
		if referer != "" && len(referer) > 2048 {
			log.Warning("Invalid Referer header: " + referer + " (" + requestIP + ": " + req.Method + " " + requestURL + ")")
			http.Error(w, FormatHttpError("Bad request"), http.StatusBadRequest)
			return
		}

		// Other headers
		// for key, value := range req.Header {
		//   if strings.ContainsAny(key, "<>\"'%;()&+") || strings.ContainsAny(value[0], "<>\"'%;()&+") {
		//     log.Warning("Invalid characters in header '" + key + "': " + value[0] + " (" + requestIP + ": " + req.Method + " " + requestURL + ")")
		//     http.Error(w, FormatHttpError("Bad request"), http.StatusBadRequest)
		//     return
		//   }
		// }

		next.ServeHTTP(w, req)
	})
}

func SetHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("no IP address stored in context")
			http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		requestURL, ok := GetRequestURL(req)
		if !ok {
			log.Warning("no URL stored in context")
			http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		// Get web server IP for CORS
		_, webserverWanIP, err := config.GetWebServerIP()
		if err != nil || strings.TrimSpace(webserverWanIP) == "" {
			log.Error("Cannot get web server IP for CORS: " + err.Error())
			http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		// Check CORS policy
		cors := http.NewCrossOriginProtection()
		cors.AddTrustedOrigin("https://" + webserverWanIP + ":1411")
		if err := cors.Check(req); err != nil {
			log.Warning("Request to " + requestURL + " blocked from " + requestIP)
			http.Error(w, FormatHttpError("CORS policy violation"), http.StatusForbidden)
			return
		}

		// Handle OPTIONS early
		if req.Method == http.MethodOptions {
			// Headers for OPTIONS request
			w.Header().Set("Access-Control-Allow-Origin", "https://"+webserverWanIP+":1411")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Set-Cookie, credentials")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "https://"+webserverWanIP+":1411")
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

func APIAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()

		// Request variables
		var requestBasicToken string
		var requestBearerToken string
		var requestCSRFToken string
		var sessionID string
		var sessionCount int64 = 0

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("no IP address stored in context")
			http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		requestURL, ok := GetRequestURL(req)
		if !ok {
			log.Warning("no URL stored in context")
			http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		queryType := strings.TrimSpace(req.URL.Query().Get("type"))
		// if strings.TrimSpace(queryType) == "" {
		// 	log.Warning("No query type defined for request: " + requestIP)
		// 	http.Error(w, FormatHttpError("Bad request"), http.StatusBadRequest)
		// 	return
		// }

		authHeaders := req.Header.Values("Authorization")
		if len(authHeaders) == 0 {
			log.Warning("No Authorization header provided: " + requestIP + " ( " + requestURL + ")")
			http.Error(w, FormatHttpError("Unauthorized"), http.StatusUnauthorized)
			return
		}
		for _, header := range authHeaders {
			// Get position (index) of space in auth header
			spaceIndex := strings.IndexByte(header, ' ')
			if spaceIndex <= 0 || spaceIndex == len(header)-1 {
				continue
			}
			headerType := header[:spaceIndex]
			token := strings.TrimSpace(header[spaceIndex+1:])
			if token == "" {
				continue
			}
			switch strings.ToLower(headerType) {
			case "basic":
				if requestBasicToken == "" {
					requestBasicToken = token
				}
			case "bearer":
				if requestBearerToken == "" {
					requestBearerToken = token
				}
			}
			if requestBasicToken != "" && requestBearerToken != "" {
				break
			}
		}
		if strings.TrimSpace(requestBearerToken) == "" || strings.TrimSpace(requestBasicToken) == "" {
			log.Warning("Empty value for Authorization header: " + requestIP + " ( " + requestURL + ")")
			http.Error(w, FormatHttpError("Unauthorized"), http.StatusUnauthorized)
			return
		}

		// No consequences for having missing CSRF (yet)
		requestCSRFToken = strings.TrimSpace(req.Header.Get("X-CSRF-Token"))

		sessionID = strings.TrimSpace(req.Header.Get("X-Session-ID"))
		if strings.TrimSpace(sessionID) == "" {
			log.Warning("No session ID provided: " + requestIP + " ( " + requestURL + ")")
			http.Error(w, FormatHttpError("Unauthorized"), http.StatusUnauthorized)
			return
		}

		sessionValid, sessionExists, err := config.CheckAuthSessionExists(sessionID, requestIP, requestBasicToken, requestBearerToken, requestCSRFToken)
		if err != nil {
			log.Error("Error checking auth session: " + err.Error())
			http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		if sessionValid && sessionExists {
			next.ServeHTTP(w, req)
		} else if sessionExists && !sessionValid {
			sessionCount = config.GetAuthSessionCount()
			log.Debug("Auth cache miss: " + requestIP + " (Sessions: " + strconv.Itoa(int(sessionCount)) + ") " + requestURL)
			if queryType == "new-token" && strings.TrimSpace(requestBasicToken) != "" {
				next.ServeHTTP(w, req)
			} else {
				http.Error(w, FormatHttpError("Unauthorized"), http.StatusUnauthorized)
				return
			}
		} else {
			log.Warning("No valid authentication found for request: " + requestIP + " ( " + requestURL + ")")
			http.Error(w, FormatHttpError("Unauthorized"), http.StatusUnauthorized)
			return
		}
	})
}

func CookieAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("no IP address stored in context")
			http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		requestURL, ok := GetRequestURL(req)
		if !ok {
			log.Warning("no URL stored in context")
			http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		uitSessionIDCookie, sessionErr := req.Cookie("uit_session_id")
		uitBasicCookie, basicErr := req.Cookie("uit_basic_token")
		uitBearerCookie, bearerErr := req.Cookie("uit_bearer_token")
		uitCSRFCookie, csrfErr := req.Cookie("uit_csrf_token")

		if sessionErr != nil || basicErr != nil || bearerErr != nil || csrfErr != nil {
			if sessionErr != nil && sessionErr != http.ErrNoCookie {
				log.Error("Error retrieving UIT cookies: " + requestIP + " (" + requestURL + ")")
			}
			http.Error(w, FormatHttpError("Unauthorized"), http.StatusUnauthorized)
			return
		}

		config.ClearExpiredAuthSessions()

		sessionValid, sessionExists, err := config.CheckAuthSessionExists(uitSessionIDCookie.Value, requestIP, uitBasicCookie.Value, uitBearerCookie.Value, uitCSRFCookie.Value)
		if err != nil {
			log.Error("Error checking auth session: " + err.Error())
			http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		if sessionValid && sessionExists && !strings.HasSuffix(requestURL, "/logout") {
			sessionIDCookie, basicCookie, bearerCookie, csrfCookie := GetAuthCookiesForResponse(uitSessionIDCookie.Value, uitBasicCookie.Value, uitBearerCookie.Value, uitCSRFCookie.Value, 20*time.Minute)
			http.SetCookie(w, sessionIDCookie)
			http.SetCookie(w, basicCookie)
			http.SetCookie(w, bearerCookie)
			http.SetCookie(w, csrfCookie)

			sessionExtended, err := config.ExtendAuthSession(uitSessionIDCookie.Value)
			if err != nil {
				log.Error("Error extending auth session: " + err.Error())
				http.Error(w, FormatHttpError("Internal server error"), http.StatusInternalServerError)
				return
			}
			if sessionExtended {
				log.Debug("Auth session extended: " + requestIP)
				next.ServeHTTP(w, req)
			} else {
				log.Debug("Auth session not found or expired: " + requestIP)
			}
		} else if sessionExists && strings.HasSuffix(requestURL, "/logout") {
			log.Debug("Logging out user: " + requestIP)
			config.DeleteAuthSession(uitSessionIDCookie.Value)
			sessionCount := config.RefreshAndGetAuthSessionCount()
			log.Info("(Cleanup) Auth session expired: " + requestIP + " (" + strconv.Itoa(int(sessionCount)) + " session(s))")
			return
		} else {
			log.Info("No valid authentication found for request: " + requestIP + " ( " + requestURL + ")")
			http.Error(w, FormatHttpError("Unauthorized"), http.StatusUnauthorized)
			return
		}
	})
}
