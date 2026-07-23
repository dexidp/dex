package tokens

import (
	"crypto/rand"
	"crypto/rsa"
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

func newTestIssuer(t *testing.T) (*Issuer, storage.Storage) {
	t.Helper()
	return newTestIssuerWithExpiry(t, NewExpiry(time.Hour, nil, ExpiryCeilings{}, RefreshTokenDefaults{}), time.Now)
}

func newTestIssuerWithExpiry(t *testing.T, expiry *Expiry, now func() time.Time) (*Issuer, storage.Storage) {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)
	store := memory.New(logger)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	sig, err := signer.NewMockSigner(key)
	require.NoError(t, err)

	issuerURL, err := url.Parse("https://issuer.example.com")
	require.NoError(t, err)

	return NewIssuer(store, sig, *issuerURL, expiry, now, logger), store
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

func TestIssuerIssue(t *testing.T) {
	ctx := t.Context()
	iss, store := newTestIssuer(t)
	auth := testAuthorization()

	// With refresh requested: access + id + refresh, and the refresh token is persisted.
	ts, err := iss.Issue(ctx, auth, "", true)
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
	ts2, err := iss.Issue(ctx, auth, "", false)
	require.NoError(t, err)
	require.NotEmpty(t, ts2.AccessToken)
	require.NotEmpty(t, ts2.IDToken)
	require.Empty(t, ts2.RefreshToken)
}

func TestSignIDTokenUsesConnectorExpiryOverride(t *testing.T) {
	ctx := t.Context()
	now := time.Now().UTC()

	expiry := NewExpiry(time.Hour, nil, ExpiryCeilings{IDTokens: time.Hour}, RefreshTokenDefaults{})
	require.NoError(t, expiry.Upsert("short", &storage.ConnectorExpiry{IDTokens: "5m"}))

	iss, _ := newTestIssuerWithExpiry(t, expiry, func() time.Time { return now })

	auth := testAuthorization()
	auth.ConnectorID = "short"
	_, expiryShort, err := iss.SignIDToken(ctx, auth, "", "")
	require.NoError(t, err)
	require.Equal(t, now.Add(5*time.Minute), expiryShort.UTC(),
		"per-connector override must apply")

	auth.ConnectorID = "unknown"
	_, expiryGlobal, err := iss.SignIDToken(ctx, auth, "", "")
	require.NoError(t, err)
	require.Equal(t, now.Add(time.Hour), expiryGlobal.UTC(),
		"unknown connector must fall back to global")
}
