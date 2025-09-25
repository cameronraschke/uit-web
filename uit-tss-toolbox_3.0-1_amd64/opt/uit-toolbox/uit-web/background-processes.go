package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"
	config "uit-toolbox/config"
)

func backgroundProcesses() {
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
	log := config.GetLogger()
	go func() {
		for {
			time.Sleep(interval)
			config.ClearExpiredAuthSessions()
			sessionCount := config.RefreshAndGetAuthSessionCount()
			log.Info("Auth session cleanup done (Sessions: " + strconv.Itoa(int(sessionCount)) + ")")
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
