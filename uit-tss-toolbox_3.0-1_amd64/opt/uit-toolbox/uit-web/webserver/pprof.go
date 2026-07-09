package webserver

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"time"
	"uit-toolbox/config"
)

func StartPprofServer(ctx context.Context) error {
	log := config.GetLogger()
	log.Info("Starting pprof server on localhost:6060...")

	pprofServer := &http.Server{Addr: "localhost:6060", Handler: nil}

	return runServerLifecycle(
		ctx,
		log,
		"pprof server",
		5*time.Second,
		pprofServer.ListenAndServe,
		pprofServer.Shutdown,
	)
}
