package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
)

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type LimiterMap struct {
	m     sync.Map
	rate  float64
	burst int
}

type BlockedMap struct {
	m         sync.Map
	banPeriod time.Duration
}

var (
	rateLimit            float64
	rateLimitBurst       int
	rateLimitBanDuration time.Duration
	webServerLimiter     *LimiterMap
	blockedIPs           *BlockedMap
)

func countAuthSessions(m *sync.Map) int {
	authSessionCount := 0
	m.Range(func(_, _ any) bool {
		authSessionCount++
		return true
	})
	return authSessionCount
}

func formatHttpError(errorString string) (jsonErrStr string) {
	jsonStr := httpErrorCodes{Message: errorString}
	jsonErr, err := json.Marshal(jsonStr)
	if err != nil {
		log.Error("Cannot parse JSON: " + err.Error())
		return
	}
	return string(jsonErr)
}

func (lm *LimiterMap) Get(ip string) *rate.Limiter {
	curTime := time.Now()
	newEntry := &limiterEntry{
		limiter:  rate.NewLimiter(rate.Limit(lm.rate), lm.burst),
		lastSeen: curTime,
	}
	queriedLimiter, exists := lm.m.LoadOrStore(ip, newEntry)
	entry := queriedLimiter.(*limiterEntry)
	entry.lastSeen = curTime
	if !exists {
		log.Debug("Created new limiter for IP: " + ip + " rate=" + fmt.Sprint(lm.rate) + " burst=" + fmt.Sprint(lm.burst))
	}

	return entry.limiter
}

func (lm *LimiterMap) Delete(ip string) {
	lm.m.Delete(ip)
}

func (bm *BlockedMap) IsBlocked(ip string) bool {
	blockTime, ok := bm.m.Load(ip)
	if !ok {
		return false
	}
	unblockTime, ok := blockTime.(time.Time)
	if !ok {
		bm.m.Delete(ip)
		return false
	}
	if time.Now().After(unblockTime) {
		bm.m.Delete(ip)
		return false
	}
	return true
}

func (bm *BlockedMap) Block(ip string) {
	bm.m.Store(ip, time.Now().Add(bm.banPeriod))
}

func GetLimiter(requestIP string) *rate.Limiter {
	return webServerLimiter.Get(requestIP)
}

func IsBlocked(requestIP string) bool {
	return blockedIPs.IsBlocked(requestIP)
}

func BlockIP(requestIP string) {
	blockedIPs.Block(requestIP)
}

func (as *AppState) Cleanup() {
	ttl := time.Now().Add(-10 * time.Minute)
	as.webServerLimiter.m.Range(func(key, value any) bool {
		entry, ok := value.(*limiterEntry)
		if !ok || entry.lastSeen.Before(ttl) {
			as.webServerLimiter.m.Delete(key)
		}
		return true
	})

	curTime := time.Now()
	as.blockedIPs.m.Range(func(key, value any) bool {
		unblockTime, ok := value.(time.Time)
		if !ok || curTime.After(unblockTime) {
			as.blockedIPs.m.Delete(key)
		}
		return true
	})
}

func (as *AppState) GetAllBlockedIPs() string {
	var blocked []string
	as.blockedIPs.m.Range(func(key, value any) bool {
		ip, ok := key.(string)
		if ok {
			blocked = append(blocked, ip)
		}
		return true
	})
	return strings.Join(blocked, ", ")
}

func checkValidIP(s string) (isValid bool, isLoopback bool, isLocal bool) {
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
	if ip, ok := r.Context().Value(ctxClientIP{}).(string); ok {
		return ip, true
	}
	return "", false
}

func GetRequestURL(r *http.Request) (string, bool) {
	if url, ok := r.Context().Value(ctxURLRequest{}).(string); ok {
		return url, true
	}
	return "", false
}

func GetRequestedFile(req *http.Request) (string, string, string, bool) {
	if fileRequest, ok := req.Context().Value(ctxFileRequest{}).(ctxFileRequest); ok {
		return fileRequest.FullPath, fileRequest.ResolvedPath, fileRequest.FileName, true
	}
	return "", "", "", false
}

