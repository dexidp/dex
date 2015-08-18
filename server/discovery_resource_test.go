package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	schema "github.com/coreos/dex/schema/workerschema"
)

func TestDiscoveryInvalidMethods(t *testing.T) {
	for _, verb := range []string{"POST", "PUT", "DELETE"} {
		res := &discoveryResource{}
		w := httptest.NewRecorder()
		r, err := http.NewRequest(verb, "http://example.com/discovery", nil)
		if err != nil {
			t.Fatalf("Failed creating http.Request: %v", err)
		}
		res.ServeHTTP(w, r)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("invalid response code for method=%s, want=%d, got=%d", verb, http.StatusMethodNotAllowed, w.Code)
		}
	}
}

func TestDiscoveryBody(t *testing.T) {
	res := &discoveryResource{}
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "http://example.com/discovery", nil)
	if err != nil {
		t.Fatalf("Failed creating http.Request: %v", err)
	}
	res.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	ct := w.HeaderMap["Content-Type"]
	if len(ct) != 1 {
		t.Errorf("Response has wrong number of Content-Type values: %v", ct)
	} else if ct[0] != "application/json" {
		t.Errorf("Expected application/json, got %s", ct)
	}
	if w.Body == nil {
		t.Error("Received nil response body")
	} else {
		if w.Body.String() != schema.DiscoveryJSON {
			t.Error("Received unexpected body!")
		}
	}
}
