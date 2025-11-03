package sql

import (
	"encoding/json"
	"testing"
)

const (
	testKey1 = "cHxZB8z3TcK9mR6vL2nY5qW8sD1fG4hJ7kM0oP3rT6u="
	testKey2 = "dG5tN3JxOGZoMnZ3YzNrNHBsOW02YjF6eHN5NXVqOGk="
)

func createEncryptor(t *testing.T, keys ...string) *fernetEncryptor {
	t.Helper()
	encryptor, err := newFernetEncryptor(keys)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}
	return encryptor
}

func TestFernetEncryptor_EncryptAndDecrypt(t *testing.T) {
	// Encrypt with old key
	oldEncryptor := createEncryptor(t, testKey1)
	plaintext := "sensitive-data-to-rotate"

	encrypted, err := oldEncryptor.encrypt(plaintext)
	if err != nil {
		t.Fatalf("encryption with old key failed: %v", err)
	}

	// Create new encryptor with both keys (new primary, old for decryption)
	rotatedEncryptor := createEncryptor(t, testKey2, testKey1)

	// Should decrypt data encrypted with old key
	decrypted, err := rotatedEncryptor.decrypt(encrypted)
	if err != nil {
		t.Fatalf("decryption with rotated keys failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("key rotation failed: expected %s, got %s", plaintext, decrypted)
	}
}

func TestFernetEncryptor_DecryptPlaintextBackwardCompatibility(t *testing.T) {
	encryptor := createEncryptor(t, testKey1)

	// During migration, some values might not be encrypted yet
	plaintextValue := "not-yet-encrypted-password"

	// Decrypt should handle plaintext values (no prefix)
	decrypted, err := encryptor.decrypt(plaintextValue)
	if err != nil {
		t.Fatalf("decrypting plaintext value failed: %v", err)
	}

	if decrypted != plaintextValue {
		t.Errorf("plaintext passthrough failed: expected %s, got %s", plaintextValue, decrypted)
	}

	t.Log("✓ Backward compatibility - plaintext values pass through unchanged")
}

func TestHasEncryptedFields(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]any
		expected bool
	}{
		{
			name: "no encrypted fields",
			config: map[string]any{
				"clientID":     "test-id",
				"clientSecret": "plain-secret",
				"redirectURI":  "https://example.com/callback",
			},
			expected: false,
		},
		{
			name: "has encrypted field",
			config: map[string]any{
				"clientID":     "test-id",
				"clientSecret": "encrypted:gAAAAABh...",
				"redirectURI":  "https://example.com/callback",
			},
			expected: true,
		},
		{
			name: "all fields encrypted",
			config: map[string]any{
				"bindPW": "encrypted:gAAAAABh123...",
				"bindDN": "encrypted:gAAAAABh456...",
			},
			expected: true,
		},
		{
			name:     "empty config",
			config:   map[string]any{},
			expected: false,
		},
		{
			name: "non-string values",
			config: map[string]any{
				"port":    5432,
				"enabled": true,
			},
			expected: false,
		},
	}

	svc := &encryptionService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configJSON, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("failed to marshal config: %v", err)
			}

			result := svc.hasEncryptedFields(configJSON)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
