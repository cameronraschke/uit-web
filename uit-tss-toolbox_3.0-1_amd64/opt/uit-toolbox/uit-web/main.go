package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
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

	startTime := time.Now()
	fmt.Fprintln(os.Stdout, "Server time: "+startTime.Format("01-02-2006 15:04:05"))

	// Recover from panics
	defer func() {
		if recoveryErr := recover(); recoveryErr != nil {
			fmt.Fprintln(os.Stderr, "Panic: "+fmt.Sprint(recoveryErr))
			fmt.Fprintln(os.Stderr, "Stack:\n"+string(debug.Stack()))
			time.Sleep(100 * time.Millisecond) // Buffer
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
	log = log.With(slog.String("func", "main"))

	go func() {
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Error("pprof server error: " + err.Error())
		}
	}()

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

	if err := config.SetDatabaseConn(dbConn); err != nil {
		log.Error("Failed to set database connection: " + err.Error())
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	errChan := make(chan error, 10) // Buffer in case of multiple errors

	// Start HTTP server
	log.Info("Starting HTTP server on http://" + httpHost + ":8080")
	wg.Go(func() {
		if err := webserver.StartFileServer(ctx, httpHost); err != nil {
			errChan <- err
		}
	})

	// Start HTTPS server
	log.Info("Starting the HTTPS server on https://*:31411")
	wg.Go(func() {
		if err := webserver.StartWebServer(ctx); err != nil {
			errChan <- err
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
		backgroundProcesses(ctx)
	})

	// Wait for sigterm signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signalChan)

	log.Info("Servers started in: " + time.Since(startTime).String())

	select {
	case sig := <-signalChan:
		log.Info("Shutdown signal received: " + sig.String())
	case err := <-errChan:
		log.Error("Application error, exiting: " + err.Error())
	}

	cancel() // Cancel context to stop web servers

	// Wait for additional errors
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
				return // Timeout reached, continue
			}
		}
	}()

	// Wait for web servers to stop with timeout
	webServerShutdownCTX, webServerCTXCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer webServerCTXCancel()

	webServerDoneChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(webServerDoneChan)
	}()

	select {
	case <-webServerDoneChan:
		log.Info("Web servers gracefully stopped")
	case <-webServerShutdownCTX.Done():
		log.Error("Timeout reached, forcing exit of web servers")
		webServerCTXCancel()
	}

	// Close database connection with timeout
	dbCloseCtx, dbCloseCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer dbCloseCancel()

	dbDoneChan := make(chan struct{})
	go func() {
		dbConn.Close()
		close(dbDoneChan)
	}()

	select {
	case <-dbDoneChan:
		log.Info("Database connection gracefully closed")
	case <-dbCloseCtx.Done():
		log.Error("Timeout reached, forcing closure of database connection")
		dbCloseCancel()
	}

	log.Info("UIT Web has been stopped.")
}
