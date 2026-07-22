package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"
	"uit-toolbox/config"
	"uit-toolbox/database"
	"uit-toolbox/webserver"
)

func main() {
	fmt.Fprintln(os.Stdout, "Starting UIT Web...")

	// Create root context
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGABRT,
		syscall.SIGTERM,
	)
	defer stop()

	startTime := time.Now()
	fmt.Fprintln(os.Stdout, "Server time: "+startTime.Format("01-02-2006 15:04:05"))

	// Recover from panics
	defer func() {
		if recoveryErr := recover(); recoveryErr != nil {
			fmt.Fprintln(os.Stderr, "Panic: "+fmt.Sprint(recoveryErr))
			fmt.Fprintln(os.Stderr, "Stack:\n"+string(debug.Stack()))
			time.Sleep(100 * time.Millisecond) // Buffer
			stop()                             // Cancel context to stop all goroutines
			return
		}
	}()

	// Initialize application
	if _, err := config.InitApp(); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to initialize application: "+err.Error())
		os.Exit(1)
	}

	log := config.GetLogger()
	if log == nil {
		fmt.Fprintln(os.Stderr, "Global logger is nil in main")
		os.Exit(1)
	}
	log = log.With(slog.String("func", "main"))

	// Get DB credentials
	dbConnectionInfo, err := config.GetDatabaseCredentials()
	if err != nil {
		log.Error("Failed to retrieve database credentials: " + err.Error())
		os.Exit(1)
	}

	// Create DB connection
	dbConn, err := database.NewDBConnection(dbConnectionInfo)
	if err != nil {
		log.Error("Failed to connect to the database: " + err.Error())
		os.Exit(1)
	}
	defer dbConn.Close()

	pgxPool, err := database.NewPGXPool(dbConnectionInfo)
	if err != nil {
		log.Error("Failed to connect to the pgx pool: " + err.Error())
		os.Exit(1)
	}
	defer pgxPool.Close()

	if err := config.SetDatabaseConn(dbConn); err != nil {
		log.Error("Failed to set database connection: " + err.Error())
		os.Exit(1)
	}
	if err := config.SetPGXPool(pgxPool); err != nil {
		log.Error("Failed to set pgx pool: " + err.Error())
		os.Exit(1)
	}

	// Create admin user
	if err = database.CreateAdminUser(); err != nil {
		log.Error("Failed to create admin user in DB: " + err.Error())
		os.Exit(1)
	}

	httpHost, _, err := config.GetWebServerIPs()
	if err != nil || strings.TrimSpace(httpHost) == "" {
		log.Error("Failed to retrieve HTTP server IP: " + err.Error())
		os.Exit(1)
	}

	as, err := config.GetAppState()
	if err != nil {
		log.Error("Failed to retrieve app state: " + err.Error())
		os.Exit(1)
	}
	if as == nil {
		log.Error("App state is nil after initialization")
		os.Exit(1)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 10) // Buffer in case of multiple errors

	wg.Go(func() {
		if err := webserver.StartPprofServer(ctx); err != nil {
			select {
			case errChan <- err:
			default:
				log.Warn("Error channel full, cannot send pprof server error (func main - StartPprofServer)")
			}
		}
	})

	// Start HTTP server
	log.Info("Starting HTTP server on http://" + httpHost + ":8080")
	wg.Go(func() {
		if err := webserver.StartFileServer(ctx, httpHost); err != nil {
			select {
			case errChan <- err:
			default:
				log.Warn("Error channel full, cannot send HTTP server error (func main - StartFileServer)")
			}
		}
	})

	// Start HTTPS server
	log.Info("Starting the HTTPS server on https://*:31411")
	wg.Go(func() {
		if err := webserver.StartWebServer(ctx); err != nil {
			select {
			case errChan <- err:
			default:
				log.Warn("Error channel full, cannot send HTTPS server error (func main - StartWebServer)")
			}
		}
	})

	// Start background processes
	wg.Go(func() {
		defer func() {
			if recoveryErr := recover(); recoveryErr != nil {
				log.Error("Background process panic: " + fmt.Sprintf("%v", recoveryErr))
				log.Error("Stack:\n" + string(debug.Stack()))
				select {
				case errChan <- fmt.Errorf("background process panic: %v", recoveryErr):
				default:
					log.Warn("Error channel full, cannot send panic error (func main - backgroundProcesses)")
				}
			}
		}()
		log.Info("Starting background processes...")
		backgroundProcesses(ctx, errChan)
	})

	log.Info("Servers started in: " + time.Since(startTime).String())

	writeLastHeardToDB := func() {
		uncancelledCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		realtimeDataMap, err := config.GetAllClientRealtimeData()
		if err != nil {
			log.Error("Failed to retrieve realtime data on app shutdown: " + err.Error())
			cancel()
			return
		}
		for tag, realtimeData := range realtimeDataMap {
			if err := database.UpdateClientLastHeard(uncancelledCtx, tag, realtimeData.LastHeard); err != nil {
				log.Error(fmt.Sprintf("Failed to write last heard for tag %d on app shutdown: %s", tag, err.Error()))
			}
		}
		cancel()
	}

	// Wait for shutdown signal or error
	select {
	case <-ctx.Done():
		log.Info("Shutdown signal received.")
		writeLastHeardToDB()
	case err := <-errChan:
		log.Error("Error received: " + err.Error())
		writeLastHeardToDB()
		stop() // Cancel context to stop all goroutines
	}

	waitCtx, waitCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer waitCancel()

	wgDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(wgDone)
	}()

	select {
	case <-wgDone:
		log.Info("All goroutines stopped gracefully")
	case <-waitCtx.Done():
		log.Error("Shutdown timeout reached: " + waitCtx.Err().Error())
		log.Error("Forcing process exit; goroutines still running")
		log.Error("Goroutine dump:\n" + string(debug.Stack()))
	}

	log.Info("UIT Web has been stopped.")
}
