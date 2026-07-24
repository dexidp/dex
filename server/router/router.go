package router

import (
	"net/http"
	"path"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/dexidp/dex/server/reqctx"
)

// Mux registers HTTP routes. New returns the implementation used by the server
// (path prefixing, per-route headers, CORS); handlers only name their routes.
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

// Config configures the Mux returned by New.
type Config struct {
	// Router is the underlying gorilla router routes are registered on.
	Router *mux.Router
	// IssuerPath is prefixed onto every registered route.
	IssuerPath string
	// Headers are added to every response.
	Headers http.Header
	// RealIPHeader, when set, names the header the real client IP is read from.
	RealIPHeader string
	// Instrument wraps a handler with request metrics.
	Instrument func(name string, h http.Handler) http.HandlerFunc
	// RealIP extracts the client IP from a request.
	RealIP func(*http.Request) (string, error)
	// CORSOrigins / CORSHeaders configure HandleCORS; CORS is skipped when
	// CORSOrigins is empty.
	CORSOrigins []string
	CORSHeaders []string
}

// New returns a Mux backed by the given config. Handle/HandleFunc/HandleCORS
// prefix the route with the issuer path and wrap it with the response headers,
// request-id/real-ip context, and instrumentation. HandlePrefix mounts static
// asset trees directly (stripped of the prefix) without that wrapping.
func New(c Config) Mux { return mountMux{c} }

type mountMux struct{ c Config }

// wrap applies the common response headers, request-id/real-ip context, and
// instrumentation to a handler.
func (m mountMux) wrap(name string, h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for k, v := range m.c.Headers {
			w.Header()[k] = v
		}
		// Context values are used for logging purposes with the log/slog logger.
		rCtx := reqctx.WithRequestID(r.Context())
		if m.c.RealIPHeader != "" {
			if realIP, err := m.c.RealIP(r); err == nil {
				rCtx = reqctx.WithRemoteIP(rCtx, realIP)
			}
		}
		m.c.Instrument(name, h)(w, r.WithContext(rCtx))
	}
}

func (m mountMux) Handle(p string, h http.Handler) {
	m.c.Router.Handle(path.Join(m.c.IssuerPath, p), m.wrap(p, h))
}

func (m mountMux) HandleFunc(p string, h http.HandlerFunc) { m.Handle(p, h) }

func (m mountMux) HandlePrefix(p string, h http.Handler) {
	prefix := path.Join(m.c.IssuerPath, p)
	m.c.Router.PathPrefix(prefix).Handler(http.StripPrefix(prefix, h))
}

func (m mountMux) HandleCORS(p string, h http.HandlerFunc) {
	var handler http.Handler = h
	if len(m.c.CORSOrigins) > 0 {
		cors := handlers.CORS(
			handlers.AllowedOrigins(m.c.CORSOrigins),
			handlers.AllowedHeaders(m.c.CORSHeaders),
		)
		handler = cors(handler)
	}
	m.Handle(p, handler)
}
