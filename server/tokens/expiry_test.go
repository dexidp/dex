package tokens

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
)

func TestValidateConnectorExpiry(t *testing.T) {
	disableRotation := true
	enableRotation := false
	tests := []struct {
		name            string
		expiry          *storage.ConnectorExpiry
		ceilings        ExpiryCeilings
		wantErrContains string
	}{
		{name: "nil expiry"},
		{
			name:     "idTokens within ceiling",
			expiry:   &storage.ConnectorExpiry{IDTokens: "10m"},
			ceilings: ExpiryCeilings{IDTokens: time.Hour},
		},
		{
			name:            "idTokens exceeds ceiling",
			expiry:          &storage.ConnectorExpiry{IDTokens: "48h"},
			ceilings:        ExpiryCeilings{IDTokens: 24 * time.Hour},
			wantErrContains: "expiry.idTokens (48h0m0s) exceeds the global value",
		},
		{
			name:   "global unset means no ceiling",
			expiry: &storage.ConnectorExpiry{IDTokens: "48h"},
		},
		{
			name:            "invalid duration rejected even without ceiling",
			expiry:          &storage.ConnectorExpiry{RefreshTokens: &storage.ConnectorRefreshExpiry{AbsoluteLifetime: "not-a-duration"}},
			wantErrContains: "parse expiry.refreshTokens.absoluteLifetime",
		},
		{
			name:            "refresh absoluteLifetime exceeds ceiling",
			expiry:          &storage.ConnectorExpiry{RefreshTokens: &storage.ConnectorRefreshExpiry{AbsoluteLifetime: "100h"}},
			ceilings:        ExpiryCeilings{RefreshAbsoluteLifetime: 24 * time.Hour},
			wantErrContains: "expiry.refreshTokens.absoluteLifetime (100h0m0s) exceeds the global value",
		},
		{
			name:            "refresh absoluteLifetime of zero disables and is rejected",
			expiry:          &storage.ConnectorExpiry{RefreshTokens: &storage.ConnectorRefreshExpiry{AbsoluteLifetime: "0s"}},
			ceilings:        ExpiryCeilings{RefreshAbsoluteLifetime: 24 * time.Hour},
			wantErrContains: "expiry.refreshTokens.absoluteLifetime cannot be 0",
		},
		{
			name:            "refresh validIfNotUsedFor of zero disables and is rejected",
			expiry:          &storage.ConnectorExpiry{RefreshTokens: &storage.ConnectorRefreshExpiry{ValidIfNotUsedFor: "0s"}},
			ceilings:        ExpiryCeilings{RefreshValidIfNotUsedFor: time.Hour},
			wantErrContains: "expiry.refreshTokens.validIfNotUsedFor cannot be 0",
		},
		{
			name:     "refresh reuseInterval of zero is stricter, accepted",
			expiry:   &storage.ConnectorExpiry{RefreshTokens: &storage.ConnectorRefreshExpiry{ReuseInterval: "0s"}},
			ceilings: ExpiryCeilings{RefreshReuseInterval: 3 * time.Second},
		},
		{
			name:            "disableRotation cannot loosen global",
			expiry:          &storage.ConnectorExpiry{RefreshTokens: &storage.ConnectorRefreshExpiry{DisableRotation: &disableRotation}},
			wantErrContains: "disableRotation cannot disable rotation when it is enabled globally",
		},
		{
			name:     "enabling rotation when globally disabled is a tightening",
			expiry:   &storage.ConnectorExpiry{RefreshTokens: &storage.ConnectorRefreshExpiry{DisableRotation: &enableRotation}},
			ceilings: ExpiryCeilings{RefreshRotationDisabled: true},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateConnectorExpiry(tc.expiry, tc.ceilings)
			if tc.wantErrContains == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErrContains)
		})
	}
}

