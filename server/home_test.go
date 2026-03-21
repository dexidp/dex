package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func newTestServerWithSessions(t *testing.T, updateConfig func(c *Config)) (*httptest.Server, *Server) {
	t.Helper()

	var server *Server
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.ServeHTTP(w, r)
	}))

	logger := newLogger(t)
	ctx := t.Context()

	sig, err := signer.NewMockSigner(testKey)
	require.NoError(t, err)

	config := Config{
		Issuer:  s.URL,
		Storage: memory.New(logger),
		Web: WebConfig{
			Dir: "../web",
		},
		Logger:             logger,
		PrometheusRegistry: prometheus.NewRegistry(),
		HealthChecker:      gosundheit.New(),
		SkipApprovalScreen: true,
		AllowedGrantTypes: []string{
			grantTypeAuthorizationCode,
			grantTypeClientCredentials,
			grantTypeRefreshToken,
			grantTypeTokenExchange,
			grantTypeDeviceCode,
		},
		Signer: sig,
		SessionConfig: &SessionConfig{
			CookieName:        "dex_session",
			AbsoluteLifetime:  24 * time.Hour,
			ValidIfNotUsedFor: 1 * time.Hour,
		},
	}
	if updateConfig != nil {
		updateConfig(&config)
	}
	s.URL = config.Issuer

	connector := storage.Connector{
		ID:              "mock",
		Type:            "mockCallback",
		Name:            "Mock",
		ResourceVersion: "1",
	}
	require.NoError(t, config.Storage.CreateConnector(ctx, connector))

	server, err = newServer(ctx, config)
	require.NoError(t, err)

	if server.refreshTokenPolicy == nil {
		server.refreshTokenPolicy, err = NewRefreshTokenPolicy(logger, false, "", "", "")
		require.NoError(t, err)
		server.refreshTokenPolicy.now = config.Now
	}

	return s, server
}

func TestHomeNoSessions(t *testing.T) {
	httpServer, server := newTestServer(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	require.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()
	require.Contains(t, body, "Dex IdP")
	require.Contains(t, body, "Discovery")
	require.NotContains(t, body, "Logout")
}

func TestHomeNotLoggedIn(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	require.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()
	require.Contains(t, body, "Discovery")
	require.Contains(t, body, "Not logged in")
	require.NotContains(t, body, "Logout")
}

func TestHomeLoggedIn(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	ctx := t.Context()
	userID := "test-user"
	connectorID := "mock"
	nonce := "testnonce"
	now := time.Now()

	require.NoError(t, server.storage.CreateAuthSession(ctx, storage.AuthSession{
		UserID:       userID,
		ConnectorID:  connectorID,
		Nonce:        nonce,
		CreatedAt:    now,
		LastActivity: now,
	}))

	require.NoError(t, server.storage.CreateUserIdentity(ctx, storage.UserIdentity{
		UserID:      userID,
		ConnectorID: connectorID,
		Claims: storage.Claims{
			UserID:            userID,
			Username:          "Test User",
			PreferredUsername: "testuser",
			Email:             "test@example.com",
			EmailVerified:     true,
			Groups:            []string{"admins", "devs"},
		},
		LastLogin: now,
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "dex_session",
		Value: sessionCookieValue(userID, connectorID, nonce),
	})

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()
	require.Contains(t, body, "Logout")
	require.Contains(t, body, "/logout")
	require.Contains(t, body, "testuser")
	require.Contains(t, body, "test@example.com")
	require.Contains(t, body, "Mock")
	require.Contains(t, body, "admins")
	require.Contains(t, body, "Discovery")
	require.NotContains(t, body, "Not logged in")
}

func TestHomeInvalidCookie(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "dex_session",
		Value: "invalid-cookie-value",
	})

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()
	require.NotContains(t, body, "Logout")
	require.Contains(t, body, "Not logged in")
	require.Contains(t, body, "Discovery")
}
