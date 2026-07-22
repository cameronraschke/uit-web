package types

import (
	"fmt"
	"net/http"
	"net/netip"
	"sync"
	"time"
)

const (
	AuthSessionTTL = 20 * time.Minute
	BasicTTL       = 20 * time.Minute
	BearerTTL      = 20 * time.Minute
	CSRFTTL        = 20 * time.Minute
)

type AuthSession struct {
	IPAddress     netip.Addr
	SessionID     string
	SessionTTL    time.Duration
	SessionCookie *http.Cookie
	BasicToken    BasicToken
	BasicCookie   *http.Cookie
	BearerToken   BearerToken
	BearerCookie  *http.Cookie
	CSRFToken     CSRFToken
	CSRFCookie    *http.Cookie
	Attributes    *SessionAttributes
}

func (authSession *AuthSession) ExtendSessionTTL(d time.Duration) (updatedSession *AuthSession, err error) {
	if authSession == nil {
		return nil, fmt.Errorf("authSession is nil")
	}

	newAuthSession := *authSession // Create a copy of the current session

	newAuthSession.SessionTTL = d
	newAuthSession.SessionCookie.Expires = time.Now().Add(d)
	newAuthSession.SessionCookie.MaxAge = int(d.Seconds())
	newAuthSession.BasicToken.Expiry = time.Now().Add(d)
	newAuthSession.BasicToken.TTL = d
	newAuthSession.BasicCookie.Expires = time.Now().Add(d)
	newAuthSession.BasicCookie.MaxAge = int(d.Seconds())
	newAuthSession.BearerToken.Expiry = time.Now().Add(d)
	newAuthSession.BearerToken.TTL = d
	newAuthSession.BearerCookie.Expires = time.Now().Add(d)
	newAuthSession.BearerCookie.MaxAge = int(d.Seconds())
	newAuthSession.CSRFToken.Expiry = time.Now().Add(d)
	newAuthSession.CSRFToken.TTL = d
	newAuthSession.CSRFCookie.Expires = time.Now().Add(d)
	newAuthSession.CSRFCookie.MaxAge = int(d.Seconds())

	return &newAuthSession, nil
}

type SessionAttributes struct {
	mu         sync.RWMutex
	attributes map[string]any
}

func (a *SessionAttributes) GetAuthAttributes(key string) (any, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	value, exists := a.attributes[key]
	return value, exists
}

func (a *SessionAttributes) SetAuthAttributes(key string, value any) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.attributes == nil {
		a.attributes = make(map[string]any)
	}
	a.attributes[key] = value
}

type LoginRequest struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	TwoFactorCode string `json:"two_factor_code"`
}

type AuthStatusResponse struct {
	Status    string        `json:"status"`
	ExpiresAt time.Time     `json:"expires_at"`
	TTL       time.Duration `json:"ttl"`
}

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
