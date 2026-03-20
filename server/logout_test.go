package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestHandleLogoutNoSessions(t *testing.T) {
	httpServer, server := newTestServer(t, nil)
	defer httpServer.Close()

	// Without sessions, /logout route is not registered — expect 404.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/logout", nil)
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandleLogoutMethodNotAllowed(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/logout", nil)
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func TestHandleLogoutNoHint(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	// Logout without id_token_hint — no session to terminate.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/logout", nil)
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, "No active session")
	require.NotContains(t, body, "successfully logged out")
}

func TestHandleLogoutInvalidHint(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/logout?id_token_hint=invalid-token", nil)
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleLogoutWithValidHint(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	ctx := t.Context()

	clientID := "test-client"
	postLogoutURI := "https://example.com/done"

	// Create a client with PostLogoutRedirectURIs.
	require.NoError(t, server.storage.CreateClient(ctx, storage.Client{
		ID:                     clientID,
		Secret:                 "secret",
		RedirectURIs:           []string{"https://example.com/callback"},
		PostLogoutRedirectURIs: []string{postLogoutURI},
	}))

	// Create a user identity and auth session.
	userID := "test-user"
	connectorID := "mock"

	require.NoError(t, server.storage.CreateUserIdentity(ctx, storage.UserIdentity{
		UserID:      userID,
		ConnectorID: connectorID,
		Claims: storage.Claims{
			UserID:   userID,
			Username: "testuser",
			Email:    "test@example.com",
		},
		CreatedAt: time.Now(),
		LastLogin: time.Now(),
	}))

	require.NoError(t, server.storage.CreateAuthSession(ctx, storage.AuthSession{
		UserID:       userID,
		ConnectorID:  connectorID,
		Nonce:        "testnonce",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}))

	// Generate an ID token to use as hint.
	idToken, _, err := server.newIDToken(ctx, clientID, storage.Claims{
		UserID:   userID,
		Username: "testuser",
		Email:    "test@example.com",
	}, []string{"openid"}, "", "", "", connectorID, time.Now())
	require.NoError(t, err)

	// Make logout request with valid hint and post_logout_redirect_uri.
	logoutURL := fmt.Sprintf("/logout?id_token_hint=%s&post_logout_redirect_uri=%s&state=mystate",
		url.QueryEscape(idToken), url.QueryEscape(postLogoutURI))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", logoutURL, nil)
	server.ServeHTTP(rr, req)

	// Should render logout page with success message and back URL containing state.
	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, "successfully logged out")
	require.Contains(t, body, "Back to Application")
	require.Contains(t, body, postLogoutURI)
	require.Contains(t, body, "state=mystate")

	// Verify session was deleted.
	_, err = server.storage.GetAuthSession(ctx, userID, connectorID)
	require.ErrorIs(t, err, storage.ErrNotFound)

	// Verify cookie was cleared.
	cookies := rr.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "dex_session" {
			require.Equal(t, -1, c.MaxAge)
		}
	}
}

