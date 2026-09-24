package authflow

import (
	"errors"
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

// TestFinalizeLoginAlreadyFinalized reproduces a double-submitted login (e.g. the
// user double-clicks the submit button, or resubmits a stale page reached via the
// browser back button): by the time the duplicate request reaches finalizeLogin,
// the first submission has already completed and the AuthRequest is gone. The
// caller needs to distinguish this from a generic storage failure, so the
// returned error must still satisfy errors.Is(err, storage.ErrNotFound).
func TestFinalizeLoginAlreadyFinalized(t *testing.T) {
	httpServer, server := newTestHandler(t, nil)
	defer httpServer.Close()

	ctx := t.Context()

	ident := connector.Identity{UserID: "user-1", Email: "user@example.com"}
	authReq := storage.AuthRequest{
		ID:          "login-req",
		ClientID:    "example-app",
		Expiry:      time.Now().Add(time.Hour),
		ConnectorID: "mock",
	}
	// Do not create the AuthRequest, simulating that an earlier, still-in-flight
	// submission already finalized and deleted it.
	_, err := server.finalizeLogin(ctx, ident, authReq, nil)
	require.Error(t, err)
	require.True(t, errors.Is(err, storage.ErrNotFound), "expected error to wrap storage.ErrNotFound, got: %v", err)
}
