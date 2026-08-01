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
		idTokens        time.Duration
		refresh         RefreshToken
		want            tokens.ExpiryCeilings
		wantErrContains string
	}{
		{
			name:     "all fields set",
			idTokens: 24 * time.Hour,
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
			name:     "refresh unset",
			idTokens: 24 * time.Hour,
			want:     tokens.ExpiryCeilings{IDTokens: 24 * time.Hour},
		},
		{
			name:     "rotation disabled propagates",
			idTokens: 24 * time.Hour,
			refresh:  RefreshToken{DisableRotation: true},
			want: tokens.ExpiryCeilings{
				IDTokens:                24 * time.Hour,
				RefreshRotationDisabled: true,
			},
		},
		{
			name:            "invalid duration",
			idTokens:        24 * time.Hour,
			refresh:         RefreshToken{AbsoluteLifetime: "not-a-duration"},
			wantErrContains: `invalid config value "not-a-duration" for expiry.refreshTokens.absoluteLifetime`,
		},
		{
			name:            "negative duration",
			idTokens:        24 * time.Hour,
			refresh:         RefreshToken{ValidIfNotUsedFor: "-1h"},
			wantErrContains: "expiry.refreshTokens.validIfNotUsedFor must not be negative",
		},
		{
			name:            "zero idTokens",
			idTokens:        0,
			wantErrContains: "expiry.idTokens must be positive",
		},
		{
			name:            "negative idTokens",
			idTokens:        -time.Hour,
			wantErrContains: "expiry.idTokens must be positive",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildExpiryCeilings(tc.idTokens, tc.refresh)
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
