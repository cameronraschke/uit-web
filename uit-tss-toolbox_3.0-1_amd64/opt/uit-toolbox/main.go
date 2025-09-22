package main

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	auth "uit-toolbox/auth"
	config "uit-toolbox/config"
	"uit-toolbox/database"
	get "uit-toolbox/get"
	logger "uit-toolbox/logger"
	middleware "uit-toolbox/middleware"

	_ "net/http/pprof"

	_ "github.com/jackc/pgx/v5/stdlib"
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

func checkAuthSession(authMap *sync.Map, requestIP string, requestBasicToken string, requestBearerToken string) (basicValid bool, bearerValid bool, basicTTL float64, bearerTTL float64, matchedSession *AuthSession) {
	basicValid = false
	bearerValid = false
	basicTTL = 0.0
	bearerTTL = 0.0
	matchedSession = nil

	authMap.Range(func(k, v any) bool {
		sessionID := k.(string)
		authSession := v.(AuthSession)
		sessionIP := strings.SplitN(sessionID, ":", 2)[0]

		basicExists := strings.TrimSpace(authSession.Basic.Token) != ""
		bearerExists := strings.TrimSpace(authSession.Bearer.Token) != ""

		if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(sessionIP) == "" {
			authMap.Delete(sessionID)
			atomic.AddInt64(&authMapEntryCount, -1)
			return true
		}

		if !basicExists && !bearerExists {
			authMap.Delete(sessionID)
			atomic.AddInt64(&authMapEntryCount, -1)
			return true
		}

		if basicExists &&
			strings.TrimSpace(requestBasicToken) == strings.TrimSpace(authSession.Basic.Token) &&
			requestIP == authSession.Basic.IP {
			if time.Now().Before(authSession.Basic.Expiry) && authSession.Basic.Valid {
				basicValid = true
				basicTTL = time.Until(authSession.Basic.Expiry).Seconds()
				matchedSession = &authSession
			} else {
				log.Debug("Basic token found but expired/invalid for IP: " + sessionIP)
			}
		}

		if bearerExists &&
			strings.TrimSpace(requestBearerToken) == strings.TrimSpace(authSession.Bearer.Token) &&
			requestIP == authSession.Bearer.IP {
			if time.Now().Before(authSession.Bearer.Expiry) && authSession.Bearer.Valid {
				bearerValid = true
				bearerTTL = time.Until(authSession.Bearer.Expiry).Seconds()
				matchedSession = &authSession
			} else {
				log.Debug("Bearer token found but expired/invalid for IP: " + sessionIP)
			}
		}

		if basicValid || bearerValid {
			return false
		}
		return true
	})

	return
}

