package tokens

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dexidp/dex/storage"
)

// expiryCeilings holds the global expiry values that per-connector overrides
// must not loosen. A zero duration field means "no ceiling".
//
// refreshRotationDisabled blocks the asymmetric case where the global enables
// rotation: a per-connector override cannot disable it, since rotation-enabled
// is the stricter policy. The reverse direction is permitted.
type expiryCeilings struct {
	idTokens                 time.Duration
	refreshAbsoluteLifetime  time.Duration
	refreshValidIfNotUsedFor time.Duration
	refreshReuseInterval     time.Duration
	refreshRotationDisabled  bool
}

// connectorExpiryOverride carries per-connector token lifetime overrides.
// A zero or nil field inherits the global value.
type connectorExpiryOverride struct {
	IDTokensValidFor time.Duration
	RefreshStrategy  *RefreshStrategy
}

// ExpiryPolicy resolves the effective token lifetimes for a connector: the global
// values, unless an override installed with Upsert says otherwise. It is
// shared by the issuer, the refresh grant, introspection and the gRPC API, so
// an override written through any of them is immediately live everywhere.
type ExpiryPolicy struct {
	idTokensValidFor time.Duration
	refreshStrategy  *RefreshStrategy
	ceilings         expiryCeilings
	now              func() time.Time

	mu        sync.Mutex
	overrides map[string]connectorExpiryOverride
}

// NewExpiryPolicy returns a registry that resolves to the given global values
// until per-connector overrides are installed. The same global values are the
// ceilings an override may not loosen. now is the clock installed into
// override refresh strategies, defaulting to time.Now when nil; pass the same
// clock the global strategy uses so both age tokens identically.
func NewExpiryPolicy(idTokensValidFor time.Duration, refresh *RefreshStrategy, now func() time.Time) *ExpiryPolicy {
	c := expiryCeilings{idTokens: idTokensValidFor}
	if refresh != nil {
		c.refreshAbsoluteLifetime = refresh.absoluteLifetime
		c.refreshValidIfNotUsedFor = refresh.validIfNotUsedFor
		c.refreshReuseInterval = refresh.reuseInterval
		c.refreshRotationDisabled = !refresh.rotate
	}
	return &ExpiryPolicy{
		idTokensValidFor: idTokensValidFor,
		refreshStrategy:  refresh,
		ceilings:         c,
		now:              now,
		overrides:        make(map[string]connectorExpiryOverride),
	}
}

// IDTokensValidFor returns the lifetime of ID tokens issued through the given
// connector.
func (e *ExpiryPolicy) IDTokensValidFor(connID string) time.Duration {
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
func (e *ExpiryPolicy) RefreshStrategy(connID string) *RefreshStrategy {
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
func (e *ExpiryPolicy) Validate(ce *storage.ConnectorExpiry) error {
	return validateConnectorExpiry(ce, e.ceilings)
}

// Upsert validates the given storage.ConnectorExpiry and, on success, updates
// the in-memory override map; nil clears the connector's override. Every code
// path that can change a connector's expiry must go through this method so the
// live token-issuance path reflects the change.
func (e *ExpiryPolicy) Upsert(connID string, ce *storage.ConnectorExpiry) error {
	if err := validateConnectorExpiry(ce, e.ceilings); err != nil {
		return err
	}
	override, err := buildConnectorExpiryOverride(ce, e.refreshStrategy, e.now)
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

// validateConnectorExpiry rejects per-connector overrides that loosen the
// global policy. Called from the static YAML load path and from every gRPC
// API write.
func validateConnectorExpiry(e *storage.ConnectorExpiry, c expiryCeilings) error {
	if e == nil {
		return nil
	}
	// idTokens="" means "inherit"; IDTokensValidFor falls back to the global.
	if err := checkCeiling("expiry.idTokens", e.IDTokens, c.idTokens, false); err != nil {
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
		{"expiry.refreshTokens.absoluteLifetime", e.RefreshTokens.AbsoluteLifetime, c.refreshAbsoluteLifetime, true},
		{"expiry.refreshTokens.validIfNotUsedFor", e.RefreshTokens.ValidIfNotUsedFor, c.refreshValidIfNotUsedFor, true},
		{"expiry.refreshTokens.reuseInterval", e.RefreshTokens.ReuseInterval, c.refreshReuseInterval, false},
	} {
		if err := checkCeiling(f.name, f.value, f.ceiling, f.zeroDisables); err != nil {
			return err
		}
	}
	if dr := e.RefreshTokens.DisableRotation; dr != nil && *dr && !c.refreshRotationDisabled {
		return errors.New("expiry.refreshTokens.disableRotation cannot disable rotation when it is enabled globally")
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
// into a connectorExpiryOverride. Fields left unset inherit from the global
// strategy so the resulting RefreshStrategy carries the correct effective
// values, and now becomes the strategy's clock.
func buildConnectorExpiryOverride(e *storage.ConnectorExpiry, global *RefreshStrategy, now func() time.Time) (connectorExpiryOverride, error) {
	var override connectorExpiryOverride
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

	rotate := true
	var absolute, valid, reuse time.Duration
	if global != nil {
		rotate, absolute, valid, reuse = global.rotate, global.absoluteLifetime, global.validIfNotUsedFor, global.reuseInterval
	}
	if rt.DisableRotation != nil {
		rotate = !*rt.DisableRotation
	}
	for _, f := range []struct {
		name  string
		value string
		dst   *time.Duration
	}{
		{"expiry.refreshTokens.absoluteLifetime", rt.AbsoluteLifetime, &absolute},
		{"expiry.refreshTokens.validIfNotUsedFor", rt.ValidIfNotUsedFor, &valid},
		{"expiry.refreshTokens.reuseInterval", rt.ReuseInterval, &reuse},
	} {
		if f.value == "" {
			continue
		}
		d, err := time.ParseDuration(f.value)
		if err != nil {
			return override, fmt.Errorf("parse %s: %v", f.name, err)
		}
		*f.dst = d
	}
	override.RefreshStrategy = NewRefreshStrategy(rotate, absolute, valid, reuse, now)
	return override, nil
}
