package grants

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/tokens"
)

func TestAudienceContains(t *testing.T) {
	require.True(t, audienceContains("gateway", "gateway"))
	require.True(t, audienceContains([]any{"other", "gateway"}, "gateway"))
	require.False(t, audienceContains("other", "gateway"))
	require.False(t, audienceContains([]any{"other"}, "gateway"))
	require.False(t, audienceContains(nil, "gateway"))
	require.False(t, audienceContains(42.0, "gateway"))
}

func TestMayActAllows(t *testing.T) {
	actor := map[string]any{"iss": "https://actors", "sub": "workload-1"}
	tests := []struct {
		name   string
		mayAct any
		want   bool
	}{
		{"sub only", map[string]any{"sub": "workload-1"}, true},
		{"sub and iss", map[string]any{"iss": "https://actors", "sub": "workload-1"}, true},
		{"sub mismatch", map[string]any{"sub": "workload-2"}, false},
		{"iss mismatch", map[string]any{"iss": "https://elsewhere", "sub": "workload-1"}, false},
		{"empty sub", map[string]any{"iss": "https://actors"}, false},
		{"not an object", "workload-1", false},
		{"nil", nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, mayActAllows(tc.mayAct, actor))
		})
	}
}

func TestEarliestExpiry(t *testing.T) {
	require.True(t, earliestExpiry(map[string]any{}, map[string]any{"exp": "soon"}).IsZero())
	got := earliestExpiry(map[string]any{"exp": float64(200)}, map[string]any{"exp": float64(100)}, map[string]any{})
	require.Equal(t, time.Unix(100, 0), got)
}

func TestNewDelegationPolicies(t *testing.T) {
	p := newDelegationPolicies([]DelegationPolicy{
		{ClientID: "a", ActorConnectors: []string{"x"}},
		{ClientID: "b", ActorConnectors: []string{"x", "y"}},
	})
	require.Equal(t, map[string]bool{"x": true, "y": true}, p.actorConnectors)
	require.Equal(t, []string{"x", "y"}, p.byClient["b"].ActorConnectors)
	_, ok := p.byClient["c"]
	require.False(t, ok)
}

func TestPriorActor(t *testing.T) {
	got, err := priorActor(nil)
	require.NoError(t, err)
	require.Nil(t, got)

	got, err = priorActor(map[string]any{"iss": "https://b", "sub": "b", "act": map[string]any{"sub": "a"}})
	require.NoError(t, err)
	require.Equal(t, &tokens.ActorClaims{Issuer: "https://b", Subject: "b", Act: &tokens.ActorClaims{Subject: "a"}}, got)

	for _, bad := range []any{"a", map[string]any{"iss": "https://b"}, map[string]any{"sub": "b", "act": map[string]any{}}} {
		_, err := priorActor(bad)
		require.Error(t, err, "%v", bad)
	}
}

func TestDelegatedScopes(t *testing.T) {
	requested := []string{"openid", "tools:read", "admin", "audience:server:client_id:backend"}
	require.Equal(t, requested, delegatedScopes(requested, nil), "no scope claim on the subject: unchanged")
	require.Equal(t, []string{"tools:read", "audience:server:client_id:backend"}, delegatedScopes(requested, "tools:read tools:write"))
	require.Equal(t, []string{"audience:server:client_id:backend"}, delegatedScopes(requested, ""))
}

func TestScopeClaim(t *testing.T) {
	require.Equal(t, "openid tools:read", scopeClaim([]string{"openid", "audience:server:client_id:backend", "tools:read"}))
	require.Empty(t, scopeClaim([]string{"audience:server:client_id:backend"}))
}
