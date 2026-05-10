package server

import (
	"context"
	"log/slog"
	"net/url"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func TestIDTokensValidForConn(t *testing.T) {
	override := &RefreshTokenPolicy{}
	s := &Server{
		idTokensValidFor: time.Hour,
		connectorExpiryOverrides: map[string]ConnectorExpiryOverride{
			"shortlived": {IDTokensValidFor: 5 * time.Minute},
			"refreshonly": {RefreshTokenPolicy: override},
		},
	}

	assert.Equal(t, 5*time.Minute, s.idTokensValidForConn("shortlived"),
		"per-connector override should win")
	assert.Equal(t, time.Hour, s.idTokensValidForConn("refreshonly"),
		"zero IDTokensValidFor should fall back to global")
	assert.Equal(t, time.Hour, s.idTokensValidForConn("unknown"),
		"missing entry should fall back to global")
}

func TestRefreshTokenPolicyForConn(t *testing.T) {
	global := &RefreshTokenPolicy{rotateRefreshTokens: true}
	perConnector := &RefreshTokenPolicy{rotateRefreshTokens: false}

	s := &Server{
		refreshTokenPolicy: global,
		connectorExpiryOverrides: map[string]ConnectorExpiryOverride{
			"custom":    {RefreshTokenPolicy: perConnector},
			"idonly":    {IDTokensValidFor: time.Minute},
			"nilpolicy": {},
		},
	}

	assert.Same(t, perConnector, s.refreshTokenPolicyForConn("custom"),
		"per-connector override should win")
	assert.Same(t, global, s.refreshTokenPolicyForConn("idonly"),
		"nil per-connector policy should fall back to global")
	assert.Same(t, global, s.refreshTokenPolicyForConn("nilpolicy"),
		"empty override should fall back to global")
	assert.Same(t, global, s.refreshTokenPolicyForConn("unknown"),
		"missing entry should fall back to global")
}

func TestIDTokensValidForConnNoOverrides(t *testing.T) {
	s := &Server{idTokensValidFor: 42 * time.Minute}
	assert.Equal(t, 42*time.Minute, s.idTokensValidForConn("any"),
		"nil override map must not panic and must return global")
}

// TestNewIDTokenUsesConnectorOverride verifies that newIDToken applies the
// per-connector idTokensValidFor override at issuance time, not the global.
func TestNewIDTokenUsesConnectorOverride(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.DiscardHandler)
	store := memory.New(logger)

	now := time.Now().UTC()
	err := store.UpdateKeys(ctx, func(keys storage.Keys) (storage.Keys, error) {
		keys.SigningKey = &jose.JSONWebKey{
			Key:       testKey,
			KeyID:     "test-rs256",
			Algorithm: string(jose.RS256),
			Use:       "sig",
		}
		keys.SigningKeyPub = &jose.JSONWebKey{
			Key:       testKey.Public(),
			KeyID:     "test-rs256",
			Algorithm: string(jose.RS256),
			Use:       "sig",
		}
		keys.NextRotation = now.Add(time.Hour)
		return keys, nil
	})
	require.NoError(t, err)

	localConfig := signer.LocalConfig{
		KeysRotationPeriod: time.Hour.String(),
		Algorithm:          jose.RS256,
	}
	sig, err := localConfig.Open(ctx, store, time.Hour, func() time.Time { return now }, logger)
	require.NoError(t, err)
	sig.Start(ctx)

	issuerURL, err := url.Parse("https://issuer.example.com")
	require.NoError(t, err)

	s := &Server{
		signer:           sig,
		issuerURL:        *issuerURL,
		logger:           logger,
		now:              func() time.Time { return now },
		idTokensValidFor: time.Hour,
		connectorExpiryOverrides: map[string]ConnectorExpiryOverride{
			"short": {IDTokensValidFor: 5 * time.Minute},
		},
	}

	_, expiryShort, err := s.newIDToken(ctx, "client",
		storage.Claims{UserID: "u1", Username: "alice"},
		[]string{"openid"}, "n", "", "", "short", time.Time{})
	require.NoError(t, err)
	assert.Equal(t, now.Add(5*time.Minute), expiryShort.UTC(),
		"per-connector override must apply")

	_, expiryGlobal, err := s.newIDToken(ctx, "client",
		storage.Claims{UserID: "u1", Username: "alice"},
		[]string{"openid"}, "n", "", "", "unknown", time.Time{})
	require.NoError(t, err)
	assert.Equal(t, now.Add(time.Hour), expiryGlobal.UTC(),
		"unknown connector must fall back to global")
}
