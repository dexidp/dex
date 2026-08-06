package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/discovery"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/tokens"
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

	newTestAuthSession(t, server, "test-user", "mock", "testnonce")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/logout", nil)
	req.AddCookie(testSessionCookie("testnonce"))
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
		ID: nonce, Secret: testSessionSecret(nonce),
		UserID: userID, ConnectorID: connectorID,
		CreatedAt: time.Now(), LastActivity: time.Now(),
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "dex_session",
		Value: internal.SessionCookieValue(nonce, testSessionSecret(nonce), testSessionKey),
	})
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "successfully logged out")

	// Session should be deleted.
	_, err := server.storage.GetAuthSession(ctx, "testnonce")
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
		ID:           "testnonce",
		Secret:       testSessionSecret("testnonce"),
		UserID:       userID,
		ConnectorID:  connectorID,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}))

	idToken, _, err := server.issuer.SignIDToken(ctx, tokens.Authorization{
		Client:      storage.Client{ID: clientID},
		Claims:      storage.Claims{UserID: userID, Username: "testuser", Email: "test@example.com"},
		Scopes:      []string{"openid"},
		ConnectorID: connectorID,
		AuthTime:    time.Now(),
	}, "", "")
	require.NoError(t, err)

	logoutURL := fmt.Sprintf("/logout?id_token_hint=%s&post_logout_redirect_uri=%s&state=mystate",
		url.QueryEscape(idToken), url.QueryEscape(postLogoutURI))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", logoutURL, nil)
	req.AddCookie(testSessionCookie("testnonce"))
	server.ServeHTTP(rr, req)

	// RP-Initiated Logout §4: a validated post_logout_redirect_uri is a redirect,
	// not a page with a link on it.
	require.Equal(t, http.StatusFound, rr.Code)
	require.Equal(t, postLogoutURI+"?state=mystate", rr.Header().Get("Location"))

	// Session deleted.
	_, err = server.storage.GetAuthSession(ctx, "testnonce")
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

	idToken, _, err := server.issuer.SignIDToken(ctx, tokens.Authorization{
		Client:      storage.Client{ID: clientID},
		Claims:      storage.Claims{UserID: "user1", Username: "testuser", Email: "test@example.com"},
		Scopes:      []string{"openid"},
		ConnectorID: "mock",
		AuthTime:    time.Now(),
	}, "", "")
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

	// post_logout_redirect_uri can only be validated against the client that
	// id_token_hint names, so without a hint it is an error either way.
	for _, method := range []string{"GET", "POST"} {
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, httptest.NewRequest(method, "/logout?post_logout_redirect_uri=https://example.com/done", nil))
		require.Equal(t, http.StatusBadRequest, rr.Code, method)
	}
}

