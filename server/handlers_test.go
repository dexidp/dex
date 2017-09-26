package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coreos/dex/storage"
)

func TestHandleHealth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpServer, server := newTestServer(ctx, t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	server.handleHealth(rr, httptest.NewRequest("GET", "/healthz", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 got %d", rr.Code)
	}

}

func TestExplicitConnector(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	explicitID := "explicit"

	addConnector := func(c *Config) {
		connector := storage.Connector{
			ID:              explicitID,
			Type:            "mockCallback",
			Name:            "Mock",
			ResourceVersion: "1",
		}
		if err := c.Storage.CreateConnector(connector); err != nil {
			t.Fatalf("create connector: %v", err)
		}
		redirectURL := "http://localhost/callback"
		client := storage.Client{
			ID:           "testid",
			Secret:       "testsecret",
			RedirectURIs: []string{redirectURL},
		}
		if err := c.Storage.CreateClient(client); err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

	}

	httpServer, server := newTestServer(ctx, t, addConnector)
	defer httpServer.Close()

	authQueryParams := "connector=explicit" +
		"&response_type=code&scope=openid&client_id=testid&redirect_uri=http://localhost/callback"
	rr := httptest.NewRecorder()
	server.handleAuthorization(rr, httptest.NewRequest("GET", "/auth?"+authQueryParams, nil))

	location := rr.Result().Header.Get("location")
	if !strings.HasPrefix(location, "/auth/"+explicitID) {
		t.Errorf("expected redirect to /auth/%s, but got %s", explicitID, location)
	}

	if rr.Code != http.StatusFound {
		t.Errorf("expected redirect 302 got %d", rr.Code)
	}
}
