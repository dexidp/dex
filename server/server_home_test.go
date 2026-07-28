package server

import (
	"net/http"
	"net/http/httptest"
	"strconv"
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

	require.NoError(t, server.storage.CreateAuthSession(ctx, storage.AuthSession{
		UserID:       userID,
		ConnectorID:  connectorID,
		Nonce:        nonce,
		CreatedAt:    sessionStart,
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
		LastLogin: lastLogin,
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "dex_session",
		Value: internal.SessionCookieValue(userID, connectorID, nonce, testSessionKey),
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

	// Both the machine-readable epoch and the text a reader without JavaScript
	// sees have to be the session's start.
	require.Contains(t, body, strconv.FormatInt(sessionStart.Unix(), 10))
	require.Contains(t, body, sessionStart.UTC().Format("2 Jan 2006, 15:04 UTC"))
	require.NotContains(t, body, strconv.FormatInt(lastLogin.Unix(), 10))
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
