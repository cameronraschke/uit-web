package config

import (
	"errors"
	"strings"
	"time"
)

// Auth session management
func GetAuthSessions() map[string]AuthSession {
	appState := GetAppState()
	if appState == nil {
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

func CreateAuthSession(sessionID string, authSession AuthSession) error {
	appState := GetAppState()
	if appState == nil {
		return errors.New("app state is not initialized")
	}
	_, ok := appState.AuthMap.LoadOrStore(sessionID, authSession)
	if !ok {
		appState.AuthMapEntryCount.Add(1)
	} else {
		appState.AuthMap.Store(sessionID, authSession)
	}
	return nil
}

func DeleteAuthSession(sessionID string) {
	appState := GetAppState()
	if appState == nil {
		return
	}
	if _, ok := appState.AuthMap.LoadAndDelete(sessionID); ok {
		newVal := appState.AuthMapEntryCount.Add(-1)
		if newVal < 0 {
			appState.AuthMapEntryCount.Store(0)
		}
	}
}

func GetAuthSessionCount() int64 {
	appState := GetAppState()
	if appState == nil {
		return 0
	}
	return appState.AuthMapEntryCount.Load()
}

func RefreshAndGetAuthSessionCount() int64 {
	appState := GetAppState()
	if appState == nil {
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

func CheckAuthSessionExists(sessionID string, ipAddress string, basicToken string, bearerToken string, csrfToken string) (bool, bool, error) {
	sessionValid := false
	sessionExists := false

	appState := GetAppState()
	if appState == nil {
		return sessionValid, sessionExists, errors.New("app state is not initialized")
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

	if strings.TrimSpace(ipAddress) == "" || strings.TrimSpace(basicToken) == "" || strings.TrimSpace(bearerToken) == "" {
		return sessionValid, sessionExists, errors.New("empty IP address or token for session ID: " + sessionID)
	}

	if authSession.Basic.Token != basicToken || authSession.Bearer.Token != bearerToken {
		return sessionValid, sessionExists, nil
	}

	if authSession.Basic.Expiry.Before(curTime) || authSession.Bearer.Expiry.Before(curTime) {
		return sessionValid, sessionExists, nil
	}

	sessionValid = true
	return sessionValid, sessionExists, nil
}
