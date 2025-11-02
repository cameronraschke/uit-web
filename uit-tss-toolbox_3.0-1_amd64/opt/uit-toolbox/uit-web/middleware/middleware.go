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
		contentLengthHeader := req.Header.Get("Content-Length")
		if strings.TrimSpace(contentLengthHeader) == "" {
			log.Warning("Request content length is missing: " + fmt.Sprintf("%d", req.ContentLength))
			WriteJsonError(w, http.StatusLengthRequired)
		}
		maxSize, err := config.GetMaxUploadSize()
		if err != nil {
			log.Error("Failed to get max upload size from config: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if req.ContentLength > maxSize {
			//req.RemoteAddr used here because the ip has not been assigned to the context yet
			log.Warning("Request content length exceeds limit: " + fmt.Sprintf("%.2fMB", float64(req.ContentLength)/1e6) + " " + req.RemoteAddr)
			WriteJsonError(w, http.StatusRequestEntityTooLarge)
			return
		}
		req.Body = http.MaxBytesReader(w, req.Body, maxSize)
		next.ServeHTTP(w, req)
	})
}

func APITimeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		apiTimeout, err := config.GetRequestTimeout("api")
		if err != nil {
			log.Error("Failed to get request API timeout from config: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		ctx, cancel := context.WithTimeout(req.Context(), apiTimeout)
		defer cancel()
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func FileServerTimeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		fileTimeout, err := config.GetRequestTimeout("file")
		if err != nil {
			log.Error("Failed to get request file timeout from config: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		ctx, cancel := context.WithTimeout(req.Context(), fileTimeout)
		defer cancel()
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func StoreClientIPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		ip, port, err := net.SplitHostPort(req.RemoteAddr)
		if err != nil {
			log.Warning("Could not parse IP address: " + err.Error())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(port) == "" {
			log.Warning("Empty port in request")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		ipValid, _, _, err := checkValidIP(ip)
		if err != nil {
			log.Warning("Error validating/storing IP address in context: " + err.Error())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !ipValid {
			log.Warning("Cannot store invalid IP address in context: " + ip)
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		ctx, err := withClientIP(req.Context(), ip)
		if err != nil {
			log.Warning("Error storing IP address in context: " + err.Error())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func RateLimitMiddleware(rateType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			log := config.GetLogger()
			requestIP, ok := GetRequestIPFromRequestContext(req)
			if !ok {
				log.Warning("no IP address stored in context")
				WriteJsonError(w, http.StatusInternalServerError)
				return
			}

			if config.IsIPBlocked(requestIP) {
				log.Debug("Blocked IP attempted request: " + requestIP.String())
				WriteJsonError(w, http.StatusTooManyRequests)
				return
			}

			limited, retryAfter := config.IsClientRateLimited(rateType, requestIP)
			if limited {
				log.Debug("Client is rate limited: " + requestIP.String() + " (retry after " + retryAfter.String() + ")")
				WriteJsonError(w, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, req)
		})
	}
}

func AllowIPRangeMiddleware(trafficSource string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			log := config.GetLogger()
			if strings.TrimSpace(trafficSource) == "" {
				log.Warning("No traffic source specified for AllowIPRangeMiddleware")
				WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			requestIP, ok := GetRequestIPFromRequestContext(req)
			if !ok {
				log.Warning("No IP address stored in context")
				WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			allowed, err := config.IsIPAllowed(trafficSource, requestIP)
			if err != nil {
				log.Error("Error checking if IP is allowed: " + err.Error())
				WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			if !allowed {
				log.Warning("IP address not in allowed range: " + requestIP.String())
				WriteJsonError(w, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}

func CheckHttpVersionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		requestIP, ok := GetRequestIPFromRequestContext(req)
		if !ok {
			log.Warning("No IP address stored in context")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if req.ProtoMajor != 2 {
			log.Warning("Client does not support HTTP/2: " + requestIP.String())

			w.Header().Set("Upgrade", "HTTP/2")
			WriteJsonError(w, http.StatusUpgradeRequired)
			return
		}
		next.ServeHTTP(w, req)
	})
}

func TLSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		requestIP, ok := GetRequestIPFromRequestContext(req)
		if !ok {
			log.Warning("No IP address stored in context")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		if req.TLS == nil || !req.TLS.HandshakeComplete {
			log.Warning("TLS handshake failed for client " + requestIP.String())
			// w.Header().Set("Location", "https://"+req.Host+req.RequestURI)
			// WriteJsonError(w, http.StatusSeeOther)
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		if req.TLS.Version < tls.VersionTLS13 {
			log.Warning("Rejected connection with weak TLS version from " + requestIP.String())
			WriteJsonError(w, http.StatusUpgradeRequired)
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
			log.Warning("Rejected connection with weak cipher suite from " + requestIP.String())
			WriteJsonError(w, http.StatusUpgradeRequired)
			return
		}

		next.ServeHTTP(w, req)
	})
}

func CheckValidURLMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		requestIP, ok := GetRequestIPFromRequestContext(req)
		if !ok {
			log.Warning("No IP address stored in context")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		// URL length
		if len(req.URL.RequestURI()) > 2048 {
			log.Warning("Request URL length exceeds limit: " + fmt.Sprintf("%d", len(req.URL.RequestURI())) + " (" + requestIP.String() + ": " + req.Method + " " + req.URL.RequestURI() + ")")
			WriteJsonError(w, http.StatusRequestURITooLong)
			return
		}

		// URL query
		queryValues := req.URL.Query()
		disallowedQueryChars := "~%$#\\<>:\"'`|?*\x00\r\n"
		for k, v := range queryValues {
			if len(k) > 128 || len(v) > 512 {
				log.Warning("Request URL query key length exceeds limit: " + fmt.Sprintf("%d", len(k)) + " (" + requestIP.String() + ": " + req.Method + " " + req.URL.RequestURI() + ")")
				WriteJsonError(w, http.StatusRequestURITooLong)
				return
			}
			if strings.ContainsAny(k, disallowedQueryChars) {
				log.Warning("Invalid characters in query key: " + "( " + requestIP.String() + ": " + req.Method + " " + req.URL.RequestURI() + ")")
				WriteJsonError(w, http.StatusBadRequest)
				return
			}
			for _, value := range v {
				if strings.ContainsAny(value, disallowedQueryChars) {
					log.Warning("Invalid characters in query value: " + "( " + requestIP.String() + ": " + req.Method + " " + req.URL.RequestURI() + ")")
					WriteJsonError(w, http.StatusBadRequest)
					return
				}
			}
		}

		// Check URL path
		var disallowedPathChars = "~%$#\\<>:\"'`|?*\x00\r\n"
		// Get unescaped path (decode URL) & normalize UTF-8
		rawPath := strings.TrimSpace(req.URL.Path)
		if rawPath == "" || len(rawPath) > 255 {
			log.Warning("Empty URL requested from: " + requestIP.String())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		unescapedPath, err := url.PathUnescape(rawPath)
		if err != nil {
			log.Warning("Cannot unescape URL path: " + err.Error())
			WriteJsonError(w, http.StatusBadRequest)
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
			log.Warning("Normalized URL path is invalid: " + requestIP.String())
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		// Clean entire path and format the URL path
		fullPath := path.Clean(normalizedPath)
		if !path.IsAbs(fullPath) ||
			strings.Contains(fullPath, "..") ||
			strings.Contains(fullPath, "../") ||
			fullPath == "." {

			log.Warning("Empty file path requested: " + requestIP.String() + " (" + fullPath + ")")
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		// Split URL path into path + file name
		fileRequestedWithQuery, err := url.Parse(fullPath)
		if err != nil {
			log.Warning("Failed to parse URL path: " + err.Error())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		_, fileRequested := path.Split(fileRequestedWithQuery.Path)
		if strings.HasPrefix(fileRequested, ".") ||
			strings.HasPrefix(fileRequested, "~") ||
			strings.HasSuffix(fileRequested, ".tmp") ||
			strings.HasSuffix(fileRequested, ".bak") ||
			strings.HasSuffix(fileRequested, ".swp") {

			log.Warning("Invalid characters in file requested")
			WriteJsonError(w, http.StatusForbidden)
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
					log.Warning("Control/non-printable character in filename: " + requestIP.String())
					WriteJsonError(w, http.StatusForbidden)
					return
				}
				if char > 127 || char > unicode.MaxASCII || char > unicode.MaxLatin1 {
					log.Warning("Non-ASCII character in filename: " + requestIP.String())
					WriteJsonError(w, http.StatusForbidden)
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
					WriteJsonError(w, http.StatusForbidden)
					return
				}

				if strings.ContainsRune(disallowedPathChars, char) {
					log.Warning("Disallowed character in filename: " + requestIP.String())
					WriteJsonError(w, http.StatusForbidden)
					return
				}
			}
		}

		ctx, err := withRequestURL(req.Context(), fullPath)
		if err != nil {
			log.Warning("Error storing URL in context: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func AllowedFilesMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		requestIP, ok := GetRequestIPFromRequestContext(req)
		if !ok {
			log.Warning("No IP address stored in context")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		requestURLString, ok := GetRequestURLFromRequestContext(req)
		if !ok {
			log.Warning("No URL stored in context")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		requestURLParsed, err := url.Parse(requestURLString)
		if err != nil {
			log.Warning("Failed to parse request URL: " + err.Error())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		requestURLPath := requestURLParsed.Path
		if requestURLPath == "/login" || requestURLPath == "/logout" || requestURLPath == "/dashboard" || requestURLPath == "/inventory" || requestURLPath == "/client_images" {
			requestURLPath = requestURLPath + ".html"
		}

		if requestURLPath == "" || requestURLPath == "/" {
			requestURLPath = "/dashboard.html"
		}

		var basePath string
		if strings.HasPrefix(requestURLString, "/client/") {
			basePath = "/srv/uit-toolbox/"
		} else {
			basePath = "/var/www/html/uit-web/"
		}

		fullPath := path.Join(basePath, requestURLPath)
		fileRequested := path.Base(requestURLPath)
		if !path.IsAbs(fullPath) ||
			strings.TrimSpace(fullPath) == "." ||
			strings.TrimSpace(fullPath) == "/" {
			log.Warning("Invalid file path: " + fullPath)
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		if !config.IsFileAllowed(fileRequested) {
			log.Warning("File not in whitelist: " + fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		resolvedPath, err := filepath.EvalSymlinks(fullPath)
		if err != nil || !strings.HasPrefix(resolvedPath, basePath) {
			log.Warning("File request error from " + requestIP.String() + " (" + resolvedPath + "): Error resolving symlink: " + err.Error())
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		if resolvedPath != fullPath {
			log.Warning("Resolved path does not match full path (" + requestIP.String() + "): " + resolvedPath + " -> " + fullPath)
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		metadata, err := os.Lstat(fullPath)
		if err != nil {
			log.Error("Cannot get metadata from file: " + fullPath + " (" + err.Error() + ")")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if metadata == nil {
			log.Error("Metadata is nil for file: " + fullPath)
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if metadata.Name() != fileRequested || !config.IsFileAllowed(metadata.Name()) {
			log.Warning("Filename mismatch: " + metadata.Name() + " != " + fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if metadata.Size() <= 0 {
			log.Warning("Attempt to access empty file: " + fileRequested)
			WriteJsonError(w, http.StatusNoContent)
			return
		}
		if metadata.IsDir() {
			log.Warning("Attempt to access directory as file: " + fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		fileMode := metadata.Mode()
		if fileMode&os.ModeSymlink != 0 {
			log.Warning("Attempt to access symbolic link: " + fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if fileMode&os.ModeDevice != 0 {
			log.Warning("Attempt to access device file: " + fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if fileMode&os.ModeNamedPipe != 0 {
			log.Warning("Attempt to access named pipe: " + fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if fileMode&os.ModeSocket != 0 {
			log.Warning("Attempt to access socket file: " + fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if fileMode&os.ModeCharDevice != 0 {
			log.Warning("Attempt to access character device file: " + fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if fileMode&os.ModeIrregular != 0 {
			log.Warning("Attempt to access irregular file: " + fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if !fileMode.IsRegular() {
			log.Warning("Attempt to access non-regular file: " + fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		ctx, err := withRequestFile(req.Context(), resolvedPath)
		if err != nil {
			log.Warning("Error storing file request in context: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func HTTPMethodMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		// Get IP address
		requestIP, ok := GetRequestIPFromRequestContext(req)
		if !ok {
			log.Warning("No IP address stored in context")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		requestURL, ok := GetRequestURLFromRequestContext(req)
		if !ok {
			log.Warning("No URL stored in context")
			WriteJsonError(w, http.StatusInternalServerError)
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
			log.Warning("Invalid request method (" + requestIP.String() + "): " + req.Method)
			WriteJsonError(w, http.StatusMethodNotAllowed)
			return
		}

		// Check Content-Type for POST/PUT
		if req.Method == http.MethodPost || req.Method == http.MethodPut {
			contentType := req.Header.Get("Content-Type")
			if contentType != "application/x-www-form-urlencoded" && contentType != "application/json" && !strings.HasPrefix(contentType, "multipart/form-data") {
				log.Warning("Invalid Content-Type header: " + contentType + " (" + requestIP.String() + ": " + req.Method + " " + requestURL + ")")
				WriteJsonError(w, http.StatusUnsupportedMediaType)
				return
			}
		}
		next.ServeHTTP(w, req)
	})
}

func CheckHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		requestIP, ok := GetRequestIPFromRequestContext(req)
		if !ok {
			log.Warning("No IP address stored in context")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		requestURL, ok := GetRequestURLFromRequestContext(req)
		if !ok {
			log.Warning("No URL stored in context")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		// Content length
		if req.ContentLength > 64<<20 {
			log.Warning("Request content length exceeds limit: " + fmt.Sprintf("%.2fMB", float64(req.ContentLength)/1e6))
			WriteJsonError(w, http.StatusRequestEntityTooLarge)
			return
		}

		// Origin header
		origin := req.Header.Get("Origin")
		if origin != "" && len(origin) > 2048 {
			log.Warning("Invalid Origin header: " + origin + " (" + requestIP.String() + ": " + req.Method + " " + requestURL + ")")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Host header
		host := strings.TrimSpace(req.Host)
		if host == "" || strings.ContainsAny(host, " <>\"'%;()&+") || len(host) > 255 {
			log.Warning("Invalid Host header: " + host + " (" + requestIP.String() + ": " + req.Method + " " + requestURL + ")")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// User-Agent header
		userAgent := req.Header.Get("User-Agent")
		if userAgent == "" || len(userAgent) > 256 {
			log.Warning("Invalid User-Agent header: " + userAgent + " (" + requestIP.String() + ": " + req.Method + " " + requestURL + ")")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Referer header
		referer := req.Header.Get("Referer")
		if referer != "" && len(referer) > 2048 {
			log.Warning("Invalid Referer header: " + referer + " (" + requestIP.String() + ": " + req.Method + " " + requestURL + ")")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Other headers
		// for key, value := range req.Header {
		//   if strings.ContainsAny(key, "<>\"'%;()&+") || strings.ContainsAny(value[0], "<>\"'%;()&+") {
		//     log.Warning("Invalid characters in header '" + key + "': " + value[0] + " (" + requestIP + ": " + req.Method + " " + requestURL + ")")
		//     WriteJsonError(w, http.StatusBadRequest)
		//     return
		//   }
		// }

		next.ServeHTTP(w, req)
	})
}

func SetHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		requestIP, ok := GetRequestIPFromRequestContext(req)
		if !ok {
			log.Warning("No IP address stored in context")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		requestURL, ok := GetRequestURLFromRequestContext(req)
		if !ok {
			log.Warning("No URL stored in context")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		// Get web server IP for CORS
		_, httpsServerIP, err := config.GetWebServerIPs()
		if err != nil || strings.TrimSpace(httpsServerIP) == "" {
			log.Error("Cannot get web server IP for CORS: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		// Check CORS policy
		cors := http.NewCrossOriginProtection()
		cors.AddTrustedOrigin("https://" + httpsServerIP + ":1411")
		if err := cors.Check(req); err != nil {
			log.Warning("Request to " + requestURL + " blocked from " + requestIP.String())
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		// Handle OPTIONS early
		if req.Method == http.MethodOptions {
			// Headers for OPTIONS request
			w.Header().Set("Access-Control-Allow-Origin", "https://"+httpsServerIP+":1411")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Set-Cookie, credentials")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "https://"+httpsServerIP+":1411")
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

func CookieAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		requestIP, ok := GetRequestIPFromRequestContext(req)
		if !ok {
			log.Warning("No IP address stored in context")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		requestURL, ok := GetRequestURLFromRequestContext(req)
		if !ok {
			log.Warning("No URL stored in context")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		uitSessionIDCookie, sessionErr := req.Cookie("uit_session_id")
		uitBasicCookie, basicErr := req.Cookie("uit_basic_token")
		uitBearerCookie, bearerErr := req.Cookie("uit_bearer_token")
		uitCSRFCookie, _ := req.Cookie("uit_csrf_token")

		if sessionErr != nil || basicErr != nil || bearerErr != nil {
			if sessionErr != nil && sessionErr != http.ErrNoCookie {
				log.Error("Error retrieving UIT cookies: " + requestIP.String() + " (" + requestURL + ")")
			}
			log.Info("No authentication cookies found for request: " + requestIP.String() + " (" + requestURL + ")")
			log.Info("Basic cookie error: " + fmt.Sprintf("%v", basicErr) + ", Bearer cookie error: " + fmt.Sprintf("%v", bearerErr) + ", SessionID cookie error: " + fmt.Sprintf("%v", sessionErr) + ")")
			// WriteJsonError(w, FormatHttpError("Unauthorized"), http.StatusUnauthorized)
			http.Redirect(w, req, "/login", http.StatusSeeOther)
			return
		}

		config.ClearExpiredAuthSessions()

		sessionValid, sessionExists, err := config.CheckAuthSessionExists(uitSessionIDCookie.Value, requestIP, uitBasicCookie.Value, uitBearerCookie.Value, uitCSRFCookie.Value)
		if err != nil {
			log.Error("Error checking auth session: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
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
				WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			if sessionExtended {
				log.Debug("Auth session extended: " + requestIP.String())
				next.ServeHTTP(w, req)
			} else {
				log.Debug("Auth session not found or expired: " + requestIP.String())
			}
		} else if sessionExists && strings.TrimSpace(requestURL) == "/logout" {
			log.Debug("Logging out user: " + requestIP.String())
			config.DeleteAuthSession(uitSessionIDCookie.Value)
			sessionCount := config.RefreshAndGetAuthSessionCount()
			log.Info("Auth session deleted: " + requestIP.String() + " (" + strconv.Itoa(int(sessionCount)) + " session(s))")
			http.Redirect(w, req, "/login", http.StatusSeeOther)
			return
		} else {
			log.Info("No valid authentication found for request: " + requestIP.String() + " (" + requestURL + ")")
			// WriteJsonError(w, FormatHttpError("Unauthorized"), http.StatusUnauthorized)
			http.Redirect(w, req, "/login", http.StatusSeeOther)
			return
		}
	})
}
