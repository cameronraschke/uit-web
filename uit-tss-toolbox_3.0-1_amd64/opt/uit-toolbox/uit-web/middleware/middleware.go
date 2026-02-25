package middleware

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"

	config "uit-toolbox/config"
	"uit-toolbox/types"
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
		justIpAddr, _, _ := net.SplitHostPort(req.RemoteAddr)
		baseLogger := config.GetLogger()
		requestLogger := baseLogger.WithGroup("request").With(
			slog.String("method", req.Method),
			slog.String("url", req.URL.String()),
			slog.String("remote_addr", justIpAddr),
		)
		ctx, err := withLogger(req.Context(), requestLogger)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error storing logger in context in StoreLoggerMiddleware: "+err.Error())
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
					fmt.Fprintln(os.Stderr, "Error getting logger from context in PanicRecoveryMiddleware: logger is nil")
				} else {
					log = log.With(slog.String("func", "PanicRecoveryMiddleware"))
					log.Error(fmt.Sprintf("Panic recovered: %v\n%s", err, string(debug.Stack())))
				}
				WriteJsonError(w, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, req)
	})
}

func LimitRequestSizeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := GetLoggerFromContext(req.Context())
		log = log.With(slog.String("func", "LimitRequestSizeMiddleware"))
		if req.Method == http.MethodPost || req.Method == http.MethodPut {
			log = log.With(slog.Int64("content_length", req.ContentLength))
		}
		if strings.TrimSpace(req.Header.Get("Content-Length")) == "" && (req.Method == http.MethodPost || req.Method == http.MethodPut) {
			log.Info("Content-Length is missing in POST/PUT request")
			WriteJsonError(w, http.StatusLengthRequired)
			return
		}
		appState, err := config.GetAppState()
		if err != nil {
			log.Error("Error retrieving app state: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		_, _, maxSize, err := appState.GetFileUploadDefaultConstraints()
		if err != nil {
			log.Error("Error getting file upload constraints from app state: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if req.ContentLength > maxSize {
			log.Warn("Request content length exceeds limit: " + fmt.Sprintf("%.2fMB", float64(req.ContentLength)/1e6) + "/" + fmt.Sprintf("%.2fMB", float64(maxSize)/1e6))
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
		log = log.With(slog.String("func", "StoreClientIPMiddleware"))
		ipStr, _, err := net.SplitHostPort(req.RemoteAddr)
		if err != nil {
			log.Error("Cannot parse request IP: " + err.Error())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		reqAddr, _, _, err := convertAndCheckIPStr(&ipStr)
		if err != nil {
			log.Warn("Cannot convert request IP: " + err.Error())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// withClientIP parses and casts the IP address to netip.Addr
		ctx, err := withClientIP(req.Context(), *reqAddr)
		if err != nil {
			log.Error("Cannot store request IP in context: " + err.Error())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func CheckIPBlockedMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := GetLoggerFromContext(req.Context())
		log = log.With(slog.String("func", "CheckIPBlockedMiddleware"))
		reqAddr, err := GetRequestIPFromContext(req.Context())
		if err != nil {
			log.Warn("Cannot retrieve request IP from context: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if config.RequestIPBlocked(reqAddr) {
			log.Debug("Request received from blocked IP")
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, req)
	})
}

func WebEndpointConfigMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := GetLoggerFromContext(req.Context())
		log = log.With(slog.String("func", "WebEndpointConfigMiddleware"))
		endpointConfigPtr, err := config.GetWebEndpointConfig(req.URL.Path)
		if err != nil {
			log.Warn("Cannot retrieve endpoint config from context: " + err.Error())
			WriteJsonError(w, http.StatusNotFound)
			return
		}
		ctx, err := withWebEndpointConfig(req.Context(), endpointConfigPtr)
		if err != nil {
			log.Error("Cannot store endpoint config in context: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func TLSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := GetLoggerFromContext(req.Context())
		log = log.With(slog.String("func", "TLSMiddleware"))
		endpointConfig, err := GetWebEndpointConfigFromContext(req.Context())
		if err != nil {
			log.Warn("Cannot retrieve endpoint config from context: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if endpointConfig.TLSRequired != nil && !*endpointConfig.TLSRequired {
			log.Debug("TLS is not required for endpoint, continuing")
			next.ServeHTTP(w, req)
			return
		}

		if req.TLS == nil || !req.TLS.HandshakeComplete {
			log.Warn("Missing or incomplete TLS connection state")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		if req.TLS.Version < tls.VersionTLS13 {
			log = log.With(slog.String("tls_version", tls.VersionName(req.TLS.Version)))
			log.Warn("TLS version is too old")
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
			log = log.With(slog.String("tls_cipher_suite", tls.CipherSuiteName(req.TLS.CipherSuite)))
			log.Warn("Request uses weak cipher suite")
			WriteJsonError(w, http.StatusUpgradeRequired)
			return
		}

		next.ServeHTTP(w, req)
	})
}

func CheckHttpVersionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := GetLoggerFromContext(req.Context())
		log = log.With(slog.String("func", "CheckHttpVersionMiddleware"))
		endpointConfig, err := GetWebEndpointConfigFromContext(req.Context())
		if err != nil {
			log.Warn("Cannot retrieve endpoint config from context: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		httpMajorVersion, httpMinorVersion, ok := http.ParseHTTPVersion(endpointConfig.HTTPVersion)
		if !ok {
			log.Warn("Invalid HTTP version in endpoint config: " + endpointConfig.HTTPVersion)
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		endpointFloatVersion, err := strconv.ParseFloat(strconv.Itoa(httpMajorVersion)+"."+strconv.Itoa(httpMinorVersion), 64)
		if err != nil {
			log.Warn("Cannot parse HTTP version in endpoint config: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		requestFloatVersion, err := strconv.ParseFloat(strconv.Itoa(req.ProtoMajor)+"."+strconv.Itoa(req.ProtoMinor), 64)
		if err != nil {
			log.Warn("Cannot parse HTTP version in request: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		if requestFloatVersion < endpointFloatVersion {
			log.Warn("Unsupported HTTP version in request: HTTP/" + fmt.Sprintf("%.2f", requestFloatVersion) + " < " + fmt.Sprintf("%.2f", endpointFloatVersion))
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
			log := GetLoggerFromContext(req.Context())
			log = log.With(slog.String("func", "AllowIPRangeMiddleware"))
			if strings.TrimSpace(trafficSource) == "" {
				log.Warn("No traffic source specified")
				WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			reqAddr, err := GetRequestIPFromContext(req.Context())
			if err != nil {
				log.Warn("Cannot retrieve IP from context: " + err.Error())
				WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			allowed, err := config.IsIPAllowed(trafficSource, reqAddr)
			if err != nil {
				log.Error("Cannot check if IP is allowed: " + err.Error())
				WriteJsonError(w, http.StatusInternalServerError)
				return
			}
			if !allowed {
				log.Warn("Request IP is not in allowed range: " + reqAddr.String())
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
			log = log.With(slog.String("func", "RateLimitMiddleware"))
			reqIP, err := GetRequestIPFromContext(req.Context())
			if err != nil {
				log.Warn("Cannot retrieve IP from context: " + err.Error())
				WriteJsonError(w, http.StatusInternalServerError)
				return
			}

			// IsClientRateLimited assigns a rate limiter to the client IP if not already present
			limited, retryAfter := config.IsClientRateLimited(rateType, reqIP)
			if limited {
				log.Debug("Client is rate limited", slog.Duration("retry_after", retryAfter))
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
			log.Error("Failed to get API timeout from config: " + err.Error())
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
			log.Error("Failed to get file server timeout from config: " + err.Error())
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
			log.Warn("Invalid request method (HTTPMethodMiddleware): " + req.Method)
			WriteJsonError(w, http.StatusMethodNotAllowed)
			return
		}

		endpointConfig, err := GetWebEndpointConfigFromContext(req.Context())
		if err != nil {
			log.Warn("Error getting endpoint config (HTTPMethodMiddleware): " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		if !slices.Contains(endpointConfig.AllowedMethods, req.Method) {
			log.Info("Method is not allowed for endpoint (HTTPMethodMiddleware): " + req.Method)
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
			log.Warn("Request URL length exceeds character limit: " + fmt.Sprintf("%d", len(req.URL.RequestURI())) + "/2048")
			WriteJsonError(w, http.StatusRequestURITooLong)
			return
		}

		// URL length
		if len(req.URL.RequestURI()) > 2048 {
			log.Warn("Request URL length exceeds character limit: " + fmt.Sprintf("%d", len(req.URL.RequestURI())) + "/2048")
			WriteJsonError(w, http.StatusRequestURITooLong)
			return
		}

		// URL path
		cleanPath, err := validateAndCleanURLPath(req.URL.Path)
		if err != nil {
			log.Warn("Invalid URL path (CheckValidURLMiddleware): " + err.Error())
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		// Validate query parameters (even if empty)
		if err := validateQueryParams(req.URL.Query()); err != nil {
			log.Warn("Invalid URL query parameters (CheckValidURLMiddleware): " + err.Error())
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Check RawQuery for null bytes and CRLF. req.URL.Query() and url.Parse() may be empty even if RawQuery is not.
		if strings.Contains(req.URL.RawQuery, "\x00") {
			log.Warn("Null byte detected in raw query string (CheckValidURLMiddleware)")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if strings.ContainsAny(req.URL.RawQuery, "\r\n") {
			log.Warn("CRLF characters detected in raw query string (CheckValidURLMiddleware)")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// MiddlewareChain context updates
		ctx := req.Context()
		// Store clean path in context (to be used later on)
		ctx, err = withRequestPath(ctx, cleanPath)
		if err != nil {
			log.Error("Error storing path in context (CheckValidURLMiddleware): " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		// Store raw query in context, even if empty (to be used later on)
		queries := req.URL.Query()
		ctx, err = withRequestQuery(ctx, &queries)
		if err != nil {
			log.Error("Error storing query in context (CheckValidURLMiddleware): " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func CheckForRedirectsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := GetLoggerFromContext(req.Context())
		log = log.With(slog.String("func", "CheckForRedirectsMiddleware"))
		endpointConfig, err := config.GetWebEndpointConfig(req.URL.Path)
		if err != nil {
			log.Warn("Error getting endpoint config in CheckForRedirectsMiddleware: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		redirectURL, err := config.GetWebEndpointRedirectURL(endpointConfig)
		if err != nil {
			next.ServeHTTP(w, req)
			return
		}
		log.Info("Redirecting request to: " + redirectURL)
		http.Redirect(w, req, redirectURL, http.StatusFound)
	})
}

func CheckHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log := GetLoggerFromContext(req.Context())

		for headerKey, headerValues := range req.Header {
			// Check header key length
			if len(headerKey) > 255 {
				log.Warn("Header key too long: " + fmt.Sprintf("%d", len(headerKey)) + "/255 bytes")
				WriteJsonError(w, http.StatusBadRequest)
				return
			}

			// Block disallowed characters in header keys
			if strings.ContainsAny(headerKey, disallowedHeaderChars) {
				log.Warn("Disallowed characters in header key (CheckHeadersMiddleware)")
				WriteJsonError(w, http.StatusBadRequest)
				return
			}

			// Header values
			for _, headerValue := range headerValues {
				if strings.ContainsAny(headerValue, disallowedHeaderChars) {
					log.Warn("Disallowed characters in header value")
					WriteJsonError(w, http.StatusBadRequest)
					return
				}
				if len(headerValue) > 8192 {
					log.Warn("Header value too long for '" + headerKey + "' " + fmt.Sprintf("%.2f", float64(len(headerValue))/1024) + " KB")
					WriteJsonError(w, http.StatusBadRequest)
					return
				}
			}
		}

		// Required headers
		// Host header (required)
		host := req.Host
		if strings.TrimSpace(host) == "" {
			log.Warn("Request is missing 'Host' header")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if len(host) > 255 {
			log.Warn("Host header is too long: " + fmt.Sprintf("%d bytes", len(host)))
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		// Block dangerous characters in Host header (already checked \x00\r\n above)
		if strings.ContainsAny(host, " <>\"'") {
			log.Warn("Invalid characters in Host header")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// User-Agent header (required)
		userAgent := strings.TrimSpace(req.Header.Get("User-Agent"))
		if userAgent == "" {
			log.Warn("Request is missing 'User-Agent' header")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if len(userAgent) > 512 {
			log.Warn("User-Agent header is too long: " + fmt.Sprintf("%d bytes", len(userAgent)))
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Content-Type (required for POST/PUT)
		contentType := strings.TrimSpace(req.Header.Get("Content-Type"))
		if len(contentType) > 256 {
			log.Warn("Content-Type header is too long: " + fmt.Sprintf("%d bytes", len(contentType)))
			WriteJsonError(w, http.StatusBadRequest)
			return
		}
		if req.Method == http.MethodPost || req.Method == http.MethodPut {
			if contentType == "" {
				log.Warn("Missing Content-Type header for POST/PUT request")
				WriteJsonError(w, http.StatusBadRequest)
				return
			}
			if contentType != "application/x-www-form-urlencoded" && contentType != "application/json" && !strings.HasPrefix(contentType, "multipart/form-data") {
				log.Warn("Invalid Content-Type header: " + contentType)
				WriteJsonError(w, http.StatusUnsupportedMediaType)
				return
			}
		}

		// Optional headers: Validate if present
		// Cookie (optional, validate length if present)
		cookieHeader := strings.TrimSpace(req.Header.Get("Cookie"))
		if len(cookieHeader) > 4096 {
			log.Warn("Cookie header is too large: " + fmt.Sprintf("%.2f KB", float64(len(cookieHeader))/1024))
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Authorization (optional, validate format if present)
		authorization := strings.TrimSpace(req.Header.Get("Authorization"))
		if authorization != "" {
			if len(authorization) > 8192 {
				log.Warn("Authorization header is too long: " + fmt.Sprintf("%.2f KB", float64(len(authorization))/1024))
				WriteJsonError(w, http.StatusBadRequest)
				return
			}
			// Must start with Bearer or Basic
			if !strings.HasPrefix(authorization, "Bearer ") && !strings.HasPrefix(authorization, "Basic ") {
				log.Warn("Invalid Authorization format (missing Basic/Bearer prefix)")
				WriteJsonError(w, http.StatusBadRequest)
				return
			}
		}

		// Connection header (disallowed, not allowed in HTTP/2)
		if req.ProtoMajor == 2 && req.Header.Get("Connection") != "" {
			log.Warn("Connection header disallowed in HTTP/2 request")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Transfer-Encoding (optional, check for request smuggling)
		transferEncoding := req.Header.Get("Transfer-Encoding")
		if transferEncoding != "" && transferEncoding != "chunked" {
			log.Warn("Suspicious Transfer-Encoding header: '" + transferEncoding + "'")
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Referer (optional, but validate length)
		referer := strings.TrimSpace(req.Header.Get("Referer"))
		if len(referer) > 2048 {
			log.Warn("Referer header is too long: " + fmt.Sprintf("%.2f KB", float64(len(referer))/1024))
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Origin (if using CORS, validate length)
		origin := strings.TrimSpace(req.Header.Get("Origin"))
		if len(origin) > 2048 {
			log.Warn("Origin header is too long: " + fmt.Sprintf("%.2f KB", float64(len(origin))/1024))
			WriteJsonError(w, http.StatusBadRequest)
			return
		}

		// Suspicious headers
		if xForwardedFor := strings.TrimSpace(req.Header.Get("X-Forwarded-For")); xForwardedFor != "" {
			log.Warn("X-Forwarded-For header present (possible proxy bypass): " + xForwardedFor)
			// Log and don't return
		}

		if xRealIP := strings.TrimSpace(req.Header.Get("X-Real-IP")); xRealIP != "" {
			log.Warn("X-Real-IP header present (possible IP spoofing): " + xRealIP)
			// Log and don't return
		}

		next.ServeHTTP(w, req)
	})
}

func SetHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		log := GetLoggerFromContext(ctx)
		log = log.With(slog.String("func", "SetHeadersMiddleware"))

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
			log.Warn("Request blocked because it violates CORS policy: " + err.Error())
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
			log.Error("Error retrieving URL path from context in AllowedFilesMiddleware")
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		fileRequested := filepath.Base(pathRequested)
		endpointConfig, err := config.GetWebEndpointConfig(pathRequested)
		if err != nil {
			log.Warn("Error getting endpoint config in AllowedFilesMiddleware " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		endpointFilePath, err := config.GetWebEndpointFilePath(endpointConfig)
		if err != nil {
			log.Warn("No file path configured for endpoint in AllowedFilesMiddleware: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		endpointType, err := config.GetWebEndpointType(endpointConfig)
		if err != nil || endpointType == "" {
			log.Warn("No valid endpoint config for URL in AllowedFilesMiddleware: " + pathRequested + " " + err.Error())
			WriteJsonError(w, http.StatusNotFound)
			return
		}
		if endpointType != "api" {
			filePath, err := config.GetWebEndpointFilePath(endpointConfig)
			if err != nil || strings.TrimSpace(filePath) == "" {
				log.Warn("No file path in context configured for AllowedFilesMiddleware: " + err.Error())
				WriteJsonError(w, http.StatusNotFound)
				return
			}
		}

		resolvedPath, err := filepath.EvalSymlinks(endpointFilePath)
		if err != nil || resolvedPath != endpointFilePath {
			log.Error("Error resolving symlink in AllowedFilesMiddleware: " + err.Error())
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		if resolvedPath != endpointFilePath {
			log.Error("Resolved path does not match full path in AllowedFilesMiddleware: " + resolvedPath + " -> " + endpointFilePath)
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
		if metadata.Size() <= 0 {
			log.Warn("Attempt to access empty file in AllowedFilesMiddleware: " + fileRequested)
			WriteJsonError(w, http.StatusNoContent)
			return
		}
		if metadata.IsDir() {
			log.Warn("Attempt to access directory as file in AllowedFilesMiddleware: " + fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		fileMode := metadata.Mode()
		if fileMode&os.ModeSymlink != 0 {
			log.Warn("Attempt to access symbolic link in AllowedFilesMiddleware: " + fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if fileMode&os.ModeDevice != 0 {
			log.Warn("Attempt to access device file in AllowedFilesMiddleware: " + fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if fileMode&os.ModeNamedPipe != 0 {
			log.Warn("Attempt to access named pipe in AllowedFilesMiddleware: " + fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if fileMode&os.ModeSocket != 0 {
			log.Warn("Attempt to access socket file in AllowedFilesMiddleware: " + fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if fileMode&os.ModeCharDevice != 0 {
			log.Warn("Attempt to access character device file in AllowedFilesMiddleware: " + fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if fileMode&os.ModeIrregular != 0 {
			log.Warn("Attempt to access irregular file in AllowedFilesMiddleware: " + fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}
		if !fileMode.IsRegular() {
			log.Warn("Attempt to access non-regular file in AllowedFilesMiddleware: " + fileRequested)
			WriteJsonError(w, http.StatusForbidden)
			return
		}

		ctx, err = withRequestFile(ctx, resolvedPath)
		if err != nil {
			log.Warn("Error storing file request in context: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func IsCookieValid(req *http.Request, cookie *http.Cookie) (bool, error) {
	if req == nil || cookie == nil {
		return false, fmt.Errorf("missing expected authentication cookie")
	}
	if err := cookie.Valid(); err != nil {
		return false, fmt.Errorf("invalid authentication cookie format: %w", err)
	}
	if strings.TrimSpace(cookie.Name) == "" || len(cookie.Name) > 255 || !IsASCIIStringPrintable(cookie.Name) {
		return false, fmt.Errorf("invalid authentication cookie name")
	}
	if cookie.Secure && req.TLS == nil {
		return false, fmt.Errorf("secure authentication cookie sent over non-TLS connection: %s", cookie.Name)
	}
	// if cookie.MaxAge <= 0 { // Expire early to allow for creation of new session
	// 	return false, fmt.Errorf("authentication cookie has MaxAge <= 0 seconds: %s", cookie.Name)
	// }
	// if cookie.Expires.Before(time.Now()) {
	// 	return false, fmt.Errorf("authentication cookie has expired: %s", cookie.Name)
	// }
	// if cookie.HttpOnly == false {
	// 	return false, fmt.Errorf("authentication cookie is not HttpOnly: %s", cookie.Name)
	// }
	// if cookie.SameSite != http.SameSiteStrictMode && cookie.SameSite != http.SameSiteLaxMode {
	// 	return false, fmt.Errorf("authentication cookie does not have SameSite=Strict or SameSite=Lax: %s", cookie.Name)
	// }
	if strings.TrimSpace(cookie.Value) == "" || len(cookie.Value) > 4096 {
		return false, fmt.Errorf("authentication cookie value out of range: %s", cookie.Name)
	}
	if !IsASCIIStringPrintable(cookie.Value) {
		return false, fmt.Errorf("authentication cookie contains invalid characters: %s", cookie.Name)
	}
	return true, nil
}

func CookieAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		log := GetLoggerFromContext(ctx)
		log = log.With(slog.String("func", "CookieAuthMiddleware"))
		reqAddr, err := GetRequestIPFromContext(ctx)
		if err != nil {
			log.Warn("Cannot retrieve IP from context: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		requestPath, err := GetRequestPathFromContext(ctx)
		if err != nil {
			log.Warn("Cannot retrieve URL path from context: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		allowedIPs, err := config.GetAllowedLANIPs()
		if err != nil {
			log.Error("Error retrieving allowed LAN IPs for IP auth: " + err.Error())
			WriteJsonError(w, http.StatusInternalServerError)
			return
		}
		for _, allowedIP := range allowedIPs {
			if allowedIP.Contains(reqAddr) {
				next.ServeHTTP(w, req)
				return
			}
		}

		redirectURL := "/login" + "?redirect=" + url.QueryEscape((&url.URL{
			Path:     requestPath,
			RawQuery: req.URL.RawQuery,
		}).String())

		uitSessionIDCookie, sessionErr := req.Cookie("uit_session_id")
		uitBasicCookie, basicErr := req.Cookie("uit_basic_token")
		uitBearerCookie, bearerErr := req.Cookie("uit_bearer_token")
		// uitCSRFCookie, csrfErr := req.Cookie("uit_csrf_token")

		if errors.Is(sessionErr, http.ErrNoCookie) ||
			errors.Is(basicErr, http.ErrNoCookie) ||
			errors.Is(bearerErr, http.ErrNoCookie) {
			/* errors.Is(csrfErr, http.ErrNoCookie) */
			// IP authentication for LAN clients (laptops)
			log.Info("Request is missing required cookies, redirecting")
			http.Redirect(w, req, redirectURL, http.StatusSeeOther)
			return
		}

		// Check cookie issues first
		cookiesToCheck := map[string]*http.Cookie{
			"uit_session_id":   uitSessionIDCookie,
			"uit_basic_token":  uitBasicCookie,
			"uit_bearer_token": uitBearerCookie,
			// "uit_csrf_token":   uitCSRFCookie,
		}
		for cookieName, cookie := range cookiesToCheck {
			cookieValid, err := IsCookieValid(req, cookie)
			if err != nil || !cookieValid {
				log.Warn("Invalid authentication cookie '" + cookieName + "': " + err.Error())
				http.Redirect(w, req, redirectURL, http.StatusSeeOther)
				return
			}
			// log.Debug("Authentication cookie '" + cookieName + "' is valid")
		}

		config.ClearExpiredAuthSessions()

		currentSession, err := config.GetAuthSessionByID(uitSessionIDCookie.Value)
		if err != nil {
			log.Error("Error retrieving auth session: " + err.Error())
			http.Redirect(w, req, redirectURL, http.StatusSeeOther)
			return
		}

		sessionValid, err := config.IsAuthSessionValid(currentSession, reqAddr)
		if err != nil || !sessionValid {
			log.Error("Error validating auth session: " + err.Error())
			http.Redirect(w, req, redirectURL, http.StatusSeeOther)
			return
		}

		switch requestPath {
		case "/logout":
			config.DeleteAuthSession(uitSessionIDCookie.Value)
			log.Info(fmt.Sprintf("Deleted auth session (%s), Session(s) active: %d", uitSessionIDCookie.Value, config.GetAuthSessionCount()))
			// Clear cookies
			http.SetCookie(w, &http.Cookie{
				Name:    "uit_session_id",
				Value:   "",
				Expires: time.Now(),
				MaxAge:  -1,
			})
			http.SetCookie(w, &http.Cookie{
				Name:    "uit_basic_token",
				Value:   "",
				Expires: time.Now(),
				MaxAge:  -1,
			})
			http.SetCookie(w, &http.Cookie{
				Name:    "uit_bearer_token",
				Value:   "",
				Expires: time.Now(),
				MaxAge:  -1,
			})
			http.SetCookie(w, &http.Cookie{
				Name:    "uit_csrf_token",
				Value:   "",
				Expires: time.Now(),
				MaxAge:  -1,
			})

			// Redirect to login page
			log.Info("Auth session deleted: " + reqAddr.String() + ", active session(s): " + strconv.Itoa(int(config.RefreshAndGetAuthSessionCount())))
			redirectURL := "/login" + "?redirect=" + url.QueryEscape(req.URL.RequestURI())
			http.Redirect(w, req, redirectURL, http.StatusSeeOther)
			return
		case "/api/check_auth", "/api/login":
			if req.Method != http.MethodPost {
				log.Warn("Invalid HTTP method for auth check endpoint: " + req.Method)
				WriteJsonError(w, http.StatusMethodNotAllowed)
				return
			}
			// Don't extend session TTL for auth check
			currentSession, err := UpdateAndGetAuthSession(currentSession, false)
			if err != nil || currentSession == nil {
				log.Error("Error generating auth cookies for response: " + err.Error())
				http.Redirect(w, req, redirectURL, http.StatusSeeOther)
				return
			}
			var returnedJson = new(types.AuthStatusResponse)
			returnedJson.Status = "authenticated"
			returnedJson.ExpiresAt = time.Now().Add(currentSession.SessionTTL)
			returnedJson.TTL = currentSession.SessionTTL
			WriteJson(w, http.StatusOK, returnedJson)
			return
		default:
			updatedSession, err := UpdateAndGetAuthSession(currentSession, true)
			if err != nil || updatedSession == nil {
				log.Error("Error generating auth cookies for response: " + err.Error())
				http.Redirect(w, req, redirectURL, http.StatusSeeOther)
				return
			}
			if updatedSession.SessionTTL <= 2*time.Minute {
				log.Debug("Auth session TTL is low (" + updatedSession.SessionTTL.String() + "), sending tokens to client: " + reqAddr.String())
				http.SetCookie(w, updatedSession.SessionCookie)
				http.SetCookie(w, updatedSession.BasicCookie)
				http.SetCookie(w, updatedSession.BearerCookie)
				// http.SetCookie(w, updatedSession.CSRFCookie)
			}
			next.ServeHTTP(w, req)
			return
		}
	})
}
