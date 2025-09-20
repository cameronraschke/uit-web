package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
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

func generateSessionID() (string, error) {
	buffer := make([]byte, 64)
	_, err := rand.Read(buffer)
	if err != nil {
		return "", err
	}
	for i := 0; i < 3; i++ {
		buffer, err = bcrypt.GenerateFromPassword(buffer, bcrypt.DefaultCost)
		if err != nil {
			return "", errors.New("failed to generate session ID: " + err.Error())
		}
	}
	return base64.StdEncoding.EncodeToString(buffer), nil
}

func createOrUpdateAuthSession(authMap *sync.Map, sessionID string, basic BasicToken, bearer BearerToken, csrfToken string) (AuthSession, bool, error) {
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

func checkAuthCredentials(ctx context.Context, username, password string) error {
	// Use context with timeout for each request
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	hashedUsername := sha256.Sum256([]byte(username))
	hashedUsernameString := hex.EncodeToString(hashedUsername[:])
	hashedPassword := sha256.Sum256([]byte(password))
	hashedPasswordString := hex.EncodeToString(hashedPassword[:])

	var authToken = hashedUsernameString + ":" + hashedPasswordString

	// Each request gets its own connection from the pool
	var newHashedPassword string
	sqlCode := `SELECT ENCODE(SHA256(CONCAT(username, ':', password)::bytea), 'hex') as token FROM logins WHERE ENCODE(SHA256(CONCAT(username, ':', password)::bytea), 'hex') = ENCODE(SHA256($1::bytea), 'hex')`
	rows, err := db.QueryContext(ctx, sqlCode, authToken)
	if err != nil {
		return errors.New("cannot query database for API Auth: " + err.Error())
	}
	defer rows.Close()

	if !rows.Next() {
		return errors.New("no matching auth token found")
	}

	err = db.QueryRowContext(queryCtx, sqlCode, authToken).Scan(&newHashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			buffer1 := make([]byte, 32)
			_, _ = rand.Read(buffer1)
			buffer2 := make([]byte, 32)
			_, _ = rand.Read(buffer2)
			pass1, _ := bcrypt.GenerateFromPassword(buffer1, bcrypt.DefaultCost)
			pass2, _ := bcrypt.GenerateFromPassword(buffer2, bcrypt.DefaultCost)
			bcrypt.CompareHashAndPassword(pass1, pass2)
			return errors.New("invalid credentials")
		}
		return errors.New("database error: " + err.Error())
	}

	return bcrypt.CompareHashAndPassword([]byte(newHashedPassword), []byte(authToken))
}

func validateAuthFormInput(username, password string) error {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 20 {
		return errors.New("invalid username length")
	}

	if len(password) < 8 || len(password) > 64 {
		return errors.New("invalid password length")
	}

	authStr := username + ":" + password

	for _, r := range username {
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			return errors.New("username contains invalid characters")
		}
	}

	// ASCII
	allowedAuthChars := "U+0020-U+007E"

	for _, char := range authStr {
		if char < 32 || char == 127 {
			log.Warning("Control/non-printable character in auth string: " + authStr)
			return errors.New("auth string contains invalid characters")
		}
		if char > 127 || char > unicode.MaxASCII || char > unicode.MaxLatin1 {
			log.Warning("Non-ASCII character in auth string: " + authStr)
			return errors.New("auth string contains invalid characters")
		}
		if unicode.IsControl(char) {
			log.Warning("Control character in auth string: " + authStr)
			return errors.New("auth string contains invalid characters")
		}
		if !strings.ContainsRune(allowedAuthChars, char) {
			log.Warning("Disallowed character in auth string: " + authStr)
			return errors.New("auth string contains invalid characters")
		}
	}

	return nil
}

func generateTokensConcurrent() (string, string, error) {
	bearerBytes := make([]byte, 32)
	csrfBytes := make([]byte, 32)

	if _, err := rand.Read(bearerBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate bearer token: %w", err)
	}

	if _, err := rand.Read(csrfBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate CSRF token: %w", err)
	}

	bearerToken := hex.EncodeToString(bearerBytes)
	csrfToken := hex.EncodeToString(csrfBytes)

	return bearerToken, csrfToken, nil
}
