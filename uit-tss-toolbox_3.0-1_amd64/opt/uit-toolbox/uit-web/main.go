package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"slices"
	"time"
	config "uit-toolbox/config"
	"uit-toolbox/database"
	"uit-toolbox/endpoints"
	"uit-toolbox/logger"
	middleware "uit-toolbox/middleware"

	_ "net/http/pprof"
)

// Mux handlers
type muxChain []func(http.Handler) http.Handler

func (chain muxChain) thenFunc(handle http.HandlerFunc) http.Handler {
	return chain.then(handle)
}

func (chain muxChain) then(handle http.Handler) http.Handler {
	for _, fn := range slices.Backward(chain) {
		handle = fn(handle)
	}
	return handle
}

func main() {
	debug.PrintStack()

	bootLog := logger.CreateLogger("console", logger.ParseLogLevel(os.Getenv("UIT_API_LOG_LEVEL")))
	bootLog.Info("Server time: " + time.Now().Format("01-02-2006 15:04:05"))
	bootLog.Info("UIT API Starting...")

	// Recover from panics
	defer func() {
		if pan := recover(); pan != nil {
			bootLog.Error("Recovered. Error: \n" + fmt.Sprintf("%v", pan))
			bootLog.Error("Trace: \n" + string(debug.Stack()))
		}
	}()

	// Initialize application
	_, err := config.InitApp()
	if err != nil {
		fmt.Println("Failed to initialize application: " + err.Error())
		os.Exit(1)
	}

	log := config.GetLogger()
	log.Info("Server time: " + time.Now().Format("01-02-2006 15:04:05"))
	log.Info("UIT API Starting...")

	// Get DB credentials
	dbName, dbHost, dbPort, dbUsername, dbPassword := config.GetDatabaseCredentials()

	// Create DB connection
	dbConn, err := database.NewDBConnection(dbName, dbHost, dbPort, dbUsername, dbPassword)
	if err != nil {
		log.Error("Failed to connect to database: " + err.Error())
		return
	}

	config.SetDatabaseConn(dbConn)
	defer dbConn.Close()

	// Create admin user
	err = database.CreateAdminUser()
	if err != nil {
		log.Error("Failed to create admin user: " + err.Error())
		return
	}

	lanServerIP, _, err := config.GetWebServerIPs()
	if err != nil || lanServerIP == "" {
		log.Error("Cannot get LAN server IP: " + err.Error())
		os.Exit(1)
	}

	fileServerBaseChain := muxChain{
		middleware.LimitRequestSizeMiddleware,
		middleware.TimeoutMiddleware,
		middleware.StoreClientIPMiddleware,
		middleware.RateLimitMiddleware("file"),
		middleware.AllowIPRangeMiddleware("lan"),
		middleware.CheckHttpVersionMiddleware,
		middleware.TLSMiddleware,
		middleware.CheckValidURLMiddleware,
		middleware.HTTPMethodMiddleware,
		middleware.CheckHeadersMiddleware,
		middleware.SetHeadersMiddleware,
	}

	fileServerMuxChain := muxChain{
		middleware.AllowedFilesMiddleware,
	}

	fileServerFullChain := append(fileServerBaseChain, fileServerMuxChain...)

	httpMux := http.NewServeMux()
	httpMux.Handle("/client/", fileServerFullChain.thenFunc(endpoints.FileServerHandler))
	httpMux.Handle("/client", fileServerFullChain.thenFunc(endpoints.RejectRequest))
	httpMux.Handle("/", fileServerBaseChain.thenFunc(endpoints.RejectRequest))

	httpServer := &http.Server{
		Addr:         lanServerIP + ":8080",
		Handler:      httpMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 1 * time.Minute,
		IdleTimeout:  2 * time.Minute,
	}
	defer httpServer.Close()

	go func() {
		log.Info("HTTP server listening on http://" + lanServerIP + ":8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error: " + err.Error())
		}
	}()

	// https handlers and middleware chains
	httpsBaseChain := muxChain{
		middleware.LimitRequestSizeMiddleware,
		middleware.TimeoutMiddleware,
		middleware.StoreClientIPMiddleware,
		middleware.RateLimitMiddleware("web"),
		middleware.AllowIPRangeMiddleware("any"),
		middleware.CheckHttpVersionMiddleware,
		middleware.TLSMiddleware,
		middleware.CheckValidURLMiddleware,
		middleware.HTTPMethodMiddleware,
		middleware.CheckHeadersMiddleware,
		middleware.SetHeadersMiddleware,
	}

	// No allowedFilesMiddleware here, as API calls do not serve files
	httpsBaseAPIChain := muxChain{
		middleware.CookieAuthMiddleware,
	}

	httpsBaseCookieAuthChain := muxChain{
		middleware.AllowedFilesMiddleware,
		middleware.CookieAuthMiddleware,
	}

	httpsBaseLoginChain := muxChain{
		middleware.AllowedFilesMiddleware,
	}

	httpsLogoutChain := muxChain{
		middleware.CookieAuthMiddleware,
	}

	httpsFullCookieAuthChain := append(httpsBaseChain, httpsBaseCookieAuthChain...)
	httpsFullLoginChain := append(httpsBaseChain, httpsBaseLoginChain...)
	httpsFullLogoutChain := append(httpsBaseChain, httpsLogoutChain...)
	httpsFullAPIChain := append(httpsBaseChain, httpsBaseAPIChain...)

	httpsMux := http.NewServeMux()
	httpsMux.Handle("GET /api/server_time", httpsFullAPIChain.thenFunc(endpoints.GetServerTime))
	httpsMux.Handle("GET /api/lookup", httpsFullAPIChain.thenFunc(endpoints.GetClientLookup))
	httpsMux.Handle("GET /api/client/hardware/ids", httpsFullAPIChain.thenFunc(endpoints.GetHardwareIdentifiers))
	httpsMux.Handle("GET /api/client/bios", httpsFullAPIChain.thenFunc(endpoints.GetBiosData))
	httpsMux.Handle("GET /api/client/os", httpsFullAPIChain.thenFunc(endpoints.GetOSData))
	httpsMux.Handle("GET /api/job_queue/overview", httpsFullAPIChain.thenFunc(endpoints.GetJobQueueOverview))
	httpsMux.Handle("GET /api/job_queue/client/queued_job", httpsFullAPIChain.thenFunc(endpoints.GetClientQueuedJobs))
	httpsMux.Handle("GET /api/job_queue/client/job_available", httpsFullAPIChain.thenFunc(endpoints.GetClientAvailableJobs))
	httpsMux.Handle("GET /api/notes", httpsFullAPIChain.thenFunc(endpoints.GetNotes))
	httpsMux.Handle("POST /api/notes", httpsFullAPIChain.thenFunc(endpoints.InsertNewNote))
	httpsMux.Handle("GET /api/dashboard/inventory_summary", httpsFullAPIChain.thenFunc(endpoints.GetDashboardInventorySummary))

	httpsMux.Handle("GET /login", httpsFullLoginChain.thenFunc(endpoints.WebServerHandler))
	httpsMux.Handle("GET /login.html", httpsFullLoginChain.thenFunc(endpoints.WebServerHandler))
	httpsMux.Handle("POST /login.html", httpsFullLoginChain.thenFunc(endpoints.WebAuthEndpoint))
	httpsMux.Handle("GET /css/login.css", httpsFullLoginChain.thenFunc(endpoints.WebServerHandler))
	httpsMux.Handle("/js/login.js", httpsFullLoginChain.thenFunc(endpoints.WebServerHandler))
	httpsMux.Handle("/css/desktop.css", httpsFullLoginChain.thenFunc(endpoints.WebServerHandler))
	httpsMux.Handle("/favicon.png", httpsFullLoginChain.thenFunc(endpoints.WebServerHandler))

	httpsMux.Handle("GET /logout", httpsFullLogoutChain.thenFunc(endpoints.LogoutHandler))

	httpsMux.Handle("/js/", httpsFullCookieAuthChain.thenFunc(endpoints.WebServerHandler))
	httpsMux.Handle("/css/", httpsFullCookieAuthChain.thenFunc(endpoints.WebServerHandler))
	httpsMux.Handle("/", httpsFullCookieAuthChain.thenFunc(endpoints.WebServerHandler))
	// httpsMux.HandleFunc("/dbstats/", GetInfoHandler)

	log.Info("Starting web server")

	tlsConfig := &tls.Config{
		// MinVersion: tls.VersionTLS12, //0x0303
		MinVersion: tls.VersionTLS13, //0x0304
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		PreferServerCipherSuites: true,
		SessionTicketsDisabled:   true,
	}

	httpsServer := http.Server{
		Addr:           ":31411",
		Handler:        httpsMux,
		TLSConfig:      tlsConfig,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB header size max
	}
	defer httpsServer.Close()

	httpsServer.Protocols = new(http.Protocols)
	httpsServer.Protocols.SetHTTP1(false)
	httpsServer.Protocols.SetUnencryptedHTTP2(false)
	httpsServer.Protocols.SetHTTP2(true)

	log.Info("Web server ready and listening for requests on https://*:31411")

	webCertFile, ok := os.LookupEnv("UIT_TLS_CERT_FILE")
	if !ok {
		log.Error("Error getting UIT_TLS_CERT_FILE: variable not set")
		os.Exit(1)
	}
	webKeyFile, ok := os.LookupEnv("UIT_TLS_KEY_FILE")
	if !ok {
		log.Error("Error getting UIT_TLS_KEY_FILE: variable not set")
		os.Exit(1)
	}

	go func() {
		log.Info("Background processes starting...")
		backgroundProcesses()
	}()

	// Start HTTPS server
	if err := httpsServer.ListenAndServeTLS(webCertFile, webKeyFile); err != nil && err != http.ErrServerClosed {
		log.Error("Cannot start web server: " + err.Error())
		os.Exit(1)
	}
}
