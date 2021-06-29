package server

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

var _ http.Handler = (*metricsHandler)(nil)

type metricsHandler struct {
	route   string
	handler http.Handler
}

func (m *metricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l, _ := otelhttp.LabelerFromContext(r.Context())
	l.Add(semconv.HTTPMethodKey.String(r.Method))
	l.Add(semconv.HTTPRouteKey.String(r.URL.EscapedPath()))

	m.handler.ServeHTTP(w, r)
}

func wrapWithMetrics(path string, handler http.Handler) http.Handler {
	h := metricsHandler{route: path, handler: handler}
	return otelhttp.NewHandler(&h, "dex")
}
