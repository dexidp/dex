package router

import "net/http"

// Mux registers HTTP routes. The server provides the implementation (path
// prefixing, per-route headers, CORS); handlers only name their routes.
//
// The registration methods take an optional list of HTTP methods. When methods
// are given the route only matches those methods and the server answers any
// other method with a uniform 405, so handlers no longer guard the method
// themselves. Passing no methods leaves the route open to every method.
type Mux interface {
	// Handle mounts h at pattern, optionally restricting it to methods.
	Handle(pattern string, h http.Handler, methods ...string)
	// HandleFunc mounts h at pattern, optionally restricting it to methods.
	HandleFunc(pattern string, h http.HandlerFunc, methods ...string)
	// HandleCORS mounts h at pattern with cross-origin support (discovery,
	// token, keys and similar public endpoints), optionally restricting it to
	// methods.
	HandleCORS(pattern string, h http.HandlerFunc, methods ...string)
	// HandlePrefix mounts h for every path under pattern, stripping the prefix.
	HandlePrefix(pattern string, h http.Handler)
}

// Handler is a self-contained domain that mounts its own routes on the given
// Mux. The server collects handlers and calls Mount on each.
type Handler interface {
	Mount(Mux)
}
