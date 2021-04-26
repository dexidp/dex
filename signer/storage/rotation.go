package storage

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"gopkg.in/square/go-jose.v2"

	"github.com/dexidp/dex/signer"
	"github.com/dexidp/dex/storage"
)

// RotationStrategy describes a strategy for generating cryptographic keys, how
// often to rotate them, and how long they can validate signatures after rotation.
type RotationStrategy struct {
	// Time between rotations.
	rotationFrequency time.Duration

	// After being rotated how long should the key be kept around for validating
	// signatues?
	idTokenValidFor time.Duration

	// Keys are always RSA keys. Though cryptopasta recommends ECDSA keys, not every
	// client may support these (e.g. github.com/coreos/go-oidc/oidc).
	key func() (*rsa.PrivateKey, error)
}

// StaticRotationStrategy returns a strategy which never rotates keys.
func StaticRotationStrategy(key *rsa.PrivateKey) RotationStrategy {
	return RotationStrategy{
		// Setting these values to 100 years is easier than having a flag indicating no rotation.
		rotationFrequency: time.Hour * 8760 * 100,
		idTokenValidFor:   time.Hour * 8760 * 100,
		key:               func() (*rsa.PrivateKey, error) { return key, nil },
	}
}

// DefaultRotationStrategy returns a strategy which rotates keys every provided period,
// holding onto the public parts for some specified amount of time.
func DefaultRotationStrategy(rotationFrequency, idTokenValidFor time.Duration) RotationStrategy {
	return RotationStrategy{
		rotationFrequency: rotationFrequency,
		idTokenValidFor:   idTokenValidFor,
		key: func() (*rsa.PrivateKey, error) {
			return rsa.GenerateKey(rand.Reader, 2048)
		},
	}
}

func (s *Signer) RotateKey() error {
	keys, err := s.storage.GetKeys()
	if err != nil && err != storage.ErrNotFound {
		return fmt.Errorf("get keys: %v", err)
	}
	if s.now().Before(keys.NextRotation) {
		return nil
	}
	s.logger.Infof("keys expired, rotating")

	key, err := s.rotationStrategy.key()
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
	err = s.storage.UpdateKeys(func(keys storage.Keys) (storage.Keys, error) {
		tNow := s.now()

		// if you are running multiple instances of dex, another instance
		// could have already rotated the keys.
		if tNow.Before(keys.NextRotation) {
			return storage.Keys{}, signer.ErrAlreadyRotated
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
				Expiry: tNow.Add(s.rotationStrategy.idTokenValidFor),
			}
			keys.VerificationKeys = append(keys.VerificationKeys, verificationKey)
		}

		nextRotation = s.now().Add(s.rotationStrategy.rotationFrequency)
		keys.SigningKey = priv
		keys.SigningKeyPub = pub
		keys.NextRotation = nextRotation
		return keys, nil
	})
	if err != nil {
		return err
	}
	s.logger.Infof("keys rotated, next rotation: %s", nextRotation)
	return nil
}
