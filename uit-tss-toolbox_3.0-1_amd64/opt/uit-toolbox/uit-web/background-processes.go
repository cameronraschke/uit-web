package main

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"
	config "uit-toolbox/config"
)

func backgroundProcesses(ctx context.Context, errChan chan error) {
	select {
	case errChan <- fmt.Errorf("background process error, not starting backgound processes"):
		return
	default:
		break
	}
	log := config.GetLogger().With(slog.String("func", "backgroundProcesses"))
	var wg sync.WaitGroup
	// Start auth map cleanup goroutine
	wg.Go(func() {
		startAuthMapCleanup(ctx, log, 20*time.Minute)
	})
	// // Start IP blocklist cleanup goroutine
	// wg.Go(func() {
	// 	startIPBlocklistCleanup(ctx, log, 5*time.Minute)
	// })
	// // Start IP limiter cleanup goroutine
	// wg.Go(func() {
	// 	startIPLimiterCleanup(ctx, log, 5*time.Minute)
	// })
	// // Start memory monitor goroutine
	// wg.Go(func() {
	// 	startMemoryMonitor(ctx, log, 4000*1024*1024, 5*time.Minute, errChan) // 4GB application memory limit
	// })

	log.Info("Background processes started")
	wg.Wait() // Wait for all background goroutines to finish
	log.Info("Background processes stopped")
}

func startAuthMapCleanup(ctx context.Context, log *slog.Logger, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info("Auth map cleanup stopping...")
			return
		case <-ticker.C:
			originalSessionCount := config.GetAuthSessionCount()
			config.ClearExpiredAuthSessions()
			newSessionCount := config.RefreshAndGetAuthSessionCount()
			if originalSessionCount != newSessionCount {
				log.Info("(Background) Auth session cleanup done (Sessions: " + fmt.Sprintf("%d", newSessionCount) + ")")
			}
		}
	}
}

func startIPBlocklistCleanup(ctx context.Context, log *slog.Logger, interval time.Duration) {
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

func startIPLimiterCleanup(ctx context.Context, log *slog.Logger, interval time.Duration) {
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

func startMemoryMonitor(ctx context.Context, log *slog.Logger, maxBytes uint64, interval time.Duration, errChan chan error) {
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
