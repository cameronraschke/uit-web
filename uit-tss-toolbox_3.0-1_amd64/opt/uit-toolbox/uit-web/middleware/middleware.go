package middleware

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"

	config "uit-toolbox/config"
)

func PanicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log := config.GetLogger()
				requestIP, _, _ := net.SplitHostPort(req.RemoteAddr)

				log.Error(fmt.Sprintf("Panic recovered: %v\n%s",
					err, string(debug.Stack())))
				log.Error("Request from: " + requestIP +
					" " + req.Method + " " + req.URL.Path)

				WriteJsonError(w, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, req)
	})
}

func LimitRequestSizeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		if req.Header.Get("Content-Length") == "" && (req.Method == http.MethodPost || req.Method == http.MethodPut) {
			contentLengthHeader := req.Header.Get("Content-Length")
			if strings.TrimSpace(contentLengthHeader) == "" {
				log.Warning("Request content length is missing: " + fmt.Sprintf("%d", req.ContentLength))
				WriteJsonError(w, http.StatusLengthRequired)
				return
			}
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

		// withClientIP parses and casts the IP address to ipnet.Addr type
		ctx, err := withClientIP(req.Context(), ip)
		if err != nil {
			log.Warning("Error storing IP address in context: " + err.Error())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func CheckIPBlockedMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		requestIP, ok := GetRequestIPFromRequestContext(req)
		if !ok {
			log.Warning("No IP address stored in context")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if config.IsIPBlocked(requestIP) {
			log.Warning("Blocked IP attempted request: " + requestIP.String())
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, req)
	})
}

func WebEndpointConfigMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		requestIP, ok := GetRequestIPFromRequestContext(req)
		if !ok {
			log.Warning("No IP address stored in context")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		endpointConfig, err := config.GetWebEndpointConfig(req.URL.Path)
		if err != nil {
			log.Warning("Error getting endpoint config (" + requestIP.String() + " " + req.Method + " " + req.URL.Path + ")")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		endpointType, err := config.GetWebEndpointType(endpointConfig)
		if err != nil || endpointType == "" {
			log.Warning("No valid endpoint config for URL: " + req.URL.Path)
			WriteJsonError(w, http.StatusNotFound)
			return
		}
		if endpointType == "static_file" {
			filePath, err := config.GetWebEndpointFilePath(endpointConfig)
			if err != nil || strings.TrimSpace(filePath) == "" {
				log.Warning("No file path configured for endpoint: " + req.URL.Path)
				WriteJsonError(w, http.StatusNotFound)
				return
			}
		}

		ctx, err := withWebEndpointConfig(req.Context(), &endpointConfig)
		if err != nil {
			log.Warning("Error storing endpoint config in context: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, req.WithContext(ctx))
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
		endpointConfig, ok := GetWebEndpointConfigFromRequestContext(req)
		if !ok {
			log.Warning("Error getting endpoint config in TLS middleware (" + requestIP.String() + " " + req.Method + " " + req.URL.Path + ")")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if endpointConfig.TLSRequired != nil && !*endpointConfig.TLSRequired {
			next.ServeHTTP(w, req)
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

			// IsClientRateLimited assigns a rate limiter to the client IP if not already present
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

func HTTPMethodMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		requestIP, ok := GetRequestIPFromRequestContext(req)
		if !ok {
			log.Warning("No IP address stored in context")
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

		endpointConfig, ok := GetWebEndpointConfigFromRequestContext(req)
		if !ok {
			log.Warning("Error getting endpoint config in HTTP method middleware (" + requestIP.String() + " " + req.Method + " " + req.URL.Path + ")")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		if !slices.Contains(endpointConfig.AllowedMethods, req.Method) {
			log.Info("Method is not allowed for endpoint (" + req.URL.Path + "): " + req.Method + " from " + requestIP.String())
			WriteJsonError(w, http.StatusMethodNotAllowed)
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
			log.Warning("Request URL length exceeds character limit: " + fmt.Sprintf("%d", len(req.URL.RequestURI())) + " (" + requestIP.String() + ": " + req.Method + " " + req.URL.RequestURI() + ")")
			WriteJsonError(w, http.StatusRequestURITooLong)
			return
		}

		// URL path
		cleanPath, err := validateAndCleanPath(req.URL.Path)
		if err != nil {
			log.Warning("Invalid path from " + requestIP.String() + ": " + err.Error())
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		// Validate query parameters (even if empty)
		if err := validateQueryParams(req.URL.Query()); err != nil {
			log.Warning("Invalid query parameters from " + requestIP.String() + ": " + err.Error())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Check RawQuery for null bytes and CRLF. req.URL.Query() and url.Parse() may be empty even if RawQuery is not.
		if strings.Contains(req.URL.RawQuery, "\x00") {
			log.Warning("Null byte detected in raw query string from " + requestIP.String())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if strings.ContainsAny(req.URL.RawQuery, "\r\n") {
			log.Warning("CRLF characters detected in raw query string from " + requestIP.String())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Chain context updates
		ctx := req.Context()
		// Store URL in context (raw path + raw query - try not to use unless necessary)
		ctx, err = withRequestURL(ctx, req.URL.RequestURI())
		if err != nil {
			log.Warning("Error storing URL in context: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		// Store clean path in context (to be used later on)
		ctx, err = withRequestPath(ctx, cleanPath)
		if err != nil {
			log.Warning("Error storing path in context: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		// Store raw query in context, even if empty (to be used later on)
		ctx, err = withRequestQuery(ctx, req.URL.RawQuery)
		if err != nil {
			log.Warning("Error storing query in context: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, req.WithContext(ctx))
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

		for headerKey, headerValues := range req.Header {
			// Block CRLF and null bytes in header keys
			if strings.ContainsAny(headerKey, "\x00\r\n") {
				log.Warning("CRLF or null byte in header key from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
				WriteJsonError(w, http.StatusBadRequest)
				return
			}
			// Check header key length and critical characters
			if len(headerKey) > 255 {
				log.Warning("Header key too long: '" + headerKey + "' (" + fmt.Sprintf("%d", len(headerKey)) + " bytes) from " + requestIP.String())
				WriteJsonError(w, http.StatusBadRequest)
				return
			}

			for _, headerValue := range headerValues {
				// Block CRLF and null bytes in ALL header values
				if strings.ContainsAny(headerValue, "\x00\r\n") {
					log.Warning("CRLF or null byte in header '" + headerKey + "' from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
					WriteJsonError(w, http.StatusBadRequest)
					return
				}
				// Default max length for header values (specific requirements below)
				if len(headerValue) > 8192 {
					log.Warning("Header value too long for '" + headerKey + "': " + fmt.Sprintf("%.2f", float64(len(headerValue))/1024) + " KB from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
					WriteJsonError(w, http.StatusBadRequest)
					return
				}
			}
		}

		// Required headers
		// Host header (required)
		host := req.Host
		if strings.TrimSpace(host) == "" {
			log.Warning("Missing Host header from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if len(host) > 255 {
			log.Warning("Host header too long: " + fmt.Sprintf("%d bytes", len(host)) + " from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		// Block dangerous characters in Host header (already checked \x00\r\n above)
		if strings.ContainsAny(host, " <>\"'") {
			log.Warning("Invalid characters in Host header from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// User-Agent header (required)
		userAgent := req.Header.Get("User-Agent")
		if userAgent == "" {
			log.Warning("Missing User-Agent header from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if len(userAgent) > 512 {
			log.Warning("User-Agent too long: " + fmt.Sprintf("%d bytes", len(userAgent)) + " from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Content-Type (required for POST/PUT)
		if req.Method == http.MethodPost || req.Method == http.MethodPut {
			contentType := req.Header.Get("Content-Type")
			if contentType == "" {
				log.Warning("Missing Content-Type for " + req.Method + " from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
				WriteJsonError(w, http.StatusBadRequest)
				return
			}
			if len(contentType) > 256 {
				log.Warning("Content-Type too long: " + fmt.Sprintf("%d bytes", len(contentType)) + " from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
				WriteJsonError(w, http.StatusBadRequest)
				return
			}
			if contentType != "application/x-www-form-urlencoded" && contentType != "application/json" && !strings.HasPrefix(contentType, "multipart/form-data") {
				log.Warning("Invalid Content-Type header: " + contentType + " (" + requestIP.String() + ": " + req.Method + " " + requestURL + ")")
				WriteJsonError(w, http.StatusUnsupportedMediaType)
				return
			}
		}

		// Optional headers: Validate if present
		// Cookie (optional, validate length if present)
		cookieHeader := req.Header.Get("Cookie")
		if len(cookieHeader) > 4096 {
			log.Warning("Cookie header too large: " + fmt.Sprintf("%.2f KB", float64(len(cookieHeader))/1024) + " from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Authorization (optional, validate format if present)
		authorization := req.Header.Get("Authorization")
		if authorization != "" {
			if len(authorization) > 8192 {
				log.Warning("Authorization header too long: " + fmt.Sprintf("%.2f KB", float64(len(authorization))/1024) + " from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
				WriteJsonError(w, http.StatusBadRequest)
				return
			}
			// Must start with Bearer or Basic
			if !strings.HasPrefix(authorization, "Bearer ") && !strings.HasPrefix(authorization, "Basic ") {
				log.Warning("Invalid Authorization format (missing Bearer/Basic prefix) from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
				WriteJsonError(w, http.StatusBadRequest)
				return
			}
		}

		// Connection header (disallowed, not allowed in HTTP/2)
		if req.ProtoMajor == 2 && req.Header.Get("Connection") != "" {
			log.Warning("Connection header present in HTTP/2 request from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Transfer-Encoding (optional, check for request smuggling)
		transferEncoding := req.Header.Get("Transfer-Encoding")
		if transferEncoding != "" && transferEncoding != "chunked" {
			log.Warning("Suspicious Transfer-Encoding: '" + transferEncoding + "' from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Referer (optional, but validate length)
		referer := req.Header.Get("Referer")
		if len(referer) > 2048 {
			log.Warning("Referer too long: " + fmt.Sprintf("%.2f KB", float64(len(referer))/1024) + " from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Origin (if using CORS, validate length)
		origin := req.Header.Get("Origin")
		if len(origin) > 2048 {
			log.Warning("Origin too long: " + fmt.Sprintf("%.2f KB", float64(len(origin))/1024) + " from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Suspicious headers
		if xForwardedFor := req.Header.Get("X-Forwarded-For"); xForwardedFor != "" {
			log.Warning("X-Forwarded-For header present (possible proxy bypass): " + xForwardedFor + " from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
			// Log and don't return
		}

		if xRealIP := req.Header.Get("X-Real-IP"); xRealIP != "" {
			log.Warning("X-Real-IP header present (possible IP spoofing): " + xRealIP + " from " + requestIP.String() + ": (" + req.Method + " " + requestURL + ")")
			// Log and don't return
		}

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

		// SERVER-SIDE CORS CHECKS
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

		// OPTIONS preflight request handling
		if req.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "https://"+httpsServerIP+":1411")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400") // Cache preflight for 24 hours
			w.Header().Set("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// CORS policy
		w.Header().Set("Access-Control-Allow-Origin", "https://"+httpsServerIP+":1411")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Vary", "Origin")

		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload") // 2 years
		w.Header().Set("Referrer-Policy", "no-referrer")
		nonce, err := GenerateNonce(24)
		if err != nil {
			log.Error("Cannot generate CSP nonce: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		ctx := req.Context()
		ctx, err = withNonce(ctx, nonce)
		if err != nil {
			log.Error("Error storing CSP nonce in context: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		req = req.WithContext(ctx)
		cspPolicy := "default-src 'self'; " +
			"style-src 'self'; " +
			"script-src 'self' 'nonce-" + nonce + "'; " +
			"worker-src 'self'; " +
			"img-src 'self' blob: data:; " +
			"font-src 'self'; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'; " +
			"upgrade-insecure-requests"
		w.Header().Set("Content-Security-Policy", cspPolicy)
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")

		// Cache headers
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")

		// Browser permissions
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=(), accelerometer=()")

		// Hide server information
		w.Header().Set("Server", "")

		next.ServeHTTP(w, req)
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
		pathRequested, ok := GetRequestPathFromRequestContext(req)
		if !ok {
			log.Warning("No URL path stored in context")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		fileRequested := filepath.Base(pathRequested)
		endpointConfig, err := config.GetWebEndpointConfig(pathRequested)
		if err != nil {
			log.Warning("Error getting endpoint config in AllowedFilesMiddleware (" + requestIP.String() + " " + req.Method + " " + req.URL.Path + ")")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		endpointFilePath, err := config.GetWebEndpointFilePath(endpointConfig)
		if err != nil {
			log.Warning("No file path configured for endpoint in AllowedFilesMiddleware (" + requestIP.String() + " " + req.Method + " " + req.URL.Path + ")")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		// if !config.IsFileAllowed(fileRequested) {
		// 	log.Warning("File not in whitelist: " + fileRequested)
		// 	WriteJsonError(w, http.StatusForbidden)
		// 	return
		// }

		resolvedPath, err := filepath.EvalSymlinks(endpointFilePath)
		if err != nil || resolvedPath != endpointFilePath {
			log.Warning("File request error from " + requestIP.String() + " (" + resolvedPath + "): Error resolving symlink: " + err.Error())
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		if resolvedPath != endpointFilePath {
			log.Warning("Resolved path does not match full path (" + requestIP.String() + "): " + resolvedPath + " -> " + endpointFilePath)
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		metadata, err := os.Lstat(endpointFilePath)
		if err != nil {
			log.Error("Cannot get metadata from file: " + endpointFilePath + " (" + err.Error() + ")")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if metadata == nil {
			log.Error("Metadata is nil for file: " + endpointFilePath)
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		// if metadata.Name() != fileRequested || !config.IsFileAllowed(metadata.Name()) {
		// 	log.Warning("Filename mismatch: " + metadata.Name() + " != " + fileRequested)
		// 	WriteJsonError(w, http.StatusForbidden)
		// 	return
		// }
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
