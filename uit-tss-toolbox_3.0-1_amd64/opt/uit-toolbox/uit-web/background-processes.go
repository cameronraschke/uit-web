package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"uit-toolbox/config"

	"golang.org/x/sync/errgroup"
)

func sendBackgroundLog(ctx context.Context, logChan chan<- string, msg string) bool {
	if logChan == nil {
		return false
	}
	select {
	case <-ctx.Done():
		return false
	case logChan <- msg:
		return true
	default:
		return false
	}
}

func backgroundProcesses(ctx context.Context, errChan chan error) {
	errGroup, errCtx := errgroup.WithContext(ctx)

	log := config.GetLogger().With(slog.String("func", "backgroundProcesses"))
	logChan := make(chan string, 10) // Buffered channel for log messages

	// Listen for log messages from background processes
	errGroup.Go(func() error {
		for {
			select {
			case msg, ok := <-logChan:
				if !ok {
					log.Info("Background process log channel closed")
					return nil
				}
				log.Info("(Background): " + msg)
			case <-errCtx.Done():
				log.Info("Background process log channel closed due to context cancellation")
				return nil
			}
		}
	})

	// Listen for errors on errChan
	errGroup.Go(func() error {
		select {
		case err := <-errChan:
			log.Info(fmt.Sprintf("Background process error, exiting: %v", err))
			return err
		case <-errCtx.Done():
			log.Info("Background processes stopping...")
			return nil
		}
	})

	// Start auth map cleanup goroutine
	errGroup.Go(func() error {
		interval := 5 * time.Minute
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-errCtx.Done():
				if !sendBackgroundLog(errCtx, logChan, "Auth map cleanup stopping...") {
					if err := errCtx.Err(); err != nil {
						return nil // No error on regular shutdown
					}
				}
				return nil
			case <-ticker.C:
				logMsg, err := startAuthMapCleanup()
				if err != nil {
					log.Error(fmt.Sprintf("Error during auth map cleanup: %v", err))
				}
				if !sendBackgroundLog(errCtx, logChan, logMsg) {
					if err := errCtx.Err(); err != nil {
						return nil // No error on regular shutdown
					}
				}
			}
		}
	})

	log.Info("Background processes started")
	if err := errGroup.Wait(); err != nil {
		log.Error(fmt.Sprintf("Background processes exited with error: %v", err))
	} else {
		log.Info("Background processes exited without error")
	}
}

func startAuthMapCleanup() (logMsg string, err error) {
	originalSessionCount := config.GetAuthSessionCount()
	config.ClearExpiredAuthSessions()
	newSessionCount := config.GetAuthSessionCount()
	sessionDiff := originalSessionCount - newSessionCount
	return fmt.Sprintf("Auth session cleanup done (Sessions: %d, Expired: %d)", newSessionCount, sessionDiff), nil
}
