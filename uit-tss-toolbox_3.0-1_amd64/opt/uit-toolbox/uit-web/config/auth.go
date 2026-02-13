package config

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

const (
	basicTTL  = 20 * time.Minute
	bearerTTL = 20 * time.Minute
	csrfTTL   = 20 * time.Minute
)

// Auth for web users
func GetAdminCredentials() (string, string, error) {
	appState, err := GetAppState()
	if err != nil {
		return "", "", fmt.Errorf("error getting app state in GetAdminCredentials: %w", err)
	}

	adminUsername := "admin"
	adminPasswd := strings.TrimSpace(appState.AppConfig.Load().UIT_WEB_USER_DEFAULT_PASSWD)
	return adminUsername, adminPasswd, nil
}

// Auth session management
func GetAuthSessions() map[string]AuthSession {
	appState, err := GetAppState()
	if err != nil {
		return nil
	}
	authSessionsMap := make(map[string]AuthSession)
	appState.AuthMap.Range(func(k, v any) bool {
		key, keyExists := k.(string)
		value, valueExists := v.(AuthSession)
		if keyExists && valueExists {
			authSessionsMap[key] = value
		}
		return true
	})
	return authSessionsMap
}

func CreateAuthSession(ipAddress netip.Addr) (string, string, string, string, error) {
	if ipAddress == (netip.Addr{}) {
		return "", "", "", "", errors.New("empty IP address")
	}
	appState, err := GetAppState()
	if err != nil {
		return "", "", "", "", fmt.Errorf("error getting app state in CreateAuthSession: %w", err)
	}

	curTime := time.Now()

	sessionID, err := GenerateSessionToken(32)
	if err != nil {
		return "", "", "", "", fmt.Errorf("generate session id: %w", err)
	}
	basicToken, err := GenerateSessionToken(32)
	if err != nil {
		return "", "", "", "", fmt.Errorf("generate basic token: %w", err)
	}
	bearerToken, err := GenerateSessionToken(32)
	if err != nil {
		return "", "", "", "", fmt.Errorf("generate bearer token: %w", err)
	}
	csrfToken, err := GenerateSessionToken(32)
	if err != nil {
		return "", "", "", "", fmt.Errorf("generate csrf token: %w", err)
	}

	basicMAC, err := HashSessionToken(basicToken)
	if err != nil {
		return "", "", "", "", fmt.Errorf("hash basic token: %w", err)
	}
	bearerMAC, err := HashSessionToken(bearerToken)
	if err != nil {
		return "", "", "", "", fmt.Errorf("hash bearer token: %w", err)
	}
	csrfMAC, err := HashSessionToken(csrfToken)
	if err != nil {
		return "", "", "", "", fmt.Errorf("hash csrf token: %w", err)
	}

	authSession := AuthSession{
		SessionID: sessionID,
		Basic: BasicToken{
			Token:     basicMAC,
			Expiry:    curTime.Add(basicTTL),
			NotBefore: curTime,
			TTL:       basicTTL.Seconds(),
			IP:        ipAddress,
			Valid:     true,
		},
		Bearer: BearerToken{
			Token:     bearerMAC,
			Expiry:    curTime.Add(bearerTTL),
			NotBefore: curTime,
			TTL:       bearerTTL.Seconds(),
			IP:        ipAddress,
			Valid:     true,
		},
		CSRF: CSRFToken{
			Token:     csrfMAC,
			Expiry:    curTime.Add(csrfTTL),
			NotBefore: curTime,
			TTL:       csrfTTL.Seconds(),
			IP:        ipAddress,
			Valid:     true,
		},
	}

	for i := range 3 {
		if i > 0 {
			if _, exists := appState.AuthMap.Load(authSession.SessionID); !exists {
				break
			}
			newID, genErr := GenerateSessionToken(32)
			if genErr != nil {
				return "", "", "", "", fmt.Errorf("regenerate authSession id: %w", genErr)
			}
			authSession.SessionID = newID
		}
	}

	// Store and bump count
	appState.AuthMap.Store(authSession.SessionID, authSession)
	appState.AuthMapEntryCount.Add(1)

	return authSession.SessionID, basicToken, bearerToken, csrfToken, nil

}

