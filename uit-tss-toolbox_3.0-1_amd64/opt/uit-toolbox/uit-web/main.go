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
	log := config.GetLogger()
	log.Info("Server time: " + time.Now().Format("01-02-2006 15:04:05"))
	log.Info("UIT API Starting...")

	// Recover from panics
	defer func() {
		if pan := recover(); pan != nil {
			log.Error("Recovered. Error: \n" + fmt.Sprintf("%v", pan))
			log.Error("Trace: \n" + string(debug.Stack()))
		}
	}()

	// Initialize application
	appState, err := config.InitApp()
	if err != nil {
		log.Error("Failed to initialize application: " + err.Error())
		os.Exit(1)
	}

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

	fileServerBaseChain := muxChain{
		middleware.LimitRequestSizeMiddleware,
		middleware.TimeoutMiddleware,
		middleware.StoreClientIPMiddleware,
		middleware.CheckValidURLMiddleware,
		middleware.AllowIPRangeMiddleware("lan"),
		middleware.RateLimitMiddleware("file"),
		middleware.HTTPMethodMiddleware,
		middleware.CheckHeadersMiddleware,
		middleware.SetHeadersMiddleware,
	}

	fileServerMuxChain := muxChain{
		middleware.AllowedFilesMiddleware(appState),
	}

	fileServerFullChain := append(fileServerBaseChain, fileServerMuxChain...)

	httpMux := http.NewServeMux()
	httpMux.Handle("/client/", fileServerFullChain.then(fileServerHandler(appState)))
	httpMux.Handle("/client", fileServerFullChain.thenFunc(rejectRequest))
	httpMux.Handle("/", fileServerBaseChain.thenFunc(rejectRequest))
	httpServer := &http.Server{
		Addr:         appConfig.UIT_LAN_IP_ADDRESS + ":8080",
		Handler:      httpMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 1 * time.Minute,
		IdleTimeout:  2 * time.Minute,
	}

	go func() {
		log.Info("HTTP server listening on http://" + appConfig.UIT_LAN_IP_ADDRESS + ":8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error: " + err.Error())
		}
	}()
	defer httpServer.Close()

	// go func() {
	//   err := http.ListenAndServe("localhost:6060", nil)
	//   if err != nil {
	//     log.Error("Profiler error: " + err.Error())
	//   }
	// }()

	// https handlers and middleware chains
	httpsBaseChain := muxChain{
		mw.LimitRequestSize(),
		mw.Timeout(),
		mw.StoreClientIP(),
		mw.CheckValidURL(),
		mw.AllowIPRange(appConfig.UIT_ALL_ALLOWED_IP),
		mw.RateLimit("web"),
		mw.TLS(),
		mw.HTTPMethod(),
		mw.CheckHeaders(),
		mw.SetHeaders(),
	}

	// No allowedFilesMiddleware here, as API calls do not serve files
	httpsBaseAPIChain := muxChain{
		middleware.APIAuth,
	}

	httpsBaseCookieAuthChain := muxChain{
		middleware.AllowedFilesMiddleware(appState),
		middleware.HTTPCookieAuth,
	}

	httpsFullCookieAuthChain := append(httpsBaseChain, httpsBaseCookieAuthChain...)
	httpsFullAPIChain := append(httpsBaseChain, httpsBaseAPIChain...)

	httpsMux := http.NewServeMux()
	httpsMux.Handle("GET /api/server_time", httpsFullAPIChain.thenFunc(get.GetServerTime))
	httpsMux.Handle("GET /api/lookup", httpsFullAPIChain.thenFunc(get.GetClientLookup))
	httpsMux.Handle("GET /api/client/hardware", httpsFullAPIChain.thenFunc(get.GetHardwareData))
	httpsMux.Handle("GET /api/client/bios", httpsFullAPIChain.thenFunc(get.GetBiosData))
	httpsMux.Handle("GET /api/client/os", httpsFullAPIChain.thenFunc(get.GetOSData))
	httpsMux.Handle("GET /api/job_queue/overview", httpsFullAPIChain.thenFunc(get.GetJobQueueOverview))
	httpsMux.Handle("GET /api/job_queue/client/queued_job", httpsFullAPIChain.thenFunc(get.GetClientQueuedJobs))
	httpsMux.Handle("GET /api/job_queue/client/job_available", httpsFullAPIChain.thenFunc(get.GetClientAvailableJobs))

	httpsMux.Handle("GET /login.html", httpsBaseChain.then(webServerHandler(appState)))
	httpsMux.Handle("POST /login.html", httpsBaseChain.thenFunc(auth.WebAuthEndpoint))
	httpsMux.Handle("/js/login.js", httpsBaseChain.then(webServerHandler(appState)))
	httpsMux.Handle("/css/desktop.css", httpsBaseChain.then(webServerHandler(appState)))
	httpsMux.Handle("/favicon.ico", httpsBaseChain.then(webServerHandler(appState)))

	httpsMux.Handle("GET /logout", httpsFullCookieAuthChain.then(logoutHandler(appState)))

	httpsMux.Handle("/js/", httpsFullCookieAuthChain.then(webServerHandler(appState)))
	httpsMux.Handle("/css/", httpsFullCookieAuthChain.then(webServerHandler(appState)))
	httpsMux.Handle("/", httpsFullCookieAuthChain.then(webServerHandler(appState)))
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

	httpsServer.Protocols = new(http.Protocols)
	httpsServer.Protocols.SetHTTP1(false)
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

	// Start HTTPS server
	if err := httpsServer.ListenAndServeTLS(webCertFile, webKeyFile); err != nil {
		log.Error("Cannot start web server: " + err.Error())
		os.Exit(1)
	}
	defer httpsServer.Close()
}
