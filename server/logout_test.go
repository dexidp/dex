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

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
)

func TestHandleLogoutNoSessions(t *testing.T) {
	httpServer, server := newTestServer(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/logout", nil)
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandleLogoutMethodNotAllowed(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/logout", nil)
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func TestHandleLogoutPOST(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/logout", nil)
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, "No active session")
}

func TestHandleLogoutNoHintGETShowsConfirmation(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/logout", nil)
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, "Do you want to log out?")
	require.Contains(t, body, `method="post"`)
	require.NotContains(t, body, "successfully logged out")
}

func TestHandleLogoutNoHintPOSTPerformsLogout(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	ctx := t.Context()
	userID := "test-user"
	connectorID := "mock"
	nonce := "testnonce"

	require.NoError(t, server.storage.CreateAuthSession(ctx, storage.AuthSession{
		UserID: userID, ConnectorID: connectorID, Nonce: nonce,
		CreatedAt: time.Now(), LastActivity: time.Now(),
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "dex_session",
		Value: sessionCookieValue(userID, connectorID, nonce, server.sessionConfig.CookieEncryptionKey),
	})
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "successfully logged out")

	// Session should be deleted.
	_, err := server.storage.GetAuthSession(ctx, userID, connectorID)
	require.ErrorIs(t, err, storage.ErrNotFound)
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

	idToken, _, err := server.newIDToken(ctx, clientID, storage.Claims{
		UserID: userID, Username: "testuser", Email: "test@example.com",
	}, []string{"openid"}, "", "", "", connectorID, time.Now())
	require.NoError(t, err)

	logoutURL := fmt.Sprintf("/logout?id_token_hint=%s&post_logout_redirect_uri=%s&state=mystate",
		url.QueryEscape(idToken), url.QueryEscape(postLogoutURI))

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", logoutURL, nil))

	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, "successfully logged out")
	require.Contains(t, body, "Back to Application")
	require.Contains(t, body, postLogoutURI)
	require.Contains(t, body, "state=mystate")

	// Session deleted.
	_, err = server.storage.GetAuthSession(ctx, userID, connectorID)
	require.ErrorIs(t, err, storage.ErrNotFound)

	// Cookie cleared.
	for _, c := range rr.Result().Cookies() {
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

	idToken, _, err := server.newIDToken(ctx, clientID, storage.Claims{
		UserID: "user1", Username: "testuser", Email: "test@example.com",
	}, []string{"openid"}, "", "", "", "mock", time.Now())
	require.NoError(t, err)

	logoutURL := fmt.Sprintf("/logout?id_token_hint=%s&post_logout_redirect_uri=%s",
		url.QueryEscape(idToken), url.QueryEscape("https://evil.com/steal"))

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", logoutURL, nil))
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleLogoutRedirectURIWithoutHint(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	// GET without hint shows confirmation page regardless of other params.
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", "/logout?post_logout_redirect_uri=https://example.com/done", nil))
	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "Do you want to log out?")

	// POST without hint but with post_logout_redirect_uri returns 400
	// because post_logout_redirect_uri requires client_id from id_token_hint.
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("POST", "/logout?post_logout_redirect_uri=https://example.com/done", nil))
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

	refreshID := storage.NewID()
	require.NoError(t, server.storage.CreateRefresh(ctx, storage.RefreshToken{
		ID: refreshID, Token: "token-value", ClientID: clientID, ConnectorID: connectorID,
		Claims: storage.Claims{UserID: userID, Username: "testuser", Email: "test@example.com"},
		Scopes: []string{"openid", "offline_access"}, CreatedAt: time.Now(), LastUsed: time.Now(),
	}))

	require.NoError(t, server.storage.CreateOfflineSessions(ctx, storage.OfflineSessions{
		UserID: userID, ConnID: connectorID,
		Refresh: map[string]*storage.RefreshTokenRef{
			clientID: {ID: refreshID, ClientID: clientID, CreatedAt: time.Now(), LastUsed: time.Now()},
		},
	}))

	require.NoError(t, server.storage.CreateAuthSession(ctx, storage.AuthSession{
		UserID: userID, ConnectorID: connectorID, Nonce: "testnonce",
		CreatedAt: time.Now(), LastActivity: time.Now(),
	}))

	idToken, _, err := server.newIDToken(ctx, clientID, storage.Claims{
		UserID: userID, Username: "testuser", Email: "test@example.com",
	}, []string{"openid"}, "", "", "", connectorID, time.Now())
	require.NoError(t, err)

	logoutURL := fmt.Sprintf("/logout?id_token_hint=%s&post_logout_redirect_uri=%s",
		url.QueryEscape(idToken), url.QueryEscape(postLogoutURI))

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", logoutURL, nil))
	require.Equal(t, http.StatusOK, rr.Code)

	_, err = server.storage.GetRefresh(ctx, refreshID)
	require.ErrorIs(t, err, storage.ErrNotFound)

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
		ID: clientID, Secret: "secret",
		RedirectURIs:           []string{"https://example.com/callback"},
		PostLogoutRedirectURIs: []string{postLogoutURI},
	}))

	require.NoError(t, server.storage.CreateAuthSession(ctx, storage.AuthSession{
		UserID: userID, ConnectorID: connectorID, Nonce: "testnonce",
		CreatedAt: time.Now(), LastActivity: time.Now(),
	}))

	idToken, _, err := server.newIDToken(ctx, clientID, storage.Claims{
		UserID: userID, Username: "testuser", Email: "test@example.com",
	}, []string{"openid"}, "", "", "", connectorID, time.Now())
	require.NoError(t, err)

	logoutURL := fmt.Sprintf("/logout?id_token_hint=%s&post_logout_redirect_uri=%s",
		url.QueryEscape(idToken), url.QueryEscape(postLogoutURI))

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", logoutURL, nil))
	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "successfully logged out")

	// Second logout — session already deleted, shows "no active session".
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", logoutURL, nil))
	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "No active session")
}