func DeleteAuthSession(sessionID string) {
	appState, err := GetAppState()
	if err != nil {
		return
	}
	if _, ok := appState.AuthMap.LoadAndDelete(sessionID); ok {
		newVal := appState.AuthMapEntryCount.Add(-1)
		if newVal < 0 {
			appState.AuthMapEntryCount.Store(0)
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
	appState.AuthMap.Range(func(k, v any) bool {
		authSession, ok := v.(AuthSession)
		if !ok {
			return true
		}
		if authSession.Basic.Expiry.Before(curTime) && authSession.Bearer.Expiry.Before(curTime) {
			DeleteAuthSession(k.(string))
			authSessionCount := GetAuthSessionCount()
			log.Info("Auth session expired: " + authSession.Basic.IP.String() + " (TTL: " + fmt.Sprintf("%.2f", authSession.Bearer.Expiry.Sub(curTime).Seconds()) + ", " + strconv.Itoa(int(authSessionCount)) + " session(s))")
		}
		return true
	})
}

func GetAuthSessionCount() int64 {
	appState, err := GetAppState()
	if err != nil {
		return 0
	}
	return appState.AuthMapEntryCount.Load()
}

func RefreshAndGetAuthSessionCount() int64 {
	appState, err := GetAppState()
	if err != nil {
		return 0
	}
	var entries int64
	appState.AuthMap.Range(func(_, _ any) bool {
		entries++
		return true
	})
	appState.AuthMapEntryCount.Store(entries)
	return entries
}

func CheckAuthSessionExists(sessionID string, ipAddress netip.Addr, basicToken string, bearerToken string, csrfToken string) (bool, bool, error) {
	sessionValid := false
	sessionExists := false

	appState, err := GetAppState()
	if err != nil {
		return sessionValid, sessionExists, fmt.Errorf("error getting app state in CheckAuthSessionExists: %w", err)
	}

	value, ok := appState.AuthMap.Load(sessionID)
	if !ok {
		return sessionValid, sessionExists, nil
	}
	sessionExists = true

	authSession, ok := value.(AuthSession)
	if !ok {
		return sessionValid, sessionExists, errors.New("invalid auth session type")
	}

	curTime := time.Now()

	if authSession.Basic.IP != ipAddress || authSession.Bearer.IP != ipAddress {
		return sessionValid, sessionExists, errors.New("IP address mismatch for session ID: " + sessionID)
	}

	if ipAddress == (netip.Addr{}) || strings.TrimSpace(basicToken) == "" || strings.TrimSpace(bearerToken) == "" {
		return sessionValid, sessionExists, errors.New("empty IP address or token for session ID: " + sessionID)
	}

	if !VerifySessionToken(basicToken, authSession.Basic.Token) ||
		!VerifySessionToken(bearerToken, authSession.Bearer.Token) {
		return sessionValid, sessionExists, nil
	}

	if authSession.Basic.Expiry.Before(curTime) || authSession.Bearer.Expiry.Before(curTime) {
		return sessionValid, sessionExists, nil
	}

	sessionValid = true
	return sessionValid, sessionExists, nil
}

func ExtendAuthSession(sessionID string) (bool, error) {
	appState, err := GetAppState()
	if err != nil {
		return false, fmt.Errorf("error getting app state in ExtendAuthSession: %w", err)
	}
	value, ok := appState.AuthMap.Load(sessionID)
	if !ok {
		return false, nil
	}
	authSession, ok := value.(AuthSession)
	if !ok {
		return false, errors.New("invalid auth session type")
	}
	curTime := time.Now()

	if authSession.Basic.Expiry.Before(curTime) || authSession.Bearer.Expiry.Before(curTime) {
		return false, nil
	}
	authSession.Basic.Expiry = curTime.Add(time.Duration(20 * time.Minute))
	authSession.Bearer.Expiry = curTime.Add(time.Duration(20 * time.Minute))
	authSession.CSRF.Expiry = curTime.Add(time.Duration(20 * time.Minute))
	appState.AuthMap.Store(sessionID, authSession)
	return true, nil
}

func GenerateSessionToken(tokenSize int) (string, error) {
	if tokenSize <= 0 {
		tokenSize = 32
	}
	b := make([]byte, tokenSize)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
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
	serverSecret := make([]byte, len(appState.SessionSecret))
	copy(serverSecret, appState.SessionSecret)
	return serverSecret, nil
}
