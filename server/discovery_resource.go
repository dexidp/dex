package server

import (
	"net/http"
	"path"

	"github.com/coreos/dex/pkg/log"
	schema "github.com/coreos/dex/schema/workerschema"
)

type discoveryResource struct{}

func registerDiscoveryResource(prefix string, mux *http.ServeMux) {
	c := &discoveryResource{}
	p := path.Join(prefix, "discovery")
	mux.Handle(p, c)
}

func (d *discoveryResource) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeAPIError(w, http.StatusMethodNotAllowed, newAPIError(errorInvalidRequest, "only HTTP GET supported against this resource"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(schema.DiscoveryJSON)); err != nil {
		log.Errorf("Failed sending discovery JSON HTTP response body: %v", err)
		writeAPIError(w, http.StatusInternalServerError, newAPIError(errorServerError, "error serving discovery JSON"))
	}
}
