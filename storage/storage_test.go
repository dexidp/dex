package storage

import (
	"crypto"
	"testing"
)

func TestGCResult_IsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		result GCResult
		want   bool
	}{
		{"empty result", GCResult{}, true},
		{"non-empty AuthRequests", GCResult{AuthRequests: 1}, false},
		{"non-empty AuthCodes", GCResult{AuthCodes: 1}, false},
		{"non-empty DeviceRequests", GCResult{DeviceRequests: 1}, false},
		{"non-empty DeviceTokens", GCResult{DeviceTokens: 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewSecureID(t *testing.T) {
	tests := []struct {
		name string
		len  int
		want int
	}{
		{"length 16", 16, 25},
		{"length 32", 32, 51},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := newSecureID(tt.len)
			if len(id) != tt.want {
				t.Errorf("newSecureID() got length %d, want %d", len(id), tt.want)
			}
		})
	}
}

func TestNewHMACKey(t *testing.T) {
	tests := []struct {
		name    string
		hash    crypto.Hash
		wantLen int
	}{
		{"SHA256", crypto.SHA256, 51},
		{"SHA512", crypto.SHA512, 102},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := NewHMACKey(tt.hash)
			if len(key) != tt.wantLen {
				t.Errorf("NewHMACKey() got length %d, want %d", len(key), tt.wantLen)
			}
		})
	}
}

func TestNewDeviceCode(t *testing.T) {
	code := NewDeviceCode()
	if len(code) == 0 {
		t.Error("NewDeviceCode() returned empty code")
	}
}

func TestNewUserCode(t *testing.T) {
	code := NewUserCode()
	if len(code) != 9 || code[4] != '-' {
		t.Errorf("NewUserCode() got %s, want format xxxx-xxxx", code)
	}
}