func fileServerHandler(appState *config.AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("no IP address stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		requestURL, ok := GetRequestURL(req)
		if !ok {
			log.Warning("no URL stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		fullPath, resolvedPath, requestedFile, ok := GetRequestedFile(req)
		if !ok {
			log.Warning("no requested file stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		if resolvedPath != fullPath {
			log.Warning("Resolved path does not match full path (" + requestIP + "): " + resolvedPath + " -> " + fullPath)
			http.Error(w, formatHttpError("Forbidden"), http.StatusForbidden)
			return
		}

		log.Debug("File request from " + requestIP + " for " + requestURL)

		// Previous path and file validation done in middleware
		// Open the file
		f, err := os.Open(fullPath)
		if err != nil {
			log.Warning("File not found: " + fullPath + " (" + err.Error() + ")")
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		defer f.Close()

		err = f.SetDeadline(time.Now().Add(30 * time.Second))
		if err != nil {
			log.Error("Cannot set file read deadline: " + fullPath + " (" + err.Error() + ")")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		metadata, err := f.Stat()
		if err != nil {
			log.Error("Cannot stat file: " + fullPath + " (" + err.Error() + ")")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		maxFileSize := int64(10) << 30 // 10 GiB
		if metadata.Size() > maxFileSize {
			log.Warning("File too large: " + fullPath + " (" + fmt.Sprintf("%d", metadata.Size()) + " bytes)")
			http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
			return
		}

		// Get file info for headers
		if strings.HasSuffix(fullPath, ".deb") {
			w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
		} else if strings.HasSuffix(fullPath, ".gz") {
			w.Header().Set("Content-Type", "application/gzip")
		} else if strings.HasSuffix(fullPath, ".img") {
			w.Header().Set("Content-Type", "application/vnd.efi.img")
		} else if strings.HasSuffix(fullPath, ".iso") {
			w.Header().Set("Content-Type", "application/vnd.efi.iso")
		} else if strings.HasSuffix(fullPath, ".squashfs") {
			w.Header().Set("Content-Type", "application/octet-stream")
		} else if strings.HasSuffix(fullPath, ".crt") {
			w.Header().Set("Content-Type", "application/x-x509-ca-cert")
		} else if strings.HasSuffix(fullPath, ".pem") {
			w.Header().Set("Content-Type", "application/pem-certificate-chain")
		} else {
			w.Header().Set("Content-Type", "application/octet-stream")
		}

		// Set headers
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", metadata.Size()))
		w.Header().Set("Content-Disposition", "attachment; filename=\""+metadata.Name()+"\"")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Last-Modified", metadata.ModTime().UTC().Format(http.TimeFormat))
		w.Header().Set("ETag", fmt.Sprintf(`"%x-%x"`, metadata.ModTime().Unix(), metadata.Size()))
		w.Header().Set("Cache-Control", "private, max-age=300")

		// Serve the file
		http.ServeContent(w, req, metadata.Name(), metadata.ModTime(), f)

		if ctx.Err() != nil {
			log.Warning("Request cancelled while serving file: " + requestedFile + " to " + requestIP + " (" + ctx.Err().Error() + ")")
			return
		}

		log.Info("Served file: " + requestedFile + " to " + requestIP)
	}
}

func webServerHandler(appState *config.AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("no IP address stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		requestURL, ok := GetRequestURL(req)
		if !ok {
			log.Warning("no URL stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		fullPath, resolvedPath, requestedFile, ok := GetRequestedFile(req)
		if !ok {
			log.Warning("no requested file stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}

		if resolvedPath != fullPath {
			log.Warning("Resolved path does not match full path (" + requestIP + "): " + resolvedPath + " -> " + fullPath)
			http.Error(w, formatHttpError("Forbidden"), http.StatusForbidden)
			return
		}

		log.Debug("File request from " + requestIP + " for " + requestURL)

		// Previous path and file validation done in middleware
		// Open the file
		f, err := os.Open(fullPath)
		if err != nil {
			log.Warning("File not found: " + fullPath + " (" + err.Error() + ")")
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		defer f.Close()

		err = f.SetDeadline(time.Now().Add(30 * time.Second))
		if err != nil {
			log.Error("Cannot set file read deadline: " + fullPath + " (" + err.Error() + ")")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		metadata, err := f.Stat()
		if err != nil {
			log.Error("Cannot stat file: " + fullPath + " (" + err.Error() + ")")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		maxFileSize := int64(128) << 20 // 128 MiB
		if metadata.Size() > maxFileSize {
			log.Warning("File too large: " + fullPath + " (" + fmt.Sprintf("%d", metadata.Size()) + " bytes)")
			http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
			return
		}

		// Set headers
		if strings.HasSuffix(requestedFile, ".html") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			// Parse the template
			htmlTemp, err := template.ParseFiles(resolvedPath)
			if err != nil {
				log.Warning("Cannot parse template file (" + resolvedPath + "): " + err.Error())
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			// Execute the template
			err = htmlTemp.Execute(w, nil)
			if err != nil {
				log.Error("Error executing template for " + resolvedPath + ": " + err.Error())
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			return
		} else if strings.HasSuffix(requestedFile, ".css") {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		} else if strings.HasSuffix(requestedFile, ".js") {
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		} else if strings.HasSuffix(requestedFile, ".ico") {
			w.Header().Set("Content-Type", "image/x-icon")
		} else {
			log.Warning("Unknown file type requested: " + requestedFile)
			http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
			return
		}

		// Set headers
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'self'")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", metadata.Size()))
		w.Header().Set("Content-Disposition", "inline; filename=\""+metadata.Name()+"\"")
		w.Header().Set("Last-Modified", metadata.ModTime().UTC().Format(http.TimeFormat))
		w.Header().Set("ETag", fmt.Sprintf(`"%x-%x"`, metadata.ModTime().Unix(), metadata.Size()))
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

		// Serve the file
		http.ServeContent(w, req, metadata.Name(), metadata.ModTime(), f)

		if ctx.Err() != nil {
			log.Warning("Request cancelled while serving file: " + requestedFile + " to " + requestIP + " (" + ctx.Err().Error() + ")")
			return
		}

		log.Info("Served file: " + requestedFile + " to " + requestIP)
	}
}

func logoutHandler(appState *config.AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		requestIP, ok := GetRequestIP(req)
		if !ok {
			log.Warning("no IP address stored in context")
			http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
			return
		}
		// Invalidate cookies
		cookie, err := req.Cookie("uit_basic_token")
		if err != nil && err != http.ErrNoCookie {
			log.Warning("Error retrieving Basic token cookie for logout: " + err.Error() + " (" + requestIP + ")")
			http.Redirect(w, req, "/login.html", http.StatusSeeOther)
			return
		}
		if cookie == nil || strings.TrimSpace(cookie.Value) == "" {
			log.Info("No Basic token cookie provided for logout: " + requestIP)
			http.Redirect(w, req, "/login.html", http.StatusSeeOther)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "uit_basic_token",
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "csrf_token",
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
		})
		authMap.Range(func(k, v any) bool {
			sessionID := k.(string)
			authSession := v.(AuthSession)
			// Diagnostic log to help determine why sessions may not match
			if authSession.Basic.Token == cookie.Value && authSession.Basic.IP == requestIP {
				log.Debug("Logout scan: sessionID=" + sessionID + " basic.token=" + authSession.Basic.Token + " basic.ip=" + authSession.Basic.IP + " cookie=" + cookie.Value + " requestIP=" + requestIP)
				authMap.Delete(sessionID)
				atomic.AddInt64(&authMapEntryCount, -1)
				log.Info("Invalidated session: " + sessionID)
				return false
			}
			return true
		})
		// Redirect to login page
		http.Redirect(w, req, "/login.html", http.StatusSeeOther)
	}
}

func rejectRequest(w http.ResponseWriter, req *http.Request) {
	requestIP, ok := GetRequestIP(req)
	if !ok {
		log.Warning("no IP address stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	requestURL, ok := GetRequestURL(req)
	if !ok {
		log.Warning("no URL stored in context")
		http.Error(w, formatHttpError("Internal server error"), http.StatusInternalServerError)
		return
	}

	log.Warning("access denied: " + requestIP + " tried to access " + requestURL)
	http.Error(w, "Access denied", http.StatusForbidden)
}

func main() {
	debug.PrintStack()

	log := logger.CreateLogger("console", logger.ParseLogLevel(os.Getenv("UIT_API_LOG_LEVEL")))

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
		return
	}

	dbName, dbHost, dbPort, dbUsername, dbPassword := config.GetDatabaseCredentials()

	dbConn, err := database.NewDBConnection(dbName, dbHost, dbPort, dbUsername, dbPassword)
	if err != nil {
		log.Error("Failed to connect to database: " + err.Error())
		return
	}

	config.SetDatabaseConn(dbConn)
	defer dbConn.Close()

	mw := NewMiddlewareFactory(appState, appConfig)

	fileServerBaseChain := muxChain{
		mw.LimitRequestSize(),
		mw.Timeout(),
		mw.StoreClientIP(),
		mw.CheckValidURL(),
		mw.AllowIPRange(appConfig.UIT_ALL_ALLOWED_IP),
		mw.RateLimit("file"),
		mw.HTTPMethod(),
		mw.CheckHeaders(),
		mw.SetHeaders(),
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
