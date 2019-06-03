package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dexidp/dex/storage"
)

func TestHandleHealth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpServer, server := newTestServer(ctx, t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", "/healthz", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 got %d", rr.Code)
	}

}

type badStorage struct {
	storage.Storage
}

func (b *badStorage) CreateAuthRequest(r storage.AuthRequest) error {
	return errors.New("storage unavailable")
}

func TestHandleHealthFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpServer, server := newTestServer(ctx, t, func(c *Config) {
		c.Storage = &badStorage{c.Storage}
	})
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", "/healthz", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 got %d", rr.Code)
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