func ParseHeaders(header http.Header) HttpHeaders {
	var headers HttpHeaders
	var authHeader AuthHeader
	for _, value := range header.Values("Authorization") {
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "Basic ") {
			basic := strings.TrimSpace(strings.TrimPrefix(value, "Basic "))
			authHeader.Basic = &basic
		}
		if strings.HasPrefix(value, "Bearer ") {
			bearer := strings.TrimSpace(strings.TrimPrefix(value, "Bearer "))
			authHeader.Bearer = &bearer
		}
	}
	headers.Authorization = authHeader
	return headers
}

func CreateOrUpdateAuthSession(authMap *sync.Map, sessionID string, basic BasicToken, bearer BearerToken, csrfToken string) (AuthSession, bool, error) {
	newSession := AuthSession{
		Basic:  basic,
		Bearer: bearer,
		CSRF:   csrfToken,
	}

	// Use LoadOrStore for atomic operation
	value, exists := authMap.LoadOrStore(sessionID, newSession)

	if !exists {
		atomic.AddInt64(&authMapEntryCount, 1)
	}

	return value.(AuthSession), exists, nil
}

func CheckAuthCredentials(ctx context.Context, username, password string) (bool, error) {
	// hashedUsername := sha256.Sum256([]byte(username))
	// hashedUsernameString := hex.EncodeToString(hashedUsername[:])
	// hashedPassword := sha256.Sum256([]byte(password))
	// hashedPasswordString := hex.EncodeToString(hashedPassword[:])

	var tmpToken = username + ":" + password
	authToken := sha256.Sum256([]byte(tmpToken))
	authTokenString := hex.EncodeToString(authToken[:])

	sqlCode := `SELECT ENCODE(SHA256(CONCAT(username, ':', password)::bytea), 'hex') as token FROM logins WHERE ENCODE(SHA256(CONCAT(username, ':', password)::bytea), 'hex') = $1`
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

func ValidateAuthFormInput(username, password string) error {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 20 {
		return errors.New("invalid username length")
	}

	if len(password) < 8 || len(password) > 64 {
		return errors.New("invalid password length")
	}

	authStr := username + ":" + password

	// ASCII characters except space
	allowedAuthChars := "U+0021-U+007E"

	for _, char := range authStr {
		if char <= 31 || char >= 127 || char > unicode.MaxASCII || char > unicode.MaxLatin1 {
			return errors.New(`auth string contains an invalid control character (beyond ASCII/Latin1): ` + fmt.Sprintf("U+%04X", char))
		}
		if unicode.IsControl(char) {
			return errors.New(`auth string contains an invalid control character: ` + fmt.Sprintf("U+%04X", char))
		}
		if unicode.IsSpace(char) {
			return errors.New(`auth string contains a whitespace character: ` + fmt.Sprintf("U+%04X", char))
		}
		if !strings.ContainsRune(allowedAuthChars, char) {
			return errors.New(`auth string contains a disallowed character: ` + fmt.Sprintf("U+%04X", char))
		}
	}

	return nil
}

func GenerateAuthTokens() (string, string, error) {
	bearerBuffer := make([]byte, 32)
	csrfBuffer := make([]byte, 32)

	if _, err := rand.Read(bearerBuffer); err != nil {
		return "", "", errors.New("failed to generate buffer for bearer token: " + err.Error())
	}
	bearerToken, err := bcrypt.GenerateFromPassword(bearerBuffer, bcrypt.DefaultCost)
	if err != nil {
		return "", "", errors.New("failed to generate bearer token: " + err.Error())
	}

	if _, err := rand.Read(csrfBuffer); err != nil {
		return "", "", errors.New("failed to generate buffer for CSRF token: " + err.Error())
	}
	csrfToken, err := bcrypt.GenerateFromPassword(csrfBuffer, bcrypt.DefaultCost)
	if err != nil {
		return "", "", errors.New("failed to generate CSRF token: " + err.Error())
	}

	bearerTokenString := hex.EncodeToString(bearerToken)
	csrfTokenString := hex.EncodeToString(csrfToken)

	return bearerTokenString, csrfTokenString, nil
}
