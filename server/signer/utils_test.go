package signer

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignatureAlgorithmFromKey(t *testing.T) {
	t.Run("RSA private key", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		alg, err := signatureAlgorithmFromKey(key)
		require.NoError(t, err)
		assert.Equal(t, jose.RS256, alg)
	})

	t.Run("ECDSA P-256 private key", func(t *testing.T) {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		alg, err := signatureAlgorithmFromKey(key)
		require.NoError(t, err)
		assert.Equal(t, jose.ES256, alg)
	})

	t.Run("Ed25519 private key", func(t *testing.T) {
		_, key, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		alg, err := signatureAlgorithmFromKey(key)
		require.NoError(t, err)
		assert.Equal(t, jose.EdDSA, alg)
	})

	t.Run("unsupported key type", func(t *testing.T) {
		alg, err := signatureAlgorithmFromKey(struct{}{})
		require.Error(t, err)
		assert.Empty(t, alg)
		assert.EqualError(t, err, "unsupported signing key type struct {}")
	})
}