// TestHandleLogoutRevokesOnlySessionBoundClients is the regression test for the bug
// where signing out of a web app destroyed the user's kubectl credentials: logout
// used to revoke every refresh token the user owned, across all clients.
//
// The rule now is the client's own RefreshTokenLifetime, and it holds even for the
// client that asked for the logout.
func TestHandleLogoutRevokesOnlySessionBoundClients(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	ctx := t.Context()
	const (
		webClient   = "web-client"
		proxyClient = "proxy"
		cliClient   = "kubectl"
		userID      = "test-user"
		connectorID = "mock"
		postLogout  = "https://example.com/done"
	)

	// Asks for the logout, but never declared its tokens tied to the session.
	require.NoError(t, server.storage.CreateClient(ctx, storage.Client{
		ID: webClient, Secret: "secret",
		RedirectURIs:           []string{"https://example.com/callback"},
		PostLogoutRedirectURIs: []string{postLogout},
	}))
	// Another client in the same session, this one bound to it.
	require.NoError(t, server.storage.CreateClient(ctx, storage.Client{
		ID: proxyClient, Secret: "secret",
		RedirectURIs:         []string{"https://example.com/callback"},
		RefreshTokenLifetime: storage.RefreshTokenLifetimeSession,
	}))

	webRefresh := newTestRefreshToken(t, server, userID, connectorID, webClient)
	proxyRefresh := newTestRefreshToken(t, server, userID, connectorID, proxyClient)
	cliRefresh := newTestRefreshToken(t, server, userID, connectorID, cliClient)

	require.NoError(t, server.storage.CreateOfflineSessions(ctx, storage.OfflineSessions{
		UserID: userID, ConnID: connectorID,
		Refresh: map[string]*storage.RefreshTokenRef{
			webClient:   {ID: webRefresh, ClientID: webClient, CreatedAt: time.Now(), LastUsed: time.Now()},
			proxyClient: {ID: proxyRefresh, ClientID: proxyClient, CreatedAt: time.Now(), LastUsed: time.Now()},
			cliClient:   {ID: cliRefresh, ClientID: cliClient, CreatedAt: time.Now(), LastUsed: time.Now()},
		},
	}))

	require.NoError(t, server.storage.CreateAuthSession(ctx, storage.AuthSession{
		ID: "testnonce", Secret: testSessionSecret("testnonce"),
		UserID: userID, ConnectorID: connectorID,
		CreatedAt: time.Now(), LastActivity: time.Now(),
		ClientStates: map[string]*storage.ClientAuthState{
			webClient:   {AuthenticatedAt: time.Now()},
			proxyClient: {AuthenticatedAt: time.Now()},
		},
	}))
	idToken := newTestIDToken(t, server, webClient, userID, connectorID, "testnonce")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", fmt.Sprintf("/logout?id_token_hint=%s&post_logout_redirect_uri=%s",
		url.QueryEscape(idToken), url.QueryEscape(postLogout)), nil)
	req.AddCookie(testSessionCookie("testnonce"))
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusFound, rr.Code)

	// The client that declared itself part of the session loses its token...
	_, err := server.storage.GetRefresh(ctx, proxyRefresh)
	require.ErrorIs(t, err, storage.ErrNotFound)

	// ...while the one that merely asked for the logout keeps its own...
	_, err = server.storage.GetRefresh(ctx, webRefresh)
	require.NoError(t, err)

	// ...and so does a client that was never in this session at all.
	_, err = server.storage.GetRefresh(ctx, cliRefresh)
	require.NoError(t, err)

	os, err := server.storage.GetOfflineSessions(ctx, userID, connectorID)
	require.NoError(t, err)
	require.NotContains(t, os.Refresh, proxyClient)
	require.Contains(t, os.Refresh, webClient)
	require.Contains(t, os.Refresh, cliClient)
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
		ID: "testnonce", Secret: testSessionSecret("testnonce"),
		UserID: userID, ConnectorID: connectorID,
		CreatedAt: time.Now(), LastActivity: time.Now(),
	}))

	idToken, _, err := server.issuer.SignIDToken(ctx, tokens.Authorization{
		Client:      storage.Client{ID: clientID},
		Claims:      storage.Claims{UserID: userID, Username: "testuser", Email: "test@example.com"},
		Scopes:      []string{"openid"},
		ConnectorID: connectorID,
		AuthTime:    time.Now(),
	}, "", "")
	require.NoError(t, err)

	logoutURL := fmt.Sprintf("/logout?id_token_hint=%s&post_logout_redirect_uri=%s",
		url.QueryEscape(idToken), url.QueryEscape(postLogoutURI))

	req := httptest.NewRequest("GET", logoutURL, nil)
	req.AddCookie(testSessionCookie("testnonce"))
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusFound, rr.Code)

	// Second logout — session already gone, but the RP still gets its redirect.
	req = httptest.NewRequest("GET", logoutURL, nil)
	req.AddCookie(testSessionCookie("testnonce"))
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusFound, rr.Code)
	require.Equal(t, postLogoutURI, rr.Header().Get("Location"))
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

	var d discovery.Document
	require.NoError(t, json.NewDecoder(rr.Result().Body).Decode(&d))
	require.Equal(t, fmt.Sprintf("%s/logout", httpServer.URL), d.EndSession)
	require.True(t, d.BackchannelLogout)
	require.True(t, d.BackchannelLogoutSession)
	require.Contains(t, d.Claims, "sid")
	require.Contains(t, d.Claims, "groups")
	require.Contains(t, d.Claims, "federated_claims")
}

