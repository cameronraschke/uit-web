package webserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
	"uit-toolbox/config"
	"uit-toolbox/endpoints"
	"uit-toolbox/middleware"
)

func StartWebServer(ctx context.Context) error {
	log := config.GetLogger()

	// https handlers and middleware chains
	httpsBaseChain := middleware.NewChain(
		middleware.StoreLoggerMiddleware,
		middleware.PanicRecoveryMiddleware,
		middleware.LimitRequestSizeMiddleware,
		middleware.StoreClientIPMiddleware,
		middleware.CheckIPBlockedMiddleware,
		middleware.WebEndpointConfigMiddleware,
		middleware.TLSMiddleware,
		middleware.CheckHttpVersionMiddleware,
		middleware.AllowIPRangeMiddleware("any"),
		middleware.RateLimitMiddleware("web"),
		middleware.APITimeoutMiddleware,
		middleware.HTTPMethodMiddleware,
		middleware.CheckValidURLMiddleware,
		middleware.CheckHeadersMiddleware,
		middleware.SetHeadersMiddleware,
		middleware.CheckForRedirectsMiddleware,
	)

	// No allowedFilesMiddleware here, as API calls do not serve files
	httpsFullAPIChain := httpsBaseChain.Append(
		middleware.CookieAuthMiddleware,
	)

	httpsFullCookieAuthChain := httpsBaseChain.Append(
		middleware.AllowedFilesMiddleware,
		middleware.CookieAuthMiddleware,
	)

	httpsFullLoginChain := httpsBaseChain.Append(
		middleware.AllowedFilesMiddleware,
	)

	httpsFullLogoutChain := httpsBaseChain.Append(
		middleware.CookieAuthMiddleware,
	)

	httpsRouter := http.NewServeMux()

	// API GET endpoints
	httpsRouter.Handle("GET /api/server_time", httpsFullAPIChain.ThenFunc(endpoints.GetServerTime))
	httpsRouter.Handle("GET /api/lookup", httpsFullAPIChain.ThenFunc(endpoints.GetClientLookup))
	httpsRouter.Handle("GET /api/all_tags", httpsFullAPIChain.ThenFunc(endpoints.GetAllTags))
	httpsRouter.Handle("GET /api/client/hardware/ids", httpsFullAPIChain.ThenFunc(endpoints.GetHardwareIdentifiers))
	httpsRouter.Handle("GET /api/client/bios", httpsFullAPIChain.ThenFunc(endpoints.GetBiosData))
	httpsRouter.Handle("GET /api/client/os", httpsFullAPIChain.ThenFunc(endpoints.GetOSData))
	httpsRouter.Handle("GET /api/job_queue/overview", httpsFullAPIChain.ThenFunc(endpoints.GetJobQueueTable))
	httpsRouter.Handle("GET /api/job_queue/client/queued_job", httpsFullAPIChain.ThenFunc(endpoints.GetClientQueuedJobs))
	httpsRouter.Handle("GET /api/job_queue/client/job_available", httpsFullAPIChain.ThenFunc(endpoints.GetClientAvailableJobs))
	httpsRouter.Handle("GET /api/client/location_form_data", httpsFullAPIChain.ThenFunc(endpoints.GetLocationFormData))
	httpsRouter.Handle("GET /api/notes", httpsFullAPIChain.ThenFunc(endpoints.GetNotes))
	httpsRouter.Handle("GET /api/dashboard/inventory_summary", httpsFullAPIChain.ThenFunc(endpoints.GetDashboardInventorySummary))
	httpsRouter.Handle("GET /api/images/manifest", httpsFullAPIChain.ThenFunc(endpoints.GetClientImagesManifest))
	httpsRouter.Handle("GET /api/images", httpsFullAPIChain.ThenFunc(endpoints.GetImage))
	httpsRouter.Handle("GET /api/inventory", httpsFullAPIChain.ThenFunc(endpoints.GetInventoryTableData))
	httpsRouter.Handle("GET /api/models", httpsFullAPIChain.ThenFunc(endpoints.GetManufacturersAndModels))
	httpsRouter.Handle("GET /api/client/health/battery", httpsFullAPIChain.ThenFunc(endpoints.GetClientBatteryHealth))
	httpsRouter.Handle("GET /api/domains", httpsFullAPIChain.ThenFunc(endpoints.GetDomains))
	httpsRouter.Handle("GET /api/departments", httpsFullAPIChain.ThenFunc(endpoints.GetDepartments))
	httpsRouter.Handle("GET /api/check_auth", httpsFullAPIChain.ThenFunc(endpoints.RejectRequest))
	httpsRouter.Handle("GET /api/reports/battery/stats", httpsFullAPIChain.ThenFunc(endpoints.GetBatteryStandardDeviation))
	httpsRouter.Handle("GET /api/job_queue/all_jobs", httpsFullAPIChain.ThenFunc(endpoints.GetAllJobs))
	httpsRouter.Handle("GET /api/locations", httpsFullAPIChain.ThenFunc(endpoints.GetAllLocations))
	httpsRouter.Handle("GET /api/all_statuses", httpsFullAPIChain.ThenFunc(endpoints.GetAllStatuses))
	httpsRouter.Handle("GET /api/all_device_types", httpsFullAPIChain.ThenFunc(endpoints.GetAllDeviceTypes))

	// API POST endpoints
	httpsRouter.Handle("POST /api/notes", httpsFullAPIChain.ThenFunc(endpoints.InsertNewNote))
	httpsRouter.Handle("POST /api/update_inventory", httpsFullAPIChain.ThenFunc(endpoints.InsertInventoryUpdateForm))
	httpsRouter.Handle("POST /api/images/toggle_pin", httpsFullAPIChain.ThenFunc(endpoints.TogglePinImage))
	httpsRouter.Handle("POST /api/client/health/battery", httpsFullAPIChain.ThenFunc(endpoints.SetClientBatteryHealth))
	httpsRouter.Handle("POST /api/job_queue/update_all_online_clients", httpsFullAPIChain.ThenFunc(endpoints.SetAllJobs))
	httpsRouter.Handle("POST /api/job_queue/update_client_job", httpsFullAPIChain.ThenFunc(endpoints.SetClientJob))
	httpsRouter.Handle("POST /api/client/memory", httpsFullAPIChain.ThenFunc(endpoints.SetClientMemoryInfo))
	httpsRouter.Handle("POST /api/client/cpu/usage", httpsFullAPIChain.ThenFunc(endpoints.SetClientCPUUsage))
	httpsRouter.Handle("POST /api/client/cpu/temp", httpsFullAPIChain.ThenFunc(endpoints.SetClientCPUTemperature))
	httpsRouter.Handle("POST /api/client/network/usage", httpsFullAPIChain.ThenFunc(endpoints.SetClientNetworkUsage))

	// API DELETE endpoints
	httpsRouter.Handle("DELETE /api/images", httpsFullAPIChain.ThenFunc(endpoints.DeleteImage))

	// Static client files
	httpsRouter.Handle("GET /client/api/configs/uit-client", httpsFullAPIChain.ThenFunc(endpoints.GetClientConfig))
	httpsRouter.Handle("GET /client/pkg/uit-client", httpsFullAPIChain.ThenFunc(endpoints.WebServerHandler))

	// Login page and assets, no auth required
	httpsRouter.Handle("GET /login", httpsFullLoginChain.ThenFunc(endpoints.WebServerHandler))
	httpsRouter.Handle("POST /login", httpsFullLoginChain.ThenFunc(endpoints.WebAuthEndpoint))
	httpsRouter.Handle("GET /js/login.js", httpsFullLoginChain.ThenFunc(endpoints.WebServerHandler))
	httpsRouter.Handle("GET /js/init.js", httpsFullLoginChain.ThenFunc(endpoints.WebServerHandler))
	httpsRouter.Handle("GET /css/desktop.css", httpsFullLoginChain.ThenFunc(endpoints.WebServerHandler))
	httpsRouter.Handle("GET /favicon.png", httpsFullLoginChain.ThenFunc(endpoints.WebServerHandler))

	// Logout
	httpsRouter.Handle("GET /logout", httpsFullLogoutChain.ThenFunc(endpoints.RejectRequest))

	// Static HTML, CSS, and JS files
	httpsRouter.Handle("/js/", httpsFullCookieAuthChain.ThenFunc(endpoints.WebServerHandler))
	httpsRouter.Handle("/css/", httpsFullCookieAuthChain.ThenFunc(endpoints.WebServerHandler))
	httpsRouter.Handle("/", httpsFullCookieAuthChain.ThenFunc(endpoints.WebServerHandler))

	// Images and icons
	httpsRouter.Handle("GET /icons/search/search.svg", httpsFullLoginChain.ThenFunc(endpoints.WebServerHandler))
	httpsRouter.Handle("GET /icons/navigation/home.svg", httpsFullLoginChain.ThenFunc(endpoints.WebServerHandler))

	log.Info("Starting HTTPS web server...")

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
		Handler:        httpsRouter,
		TLSConfig:      tlsConfig,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB header size max
		BaseContext: func(_ net.Listener) context.Context {
			return ctx // Propagate cancellation to requests
		},
	}

	httpsServer.Protocols = new(http.Protocols)
	httpsServer.Protocols.SetHTTP1(false)
	httpsServer.Protocols.SetUnencryptedHTTP2(false)
	httpsServer.Protocols.SetHTTP2(true)

	webCertFile, webKeyFile, err := config.GetTLSCertFiles()
	if err != nil || (strings.TrimSpace(webCertFile) == "" || strings.TrimSpace(webKeyFile) == "") {
		return fmt.Errorf("error getting TLS cert files for HTTPS web server: %w", err)
	}

	// Start HTTPS server
	serverErr := make(chan error, 1)
	go func() {
		if err := httpsServer.ListenAndServeTLS(webCertFile, webKeyFile); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("Shutting down HTTPS web server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := httpsServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("error shutting down HTTPS server: %w", err)
		}
		log.Info("HTTPS web server stopped")
		return nil
	case err := <-serverErr:
		return err
	}
}
