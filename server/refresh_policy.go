package server

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/dexidp/dex/server/tokens"
)

// NewRefreshTokenPolicy parses the refresh-token configuration into a rotation
// strategy. It is the config-reading adapter; the strategy object itself lives
// in the tokens package, independent of how its intervals are configured.
func NewRefreshTokenPolicy(logger *slog.Logger, rotation bool, validIfNotUsedFor, absoluteLifetime, reuseInterval string) (*tokens.RefreshStrategy, error) {
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
	return tokens.NewRefreshStrategy(rotate, absoluteDur, validDur, reuseDur, time.Now), nil
}
