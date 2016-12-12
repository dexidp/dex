package server

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"golang.org/x/net/context"
	"gopkg.in/square/go-jose.v2"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/dex/storage"
)

// rotationStrategy describes a strategy for generating cryptographic keys, how
// often to rotate them, and how long they can validate signatures after rotation.
type rotationStrategy struct {
	// Time between rotations.
	rotationFrequency time.Duration

	// After being rotated how long can a key validate signatues?
	verifyFor time.Duration

	// Keys are always RSA keys. Though cryptopasta recommends ECDSA keys, not every
	// client may support these (e.g. github.com/coreos/go-oidc/oidc).
	key func() (*rsa.PrivateKey, error)
}

// staticRotationStrategy returns a strategy which never rotates keys.
func staticRotationStrategy(key *rsa.PrivateKey) rotationStrategy {
	return rotationStrategy{
		// Setting these values to 100 years is easier than having a flag indicating no rotation.
		rotationFrequency: time.Hour * 8760 * 100,
		verifyFor:         time.Hour * 8760 * 100,
		key:               func() (*rsa.PrivateKey, error) { return key, nil },
	}
}

// defaultRotationStrategy returns a strategy which rotates keys every provided period,
// holding onto the public parts for some specified amount of time.
func defaultRotationStrategy(rotationFrequency, verifyFor time.Duration) rotationStrategy {
	return rotationStrategy{
		rotationFrequency: rotationFrequency,
		verifyFor:         verifyFor,
		key: func() (*rsa.PrivateKey, error) {
			return rsa.GenerateKey(rand.Reader, 2048)
		},
	}
}

type keyRotater struct {
	storage.Storage

	strategy rotationStrategy
	now      func() time.Time

	logger logrus.FieldLogger
}

// startKeyRotation begins key rotation in a new goroutine, closing once the context is canceled.
//
// The method blocks until after the first attempt to rotate keys has completed. That way
// healthy storages will return from this call with valid keys.
func (s *Server) startKeyRotation(ctx context.Context, strategy rotationStrategy, now func() time.Time) {
	rotater := keyRotater{s.storage, strategy, now, s.logger}

	// Try to rotate immediately so properly configured storages will have keys.
	if err := rotater.rotate(); err != nil {
		s.logger.Errorf("failed to rotate keys: %v", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second * 30):
				if err := rotater.rotate(); err != nil {
					s.logger.Errorf("failed to rotate keys: %v", err)
				}
			}
		}
	}()
	return
}

func (k keyRotater) rotate() error {
	keys, err := k.GetKeys()
	if err != nil && err != storage.ErrNotFound {
		return fmt.Errorf("get keys: %v", err)
	}
	if k.now().Before(keys.NextRotation) {
		return nil
	}
	k.logger.Infof("keys expired, rotating")

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
	err = k.Storage.UpdateKeys(func(keys storage.Keys) (storage.Keys, error) {
		tNow := k.now()
		if tNow.Before(keys.NextRotation) {
			return storage.Keys{}, errors.New("keys already rotated")
		}

		// Remove expired verification keys.
		i := 0
		for _, key := range keys.VerificationKeys {
			if !key.Expiry.After(tNow) {
				keys.VerificationKeys[i] = key
				i++
			}
		}
		keys.VerificationKeys = keys.VerificationKeys[:i]

		if keys.SigningKeyPub != nil {
			// Move current signing key to a verification only key.
			verificationKey := storage.VerificationKey{
				PublicKey: keys.SigningKeyPub,
				Expiry:    tNow.Add(k.strategy.verifyFor),
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
	k.logger.Infof("keys rotated, next rotation: %s", nextRotation)
	return nil
}
