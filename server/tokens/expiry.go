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

// ExpiryPolicy resolves the effective token lifetimes for a connector: the
// global values, unless an override installed with Upsert says otherwise. The
// issuer, the refresh grant, introspection and the gRPC API all resolve
// through it, so an override installed through the gRPC API is immediately
// visible to all of them.
//
// Overrides are installed at startup from storage and kept current by the
// gRPC API on connector writes. A change written by another replica or
// out-of-band (for example a Connector custom resource applied directly)
// becomes visible here at the next restart.
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
// ceilings an override may not loosen, and override refresh strategies run on
// the global strategy's clock, so both age tokens identically.
func NewExpiryPolicy(idTokensValidFor time.Duration, refresh *RefreshStrategy) *ExpiryPolicy {
	c := expiryCeilings{idTokens: idTokensValidFor}
	now := time.Now
	if refresh != nil {
		c.refreshAbsoluteLifetime = refresh.absoluteLifetime
		c.refreshValidIfNotUsedFor = refresh.validIfNotUsedFor
		c.refreshReuseInterval = refresh.reuseInterval
		c.refreshRotationDisabled = !refresh.rotate
		now = refresh.now
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
	_, err := e.build(ce)
	return err
}

// Upsert validates the given storage.ConnectorExpiry and, on success, updates
// the in-memory override map; nil clears the connector's override. Every code
// path that can change a connector's expiry must go through this method so the
// live token-issuance path reflects the change.
func (e *ExpiryPolicy) Upsert(connID string, ce *storage.ConnectorExpiry) error {
	override, err := e.build(ce)
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

// build validates a per-connector override against the ceilings and parses it
// into a connectorExpiryOverride, with fields left unset inheriting from the
// global strategy so the resulting RefreshStrategy carries the correct
// effective values.
func (e *ExpiryPolicy) build(ce *storage.ConnectorExpiry) (connectorExpiryOverride, error) {
	var override connectorExpiryOverride
	if ce == nil {
		return override, nil
	}

	// idTokens="" means "inherit", resolved by IDTokensValidFor.
	d, err := checkCeiling("expiry.idTokens", ce.IDTokens, e.ceilings.idTokens, false)
	if err != nil {
		return override, err
	}
	override.IDTokensValidFor = d

	rt := ce.RefreshTokens
	if rt == nil {
		return override, nil
	}

	rotate := true
	var absolute, valid, reuse time.Duration
	if g := e.refreshStrategy; g != nil {
		rotate, absolute, valid, reuse = g.rotate, g.absoluteLifetime, g.validIfNotUsedFor, g.reuseInterval
	}
	if rt.DisableRotation != nil {
		if *rt.DisableRotation && !e.ceilings.refreshRotationDisabled {
			return override, errors.New("expiry.refreshTokens.disableRotation cannot disable rotation when it is enabled globally")
		}
		rotate = !*rt.DisableRotation
	}
	for _, f := range []struct {
		name         string
		value        string
		ceiling      time.Duration
		zeroDisables bool // RefreshStrategy treats 0 as "expiration disabled" for this field
		dst          *time.Duration
	}{
		{"expiry.refreshTokens.absoluteLifetime", rt.AbsoluteLifetime, e.ceilings.refreshAbsoluteLifetime, true, &absolute},
		{"expiry.refreshTokens.validIfNotUsedFor", rt.ValidIfNotUsedFor, e.ceilings.refreshValidIfNotUsedFor, true, &valid},
		{"expiry.refreshTokens.reuseInterval", rt.ReuseInterval, e.ceilings.refreshReuseInterval, false, &reuse},
	} {
		d, err := checkCeiling(f.name, f.value, f.ceiling, f.zeroDisables)
		if err != nil {
			return override, err
		}
		if f.value != "" {
			*f.dst = d
		}
	}
	override.RefreshStrategy = NewRefreshStrategy(rotate, absolute, valid, reuse, e.now)
	return override, nil
}

// checkCeiling parses a per-connector duration and enforces that it is at
// least as strict as the global ceiling, returning 0 for an unset value. When
// zeroDisables is true, an override of 0 is rejected in the presence of a
// positive ceiling because RefreshStrategy treats 0 as "no expiration" for
// that field — strictly looser than any positive global.
func checkCeiling(field, value string, ceiling time.Duration, zeroDisables bool) (time.Duration, error) {
	if value == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %v", field, err)
	}
	if d < 0 {
		return 0, fmt.Errorf("%s must not be negative, got %v", field, d)
	}
	if ceiling <= 0 {
		return d, nil
	}
	if d > ceiling {
		return 0, fmt.Errorf("%s (%s) exceeds the global value (%s)", field, d, ceiling)
	}
	if zeroDisables && d == 0 {
		return 0, fmt.Errorf("%s cannot be 0 (disables expiration) when the global value (%s) is set", field, ceiling)
	}
	return d, nil
}
