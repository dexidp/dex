package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/signer"
)

// startKeyRotation begins key rotation in a new goroutine, closing once the context is canceled.
//
// The method blocks until after the first attempt to rotate keys has completed. That way
// healthy storages will return from this call with valid keys.
func (s *Server) startKeyRotation(ctx context.Context) {
	// Try to rotate immediately so properly configured storages will have keys.
	if err := s.signer.RotateKey(); err != nil {
		//nolint:gocritic
		if errors.Is(err, signer.ErrRotationNotSupported) {
			return
		} else if errors.Is(err, signer.ErrAlreadyRotated) {
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
				if err := s.signer.RotateKey(); err != nil {
					s.logger.Errorf("failed to rotate keys: %v", err)
				}
			}
		}
	}()
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
