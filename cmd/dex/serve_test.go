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

func TestParseCurvePreferences(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		want        []tls.CurveID
		wantErr     bool
		errContains string
	}{
		{
			name:    "empty",
			input:   nil,
			want:    nil,
			wantErr: false,
		},
		{
			name:  "single valid curve",
			input: []string{"X25519"},
			want:  []tls.CurveID{tls.X25519},
		},
		{
			name:  "multiple valid curves",
			input: []string{"X25519", "CurveP256", "CurveP384", "CurveP521"},
			want: []tls.CurveID{
				tls.X25519,
				tls.CurveP256,
				tls.CurveP384,
				tls.CurveP521,
			},
		},
		{
			name:        "unknown curve",
			input:       []string{"X25519", "UnknownCurve"},
			want:        nil,
			wantErr:     true,
			errContains: `unknown curve: "UnknownCurve"`,
		},
		{
			name:        "unknown curve as first entry",
			input:       []string{"UnknownCurve"},
			want:        nil,
			wantErr:     true,
			errContains: `unknown curve: "UnknownCurve"`,
		},
		{
			name:        "unknown curve after valid curves",
			input:       []string{"CurveP256", "UnknownCurve", "CurveP384"},
			want:        nil,
			wantErr:     true,
			errContains: `unknown curve: "UnknownCurve"`,
		},
		{
			name:  "duplicate curves",
			input: []string{"CurveP256", "CurveP256", "X25519"},
			want: []tls.CurveID{
				tls.CurveP256,
				tls.CurveP256,
				tls.X25519,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCurvePreferences(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				assert.Contains(t, err.Error(), tt.errContains)
				if got != nil {
					t.Errorf("got curves = %v, want nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
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
