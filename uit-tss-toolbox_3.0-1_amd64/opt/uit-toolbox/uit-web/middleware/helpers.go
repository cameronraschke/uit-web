package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"uit-toolbox/config"
	"uit-toolbox/types"
)

type ctxClientIPKey struct{}
type ctxPathRequestKey struct{}
type ctxQueryRequestKey struct{}
type ctxFileRequestKey struct{}
type ctxRequestUUIDKey struct{}
type ctxRequestEndpointKey struct{}
type ctxRequestLoggerKey struct{}
type ctxNonceKey struct{}

type ReturnedJsonToken struct {
	Token string  `json:"token"`
	TTL   float64 `json:"ttl"`
	Valid bool    `json:"valid"`
}

type JsonError struct {
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

const (
	disallowedQueryChars  = "\x00\r\n<>`:"
	disallowedHeaderChars = "\x00\r\n"
	// Block: space, brackets, quotes, pipe, backslash, star, dollar, percent, hash, question, tilde, colon, semicolon, braces, parenthesis, caret, ampersand, null, CRLF
	disallowedPathChars = " <>\"'`|\\*$%#?~:;{}[]()^&\x00\r\n"
	maxQueryKeyLen      = 128
	maxQueryValueLen    = 512
	maxQueryParams      = 64
	minURLPathLen       = 1
	maxURLPathLen       = 255
)

var (
	disallowedFileExtensions = []string{
		".tmp", ".bak", ".swp",
	}
)

var (
	clientIPKey        ctxClientIPKey
	pathRequestKey     ctxPathRequestKey
	queryRequestKey    ctxQueryRequestKey
	fileRequestKey     ctxFileRequestKey
	requestEndpointKey ctxRequestEndpointKey
	requestUUIDKey     ctxRequestUUIDKey
	nonceKey           ctxNonceKey
	loggerKey          ctxRequestLoggerKey

	allowedQueryKeyRegex = regexp.MustCompile(`^[A-Za-z0-9._\-]+$`)
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

// Apply the middleware chain to the final handler
func (chain MiddlewareChain) Then(finalHandler http.Handler) http.Handler {
	for i := len(chain.middlewares) - 1; i >= 0; i-- {
		finalHandler = chain.middlewares[i](finalHandler)
	}
	return finalHandler
}

// Apply the middleware chain to a handler function
func (chain MiddlewareChain) ThenFunc(finalHandlerFunc http.HandlerFunc) http.Handler {
	return chain.Then(finalHandlerFunc)
}

func WriteJson(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteJsonError(w http.ResponseWriter, httpStatusCode int) {
	WriteJsonErrorCustomMessage(w, httpStatusCode, http.StatusText(httpStatusCode))
}

func WriteJsonErrorCustomMessage(w http.ResponseWriter, httpStatusCode int, customMessage string) {
	if httpStatusCode <= 0 {
		httpStatusCode = http.StatusInternalServerError
	}
	responseController := http.NewResponseController(w)
	if responseController != nil {
		_ = responseController.SetWriteDeadline(time.Now().Add(10 * time.Second))
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatusCode)
	jsonStruct := &JsonError{
		ErrorCode:    httpStatusCode,
		ErrorMessage: customMessage,
	}

	err := json.NewEncoder(w).Encode(jsonStruct)
	if err != nil {
		return
	}

	_ = responseController.Flush()
}

func WritePlainTextResponse(w http.ResponseWriter, message string) {
	responseController := http.NewResponseController(w)
	if responseController != nil {
		_ = responseController.SetWriteDeadline(time.Now().Add(10 * time.Second))
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_, err := w.Write([]byte(message))
	if err != nil {
		return
	}

	_ = responseController.Flush()
}

func generateNonce(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func withNonce(ctx context.Context, nonce string) (context.Context, error) {
	if strings.TrimSpace(nonce) == "" {
		return ctx, errors.New("empty nonce")
	}
	return context.WithValue(ctx, nonceKey, nonce), nil
}

func GetNonceFromContext(ctx context.Context) (nonce string, ok bool) {
	nonce, ok = ctx.Value(nonceKey).(string)
	return nonce, ok
}

func withWebEndpointConfig(ctx context.Context, endpoint *config.WebEndpointConfig) (context.Context, error) {
	if ctx == nil {
		return ctx, errors.New("nil context in withWebEndpointConfig")
	}
	if endpoint == nil {
		return ctx, errors.New("nil endpoint config in withWebEndpointConfig")
	}
	return context.WithValue(ctx, requestEndpointKey, *endpoint), nil
}
func GetWebEndpointConfigFromContext(ctx context.Context) (endpoint config.WebEndpointConfig, err error) {
	endpoint, ok := ctx.Value(requestEndpointKey).(config.WebEndpointConfig)
	if !ok {
		return config.WebEndpointConfig{}, fmt.Errorf("web endpoint config not found in context")
	}
	return endpoint, nil
}

func withClientIP(ctx context.Context, ip netip.Addr) (context.Context, error) {
	if err := types.ValidateIPAddress(&ip); err != nil {
		return ctx, fmt.Errorf("IP address stored in context is invalid: %w", err)
	}
	// Use validated IP address here from checkValidIP
	return context.WithValue(ctx, clientIPKey, ip), nil
}
func GetRequestIPFromContext(ctx context.Context) (ipAddr netip.Addr, err error) {
	ip, ok := ctx.Value(clientIPKey).(netip.Addr)
	if !ok {
		return netip.Addr{}, fmt.Errorf("IP address not found in context")
	}
	if err := types.ValidateIPAddress(&ip); err != nil {
		return netip.Addr{}, fmt.Errorf("IP address stored in context is invalid: %w", err)
	}
	return ip, nil
}

func withRequestPath(ctx context.Context, reqPath string) (context.Context, error) {
	if strings.TrimSpace(reqPath) == "" {
		return ctx, errors.New("empty request path")
	}
	return context.WithValue(ctx, pathRequestKey, reqPath), nil
}
func GetRequestPathFromContext(ctx context.Context) (reqPath string, err error) {
	p, ok := ctx.Value(pathRequestKey).(string)
	if !ok {
		return "", fmt.Errorf("URL path not found in context")
	}

	if strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("invalid/empty URL path stored in context")
	}

	return p, nil
}

func GetStrQuery(queries url.Values, key string) *string {
	trimmedKey := strings.TrimSpace(key)
	if len(queries) == 0 || trimmedKey == "" {
		return nil
	}
	val := strings.TrimSpace(queries.Get(trimmedKey))
	if val == "" {
		return nil
	}
	return &val
}
func GetInt64Query(queries url.Values, key string) *int64 {
	strVal := GetStrQuery(queries, key)
	if strVal == nil {
		return nil
	}
	intVal, err := strconv.ParseInt(*strVal, 10, 64)
	if err != nil {
		return nil
	}
	return &intVal
}
func GetBoolQuery(queries url.Values, key string) *bool {
	strVal := GetStrQuery(queries, key)
	if strVal == nil {
		return nil
	}
	boolVal, err := strconv.ParseBool(*strVal)
	if err != nil {
		return nil
	}
	return &boolVal
}

func withRequestFile(ctx context.Context, file string) (context.Context, error) {
	// if strings.TrimSpace(file) == "" {
	// 	return ctx, errors.New("empty request file")
	// }
	return context.WithValue(ctx, fileRequestKey, file), nil
}
func GetRequestFileFromContext(ctx context.Context) (file string, err error) {
	file, ok := ctx.Value(fileRequestKey).(string)
	if !ok {
		return "", fmt.Errorf("file not found in context")
	}
	return file, nil
}

func withRequestUUID(ctx context.Context, uuid string) (context.Context, error) {
	if strings.TrimSpace(uuid) == "" {
		return ctx, errors.New("empty request UUID")
	}
	return context.WithValue(ctx, requestUUIDKey, uuid), nil
}
func GetRequestUUIDFromContext(ctx context.Context) (uuid string, err error) {
	uuid, ok := ctx.Value(requestUUIDKey).(string)
	if !ok {
		return "", fmt.Errorf("UUID not found in context")
	}
	return uuid, nil
}

func withLogger(ctx context.Context, logger *slog.Logger) (context.Context, error) {
	if logger == nil {
		return ctx, errors.New("nil logger")
	}
	return context.WithValue(ctx, loggerKey, logger), nil
}
func GetLoggerFromContext(ctx context.Context) *slog.Logger {
	log, ok := ctx.Value(loggerKey).(*slog.Logger)
	if !ok {
		log = config.GetLogger()
	}
	return log
}

func UpdateAndGetAuthSession(requestAuthSession *types.AuthSession, extendTTL bool) (*types.AuthSession, error) {
	if requestAuthSession == nil {
		return nil, errors.New("nil auth session provided to UpdateAndGetAuthSession")
	}

	curTime := time.Now()
	if requestAuthSession.SessionID == "" {
		return nil, errors.New("empty session ID in auth session provided to UpdateAndGetAuthSession")
	}
	sessionTTL := types.AuthSessionTTL
	basicTTL := types.BasicTTL
	if !extendTTL {
		basicTTL = time.Until(requestAuthSession.BasicToken.Expiry)
	}
	bearerTTL := types.BearerTTL
	if !extendTTL {
		bearerTTL = time.Until(requestAuthSession.BearerToken.Expiry)
	}
	csrfTTL := types.CSRFTTL
	if !extendTTL {
		csrfTTL = time.Until(requestAuthSession.CSRFToken.Expiry)
	}

	newAuthSession := new(types.AuthSession)
	mergedAuthSession := *requestAuthSession

	mergedAuthSession.SessionTTL = sessionTTL
	mergedAuthSession.SessionCookie.Expires = curTime.Add(sessionTTL)
	mergedAuthSession.SessionCookie.MaxAge = int(sessionTTL.Seconds())

	mergedAuthSession.BasicToken.Expiry = curTime.Add(basicTTL)
	mergedAuthSession.BasicCookie.Expires = curTime.Add(basicTTL)
	mergedAuthSession.BasicCookie.MaxAge = int(basicTTL.Seconds())

	mergedAuthSession.BearerToken.Expiry = curTime.Add(bearerTTL)
	mergedAuthSession.BearerCookie.Expires = curTime.Add(bearerTTL)
	mergedAuthSession.BearerCookie.MaxAge = int(bearerTTL.Seconds())

	mergedAuthSession.CSRFToken.Expiry = curTime.Add(csrfTTL)
	mergedAuthSession.CSRFCookie.Expires = curTime.Add(csrfTTL)
	mergedAuthSession.CSRFCookie.MaxAge = int(csrfTTL.Seconds())

	*newAuthSession = mergedAuthSession

	config.UpdateAuthSession(requestAuthSession.SessionID, newAuthSession)

	return newAuthSession, nil
}

func ValidateAuthFormInput(username, password string) error {
	usernameRegex := regexp.MustCompile(`^[A-Za-z0-9._-]{3,20}$`)
	passwordRegex := regexp.MustCompile(`^[\x21-\x7E]{8,64}$`)

	username = strings.TrimSpace(username)
	usernameLen := utf8.RuneCountInString(username)
	if usernameLen < 3 || usernameLen > 20 {
		return errors.New("invalid username length")
	}

	password = strings.TrimSpace(password)
	passwordLen := utf8.RuneCountInString(password)
	if passwordLen < 8 || passwordLen > 64 {
		return errors.New("invalid password length")
	}

	if !usernameRegex.MatchString(username) {
		return errors.New("username does not match regex")
	}
	if !passwordRegex.MatchString(password) {
		return errors.New("password does not match regex")
	}

	authStr := username + ":" + password

	// Check for non-printable ASCII characters
	if !types.IsPrintableASCII([]byte(authStr)) {
		return errors.New("credentials contain non-printable ASCII characters")
	}

	return nil
}

// func IsBodyTooLarge(err error) bool {
// 	if err == nil {
// 		return false
// 	}
// 	return  || strings.Contains(err.Error(), "http: request body too large")
// }

func validateQueryParams(query url.Values) error {
	if len(query) == 0 {
		return nil
	}

	if len(query) > maxQueryParams {
		return fmt.Errorf("too many query parameters in URL (%d > %d)", len(query), maxQueryParams)
	}

	for key, values := range query {
		// query keys
		if len(key) == 0 {
			return fmt.Errorf("empty query key not allowed")
		}

		if len(key) > maxQueryKeyLen {
			return fmt.Errorf("query key too long: %d chars", len(key))
		}

		if !allowedQueryKeyRegex.MatchString(key) {
			return fmt.Errorf("query key does not match allowed regex pattern")
		}

		// query values
		for _, value := range values {
			if len(value) > maxQueryValueLen {
				return fmt.Errorf("query value too long: %d > %d chars", len(value), maxQueryValueLen)
			}

			if strings.ContainsAny(value, disallowedQueryChars) {
				return fmt.Errorf("query value contains disallowed characters")
			}
		}
	}

	return nil
}

func validateAndCleanURLPath(rawPath string) (string, error) {
	if len(rawPath) > maxURLPathLen {
		return "", fmt.Errorf("URL path too long: %d/%d chars", len(rawPath), maxURLPathLen)
	}

	trimmedPath := strings.TrimSpace(rawPath)

	if len(trimmedPath) < minURLPathLen {
		return "", fmt.Errorf("URL path too short: %d/%d chars", len(trimmedPath), minURLPathLen)
	}

	if !types.IsASCIIStringPrintable(trimmedPath) {
		return "", fmt.Errorf("URL path contains non-printable or non-ASCII characters")
	}

	if strings.ContainsAny(rawPath, disallowedPathChars) { // Check rawPath here
		return "", fmt.Errorf("URL path contains disallowed characters")
	}

	if !path.IsAbs(trimmedPath) {
		return "", fmt.Errorf("URL path must start with /")
	}

	joinedPath := path.Join("/", trimmedPath) // Ensure path is rooted and cleaned
	cleanPath := path.Clean(joinedPath)

	if cleanPath == "." {
		return "", fmt.Errorf("empty path after cleaning")
	}

	if slices.Contains(disallowedFileExtensions, path.Ext(cleanPath)) {
		return "", fmt.Errorf("disallowed file extension in URL path")
	}

	// validate each path segment
	segments := strings.Split(strings.Trim(cleanPath, "/"), "/")
	for _, segment := range segments {
		if strings.TrimSpace(segment) == "" {
			continue
		}

		if strings.HasPrefix(segment, ".") {
			return "", fmt.Errorf("hidden path segment not allowed (starts with dot)")
		}

		if strings.HasSuffix(segment, ".") {
			return "", fmt.Errorf("invalid path segment (ends with dot)")
		}
	}

	return cleanPath, nil
}

func IsCookieValid(req *http.Request, cookie *http.Cookie) (bool, error) {
	if req == nil || cookie == nil {
		return false, fmt.Errorf("missing expected authentication cookie")
	}
	if err := cookie.Valid(); err != nil {
		return false, fmt.Errorf("invalid authentication cookie format: %w", err)
	}
	if strings.TrimSpace(cookie.Name) == "" || len(cookie.Name) > 255 || !types.IsASCIIStringPrintable(cookie.Name) {
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
	if !types.IsASCIIStringPrintable(cookie.Value) {
		return false, fmt.Errorf("authentication cookie contains invalid characters: %s", cookie.Name)
	}
	return true, nil
}
