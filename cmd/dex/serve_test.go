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

func TestBuildConnectorExpiryOverrides(t *testing.T) {
	disable := true
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
					RefreshTokens: &ConnectorRefreshToken{DisableRotation: &disable},
				},
			},
		},
		24*time.Hour,
		RefreshToken{AbsoluteLifetime: "48h"},
	)
	require.NoError(t, err)

	assert.NotContains(t, overrides, "a", "connector without expiry field should not appear")
	assert.Equal(t, 30*time.Minute, overrides["b"].IDTokensValidFor)

	policy := overrides["c"].RefreshTokenPolicy
	require.NotNil(t, policy)
	assert.False(t, policy.RotationEnabled(), "per-connector DisableRotation=true should disable rotation")
}
