package config

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

type BasicToken struct {
	Token     string        `json:"token"`
	Expiry    time.Time     `json:"expiry"`
	NotBefore time.Time     `json:"not_before"`
	TTL       time.Duration `json:"ttl"`
	IP        netip.Addr    `json:"ip"`
	Valid     bool          `json:"valid"`
}

type BearerToken struct {
	Token     string        `json:"token"`
	Expiry    time.Time     `json:"expiry"`
	NotBefore time.Time     `json:"not_before"`
	TTL       time.Duration `json:"ttl"`
	IP        netip.Addr    `json:"ip"`
	Valid     bool          `json:"valid"`
}

type CSRFToken struct {
	Token     string        `json:"token"`
	Expiry    time.Time     `json:"expiry"`
	NotBefore time.Time     `json:"not_before"`
	TTL       time.Duration `json:"ttl"`
	IP        netip.Addr    `json:"ip"`
	Valid     bool          `json:"valid"`
}

type AuthSession struct {
	IPAddress     netip.Addr
	SessionID     string
	SessionCookie *http.Cookie
	BasicToken    BasicToken
	BasicCookie   *http.Cookie
	BearerToken   BearerToken
	BearerCookie  *http.Cookie
	CSRFToken     CSRFToken
	CSRFCookie    *http.Cookie
}

const (
	BasicTTL  = 20 * time.Minute
	BearerTTL = 20 * time.Minute
	CSRFTTL   = 20 * time.Minute
)

// Auth for web users
func GetAdminCredentials() (string, string, error) {
	appState, err := GetAppState()
	if err != nil {
		return "", "", fmt.Errorf("error getting app state in GetAdminCredentials: %w", err)
	}

	adminUsername := "admin"
	adminPasswd := strings.TrimSpace(appState.appConfig.Load().WebUserDefaultPasswd)
	return adminUsername, adminPasswd, nil
}

// Auth session management
func GetAuthSessions() map[string]AuthSession {
	appState, err := GetAppState()
	if err != nil {
		return nil
	}
	authSessionsMap := make(map[string]AuthSession)
	appState.authMap.Range(func(k, v any) bool {
		key, keyExists := k.(string)
		value, valueExists := v.(AuthSession)
		if keyExists && valueExists {
			authSessionsMap[key] = value
		}
		return true
	})
	return authSessionsMap
}

func CreateAuthSession(ipAddress netip.Addr) (*AuthSession, error) {
	if ipAddress == (netip.Addr{}) || !ipAddress.IsValid() {
		return nil, errors.New("empty or invalid IP address")
	}
	appState, err := GetAppState()
	if err != nil {
		return nil, fmt.Errorf("error getting app state in CreateAuthSession: %w", err)
	}

	curTime := time.Now()

	sessionID := rand.Text()
	basicToken := rand.Text()
	bearerToken := rand.Text()
	csrfToken := rand.Text()

	basicMAC, err := HashSessionToken(basicToken)
	if err != nil {
		return nil, fmt.Errorf("error hashing basic token: %w", err)
	}
	bearerMAC, err := HashSessionToken(bearerToken)
	if err != nil {
		return nil, fmt.Errorf("error hashing bearer token: %w", err)
	}
	csrfMAC, err := HashSessionToken(csrfToken)
	if err != nil {
		return nil, fmt.Errorf("error hashing csrf token: %w", err)
	}

	authSession := AuthSession{
		SessionID: sessionID,
		BasicToken: BasicToken{
			Token:     basicMAC,
			Expiry:    curTime.Add(BasicTTL),
			NotBefore: curTime,
			TTL:       BasicTTL.Seconds(),
			IP:        ipAddress,
			Valid:     true,
		},
		BearerToken: BearerToken{
			Token:     bearerMAC,
			Expiry:    curTime.Add(BearerTTL),
			NotBefore: curTime,
			TTL:       BearerTTL.Seconds(),
			IP:        ipAddress,
			Valid:     true,
		},
		CSRFToken: CSRFToken{
			Token:     csrfMAC,
			Expiry:    curTime.Add(CSRFTTL),
			NotBefore: curTime,
			TTL:       CSRFTTL.Seconds(),
			IP:        ipAddress,
			Valid:     true,
		},
	}

	for i := range 3 {
		if i > 0 {
			if _, exists := appState.authMap.Load(authSession.SessionID); !exists {
				break
			}
			newID := rand.Text()
			authSession.SessionID = newID
		}
	}
	appState.authMap.Store(authSession.SessionID, authSession)
	appState.authMapEntryCount.Add(1)

	return &authSession, nil
}

func DeleteAuthSession(sessionID string) {
	appState, err := GetAppState()
	if err != nil {
		return
	}
	if _, ok := appState.authMap.LoadAndDelete(sessionID); ok {
		newVal := appState.authMapEntryCount.Add(-1)
		if newVal < 0 {
			appState.authMapEntryCount.Store(0)
		}
	}
}

