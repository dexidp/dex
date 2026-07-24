package tokens

import (
	"fmt"
	"log/slog"
	"time"
)

// RefreshStrategy decides how a refresh token ages and rotates. It holds only the
// resolved intervals; parsing them from configuration is the caller's job.
type RefreshStrategy struct {
	rotate bool // whether rotation is enabled

	absoluteLifetime  time.Duration // creation to end of life
	validIfNotUsedFor time.Duration // last use to end of life
	reuseInterval     time.Duration // window in which an old token may still be reused

	now func() time.Time
}

// NewRefreshStrategy builds a RefreshStrategy from resolved durations. now defaults to
// time.Now when nil.
func NewRefreshStrategy(rotate bool, absoluteLifetime, validIfNotUsedFor, reuseInterval time.Duration, now func() time.Time) *RefreshStrategy {
	if now == nil {
		now = time.Now
	}
	return &RefreshStrategy{
		rotate:            rotate,
		absoluteLifetime:  absoluteLifetime,
		validIfNotUsedFor: validIfNotUsedFor,
		reuseInterval:     reuseInterval,
		now:               now,
	}
}

// NewRefreshTokenPolicy parses the refresh-token configuration into a rotation
// strategy — the config-reading adapter over NewRefreshStrategy. now defaults
// to time.Now when nil.
func NewRefreshTokenPolicy(logger *slog.Logger, rotation bool, validIfNotUsedFor, absoluteLifetime, reuseInterval string, now func() time.Time) (*RefreshStrategy, error) {
	var validDur, absoluteDur, reuseDur time.Duration
	var err error

	if validIfNotUsedFor != "" {
		validDur, err = time.ParseDuration(validIfNotUsedFor)
		if err != nil {
			return nil, fmt.Errorf("invalid config value %q for refresh token valid if not used for: %v", validIfNotUsedFor, err)
		}
		logger.Info("config refresh tokens", "valid_if_not_used_for", validIfNotUsedFor)
	}

	if absoluteLifetime != "" {
		absoluteDur, err = time.ParseDuration(absoluteLifetime)
		if err != nil {
			return nil, fmt.Errorf("invalid config value %q for refresh tokens absolute lifetime: %v", absoluteLifetime, err)
		}
		logger.Info("config refresh tokens", "absolute_lifetime", absoluteLifetime)
	}

	if reuseInterval != "" {
		reuseDur, err = time.ParseDuration(reuseInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid config value %q for refresh tokens reuse interval: %v", reuseInterval, err)
		}
		logger.Info("config refresh tokens", "reuse_interval", reuseInterval)
	}

	rotate := !rotation
	logger.Info("config refresh tokens rotation", "enabled", rotate)
	return NewRefreshStrategy(rotate, absoluteDur, validDur, reuseDur, now), nil
}

// RotationEnabled reports whether refresh tokens are rotated on use.
func (s *RefreshStrategy) RotationEnabled() bool {
	return s.rotate
}

// AbsoluteLifetime is the interval from creation to the end of a token's life
// (zero means no absolute expiry).
func (s *RefreshStrategy) AbsoluteLifetime() time.Duration {
	return s.absoluteLifetime
}

// CompletelyExpired reports whether a token created at the given time has passed
// its absolute lifetime.
func (s *RefreshStrategy) CompletelyExpired(lastUsed time.Time) bool {
	if s.absoluteLifetime == 0 {
		return false // expiration disabled
	}
	return s.now().After(lastUsed.Add(s.absoluteLifetime))
}

// ExpiredBecauseUnused reports whether a token has been idle past its inactivity
// window.
func (s *RefreshStrategy) ExpiredBecauseUnused(lastUsed time.Time) bool {
	if s.validIfNotUsedFor == 0 {
		return false // expiration disabled
	}
	return s.now().After(lastUsed.Add(s.validIfNotUsedFor))
}

// AllowedToReuse reports whether a just-rotated token may still be reused within
// the reuse window.
func (s *RefreshStrategy) AllowedToReuse(lastUsed time.Time) bool {
	if s.reuseInterval == 0 {
		return false // reuse disabled
	}
	return !s.now().After(lastUsed.Add(s.reuseInterval))
}
