package cel_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dexcel "github.com/dexidp/dex/pkg/cel"
)

func TestEstimateCost(t *testing.T) {
	vars := dexcel.IdentityVariables()
	compiler, err := dexcel.NewCompiler(vars)
	require.NoError(t, err)

	tests := map[string]struct {
		expr string
	}{
		"simple bool": {
			expr: "true",
		},
		"string comparison": {
			expr: "identity.email == 'test@example.com'",
		},
		"group membership": {
			expr: "identity.groups.exists(g, g == 'admin')",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			prog, err := compiler.Compile(tc.expr)
			require.NoError(t, err)

			est, err := compiler.EstimateCost(prog)
			require.NoError(t, err)
			assert.True(t, est.Max >= est.Min, "max cost should be >= min cost")
			assert.True(t, est.Max <= dexcel.DefaultCostBudget,
				"estimated max cost %d should be within default budget %d", est.Max, dexcel.DefaultCostBudget)
		})
	}
}

func TestCompileTimeCostAcceptsSimpleExpressions(t *testing.T) {
	vars := append(dexcel.IdentityVariables(), dexcel.RequestVariables()...)
	compiler, err := dexcel.NewCompiler(vars)
	require.NoError(t, err)

	tests := map[string]string{
		"literal":         "true",
		"email endsWith":  "identity.email.endsWith('@example.com')",
		"group check":     "'admin' in identity.groups",
		"emailDomain":     `dex.emailDomain(identity.email)`,
		"groupMatches":    `dex.groupMatches(identity.groups, "team:*")`,
		"groupFilter":     `dex.groupFilter(identity.groups, ["admin", "dev"])`,
		"combined policy": `identity.email.endsWith('@example.com') && 'admin' in identity.groups`,
		"complex policy": `identity.email.endsWith('@example.com') &&
			identity.groups.exists(g, g == 'admin') &&
			request.connector_id == 'okta' &&
			request.scopes.exists(s, s == 'openid')`,
		"filter+map chain": `identity.groups
			.filter(g, g.startsWith('team:'))
			.map(g, g.replace('team:', ''))
			.size() > 0`,
	}

	for name, expr := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := compiler.Compile(expr)
			assert.NoError(t, err, "expression should compile within default budget")
		})
	}
}

func TestCompileTimeCostRejection(t *testing.T) {
	vars := append(dexcel.IdentityVariables(), dexcel.RequestVariables()...)

	tests := map[string]struct {
		budget uint64
		expr   string
	}{
		"simple exists exceeds tiny budget": {
			budget: 1,
			expr:   "identity.groups.exists(g, g == 'admin')",
		},
		"endsWith exceeds tiny budget": {
			budget: 2,
			expr:   "identity.email.endsWith('@example.com')",
		},
		"nested comprehension over groups exceeds moderate budget": {
			// Two nested iterations over groups: O(n^2) where n=100 → ~280K
			budget: 10_000,
			expr: `identity.groups.exists(g1,
				identity.groups.exists(g2,
					g1 != g2 && g1.startsWith(g2)
				)
			)`,
		},
		"cross-variable comprehension exceeds moderate budget": {
			// filter groups then check each against scopes: O(n*m) → ~162K
			budget: 10_000,
			expr: `identity.groups
				.filter(g, g.startsWith('team:'))
				.exists(g, request.scopes.exists(s, s == g))`,
		},
		"chained filter+map+filter+map exceeds small budget": {
			budget: 1000,
			expr: `identity.groups
				.filter(g, g.startsWith('team:'))
				.map(g, g.replace('team:', ''))
				.filter(g, g.size() > 3)
				.map(g, g.upperAscii())
				.size() > 0`,
		},
		"many independent exists exceeds small budget": {
			budget: 5000,
			expr: `identity.groups.exists(g, g.contains('a')) &&
				identity.groups.exists(g, g.contains('b')) &&
				identity.groups.exists(g, g.contains('c')) &&
				identity.groups.exists(g, g.contains('d')) &&
				identity.groups.exists(g, g.contains('e'))`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			compiler, err := dexcel.NewCompiler(vars, dexcel.WithCostBudget(tc.budget))
			require.NoError(t, err)

			_, err = compiler.Compile(tc.expr)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "estimated cost")
			assert.Contains(t, err.Error(), "exceeds budget")
		})
	}
}
