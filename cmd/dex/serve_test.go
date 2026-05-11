package main

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		logger, err := newLogger(slog.LevelInfo, "json", nil)
		require.NoError(t, err)
		require.NotEqual(t, (*slog.Logger)(nil), logger)
	})

	t.Run("Text", func(t *testing.T) {
		logger, err := newLogger(slog.LevelError, "text", nil)
		require.NoError(t, err)
		require.NotEqual(t, (*slog.Logger)(nil), logger)
	})

	t.Run("Unknown", func(t *testing.T) {
		logger, err := newLogger(slog.LevelError, "gofmt", nil)
		require.Error(t, err)
		require.Equal(t, "log format is not one of the supported values (json, text): gofmt", err.Error())
		require.Equal(t, (*slog.Logger)(nil), logger)
	})
}

func TestBuildConnectorExpiryOverrides_NoConnectors(t *testing.T) {
	overrides, err := buildConnectorExpiryOverrides(
		slog.New(slog.DiscardHandler), nil, 24*time.Hour, RefreshToken{},
	)
	require.NoError(t, err)
	assert.Empty(t, overrides)
}

func TestBuildConnectorExpiryOverrides_NoExpiryField(t *testing.T) {
	overrides, err := buildConnectorExpiryOverrides(
		slog.New(slog.DiscardHandler),
		[]Connector{{ID: "c1", Type: "mock", Name: "c1"}},
		24*time.Hour, RefreshToken{},
	)
	require.NoError(t, err)
	assert.Empty(t, overrides, "connector without expiry field should not appear in override map")
}

func TestBuildConnectorExpiryOverrides_IDTokens(t *testing.T) {
	overrides, err := buildConnectorExpiryOverrides(
		slog.New(slog.DiscardHandler),
		[]Connector{{
			ID: "c1", Type: "mock", Name: "c1",
			Expiry: &ConnectorExpiry{IDTokens: "15m"},
		}},
		24*time.Hour, RefreshToken{},
	)
	require.NoError(t, err)
	require.Contains(t, overrides, "c1")
	assert.Equal(t, 15*time.Minute, overrides["c1"].IDTokensValidFor)
	assert.Nil(t, overrides["c1"].RefreshTokenPolicy)
}

func TestBuildConnectorExpiryOverrides_IDTokensExceedsGlobal(t *testing.T) {
	_, err := buildConnectorExpiryOverrides(
		slog.New(slog.DiscardHandler),
		[]Connector{{
			ID: "c1", Type: "mock", Name: "c1",
			Expiry: &ConnectorExpiry{IDTokens: "48h"},
		}},
		24*time.Hour, RefreshToken{},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expiry.idTokens (48h0m0s) exceeds the global value (24h0m0s)")
}

func TestBuildConnectorExpiryOverrides_IDTokensInvalidDuration(t *testing.T) {
	_, err := buildConnectorExpiryOverrides(
		slog.New(slog.DiscardHandler),
		[]Connector{{
			ID: "c1", Type: "mock", Name: "c1",
			Expiry: &ConnectorExpiry{IDTokens: "not-a-duration"},
		}},
		24*time.Hour, RefreshToken{},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse expiry.idTokens")
}

func TestBuildConnectorExpiryOverrides_RefreshTokensOverrideAndInherit(t *testing.T) {
	disable := true
	global := RefreshToken{
		DisableRotation:   false,
		ReuseInterval:     "3s",
		AbsoluteLifetime:  "100h",
		ValidIfNotUsedFor: "24h",
	}
	overrides, err := buildConnectorExpiryOverrides(
		slog.New(slog.DiscardHandler),
		[]Connector{{
			ID: "c1", Type: "mock", Name: "c1",
			Expiry: &ConnectorExpiry{
				RefreshTokens: &ConnectorRefreshToken{
					DisableRotation:  &disable,
					AbsoluteLifetime: "10h",
					// ReuseInterval and ValidIfNotUsedFor omitted: inherit from global
				},
			},
		}},
		24*time.Hour, global,
	)
	require.NoError(t, err)
	require.Contains(t, overrides, "c1")

	policy := overrides["c1"].RefreshTokenPolicy
	require.NotNil(t, policy)
	assert.False(t, policy.RotationEnabled(), "DisableRotation=true should disable rotation")

	// Probe the numeric fields via the boundary-crossing behavior of the policy methods,
	// using time.Now offsets since the policy's internal clock uses time.Now.
	now := time.Now()
	assert.False(t, policy.CompletelyExpired(now.Add(-9*time.Hour)), "9h < absoluteLifetime=10h")
	assert.True(t, policy.CompletelyExpired(now.Add(-11*time.Hour)), "11h > absoluteLifetime=10h")

	assert.False(t, policy.ExpiredBecauseUnused(now.Add(-23*time.Hour)), "23h < validIfNotUsedFor=24h")
	assert.True(t, policy.ExpiredBecauseUnused(now.Add(-25*time.Hour)), "25h > validIfNotUsedFor=24h")

	assert.True(t, policy.AllowedToReuse(now.Add(-2*time.Second)), "2s < reuseInterval=3s")
	assert.False(t, policy.AllowedToReuse(now.Add(-4*time.Second)), "4s > reuseInterval=3s")
}

func TestBuildConnectorExpiryOverrides_MultipleConnectors(t *testing.T) {
	overrides, err := buildConnectorExpiryOverrides(
		slog.New(slog.DiscardHandler),
		[]Connector{
			{ID: "a", Type: "mock", Name: "a"},
			{
				ID: "b", Type: "mock", Name: "b",
				Expiry: &ConnectorExpiry{IDTokens: "30m"},
			},
			{
				ID: "c", Type: "mock", Name: "c",
				Expiry: &ConnectorExpiry{
					RefreshTokens: &ConnectorRefreshToken{AbsoluteLifetime: "12h"},
				},
			},
		},
		24*time.Hour, RefreshToken{AbsoluteLifetime: "48h"},
	)
	require.NoError(t, err)
	assert.NotContains(t, overrides, "a")
	assert.Equal(t, 30*time.Minute, overrides["b"].IDTokensValidFor)
	require.NotNil(t, overrides["c"].RefreshTokenPolicy)

	// Confirm 12h absoluteLifetime took effect.
	now := time.Now()
	policy := overrides["c"].RefreshTokenPolicy
	assert.False(t, policy.CompletelyExpired(now.Add(-11*time.Hour)))
	assert.True(t, policy.CompletelyExpired(now.Add(-13*time.Hour)))
}
