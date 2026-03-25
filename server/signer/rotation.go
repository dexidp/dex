package signer

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/dexidp/dex/storage"
)

var errAlreadyRotated = errors.New("keys already rotated by another server instance")

// rotationStrategy describes a strategy for generating cryptographic keys, how
// often to rotate them, and how long they can validate signatures after rotation.
type rotationStrategy struct {
	// Time between rotations.
	rotationFrequency time.Duration

	// After being rotated how long should the key be kept around for validating
	// signatures?
	idTokenValidFor time.Duration

	// Algorithm used for newly generated signing keys.
	algorithm jose.SignatureAlgorithm

	// Local signer keys can be RSA or ECDSA depending on the configured algorithm.
	key func() (crypto.Signer, error)
}

// rotationStrategyForAlgorithm builds a key rotation strategy for the provided
// local signer algorithm.
func rotationStrategyForAlgorithm(rotationFrequency, idTokenValidFor time.Duration, algorithm jose.SignatureAlgorithm) (rotationStrategy, error) {
	strategy := rotationStrategy{
		rotationFrequency: rotationFrequency,
		idTokenValidFor:   idTokenValidFor,
		algorithm:         algorithm,
	}
	// Only RS256 and ES256 are supported for local key rotation; all other algorithms are handled by the default case.
	switch algorithm { //nolint:exhaustive
	case jose.RS256:
		strategy.key = func() (crypto.Signer, error) {
			return rsa.GenerateKey(rand.Reader, 2048)
		}
	case jose.ES256:
		strategy.key = func() (crypto.Signer, error) {
			return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		}
	default:
		return rotationStrategy{}, fmt.Errorf("unsupported local signer algorithm %q", algorithm)
	}
	return strategy, nil
}

func newJWKPair(key crypto.Signer, algorithm jose.SignatureAlgorithm) (priv, pub *jose.JSONWebKey, err error) {
	b := make([]byte, 20)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, nil, fmt.Errorf("generate key id: %v", err)
	}
	keyID := hex.EncodeToString(b)

	return &jose.JSONWebKey{
			Key:       key,
			KeyID:     keyID,
			Algorithm: string(algorithm),
			Use:       "sig",
		}, &jose.JSONWebKey{
			Key:       key.Public(),
			KeyID:     keyID,
			Algorithm: string(algorithm),
			Use:       "sig",
		}, nil
}

type keyRotator struct {
	storage.Storage

	strategy rotationStrategy
	now      func() time.Time

	logger *slog.Logger
}

func (k keyRotator) rotationReason(keys storage.Keys, tNow time.Time) string {
	if keys.SigningKey == nil {
		return "missing signing key"
	}

	if tNow.Before(keys.NextRotation) {
		return ""
	}
	return "expired"
}

func (k keyRotator) rotate() error {
	keys, err := k.GetKeys(context.Background())
	if err != nil && err != storage.ErrNotFound {
		return fmt.Errorf("get keys: %v", err)
	}

	reason := k.rotationReason(keys, k.now())
	if reason == "" {
		return nil
	}
	k.logger.Info("rotating signing keys", "reason", reason)

	// Generate the key outside of a storage transaction.
	key, err := k.strategy.key()
	if err != nil {
		return fmt.Errorf("generate key: %v", err)
	}
	priv, pub, err := newJWKPair(key, k.strategy.algorithm)
	if err != nil {
		return fmt.Errorf("generate JWK pair: %v", err)
	}

	var nextRotation time.Time
	err = k.Storage.UpdateKeys(context.Background(), func(keys storage.Keys) (storage.Keys, error) {
		tNow := k.now()
		reason := k.rotationReason(keys, tNow)

		// if you are running multiple instances of dex, another instance
		// could have already rotated the keys.
		if reason == "" {
			return storage.Keys{}, errAlreadyRotated
		}

		expired := func(key storage.VerificationKey) bool {
			return tNow.After(key.Expiry)
		}

		// Remove any verification keys that have expired.
		i := 0
		for _, key := range keys.VerificationKeys {
			if !expired(key) {
				keys.VerificationKeys[i] = key
				i++
			}
		}
		keys.VerificationKeys = keys.VerificationKeys[:i]

		if keys.SigningKeyPub != nil {
			// Move current signing key to a verification only key, throwing
			// away the private part.
			verificationKey := storage.VerificationKey{
				PublicKey: keys.SigningKeyPub,
				// After demoting the signing key, keep the token around for at least
				// the amount of time an ID Token is valid for. This ensures the
				// verification key won't expire until all ID Tokens it's signed
				// expired as well.
				Expiry: tNow.Add(k.strategy.idTokenValidFor),
			}
			keys.VerificationKeys = append(keys.VerificationKeys, verificationKey)
		}

		nextRotation = k.now().Add(k.strategy.rotationFrequency)
		keys.SigningKey = priv
		keys.SigningKeyPub = pub
		keys.NextRotation = nextRotation
		return keys, nil
	})
	if err != nil {
		return err
	}
	k.logger.Info("keys rotated", "reason", reason, "next_rotation", nextRotation)
	return nil
}
