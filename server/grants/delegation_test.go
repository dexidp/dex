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
		name         string
		mayAct       any
		actor        map[string]any
		singleIssuer bool
		want         bool
	}{
		{name: "sub and iss", mayAct: map[string]any{"iss": "https://actors", "sub": "workload-1"}, actor: actor, want: true},
		{name: "sub only, single actor connector", mayAct: map[string]any{"sub": "workload-1"}, actor: actor, singleIssuer: true, want: true},
		{name: "sub only, several actor connectors", mayAct: map[string]any{"sub": "workload-1"}, actor: actor},
		{name: "client_id only", mayAct: map[string]any{"client_id": "gateway"}, actor: actor, want: true},
		{name: "sub and client_id", mayAct: map[string]any{"iss": "https://actors", "sub": "workload-1", "client_id": "gateway"}, actor: actor, want: true},
		{name: "informational members ignored", mayAct: map[string]any{"client_id": "gateway", "email": "a@b", "email_verified": true, "name": "A", "preferred_username": "a"}, actor: actor, want: true},
		{name: "sub mismatch", mayAct: map[string]any{"iss": "https://actors", "sub": "workload-2"}, actor: actor},
		{name: "iss mismatch", mayAct: map[string]any{"iss": "https://elsewhere", "sub": "workload-1"}, actor: actor},
		{name: "client_id mismatch", mayAct: map[string]any{"client_id": "other"}, actor: actor},
		{name: "iss without sub", mayAct: map[string]any{"iss": "https://actors", "client_id": "gateway"}, actor: actor},
		{name: "neither sub nor client_id", mayAct: map[string]any{"email": "a@b"}, actor: actor},
		{name: "unknown member", mayAct: map[string]any{"client_id": "gateway", "groups": []any{"x"}}, actor: actor},
		{name: "non-string sub", mayAct: map[string]any{"sub": 42.0}, actor: actor, singleIssuer: true},
		{name: "not an object", mayAct: "workload-1", actor: actor},
		{name: "nil", mayAct: nil, actor: actor},
		{name: "impersonation, client_id", mayAct: map[string]any{"client_id": "gateway"}, want: true},
		{name: "impersonation, other client", mayAct: map[string]any{"client_id": "other"}},
		{name: "impersonation, sub needs an actor token", mayAct: map[string]any{"sub": "workload-1", "client_id": "gateway"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, mayActAllows(tc.mayAct, tc.actor, "gateway", tc.singleIssuer))
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
	requested := []string{"openid", "tools:read", "admin"}
	require.Equal(t, requested, delegatedScopes(requested, nil, false), "first hop: unchanged")
	require.Empty(t, delegatedScopes(requested, nil, true), "delegated subject without scope grants none")
	require.Equal(t, []string{"tools:read"}, delegatedScopes(requested, "tools:read tools:write", true))
	require.Equal(t, []string{"openid", "tools:read"}, delegatedScopes(requested, "openid tools:read", false))
	require.Empty(t, delegatedScopes(requested, "other", true))
}
