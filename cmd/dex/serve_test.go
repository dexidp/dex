package main

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/tokens"
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
	tests := []struct {
		name            string
		refresh         RefreshToken
		want            tokens.ExpiryCeilings
		wantErrContains string
	}{
		{
			name: "all fields set",
			refresh: RefreshToken{
				AbsoluteLifetime:  "100h",
				ValidIfNotUsedFor: "24h",
				ReuseInterval:     "3s",
			},
			want: tokens.ExpiryCeilings{
				IDTokens:                 24 * time.Hour,
				RefreshAbsoluteLifetime:  100 * time.Hour,
				RefreshValidIfNotUsedFor: 24 * time.Hour,
				RefreshReuseInterval:     3 * time.Second,
			},
		},
		{
			name: "refresh unset",
			want: tokens.ExpiryCeilings{IDTokens: 24 * time.Hour},
		},
		{
			name:    "rotation disabled propagates",
			refresh: RefreshToken{DisableRotation: true},
			want: tokens.ExpiryCeilings{
				IDTokens:                24 * time.Hour,
				RefreshRotationDisabled: true,
			},
		},
		{
			name:            "invalid duration",
			refresh:         RefreshToken{AbsoluteLifetime: "not-a-duration"},
			wantErrContains: "parse expiry.refreshTokens.absoluteLifetime",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildExpiryCeilings(24*time.Hour, tc.refresh)
			if tc.wantErrContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrContains)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
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
