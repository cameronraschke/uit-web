package webserver

import (
	"context"
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
		middleware.CheckForRedirectsMiddleware,
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
		ReadTimeout:    1 * time.Minute,
		WriteTimeout:   1 * time.Minute,
		IdleTimeout:    2 * time.Minute,
		MaxHeaderBytes: 1 << 20,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx // Propagate cancellation to requests
		},
	}

	return runServerLifecycle(
		ctx,
		log,
		"HTTP file server",
		1*time.Minute,
		httpServer.ListenAndServe,
		httpServer.Shutdown,
	)
}
