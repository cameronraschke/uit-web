package middleware

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	config "uit-toolbox/config"

	"golang.org/x/crypto/bcrypt"
)

type ctxClientIPKey struct{}
type ctxURLRequestKey struct{}
type ctxPathRequestKey struct{}
type ctxQueryRequestKey struct{}
type ctxFileRequestKey struct{}
type ctxRequestUUIDKey struct{}

type ReturnedJsonToken struct {
	Token string  `json:"token"`
	TTL   float64 `json:"ttl"`
	Valid bool    `json:"valid"`
}

type JsonError struct {
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

var (
	clientIPKey     ctxClientIPKey
	urlRequestKey   ctxURLRequestKey
	pathRequestKey  ctxPathRequestKey
	queryRequestKey ctxQueryRequestKey
	fileRequestKey  ctxFileRequestKey
	requestUUIDKey  ctxRequestUUIDKey
)

func WriteJson(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteJsonError(w http.ResponseWriter, httpStatusCode int) {
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
		ErrorMessage: http.StatusText(httpStatusCode),
	}

	err := json.NewEncoder(w).Encode(jsonStruct)
	if err != nil {
		return
	}

	_ = responseController.Flush()
}

func withClientIP(ctx context.Context, ipStr string) (context.Context, error) {
	if strings.TrimSpace(ipStr) == "" {
		return ctx, errors.New("empty IP address")
	}
	ipAddr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return ctx, fmt.Errorf("failed to parse IP address: %w", err)
	}
	if !ipAddr.IsValid() {
		return ctx, errors.New("invalid IP address: " + ipStr)
	}
	return context.WithValue(ctx, clientIPKey, ipAddr), nil
}
func GetRequestIPFromContext(ctx context.Context) (ipAddr netip.Addr, ok bool) {
	ipAddr, ok = ctx.Value(clientIPKey).(netip.Addr)
	return ipAddr, ok
}
func GetRequestIPFromRequestContext(r *http.Request) (ipAddr netip.Addr, ok bool) {
	return GetRequestIPFromContext(r.Context())
}

func withRequestURL(ctx context.Context, url string) (context.Context, error) {
	// if strings.TrimSpace(url) == "" {
	// 	return ctx, errors.New("empty request URL")
	// }
	return context.WithValue(ctx, urlRequestKey, url), nil
}
func GetRequestURLFromContext(ctx context.Context) (url string, ok bool) {
	url, ok = ctx.Value(urlRequestKey).(string)
	return url, ok
}
func GetRequestURLFromRequestContext(r *http.Request) (url string, ok bool) {
	return GetRequestURLFromContext(r.Context())
}

func withRequestPath(ctx context.Context, path string) (context.Context, error) {
	// if strings.TrimSpace(path) == "" {
	// 	return ctx, errors.New("empty request path")
	// }
	return context.WithValue(ctx, pathRequestKey, path), nil
}
func GetRequestPathFromContext(ctx context.Context) (path string, ok bool) {
	path, ok = ctx.Value(pathRequestKey).(string)
	return path, ok
}
func GetRequestPathFromRequestContext(r *http.Request) (path string, ok bool) {
	return GetRequestPathFromContext(r.Context())
}

func withRequestQuery(ctx context.Context, query string) (context.Context, error) {
	// if strings.TrimSpace(query) == "" {
	// 	return ctx, errors.New("empty request query")
	// }
	return context.WithValue(ctx, queryRequestKey, query), nil
}
func GetRequestQueryFromContext(ctx context.Context) (query string, ok bool) {
	query, ok = ctx.Value(queryRequestKey).(string)
	return query, ok
}
func GetRequestQueryFromRequestContext(r *http.Request) (query string, ok bool) {
	return GetRequestQueryFromContext(r.Context())
}

func withRequestFile(ctx context.Context, file string) (context.Context, error) {
	// if strings.TrimSpace(file) == "" {
	// 	return ctx, errors.New("empty request file")
	// }
	return context.WithValue(ctx, fileRequestKey, file), nil
}
func GetRequestFileFromContext(ctx context.Context) (file string, ok bool) {
	file, ok = ctx.Value(fileRequestKey).(string)
	return file, ok
}
func GetRequestFileFromRequestContext(r *http.Request) (file string, ok bool) {
	return GetRequestFileFromContext(r.Context())
}

