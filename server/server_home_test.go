package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

func TestHomeNoSessions(t *testing.T) {
	httpServer, server := newTestServer(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	require.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()
	require.Contains(t, body, "Dex IdP")
	require.Contains(t, body, "Discovery")
	require.NotContains(t, body, "/logout")
}

func TestHomeNotLoggedIn(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	require.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()
	require.Contains(t, body, ".well-known/openid-configuration")
	require.Contains(t, body, "Not signed in")
	require.NotContains(t, body, "/logout")
}

func TestHomeLoggedIn(t *testing.T) {
	httpServer, server := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	ctx := t.Context()
	userID := "test-user"
	connectorID := "mock"
	nonce := "testnonce"
	now := time.Now()

	// The identity's last login is deliberately later than this session's start:
	// the page reports the session it is describing, not the last time the same
	// user signed in somewhere else.
	sessionStart := now.Add(-3 * time.Hour)
	lastLogin := now.Add(-5 * time.Minute)

	// The idle deadline is the earlier of the two, so that is the one the page
	// has to report — and it has to say the deadline is the sliding one.
	idleExpiry := now.Add(30 * time.Minute)
	absoluteExpiry := now.Add(21 * time.Hour)

	require.NoError(t, server.storage.CreateAuthSession(ctx, storage.AuthSession{
		ID:             nonce,
		Secret:         testSessionSecret(nonce),
		UserID:         userID,
		ConnectorID:    connectorID,
		CreatedAt:      sessionStart,
		LastActivity:   now,
		IdleExpiry:     idleExpiry,
		AbsoluteExpiry: absoluteExpiry,
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
		LastLogin: lastLogin,
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "dex_session",
		Value: internal.SessionCookieValue(nonce, testSessionSecret(nonce), testSessionKey),
	})

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()
	require.Contains(t, body, "/logout")
	require.Contains(t, body, "testuser")
	require.Contains(t, body, "test@example.com")
	require.Contains(t, body, "Mock")
	// Groups are listed outright rather than hidden behind a disclosure.
	require.Contains(t, body, "admins")
	require.Contains(t, body, "devs")
	require.Contains(t, body, ".well-known/openid-configuration")
	require.NotContains(t, body, "Not signed in")

	// Both the datetime attribute and the text a reader without JavaScript sees
	// have to be the session's start, not the identity's last login.
	require.Contains(t, body, sessionStart.UTC().Format(time.RFC3339))
	require.Contains(t, body, sessionStart.UTC().Format("2 Jan 2006, 15:04 UTC"))
	require.NotContains(t, body, lastLogin.UTC().Format(time.RFC3339))

	// The expiry row reports the earlier deadline and names it as the idle one.
	require.Contains(t, body, "Expires if idle")
	require.NotContains(t, body, "Session ends")
	require.Contains(t, body, idleExpiry.UTC().Format(time.RFC3339))
	require.NotContains(t, body, absoluteExpiry.UTC().Format(time.RFC3339))
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
	require.NotContains(t, body, "/logout")
	require.Contains(t, body, "Not signed in")
	require.Contains(t, body, ".well-known/openid-configuration")
}