func TestHandleLogoutUnregisteredRedirectURI(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	ctx := t.Context()

	clientID := "test-client"
	require.NoError(t, server.storage.CreateClient(ctx, storage.Client{
		ID:                     clientID,
		Secret:                 "secret",
		RedirectURIs:           []string{"https://example.com/callback"},
		PostLogoutRedirectURIs: []string{"https://example.com/done"},
	}))

	// Generate an ID token.
	idToken, _, err := server.newIDToken(ctx, clientID, storage.Claims{
		UserID:   "user1",
		Username: "testuser",
		Email:    "test@example.com",
	}, []string{"openid"}, "", "", "", "mock", time.Now())
	require.NoError(t, err)

	// Try with unregistered URI.
	logoutURL := fmt.Sprintf("/logout?id_token_hint=%s&post_logout_redirect_uri=%s",
		url.QueryEscape(idToken), url.QueryEscape("https://evil.com/steal"))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", logoutURL, nil)
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleLogoutRedirectURIWithoutHint(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	// post_logout_redirect_uri without id_token_hint should fail.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/logout?post_logout_redirect_uri=https://example.com/done", nil)
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleLogoutRevokesRefreshTokens(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	ctx := t.Context()

	clientID := "test-client"
	postLogoutURI := "https://example.com/done"
	userID := "test-user"
	connectorID := "mock"

	require.NoError(t, server.storage.CreateClient(ctx, storage.Client{
		ID:                     clientID,
		Secret:                 "secret",
		RedirectURIs:           []string{"https://example.com/callback"},
		PostLogoutRedirectURIs: []string{postLogoutURI},
	}))

	// Create offline session with refresh token.
	refreshID := storage.NewID()
	require.NoError(t, server.storage.CreateRefresh(ctx, storage.RefreshToken{
		ID:          refreshID,
		Token:       "token-value",
		ClientID:    clientID,
		ConnectorID: connectorID,
		Claims: storage.Claims{
			UserID:   userID,
			Username: "testuser",
			Email:    "test@example.com",
		},
		Scopes:    []string{"openid", "offline_access"},
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}))

	require.NoError(t, server.storage.CreateOfflineSessions(ctx, storage.OfflineSessions{
		UserID: userID,
		ConnID: connectorID,
		Refresh: map[string]*storage.RefreshTokenRef{
			clientID: {
				ID:        refreshID,
				ClientID:  clientID,
				CreatedAt: time.Now(),
				LastUsed:  time.Now(),
			},
		},
	}))

	require.NoError(t, server.storage.CreateUserIdentity(ctx, storage.UserIdentity{
		UserID:      userID,
		ConnectorID: connectorID,
		Claims: storage.Claims{
			UserID:   userID,
			Username: "testuser",
			Email:    "test@example.com",
		},
		CreatedAt: time.Now(),
		LastLogin: time.Now(),
	}))

	require.NoError(t, server.storage.CreateAuthSession(ctx, storage.AuthSession{
		UserID:       userID,
		ConnectorID:  connectorID,
		Nonce:        "testnonce",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}))

	// Generate ID token.
	idToken, _, err := server.newIDToken(ctx, clientID, storage.Claims{
		UserID:   userID,
		Username: "testuser",
		Email:    "test@example.com",
	}, []string{"openid"}, "", "", "", connectorID, time.Now())
	require.NoError(t, err)

	logoutURL := fmt.Sprintf("/logout?id_token_hint=%s&post_logout_redirect_uri=%s",
		url.QueryEscape(idToken), url.QueryEscape(postLogoutURI))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", logoutURL, nil)
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "Logged Out")

	// Verify refresh token was deleted.
	_, err = server.storage.GetRefresh(ctx, refreshID)
	require.ErrorIs(t, err, storage.ErrNotFound)

	// Verify offline session still exists but Refresh map is cleared.
	os, err := server.storage.GetOfflineSessions(ctx, userID, connectorID)
	require.NoError(t, err)
	require.Empty(t, os.Refresh)
}

func TestHandleLogoutRepeat(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	ctx := t.Context()
	clientID := "test-client"
	postLogoutURI := "https://example.com/done"
	userID := "test-user"
	connectorID := "mock"

	require.NoError(t, server.storage.CreateClient(ctx, storage.Client{
		ID:                     clientID,
		Secret:                 "secret",
		RedirectURIs:           []string{"https://example.com/callback"},
		PostLogoutRedirectURIs: []string{postLogoutURI},
	}))

	require.NoError(t, server.storage.CreateAuthSession(ctx, storage.AuthSession{
		UserID:       userID,
		ConnectorID:  connectorID,
		Nonce:        "testnonce",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}))

	require.NoError(t, server.storage.CreateUserIdentity(ctx, storage.UserIdentity{
		UserID:      userID,
		ConnectorID: connectorID,
		Claims:      storage.Claims{UserID: userID, Username: "testuser", Email: "test@example.com"},
		CreatedAt:   time.Now(),
		LastLogin:   time.Now(),
	}))

	idToken, _, err := server.newIDToken(ctx, clientID, storage.Claims{
		UserID: userID, Username: "testuser", Email: "test@example.com",
	}, []string{"openid"}, "", "", "", connectorID, time.Now())
	require.NoError(t, err)

	logoutURL := fmt.Sprintf("/logout?id_token_hint=%s&post_logout_redirect_uri=%s",
		url.QueryEscape(idToken), url.QueryEscape(postLogoutURI))

	// First logout — should say "successfully logged out".
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", logoutURL, nil))
	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "successfully logged out")

	// Second logout — session is gone, should say "No active session".
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", logoutURL, nil))
	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, "No active session")
	require.NotContains(t, body, "successfully logged out")
}

