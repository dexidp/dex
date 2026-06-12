package signer

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"log/slog"
	"sort"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func signingKeyID(t *testing.T, s storage.Storage) string {
	keys, err := s.GetKeys(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	return keys.SigningKey.KeyID
}

func verificationKeyIDs(t *testing.T, s storage.Storage) (ids []string) {
	keys, err := s.GetKeys(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range keys.VerificationKeys {
		ids = append(ids, key.PublicKey.KeyID)
	}
	return ids
}

// slicesEq compare two string slices without modifying the ordering
// of the slices.
func slicesEq(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}

	cp := func(s []string) []string {
		c := make([]string, len(s))
		copy(c, s)
		return c
	}

	cp1 := cp(s1)
	cp2 := cp(s2)
	sort.Strings(cp1)
	sort.Strings(cp2)

	for i, el := range cp1 {
		if el != cp2[i] {
			return false
		}
	}
	return true
}

func TestKeyRotator(t *testing.T) {
	now := time.Now()

	delta := time.Millisecond
	rotationFrequency := time.Second * 5
	validFor := time.Second * 21

	// Only the last 5 verification keys are expected to be kept around.
	maxVerificationKeys := 5

	l := slog.New(slog.DiscardHandler)
	strategy, err := rotationStrategyForAlgorithm(rotationFrequency, validFor, jose.RS256)
	if err != nil {
		t.Fatal(err)
	}

	r := &keyRotator{
		Storage:  memory.New(l),
		strategy: strategy,
		now:      func() time.Time { return now },
		logger:   l,
	}

	var expVerificationKeys []string

	for i := 0; i < 10; i++ {
		now = now.Add(rotationFrequency + delta)
		if err := r.rotate(); err != nil {
			t.Fatal(err)
		}

		got := verificationKeyIDs(t, r.Storage)

		if !slicesEq(expVerificationKeys, got) {
			t.Errorf("after %d rotation, expected verification keys %q, got %q", i+1, expVerificationKeys, got)
		}

		expVerificationKeys = append(expVerificationKeys, signingKeyID(t, r.Storage))
		if n := len(expVerificationKeys); n > maxVerificationKeys {
			expVerificationKeys = expVerificationKeys[n-maxVerificationKeys:]
		}
	}
}

func TestRotationStrategyForAlgorithm(t *testing.T) {
	frequency := time.Hour
	validFor := time.Hour

	t.Run("RS256 generates an RSA key", func(t *testing.T) {
		strategy, err := rotationStrategyForAlgorithm(frequency, validFor, jose.RS256)
		require.NoError(t, err)
		assert.Equal(t, jose.RS256, strategy.algorithm)

		key, err := strategy.key()
		require.NoError(t, err)
		assert.IsType(t, &rsa.PrivateKey{}, key)
	})

	t.Run("ES256 generates an ECDSA key", func(t *testing.T) {
		strategy, err := rotationStrategyForAlgorithm(frequency, validFor, jose.ES256)
		require.NoError(t, err)
		assert.Equal(t, jose.ES256, strategy.algorithm)

		key, err := strategy.key()
		require.NoError(t, err)
		assert.IsType(t, &ecdsa.PrivateKey{}, key)
	})

	t.Run("EdDSA generates an Ed25519 key", func(t *testing.T) {
		strategy, err := rotationStrategyForAlgorithm(frequency, validFor, jose.EdDSA)
		require.NoError(t, err)
		assert.Equal(t, jose.EdDSA, strategy.algorithm)

		key, err := strategy.key()
		require.NoError(t, err)
		assert.IsType(t, ed25519.PrivateKey{}, key)
	})

	t.Run("unsupported algorithm errors", func(t *testing.T) {
		_, err := rotationStrategyForAlgorithm(frequency, validFor, jose.PS256)
		require.Error(t, err)
	})
}
