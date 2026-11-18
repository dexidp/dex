package authflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/session"
	"github.com/dexidp/dex/storage"
)

func TestFinalizeLoginBlockedAccount(t *testing.T) {
	t.Setenv("DEX_SESSIONS_ENABLED", "true")

	httpServer, server := newTestHandler(t, func(c *testFlowConfig) {
		c.SessionConfig = &session.Config{AbsoluteLifetime: time.Hour, ValidIfNotUsedFor: time.Hour}
	})
	defer httpServer.Close()

	ctx := t.Context()

	ident := connector.Identity{UserID: "user-1", Email: "user@example.com"}
	authReq := storage.AuthRequest{
		ID:          "login-req",
		ClientID:    "example-app",
		Expiry:      time.Now().Add(time.Hour),
		ConnectorID: "mock",
	}
	require.NoError(t, server.Storage.CreateAuthRequest(ctx, authReq))
	require.NoError(t, server.Storage.CreateUserIdentity(ctx, storage.UserIdentity{
		UserID:              "user-1",
		ConnectorID:         "mock",
		Claims:              storage.Claims{UserID: "user-1", Email: "user@example.com"},
		Consents:            map[string][]string{},
		MFASecrets:          map[string]*storage.MFASecret{},
		WebAuthnCredentials: map[string][]storage.WebAuthnCredential{},
		CreatedAt:           time.Now(),
		LastLogin:           time.Now(),
		BlockedUntil:        time.Now().Add(time.Hour),
	}))

	// Blocked: finalizeLogin must reject without marking the request logged in.
	_, err := server.finalizeLogin(ctx, ident, authReq, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "locked")

	updated, err := server.Storage.GetAuthRequest(ctx, authReq.ID)
	require.NoError(t, err)
	require.False(t, updated.LoggedIn, "blocked account must not be logged in")

	// Clear the block: login should now proceed.
	require.NoError(t, server.Storage.UpdateUserIdentity(ctx, "user-1", "mock", func(u storage.UserIdentity) (storage.UserIdentity, error) {
		u.BlockedUntil = time.Time{}
		return u, nil
	}))
	_, err = server.finalizeLogin(ctx, ident, authReq, nil)
	require.NoError(t, err)
}

// The authorization code flow's success line is the only record of a login that
// operators get, and without the client it cannot answer which application was
// signed in to when several share a connector.
func TestFinalizeLoginLogsClientID(t *testing.T) {
	var logBuf bytes.Buffer
	httpServer, server := newTestHandler(t, func(c *testFlowConfig) {
		c.Logger = slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	})
	defer httpServer.Close()

	ctx := t.Context()

	authReq := storage.AuthRequest{
		ID:          "login-req",
		ClientID:    "example-app",
		Expiry:      time.Now().Add(time.Hour),
		ConnectorID: "mock",
	}
	require.NoError(t, server.Storage.CreateAuthRequest(ctx, authReq))

	ident := connector.Identity{UserID: "user-1", Email: "user@example.com"}
	_, err := server.finalizeLogin(ctx, ident, authReq, nil)
	require.NoError(t, err)

	var logged bool
	for _, line := range strings.Split(strings.TrimSpace(logBuf.String()), "\n") {
		var record map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &record))
		if record["msg"] != "login successful" {
			continue
		}
		logged = true
		require.Equal(t, "example-app", record["client_id"])
	}
	require.True(t, logged, "finalizeLogin did not log a successful login")
}

// statefulLogoutConnector stands for a connector whose login data holds claims in front of
// the fields its logout needs; it keeps only the latter.
type statefulLogoutConnector struct{}

func (statefulLogoutConnector) LogoutSessionData(connectorData []byte) ([]byte, error) {
	if bytes.HasPrefix(connectorData, []byte("fail")) {
		return nil, errors.New("unusable connector data")
	}
	return bytes.TrimPrefix(connectorData, []byte("login-claims+")), nil
}

func (statefulLogoutConnector) LogoutURLWithState(context.Context, []byte, string) (string, []byte, error) {
	return "", nil, nil
}

func (statefulLogoutConnector) HandleLogoutCallbackWithState(context.Context, *http.Request, []byte) error {
	return nil
}

// plainConnector implements no logout capability, like the OIDC and OAuth2 connectors.
type plainConnector struct{}

func TestSessionAuthRequestConnectorData(t *testing.T) {
	httpServer, server := newTestHandler(t, nil)
	defer httpServer.Close()

	ctx := t.Context()
	authReq := storage.AuthRequest{ID: "req", ConnectorData: []byte("login-claims+logout-fields")}

	t.Run("StatefulLogoutConnectorKeepsWhatItAsksFor", func(t *testing.T) {
		got := server.sessionAuthRequest(ctx, authReq, statefulLogoutConnector{})
		require.Equal(t, []byte("logout-fields"), got.ConnectorData)
		require.Equal(t, authReq.ID, got.ID)
		require.Equal(t, []byte("login-claims+logout-fields"), authReq.ConnectorData, "the login's own request must keep its data")
	})

	t.Run("OtherConnectorsDropData", func(t *testing.T) {
		got := server.sessionAuthRequest(ctx, authReq, plainConnector{})
		require.Nil(t, got.ConnectorData)
		require.Equal(t, authReq.ID, got.ID)
	})

	t.Run("UnusableDataIsDropped", func(t *testing.T) {
		got := server.sessionAuthRequest(ctx, storage.AuthRequest{ID: "req", ConnectorData: []byte("fail")}, statefulLogoutConnector{})
		require.Nil(t, got.ConnectorData)
	})
}
