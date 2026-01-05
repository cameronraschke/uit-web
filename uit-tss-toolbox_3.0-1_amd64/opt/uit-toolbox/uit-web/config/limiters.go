package config

import (
	"errors"
	"net/netip"
	"time"

	"golang.org/x/time/rate"
)

func GetLimiter(limiterType string) *RateLimiter {
	appState := GetAppState()
	if appState == nil {
		return nil
	}

	switch limiterType {
	case "file":
		return appState.FileLimiter.Load()
	case "web":
		return appState.WebServerLimiter.Load()
	case "api":
		return appState.APILimiter.Load()
	case "auth":
		return appState.AuthLimiter.Load()
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

func (blockedClients *BlockedClients) Block(ip netip.Addr) {
	if blockedClients == nil || ip == (netip.Addr{}) {
		return
	}
	blockedClients.ClientMap.Store(ip, ClientLimiter{LastSeen: time.Now()})
}

func IsClientRateLimited(limiterType string, ip netip.Addr) (limited bool, retryAfter time.Duration) {
	appState := GetAppState()
	if appState == nil || ip == (netip.Addr{}) {
		return false, 0
	}

	// Check if IP is currently blocked
	if clientMapValue, clientBlocked := appState.BlockedIPs.Load().ClientMap.Load(ip); clientBlocked {
		if clientLimiter, ok := clientMapValue.(ClientLimiter); ok {
			blockedUntil := clientLimiter.LastSeen.Add(appState.BlockedIPs.Load().BanPeriod)
			if curTime := time.Now(); curTime.Before(blockedUntil) {
				return true, blockedUntil.Sub(curTime)
			}
			// If ban has expired, remove from blocked list
			appState.BlockedIPs.Load().ClientMap.Delete(ip)
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
		appState.BlockedIPs.Load().Block(ip)
		return true, appState.BlockedIPs.Load().BanPeriod
	}

	return false, 0
}

func CleanupOldLimiterEntries() (int64, error) {
	appState := GetAppState()
	if appState == nil {
		return 0, errors.New("app state is not initialized")
	}
	now := time.Now()

	var count int
	// Clean up WebServerLimiter
	appState.WebServerLimiter.Load().ClientMap.Range(func(key, value any) bool {
		clientLimiter, ok := value.(ClientLimiter)
		if !ok {
			return true
		}
		if now.Sub(clientLimiter.LastSeen) > 3*time.Minute {
			appState.WebServerLimiter.Load().ClientMap.Delete(key)
			count++
		}
		return true
	})
	// File server limiter
	appState.FileLimiter.Load().ClientMap.Range(func(key, value any) bool {
		clientLimiter, ok := value.(ClientLimiter)
		if !ok {
			return true
		}
		if now.Sub(clientLimiter.LastSeen) > 3*time.Minute {
			appState.FileLimiter.Load().ClientMap.Delete(key)
			count++
		}
		return true
	})
	return int64(count), nil
}
