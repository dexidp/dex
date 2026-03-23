package library_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dexcel "github.com/dexidp/dex/pkg/cel"
)

func TestGroupMatches(t *testing.T) {
	vars := dexcel.IdentityVariables()
	compiler, err := dexcel.NewCompiler(vars)
	require.NoError(t, err)

	tests := map[string]struct {
		expr   string
		groups []string
		want   []string
	}{
		"wildcard pattern": {
			expr:   `dex.groupMatches(identity.groups, "team:*")`,
			groups: []string{"team:dev", "team:ops", "admin"},
			want:   []string{"team:dev", "team:ops"},
		},
		"exact match": {
			expr:   `dex.groupMatches(identity.groups, "admin")`,
			groups: []string{"team:dev", "admin", "user"},
			want:   []string{"admin"},
		},
		"no matches": {
			expr:   `dex.groupMatches(identity.groups, "nonexistent")`,
			groups: []string{"team:dev", "admin"},
			want:   []string{},
		},
		"question mark pattern": {
			expr:   `dex.groupMatches(identity.groups, "team?")`,
			groups: []string{"teamA", "teamB", "teams-long"},
			want:   []string{"teamA", "teamB"},
		},
		"match all": {
			expr:   `dex.groupMatches(identity.groups, "*")`,
			groups: []string{"a", "b", "c"},
			want:   []string{"a", "b", "c"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			prog, err := compiler.CompileStringList(tc.expr)
			require.NoError(t, err)

			out, err := dexcel.Eval(context.Background(), prog, map[string]any{
				"identity": dexcel.IdentityVal{Groups: tc.groups},
			})
			require.NoError(t, err)

			nativeVal, err := out.ConvertToNative(reflect.TypeOf([]string{}))
			require.NoError(t, err)

			got, ok := nativeVal.([]string)
			require.True(t, ok, "expected []string, got %T", nativeVal)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGroupMatchesInvalidPattern(t *testing.T) {
	vars := dexcel.IdentityVariables()
	compiler, err := dexcel.NewCompiler(vars)
	require.NoError(t, err)

	prog, err := compiler.CompileStringList(`dex.groupMatches(identity.groups, "[invalid")`)
	require.NoError(t, err)

	_, err = dexcel.Eval(context.Background(), prog, map[string]any{
		"identity": dexcel.IdentityVal{Groups: []string{"admin"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid pattern")
}

func TestGroupFilter(t *testing.T) {
	vars := dexcel.IdentityVariables()
	compiler, err := dexcel.NewCompiler(vars)
	require.NoError(t, err)

	tests := map[string]struct {
		expr   string
		groups []string
		want   []string
	}{
		"filter to allowed": {
			expr:   `dex.groupFilter(identity.groups, ["admin", "ops"])`,
			groups: []string{"admin", "dev", "ops"},
			want:   []string{"admin", "ops"},
		},
		"no overlap": {
			expr:   `dex.groupFilter(identity.groups, ["marketing"])`,
			groups: []string{"admin", "dev"},
			want:   []string{},
		},
		"all allowed": {
			expr:   `dex.groupFilter(identity.groups, ["a", "b", "c"])`,
			groups: []string{"a", "b", "c"},
			want:   []string{"a", "b", "c"},
		},
		"empty allowed list": {
			expr:   `dex.groupFilter(identity.groups, [])`,
			groups: []string{"admin", "dev"},
			want:   []string{},
		},
		"preserves order": {
			expr:   `dex.groupFilter(identity.groups, ["z", "a"])`,
			groups: []string{"a", "b", "z"},
			want:   []string{"a", "z"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			prog, err := compiler.CompileStringList(tc.expr)
			require.NoError(t, err)

			out, err := dexcel.Eval(context.Background(), prog, map[string]any{
				"identity": dexcel.IdentityVal{Groups: tc.groups},
			})
			require.NoError(t, err)

			nativeVal, err := out.ConvertToNative(reflect.TypeOf([]string{}))
			require.NoError(t, err)

			got, ok := nativeVal.([]string)
			require.True(t, ok, "expected []string, got %T", nativeVal)
			assert.Equal(t, tc.want, got)
		})
	}
}
