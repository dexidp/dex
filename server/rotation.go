package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/dexidp/dex/pkg/log"
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

// staticRotationStrategy returns a strategy which never rotates keys.
func staticRotationStrategy(key *rsa.PrivateKey) rotationStrategy {
	return rotationStrategy{
		// Setting these values to 100 years is easier than having a flag indicating no rotation.
		rotationFrequency: time.Hour * 8760 * 100,
		idTokenValidFor:   time.Hour * 8760 * 100,
		key:               func() (*rsa.PrivateKey, error) { return key, nil },
	}
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

	logger log.Logger
}

// startKeyRotation begins key rotation in a new goroutine, closing once the context is canceled.
//
// The method blocks until after the first attempt to rotate keys has completed. That way
// healthy storages will return from this call with valid keys.
func (s *Server) startKeyRotation(ctx context.Context, strategy rotationStrategy, now func() time.Time) {
	rotator := keyRotator{s.storage, strategy, now, s.logger}

	// Try to rotate immediately so properly configured storages will have keys.
	if err := rotator.rotate(); err != nil {
		if err == errAlreadyRotated {
			s.logger.Infof("Key rotation not needed: %v", err)
		} else {
			s.logger.Errorf("failed to rotate keys: %v", err)
		}
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second * 30):
				if err := rotator.rotate(); err != nil {
					s.logger.Errorf("failed to rotate keys: %v", err)
				}
			}
		}
	}()
}

func (k keyRotator) rotate() error {
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
	k.logger.Infof("keys rotated, next rotation: %s", nextRotation)
	return nil
}

type RefreshTokenPolicy struct {
	rotateRefreshTokens bool // enable rotation

	absoluteLifetime  time.Duration // interval from token creation to the end of its life
	validIfNotUsedFor time.Duration // interval from last token update to the end of its life
	reuseInterval     time.Duration // interval within which old refresh token is allowed to be reused

	now func() time.Time

	logger log.Logger
}

func NewRefreshTokenPolicy(logger log.Logger, rotation bool, validIfNotUsedFor, absoluteLifetime, reuseInterval string) (*RefreshTokenPolicy, error) {
	r := RefreshTokenPolicy{now: time.Now, logger: logger}
	var err error

	if validIfNotUsedFor != "" {
		r.validIfNotUsedFor, err = time.ParseDuration(validIfNotUsedFor)
		if err != nil {
			return nil, fmt.Errorf("invalid config value %q for refresh token valid if not used for: %v", validIfNotUsedFor, err)
		}
		logger.Infof("config refresh tokens valid if not used for: %v", validIfNotUsedFor)
	}

	if absoluteLifetime != "" {
		r.absoluteLifetime, err = time.ParseDuration(absoluteLifetime)
		if err != nil {
			return nil, fmt.Errorf("invalid config value %q for refresh tokens absolute lifetime: %v", absoluteLifetime, err)
		}
		logger.Infof("config refresh tokens absolute lifetime: %v", absoluteLifetime)
	}

	if reuseInterval != "" {
		r.reuseInterval, err = time.ParseDuration(reuseInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid config value %q for refresh tokens reuse interval: %v", reuseInterval, err)
		}
		logger.Infof("config refresh tokens reuse interval: %v", reuseInterval)
	}

	r.rotateRefreshTokens = !rotation
	logger.Infof("config refresh tokens rotation enabled: %v", r.rotateRefreshTokens)
	return &r, nil
}

func (r *RefreshTokenPolicy) RotationEnabled() bool {
	return r.rotateRefreshTokens
}

func (r *RefreshTokenPolicy) CompletelyExpired(lastUsed time.Time) bool {
	if r.absoluteLifetime == 0 {
		return false // expiration disabled
	}
	return r.now().After(lastUsed.Add(r.absoluteLifetime))
}

func (r *RefreshTokenPolicy) ExpiredBecauseUnused(lastUsed time.Time) bool {
	if r.validIfNotUsedFor == 0 {
		return false // expiration disabled
	}
	return r.now().After(lastUsed.Add(r.validIfNotUsedFor))
}

func (r *RefreshTokenPolicy) AllowedToReuse(lastUsed time.Time) bool {
	if r.reuseInterval == 0 {
		return false // expiration disabled
	}
	return !r.now().After(lastUsed.Add(r.reuseInterval))
}
