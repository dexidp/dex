package main

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server"
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

func TestBuildExpiryCeilings(t *testing.T) {
	c, err := buildExpiryCeilings(24*time.Hour, RefreshToken{
		AbsoluteLifetime:  "100h",
		ValidIfNotUsedFor: "24h",
		ReuseInterval:     "3s",
	})
	require.NoError(t, err)
	assert.Equal(t, server.ExpiryCeilings{
		IDTokens:                 24 * time.Hour,
		RefreshAbsoluteLifetime:  100 * time.Hour,
		RefreshValidIfNotUsedFor: 24 * time.Hour,
		RefreshReuseInterval:     3 * time.Second,
	}, c)
}

func TestBuildExpiryCeilingsRefreshUnset(t *testing.T) {
	c, err := buildExpiryCeilings(24*time.Hour, RefreshToken{})
	require.NoError(t, err)
	assert.Equal(t, server.ExpiryCeilings{IDTokens: 24 * time.Hour}, c)
}

func TestBuildExpiryCeilingsRotationDisabled(t *testing.T) {
	c, err := buildExpiryCeilings(24*time.Hour, RefreshToken{DisableRotation: true})
	require.NoError(t, err)
	assert.True(t, c.RefreshRotationDisabled)
}

func TestBuildExpiryCeilingsInvalidDuration(t *testing.T) {
	_, err := buildExpiryCeilings(24*time.Hour, RefreshToken{AbsoluteLifetime: "not-a-duration"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse expiry.refreshTokens.absoluteLifetime")
}

func TestToStorageConnectorCarriesExpiry(t *testing.T) {
	disable := true
	sc, err := ToStorageConnector(Connector{
		ID: "c1", Type: "mockCallback", Name: "c1",
		Expiry: &ConnectorExpiry{
			IDTokens: "15m",
			RefreshTokens: &ConnectorRefreshExpiry{
				DisableRotation:   &disable,
				AbsoluteLifetime:  "24h",
				ValidIfNotUsedFor: "1h",
				ReuseInterval:     "3s",
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, sc.Expiry)
	assert.Equal(t, "15m", sc.Expiry.IDTokens)
	require.NotNil(t, sc.Expiry.RefreshTokens)
	assert.Equal(t, "24h", sc.Expiry.RefreshTokens.AbsoluteLifetime)
	require.NotNil(t, sc.Expiry.RefreshTokens.DisableRotation)
	assert.True(t, *sc.Expiry.RefreshTokens.DisableRotation)

	sc, err = ToStorageConnector(Connector{ID: "c1", Type: "mockCallback", Name: "c1"})
	require.NoError(t, err)
	assert.Nil(t, sc.Expiry)
}
