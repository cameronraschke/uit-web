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
	startIPBlocklistCleanup(5 * time.Minute)
	// Start IP limiter cleanup goroutine
	startIPLimiterCleanup(5 * time.Minute)
	// Start memory monitor goroutine
	startMemoryMonitor(4000*1024*1024, 5*time.Second) // 4GB limit, check every 5s
}

func startAuthMapCleanup(interval time.Duration) {
	log := config.GetLogger()
	go func() {
		for {
			time.Sleep(interval)
			originalCount := config.GetAuthSessionCount()
			config.ClearExpiredAuthSessions()
			sessionCount := config.RefreshAndGetAuthSessionCount()
			if originalCount != sessionCount {
				log.Info("(Background) Auth session cleanup done (Sessions: " + strconv.Itoa(int(sessionCount)) + ")")
			}
		}
	}()
}

func startIPBlocklistCleanup(interval time.Duration) {
	log := config.GetLogger()
	go func() {
		for {
			time.Sleep(interval)
			config.CleanupBlockedIPs()
			log.Info("(Background) IP blocklist cleanup done")
		}
	}()
}

func startIPLimiterCleanup(interval time.Duration) {
	log := config.GetLogger()
	go func() {
		for {
			time.Sleep(interval)
			config.CleanupOldLimiterEntries()
			log.Info("(Background) IP limiter cleanup done")
		}
	}()
}

func startMemoryMonitor(maxBytes uint64, interval time.Duration) {
	go func() {
		var memStats runtime.MemStats
		for {
			time.Sleep(interval)
			runtime.ReadMemStats(&memStats)
			if memStats.Alloc > maxBytes {
				fmt.Printf("Memory usage exceeded: %d bytes > %d bytes", memStats.Alloc, maxBytes)
				os.Exit(1)
			}
		}
	}()
}