func TestBuildConnectorExpiryOverride(t *testing.T) {
	disableRotation := true
	tests := []struct {
		name           string
		expiry         *storage.ConnectorExpiry
		defaults       RefreshTokenDefaults
		wantIDTokens   time.Duration
		wantStrategy   bool
		wantRotationOn bool
	}{
		{name: "nil expiry yields zero override"},
		{
			name:         "idTokens only",
			expiry:       &storage.ConnectorExpiry{IDTokens: "5m"},
			wantIDTokens: 5 * time.Minute,
		},
		{
			name: "refresh override inherits unset fields from defaults",
			expiry: &storage.ConnectorExpiry{RefreshTokens: &storage.ConnectorRefreshExpiry{
				DisableRotation:  &disableRotation,
				AbsoluteLifetime: "1h",
			}},
			defaults: RefreshTokenDefaults{
				ValidIfNotUsedFor: "30m", AbsoluteLifetime: "100h", ReuseInterval: "3s",
			},
			wantStrategy:   true,
			wantRotationOn: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildConnectorExpiryOverride(tc.expiry, tc.defaults, nil)
			require.NoError(t, err)
			require.Equal(t, tc.wantIDTokens, got.IDTokensValidFor)
			if !tc.wantStrategy {
				require.Nil(t, got.RefreshStrategy)
				return
			}
			require.NotNil(t, got.RefreshStrategy)
			require.Equal(t, tc.wantRotationOn, got.RefreshStrategy.RotationEnabled())
		})
	}
}

func TestExpiryIDTokensValidFor(t *testing.T) {
	e := NewExpiry(time.Hour, nil, ExpiryCeilings{}, RefreshTokenDefaults{}, nil)
	require.NoError(t, e.Upsert("shortlived", &storage.ConnectorExpiry{IDTokens: "5m"}))
	require.NoError(t, e.Upsert("refreshonly", &storage.ConnectorExpiry{
		RefreshTokens: &storage.ConnectorRefreshExpiry{AbsoluteLifetime: "1h"},
	}))

	assert.Equal(t, 5*time.Minute, e.IDTokensValidFor("shortlived"),
		"per-connector override should win")
	assert.Equal(t, time.Hour, e.IDTokensValidFor("refreshonly"),
		"refresh-only override should fall back to global for ID tokens")
	assert.Equal(t, time.Hour, e.IDTokensValidFor("unknown"),
		"missing entry should fall back to global")
}

func TestExpiryRefreshStrategy(t *testing.T) {
	global := NewRefreshStrategy(true, 0, 0, 0, nil)

	e := NewExpiry(time.Hour, global, ExpiryCeilings{}, RefreshTokenDefaults{}, nil)
	require.NoError(t, e.Upsert("custom", &storage.ConnectorExpiry{
		RefreshTokens: &storage.ConnectorRefreshExpiry{AbsoluteLifetime: "1h"},
	}))
	require.NoError(t, e.Upsert("idonly", &storage.ConnectorExpiry{IDTokens: "1m"}))

	custom := e.RefreshStrategy("custom")
	assert.NotSame(t, global, custom, "per-connector override should win")
	assert.Equal(t, time.Hour, custom.AbsoluteLifetime())
	assert.Same(t, global, e.RefreshStrategy("idonly"),
		"id-token-only override should fall back to global")
	assert.Same(t, global, e.RefreshStrategy("unknown"),
		"missing entry should fall back to global")
}

func TestExpiryUpsert(t *testing.T) {
	e := NewExpiry(time.Hour, nil, ExpiryCeilings{IDTokens: time.Hour}, RefreshTokenDefaults{}, nil)

	// Accept a tighter override.
	require.NoError(t, e.Upsert("c1", &storage.ConnectorExpiry{IDTokens: "5m"}))
	require.Equal(t, 5*time.Minute, e.IDTokensValidFor("c1"))

	// Reject a looser override; map is left untouched.
	err := e.Upsert("c2", &storage.ConnectorExpiry{IDTokens: "48h"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds the global value")
	require.Equal(t, time.Hour, e.IDTokensValidFor("c2"), "rejected override must not be installed")

	// Clearing the override via nil reverts to the global.
	require.NoError(t, e.Upsert("c1", nil))
	require.Equal(t, time.Hour, e.IDTokensValidFor("c1"))
}

func TestExpiryOverrideUsesInjectedClock(t *testing.T) {
	// t0 is far in the future so a strategy running on wall time instead of
	// the injected clock gives the opposite answer.
	t0 := time.Date(2050, 1, 1, 0, 0, 0, 0, time.UTC)
	now := func() time.Time { return t0.Add(2 * time.Minute) }

	e := NewExpiry(time.Hour, nil, ExpiryCeilings{}, RefreshTokenDefaults{}, now)
	require.NoError(t, e.Upsert("c", &storage.ConnectorExpiry{
		RefreshTokens: &storage.ConnectorRefreshExpiry{ValidIfNotUsedFor: "1m"},
	}))

	s := e.RefreshStrategy("c")
	assert.True(t, s.ExpiredBecauseUnused(t0),
		"override strategy must age tokens on the injected clock")
	assert.False(t, s.ExpiredBecauseUnused(t0.Add(90*time.Second)))
}
