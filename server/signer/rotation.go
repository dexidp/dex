package signer

import (
	"context"
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

	// Keys are always RSA keys. Though cryptopasta recommends ECDSA keys, not every
	// client may support these (e.g. github.com/coreos/go-oidc/oidc).
	key func() (*rsa.PrivateKey, error)
}

// defaultRotationStrategy returns a strategy which rotates keys every provided period,
// holding onto the public parts for some specified amount of time.
func defaultRotationStrategy(rotationFrequency, idTokenValidFor time.Duration) rotationStrategy {
	return rotationStrategy{
		rotationFrequency: rotationFrequency,
		idTokenValidFor:   idTokenValidFor,
		key: func() (*rsa.PrivateKey, error) {
			return rsa.GenerateKey(rand.Reader, 2048)
		},
	}
}

type keyRotator struct {
	storage.Storage

	strategy rotationStrategy
	now      func() time.Time

	logger *slog.Logger
}

func (k keyRotator) rotate() error {
	keys, err := k.GetKeys(context.Background())
	if err != nil && err != storage.ErrNotFound {
		return fmt.Errorf("get keys: %v", err)
	}
	if k.now().Before(keys.NextRotation) {
		return nil
	}
	k.logger.Info("keys expired, rotating")

	// Generate the key outside of a storage transaction.
	key, err := k.strategy.key()
	if err != nil {
		return fmt.Errorf("generate key: %v", err)
	}
	b := make([]byte, 20)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(err)
	}
	keyID := hex.EncodeToString(b)
	priv := &jose.JSONWebKey{
		Key:       key,
		KeyID:     keyID,
		Algorithm: "RS256",
		Use:       "sig",
	}
	pub := &jose.JSONWebKey{
		Key:       key.Public(),
		KeyID:     keyID,
		Algorithm: "RS256",
		Use:       "sig",
	}

	var nextRotation time.Time
	err = k.Storage.UpdateKeys(context.Background(), func(keys storage.Keys) (storage.Keys, error) {
		tNow := k.now()

		// if you are running multiple instances of dex, another instance
		// could have already rotated the keys.
		if tNow.Before(keys.NextRotation) {
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
	k.logger.Info("keys rotated", "next_rotation", nextRotation)
	return nil
}
