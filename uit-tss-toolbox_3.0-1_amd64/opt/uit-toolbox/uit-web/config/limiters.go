package config

import (
	"errors"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

type ClientLimiter struct {
	IPAddr   netip.Addr
	Limiter  *rate.Limiter
	LastSeen time.Time
}

type RateLimiter struct {
	Type      string
	ClientMap sync.Map // map[netip.Addr]ClientLimiter
	Rate      float64
	Burst     int
}

var (
	webRateLimiter  atomic.Pointer[RateLimiter]
	apiRateLimiter  atomic.Pointer[RateLimiter]
	authRateLimiter atomic.Pointer[RateLimiter]
	fileRateLimiter atomic.Pointer[RateLimiter]
)

func GetLimiter(limiterType string) *RateLimiter {
	appState, err := GetAppState()
	if err != nil {
		return nil
	}

	switch limiterType {
	case "file":
		return appState.fileLimiter.Load()
	case "web":
		return appState.webServerLimiter.Load()
	case "api":
		return appState.apiLimiter.Load()
	case "auth":
		return appState.authLimiter.Load()
	default:
		return nil
	}
}

func (rateLimiter *RateLimiter) Get(ipAddr netip.Addr) *rate.Limiter {
	if rateLimiter == nil {
		return nil
	}
	if ipKey, ok := rateLimiter.ClientMap.Load(ipAddr); ok {
		if clientLimiter, ok2 := ipKey.(ClientLimiter); ok2 && clientLimiter.Limiter != nil {
			// Update last seen time
			clientLimiter.LastSeen = time.Now()
			// Store updated entry
			rateLimiter.ClientMap.Store(ipAddr, clientLimiter)
			return clientLimiter.Limiter
		}
	}
	limiter := rate.NewLimiter(rate.Limit(rateLimiter.Rate), rateLimiter.Burst)
	rateLimiter.ClientMap.Store(ipAddr, ClientLimiter{Limiter: limiter, LastSeen: time.Now()})
	return limiter
}

func (bannedClients *BanList) Block(ip netip.Addr) {
	if bannedClients == nil || ip == (netip.Addr{}) {
		return
	}
	bannedClients.bannedClients.Store(ip, ClientLimiter{LastSeen: time.Now()})
}

func IsClientRateLimited(limiterType string, ip netip.Addr) (limited bool, retryAfter time.Duration) {
	appState, err := GetAppState()
	if err != nil || ip == (netip.Addr{}) {
		return false, 0
	}

	// Check if IP is currently blocked
	if clientMapValue, clientBlocked := appState.banList.Load().bannedClients.Load(ip); clientBlocked {
		if clientLimiter, ok := clientMapValue.(ClientLimiter); ok {
			blockedUntil := clientLimiter.LastSeen.Add(appState.banList.Load().banPeriod)
			if curTime := time.Now(); curTime.Before(blockedUntil) {
				return true, blockedUntil.Sub(curTime)
			}
			// If ban has expired, remove from blocked list
			appState.banList.Load().bannedClients.Delete(ip)
		}
	}

	rateLimiter := GetLimiter(limiterType)
	if rateLimiter == nil {
		return false, 0
	}

	limiter := rateLimiter.Get(ip)

	// Use Allow() to check if the request can proceed immediately.
	// If it returns false, the rate limit has been exceeded.
	if !limiter.Allow() {
		appState.banList.Load().Block(ip)
		return true, appState.banList.Load().banPeriod
	}

	return false, 0
}

func CleanupOldLimiterEntries() (int64, error) {
	appState, err := GetAppState()
	if err != nil {
		return 0, errors.New("app state is not initialized")
	}
	now := time.Now()

	var count int
	// Clean up webServerLimiter
	appState.webServerLimiter.Load().ClientMap.Range(func(key, value any) bool {
		clientLimiter, ok := value.(ClientLimiter)
		if !ok {
			return true
		}
		if now.Sub(clientLimiter.LastSeen) > 3*time.Minute {
			appState.webServerLimiter.Load().ClientMap.Delete(key)
			count++
		}
		return true
	})
	// File server limiter
	appState.fileLimiter.Load().ClientMap.Range(func(key, value any) bool {
		clientLimiter, ok := value.(ClientLimiter)
		if !ok {
			return true
		}
		if now.Sub(clientLimiter.LastSeen) > 3*time.Minute {
			appState.fileLimiter.Load().ClientMap.Delete(key)
			count++
		}
		return true
	})
	return int64(count), nil
}
