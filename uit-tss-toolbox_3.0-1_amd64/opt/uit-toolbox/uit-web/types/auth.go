package types

import (
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