func withRequestUUID(ctx context.Context, uuid string) (context.Context, error) {
	if strings.TrimSpace(uuid) == "" {
		return ctx, errors.New("empty request UUID")
	}
	return context.WithValue(ctx, requestUUIDKey, uuid), nil
}
func GetRequestUUIDFromContext(ctx context.Context) (uuid string, ok bool) {
	uuid, ok = ctx.Value(requestUUIDKey).(string)
	return uuid, ok
}
func GetRequestUUIDFromRequestContext(r *http.Request) (uuid string, ok bool) {
	return GetRequestUUIDFromContext(r.Context())
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

func checkValidIP(ip string) (isValid bool, isLoopback bool, isLocal bool, err error) {
	maxStringSize := int64(128)

	if len(ip) > int(maxStringSize) {
		return false, false, false, fmt.Errorf("IP address string length exceeded %d bytes", maxStringSize)
	}
	ipStr := strings.TrimSpace(ip)
	if ipStr == "" {
		return false, false, false, errors.New("IP address string is empty")
	}
	if !utf8.ValidString(ipStr) {
		return false, false, false, errors.New("IP address string is not valid UTF-8")
	}

	parsedIP, err := netip.ParseAddr(ipStr)
	if err != nil {
		return false, false, false, fmt.Errorf("failed to parse IP address string: %w", err)
	}

	// If unspecified, empty, or wrong byte size
	if parsedIP.BitLen() != 32 && parsedIP.BitLen() != 128 {
		return false, false, false, errors.New("parsed IP address is the incorrect length")
	}

	if parsedIP.IsUnspecified() || !parsedIP.IsValid() {
		return false, false, false, errors.New("parsed IP address is unspecified or invalid: " + parsedIP.String())
	}

	// if !parsedIP.Is4() || parsedIP.Is4In6() || parsedIP.Is6() {
	// 	return false, false, false, errors.New("IP address is not IPv4: " + parsedIP.String())
	// }

	if parsedIP.IsInterfaceLocalMulticast() || parsedIP.IsLinkLocalMulticast() || parsedIP.IsMulticast() {
		return false, false, false, errors.New("parsed IP address is multicast: " + parsedIP.String())
	}

	return true, parsedIP.IsLoopback(), parsedIP.IsPrivate(), nil
}

func CheckAuthCredentials(ctx context.Context, username, password string) (bool, error) {
	db := config.GetDatabaseConn()
	if db == nil {
		return false, errors.New("database is not initialized")
	}

	sqlCode := `SELECT password FROM logins WHERE username = $1 LIMIT 1;`
	var dbBcryptHash sql.NullString
	err := db.QueryRowContext(ctx, sqlCode, username).Scan(&dbBcryptHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			buffer1 := make([]byte, 32)
			_, _ = rand.Read(buffer1)
			buffer2 := make([]byte, 32)
			_, _ = rand.Read(buffer2)
			pass1, _ := bcrypt.GenerateFromPassword(buffer1, bcrypt.DefaultCost)
			pass2, _ := bcrypt.GenerateFromPassword(buffer2, bcrypt.DefaultCost)
			bcrypt.CompareHashAndPassword(pass1, pass2)
			return false, errors.New("invalid credentials")
		}

		return false, errors.New("query error")
	}

	// Compare supplied (already SHA256 hex or plaintext per your chosen model) versus stored bcrypt
	if bcrypt.CompareHashAndPassword([]byte(dbBcryptHash.String), []byte(password)) != nil {
		return false, errors.New("invalid credentials")
	}

	return true, nil
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

func IsAlphanumericAscii(b []byte) bool {
	for i := range b {
		char := b[i]
		if (char < '0' || char > '9') && (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') && char != '_' && char != '-' && char != ' ' {
			return false
		}
	}
	return true
}

func IsNumericAscii(b []byte) bool {
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

func ValidateAuthFormInputSHA256(username, password string) error {
	username = strings.TrimSpace(username)
	usernameLength := utf8.RuneCountInString(username)
	if usernameLength != 64 {
		return errors.New("invalid SHA hash length for username")
	}

	password = strings.TrimSpace(password)
	passwordLength := utf8.RuneCountInString(password)
	if passwordLength != 64 {
		return errors.New("invalid SHA hash length for password")
	}

	if err := IsSHA256String(username); err != nil {
		return errors.New("username does not match SHA regex")
	}
	if err := IsSHA256String(password); err != nil {
		return errors.New("password does not match SHA regex")
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