func TestLogoutCallbackNoState(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", "/logout/callback", nil))
	require.Equal(t, http.StatusBadRequest, rr.Code)
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

// TestHandleLogoutFromCookie tests logout without id_token_hint,
// where the user is identified by their session cookie alone.
func TestHandleLogoutFromCookie(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	ctx := t.Context()
	userID := "test-user"
	connectorID := "mock"
	nonce := "testnonce"

	require.NoError(t, server.storage.CreateAuthSession(ctx, storage.AuthSession{
		UserID: userID, ConnectorID: connectorID, Nonce: nonce,
		CreatedAt: time.Now(), LastActivity: time.Now(),
	}))

	// POST performs logout (GET without hint shows confirmation).
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "dex_session",
		Value: sessionCookieValue(userID, connectorID, nonce, server.sessionConfig.CookieEncryptionKey),
	})
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "successfully logged out")

	// Session should be deleted.
	_, err := server.storage.GetAuthSession(ctx, userID, connectorID)
	require.ErrorIs(t, err, storage.ErrNotFound)

	// Cookie should be cleared.
	for _, c := range rr.Result().Cookies() {
		if c.Name == "dex_session" {
			require.Equal(t, -1, c.MaxAge)
		}
	}
}

// TestLogoutCallbackWithExpiredSession tests that /logout/callback
// returns an error when the session has expired or been deleted.
func TestLogoutCallbackWithExpiredSession(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	// No session created — cookie points to nonexistent session.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/logout/callback", nil)
	req.AddCookie(&http.Cookie{
		Name:  "dex_session",
		Value: sessionCookieValue("user-1", "mock", "nonce", server.sessionConfig.CookieEncryptionKey),
	})
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRevokeRefreshTokensReturnsConnectorData(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	ctx := context.Background()
	userID := "user1"
	connectorID := "mock"
	expectedConnData := []byte(`{"RefreshToken":"abc"}`)

	refreshID := storage.NewID()
	require.NoError(t, server.storage.CreateRefresh(ctx, storage.RefreshToken{
		ID: refreshID, Token: "tok", ClientID: "client1", ConnectorID: connectorID,
		Claims: storage.Claims{UserID: userID}, CreatedAt: time.Now(), LastUsed: time.Now(),
	}))

	require.NoError(t, server.storage.CreateOfflineSessions(ctx, storage.OfflineSessions{
		UserID: userID, ConnID: connectorID, ConnectorData: expectedConnData,
		Refresh: map[string]*storage.RefreshTokenRef{"client1": {ID: refreshID, ClientID: "client1"}},
	}))

	connData := server.revokeRefreshTokens(ctx, userID, connectorID)
	require.Equal(t, expectedConnData, connData)

	_, err := server.storage.GetRefresh(ctx, refreshID)
	require.ErrorIs(t, err, storage.ErrNotFound)

	os, err := server.storage.GetOfflineSessions(ctx, userID, connectorID)
	require.NoError(t, err)
	require.Empty(t, os.Refresh)
	require.Equal(t, expectedConnData, os.ConnectorData)
}
