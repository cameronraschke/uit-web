package auth

import (
	"sync"
	"time"
)

type BasicToken struct {
	Token     string    `json:"token"`
	Expiry    time.Time `json:"expiry"`
	NotBefore time.Time `json:"not_before"`
	TTL       float64   `json:"ttl"`
	IP        string    `json:"ip"`
	Valid     bool      `json:"valid"`
}

type BearerToken struct {
	Token     string    `json:"token"`
	Expiry    time.Time `json:"expiry"`
	NotBefore time.Time `json:"not_before"`
	TTL       float64   `json:"ttl"`
	IP        string    `json:"ip"`
	Valid     bool      `json:"valid"`
}

type AuthSession struct {
	Basic  BasicToken
	Bearer BearerToken
	CSRF   string
}

func CountAuthSessions(m *sync.Map) int {
	authSessionCount := 0
	m.Range(func(_, _ any) bool {
		authSessionCount++
		return true
	})
	return authSessionCount
}
