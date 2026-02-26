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
func GetAuthSessions() map[string]types.AuthSession {
	appState, err := GetAppState()
	if err != nil {
		return nil
	}
	authSessionsMap := make(map[string]types.AuthSession)
	appState.authMap.Range(func(k, v any) bool {
		key, keyExists := k.(string)
		value, valueExists := v.(types.AuthSession)
		if keyExists && valueExists {
			authSessionsMap[key] = value
		}
		return true
	})
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
		authSession, ok := v.(types.AuthSession)
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

func IsAuthSessionValid(checkedAuthSession *types.AuthSession, requestIP netip.Addr) (bool, error) {
	if checkedAuthSession == nil || checkedAuthSession.SessionID == "" || requestIP == (netip.Addr{}) || !requestIP.IsValid() {
		return false, fmt.Errorf("auth session and/or request IP is nil or invalid (IsAuthSessionValid)")
	}
	appState, err := GetAppState()
	if err != nil {
		return false, fmt.Errorf("cannot retrieve app state (IsAuthSessionValid): %w", err)
	}

	value, ok := appState.authMap.Load(checkedAuthSession.SessionID)
	if !ok {
		return false, nil
	}

	existingAuthSession, ok := value.(types.AuthSession)
	if !ok {
		return false, fmt.Errorf("invalid auth session type")
	}

	if existingAuthSession.SessionTTL <= 0 ||
		existingAuthSession.BasicToken.TTL <= 0 ||
		existingAuthSession.BearerToken.TTL <= 0 {
		// existingAuthSession.CSRFToken.TTL <= 0
		return false, fmt.Errorf("auth tokens have reached their TTL")
	}

	if existingAuthSession.SessionID != checkedAuthSession.SessionID ||
		existingAuthSession.BasicToken.Token != checkedAuthSession.BasicToken.Token ||
		existingAuthSession.BearerToken.Token != checkedAuthSession.BearerToken.Token {
		// existingAuthSession.CSRFToken.Token != checkedAuthSession.CSRFToken.Token
		return false, fmt.Errorf("request tokens do not match stored session tokens")
	}

	if existingAuthSession.BasicToken.Expiry.Before(time.Now()) ||
		existingAuthSession.BearerToken.Expiry.Before(time.Now()) {
		// existingAuthSession.CSRFToken.Expiry.Before(time.Now())
		return false, fmt.Errorf("auth tokens have expired")
	}

	if existingAuthSession.IPAddress != requestIP ||
		existingAuthSession.BasicToken.IP != requestIP ||
		existingAuthSession.BearerToken.IP != requestIP {
		// existingAuthSession.CSRFToken.IP != requestIP
		return false, fmt.Errorf("request IP does not match stored token IP")
	}

	return true, nil
}

func GetAuthSessionByID(sessionID string) (*types.AuthSession, error) {
	appState, err := GetAppState()
	if err != nil {
		return nil, fmt.Errorf("error getting app state in GetAuthSessionByID: %w", err)
	}
	value, ok := appState.authMap.Load(sessionID)
	if !ok {
		return nil, nil
	}
	authSession, ok := value.(types.AuthSession)
	if !ok {
		return nil, errors.New("invalid auth session type")
	}
	return &authSession, nil
}

func UpdateAuthSession(sessionID string, authSession *types.AuthSession) error {
	appState, err := GetAppState()
	if err != nil {
		return fmt.Errorf("error getting app state in UpdateAuthSession: %w", err)
	}
	_, ok := appState.authMap.Load(sessionID)
	if !ok {
		return fmt.Errorf("session ID not found: %s", sessionID)
	}
	appState.authMap.Store(sessionID, *authSession)
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
