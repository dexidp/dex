package tokens

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dexidp/dex/storage"
)

// ExpiryCeilings holds the parsed global expiry values that per-connector
// overrides must not loosen. A zero duration field means "no ceiling".
//
// RefreshRotationDisabled blocks the asymmetric case where the global enables
// rotation: a per-connector override cannot disable it, since rotation-enabled
// is the stricter policy. The reverse direction is permitted.
type ExpiryCeilings struct {
	IDTokens                 time.Duration
	RefreshAbsoluteLifetime  time.Duration
	RefreshValidIfNotUsedFor time.Duration
	RefreshReuseInterval     time.Duration
	RefreshRotationDisabled  bool
}

// RefreshTokenDefaults are the inheritance roots for per-connector overrides
// that leave fields unset.
type RefreshTokenDefaults struct {
	DisableRotation   bool
	ValidIfNotUsedFor string
	AbsoluteLifetime  string
	ReuseInterval     string
}

// ConnectorExpiryOverride carries per-connector token lifetime overrides.
// A zero or nil field inherits the global value.
type ConnectorExpiryOverride struct {
	IDTokensValidFor time.Duration
	RefreshStrategy  *RefreshStrategy
}

// Expiry resolves the effective token lifetimes for a connector: the global
// values, unless an override installed with Upsert says otherwise. It is
// shared by the issuer, the refresh grant, introspection and the gRPC API, so
// an override written through any of them is immediately live everywhere.
type Expiry struct {
	idTokensValidFor time.Duration
	refreshStrategy  *RefreshStrategy
	ceilings         ExpiryCeilings
	refreshDefaults  RefreshTokenDefaults
	now              func() time.Time

	mu        sync.Mutex
	overrides map[string]ConnectorExpiryOverride
}

// NewExpiry returns a registry that resolves to the given global values until
// per-connector overrides are installed. ceilings bound how loose an override
// may be; defaults seed override fields left unset. now is the clock installed
// into override refresh strategies, defaulting to time.Now when nil; pass the
// same clock the global strategy uses so both age tokens identically.
func NewExpiry(idTokensValidFor time.Duration, refresh *RefreshStrategy, ceilings ExpiryCeilings, defaults RefreshTokenDefaults, now func() time.Time) *Expiry {
	return &Expiry{
		idTokensValidFor: idTokensValidFor,
		refreshStrategy:  refresh,
		ceilings:         ceilings,
		refreshDefaults:  defaults,
		now:              now,
		overrides:        map[string]ConnectorExpiryOverride{},
	}
}

// IDTokensValidFor returns the lifetime of ID tokens issued through the given
// connector.
func (e *Expiry) IDTokensValidFor(connID string) time.Duration {
	e.mu.Lock()
	o := e.overrides[connID]
	e.mu.Unlock()
	if o.IDTokensValidFor != 0 {
		return o.IDTokensValidFor
	}
	return e.idTokensValidFor
}

// RefreshStrategy returns the refresh-token strategy for tokens issued through
// the given connector.
func (e *Expiry) RefreshStrategy(connID string) *RefreshStrategy {
	e.mu.Lock()
	o := e.overrides[connID]
	e.mu.Unlock()
	if o.RefreshStrategy != nil {
		return o.RefreshStrategy
	}
	return e.refreshStrategy
}

// Validate rejects a per-connector override that loosens the global policy,
// without installing it. The gRPC API uses it to fail a write before anything
// is persisted.
func (e *Expiry) Validate(ce *storage.ConnectorExpiry) error {
	return validateConnectorExpiry(ce, e.ceilings)
}

// Upsert validates the given storage.ConnectorExpiry and, on success, updates
// the in-memory override map; nil clears the connector's override. Every code
// path that can change a connector's expiry must go through this method so the
// live token-issuance path reflects the change.
func (e *Expiry) Upsert(connID string, ce *storage.ConnectorExpiry) error {
	if err := validateConnectorExpiry(ce, e.ceilings); err != nil {
		return err
	}
	override, err := buildConnectorExpiryOverride(ce, e.refreshDefaults, e.now)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if ce == nil {
		delete(e.overrides, connID)
		return nil
	}
	e.overrides[connID] = override
	return nil
}

