package server

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
)

func TestValidateConnectorExpiry_Nil(t *testing.T) {
	require.NoError(t, ValidateConnectorExpiry(nil, ExpiryCeilings{}))
}

func TestValidateConnectorExpiry_IDTokensExceeds(t *testing.T) {
	err := ValidateConnectorExpiry(
		&storage.ConnectorExpiry{IDTokens: "48h"},
		ExpiryCeilings{IDTokens: 24 * time.Hour},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expiry.idTokens (48h0m0s) exceeds the global value (24h0m0s)")
}

func TestValidateConnectorExpiry_NoCeiling(t *testing.T) {
	// Global unset → no ceiling.
	require.NoError(t, ValidateConnectorExpiry(
		&storage.ConnectorExpiry{IDTokens: "48h"},
		ExpiryCeilings{},
	))
}

func TestValidateConnectorExpiry_RefreshAbsoluteLifetimeExceeds(t *testing.T) {
	err := ValidateConnectorExpiry(
		&storage.ConnectorExpiry{
			RefreshTokens: &storage.ConnectorRefreshExpiry{AbsoluteLifetime: "100h"},
		},
		ExpiryCeilings{RefreshAbsoluteLifetime: 24 * time.Hour},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expiry.refreshTokens.absoluteLifetime (100h0m0s) exceeds the global value (24h0m0s)")
}

func TestValidateConnectorExpiry_AllFieldsBelowCeiling(t *testing.T) {
	disable := true
	require.NoError(t, ValidateConnectorExpiry(
		&storage.ConnectorExpiry{
			IDTokens: "10m",
			RefreshTokens: &storage.ConnectorRefreshExpiry{
				DisableRotation:   &disable,
				ReuseInterval:     "1s",
				AbsoluteLifetime:  "1h",
				ValidIfNotUsedFor: "30m",
			},
		},
		ExpiryCeilings{
			IDTokens:                 1 * time.Hour,
			RefreshAbsoluteLifetime:  24 * time.Hour,
			RefreshValidIfNotUsedFor: 1 * time.Hour,
			RefreshReuseInterval:     3 * time.Second,
		},
	))
}

func TestBuildConnectorExpiryOverride_Nil(t *testing.T) {
	got, err := buildConnectorExpiryOverride(slog.New(slog.DiscardHandler), "c1", nil, RefreshTokenDefaults{})
	require.NoError(t, err)
	assert.Zero(t, got.IDTokensValidFor)
	assert.Nil(t, got.RefreshTokenPolicy)
}

func TestBuildConnectorExpiryOverride_IDTokensOnly(t *testing.T) {
	got, err := buildConnectorExpiryOverride(
		slog.New(slog.DiscardHandler),
		"c1",
		&storage.ConnectorExpiry{IDTokens: "5m"},
		RefreshTokenDefaults{},
	)
	require.NoError(t, err)
	assert.Equal(t, 5*time.Minute, got.IDTokensValidFor)
	assert.Nil(t, got.RefreshTokenPolicy)
}

func TestBuildConnectorExpiryOverride_RefreshInheritsGlobals(t *testing.T) {
	disable := true
	got, err := buildConnectorExpiryOverride(
		slog.New(slog.DiscardHandler),
		"c1",
		&storage.ConnectorExpiry{
			RefreshTokens: &storage.ConnectorRefreshExpiry{
				DisableRotation:  &disable,
				AbsoluteLifetime: "1h",
				// ValidIfNotUsedFor and ReuseInterval omitted: inherit from defaults
			},
		},
		RefreshTokenDefaults{
			DisableRotation:   false,
			ValidIfNotUsedFor: "30m",
			AbsoluteLifetime:  "100h",
			ReuseInterval:     "3s",
		},
	)
	require.NoError(t, err)
	require.NotNil(t, got.RefreshTokenPolicy)
	assert.False(t, got.RefreshTokenPolicy.RotationEnabled(), "DisableRotation=true overrides global")
}

func TestUpsertConnectorExpiryOverride(t *testing.T) {
	s := &Server{
		logger:                   slog.New(slog.DiscardHandler),
		idTokensValidFor:         time.Hour,
		expiryCeilings:           ExpiryCeilings{IDTokens: time.Hour},
		connectorExpiryOverrides: map[string]ConnectorExpiryOverride{},
	}

	// Accept a tighter override.
	require.NoError(t, s.upsertConnectorExpiryOverride("c1", &storage.ConnectorExpiry{IDTokens: "5m"}))
	assert.Equal(t, 5*time.Minute, s.idTokensValidForConn("c1"))

	// Reject a looser override; map is left untouched.
	err := s.upsertConnectorExpiryOverride("c2", &storage.ConnectorExpiry{IDTokens: "48h"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the global value")
	assert.Equal(t, time.Hour, s.idTokensValidForConn("c2"), "rejected override must not be installed")

	// Clearing the override via nil reverts to the global.
	require.NoError(t, s.upsertConnectorExpiryOverride("c1", nil))
	assert.Equal(t, time.Hour, s.idTokensValidForConn("c1"))
}
