package library_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dexcel "github.com/dexidp/dex/pkg/cel"
)

func TestEmailDomain(t *testing.T) {
	compiler, err := dexcel.NewCompiler(nil)
	require.NoError(t, err)

	tests := map[string]struct {
		expr string
		want string
	}{
		"standard email": {
			expr: `dex.emailDomain("user@example.com")`,
			want: "example.com",
		},
		"subdomain": {
			expr: `dex.emailDomain("admin@sub.domain.org")`,
			want: "sub.domain.org",
		},
		"no at sign": {
			expr: `dex.emailDomain("nodomain")`,
			want: "",
		},
		"empty string": {
			expr: `dex.emailDomain("")`,
			want: "",
		},
		"multiple at signs": {
			expr: `dex.emailDomain("user@name@example.com")`,
			want: "name@example.com",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			prog, err := compiler.CompileString(tc.expr)
			require.NoError(t, err)

			result, err := dexcel.EvalString(context.Background(), prog, map[string]any{})
			require.NoError(t, err)
			assert.Equal(t, tc.want, result)
		})
	}
}

func TestEmailLocalPart(t *testing.T) {
	compiler, err := dexcel.NewCompiler(nil)
	require.NoError(t, err)

	tests := map[string]struct {
		expr string
		want string
	}{
		"standard email": {
			expr: `dex.emailLocalPart("user@example.com")`,
			want: "user",
		},
		"no at sign": {
			expr: `dex.emailLocalPart("justuser")`,
			want: "justuser",
		},
		"empty string": {
			expr: `dex.emailLocalPart("")`,
			want: "",
		},
		"multiple at signs": {
			expr: `dex.emailLocalPart("user@name@example.com")`,
			want: "user",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			prog, err := compiler.CompileString(tc.expr)
			require.NoError(t, err)

			result, err := dexcel.EvalString(context.Background(), prog, map[string]any{})
			require.NoError(t, err)
			assert.Equal(t, tc.want, result)
		})
	}
}

func TestEmailDomainWithIdentityVariable(t *testing.T) {
	vars := dexcel.IdentityVariables()
	compiler, err := dexcel.NewCompiler(vars)
	require.NoError(t, err)

	prog, err := compiler.CompileString(`dex.emailDomain(identity.email)`)
	require.NoError(t, err)

	result, err := dexcel.EvalString(context.Background(), prog, map[string]any{
		"identity": dexcel.IdentityVal{Email: "admin@corp.example.com"},
	})
	require.NoError(t, err)
	assert.Equal(t, "corp.example.com", result)
}