func TestLogoutCallback(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	// Encode state.
	ls := logoutState{
		PostLogoutRedirectURI: "https://example.com/done",
		State:                 "xyz",
		ClientID:              "test-client",
	}
	stateParam, err := encodeLogoutState(server.logoutHMACKey, ls)
	require.NoError(t, err)

	callbackURL := fmt.Sprintf("/logout/callback?state=%s", url.QueryEscape(stateParam))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", callbackURL, nil)
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, "Logged Out")
	require.Contains(t, body, "Back to Application")
	require.Contains(t, body, "https://example.com/done")
	require.Contains(t, body, "state=xyz")
}

func TestLogoutCallbackNoState(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/logout/callback", nil)
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestLogoutStateEncodeDecode(t *testing.T) {
	key := []byte("test-hmac-key-for-logout-state")
	tests := []logoutState{
		{PostLogoutRedirectURI: "https://example.com/done", State: "abc", ClientID: "client1"},
		{PostLogoutRedirectURI: "", State: "", ClientID: ""},
		{PostLogoutRedirectURI: "https://example.com/path?foo=bar", State: "s=t&a=b", ClientID: "my-client"},
	}

	for _, ls := range tests {
		encoded, err := encodeLogoutState(key, ls)
		require.NoError(t, err)

		decoded, err := decodeLogoutState(key, encoded)
		require.NoError(t, err)
		require.Equal(t, ls, decoded)
	}
}

func TestLogoutStateTampered(t *testing.T) {
	key := []byte("test-hmac-key-for-logout-state")
	ls := logoutState{
		PostLogoutRedirectURI: "https://example.com/done",
		State:                 "abc",
		ClientID:              "client1",
	}

	encoded, err := encodeLogoutState(key, ls)
	require.NoError(t, err)

	// Tamper with the payload (flip a character before the dot).
	tampered := "x" + encoded[1:]
	_, err = decodeLogoutState(key, tampered)
	require.Error(t, err)

	// Wrong key.
	_, err = decodeLogoutState([]byte("wrong-key"), encoded)
	require.Error(t, err)
}

func TestDiscoveryWithSessions(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", "/.well-known/openid-configuration", nil))
	require.Equal(t, http.StatusOK, rr.Code)

	var d discovery
	require.NoError(t, json.NewDecoder(rr.Result().Body).Decode(&d))
	require.Equal(t, fmt.Sprintf("%s/logout", httpServer.URL), d.EndSession)
}

func TestDiscoveryWithoutSessions(t *testing.T) {
	httpServer, server := newTestServer(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", "/.well-known/openid-configuration", nil))
	require.Equal(t, http.StatusOK, rr.Code)

	var d discovery
	require.NoError(t, json.NewDecoder(rr.Result().Body).Decode(&d))
	require.Empty(t, d.EndSession)
}

// Verify that revokeRefreshTokens returns connector data.
func TestRevokeRefreshTokensReturnsConnectorData(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	ctx := context.Background()
	userID := "user1"
	connectorID := "mock"
	expectedConnData := []byte(`{"RefreshToken":"abc"}`)

	refreshID := storage.NewID()
	require.NoError(t, server.storage.CreateRefresh(ctx, storage.RefreshToken{
		ID:          refreshID,
		Token:       "tok",
		ClientID:    "client1",
		ConnectorID: connectorID,
		Claims:      storage.Claims{UserID: userID},
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
	}))

	require.NoError(t, server.storage.CreateOfflineSessions(ctx, storage.OfflineSessions{
		UserID:        userID,
		ConnID:        connectorID,
		ConnectorData: expectedConnData,
		Refresh: map[string]*storage.RefreshTokenRef{
			"client1": {ID: refreshID, ClientID: "client1"},
		},
	}))

	connData := server.revokeRefreshTokens(ctx, userID, connectorID)
	require.Equal(t, expectedConnData, connData)

	// Token should be gone.
	_, err := server.storage.GetRefresh(ctx, refreshID)
	require.ErrorIs(t, err, storage.ErrNotFound)

	// Offline session should still exist with empty Refresh map.
	os, err := server.storage.GetOfflineSessions(ctx, userID, connectorID)
	require.NoError(t, err)
	require.Empty(t, os.Refresh)
	require.Equal(t, expectedConnData, os.ConnectorData)
}
