package server

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/dexidp/dex/storage"
)

// ExpiryCeilings holds the parsed global expiry values that per-connector
// overrides must not exceed. A zero duration means "no ceiling" — i.e. the
// global value is unset/disabled, so any override is acceptable.
type ExpiryCeilings struct {
	IDTokens                 time.Duration
	RefreshAbsoluteLifetime  time.Duration
	RefreshValidIfNotUsedFor time.Duration
	RefreshReuseInterval     time.Duration
}

// RefreshTokenDefaults are the global refresh-token configuration strings.
// Per-connector overrides inherit unset fields from these values when
// constructing a RefreshTokenPolicy.
type RefreshTokenDefaults struct {
	DisableRotation   bool
	ValidIfNotUsedFor string
	AbsoluteLifetime  string
	ReuseInterval     string
}

// ValidateConnectorExpiry rejects per-connector overrides that loosen the
// global policy. DisableRotation is exempt: it's a policy toggle, not a
// quantity, so "stricter" has no natural direction.
//
// This function is the single source of truth for the hierarchy rule. It is
// called from both the static YAML load path and every gRPC API write so
// that no configuration modification can ever bypass it.
func ValidateConnectorExpiry(e *storage.ConnectorExpiry, c ExpiryCeilings) error {
	if e == nil {
		return nil
	}
	if err := checkCeiling("expiry.idTokens", e.IDTokens, c.IDTokens); err != nil {
		return err
	}
	if e.RefreshTokens == nil {
		return nil
	}
	for _, f := range []struct {
		name    string
		value   string
		ceiling time.Duration
	}{
		{"expiry.refreshTokens.absoluteLifetime", e.RefreshTokens.AbsoluteLifetime, c.RefreshAbsoluteLifetime},
		{"expiry.refreshTokens.validIfNotUsedFor", e.RefreshTokens.ValidIfNotUsedFor, c.RefreshValidIfNotUsedFor},
		{"expiry.refreshTokens.reuseInterval", e.RefreshTokens.ReuseInterval, c.RefreshReuseInterval},
	} {
		if err := checkCeiling(f.name, f.value, f.ceiling); err != nil {
			return err
		}
	}
	return nil
}

func checkCeiling(field, value string, ceiling time.Duration) error {
	if value == "" || ceiling == 0 {
		return nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("parse %s: %v", field, err)
	}
	if d > ceiling {
		return fmt.Errorf("%s (%s) exceeds the global value (%s)", field, d, ceiling)
	}
	return nil
}

// buildConnectorExpiryOverride parses a (pre-validated) storage.ConnectorExpiry
// into a ConnectorExpiryOverride. Unset string fields inherit from the global
// refresh defaults so the resulting RefreshTokenPolicy carries the correct
// effective values.
func buildConnectorExpiryOverride(
	logger *slog.Logger,
	connectorID string,
	e *storage.ConnectorExpiry,
	defaults RefreshTokenDefaults,
) (ConnectorExpiryOverride, error) {
	var override ConnectorExpiryOverride
	if e == nil {
		return override, nil
	}

	if e.IDTokens != "" {
		d, err := time.ParseDuration(e.IDTokens)
		if err != nil {
			return override, fmt.Errorf("parse expiry.idTokens: %v", err)
		}
		override.IDTokensValidFor = d
	}

	rt := e.RefreshTokens
	if rt == nil {
		return override, nil
	}

	disableRotation := defaults.DisableRotation
	if rt.DisableRotation != nil {
		disableRotation = *rt.DisableRotation
	}
	validIfNotUsedFor := rt.ValidIfNotUsedFor
	if validIfNotUsedFor == "" {
		validIfNotUsedFor = defaults.ValidIfNotUsedFor
	}
	absoluteLifetime := rt.AbsoluteLifetime
	if absoluteLifetime == "" {
		absoluteLifetime = defaults.AbsoluteLifetime
	}
	reuseInterval := rt.ReuseInterval
	if reuseInterval == "" {
		reuseInterval = defaults.ReuseInterval
	}

	policy, err := NewRefreshTokenPolicy(
		logger.With("connector_id", connectorID),
		disableRotation, validIfNotUsedFor, absoluteLifetime, reuseInterval,
	)
	if err != nil {
		return override, fmt.Errorf("refresh token policy: %v", err)
	}
	override.RefreshTokenPolicy = policy
	return override, nil
}
