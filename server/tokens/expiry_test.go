package tokens

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
)

func TestExpiryPolicyValidate(t *testing.T) {
	disableRotation := true
	enableRotation := false
	tests := []struct {
		name            string
		expiry          *storage.ConnectorExpiry
		idTokens        time.Duration
		global          *RefreshStrategy
		wantErrContains string
	}{
		{name: "nil expiry"},
		{
			name:     "idTokens within ceiling",
			expiry:   &storage.ConnectorExpiry{IDTokens: "10m"},
			idTokens: time.Hour,
		},
		{
			name:            "idTokens exceeds ceiling",
			expiry:          &storage.ConnectorExpiry{IDTokens: "48h"},
			idTokens:        24 * time.Hour,
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
			name:            "negative duration rejected even without ceiling",
			expiry:          &storage.ConnectorExpiry{RefreshTokens: &storage.ConnectorRefreshExpiry{AbsoluteLifetime: "-1h"}},
			wantErrContains: "expiry.refreshTokens.absoluteLifetime must not be negative",
		},
		{
			name:            "refresh absoluteLifetime exceeds ceiling",
			expiry:          &storage.ConnectorExpiry{RefreshTokens: &storage.ConnectorRefreshExpiry{AbsoluteLifetime: "100h"}},
			global:          NewRefreshStrategy(true, 24*time.Hour, 0, 0, nil),
			wantErrContains: "expiry.refreshTokens.absoluteLifetime (100h0m0s) exceeds the global value",
		},
		{
			name:            "refresh absoluteLifetime of zero disables and is rejected",
			expiry:          &storage.ConnectorExpiry{RefreshTokens: &storage.ConnectorRefreshExpiry{AbsoluteLifetime: "0s"}},
			global:          NewRefreshStrategy(true, 24*time.Hour, 0, 0, nil),
			wantErrContains: "expiry.refreshTokens.absoluteLifetime cannot be 0",
		},
		{
			name:            "refresh validIfNotUsedFor of zero disables and is rejected",
			expiry:          &storage.ConnectorExpiry{RefreshTokens: &storage.ConnectorRefreshExpiry{ValidIfNotUsedFor: "0s"}},
			global:          NewRefreshStrategy(true, 0, time.Hour, 0, nil),
			wantErrContains: "expiry.refreshTokens.validIfNotUsedFor cannot be 0",
		},
		{
			name:   "refresh reuseInterval of zero is stricter, accepted",
			expiry: &storage.ConnectorExpiry{RefreshTokens: &storage.ConnectorRefreshExpiry{ReuseInterval: "0s"}},
			global: NewRefreshStrategy(true, 0, 0, 3*time.Second, nil),
		},
		{
			name:            "disableRotation cannot loosen global",
			expiry:          &storage.ConnectorExpiry{RefreshTokens: &storage.ConnectorRefreshExpiry{DisableRotation: &disableRotation}},
			wantErrContains: "disableRotation cannot disable rotation when it is enabled globally",
		},
		{
			name:   "enabling rotation when globally disabled is a tightening",
			expiry: &storage.ConnectorExpiry{RefreshTokens: &storage.ConnectorRefreshExpiry{DisableRotation: &enableRotation}},
			global: NewRefreshStrategy(false, 0, 0, 0, nil),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := NewExpiryPolicy(tc.idTokens, tc.global).Validate(tc.expiry)
			if tc.wantErrContains == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErrContains)
		})
	}
}

func TestExpiryIDTokensValidFor(t *testing.T) {
	e := NewExpiryPolicy(time.Hour, nil)
	require.NoError(t, e.Upsert("shortlived", &storage.ConnectorExpiry{IDTokens: "5m"}))
	require.NoError(t, e.Upsert("refreshonly", &storage.ConnectorExpiry{
		RefreshTokens: &storage.ConnectorRefreshExpiry{AbsoluteLifetime: "1h"},
	}))

	require.Equal(t, 5*time.Minute, e.IDTokensValidFor("shortlived"),
		"per-connector override should win")
	require.Equal(t, time.Hour, e.IDTokensValidFor("refreshonly"),
		"refresh-only override should fall back to global for ID tokens")
	require.Equal(t, time.Hour, e.IDTokensValidFor("unknown"),
		"missing entry should fall back to global")
}

func TestExpiryRefreshStrategy(t *testing.T) {
	global := NewRefreshStrategy(true, 0, 30*time.Minute, 3*time.Second, nil)

	e := NewExpiryPolicy(time.Hour, global)
	require.NoError(t, e.Upsert("custom", &storage.ConnectorExpiry{
		RefreshTokens: &storage.ConnectorRefreshExpiry{AbsoluteLifetime: "1h"},
	}))
	require.NoError(t, e.Upsert("idonly", &storage.ConnectorExpiry{IDTokens: "1m"}))

	custom := e.RefreshStrategy("custom")
	require.NotSame(t, global, custom, "per-connector override should win")
	require.Equal(t, time.Hour, custom.AbsoluteLifetime())
	require.Equal(t, 30*time.Minute, custom.validIfNotUsedFor,
		"unset fields should inherit from the global strategy")
	require.Equal(t, 3*time.Second, custom.reuseInterval,
		"unset fields should inherit from the global strategy")
	require.True(t, custom.RotationEnabled(),
		"unset rotation should inherit from the global strategy")
	require.Same(t, global, e.RefreshStrategy("idonly"),
		"id-token-only override should fall back to global")
	require.Same(t, global, e.RefreshStrategy("unknown"),
		"missing entry should fall back to global")
}

func TestExpiryUpsert(t *testing.T) {
	e := NewExpiryPolicy(time.Hour, nil)

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

func TestExpiryOverrideUsesGlobalClock(t *testing.T) {
	// t0 is far in the future so a strategy running on wall time instead of
	// the global strategy's clock gives the opposite answer.
	t0 := time.Date(2050, 1, 1, 0, 0, 0, 0, time.UTC)
	now := func() time.Time { return t0.Add(2 * time.Minute) }

	e := NewExpiryPolicy(time.Hour, NewRefreshStrategy(true, 0, 0, 0, now))
	require.NoError(t, e.Upsert("c", &storage.ConnectorExpiry{
		RefreshTokens: &storage.ConnectorRefreshExpiry{ValidIfNotUsedFor: "1m"},
	}))

	s := e.RefreshStrategy("c")
	require.True(t, s.ExpiredBecauseUnused(t0),
		"override strategy must age tokens on the global strategy's clock")
	require.False(t, s.ExpiredBecauseUnused(t0.Add(90*time.Second)))
}
