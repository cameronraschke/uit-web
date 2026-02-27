package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"
	config "uit-toolbox/config"
	"uit-toolbox/database"
	"uit-toolbox/webserver"

	_ "net/http/pprof"
)

func main() {
	fmt.Fprintln(os.Stdout, "Starting UIT API...")

	startTime := time.Now()
	fmt.Fprintln(os.Stdout, "Server time: "+startTime.Format("01-02-2006 15:04:05"))
	fmt.Fprintln(os.Stdout, "UIT API Starting...")

	// Recover from panics
	defer func() {
		if recoveryErr := recover(); recoveryErr != nil {
			fmt.Fprintln(os.Stderr, "Panic: "+fmt.Sprint(recoveryErr))
			fmt.Fprintln(os.Stderr, "Stack:\n"+string(debug.Stack()))
			time.Sleep(10 * time.Millisecond) // Buffer
			os.Exit(1)
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

	// Get DB credentials
	dbName, dbHost, dbPort, dbUsername, dbPassword, err := config.GetDatabaseCredentials()
	if err != nil {
		log.Error("Failed to get database credentials: " + err.Error())
		os.Exit(1)
	}

	// Create DB connection
	dbConn, err := database.NewDBConnection(dbName, dbHost, dbPort, dbUsername, dbPassword)
	if err != nil {
		log.Error("Failed to connect to database: " + err.Error())
		os.Exit(1)
	}
	defer dbConn.Close()

	if err := config.SetDatabaseConn(dbConn); err != nil {
		log.Error("Failed to set database connection: " + err.Error())
		os.Exit(1)
	}

	// Create admin user
	if err = database.CreateAdminUser(); err != nil {
		log.Error("Failed to create admin user: " + err.Error())
		os.Exit(1)
	}

	httpHost, _, err := config.GetWebServerIPs()
	if err != nil || strings.TrimSpace(httpHost) == "" {
		log.Error("Cannot get HTTP server IP: " + err.Error())
		os.Exit(1)
	}

	// Log allowed IPs
	as, err := config.GetAppState()
	if err != nil {
		log.Error("Failed to get app state: " + err.Error())
		os.Exit(1)
	}
	if as == nil {
		log.Error("App state is nil in main")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	errChan := make(chan error, 10) // Buffer for multiple errors

	// Start HTTP file server
	log.Info("Starting HTTP file server on http://" + httpHost + ":8080")
	wg.Go(func() {
		if err := webserver.StartFileServer(ctx, httpHost); err != nil {
			errChan <- err
		}
	})

	// Start HTTPS web server
	log.Info("Starting HTTPS web server on https://*:31411")
	wg.Go(func() {
		if err := webserver.StartWebServer(ctx); err != nil {
			errChan <- err
		}
	})

	// Start background processes
	wg.Go(func() {
		defer func() {
			if recoveryErr := recover(); recoveryErr != nil {
				log.Error("Background process panic: " + fmt.Sprint(recoveryErr))
				log.Error("Stack:\n" + string(debug.Stack()))
				select {
				case errChan <- fmt.Errorf("background process panic: %v", recoveryErr):
				default:
					log.Warn("Error channel full, cannot send panic error (func main - backgroundProcesses)")
				}
			}
		}()
		log.Info("Starting background processes...")
		backgroundProcesses(ctx)
	})

	// Wait for shutdown signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdown)

	log.Info("Servers started in: " + time.Since(startTime).String())

	select {
	case sig := <-shutdown:
		log.Info("Shutdown signal received: " + sig.String())
	case err := <-errChan:
		log.Error("Server error, exiting: " + err.Error())
	}

	cancel() // Cancel context to stop web servers
	// Drain additional errors
	go func() {
		deadline := time.After(1 * time.Second)
		for {
			select {
			case err, ok := <-errChan:
				if !ok {
					return // Channel closed, continue
				}
				log.Error("Additional error logged while shutting down: " + err.Error())
			case <-deadline:
				return // Timeout reached, stop draining and continue
			}
		}
	}()

	// Wait for web servers to stop with timeout
	webServerShutdownCTX, webServerCTXCancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer webServerCTXCancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		time.Sleep(50 * time.Millisecond)
		log.Info("Web servers stopped gracefully")
	case <-webServerShutdownCTX.Done():
		log.Error("Shutdown timed out, forcing exit")
	}

	// Close database connection with timeout
	dbCloseCtx, dbCloseCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dbCloseCancel()

	dbCloseDone := make(chan struct{})
	go func() {
		dbConn.Close()
		close(dbCloseDone)
	}()

	select {
	case <-dbCloseDone:
		log.Info("Database connection closed")
	case <-dbCloseCtx.Done():
		log.Error("Database close timed out")
	}

	log.Info("UIT API stopped.")
}
