package middleware

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	config "uit-toolbox/config"

	"golang.org/x/crypto/bcrypt"
)

type CTXClientIP struct{}
type CTXURLRequest struct{}
type CTXFileRequest struct {
	FullPath     string
	ResolvedPath string
	FileName     string
}

type HTTPErrorCodes struct {
	Message string `json:"message"`
}

type ReturnedJsonToken struct {
	Token string  `json:"token"`
	TTL   float64 `json:"ttl"`
	Valid bool    `json:"valid"`
}

func GetAuthCookiesForResponse(uitSessionIDValue, uitBasicValue, uitBearerValue, uitCSRFValue string, timeout time.Duration) (*http.Cookie, *http.Cookie, *http.Cookie, *http.Cookie) {
	sessionIDCookie := &http.Cookie{
		Name:     "uit_session_id",
		Value:    uitSessionIDValue,
		Path:     "/",
		Expires:  time.Now().Add(timeout),
		MaxAge:   20 * 60,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	basicCookie := &http.Cookie{
		Name:     "uit_basic_token",
		Value:    uitBasicValue,
		Path:     "/",
		Expires:  time.Now().Add(timeout),
		MaxAge:   20 * 60,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	bearerCookie := &http.Cookie{
		Name:     "uit_bearer_token",
		Value:    uitBearerValue,
		Path:     "/",
		Expires:  time.Now().Add(timeout),
		MaxAge:   20 * 60,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	csrfCookie := &http.Cookie{
		Name:     "uit_csrf_token",
		Value:    uitCSRFValue,
		Path:     "/",
		Expires:  time.Now().Add(timeout),
		MaxAge:   20 * 60,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	return sessionIDCookie, basicCookie, bearerCookie, csrfCookie
}

func FormatHttpError(errorString string) (jsonErrStr string) {
	jsonStr := HTTPErrorCodes{Message: errorString}
	jsonErr, err := json.Marshal(jsonStr)
	if err != nil {
		return ""
	}
	return string(jsonErr)
}

func checkValidIP(s string) (isValid bool, isLoopback bool, isLocal bool) {
	log := config.GetLogger()
	maxStringSize := int64(128)
	maxCharSize := int(4)

	ipBytes := &io.LimitedReader{
		R: strings.NewReader(s),
		N: maxStringSize,
	}
	reader := bufio.NewReader(ipBytes)

	var totalBytes int64
	var b strings.Builder
	for {
		char, charSize, err := reader.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Warning("read error in checkValidIP" + err.Error())
			return false, false, false
		}
		if charSize > maxCharSize {
			log.Warning("IP address contains an invalid Unicode character")
			return false, false, false
		}
		if char == utf8.RuneError && charSize == 1 {
			return false, false, false
		}
		if (char >= '0' && char <= '9') && (char == '.' || char == ':') {
			log.Warning("IP address contains an invalid character")
			return false, false, false
		}
		totalBytes += int64(charSize)
		if totalBytes > maxStringSize {
			log.Warning("IP length exceeded " + strconv.FormatInt(maxStringSize, 10) + " bytes")
			return false, false, false
		}
		b.WriteRune(char)
	}

	ip := strings.TrimSpace(b.String())
	if ip == "" {
		return false, false, false
	}

	// Reset string builder so GC can get rid of it
	b.Reset()

	parsedIP, err := netip.ParseAddr(ip)
	if err != nil {
		return false, false, false
	}

	// If unspecified, empty, or wrong byte size
	if parsedIP.BitLen() != 32 && parsedIP.BitLen() != 128 {
		log.Warning("IP Address is the incorrect length")
		return false, false, false
	}

	if parsedIP.IsUnspecified() || !parsedIP.IsValid() {
		log.Warning("IP address is unspecified or invalid: " + string(parsedIP.String()))
		return false, false, false
	}

	if !parsedIP.Is4() || parsedIP.Is4In6() || parsedIP.Is6() {
		log.Warning("IP address is not IPv4: " + string(parsedIP.String()))
		return false, false, false
	}

	if parsedIP.IsInterfaceLocalMulticast() || parsedIP.IsLinkLocalMulticast() || parsedIP.IsMulticast() {
		log.Warning("IP address is multicast: " + string(parsedIP.String()))
		return false, false, false
	}

	return true, parsedIP.IsLoopback(), parsedIP.IsPrivate()
}

func GetRequestIP(r *http.Request) (string, bool) {
	if ip, ok := r.Context().Value(CTXClientIP{}).(string); ok {
		return ip, true
	}
	return "", false
}

func GetRequestURL(r *http.Request) (string, bool) {
	if url, ok := r.Context().Value(CTXURLRequest{}).(string); ok {
		return url, true
	}
	return "", false
}

func GetRequestedFile(req *http.Request) (string, string, string, bool) {
	if fileRequest, ok := req.Context().Value(CTXFileRequest{}).(CTXFileRequest); ok {
		return fileRequest.FullPath, fileRequest.ResolvedPath, fileRequest.FileName, true
	}
	return "", "", "", false
}

func CheckAuthCredentials(ctx context.Context, username, password string) (bool, error) {
	db := config.GetDatabaseConn()
	if db == nil {
		return false, errors.New("database is not initialized")
	}
	var tmpToken = username + ":" + password
	authToken := sha256.Sum256([]byte(tmpToken))
	authTokenString := hex.EncodeToString(authToken[:])

	sqlCode := `SELECT ENCODE(SHA256(CONCAT(username, ':', password)::bytea), 'hex') FROM logins WHERE ENCODE(SHA256(CONCAT(username, ':', password)::bytea), 'hex') = $1`
	rows, err := db.QueryContext(ctx, sqlCode, authTokenString)
	for rows.Next() {
		var dbToken string
		if err := rows.Scan(&dbToken); err != nil {
			return false, errors.New("cannot scan database row for API Auth: " + err.Error())
		}
		if dbToken == authTokenString {
			err := bcrypt.CompareHashAndPassword([]byte(dbToken), []byte(authTokenString))
			if err != nil {
				return false, errors.New("invalid credentials - bcrypt mismatch")
			}
			return true, nil
		}
	}
	if err == sql.ErrNoRows {
		buffer1 := make([]byte, 32)
		_, _ = rand.Read(buffer1)
		buffer2 := make([]byte, 32)
		_, _ = rand.Read(buffer2)
		pass1, _ := bcrypt.GenerateFromPassword(buffer1, bcrypt.DefaultCost)
		pass2, _ := bcrypt.GenerateFromPassword(buffer2, bcrypt.DefaultCost)
		bcrypt.CompareHashAndPassword(pass1, pass2)
		return false, errors.New("invalid credentials")
	}
	if err != nil {
		return false, errors.New("cannot query database for API Auth: " + err.Error())
	}
	defer rows.Close()

	if !rows.Next() {
		return false, errors.New("no matching auth token found")
	}

	return false, errors.New("unknown error during authentication")
}

func IsPrintableASCII(b []byte) bool {
	for i := range b {
		c := b[i]
		if c < 0x21 || c > 0x7E {
			return false
		}
	}
	return true
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