// discardLogger is used when a constructor logs at Info level for global
// startup config but the call is part of a per-connector hot path.
var discardLogger = slog.New(slog.DiscardHandler)

// validateConnectorExpiry rejects per-connector overrides that loosen the
// global policy. Called from the static YAML load path and from every gRPC
// API write.
func validateConnectorExpiry(e *storage.ConnectorExpiry, c ExpiryCeilings) error {
	if e == nil {
		return nil
	}
	// idTokens="" means "inherit"; IDTokensValidFor falls back to the global.
	if err := checkCeiling("expiry.idTokens", e.IDTokens, c.IDTokens, false); err != nil {
		return err
	}
	if e.RefreshTokens == nil {
		return nil
	}
	for _, f := range []struct {
		name         string
		value        string
		ceiling      time.Duration
		zeroDisables bool // RefreshStrategy treats 0 as "expiration disabled" for this field
	}{
		{"expiry.refreshTokens.absoluteLifetime", e.RefreshTokens.AbsoluteLifetime, c.RefreshAbsoluteLifetime, true},
		{"expiry.refreshTokens.validIfNotUsedFor", e.RefreshTokens.ValidIfNotUsedFor, c.RefreshValidIfNotUsedFor, true},
		{"expiry.refreshTokens.reuseInterval", e.RefreshTokens.ReuseInterval, c.RefreshReuseInterval, false},
	} {
		if err := checkCeiling(f.name, f.value, f.ceiling, f.zeroDisables); err != nil {
			return err
		}
	}
	if dr := e.RefreshTokens.DisableRotation; dr != nil && *dr && !c.RefreshRotationDisabled {
		return fmt.Errorf("expiry.refreshTokens.disableRotation cannot disable rotation when it is enabled globally")
	}
	return nil
}

// checkCeiling enforces that a per-connector duration is at least as strict as
// the global ceiling. When zeroDisables is true, an override of 0 is rejected
// in the presence of a positive ceiling because RefreshStrategy treats 0 as
// "no expiration" for that field — strictly looser than any positive global.
func checkCeiling(field, value string, ceiling time.Duration, zeroDisables bool) error {
	if value == "" {
		return nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("parse %s: %v", field, err)
	}
	if ceiling <= 0 {
		return nil
	}
	if d > ceiling {
		return fmt.Errorf("%s (%s) exceeds the global value (%s)", field, d, ceiling)
	}
	if zeroDisables && d == 0 {
		return fmt.Errorf("%s cannot be 0 (disables expiration) when the global value (%s) is set", field, ceiling)
	}
	return nil
}

// buildConnectorExpiryOverride parses a (pre-validated) storage.ConnectorExpiry
// into a ConnectorExpiryOverride. Unset string fields inherit from the global
// refresh defaults so the resulting RefreshStrategy carries the correct
// effective values, and now becomes the strategy's clock.
func buildConnectorExpiryOverride(e *storage.ConnectorExpiry, defaults RefreshTokenDefaults, now func() time.Time) (ConnectorExpiryOverride, error) {
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
	// NewRefreshTokenPolicy emits one Info line per field; useful for the single
	// global policy but would spam logs at N connectors × 4 fields on every API
	// write. Pass a discard logger and let the caller summarize.
	strategy, err := NewRefreshTokenPolicy(
		discardLogger,
		disableRotation,
		defaultTo(rt.ValidIfNotUsedFor, defaults.ValidIfNotUsedFor),
		defaultTo(rt.AbsoluteLifetime, defaults.AbsoluteLifetime),
		defaultTo(rt.ReuseInterval, defaults.ReuseInterval),
		now,
	)
	if err != nil {
		return override, fmt.Errorf("refresh token policy: %v", err)
	}
	override.RefreshStrategy = strategy
	return override, nil
}

func defaultTo(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
