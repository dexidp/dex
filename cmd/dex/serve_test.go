package main

import (
	"crypto/tls"
	"log/slog"
	"strings"
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

func TestParseCipherSuites(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		wantErr     bool
		errContains string
		validate    func(t *testing.T, got []uint16)
	}{
		{
			name:    "empty input",
			input:   []string{},
			wantErr: false,
			validate: func(t *testing.T, got []uint16) {
				assert.Empty(t, got)
			},
		},
		{
			name:    "single valid cipher",
			input:   []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
			wantErr: false,
			validate: func(t *testing.T, got []uint16) {
				require.Len(t, got, 1)
				assert.Equal(t, tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, got[0])
			},
		},
		{
			name: "multiple valid ciphers",
			input: []string{
				"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
				"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			},
			wantErr: false,
			validate: func(t *testing.T, got []uint16) {
				require.Len(t, got, 2)
				assert.Equal(t, tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, got[0])
				assert.Equal(t, tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384, got[1])
			},
		},
		{
			name:    "insecure cipher",
			input:   []string{"TLS_RSA_WITH_3DES_EDE_CBC_SHA"},
			wantErr: false,
			validate: func(t *testing.T, got []uint16) {
				require.Len(t, got, 1)
				assert.Equal(t, tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA, got[0])
			},
		},
		{
			name:        "unsupported cipher",
			input:       []string{"TLS_FAKE_CIPHER"},
			wantErr:     true,
			errContains: `unsupported cipher suite "TLS_FAKE_CIPHER"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCipherSuites(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.errContains))
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			tt.validate(t, got)
		})
	}
}
