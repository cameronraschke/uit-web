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
	"uit-toolbox/types"
)

var ErrTooManyAuthSessions = errors.New("auth session limit reached")

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

// Returns all active auth sessions as a map[string]types.AuthSession
func GetAuthSessions() map[string]types.AuthSession {
	appState, err := GetAppState()
	if err != nil {
		return nil
	}
	authSessionsMap := make(map[string]types.AuthSession)
	appState.authMapMutex.RLock()
	defer appState.authMapMutex.RUnlock()
	for sessionID, value := range appState.authMap {
		if sessionID == "" && value != (types.AuthSession{}) {
			continue
		}
		authSessionsMap[sessionID] = value
	}
	return authSessionsMap
}

func CreateAuthSession(requestIP netip.Addr) (*types.AuthSession, error) {
	if requestIP == (netip.Addr{}) || !requestIP.IsValid() {
		return nil, errors.New("empty or invalid IP address")
	}
	appState, err := GetAppState()
	if err != nil {
		return nil, fmt.Errorf("error getting app state in CreateAuthSession: %w", err)
	}

	appState.authMapMutex.RLock()
	if len(appState.authMap) >= 1000 {
		appState.authMapMutex.RUnlock()
		return nil, ErrTooManyAuthSessions
	}
	appState.authMapMutex.RUnlock()

	curTime := time.Now()

	sessionID := rand.Text()
	basicToken := rand.Text()
	bearerToken := rand.Text()
	csrfToken := rand.Text()

	authSession := types.AuthSession{
		IPAddress:  requestIP,
		SessionID:  sessionID,
		SessionTTL: types.AuthSessionTTL,
		SessionCookie: &http.Cookie{
			Name:     "uit_session_id",
			Value:    sessionID,
			Path:     "/",
			Expires:  curTime.Add(types.AuthSessionTTL),
			MaxAge:   int(types.AuthSessionTTL.Seconds()),
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
		BasicToken: types.BasicToken{
			Token:     basicToken,
			Expiry:    curTime.Add(types.BasicTTL),
			NotBefore: curTime,
			TTL:       types.BasicTTL,
			IP:        requestIP,
			Valid:     true,
		},
		BasicCookie: &http.Cookie{
			Name:     "uit_basic_token",
			Value:    basicToken,
			Path:     "/",
			Expires:  curTime.Add(types.BasicTTL),
			MaxAge:   int(types.BasicTTL.Seconds()),
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
		BearerToken: types.BearerToken{
			Token:     bearerToken,
			Expiry:    curTime.Add(types.BearerTTL),
			NotBefore: curTime,
			TTL:       types.BearerTTL,
			IP:        requestIP,
			Valid:     true,
		},
		BearerCookie: &http.Cookie{
			Name:     "uit_bearer_token",
			Value:    bearerToken,
			Path:     "/",
			Expires:  curTime.Add(types.BearerTTL),
			MaxAge:   int(types.BearerTTL.Seconds()),
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
		CSRFToken: types.CSRFToken{
			Token:     csrfToken,
			Expiry:    curTime.Add(types.CSRFTTL),
			NotBefore: curTime,
			TTL:       types.CSRFTTL,
			IP:        requestIP,
			Valid:     true,
		},
		CSRFCookie: &http.Cookie{
			Name:     "uit_csrf_token",
			Value:    csrfToken,
			Path:     "/",
			Expires:  curTime.Add(types.CSRFTTL),
			MaxAge:   int(types.CSRFTTL.Seconds()),
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
	}

	appState.authMapMutex.Lock()
	defer appState.authMapMutex.Unlock()
	appState.authMap[authSession.SessionID] = authSession

	return &authSession, nil
}

func DeleteAuthSessions(sessionIDs []string) []error {
	log := GetLogger()
	appState, err := GetAppState()

	errSlice := make([]error, 0, len(sessionIDs))

	if err != nil {
		errSlice = append(errSlice, fmt.Errorf("failed to get app state: %w", err))
		return errSlice
	}

	stringsToLog := make([]string, 0, len(sessionIDs))

	appState.authMapMutex.Lock()
	for _, sessionID := range sessionIDs {
		if _, ok := appState.authMap[sessionID]; ok {
			ipAddress := appState.authMap[sessionID].IPAddress.String()
			delete(appState.authMap, sessionID)
			sessionCount := len(appState.authMap)
			stringsToLog = append(stringsToLog, "Deleted auth session with ID: "+sessionID+" (IP: "+ipAddress+", active sessions: "+strconv.Itoa(sessionCount)+")")
		} else {
			errSlice = append(errSlice, fmt.Errorf("Attempted to delete non-existent auth session with ID: %s", sessionID))
		}
	}
	appState.authMapMutex.Unlock()

	for _, msg := range stringsToLog {
		log.Info(msg)
	}
	return errSlice
}

func ClearExpiredAuthSessions() {
	log := GetLogger()
	appState, err := GetAppState()
	if err != nil {
		return
	}
	curTime := time.Now()

	expiredAuthSessions := make([]string, 0, 10)
	stringsToLog := make([]string, 0, 10)

	appState.authMapMutex.RLock()
	for sessionID, authSession := range appState.authMap {
		if authSession.BasicToken.Expiry.Before(curTime) &&
			authSession.BearerToken.Expiry.Before(curTime) {
			expiredAuthSessions = append(expiredAuthSessions, sessionID)
			stringsToLog = append(stringsToLog, "Auth session expired: "+authSession.BasicToken.IP.String()+" (TTL: "+fmt.Sprintf("%.2f", authSession.BearerToken.Expiry.Sub(curTime).Seconds())+")")
		}
	}
	appState.authMapMutex.RUnlock()

	for _, msg := range stringsToLog {
		log.Info(msg)
	}

	if errSlice := DeleteAuthSessions(expiredAuthSessions); len(errSlice) > 0 {
		for _, err := range errSlice {
			log.Warn(err.Error())
		}
	}
}

func GetAuthSessionCount() int64 {
	appState, err := GetAppState()
	if err != nil {
		return 0
	}
	appState.authMapMutex.RLock()
	defer appState.authMapMutex.RUnlock()
	return int64(len(appState.authMap))
}

func IsAuthSessionValid(checkedAuthSession *types.AuthSession, requestIP netip.Addr) (bool, error) {
	curTime := time.Now()

	if checkedAuthSession == nil || checkedAuthSession.SessionID == "" || requestIP == (netip.Addr{}) || !requestIP.IsValid() {
		return false, fmt.Errorf("auth session and/or request IP is nil or invalid (IsAuthSessionValid)")
	}

	authSession, err := GetAuthSessionByID(checkedAuthSession.SessionID)
	if err != nil {
		return false, fmt.Errorf("error retrieving auth session by ID: %w", err)
	}

	if authSession.SessionTTL <= 0 ||
		authSession.BasicToken.TTL <= 0 ||
		authSession.BearerToken.TTL <= 0 {
		// authSession.CSRFToken.TTL <= 0
		return false, fmt.Errorf("auth tokens have reached their TTL")
	}

	if authSession.SessionID != checkedAuthSession.SessionID ||
		authSession.BasicToken.Token != checkedAuthSession.BasicToken.Token ||
		authSession.BearerToken.Token != checkedAuthSession.BearerToken.Token {
		// authSession.CSRFToken.Token != checkedAuthSession.CSRFToken.Token
		return false, fmt.Errorf("request tokens do not match stored session tokens")
	}

	if authSession.BasicToken.Expiry.Before(curTime) ||
		authSession.BearerToken.Expiry.Before(curTime) {
		// authSession.CSRFToken.Expiry.Before(curTime)
		return false, fmt.Errorf("auth tokens have expired")
	}

	if authSession.IPAddress != requestIP ||
		authSession.BasicToken.IP != requestIP ||
		authSession.BearerToken.IP != requestIP {
		// authSession.CSRFToken.IP != requestIP
		return false, fmt.Errorf("request IP does not match stored token IP")
	}

	return true, nil
}

func GetAuthSessionByID(sessionID string) (*types.AuthSession, error) {
	appState, err := GetAppState()
	if err != nil {
		return nil, fmt.Errorf("error getting app state in GetAuthSessionByID: %w", err)
	}

	appState.authMapMutex.RLock()
	authSession, ok := appState.authMap[sessionID]
	appState.authMapMutex.RUnlock()
	if !ok {
		return nil, fmt.Errorf("auth session not found")
	}
	return &authSession, nil
}

func UpdateAuthSession(sessionID string, newAuthSession *types.AuthSession) error {
	appState, err := GetAppState()
	if err != nil {
		return fmt.Errorf("error getting app state in UpdateAuthSession: %w", err)
	}

	if newAuthSession == nil || newAuthSession.SessionID == "" {
		return fmt.Errorf("new auth session is nil or has empty session ID")
	}

	appState.authMapMutex.Lock()
	defer appState.authMapMutex.Unlock()
	if authSession, ok := appState.authMap[sessionID]; !ok || authSession.SessionID == "" {
		return fmt.Errorf("auth session not found for session ID: %s", sessionID)
	}

	appState.authMap[sessionID] = *newAuthSession
	return nil
}

// SignSessionToken returns HMAC-SHA256(token) using a server-side secret key.
func SignSessionToken(clientToken string, serverSecret []byte) (string, error) {
	hmacHash := hmac.New(sha256.New, serverSecret)
	hmacHash.Write([]byte(clientToken))
	return hex.EncodeToString(hmacHash.Sum(nil)), nil
}

// Check SessionToken by hashing the client token and comparing to server-side hash
func IsSessionTokenValid(clientToken string, serverSecret []byte) bool {
	hmacHash := hmac.New(sha256.New, serverSecret)
	hmacHash.Write([]byte(clientToken))
	computedHash := hmacHash.Sum(nil)
	return hmac.Equal(computedHash, serverSecret)
}

func GetServerSecret() ([]byte, error) {
	appState, err := GetAppState()
	if err != nil {
		return nil, fmt.Errorf("error getting app state in GetServerSecret: %w", err)
	}
	return appState.sessionSecret, nil
}
