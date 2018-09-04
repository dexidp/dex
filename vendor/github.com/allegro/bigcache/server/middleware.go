package main

import (
	"log"
	"net/http"
	"time"
)

// our base middleware implementation.
type service func(http.Handler) http.Handler

// chain load middleware services.
func serviceLoader(h http.Handler, svcs ...service) http.Handler {
	for _, svc := range svcs {
		h = svc(h)
	}
	return h
}

// middleware for request length metrics.
func requestMetrics(l *log.Logger) service {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			h.ServeHTTP(w, r)
			l.Printf("%s request to %s took %vns.", r.Method, r.URL.Path, time.Now().Sub(start).Nanoseconds())
		})
	}
}
