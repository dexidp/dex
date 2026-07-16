package tokens

import "time"

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