func TestDiscoveryWithoutSessions(t *testing.T) {
	httpServer, server := newTestServer(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", "/.well-known/openid-configuration", nil))
	require.Equal(t, http.StatusOK, rr.Code)

	var d discovery.Document
	require.NoError(t, json.NewDecoder(rr.Result().Body).Decode(&d))
	require.Empty(t, d.EndSession)
	require.False(t, d.BackchannelLogout)
	require.False(t, d.BackchannelLogoutSession)
	require.NotContains(t, d.Claims, "sid")
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
		ID: nonce, Secret: testSessionSecret(nonce),
		UserID: userID, ConnectorID: connectorID,
		CreatedAt: time.Now(), LastActivity: time.Now(),
	}))

	// POST performs logout (GET without hint shows confirmation).
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "dex_session",
		Value: internal.SessionCookieValue(nonce, testSessionSecret(nonce), testSessionKey),
	})
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "successfully logged out")

	// Session should be deleted.
	_, err := server.storage.GetAuthSession(ctx, "testnonce")
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
		Value: internal.SessionCookieValue("nonce", testSessionSecret("nonce"), testSessionKey),
	})
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRevokeRefreshTokens(t *testing.T) {
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

	server.issuer.Refresh.RevokeAll(ctx, userID, connectorID)

	_, err := server.storage.GetRefresh(ctx, refreshID)
	require.ErrorIs(t, err, storage.ErrNotFound)

	// Revocation clears the refresh references but keeps the session object.
	os, err := server.storage.GetOfflineSessions(ctx, userID, connectorID)
	require.NoError(t, err)
	require.Empty(t, os.Refresh)
	require.Equal(t, expectedConnData, os.ConnectorData)
}

// --- helpers ---

// testSessionSecret is the secret paired with a session id in these tests. Sessions
// are named by an id the tokens publish, and proved by a secret only the cookie ever
// carries, so tests need both.
func testSessionSecret(sessionID string) string { return sessionID + "-secret" }

func testSessionCookie(sessionID string) *http.Cookie {
	return &http.Cookie{
		Name:  "dex_session",
		Value: internal.SessionCookieValue(sessionID, testSessionSecret(sessionID), testSessionKey),
	}
}

func newTestAuthSession(t *testing.T, server *Server, userID, connectorID, sessionID string) {
	t.Helper()
	require.NoError(t, server.storage.CreateAuthSession(t.Context(), storage.AuthSession{
		ID: sessionID, Secret: testSessionSecret(sessionID),
		UserID: userID, ConnectorID: connectorID,
		CreatedAt: time.Now(), LastActivity: time.Now(),
	}))
}

func newTestRefreshToken(t *testing.T, server *Server, userID, connectorID, clientID string) string {
	t.Helper()
	id := storage.NewID()
	require.NoError(t, server.storage.CreateRefresh(t.Context(), storage.RefreshToken{
		ID: id, Token: storage.NewID(), ClientID: clientID, ConnectorID: connectorID,
		Claims:    storage.Claims{UserID: userID},
		Scopes:    []string{"openid", "offline_access"},
		CreatedAt: time.Now(), LastUsed: time.Now(),
	}))
	return id
}

// newTestIDToken mints a token the way a browser flow would: the sid names the
// session the token came from, which is what the authorization code carries.
func newTestIDToken(t *testing.T, server *Server, clientID, userID, connectorID, sessionID string) string {
	t.Helper()

	idToken, _, err := server.issuer.SignIDToken(t.Context(), tokens.Authorization{
		Client:      storage.Client{ID: clientID},
		Claims:      storage.Claims{UserID: userID, Username: "testuser", Email: "test@example.com"},
		Scopes:      []string{"openid"},
		ConnectorID: connectorID,
		AuthTime:    time.Now(),
		SessionID:   sessionID,
	}, "", "")
	require.NoError(t, err)
	return idToken
}

// idTokenClaims decodes a signed token's payload without verifying it.
func idTokenClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims map[string]any
	require.NoError(t, json.Unmarshal(payload, &claims))
	return claims
}

// --- sid ---

// TestIDTokenSIDNamesTheSession: the claim is the session's own identifier, and the
// secret that proves the cookie never travels with it.
func TestIDTokenSIDNamesTheSession(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	const userID, connectorID = "test-user", "mock"
	newTestAuthSession(t, server, userID, connectorID, "session-a")

	claims := idTokenClaims(t, newTestIDToken(t, server, "c", userID, connectorID, "session-a"))
	require.Equal(t, "session-a", claims["sid"])
	require.NotContains(t, newTestIDToken(t, server, "c", userID, connectorID, "session-a"),
		testSessionSecret("session-a"))

	// A second device is a second session, so its tokens name a different one.
	newTestAuthSession(t, server, userID, connectorID, "session-b")
	other := idTokenClaims(t, newTestIDToken(t, server, "c", userID, connectorID, "session-b"))
	require.Equal(t, "session-b", other["sid"])
}

func TestIDTokenHasNoSIDWithoutSessions(t *testing.T) {
	httpServer, server := newTestServer(t, nil)
	defer httpServer.Close()

	claims := idTokenClaims(t, newTestIDToken(t, server, "c", "test-user", "mock", ""))
	require.NotContains(t, claims, "sid")
}

