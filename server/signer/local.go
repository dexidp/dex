package signer

import (
	"context"
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
}

// Open creates a new local signer.
func (c *LocalConfig) Open(_ context.Context, s storage.Storage, idTokenValidFor time.Duration, now func() time.Time, logger *slog.Logger) (Signer, error) {
	rotateKeysAfter, err := time.ParseDuration(c.KeysRotationPeriod)
	if err != nil {
		return nil, fmt.Errorf("invalid config value %q for local signer rotation period: %v", c.KeysRotationPeriod, err)
	}

	strategy := defaultRotationStrategy(rotateKeysAfter, idTokenValidFor)
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
		if err == errAlreadyRotated {
			l.logger.Info("key rotation not needed", "err", err)
		} else {
			l.logger.Error("failed to rotate keys", "err", err)
		}
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second * 30):
				if err := l.rotator.rotate(); err != nil {
					l.logger.Error("failed to rotate keys", "err", err)
				}
			}
		}
	}()
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
	if err != nil {
		return "", fmt.Errorf("failed to get keys: %v", err)
	}
	if keys.SigningKey == nil {
		return "", fmt.Errorf("no signing key found")
	}
	return signatureAlgorithm(keys.SigningKey)
}