func ClearExpiredAuthSessions() {
	log := GetLogger()
	appState, err := GetAppState()
	if err != nil {
		return
	}
	curTime := time.Now()
	appState.authMap.Range(func(k, v any) bool {
		sessionID, ok := k.(string)
		if !ok {
			return true
		}
		authSession, ok := v.(AuthSession)
		if !ok {
			return true
		}
		if authSession.BasicToken.Expiry.Before(curTime) && authSession.BearerToken.Expiry.Before(curTime) {
			DeleteAuthSession(sessionID)
			authSessionCount := GetAuthSessionCount()
			log.Info("Auth session expired: " + authSession.BasicToken.IP.String() + " (TTL: " + fmt.Sprintf("%.2f", authSession.BearerToken.Expiry.Sub(curTime).Seconds()) + ", " + strconv.Itoa(int(authSessionCount)) + " session(s) active)")
		}
		return true
	})
}

func GetAuthSessionCount() int64 {
	appState, err := GetAppState()
	if err != nil {
		return 0
	}
	return appState.authMapEntryCount.Load()
}

func RefreshAndGetAuthSessionCount() int64 {
	appState, err := GetAppState()
	if err != nil {
		return 0
	}
	var entries int64
	appState.authMap.Range(func(_, _ any) bool {
		entries++
		return true
	})
	appState.authMapEntryCount.Store(entries)
	return entries
}

func AuthSessionValid(checkedAuthSession *AuthSession) (sessionValid bool, err error) {
	if checkedAuthSession == nil {
		return sessionValid, fmt.Errorf("auth session is nil")
	}
	appState, err := GetAppState()
	if err != nil {
		return sessionValid, fmt.Errorf("cannot retrieve app state (AuthSessionValid): %w", err)
	}

	value, ok := appState.authMap.Load(checkedAuthSession.SessionID)
	if !ok {
		return sessionValid, nil
	}

	existingAuthSession, ok := value.(AuthSession)
	if !ok {
		return sessionValid, fmt.Errorf("invalid auth session type")
	}

	curTime := time.Now()

	if checkedAuthSession.BasicToken.IP != existingAuthSession.BasicToken.IP || checkedAuthSession.BearerToken.IP != existingAuthSession.BearerToken.IP {
		return sessionValid, fmt.Errorf("IP address mismatch for session ID: %s", checkedAuthSession.SessionID)
	}

	if existingAuthSession.BasicToken.IP == (netip.Addr{}) || existingAuthSession.BearerToken.IP == (netip.Addr{}) || strings.TrimSpace(existingAuthSession.BasicToken.Token) == "" || strings.TrimSpace(existingAuthSession.BearerToken.Token) == "" {
		return sessionValid, fmt.Errorf("empty IP address or token for session ID: %s", checkedAuthSession.SessionID)
	}

	if !VerifySessionToken(checkedAuthSession.BasicToken.Token, existingAuthSession.BasicToken.Token) ||
		!VerifySessionToken(checkedAuthSession.BearerToken.Token, existingAuthSession.BearerToken.Token) {
		return sessionValid, nil
	}

	if checkedAuthSession.BasicToken.Expiry.Before(curTime) || checkedAuthSession.BearerToken.Expiry.Before(curTime) {
		return sessionValid, nil
	}

	sessionValid = true
	return sessionValid, nil
}

func ExtendAuthSession(sessionID string) (bool, error) {
	appState, err := GetAppState()
	if err != nil {
		return false, fmt.Errorf("error getting app state in ExtendAuthSession: %w", err)
	}
	value, ok := appState.authMap.Load(sessionID)
	if !ok {
		return false, nil
	}
	authSession, ok := value.(AuthSession)
	if !ok {
		return false, errors.New("invalid auth session type")
	}
	curTime := time.Now()

	if authSession.BasicToken.Expiry.Before(curTime) || authSession.BearerToken.Expiry.Before(curTime) {
		return false, nil
	}
	authSession.BasicToken.Expiry = curTime.Add(time.Duration(20 * time.Minute))
	authSession.BearerToken.Expiry = curTime.Add(time.Duration(20 * time.Minute))
	authSession.CSRFToken.Expiry = curTime.Add(time.Duration(20 * time.Minute))
	appState.authMap.Store(sessionID, authSession)
	return true, nil
}

// HashSessionToken returns HMAC-SHA256(token) using a server-side secret key.
func HashSessionToken(clientToken string) (string, error) {
	serverSecret, err := GetServerSecret()
	if err != nil {
		return "", err
	}
	hmacHash := hmac.New(sha256.New, serverSecret)
	hmacHash.Write([]byte(clientToken))
	return hex.EncodeToString(hmacHash.Sum(nil)), nil
}

// VerifySessionToken checks token against a stored HMAC hex string.
func VerifySessionToken(clientToken string, storedHex string) bool {
	serverSecret, err := GetServerSecret()
	if err != nil {
		return false
	}
	want, err := hex.DecodeString(storedHex)
	if err != nil {
		return false
	}
	hmacHash := hmac.New(sha256.New, serverSecret)
	hmacHash.Write([]byte(clientToken))
	return hmac.Equal(hmacHash.Sum(nil), want)
}

func GetServerSecret() ([]byte, error) {
	appState, err := GetAppState()
	if err != nil {
		return nil, fmt.Errorf("error getting app state in GetServerSecret: %w", err)
	}
	return appState.sessionSecret, nil
}
