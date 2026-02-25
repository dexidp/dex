package signer

import (
	"encoding/json"
	"os"
	"testing"
)

func TestVaultConfigUnmarshalJSON_WithEnvVars(t *testing.T) {
	// Save original environment variables
	originalAddr := os.Getenv("VAULT_ADDR")
	originalToken := os.Getenv("VAULT_TOKEN")
	defer func() {
		os.Setenv("VAULT_ADDR", originalAddr)
		os.Setenv("VAULT_TOKEN", originalToken)
	}()

	// Set environment variables
	os.Setenv("VAULT_ADDR", "http://vault.example.com:8200")
	os.Setenv("VAULT_TOKEN", "s.xxxxxxxxxxxxxxxx")

	tests := []struct {
		name    string
		json    string
		want    VaultConfig
		wantErr bool
	}{
		{
			name: "empty config uses env vars",
			json: `{"keyName": "signing-key"}`,
			want: VaultConfig{
				Addr:    "http://vault.example.com:8200",
				Token:   "s.xxxxxxxxxxxxxxxx",
				KeyName: "signing-key",
			},
			wantErr: false,
		},
		{
			name: "config values override env vars",
			json: `{"addr": "http://custom.vault.com:8200", "token": "s.custom", "keyName": "signing-key"}`,
			want: VaultConfig{
				Addr:    "http://custom.vault.com:8200",
				Token:   "s.custom",
				KeyName: "signing-key",
			},
			wantErr: false,
		},
		{
			name: "partial config uses env vars for missing values",
			json: `{"addr": "http://custom.vault.com:8200", "keyName": "signing-key"}`,
			want: VaultConfig{
				Addr:    "http://custom.vault.com:8200",
				Token:   "s.xxxxxxxxxxxxxxxx",
				KeyName: "signing-key",
			},
			wantErr: false,
		},
		{
			name: "empty token in config uses env var",
			json: `{"addr": "http://custom.vault.com:8200", "token": "", "keyName": "signing-key"}`,
			want: VaultConfig{
				Addr:    "http://custom.vault.com:8200",
				Token:   "s.xxxxxxxxxxxxxxxx",
				KeyName: "signing-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got VaultConfig
			err := json.Unmarshal([]byte(tt.json), &got)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got.Addr != tt.want.Addr {
				t.Errorf("Addr: got %q, want %q", got.Addr, tt.want.Addr)
			}
			if got.Token != tt.want.Token {
				t.Errorf("Token: got %q, want %q", got.Token, tt.want.Token)
			}
			if got.KeyName != tt.want.KeyName {
				t.Errorf("KeyName: got %q, want %q", got.KeyName, tt.want.KeyName)
			}
		})
	}
}

func TestVaultConfigUnmarshalJSON_WithoutEnvVars(t *testing.T) {
	// Save original environment variables
	originalAddr := os.Getenv("VAULT_ADDR")
	originalToken := os.Getenv("VAULT_TOKEN")
	defer func() {
		os.Setenv("VAULT_ADDR", originalAddr)
		os.Setenv("VAULT_TOKEN", originalToken)
	}()

	// Unset environment variables
	os.Unsetenv("VAULT_ADDR")
	os.Unsetenv("VAULT_TOKEN")

	tests := []struct {
		name    string
		json    string
		want    VaultConfig
		wantErr bool
	}{
		{
			name: "config values used when env vars not set",
			json: `{"addr": "http://vault.example.com:8200", "token": "s.xxxxxxxxxxxxxxxx", "keyName": "signing-key"}`,
			want: VaultConfig{
				Addr:    "http://vault.example.com:8200",
				Token:   "s.xxxxxxxxxxxxxxxx",
				KeyName: "signing-key",
			},
			wantErr: false,
		},
		{
			name: "empty config when env vars not set",
			json: `{"keyName": "signing-key"}`,
			want: VaultConfig{
				Addr:    "",
				Token:   "",
				KeyName: "signing-key",
			},
			wantErr: false,
		},
		{
			name: "only keyName required in config",
			json: `{"keyName": "my-key"}`,
			want: VaultConfig{
				Addr:    "",
				Token:   "",
				KeyName: "my-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got VaultConfig
			err := json.Unmarshal([]byte(tt.json), &got)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got.Addr != tt.want.Addr {
				t.Errorf("Addr: got %q, want %q", got.Addr, tt.want.Addr)
			}
			if got.Token != tt.want.Token {
				t.Errorf("Token: got %q, want %q", got.Token, tt.want.Token)
			}
			if got.KeyName != tt.want.KeyName {
				t.Errorf("KeyName: got %q, want %q", got.KeyName, tt.want.KeyName)
			}
		})
	}
}

func TestVaultConfigUnmarshalJSON_InvalidJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "invalid json",
			json:    `{invalid json}`,
			wantErr: true,
		},
		{
			name:    "empty json",
			json:    `{}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got VaultConfig
			err := json.Unmarshal([]byte(tt.json), &got)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
