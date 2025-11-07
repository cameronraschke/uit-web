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
	"uit-toolbox/logger"
	"uit-toolbox/webserver"

	_ "net/http/pprof"
)

func main() {
	bootLog := logger.CreateLogger("console", logger.ParseLogLevel("info"))
	startTime := time.Now()
	bootLog.Info("Server time: " + startTime.Format("01-02-2006 15:04:05"))
	bootLog.Info("UIT API Starting...")

	// Recover from panics
	defer func() {
		if recoveryErr := recover(); recoveryErr != nil {
			bootLog.Error("Panic: " + fmt.Sprint(recoveryErr))
			bootLog.Error("Stack:\n" + string(debug.Stack()))
			time.Sleep(10 * time.Millisecond) // Buffer
			os.Exit(1)
		}
	}()

	// Initialize application
	if _, err := config.InitApp(); err != nil {
		bootLog.Error("Failed to initialize application: " + err.Error())
		os.Exit(1)
	}

	log := config.GetLogger()

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

	config.SetDatabaseConn(dbConn)

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	// Start HTTP file server
	log.Info("Starting HTTP file server on http://" + httpHost + ":8080")
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := webserver.StartFileServer(ctx, httpHost); err != nil {
			errChan <- err
		}
	}()

	// Start HTTPS web server
	log.Info("Starting HTTPS web server on https://*:31411")
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := webserver.StartWebServer(ctx); err != nil {
			errChan <- err
		}
	}()

	// Start background processes
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if recoveryErr := recover(); recoveryErr != nil {
				log.Error("Background process panic: " + fmt.Sprint(recoveryErr))
			}
		}()
		log.Info("Starting background processes...")
		backgroundProcesses(ctx, errChan)
	}()

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
	go func() {
		timeout := time.After(1 * time.Second)
		for {
			select {
			case err := <-errChan:
				log.Error("Additional error: " + err.Error())
			case <-timeout:
				return
			}
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("UIT API stopped.")
	case <-time.After(35 * time.Second): // 5s buffer beyond server timeout
		log.Error("Shutdown timed out, forcing exit")
	}
}
