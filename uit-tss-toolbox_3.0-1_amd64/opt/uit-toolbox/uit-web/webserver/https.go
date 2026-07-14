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

	// API endpoints //
	// Overviews
	httpsRouter.Handle("GET /api/overview/inventory_table", httpsFullAPIChain.ThenFunc(endpoints.GetInventoryTableData))
	httpsRouter.Handle("GET /api/overview/job_queue_table", httpsFullAPIChain.ThenFunc(endpoints.GetJobQueueTable))
	httpsRouter.Handle("GET /api/overview/all_client_ids", httpsFullAPIChain.ThenFunc(endpoints.GetAllClientIDs))
	httpsRouter.Handle("GET /api/overview/all_models", httpsFullAPIChain.ThenFunc(endpoints.GetManufacturersAndModels))
	httpsRouter.Handle("GET /api/overview/all_domains", httpsFullAPIChain.ThenFunc(endpoints.GetAllDomains))
	httpsRouter.Handle("GET /api/overview/all_departments", httpsFullAPIChain.ThenFunc(endpoints.GetDepartments))
	httpsRouter.Handle("GET /api/overview/all_statuses", httpsFullAPIChain.ThenFunc(endpoints.GetAllStatuses))
	httpsRouter.Handle("GET /api/overview/all_locations", httpsFullAPIChain.ThenFunc(endpoints.GetAllLocations))
	httpsRouter.Handle("GET /api/overview/all_buildings_and_rooms", httpsFullAPIChain.ThenFunc(endpoints.FetchAllBuildingsAndRooms))
	httpsRouter.Handle("GET /api/overview/all_device_types", httpsFullAPIChain.ThenFunc(endpoints.GetAllDeviceTypes))
	httpsRouter.Handle("GET /api/overview/job_queue/all_jobs", httpsFullAPIChain.ThenFunc(endpoints.GetAllJobs))
	httpsRouter.Handle("GET /api/overview/note", httpsFullAPIChain.ThenFunc(endpoints.GetNotes))
	httpsRouter.Handle("POST /api/overview/note", httpsFullAPIChain.ThenFunc(endpoints.InsertNewNote))

	// Reports

	// Client-specific
	httpsRouter.Handle("GET /api/client/job_queue/job_available", httpsFullAPIChain.ThenFunc(endpoints.IsClientJobAvailable))
	httpsRouter.Handle("GET /api/client/hardware", httpsFullAPIChain.ThenFunc(endpoints.FetchClientHardwareData))
	httpsRouter.Handle("GET /api/checkout", httpsFullAPIChain.ThenFunc(endpoints.FetchCheckoutData))

	// Job queue
	httpsRouter.Handle("POST /api/client/job_stats", httpsFullAPIChain.ThenFunc(endpoints.UpdateJobStats))
	httpsRouter.Handle("GET /api/client/job_name", httpsFullAPIChain.ThenFunc(endpoints.FetchClientJobName))
	httpsRouter.Handle("GET /api/client/job_name_formatted", httpsFullAPIChain.ThenFunc(endpoints.FetchFormattedJobName))
	httpsRouter.Handle("GET /api/client/job_queue_position", httpsFullAPIChain.ThenFunc(endpoints.FetchClientJobQueuePosition))
	httpsRouter.Handle("POST /api/client/job_queue/update_job", httpsFullAPIChain.ThenFunc(endpoints.SetClientJob))
	httpsRouter.Handle("POST /api/client/job_queued_at", httpsFullAPIChain.ThenFunc(endpoints.SetJobQueuedAt))
	httpsRouter.Handle("POST /api/client/last_heard", httpsFullAPIChain.ThenFunc(endpoints.SetClientLastHeard))
	httpsRouter.Handle("GET /api/client/job/disk_image/name", httpsFullAPIChain.ThenFunc(endpoints.GetDiskImageNameByModel))

	// Web and auth
	httpsRouter.Handle("GET /api/check_auth", httpsFullAPIChain.ThenFunc(endpoints.RejectRequest))

	// Misc
	httpsRouter.Handle("GET /api/server_time", httpsFullAPIChain.ThenFunc(endpoints.GetServerTime))
	httpsRouter.Handle("GET /api/new_transaction_uuid", httpsFullAPIChain.ThenFunc(endpoints.GetNewTransactionUUID))

	// Overviews
	httpsRouter.Handle("GET /api/client", httpsFullAPIChain.ThenFunc(endpoints.GetClientInfo))
	httpsRouter.Handle("POST /api/job_queue/all_clients/update_job", httpsFullAPIChain.ThenFunc(endpoints.SetAllJobs))

	// Client hardware
	httpsRouter.Handle("POST /api/client/init", httpsFullAPIChain.ThenFunc(endpoints.InitClient))
	httpsRouter.Handle("POST /api/client/uptime", httpsFullAPIChain.ThenFunc(endpoints.SetClientUptime))
	httpsRouter.Handle("POST /api/client/hardware", httpsFullAPIChain.ThenFunc(endpoints.SetClientHardwareData))
	httpsRouter.Handle("POST /api/client/hardware/battery", httpsFullAPIChain.ThenFunc(endpoints.UpdateClientBatteryChargePcnt))
	httpsRouter.Handle("POST /api/client/health", httpsFullAPIChain.ThenFunc(endpoints.UpdateClientHealthCheck))
	httpsRouter.Handle("POST /api/windows-client-info", httpsFullAPIChain.ThenFunc(endpoints.ReceiveWindowsClientInfo))
	httpsRouter.Handle("POST /api/client/memory/usage", httpsFullAPIChain.ThenFunc(endpoints.SetClientMemoryUsageKB))
	httpsRouter.Handle("POST /api/client/memory/capacity", httpsFullAPIChain.ThenFunc(endpoints.SetClientMemoryCapacityKB))
	httpsRouter.Handle("POST /api/client/cpu/usage", httpsFullAPIChain.ThenFunc(endpoints.SetClientCPUUsage))
	httpsRouter.Handle("POST /api/client/cpu/mhz", httpsFullAPIChain.ThenFunc(endpoints.SetClientCPUMHz))
	httpsRouter.Handle("POST /api/client/cpu/temp", httpsFullAPIChain.ThenFunc(endpoints.SetClientCPUTemperature))
	httpsRouter.Handle("POST /api/client/os/network/usage", httpsFullAPIChain.ThenFunc(endpoints.SetClientNetworkUsage))

	// Client OS and software
	httpsRouter.Handle("DELETE /api/client/os/delete_all", httpsFullAPIChain.ThenFunc(endpoints.DeleteOSInfoByTagnumber))

	// Inventory
	httpsRouter.Handle("GET /api/client/lookup_ids", httpsFullAPIChain.ThenFunc(endpoints.GetClientIDs))
	httpsRouter.Handle("GET /api/client/location_form_data", httpsFullAPIChain.ThenFunc(endpoints.GetLocationFormData))

	httpsRouter.Handle("POST /api/inventory/update_client_data", httpsFullAPIChain.ThenFunc(endpoints.InsertInventoryUpdate))
	httpsRouter.Handle("POST /api/inventory/bulk_update_location", httpsFullAPIChain.ThenFunc(endpoints.BulkUpdateInventoryLocation))

	// Files
	httpsRouter.Handle("GET /api/client/files/manifest", httpsFullAPIChain.ThenFunc(endpoints.GetClientImagesManifest))
	httpsRouter.Handle("GET /api/client/files", httpsFullAPIChain.ThenFunc(endpoints.GetImage))
	httpsRouter.Handle("GET /api/client/live_screenshot", httpsFullAPIChain.ThenFunc(endpoints.DownloadLiveImage))
	httpsRouter.Handle("POST /api/client/files/upload", httpsFullAPIChain.ThenFunc(endpoints.UploadClientImage))
	httpsRouter.Handle("POST /api/files/toggle_pin", httpsFullAPIChain.ThenFunc(endpoints.TogglePinImage))
	httpsRouter.Handle("POST /api/client/live_screenshot", httpsFullAPIChain.ThenFunc(endpoints.UploadLiveImage))
	httpsRouter.Handle("DELETE /api/client/files", httpsFullAPIChain.ThenFunc(endpoints.DeleteImage))

	// Static files //
	// Login/logout page and assets, no auth required
	httpsRouter.Handle("GET /login", httpsFullLoginChain.ThenFunc(endpoints.WebServerHandler))
	httpsRouter.Handle("POST /login", httpsFullLoginChain.ThenFunc(endpoints.WebAuthEndpoint))
	httpsRouter.Handle("GET /js/login.js", httpsFullLoginChain.ThenFunc(endpoints.WebServerHandler))
	httpsRouter.Handle("GET /js/init.js", httpsFullLoginChain.ThenFunc(endpoints.WebServerHandler))
	httpsRouter.Handle("GET /css/desktop.css", httpsFullLoginChain.ThenFunc(endpoints.WebServerHandler))
	httpsRouter.Handle("GET /favicon.png", httpsFullLoginChain.ThenFunc(endpoints.WebServerHandler))
	httpsRouter.Handle("GET /logout", httpsFullLogoutChain.ThenFunc(endpoints.RejectRequest))

	// Static HTML, CSS, and JS files
	httpsRouter.Handle("/js/", httpsFullCookieAuthChain.ThenFunc(endpoints.WebServerHandler))
	httpsRouter.Handle("/css/", httpsFullCookieAuthChain.ThenFunc(endpoints.WebServerHandler))
	httpsRouter.Handle("/", httpsFullCookieAuthChain.ThenFunc(endpoints.WebServerHandler))

	// Images and icons
	httpsRouter.Handle("GET /icons/search/search.svg", httpsFullLoginChain.ThenFunc(endpoints.WebServerHandler))
	httpsRouter.Handle("GET /icons/navigation/home.svg", httpsFullLoginChain.ThenFunc(endpoints.WebServerHandler))

	// For clients plugged into server
	httpsRouter.Handle("GET /static/client/configs/uit-client", httpsFullAPIChain.ThenFunc(endpoints.GetClientConfig))
	httpsRouter.Handle("GET /client/pkg/uit-client", httpsFullAPIChain.ThenFunc(endpoints.WebServerHandler))

	log.Info("Starting HTTPS web server...")

	apiTimeout, err := config.GetRequestTimeout("api")
	if err != nil {
		return fmt.Errorf("cannot get API request timeout for HTTPS server: %w", err)
	}
	if apiTimeout <= 0 {
		return fmt.Errorf("invalid API request timeout configured for HTTPS server: %s", apiTimeout)
	}

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
		SessionTicketsDisabled:   false,
	}

	httpsServer := &http.Server{
		Addr:              ":31411",
		Handler:           httpsRouter,
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       apiTimeout,
		WriteTimeout:      apiTimeout,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB header size max
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

	return runServerLifecycle(
		ctx,
		log,
		"HTTPS web server",
		30*time.Second,
		func() error {
			return httpsServer.ListenAndServeTLS(webCertFile, webKeyFile)
		},
		httpsServer.Shutdown,
	)
}