// --- replay ---

// TestHandleLogoutHintReplay covers the attack the cookie-authoritative rule exists
// to stop: a leaked id_token_hint is accepted past its expiry, so on its own it must
// never be enough to end anybody's session.
func TestHandleLogoutHintReplay(t *testing.T) {
	const (
		clientID    = "test-client"
		userID      = "victim"
		connectorID = "mock"
	)

	setup := func(t *testing.T) (*Server, string, string) {
		t.Helper()
		httpServer, server := newTestServerWithSessions(t, nil)
		t.Cleanup(httpServer.Close)

		require.NoError(t, server.storage.CreateClient(t.Context(), storage.Client{
			ID: clientID, Secret: "secret",
			RedirectURIs: []string{"https://example.com/callback"},
		}))
		newTestAuthSession(t, server, userID, connectorID, "testnonce")
		refreshID := newTestRefreshToken(t, server, userID, connectorID, clientID)
		require.NoError(t, server.storage.CreateOfflineSessions(t.Context(), storage.OfflineSessions{
			UserID: userID, ConnID: connectorID,
			Refresh: map[string]*storage.RefreshTokenRef{
				clientID: {ID: refreshID, ClientID: clientID},
			},
		}))
		return server, newTestIDToken(t, server, clientID, userID, connectorID, "testnonce"), refreshID
	}

	assertUntouched := func(t *testing.T, server *Server, sessionID, refreshID string) {
		t.Helper()
		_, err := server.storage.GetAuthSession(t.Context(), sessionID)
		require.NoError(t, err, "session must survive a replayed hint")
		_, err = server.storage.GetRefresh(t.Context(), refreshID)
		require.NoError(t, err, "refresh token must survive a replayed hint")
	}

	t.Run("no cookie", func(t *testing.T) {
		server, idToken, refreshID := setup(t)

		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, httptest.NewRequest("GET", "/logout?id_token_hint="+url.QueryEscape(idToken), nil))

		require.Equal(t, http.StatusOK, rr.Code)
		assertUntouched(t, server, "testnonce", refreshID)
	})

	// A mismatched hint is refused on POST as well as GET, so a cross-site form
	// submission cannot turn a leaked token into a logout either.
	for _, method := range []string{"GET", "POST"} {
		t.Run("cookie for a different session/"+method, func(t *testing.T) {
			server, idToken, refreshID := setup(t)
			newTestAuthSession(t, server, "attacker", connectorID, "attackernonce")

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(method, "/logout?id_token_hint="+url.QueryEscape(idToken), nil)
			req.AddCookie(testSessionCookie("attackernonce"))
			server.ServeHTTP(rr, req)

			// The hint does not describe the session named by the cookie, so dex asks
			// rather than acts — RP-Initiated Logout §2.
			require.Equal(t, http.StatusOK, rr.Code)
			require.Contains(t, rr.Body.String(), "Do you want to log out?")
			assertUntouched(t, server, "testnonce", refreshID)

			// The attacker's own session is untouched too.
			_, err := server.storage.GetAuthSession(t.Context(), "attackernonce")
			require.NoError(t, err)
		})
	}

	t.Run("stale sid for the same user", func(t *testing.T) {
		server, idToken, refreshID := setup(t)

		// The user signed out and back in: same subject, new session, so the old
		// token's sid no longer matches.
		require.NoError(t, server.storage.DeleteAuthSession(t.Context(), "testnonce"))
		newTestAuthSession(t, server, userID, connectorID, "newnonce")

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/logout?id_token_hint="+url.QueryEscape(idToken), nil)
		req.AddCookie(testSessionCookie("newnonce"))
		server.ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code)
		require.Contains(t, rr.Body.String(), "Do you want to log out?")
		assertUntouched(t, server, "newnonce", refreshID)
	})
}

// TestHandleLogoutEndsOneDevice: two browsers of the same user are two sessions, so
// signing out of one leaves the other signed in — the thing that was impossible while
// a session was keyed by the user and the connector it belonged to.
func TestHandleLogoutEndsOneDevice(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	ctx := t.Context()
	const userID, connectorID = "test-user", "mock"

	newTestAuthSession(t, server, userID, connectorID, "laptop")
	newTestAuthSession(t, server, userID, connectorID, "phone")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(testSessionCookie("phone"))
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "successfully logged out")

	_, err := server.storage.GetAuthSession(ctx, "phone")
	require.ErrorIs(t, err, storage.ErrNotFound)

	laptop, err := server.storage.GetAuthSession(ctx, "laptop")
	require.NoError(t, err, "signing out of one device must leave the other alone")
	require.Equal(t, userID, laptop.UserID)

	// And the laptop's cookie still works, which is what "left alone" has to mean.
	rr = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/logout", nil)
	req.AddCookie(testSessionCookie("laptop"))
	server.ServeHTTP(rr, req)
	require.Contains(t, rr.Body.String(), "Do you want to log out?")
}

