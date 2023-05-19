package tokens

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"log/slog"
	"net/url"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func newTestIssuer(t *testing.T) (*Issuer, storage.Storage) {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)
	store := memory.New(logger)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	sig, err := signer.NewMockSigner(key)
	require.NoError(t, err)

	issuerURL, err := url.Parse("https://issuer.example.com")
	require.NoError(t, err)

	return NewIssuer(store, sig, *issuerURL, time.Hour, time.Now, logger), store
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

// extendingConnector implements connector.PayloadExtender to verify SignIDToken
// offers connectors a chance to add claims from their stashed connector data.
// overrideExp, when non-zero, additionally rewrites the exp claim - mirroring
// the HSDP connector's real behavior of shortening it for Service identities.
type extendingConnector struct {
	err         error
	overrideExp int64
}

func (e *extendingConnector) ExtendPayload(scopes []string, payload, connectorData []byte) ([]byte, error) {
	if e.err != nil {
		return nil, e.err
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	claims["connector_data"] = string(connectorData)
	if e.overrideExp != 0 {
		claims["exp"] = e.overrideExp
	}
	return json.Marshal(claims)
}

func decodeIDTokenClaims(t *testing.T, idToken string) map[string]any {
	t.Helper()
	parsed, err := jose.ParseSigned(idToken, []jose.SignatureAlgorithm{jose.RS256})
	require.NoError(t, err)

	var claims map[string]any
	require.NoError(t, json.Unmarshal(parsed.UnsafePayloadWithoutVerification(), &claims))
	return claims
}

// newTestConnectorCache registers conn under "mock" in a cache backed by store,
// mirroring how the server wires Issuer.Connectors to its live connector cache.
func newTestConnectorCache(t *testing.T, store storage.Storage, conn connector.Connector) *connectors.Cache {
	t.Helper()
	cache := connectors.NewCache(store, func(storage.Connector) (connector.Connector, error) {
		return conn, nil
	})
	require.NoError(t, store.CreateConnector(t.Context(), storage.Connector{ID: "mock", ResourceVersion: "1"}))
	return cache
}

func TestSignIDTokenExtendsPayloadViaConnector(t *testing.T) {
	ctx := t.Context()
	iss, store := newTestIssuer(t)
	iss.Connectors = newTestConnectorCache(t, store, &extendingConnector{})

	auth := testAuthorization()
	idToken, _, err := iss.SignIDToken(ctx, auth, "", "")
	require.NoError(t, err)

	claims := decodeIDTokenClaims(t, idToken)
	require.Equal(t, string(auth.ConnectorData), claims["connector_data"])
}

func TestSignIDTokenIgnoresPayloadExtenderError(t *testing.T) {
	ctx := t.Context()
	iss, store := newTestIssuer(t)
	iss.Connectors = newTestConnectorCache(t, store, &extendingConnector{err: errors.New("boom")})

	auth := testAuthorization()
	idToken, _, err := iss.SignIDToken(ctx, auth, "", "")
	require.NoError(t, err)

	claims := decodeIDTokenClaims(t, idToken)
	require.NotContains(t, claims, "connector_data")
}

func TestSignIDTokenSkipsExtenderWithoutConnectorData(t *testing.T) {
	ctx := t.Context()
	iss, store := newTestIssuer(t)
	iss.Connectors = newTestConnectorCache(t, store, &extendingConnector{})

	auth := testAuthorization()
	auth.ConnectorData = nil
	idToken, _, err := iss.SignIDToken(ctx, auth, "", "")
	require.NoError(t, err)

	claims := decodeIDTokenClaims(t, idToken)
	require.NotContains(t, claims, "connector_data")
}

// TestSignIDToken_ReturnedExpiryReflectsConnectorOverriddenExp guards the
// real, confirmed-live root cause (2026-09-11) of centcom's cron jobs
// consistently failing on every retry: the HSDP connector deliberately
// shortens the exp claim for Service identity tokens (to match the
// underlying IAM service token's own lifetime), but SignIDToken kept
// returning the ORIGINAL, pre-ExtendPayload expiry - so the token-exchange
// grant's expires_in response field (computed from that stale expiry)
// reported ~8h while the actual signed token expired in ~30 minutes. Any
// caller trusting expires_in over the token's own exp claim (as centcom's
// ServiceTokenProvider did) cached an already-expired token for hours.
// The returned expiry must always match whatever exp ends up in the
// signed payload, connector override or not.
func TestSignIDToken_ReturnedExpiryReflectsConnectorOverriddenExp(t *testing.T) {
	ctx := t.Context()
	iss, store := newTestIssuer(t) // idTokensValidFor = 1 hour
	shortExpiry := time.Now().Add(2 * time.Minute).Truncate(time.Second)
	iss.Connectors = newTestConnectorCache(t, store, &extendingConnector{overrideExp: shortExpiry.Unix()})

	auth := testAuthorization()
	idToken, returnedExpiry, err := iss.SignIDToken(ctx, auth, "", "")
	require.NoError(t, err)

	claims := decodeIDTokenClaims(t, idToken)
	require.EqualValues(t, shortExpiry.Unix(), claims["exp"], "the signed token's own exp claim should be the connector-overridden value")

	require.WithinDuration(t, shortExpiry, returnedExpiry, time.Second,
		"SignIDToken's returned expiry must match the token's actual (overridden) exp claim, not the original idTokensValidFor-based value")
}

var _ connector.PayloadExtender = (*extendingConnector)(nil)
