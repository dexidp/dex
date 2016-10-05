package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleHealth(t *testing.T) {
	httpServer, server := newTestServer(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	server.handleHealth(rr, httptest.NewRequest("GET", "/healthz", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 got %d", rr.Code)
	}

}
