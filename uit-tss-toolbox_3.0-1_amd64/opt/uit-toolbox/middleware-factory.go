package main

import (
	"net/http"
	config "uit-toolbox/config"
	middleware "uit-toolbox/middleware"
)

type MiddlewareFactory struct {
	appState  *config.AppState
	appConfig *config.AppConfig
}

func NewMiddlewareFactory(appState *config.AppState, appConfig *config.AppConfig) *MiddlewareFactory {
	return &MiddlewareFactory{
		appState:  appState,
		appConfig: appConfig,
	}
}

// Each middleware becomes a method that doesn't need parameters
func (mf *MiddlewareFactory) LimitRequestSize() func(http.Handler) http.Handler {
	return middleware.LimitRequestSizeMiddleware(mf.appState)
}

func (mf *MiddlewareFactory) Timeout() func(http.Handler) http.Handler {
	return middleware.TimeoutMiddleware(mf.appState)
}

func (mf *MiddlewareFactory) StoreClientIP() func(http.Handler) http.Handler {
	return middleware.StoreClientIPMiddleware(mf.appState)
}

func (mf *MiddlewareFactory) RateLimit(limitType string) func(http.Handler) http.Handler {
	return middleware.RateLimitMiddleware(mf.appState, limitType)
}

func (mf *MiddlewareFactory) AllowIPRange(allowedIPs []string) func(http.Handler) http.Handler {
	return middleware.AllowIPRangeMiddleware(mf.appState, allowedIPs)
}

func (mf *MiddlewareFactory) CheckValidURL() func(http.Handler) http.Handler {
	return middleware.CheckValidURLMiddleware(mf.appState)
}

func (mf *MiddlewareFactory) HTTPMethod() func(http.Handler) http.Handler {
	return middleware.HTTPMethodMiddleware(mf.appState)
}

func (mf *MiddlewareFactory) CheckHeaders() func(http.Handler) http.Handler {
	return middleware.CheckHeadersMiddleware(mf.appState)
}

func (mf *MiddlewareFactory) SetHeaders() func(http.Handler) http.Handler {
	return middleware.SetHeadersMiddleware(mf.appState)
}

func (mf *MiddlewareFactory) TLS() func(http.Handler) http.Handler {
	return middleware.TLSMiddleware(mf.appState)
}

func (mf *MiddlewareFactory) AllowedFiles() func(http.Handler) http.Handler {
	return middleware.AllowedFilesMiddleware(mf.appState)
}

func (mf *MiddlewareFactory) APIAuth() func(http.Handler) http.Handler {
	return middleware.APIAuth(mf.appState)
}

func (mf *MiddlewareFactory) HTTPCookieAuth() func(http.Handler) http.Handler {
	return middleware.HTTPCookieAuth(mf.appState)
}
