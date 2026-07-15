package server

import (
	"log/slog"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func newTestIssuer(t *testing.T) (*tokenIssuer, storage.Storage) {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)
	store := memory.New(logger)
	sig, err := signer.NewMockSigner(testKey)
	require.NoError(t, err)
	issuerURL, err := url.Parse("https://issuer.example.com")
	require.NoError(t, err)

	iss := &tokenIssuer{
		signer: &tokenSigner{
			storage: store, signer: sig, issuerURL: *issuerURL,
			idTokensValidFor: time.Hour, now: time.Now, logger: logger,
		},
		refresh: &refreshTokens{storage: store, now: time.Now, logger: logger},
	}
	return iss, store
}

func testAuthorization() Authorization {
	return Authorization{
		Client:        storage.Client{ID: "client-1"},
		Claims:        storage.Claims{UserID: "u1", Username: "alice", Email: "alice@example.com"},
		Scopes:        []string{"openid", "email", "offline_access"},
		ConnectorID:   "mock",
		Nonce:         "n",
		ConnectorData: []byte(`{"conn":"data"}`),
	}
}

func TestTokenIssuerIssue(t *testing.T) {
	ctx := t.Context()
	iss, store := newTestIssuer(t)
	auth := testAuthorization()

	// With refresh requested: access + id + refresh, and the refresh token is persisted.
	ts, err := iss.Issue(ctx, auth, true)
	require.NoError(t, err)
	require.NotEmpty(t, ts.AccessToken)
	require.NotEmpty(t, ts.IDToken)
	require.NotEmpty(t, ts.RefreshToken)

	var rt internal.RefreshToken
	require.NoError(t, internal.Unmarshal(ts.RefreshToken, &rt))
	stored, err := store.GetRefresh(ctx, rt.RefreshId)
	require.NoError(t, err)
	require.Equal(t, "client-1", stored.ClientID)
	require.Equal(t, "mock", stored.ConnectorID)
	require.Equal(t, auth.ConnectorData, stored.ConnectorData)

	sess, err := store.GetOfflineSessions(ctx, "u1", "mock")
	require.NoError(t, err)
	require.Contains(t, sess.Refresh, "client-1")

	// Without refresh: access + id only.
	ts2, err := iss.Issue(ctx, auth, false)
	require.NoError(t, err)
	require.NotEmpty(t, ts2.AccessToken)
	require.NotEmpty(t, ts2.IDToken)
	require.Empty(t, ts2.RefreshToken)
}

func TestRefreshTokensCreateReplacesExisting(t *testing.T) {
	ctx := t.Context()
	iss, store := newTestIssuer(t)
	auth := testAuthorization()

	first, err := iss.refresh.create(ctx, auth)
	require.NoError(t, err)
	var firstTok internal.RefreshToken
	require.NoError(t, internal.Unmarshal(first, &firstTok))

	// A second create for the same client replaces the old token: the old one is
	// deleted and the offline session points at the new one.
	second, err := iss.refresh.create(ctx, auth)
	require.NoError(t, err)
	var secondTok internal.RefreshToken
	require.NoError(t, internal.Unmarshal(second, &secondTok))
	require.NotEqual(t, firstTok.RefreshId, secondTok.RefreshId)

	if _, err := store.GetRefresh(ctx, firstTok.RefreshId); err == nil {
		t.Error("expected the previous refresh token to be deleted")
	}
	stored, err := store.GetRefresh(ctx, secondTok.RefreshId)
	require.NoError(t, err)
	require.Equal(t, "client-1", stored.ClientID)

	sess, err := store.GetOfflineSessions(ctx, "u1", "mock")
	require.NoError(t, err)
	require.Equal(t, secondTok.RefreshId, sess.Refresh["client-1"].ID)
}
