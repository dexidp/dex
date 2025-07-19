package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/dexidp/dex/pkg/otel/traces"
)

func (s *Server) discoveryHandler() (http.HandlerFunc, error) {
	d := s.constructDiscovery()

	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal discovery data: %v", err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, span := traces.InstrumentHandler(r)
		defer span.End()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.Write(data)
	}), nil
}
