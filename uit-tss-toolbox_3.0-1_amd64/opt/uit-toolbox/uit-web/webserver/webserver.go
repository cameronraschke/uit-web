package webserver

import (
	"net/http"
	"slices"
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
