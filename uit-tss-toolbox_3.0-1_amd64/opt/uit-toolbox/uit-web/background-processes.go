package main

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"
	config "uit-toolbox/config"
	"uit-toolbox/logger"
)

func backgroundProcesses(ctx context.Context, errChan chan error) {
	log := config.GetLogger()
	var wg sync.WaitGroup
	// Start auth map cleanup goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		startAuthMapCleanup(ctx, log, 15*time.Second)
	}()
	// Start IP blocklist cleanup goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		startIPBlocklistCleanup(ctx, log, 5*time.Minute)
	}()
	// Start IP limiter cleanup goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		startIPLimiterCleanup(ctx, log, 5*time.Minute)
	}()
	// Start memory monitor goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		startMemoryMonitor(ctx, log, 4000*1024*1024, 5*time.Second, errChan) // 4GB limit, check every 5s
	}()

	log.Info("Background processes started")
	wg.Wait() // Wait for all background goroutines to finish
	log.Info("Background processes stopped")
}

func startAuthMapCleanup(ctx context.Context, log logger.Logger, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info("Auth map cleanup stopping...")
			return
		case <-ticker.C:
			originalCount := config.GetAuthSessionCount()
			config.ClearExpiredAuthSessions()
			sessionCount := config.RefreshAndGetAuthSessionCount()
			if originalCount != sessionCount {
				log.Info("(Background) Auth session cleanup done (Sessions: " + strconv.Itoa(int(sessionCount)) + ")")
			}
		}
	}
}

func startIPBlocklistCleanup(ctx context.Context, log logger.Logger, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info("IP blocklist cleanup stopping...")
			return
		case <-ticker.C:
			config.CleanupBlockedIPs()
			log.Info("(Background) IP blocklist cleanup done")
		}
	}
}

func startIPLimiterCleanup(ctx context.Context, log logger.Logger, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info("IP limiter cleanup stopping...")
			return
		case <-ticker.C:
			config.CleanupOldLimiterEntries()
			log.Info("(Background) IP limiter cleanup done")
		}
	}
}

func startMemoryMonitor(ctx context.Context, log logger.Logger, maxBytes uint64, interval time.Duration, errChan chan error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var memStats runtime.MemStats
	for {
		select {
		case <-ctx.Done():
			log.Info("Memory monitor stopping...")
			return
		case <-ticker.C:
			runtime.ReadMemStats(&memStats)
			if memStats.Alloc > maxBytes {
				select {
				case errChan <- fmt.Errorf("memory usage exceeded: %d bytes > %d bytes", memStats.Alloc, maxBytes):
					return
				default:
					log.Error(fmt.Sprintf("Memory exceeded but cannot send error: %d > %d", memStats.Alloc, maxBytes))
				}
			}
		}
	}
}
