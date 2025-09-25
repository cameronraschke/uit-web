package config

import (
	"errors"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

func GetLimiter(limiterType string) *LimiterMap {
	appState := GetAppState()
	if appState == nil {
		return nil
	}

	switch limiterType {
	case "file":
		return appState.FileLimiter
	case "web":
		return appState.WebServerLimiter
	case "api":
		return appState.APILimiter
	case "auth":
		return appState.AuthLimiter
	default:
		return nil
	}
}

func (limiterMap *LimiterMap) Get(key string) *rate.Limiter {
	if limiterMap == nil {
		return nil
	}
	if value, ok := limiterMap.M.Load(key); ok {
		if e, ok2 := value.(LimiterEntry); ok2 && e.Limiter != nil {
			e.LastSeen = time.Now()
			limiterMap.M.Store(key, e)
			return e.Limiter
		}
	}
	limiter := rate.NewLimiter(rate.Limit(limiterMap.Rate), limiterMap.Burst)
	limiterMap.M.Store(key, LimiterEntry{Limiter: limiter, LastSeen: time.Now()})
	return limiter
}

func (blockedMap *BlockedMap) Block(ip string) {
	if blockedMap == nil || ip == "" {
		return
	}
	blockedMap.M.Store(ip, LimiterEntry{LastSeen: time.Now()})
}

func IsClientRateLimited(limiterType, ip string) (limited bool, retryAfter time.Duration) {
	appState := GetAppState()
	if appState == nil || strings.TrimSpace(ip) == "" {
		return false, 0
	}

	// Check if IP is currently blocked
	if v, ok := appState.BlockedIPs.M.Load(ip); ok {
		if e, ok2 := v.(LimiterEntry); ok2 {
			blockedUntil := e.LastSeen.Add(appState.BlockedIPs.BanPeriod)
			if curTime := time.Now(); curTime.Before(blockedUntil) {
				return true, blockedUntil.Sub(curTime)
			}
			// If ban has expired, remove from blocked list
			appState.BlockedIPs.M.Delete(ip)
		}
	}

	limiterMap := GetLimiter(limiterType)
	if limiterMap == nil {
		return false, 0
	}

	limiter := limiterMap.Get(ip)
	curTime := time.Now()
	// Reserve 1 token
	reserve := limiter.ReserveN(curTime, 1)
	delay := reserve.DelayFrom(curTime)

	// Wait until we can proceed, or block if too many requests
	if delay > 0 || !reserve.OK() {
		reserve.Cancel()
		appState.BlockedIPs.Block(ip)
		return true, delay
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
	appState.WebServerLimiter.M.Range(func(key, value any) bool {
		limiterEntry, ok := value.(*LimiterEntry)
		if !ok {
			return true
		}
		if now.Sub(limiterEntry.LastSeen) > 3*time.Minute {
			appState.WebServerLimiter.M.Delete(key)
			count++
		}
		return true
	})
	return int64(count), nil
}
