package signer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/dexidp/dex/storage"
)

// LocalConfig holds configuration for the local signer.
type LocalConfig struct {
	// KeysRotationPeriod defines the duration of time after which the signing keys will be rotated.
	KeysRotationPeriod string `json:"keysRotationPeriod"`
	// Algorithm defines the signing algorithm used for newly generated local keys.
	// Changing it does not replace the current signing key immediately. The new
	// algorithm is applied when Dex generates the next signing key during rotation.
	// Supported values are RS256 and ES256.
	Algorithm jose.SignatureAlgorithm `json:"algorithm"`
}

func (c *LocalConfig) signingAlgorithm() (jose.SignatureAlgorithm, error) {
	if c.Algorithm == "" {
		return jose.RS256, nil
	}
	if c.Algorithm == jose.RS256 || c.Algorithm == jose.ES256 {
		return c.Algorithm, nil
	}
	return "", fmt.Errorf("unsupported local signer algorithm %q", c.Algorithm)
}

// Open creates a new local signer.
func (c *LocalConfig) Open(_ context.Context, s storage.Storage, idTokenValidFor time.Duration, now func() time.Time, logger *slog.Logger) (Signer, error) {
	rotateKeysAfter, err := time.ParseDuration(c.KeysRotationPeriod)
	if err != nil {
		return nil, fmt.Errorf("invalid config value %q for local signer rotation period: %v", c.KeysRotationPeriod, err)
	}

	alg, err := c.signingAlgorithm()
	if err != nil {
		return nil, fmt.Errorf("invalid config value %q for local signer algorithm: %v", c.Algorithm, err)
	}

	strategy, err := rotationStrategyForAlgorithm(rotateKeysAfter, idTokenValidFor, alg)
	if err != nil {
		return nil, err
	}
	r := &keyRotator{s, strategy, now, logger}
	return &localSigner{
		storage: s,
		rotator: r,
		logger:  logger,
	}, nil
}

// localSigner signs payloads using keys stored in the Dex storage.
// It manages key rotation and storage using the existing keyRotator logic.
type localSigner struct {
	storage storage.Storage
	rotator *keyRotator
	logger  *slog.Logger
}

// Start begins key rotation in a new goroutine, closing once the context is canceled.
//
// The method blocks until after the first attempt to rotate keys has completed. That way
// healthy storages will return from this call with valid keys.
func (l *localSigner) Start(ctx context.Context) {
	// Try to rotate immediately so properly configured storages will have keys.
	if err := l.rotator.rotate(); err != nil {
		l.logRotateError(err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second * 30):
				if err := l.rotator.rotate(); err != nil {
					l.logRotateError(err)
				}
			}
		}
	}()
}

func (l *localSigner) logRotateError(err error) {
	if errors.Is(err, errAlreadyRotated) {
		l.logger.Info("key rotation not needed", "err", err)
		return
	}
	l.logger.Error("failed to rotate keys", "err", err)
}

func (l *localSigner) Sign(ctx context.Context, payload []byte) (string, error) {
	keys, err := l.storage.GetKeys(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get keys: %v", err)
	}

	signingKey := keys.SigningKey
	if signingKey == nil {
		return "", fmt.Errorf("no key to sign payload with")
	}
	signingAlg, err := signatureAlgorithm(signingKey)
	if err != nil {
		return "", err
	}

	return signPayload(signingKey, signingAlg, payload)
}

func (l *localSigner) ValidationKeys(ctx context.Context) ([]*jose.JSONWebKey, error) {
	keys, err := l.storage.GetKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %v", err)
	}

	if keys.SigningKeyPub == nil {
		return nil, fmt.Errorf("no public keys found")
	}

	jwks := make([]*jose.JSONWebKey, len(keys.VerificationKeys)+1)
	jwks[0] = keys.SigningKeyPub
	for i, verificationKey := range keys.VerificationKeys {
		jwks[i+1] = verificationKey.PublicKey
	}
	return jwks, nil
}

func (l *localSigner) Algorithm(ctx context.Context) (jose.SignatureAlgorithm, error) {
	keys, err := l.storage.GetKeys(ctx)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return "", fmt.Errorf("failed to get keys: %v", err)
	}
	if keys.SigningKey == nil {
		return l.rotator.strategy.algorithm, nil
	}
	return signatureAlgorithm(keys.SigningKey)
}
