package auth

import (
	"sync"
)

func CountAuthSessions(m *sync.Map) int {
	authSessionCount := 0
	m.Range(func(_, _ any) bool {
		authSessionCount++
		return true
	})
	return authSessionCount
}
