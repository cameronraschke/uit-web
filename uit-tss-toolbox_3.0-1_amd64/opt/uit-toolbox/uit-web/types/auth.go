package types

import (
	"net/http"
	"net/netip"
	"time"
)

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
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
}

const (
	AuthSessionTTL = 20 * time.Minute
	BasicTTL       = 20 * time.Minute
	BearerTTL      = 20 * time.Minute
	CSRFTTL        = 20 * time.Minute
)
