package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// index for stats handle
func statsIndexHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getCacheStatsHandler(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// returns the cache's statistics.
func getCacheStatsHandler(w http.ResponseWriter, r *http.Request) {
	target, err := json.Marshal(cache.Stats())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("cannot marshal cache stats. error: %s", err)
		return
	}
	// since we're sending a struct, make it easy for consumers to interface.
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(target)
	return
}
