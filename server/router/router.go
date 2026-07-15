// Package router defines the abstraction domain handlers use to mount their own
// HTTP routes, so the top-level server does not hardcode each handler's paths.
package router

import "net/http"

// Mux registers HTTP routes. The server provides the implementation (path
// prefixing, per-route headers, CORS); handlers only name their routes.
type Mux interface {
	// Handle mounts h at pattern.
	Handle(pattern string, h http.Handler)
	// HandleFunc mounts h at pattern.
	HandleFunc(pattern string, h http.HandlerFunc)
	// HandleCORS mounts h at pattern with cross-origin support (discovery,
	// token, keys and similar public endpoints).
	HandleCORS(pattern string, h http.HandlerFunc)
	// HandlePrefix mounts h for every path under pattern, stripping the prefix.
	HandlePrefix(pattern string, h http.Handler)
}

// Handler is a self-contained domain that mounts its own routes on the given
// Mux. The server collects handlers and calls Mount on each.
type Handler interface {
	Mount(Mux)
}