// --- back-channel ---

type backchannelCall struct {
	clientID string
	token    string
}

// startBackchannelRP returns a stub relying party that records the logout tokens it
// receives, keyed by the client id passed in the path, and a function that collects
// them.
//
// Delivery runs off the request so that a slow relying party cannot hold up the
// user's logout, which means the response can land before the tokens do. Callers say
// how many they expect: wait(n) blocks until n arrive, wait(0) asserts none ever do.
func startBackchannelRP(t *testing.T, status int) (*httptest.Server, func(n int) []backchannelCall) {
	t.Helper()

	var mu sync.Mutex
	var calls []backchannelCall

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		require.NoError(t, r.ParseForm())

		mu.Lock()
		calls = append(calls, backchannelCall{
			clientID: strings.TrimPrefix(r.URL.Path, "/"),
			token:    r.PostForm.Get("logout_token"),
		})
		mu.Unlock()

		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)

	snapshot := func() []backchannelCall {
		mu.Lock()
		defer mu.Unlock()
		return append([]backchannelCall(nil), calls...)
	}

	return srv, func(n int) []backchannelCall {
		if n == 0 {
			require.Never(t, func() bool { return len(snapshot()) > 0 }, 200*time.Millisecond, 10*time.Millisecond)
		} else {
			require.Eventually(t, func() bool { return len(snapshot()) >= n }, 2*time.Second, 10*time.Millisecond)
		}
		return snapshot()
	}
}

func TestBackchannelLogoutNotifiesSessionClients(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	rp, wait := startBackchannelRP(t, http.StatusOK)

	ctx := t.Context()
	const userID, connectorID = "test-user", "mock"

	// Two clients registered for back-channel logout, one that never registered.
	for _, id := range []string{"web-a", "web-b"} {
		require.NoError(t, server.storage.CreateClient(ctx, storage.Client{
			ID: id, Secret: "secret",
			RedirectURIs:         []string{"https://example.com/callback"},
			BackchannelLogoutURI: rp.URL + "/" + id,
		}))
	}
	require.NoError(t, server.storage.CreateClient(ctx, storage.Client{
		ID: "silent", Secret: "secret", RedirectURIs: []string{"https://example.com/callback"},
	}))

	require.NoError(t, server.storage.CreateAuthSession(ctx, storage.AuthSession{
		ID: "testnonce", Secret: testSessionSecret("testnonce"),
		UserID: userID, ConnectorID: connectorID,
		CreatedAt: time.Now(), LastActivity: time.Now(),
		ClientStates: map[string]*storage.ClientAuthState{
			"web-a":  {AuthenticatedAt: time.Now()},
			"web-b":  {AuthenticatedAt: time.Now()},
			"silent": {AuthenticatedAt: time.Now()},
		},
	}))

	idToken := newTestIDToken(t, server, "web-a", userID, connectorID, "testnonce")
	wantSID, _ := idTokenClaims(t, idToken)["sid"].(string)
	require.NotEmpty(t, wantSID)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/logout?id_token_hint="+url.QueryEscape(idToken), nil)
	req.AddCookie(testSessionCookie("testnonce"))
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	got := wait(2)
	require.Len(t, got, 2, "only clients with a backchannel_logout_uri are notified")

	seen := map[string]bool{}
	for _, call := range got {
		seen[call.clientID] = true

		claims := idTokenClaims(t, call.token)
		require.Equal(t, wantSID, claims["sid"], "logout token carries the session's sid")
		require.Equal(t, call.clientID, claims["aud"])
		require.NotEmpty(t, claims["jti"])
		require.NotEmpty(t, claims["sub"])
		require.NotContains(t, claims, "nonce", "Back-Channel Logout §2.4 forbids nonce")

		events, ok := claims["events"].(map[string]any)
		require.True(t, ok)
		require.Contains(t, events, "http://schemas.openid.net/event/backchannel-logout")

		// Short lifetime bounds the replay window; the spec recommends two minutes.
		require.InDelta(t, claims["iat"].(float64)+120, claims["exp"].(float64), 1)
	}
	require.Equal(t, map[string]bool{"web-a": true, "web-b": true}, seen)
}

