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
	"unicode"
	"unicode/utf8"

	config "uit-toolbox/config"
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
	if err := validateIPAddress(&ip); err != nil {
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
	if err := validateIPAddress(&ip); err != nil {
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

func withRequestQuery(ctx context.Context, query *url.Values) (context.Context, error) {
	if query == nil {
		return ctx, nil
	}
	return context.WithValue(ctx, queryRequestKey, query), nil
}

func GetRequestQueryFromContext(ctx context.Context) (query *url.Values, err error) {
	q, ok := ctx.Value(queryRequestKey).(*url.Values)
	if !ok {
		return nil, fmt.Errorf("invalid/empty URL query found in context: type assertion failed")
	}
	if q == nil {
		return nil, fmt.Errorf("nil URL query found in context")
	}
	queries := *q

	return &queries, nil
}

func GetStrQuery(q *url.Values, key string) *string {
	s := strings.TrimSpace(q.Get(key))
	if s == "" {
		return nil
	}
	return &s
}
func GetInt64Query(q *url.Values, key string) *int64 {
	s := GetStrQuery(q, key)
	if s == nil {
		return nil
	}
	v, err := strconv.ParseInt(*s, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}
func GetBoolQuery(q *url.Values, key string) *bool {
	s := GetStrQuery(q, key)
	if s == nil {
		return nil
	}
	v, err := strconv.ParseBool(*s)
	if err != nil {
		return nil
	}
	return &v
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

func GetAuthCookiesForResponse(uitSessionIDValue, uitBasicValue, uitBearerValue, uitCSRFValue string, timeout time.Duration) (*http.Cookie, *http.Cookie, *http.Cookie, *http.Cookie) {
	sessionIDCookie := &http.Cookie{
		Name:     "uit_session_id",
		Value:    uitSessionIDValue,
		Path:     "/",
		Expires:  time.Now().Add(timeout),
		MaxAge:   int(timeout.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	basicCookie := &http.Cookie{
		Name:     "uit_basic_token",
		Value:    uitBasicValue,
		Path:     "/",
		Expires:  time.Now().Add(timeout),
		MaxAge:   int(timeout.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	bearerCookie := &http.Cookie{
		Name:     "uit_bearer_token",
		Value:    uitBearerValue,
		Path:     "/",
		Expires:  time.Now().Add(timeout),
		MaxAge:   int(timeout.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	csrfCookie := &http.Cookie{
		Name:     "uit_csrf_token",
		Value:    uitCSRFValue,
		Path:     "/",
		Expires:  time.Now().Add(timeout),
		MaxAge:   int(timeout.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	return sessionIDCookie, basicCookie, bearerCookie, csrfCookie
}

func validateIPAddress(ipAddr *netip.Addr) error {
	if ipAddr == nil {
		return fmt.Errorf("nil IP address")
	}
	if ipAddr.IsUnspecified() || !ipAddr.IsValid() {
		return fmt.Errorf("unspecified or invalid IP address: %s", ipAddr.String())
	}
	if ipAddr.IsInterfaceLocalMulticast() || ipAddr.IsLinkLocalMulticast() || ipAddr.IsMulticast() {
		return fmt.Errorf("multicast IP address not allowed: %s", ipAddr.String())
	}
	return nil
}

func convertAndCheckIPStr(ipPtr *string) (ipAddr *netip.Addr, isLoopback bool, isLocal bool, err error) {
	if ipPtr == nil {
		return nil, false, false, fmt.Errorf("nil IP address")
	}

	ipStr := strings.TrimSpace(*ipPtr)
	if ipStr == "" {
		return nil, false, false, fmt.Errorf("empty IP address")
	}

	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		return nil, false, false, fmt.Errorf("failed to parse IP address: %w", err)
	}

	if err := validateIPAddress(&ip); err != nil {
		return nil, false, false, fmt.Errorf("invalid IP address: %w", err)
	}

	return &ip, ip.IsLoopback(), ip.IsPrivate(), nil
}

func IsPrintableASCII(b []byte) bool {
	for i := range b {
		char := b[i]
		if char < 0x20 || char > 0x7E { // Space (0x20) to tilde (0x7E)
			return false
		}
	}
	return true
}

func IsASCIIStringPrintable(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, char := range s {
		if char < 32 || char > 126 {
			return false
		}
	}
	return true
}

func IsPrintableUnicodeString(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, char := range s {
		if !unicode.IsPrint(char) && !unicode.IsSpace(char) {
			return false
		}
	}
	return true
}

func IsNumericAscii(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	if !utf8.Valid(b) {
		return false
	}
	for i := range b {
		char := b[i]
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func CountDigits(n int64) int {
	if n == 0 {
		return 1
	}
	count := 0
	for n != 0 {
		n /= 10
		count++
	}
	return count
}

func IsSHA256String(s string) error {
	sha256HexRegex := regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
	s = strings.TrimSpace(s)
	if !sha256HexRegex.MatchString(s) {
		return errors.New("invalid digest")
	}
	return nil
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
	if !IsPrintableASCII([]byte(authStr)) {
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
	if len(rawPath) > 255 {
		return "", fmt.Errorf("URL path too long: %d/%d chars", len(rawPath), 255)
	}

	rawPath = strings.TrimSpace(rawPath)

	if rawPath == "" {
		return "", fmt.Errorf("URL path is empty")
	}

	// ASCII printable characters only (32-126)
	// This also implicitly checks for valid UTF-8
	if !IsASCIIStringPrintable(rawPath) {
		return "", fmt.Errorf("URL path contains non-printable or non-ASCII characters")
	}

	if strings.ContainsAny(rawPath, disallowedPathChars) {
		return "", fmt.Errorf("URL path contains disallowed characters")
	}

	if !path.IsAbs(rawPath) {
		return "", fmt.Errorf("URL path must start with /")
	}

	cleanPath := path.Clean(rawPath)

	if cleanPath == "." {
		return "", fmt.Errorf("empty path after cleaning")
	}

	if slices.Contains(disallowedFileExtensions, path.Ext(cleanPath)) {
		return "", fmt.Errorf("disallowed file extension in URL path")
	}

	// validate each path segment
	segments := strings.Split(strings.Trim(cleanPath, "/"), "/")
	for _, segment := range segments {
		if segment == "" {
			continue
		}

		if err := validatePathSegment(segment); err != nil {
			return "", fmt.Errorf("invalid path segment '%s': %w", segment, err)
		}
	}

	return cleanPath, nil
}

func validatePathSegment(segment string) error {
	// Reject hidden files/directories (both prefix and suffix checks)
	if strings.HasPrefix(segment, ".") {
		return fmt.Errorf("hidden file/directory not allowed (starts with dot)")
	}

	if strings.HasSuffix(segment, ".") {
		return fmt.Errorf("invalid filename (ends with dot)")
	}

	return nil
}
