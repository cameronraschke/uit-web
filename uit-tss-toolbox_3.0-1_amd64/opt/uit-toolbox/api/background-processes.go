package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

func backgroundProcesses(appState *AppState) {
	// Start auth map cleanup goroutine
	startAuthMapCleanup(15 * time.Second)
	// Start IP blocklist cleanup goroutine
	startIPBlocklistCleanup(appState, 1*time.Minute)
	// Print auth session count every minute
	printAuthMapCount(1 * time.Minute)
	// Start memory monitor goroutine
	startMemoryMonitor(4000*1024*1024, 5*time.Second) // 4GB limit, check every 5s
}

func startAuthMapCleanup(interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			authMap.Range(func(k, v any) bool {
				sessionID := k.(string)
				authSession := v.(AuthSession)
				sessionIP := strings.SplitN(sessionID, ":", 2)[0]

				// basicExpiry := authSession.Basic.Expiry.Sub(time.Now())
				bearerExpiry := time.Until(authSession.Bearer.Expiry)

				// Auth cache entry expires once countdown reaches zero
				if bearerExpiry.Seconds() <= 0 {
					authMap.Delete(sessionID)
					atomic.AddInt64(&authMapEntryCount, -1)
					sessionCount := countAuthSessions(&authMap)
					log.Info("(Cleanup) Auth session expired: " + sessionIP + " (TTL: " + fmt.Sprintf("%.2f", bearerExpiry.Seconds()) + ", " + strconv.Itoa(int(sessionCount)) + " session(s))")
				}
				return true
			})
		}
	}()
}

func printAuthMapCount(interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			sessionCount := countAuthSessions(&authMap)
			log.Info("Current auth sessions: " + strconv.Itoa(int(sessionCount)))
		}
	}()
}

func startIPBlocklistCleanup(appState *AppState, interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)

			// Get all banned IPs
			log.Info("(Background) Current blocked IPs: " + appState.GetAllBlockedIPs())

			appState.Cleanup()
		}
	}()
}

func startMemoryMonitor(maxBytes uint64, interval time.Duration) {
	go func() {
		var m runtime.MemStats
		for {
			time.Sleep(interval)
			runtime.ReadMemStats(&m)
			if m.Alloc > maxBytes {
				log.Error(fmt.Sprintf("Memory usage exceeded: %d bytes > %d bytes", m.Alloc, maxBytes))
				os.Exit(1)
			}
		}
	}()
}
