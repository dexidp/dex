package server

import (
	"encoding/json"
	"net/http"

	"github.com/coreos/dex/pkg/log"
)

// writeResponseWithBody attempts to marshal an arbitrary thing to JSON then write
// it to the http.ResponseWriter
func writeResponseWithBody(w http.ResponseWriter, code int, resp interface{}) {
	enc, err := json.Marshal(resp)
	if err != nil {
		log.Errorf("Failed JSON-encoding HTTP response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if _, err = w.Write(enc); err != nil {
		log.Errorf("Failed writing HTTP response: %v", err)
	}
}
