package main

import (
	"log/slog"
	"testing"

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
