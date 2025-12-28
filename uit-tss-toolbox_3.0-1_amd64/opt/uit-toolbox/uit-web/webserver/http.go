package webserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
	"uit-toolbox/config"
	"uit-toolbox/endpoints"
	"uit-toolbox/middleware"
)

func StartFileServer(ctx context.Context, serverHost string) error {
	log := config.GetLogger()
	httpBaseChain := middleware.NewChain(
		middleware.StoreLoggerMiddleware,
		middleware.PanicRecoveryMiddleware,
		middleware.LimitRequestSizeMiddleware,
		middleware.StoreClientIPMiddleware,
		middleware.CheckIPBlockedMiddleware,
		middleware.WebEndpointConfigMiddleware,
		middleware.TLSMiddleware,
		middleware.CheckHttpVersionMiddleware,
		middleware.AllowIPRangeMiddleware("lan"),
		middleware.RateLimitMiddleware("file"),
		middleware.FileServerTimeoutMiddleware,
		middleware.HTTPMethodMiddleware,
		middleware.CheckValidURLMiddleware,
		middleware.CheckHeadersMiddleware,
		middleware.SetHeadersMiddleware,
	)

	fileServerChain := httpBaseChain.Append(
		middleware.AllowedFilesMiddleware,
	)

	httpMux := http.NewServeMux()
	httpMux.Handle("/client/", fileServerChain.ThenFunc(endpoints.FileServerHandler))
	httpMux.Handle("/client", fileServerChain.ThenFunc(endpoints.RejectRequest))
	httpMux.Handle("/", httpBaseChain.ThenFunc(endpoints.RejectRequest))
	log.Info("Starting HTTP file server...")

	httpServer := &http.Server{
		Addr:           serverHost + ":8080",
		Handler:        httpMux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   1 * time.Minute,
		IdleTimeout:    2 * time.Minute,
		MaxHeaderBytes: 1 << 20,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx // Propagate cancellation to requests
		},
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("HTTP file server shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("error shutting down HTTP file server: %w", err)
		}
		log.Info("HTTP file server stopped")
		return nil
	case err := <-serverErr:
		return err
	}
}
