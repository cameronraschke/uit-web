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
	httpsBaseChain := muxChain{
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
	httpsMux.Handle("GET /api/all_tags", httpsFullAPIChain.thenFunc(endpoints.GetAllTags))
	httpsMux.Handle("GET /api/client/hardware/ids", httpsFullAPIChain.thenFunc(endpoints.GetHardwareIdentifiers))
	httpsMux.Handle("GET /api/client/bios", httpsFullAPIChain.thenFunc(endpoints.GetBiosData))
	httpsMux.Handle("GET /api/client/os", httpsFullAPIChain.thenFunc(endpoints.GetOSData))
	httpsMux.Handle("GET /api/job_queue/overview", httpsFullAPIChain.thenFunc(endpoints.GetJobQueueOverview))
	httpsMux.Handle("GET /api/job_queue/client/queued_job", httpsFullAPIChain.thenFunc(endpoints.GetClientQueuedJobs))
	httpsMux.Handle("GET /api/job_queue/client/job_available", httpsFullAPIChain.thenFunc(endpoints.GetClientAvailableJobs))
	httpsMux.Handle("GET /api/client/location_form_data", httpsFullAPIChain.thenFunc(endpoints.GetLocationFormData))
	httpsMux.Handle("GET /api/notes", httpsFullAPIChain.thenFunc(endpoints.GetNotes))
	httpsMux.Handle("GET /api/dashboard/inventory_summary", httpsFullAPIChain.thenFunc(endpoints.GetDashboardInventorySummary))
	httpsMux.Handle("GET /api/images/manifest", httpsFullAPIChain.thenFunc(endpoints.GetClientImagesManifest))
	httpsMux.Handle("GET /api/images", httpsFullAPIChain.thenFunc(endpoints.GetImage))
	httpsMux.Handle("GET /api/inventory", httpsFullAPIChain.thenFunc(endpoints.GetInventoryTableData))

	httpsMux.Handle("POST /api/notes", httpsFullAPIChain.thenFunc(endpoints.InsertNewNote))
	httpsMux.Handle("POST /api/update_inventory", httpsFullAPIChain.thenFunc(endpoints.UpdateInventory))
	httpsMux.Handle("POST /api/images/toggle_pin", httpsFullAPIChain.thenFunc(endpoints.TogglePinImage))

	httpsMux.Handle("DELETE /api/images", httpsFullAPIChain.thenFunc(endpoints.DeleteImage))

	httpsMux.Handle("GET /login", httpsFullLoginChain.thenFunc(endpoints.WebServerHandler))
	httpsMux.Handle("POST /login", httpsFullLoginChain.thenFunc(endpoints.WebAuthEndpoint))
	httpsMux.Handle("GET /css/login.css", httpsFullLoginChain.thenFunc(endpoints.WebServerHandler))
	httpsMux.Handle("/js/login.js", httpsFullLoginChain.thenFunc(endpoints.WebServerHandler))
	httpsMux.Handle("/css/desktop.css", httpsFullLoginChain.thenFunc(endpoints.WebServerHandler))
	httpsMux.Handle("/favicon.png", httpsFullLoginChain.thenFunc(endpoints.WebServerHandler))

	httpsMux.Handle("GET /logout", httpsFullLogoutChain.thenFunc(endpoints.LogoutHandler))

	httpsMux.Handle("/js/", httpsFullCookieAuthChain.thenFunc(endpoints.WebServerHandler))
	httpsMux.Handle("/css/", httpsFullCookieAuthChain.thenFunc(endpoints.WebServerHandler))
	httpsMux.Handle("/", httpsFullCookieAuthChain.thenFunc(endpoints.WebServerHandler))

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
		Handler:        httpsMux,
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
	if err != nil || strings.TrimSpace(webCertFile) == "" || strings.TrimSpace(webKeyFile) == "" {
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
