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

type Middleware func(http.Handler) http.Handler

type MiddlewareChain struct {
	middlewares []Middleware
}

func NewChain(middlewares ...Middleware) MiddlewareChain {
	return MiddlewareChain{
		middlewares: append([]Middleware{}, middlewares...),
	}
}

// Append extends the chain with additional middlewares, returning a new MiddlewareChain
func (chain MiddlewareChain) Append(middlewares ...Middleware) MiddlewareChain {
	newMiddlewares := make([]Middleware, 0, len(chain.middlewares)+len(middlewares))
	newMiddlewares = append(newMiddlewares, chain.middlewares...)
	newMiddlewares = append(newMiddlewares, middlewares...)

	return MiddlewareChain{
		middlewares: newMiddlewares,
	}
}

// Then applies the middleware chain to the final handler
func (chain MiddlewareChain) Then(finalHandler http.Handler) http.Handler {
	for i := len(chain.middlewares) - 1; i >= 0; i-- {
		finalHandler = chain.middlewares[i](finalHandler)
	}
	return finalHandler
}

// ThenFunc applies the middleware chain to a handler function
func (chain MiddlewareChain) ThenFunc(finalHandlerFunc http.HandlerFunc) http.Handler {
	return chain.Then(finalHandlerFunc)
}

func StoreLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := config.GetLogger()
		ctx, err := withLogger(req.Context(), log)
		if err != nil {
			fmt.Println("Error storing logger in context: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func PanicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log := GetLoggerFromContext(req.Context())
				if log == nil {
					fmt.Println("Error getting logger from context in panic recovery middleware, attempting to use global logger")
					log := config.GetLogger()
					if log == nil {
						fmt.Println("Error getting global logger in panic recovery middleware: logger is nil")
						WriteJsonError(w, http.StatusInternalServerError)
						return
					}
				}

				log.Error(fmt.Sprintf("Panic recovered: %v\n%s", err, string(debug.Stack())))
				log.HTTPError(req, "Request panicked")

				WriteJsonError(w, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, req)
	})
}

func LimitRequestSizeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := GetLoggerFromContext(req.Context())
		if strings.TrimSpace(req.Header.Get("Content-Length")) == "" && (req.Method == http.MethodPost || req.Method == http.MethodPut) {
			log.HTTPWarning(req, "Request content length is missing")
			WriteJsonError(w, http.StatusLengthRequired)
			return
		}
		maxSize, err := config.GetMaxUploadSize()
		if err != nil {
			log.HTTPError(req, "Failed to get max upload size from config: "+err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if req.ContentLength > maxSize {
			log.HTTPWarning(req, "Request content length exceeds limit: "+fmt.Sprintf("%.2fMB", float64(req.ContentLength)/1e6)+" > "+fmt.Sprintf("%.2fMB", float64(maxSize)/1e6))
			WriteJsonError(w, http.StatusRequestEntityTooLarge)
			return
		}
		req.Body = http.MaxBytesReader(w, req.Body, maxSize)
		next.ServeHTTP(w, req)
	})
}

func StoreClientIPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := GetLoggerFromContext(req.Context())
		ipStr, port, err := net.SplitHostPort(req.RemoteAddr)
		if err != nil {
			log.HTTPError(req, "Error parsing client IP from RemoteAddr: "+err.Error())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(port) == "" {
			log.HTTPError(req, "Error parsing client IP from RemoteAddr: missing port")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		ip, ipValid, _, _, err := checkValidIP(ipStr)
		if err != nil {
			log.HTTPWarning(req, "Error validating IP address for use in context: "+err.Error())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if !ipValid {
			log.HTTPWarning(req, "Cannot store invalid IP address in context: "+ip.String())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// withClientIP parses and casts the IP address to ipnet.Addr type
		ctx, err := withClientIP(req.Context(), ip)
		if err != nil {
			log.HTTPError(req, "Error storing IP address in context: "+err.Error())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func CheckIPBlockedMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := GetLoggerFromContext(req.Context())
		requestIP, err := GetRequestIPFromContext(req.Context())
		if err != nil {
			log.HTTPWarning(req, "Error retrieving IP address from context in CheckIPBlockedMiddleware: "+err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if config.IsIPBlocked(requestIP) {
			log.HTTPWarning(req, "Blocked IP attempted request")
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, req)
	})
}

func WebEndpointConfigMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := GetLoggerFromContext(req.Context())
		endpointConfig, err := config.GetWebEndpointConfig(req.URL.Path)
		if err != nil {
			log.HTTPWarning(req, "Error getting endpoint config for WebEndpointConfigMiddleware: "+err.Error())
			WriteJsonError(w, http.StatusNotFound)
			return
		}
		ctx, err := withWebEndpointConfig(req.Context(), endpointConfig)
		if err != nil {
			log.HTTPError(req, "Error storing endpoint config for WebEndpointConfigMiddleware in context: "+err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func TLSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := GetLoggerFromContext(req.Context())
		endpointConfig, ok := GetWebEndpointConfigFromContext(req.Context())
		if !ok {
			log.HTTPWarning(req, "Error getting endpoint config in TLS middleware")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if endpointConfig.TLSRequired != nil && !*endpointConfig.TLSRequired {
			next.ServeHTTP(w, req)
			return
		}

		if req.TLS == nil || !req.TLS.HandshakeComplete {
			log.HTTPWarning(req, "TLS handshake failed for client, missing or incomplete TLS connection state")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		if req.TLS.Version < tls.VersionTLS13 {
			log.HTTPWarning(req, "Rejected connection with weak TLS version")
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
			log.HTTPWarning(req, "Rejected connection with weak cipher suite")
			WriteJsonError(w, http.StatusUpgradeRequired)
			return
		}

		next.ServeHTTP(w, req)
	})
}

func CheckHttpVersionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := GetLoggerFromContext(req.Context())
		endpointConfig, ok := GetWebEndpointConfigFromContext(req.Context())
		if !ok {
			log.HTTPWarning(req, "Error getting endpoint config in HTTP version middleware")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		endpointConfigMajorVersion, _, ok := http.ParseHTTPVersion(endpointConfig.HTTPVersion)
		if !ok {
			log.HTTPWarning(req, "Invalid HTTP version in endpoint config: "+endpointConfig.HTTPVersion)
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		if endpointConfigMajorVersion == 1 && req.ProtoMajor >= 1 { // HTTP/1.x
			next.ServeHTTP(w, req)
			return
		} else if endpointConfigMajorVersion == 2 && req.ProtoMajor == 2 { // HTTP/2
			next.ServeHTTP(w, req)
			return
		}

		log.HTTPWarning(req, "Unsupported HTTP version: HTTP/"+strconv.Itoa(req.ProtoMajor)+"."+strconv.Itoa(req.ProtoMinor)+" < "+endpointConfig.HTTPVersion)
		w.Header().Set("Upgrade", "HTTP/2")
		WriteJsonError(w, http.StatusUpgradeRequired)
	})
}

func AllowIPRangeMiddleware(trafficSource string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			log := GetLoggerFromContext(req.Context())
			if strings.TrimSpace(trafficSource) == "" {
				log.HTTPWarning(req, "No traffic source specified for AllowIPRangeMiddleware")
				WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			requestIP, err := GetRequestIPFromContext(req.Context())
			if err != nil {
				log.HTTPWarning(req, "Error retrieving IP address from context in AllowIPRangeMiddleware")
				WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			allowed, err := config.IsIPAllowed(trafficSource, requestIP)
			if err != nil {
				log.HTTPError(req, "Error checking if IP is allowed: "+err.Error())
				WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			if !allowed {
				log.HTTPWarning(req, "IP address not in allowed range: "+requestIP.String())
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
			log := GetLoggerFromContext(req.Context())
			requestIP, err := GetRequestIPFromContext(req.Context())
			if err != nil {
				log.HTTPWarning(req, "Error retrieving IP address from context in RateLimitMiddleware")
				WriteJsonError(w, http.StatusInternalServerError)
				return
			}

			// IsClientRateLimited assigns a rate limiter to the client IP if not already present
			limited, retryAfter := config.IsClientRateLimited(rateType, requestIP)
			if limited {
				log.HTTPDebug(req, "Client is rate limited: (retry after "+retryAfter.String()+")")
				WriteJsonError(w, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, req)
		})
	}
}

func APITimeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := GetLoggerFromContext(req.Context())
		apiTimeout, err := config.GetRequestTimeout("api")
		if err != nil {
			log.HTTPError(req, "Failed to get API timeout from config: "+err.Error())
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
		log := GetLoggerFromContext(req.Context())
		fileTimeout, err := config.GetRequestTimeout("file")
		if err != nil {
			log.HTTPError(req, "Failed to get file server timeout from config: "+err.Error())
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
		log := GetLoggerFromContext(req.Context())

		// Check method
		validMethods := map[string]bool{
			http.MethodOptions: true,
			http.MethodGet:     true,
			http.MethodPost:    true,
			http.MethodPut:     true,
			http.MethodDelete:  true,
		}
		if !validMethods[req.Method] {
			log.HTTPWarning(req, "Invalid request method")
			WriteJsonError(w, http.StatusMethodNotAllowed)
			return
		}

		endpointConfig, ok := GetWebEndpointConfigFromContext(req.Context())
		if !ok {
			log.HTTPWarning(req, "Error getting endpoint config in HTTP method middleware")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		if !slices.Contains(endpointConfig.AllowedMethods, req.Method) {
			log.HTTPInfo(req, "Method is not allowed for endpoint")
			WriteJsonError(w, http.StatusMethodNotAllowed)
			return
		}

		next.ServeHTTP(w, req)
	})
}

func CheckValidURLMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := GetLoggerFromContext(req.Context())

		// URL length
		if len(req.URL.RequestURI()) > 2048 {
			log.HTTPWarning(req, "Request URL length exceeds character limit: "+fmt.Sprintf("%d", len(req.URL.RequestURI()))+"/2048")
			WriteJsonError(w, http.StatusRequestURITooLong)
			return
		}

		// URL length
		if len(req.URL.RequestURI()) > 2048 {
			log.HTTPWarning(req, "Request URL length exceeds character limit: "+fmt.Sprintf("%d", len(req.URL.RequestURI()))+"/2048")
			WriteJsonError(w, http.StatusRequestURITooLong)
			return
		}

		// URL path
		cleanPath, err := validateAndCleanURLPath(req.URL.Path)
		if err != nil {
			log.HTTPWarning(req, "Invalid URL path: "+err.Error())
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		// Validate query parameters (even if empty)
		if err := validateQueryParams(req.URL.Query()); err != nil {
			log.HTTPWarning(req, "Invalid URL query parameters: "+err.Error())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Check RawQuery for null bytes and CRLF. req.URL.Query() and url.Parse() may be empty even if RawQuery is not.
		if strings.Contains(req.URL.RawQuery, "\x00") {
			log.HTTPWarning(req, "Null byte detected in raw query string")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if strings.ContainsAny(req.URL.RawQuery, "\r\n") {
			log.HTTPWarning(req, "CRLF characters detected in raw query string")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// MiddlewareChain context updates
		ctx := req.Context()
		// Store clean path in context (to be used later on)
		ctx, err = withRequestPath(ctx, cleanPath)
		if err != nil {
			log.HTTPError(req, "Error storing path in context: "+err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		// Store raw query in context, even if empty (to be used later on)
		queries := req.URL.Query()
		ctx, err = withRequestQuery(ctx, &queries)
		if err != nil {
			log.HTTPError(req, "Error storing query in context: "+err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func CheckHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := GetLoggerFromContext(req.Context())

		for headerKey, headerValues := range req.Header {
			// Check header key length
			if len(headerKey) > 255 {
				log.HTTPWarning(req, "Header key too long: "+fmt.Sprintf("%d", len(headerKey))+"/255 bytes")
				WriteJsonError(w, http.StatusBadRequest)
				return
			}

			// Block disallowed characters in header keys
			if strings.ContainsAny(headerKey, disallowedHeaderChars) {
				log.HTTPWarning(req, "Disallowed characters in header key")
				WriteJsonError(w, http.StatusBadRequest)
				return
			}

			// Header values
			for _, headerValue := range headerValues {
				if strings.ContainsAny(headerValue, disallowedHeaderChars) {
					log.HTTPWarning(req, "Disallowed characters in header value")
					WriteJsonError(w, http.StatusBadRequest)
					return
				}
				if len(headerValue) > 8192 {
					log.HTTPWarning(req, "Header value too long for '"+headerKey+"' "+fmt.Sprintf("%.2f", float64(len(headerValue))/1024)+" KB")
					WriteJsonError(w, http.StatusBadRequest)
					return
				}
			}
		}

		// Required headers
		// Host header (required)
		host := req.Host
		if strings.TrimSpace(host) == "" {
			log.HTTPWarning(req, "Request is missing 'Host' header")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if len(host) > 255 {
			log.HTTPWarning(req, "Host header is too long: "+fmt.Sprintf("%d bytes", len(host)))
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		// Block dangerous characters in Host header (already checked \x00\r\n above)
		if strings.ContainsAny(host, " <>\"'") {
			log.HTTPWarning(req, "Invalid characters in Host header")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// User-Agent header (required)
		userAgent := strings.TrimSpace(req.Header.Get("User-Agent"))
		if userAgent == "" {
			log.HTTPWarning(req, "Request is missing 'User-Agent' header")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if len(userAgent) > 512 {
			log.HTTPWarning(req, "User-Agent header is too long: "+fmt.Sprintf("%d bytes", len(userAgent)))
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Content-Type (required for POST/PUT)
		contentType := strings.TrimSpace(req.Header.Get("Content-Type"))
		if len(contentType) > 256 {
			log.HTTPWarning(req, "Content-Type header is too long: "+fmt.Sprintf("%d bytes", len(contentType)))
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if req.Method == http.MethodPost || req.Method == http.MethodPut {
			if contentType == "" {
				log.HTTPWarning(req, "Missing Content-Type header for POST/PUT request")
				WriteJsonError(w, http.StatusBadRequest)
				return
			}
			if contentType != "application/x-www-form-urlencoded" && contentType != "application/json" && !strings.HasPrefix(contentType, "multipart/form-data") {
				log.HTTPWarning(req, "Invalid Content-Type header: "+contentType)
				WriteJsonError(w, http.StatusUnsupportedMediaType)
				return
			}
		}

		// Optional headers: Validate if present
		// Cookie (optional, validate length if present)
		cookieHeader := strings.TrimSpace(req.Header.Get("Cookie"))
		if len(cookieHeader) > 4096 {
			log.HTTPWarning(req, "Cookie header is too large: "+fmt.Sprintf("%.2f KB", float64(len(cookieHeader))/1024))
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Authorization (optional, validate format if present)
		authorization := strings.TrimSpace(req.Header.Get("Authorization"))
		if authorization != "" {
			if len(authorization) > 8192 {
				log.HTTPWarning(req, "Authorization header is too long: "+fmt.Sprintf("%.2f KB", float64(len(authorization))/1024))
				WriteJsonError(w, http.StatusBadRequest)
				return
			}
			// Must start with Bearer or Basic
			if !strings.HasPrefix(authorization, "Bearer ") && !strings.HasPrefix(authorization, "Basic ") {
				log.HTTPWarning(req, "Invalid Authorization format (missing Basic/Bearer prefix)")
				WriteJsonError(w, http.StatusBadRequest)
				return
			}
		}

		// Connection header (disallowed, not allowed in HTTP/2)
		if req.ProtoMajor == 2 && req.Header.Get("Connection") != "" {
			log.HTTPWarning(req, "Connection header disallowed in HTTP/2 request")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Transfer-Encoding (optional, check for request smuggling)
		transferEncoding := req.Header.Get("Transfer-Encoding")
		if transferEncoding != "" && transferEncoding != "chunked" {
			log.HTTPWarning(req, "Suspicious Transfer-Encoding header: '"+transferEncoding+"'")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Referer (optional, but validate length)
		referer := strings.TrimSpace(req.Header.Get("Referer"))
		if len(referer) > 2048 {
			log.HTTPWarning(req, "Referer header is too long: "+fmt.Sprintf("%.2f KB", float64(len(referer))/1024))
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Origin (if using CORS, validate length)
		origin := strings.TrimSpace(req.Header.Get("Origin"))
		if len(origin) > 2048 {
			log.HTTPWarning(req, "Origin header is too long: "+fmt.Sprintf("%.2f KB", float64(len(origin))/1024))
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Suspicious headers
		if xForwardedFor := strings.TrimSpace(req.Header.Get("X-Forwarded-For")); xForwardedFor != "" {
			log.HTTPWarning(req, "X-Forwarded-For header present (possible proxy bypass): "+xForwardedFor)
			// Log and don't return
		}

		if xRealIP := strings.TrimSpace(req.Header.Get("X-Real-IP")); xRealIP != "" {
			log.HTTPWarning(req, "X-Real-IP header present (possible IP spoofing): "+xRealIP)
			// Log and don't return
		}

		next.ServeHTTP(w, req)
	})
}

func SetHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		log := GetLoggerFromContext(ctx)

		// SERVER-SIDE CORS CHECKS
		// Get web server IP for CORS
		// _, httpsServerIP, err := config.GetWebServerIPs()
		// if err != nil || strings.TrimSpace(httpsServerIP) == "" {
		// 	log.Error("Cannot get web server IP for CORS: " + err.Error())
		// 	WriteJsonError(w, http.StatusInternalServerError)
		// 	return
		// }
		// Check CORS policy
		cors := http.NewCrossOriginProtection()
		// cors.AddTrustedOrigin("https://" + httpsServerIP + ":1411")
		if err := cors.Check(req); err != nil {
			log.HTTPWarning(req, "Request blocked because it violates CORS policy: "+err.Error())
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		// OPTIONS preflight request handling
		if req.Method == http.MethodOptions {
			// w.Header().Set("Access-Control-Allow-Origin", "https://"+httpsServerIP+":1411")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400") // Cache preflight for 24 hours
			w.Header().Set("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// CORS policy
		// w.Header().Set("Access-Control-Allow-Origin", "https://"+httpsServerIP+":1411")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Vary", "Origin")

		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload") // 2 years
		w.Header().Set("Referrer-Policy", "no-referrer")
		nonce, err := generateNonce(24)
		if err != nil {
			log.Error("Cannot generate CSP nonce: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
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
		ctx := req.Context()
		log := GetLoggerFromContext(ctx)
		pathRequested, err := GetRequestPathFromContext(ctx)
		if err != nil {
			log.HTTPError(req, "Error retrieving URL path from context in AllowedFilesMiddleware")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		fileRequested := filepath.Base(pathRequested)
		endpointConfig, err := config.GetWebEndpointConfig(pathRequested)
		if err != nil {
			log.HTTPWarning(req, "Error getting endpoint config in AllowedFilesMiddleware "+err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		endpointFilePath, err := config.GetWebEndpointFilePath(endpointConfig)
		if err != nil {
			log.HTTPWarning(req, "No file path configured for endpoint in AllowedFilesMiddleware: "+err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		endpointType, err := config.GetWebEndpointType(endpointConfig)
		if err != nil || endpointType == "" {
			log.HTTPWarning(req, "No valid endpoint config for URL in AllowedFilesMiddleware: "+pathRequested+" "+err.Error())
			WriteJsonError(w, http.StatusNotFound)
			return
		}
		if endpointType != "api" {
			filePath, err := config.GetWebEndpointFilePath(endpointConfig)
			if err != nil || strings.TrimSpace(filePath) == "" {
				log.HTTPWarning(req, "No file path in context configured for AllowedFilesMiddleware: "+err.Error())
				WriteJsonError(w, http.StatusNotFound)
				return
			}
		}

		resolvedPath, err := filepath.EvalSymlinks(endpointFilePath)
		if err != nil || resolvedPath != endpointFilePath {
			log.HTTPError(req, "Error resolving symlink in AllowedFilesMiddleware: "+err.Error())
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		if resolvedPath != endpointFilePath {
			log.HTTPError(req, "Resolved path does not match full path in AllowedFilesMiddleware: "+resolvedPath+" -> "+endpointFilePath)
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
			log.HTTPError(req, "Metadata is nil for file: "+endpointFilePath)
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if metadata.Size() <= 0 {
			log.HTTPWarning(req, "Attempt to access empty file in AllowedFilesMiddleware: "+fileRequested)
			WriteJsonError(w, http.StatusNoContent)
			return
		}
		if metadata.IsDir() {
			log.HTTPWarning(req, "Attempt to access directory as file in AllowedFilesMiddleware: "+fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		fileMode := metadata.Mode()
		if fileMode&os.ModeSymlink != 0 {
			log.HTTPWarning(req, "Attempt to access symbolic link in AllowedFilesMiddleware: "+fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if fileMode&os.ModeDevice != 0 {
			log.HTTPWarning(req, "Attempt to access device file in AllowedFilesMiddleware: "+fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if fileMode&os.ModeNamedPipe != 0 {
			log.HTTPWarning(req, "Attempt to access named pipe in AllowedFilesMiddleware: "+fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if fileMode&os.ModeSocket != 0 {
			log.HTTPWarning(req, "Attempt to access socket file in AllowedFilesMiddleware: "+fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if fileMode&os.ModeCharDevice != 0 {
			log.HTTPWarning(req, "Attempt to access character device file in AllowedFilesMiddleware: "+fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if fileMode&os.ModeIrregular != 0 {
			log.HTTPWarning(req, "Attempt to access irregular file in AllowedFilesMiddleware: "+fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if !fileMode.IsRegular() {
			log.HTTPWarning(req, "Attempt to access non-regular file in AllowedFilesMiddleware: "+fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		ctx, err = withRequestFile(ctx, resolvedPath)
		if err != nil {
			log.HTTPWarning(req, "Error storing file request in context: "+err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func CookieAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		log := GetLoggerFromContext(ctx)
		requestIP, err := GetRequestIPFromContext(ctx)
		if err != nil {
			log.HTTPWarning(req, "Error retrieving IP address from context: "+err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		requestPath, err := GetRequestPathFromContext(ctx)
		if err != nil {
			log.HTTPWarning(req, "Error retrieving URL from context: "+err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		uitSessionIDCookie, sessionErr := req.Cookie("uit_session_id")
		uitBasicCookie, basicErr := req.Cookie("uit_basic_token")
		uitBearerCookie, bearerErr := req.Cookie("uit_bearer_token")
		uitCSRFCookie, _ := req.Cookie("uit_csrf_token")

		if sessionErr != nil || basicErr != nil || bearerErr != nil {
			if sessionErr != nil && sessionErr != http.ErrNoCookie {
				log.HTTPWarning(req, "Error retrieving UIT auth cookies: "+sessionErr.Error())
			}

			// IP authentication for LAN IPs
			allowedIPs, err := config.GetAllowedLANIPs()
			if err != nil {
				log.HTTPError(req, "Error getting allowed LAN IPs: "+err.Error())
				WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			for _, allowedIP := range allowedIPs {
				if allowedIP.Contains(requestIP) {
					next.ServeHTTP(w, req)
					return
				}
			}
			log.HTTPInfo(req, "No auth cookies found in request: sessionID error: "+fmt.Sprintf("%v", sessionErr)+", basic cookie error: "+fmt.Sprintf("%v", basicErr)+", bearer cookie error: "+fmt.Sprintf("%v", bearerErr))
			// WriteJsonError(w, http.StatusUnauthorized)
			http.Redirect(w, req, "/login", http.StatusSeeOther)
			return
		}

		config.ClearExpiredAuthSessions()

		sessionValid, sessionExists, err := config.CheckAuthSessionExists(uitSessionIDCookie.Value, requestIP, uitBasicCookie.Value, uitBearerCookie.Value, uitCSRFCookie.Value)
		if err != nil {
			log.HTTPError(req, "Error validating auth session: "+err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		if sessionValid && sessionExists && !strings.HasSuffix(requestPath, "/logout") {
			sessionIDCookie, basicCookie, bearerCookie, csrfCookie := GetAuthCookiesForResponse(uitSessionIDCookie.Value, uitBasicCookie.Value, uitBearerCookie.Value, uitCSRFCookie.Value, 20*time.Minute)
			http.SetCookie(w, sessionIDCookie)
			http.SetCookie(w, basicCookie)
			http.SetCookie(w, bearerCookie)
			http.SetCookie(w, csrfCookie)

			sessionExtended, err := config.ExtendAuthSession(uitSessionIDCookie.Value)
			if err != nil {
				log.HTTPError(req, "Error extending auth session: "+err.Error())
				WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			if sessionExtended {
				log.HTTPDebug(req, "Auth session extended")
				next.ServeHTTP(w, req)
			} else {
				log.HTTPDebug(req, "Auth session not found or expired when attempting to extend session")
			}
		} else if sessionExists && strings.TrimSpace(requestPath) == "/logout" {
			log.Debug("Logging out user and deleting auth session: " + requestIP.String())
			config.DeleteAuthSession(uitSessionIDCookie.Value)
			sessionCount := config.RefreshAndGetAuthSessionCount()
			log.Info("Auth session deleted: " + requestIP.String() + " (" + strconv.Itoa(int(sessionCount)) + " session(s))")
			http.Redirect(w, req, "/login", http.StatusSeeOther)
			return
		} else {
			log.HTTPInfo(req, "No valid authentication methods found for request")
			// WriteJsonError(w, FormatHttpError("Unauthorized"), http.StatusUnauthorized)
			http.Redirect(w, req, "/login", http.StatusSeeOther)
			return
		}
	})
}