// TestBackchannelLogoutToleratesFailingRP: a wedged relying party must not be able to
// keep the user signed in.
func TestBackchannelLogoutToleratesFailingRP(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	rp, wait := startBackchannelRP(t, http.StatusInternalServerError)

	ctx := t.Context()
	const userID, connectorID = "test-user", "mock"

	require.NoError(t, server.storage.CreateClient(ctx, storage.Client{
		ID: "web-a", Secret: "secret",
		RedirectURIs:         []string{"https://example.com/callback"},
		BackchannelLogoutURI: rp.URL + "/web-a",
	}))
	require.NoError(t, server.storage.CreateAuthSession(ctx, storage.AuthSession{
		ID: "testnonce", Secret: testSessionSecret("testnonce"),
		UserID: userID, ConnectorID: connectorID,
		CreatedAt: time.Now(), LastActivity: time.Now(),
		ClientStates: map[string]*storage.ClientAuthState{"web-a": {AuthenticatedAt: time.Now()}},
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(testSessionCookie("testnonce"))
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Len(t, wait(1), 1)

	_, err := server.storage.GetAuthSession(ctx, "testnonce")
	require.ErrorIs(t, err, storage.ErrNotFound, "logout completes despite the RP failing")
}

// TestBackchannelLogoutSSOSharesOneSID: clients reached through ssoSharedWith live in
// the same AuthSession, so they share a sid and are all notified together, whatever
// each of them declared about its refresh tokens.
func TestBackchannelLogoutSSOSharesOneSID(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	rp, wait := startBackchannelRP(t, http.StatusOK)

	ctx := t.Context()
	const userID, connectorID = "test-user", "mock"

	require.NoError(t, server.storage.CreateClient(ctx, storage.Client{
		ID: "app-a", Secret: "secret",
		RedirectURIs:         []string{"https://example.com/callback"},
		BackchannelLogoutURI: rp.URL + "/app-a",
		SSOSharedWith:        []string{"app-b"},
		RefreshTokenLifetime: storage.RefreshTokenLifetimeSession,
	}))
	require.NoError(t, server.storage.CreateClient(ctx, storage.Client{
		ID: "app-b", Secret: "secret",
		RedirectURIs:         []string{"https://example.com/callback"},
		BackchannelLogoutURI: rp.URL + "/app-b",
	}))

	aRefresh := newTestRefreshToken(t, server, userID, connectorID, "app-a")
	bRefresh := newTestRefreshToken(t, server, userID, connectorID, "app-b")
	require.NoError(t, server.storage.CreateOfflineSessions(ctx, storage.OfflineSessions{
		UserID: userID, ConnID: connectorID,
		Refresh: map[string]*storage.RefreshTokenRef{
			"app-a": {ID: aRefresh, ClientID: "app-a"},
			"app-b": {ID: bRefresh, ClientID: "app-b"},
		},
	}))

	require.NoError(t, server.storage.CreateAuthSession(ctx, storage.AuthSession{
		ID: "testnonce", Secret: testSessionSecret("testnonce"),
		UserID: userID, ConnectorID: connectorID,
		CreatedAt: time.Now(), LastActivity: time.Now(),
		ClientStates: map[string]*storage.ClientAuthState{
			"app-a": {AuthenticatedAt: time.Now()},
			"app-b": {AuthenticatedAt: time.Now(), ViaSSO: true},
		},
	}))

	idToken := newTestIDToken(t, server, "app-a", userID, connectorID, "testnonce")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/logout?id_token_hint="+url.QueryEscape(idToken), nil)
	req.AddCookie(testSessionCookie("testnonce"))
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	got := wait(2)
	require.Len(t, got, 2)
	require.Equal(t, idTokenClaims(t, got[0].token)["sid"], idTokenClaims(t, got[1].token)["sid"],
		"one session, one sid, however many clients share it")

	// The SSO peer is logged out over the back channel, not by having its credential
	// pulled out from under it: it never asked to be bound to the session.
	_, err := server.storage.GetRefresh(ctx, aRefresh)
	require.ErrorIs(t, err, storage.ErrNotFound)
	_, err = server.storage.GetRefresh(ctx, bRefresh)
	require.NoError(t, err)
}

// TestIntrospectionFollowsSession is the gateway-side half of logout: RFC 7662 §4
// requires an authorization server to report a revoked token as inactive, and once
// a token is bound to a session, ending the session is what revokes it.
func TestIntrospectionFollowsSession(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	ctx := t.Context()
	const clientID, userID, connectorID = "test-client", "test-user", "mock"

	require.NoError(t, server.storage.CreateClient(ctx, storage.Client{
		ID: clientID, Secret: "secret",
		RedirectURIs: []string{"https://example.com/callback"},
		// Only a client that declared its tokens session-bound is judged by its
		// session here; see sessionAlive in server/introspection.
		RefreshTokenLifetime: storage.RefreshTokenLifetimeSession,
	}))
	newTestAuthSession(t, server, userID, connectorID, "testnonce")

	accessToken := newTestIDToken(t, server, clientID, userID, connectorID, "testnonce")
	require.NotEmpty(t, idTokenClaims(t, accessToken)["sid"])

	introspect := func(t *testing.T) bool {
		t.Helper()
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/token/introspect",
			strings.NewReader(url.Values{"token": {accessToken}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth(clientID, "secret")
		server.ServeHTTP(rr, req)

		var resp struct {
			Active bool `json:"active"`
		}
		require.NoError(t, json.NewDecoder(rr.Result().Body).Decode(&resp))
		return resp.Active
	}

	require.True(t, introspect(t), "a token of a live session is active")

	require.NoError(t, server.storage.DeleteAuthSession(ctx, "testnonce"))
	require.False(t, introspect(t), "ending the session revokes the token")

	// Signing in again creates a session with a new nonce, so a new sid. The old
	// token names the old one and must not come back to life.
	newTestAuthSession(t, server, userID, connectorID, "anothernonce")
	require.False(t, introspect(t), "a fresh session must not revive an old token")

	// The same token, for a client that never asked to be bound to the session, is
	// still active: outliving the session is what standalone means, and it is the
	// half of this that keeps a CLI signed in.
	require.NoError(t, server.storage.UpdateClient(ctx, clientID, func(old storage.Client) (storage.Client, error) {
		old.RefreshTokenLifetime = storage.RefreshTokenLifetimeStandalone
		return old, nil
	}))
	require.True(t, introspect(t), "a standalone client's token is not judged by the session")
}

// TestIntrospectionIgnoresTokensWithoutSession: a token minted without a session
// carries no sid, so there is nothing to check and nothing changes for it.
func TestIntrospectionIgnoresTokensWithoutSession(t *testing.T) {
	httpServer, server := newTestServer(t, nil)
	defer httpServer.Close()

	const clientID = "test-client"
	require.NoError(t, server.storage.CreateClient(t.Context(), storage.Client{
		ID: clientID, Secret: "secret",
		RedirectURIs: []string{"https://example.com/callback"},
	}))

	accessToken := newTestIDToken(t, server, clientID, "test-user", "mock", "")
	require.NotContains(t, idTokenClaims(t, accessToken), "sid")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/token/introspect",
		strings.NewReader(url.Values{"token": {accessToken}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, "secret")
	server.ServeHTTP(rr, req)

	var resp struct {
		Active bool `json:"active"`
	}
	require.NoError(t, json.NewDecoder(rr.Result().Body).Decode(&resp))
	require.True(t, resp.Active)
}

// TestSIDOnlyFromBrowserFlows: the sid says which browser session a token belongs
// to, so a flow that never passed through a browser must not acquire one. Resolving
// it from the user's identity used to hand a password-grant token whatever session
// that user happened to have open — and, with introspection now following the
// session, signing out of the browser would then kill an unrelated token.
func TestSIDOnlyFromBrowserFlows(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	const userID, connectorID = "test-user", "mock"
	newTestAuthSession(t, server, userID, connectorID, "browsernonce")

	// A browser flow supplies the session it belongs to.
	browserToken, _, err := server.issuer.SignIDToken(t.Context(), tokens.Authorization{
		Client:      storage.Client{ID: "web"},
		Claims:      storage.Claims{UserID: userID},
		Scopes:      []string{"openid"},
		ConnectorID: connectorID,
		SessionID:   "browsernonce",
	}, "", "")
	require.NoError(t, err)
	require.NotEmpty(t, idTokenClaims(t, browserToken)["sid"])

	// Everything else leaves it unset, even for a user who is signed in elsewhere.
	otherToken, _, err := server.issuer.SignIDToken(t.Context(), tokens.Authorization{
		Client:      storage.Client{ID: "cli"},
		Claims:      storage.Claims{UserID: userID},
		Scopes:      []string{"openid"},
		ConnectorID: connectorID,
	}, "", "")
	require.NoError(t, err)
	require.NotContains(t, idTokenClaims(t, otherToken), "sid")
}

// TestRefreshSIDFollowsOrigin covers what a refresh does with the sid: it names the
// session the token came from, and is never invented for a token that was not part
// of one.
func TestRefreshSIDFollowsOrigin(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	ctx := t.Context()
	const userID, connectorID = "test-user", "mock"
	newTestAuthSession(t, server, userID, connectorID, "browsernonce")
	const sid = "browsernonce"

	// A refresh token from a browser flow records the session it came from; one
	// from any other flow records nothing.
	fromBrowser, err := server.issuer.Refresh.Create(ctx, tokens.Authorization{
		Client:      storage.Client{ID: "web"},
		Claims:      storage.Claims{UserID: userID},
		Scopes:      []string{"openid", "offline_access"},
		ConnectorID: connectorID,
		SessionID:   sid,
	})
	require.NoError(t, err)
	require.NotEmpty(t, fromBrowser)

	_, err = server.issuer.Refresh.Create(ctx, tokens.Authorization{
		Client:      storage.Client{ID: "cli"},
		Claims:      storage.Claims{UserID: userID},
		Scopes:      []string{"openid", "offline_access"},
		ConnectorID: connectorID,
	})
	require.NoError(t, err)

	offline, err := server.storage.GetOfflineSessions(ctx, userID, connectorID)
	require.NoError(t, err)
	require.Equal(t, sid, offline.Refresh["web"].SessionID, "browser flow records its session")
	require.Empty(t, offline.Refresh["cli"].SessionID, "other flows record none")
}

// TestExpiredSessionHasNoSID: garbage collection removes an expired session only
// every GCFrequency, so between expiry and collection the row is still there. A sid
// resolved from it would tell introspection that tokens of an ended session are
// still good, which is exactly the window this feature exists to close.
func TestExpiredSessionHasNoSID(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	ctx := t.Context()
	const clientID, userID, connectorID = "test-client", "test-user", "mock"
	require.NoError(t, server.storage.CreateClient(ctx, storage.Client{
		ID: clientID, Secret: "secret", RedirectURIs: []string{"https://example.com/cb"},
	}))

	past := time.Now().Add(-time.Hour)
	for _, tc := range []struct {
		name     string
		absolute time.Time
		idle     time.Time
	}{
		{"absolute lifetime", past, time.Now().Add(time.Hour)},
		{"idle timeout", time.Now().Add(time.Hour), past},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := server.storage.DeleteAuthSession(ctx, "expirednonce"); err != nil {
				require.ErrorIs(t, err, storage.ErrNotFound)
			}
			require.NoError(t, server.storage.CreateAuthSession(ctx, storage.AuthSession{
				ID: "expirednonce", Secret: testSessionSecret("expirednonce"),
				UserID: userID, ConnectorID: connectorID,
				CreatedAt: past, LastActivity: past,
				AbsoluteExpiry: tc.absolute, IdleExpiry: tc.idle,
			}))

			// The session is still stored...
			_, err := server.storage.GetAuthSession(ctx, "expirednonce")
			require.NoError(t, err)

			// ...but it is over, so nothing may claim it. Garbage collection runs on
			// its own schedule; a token must not outlive its session by that window.
			require.False(t, server.sessions.Alive(ctx, "expirednonce"))
		})
	}
}

// TestHandleLogoutExpiredSession: an expired session is nothing left to end, so
// logout must not report otherwise or fan out logout tokens for it.
func TestHandleLogoutExpiredSession(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	rp, wait := startBackchannelRP(t, http.StatusOK)

	ctx := t.Context()
	const userID, connectorID = "test-user", "mock"
	require.NoError(t, server.storage.CreateClient(ctx, storage.Client{
		ID: "web-a", Secret: "secret",
		RedirectURIs:         []string{"https://example.com/cb"},
		BackchannelLogoutURI: rp.URL + "/web-a",
	}))

	past := time.Now().Add(-time.Hour)
	require.NoError(t, server.storage.CreateAuthSession(ctx, storage.AuthSession{
		ID: "expirednonce", Secret: testSessionSecret("expirednonce"),
		UserID: userID, ConnectorID: connectorID,
		CreatedAt: past, LastActivity: past,
		AbsoluteExpiry: past, IdleExpiry: past,
		ClientStates: map[string]*storage.ClientAuthState{"web-a": {AuthenticatedAt: time.Now()}},
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(testSessionCookie("expirednonce"))
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "No active session")
	require.Empty(t, wait(0), "an expired session has no relying parties left to tell")
}
