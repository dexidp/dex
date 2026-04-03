package signer

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"io"

	"github.com/go-jose/go-jose/v4"
)

// MockConfig creates a mock signer with a static key for testing.
type MockConfig struct {
	Key *rsa.PrivateKey
}

// Open creates a new mock signer.
func (c *MockConfig) Open(_ context.Context) (Signer, error) {
	if c.Key == nil {
		// Generate a new key if not provided
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}
		c.Key = key
	}

	// Generate a key ID
	b := make([]byte, 20)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(err)
	}
	keyID := hex.EncodeToString(b)

	return &mockSigner{
		key: &jose.JSONWebKey{
			Key:       c.Key,
			KeyID:     keyID,
			Algorithm: "RS256",
			Use:       "sig",
		},
		pubKey: &jose.JSONWebKey{
			Key:       c.Key.Public(),
			KeyID:     keyID,
			Algorithm: "RS256",
			Use:       "sig",
		},
	}, nil
}

// mockSigner is a simple signer that uses a static RSA key for testing.
type mockSigner struct {
	key    *jose.JSONWebKey
	pubKey *jose.JSONWebKey
}

func (m *mockSigner) Sign(_ context.Context, payload []byte) (string, error) {
	return signPayload(m.key, jose.RS256, payload)
}

func (m *mockSigner) ValidationKeys(_ context.Context) ([]*jose.JSONWebKey, error) {
	return []*jose.JSONWebKey{m.pubKey}, nil
}

func (m *mockSigner) Algorithm(_ context.Context) (jose.SignatureAlgorithm, error) {
	return jose.RS256, nil
}

func (m *mockSigner) Start(_ context.Context) {
	// Nothing to do for mock signer
}

// NewMockSigner creates a mock signer with the provided key for testing.
// If key is nil, a new one will be generated.
func NewMockSigner(key *rsa.PrivateKey) (Signer, error) {
	return (&MockConfig{Key: key}).Open(context.Background())
}
