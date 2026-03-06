package cel_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/connector"
	dexcel "github.com/dexidp/dex/pkg/cel"
)

func TestCompileBool(t *testing.T) {
	compiler, err := dexcel.NewCompiler(nil)
	require.NoError(t, err)

	tests := map[string]struct {
		expr    string
		wantErr bool
	}{
		"true literal": {
			expr: "true",
		},
		"comparison": {
			expr: "1 == 1",
		},
		"string type mismatch": {
			expr:    "'hello'",
			wantErr: true,
		},
		"int type mismatch": {
			expr:    "42",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := compiler.CompileBool(tc.expr)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestCompileString(t *testing.T) {
	compiler, err := dexcel.NewCompiler(nil)
	require.NoError(t, err)

	tests := map[string]struct {
		expr    string
		wantErr bool
	}{
		"string literal": {
			expr: "'hello'",
		},
		"string concatenation": {
			expr: "'hello' + ' ' + 'world'",
		},
		"bool type mismatch": {
			expr:    "true",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := compiler.CompileString(tc.expr)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestCompileStringList(t *testing.T) {
	compiler, err := dexcel.NewCompiler(nil)
	require.NoError(t, err)

	result, err := compiler.CompileStringList("['a', 'b', 'c']")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	_, err = compiler.CompileStringList("'not a list'")
	assert.Error(t, err)
}

func TestCompile(t *testing.T) {
	compiler, err := dexcel.NewCompiler(nil)
	require.NoError(t, err)

	// Compile accepts any type
	result, err := compiler.Compile("true")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = compiler.Compile("'hello'")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = compiler.Compile("42")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCompileErrors(t *testing.T) {
	compiler, err := dexcel.NewCompiler(nil)
	require.NoError(t, err)

	tests := map[string]struct {
		expr string
	}{
		"syntax error": {
			expr: "1 +",
		},
		"undefined variable": {
			expr: "undefined_var",
		},
		"undefined function": {
			expr: "undefinedFunc()",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := compiler.Compile(tc.expr)
			assert.Error(t, err)
		})
	}
}

func TestMaxExpressionLength(t *testing.T) {
	compiler, err := dexcel.NewCompiler(nil)
	require.NoError(t, err)

	longExpr := "'" + strings.Repeat("a", dexcel.MaxExpressionLength) + "'"
	_, err = compiler.Compile(longExpr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum length")
}

func TestEvalBool(t *testing.T) {
	vars := dexcel.IdentityVariables()
	compiler, err := dexcel.NewCompiler(vars)
	require.NoError(t, err)

	tests := map[string]struct {
		expr     string
		identity map[string]any
		want     bool
	}{
		"email endsWith": {
			expr: "identity.email.endsWith('@example.com')",
			identity: map[string]any{
				"email": "user@example.com",
			},
			want: true,
		},
		"email endsWith false": {
			expr: "identity.email.endsWith('@example.com')",
			identity: map[string]any{
				"email": "user@other.com",
			},
			want: false,
		},
		"email_verified": {
			expr: "identity.email_verified == true",
			identity: map[string]any{
				"email_verified": true,
			},
			want: true,
		},
		"group membership": {
			expr: "identity.groups.exists(g, g == 'admin')",
			identity: map[string]any{
				"groups": []string{"admin", "dev"},
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			prog, err := compiler.CompileBool(tc.expr)
			require.NoError(t, err)

			result, err := dexcel.EvalBool(context.Background(), prog, map[string]any{
				"identity": tc.identity,
			})
			require.NoError(t, err)
			assert.Equal(t, tc.want, result)
		})
	}
}

func TestEvalString(t *testing.T) {
	vars := dexcel.IdentityVariables()
	compiler, err := dexcel.NewCompiler(vars)
	require.NoError(t, err)

	// identity.email returns dyn from map access, use Compile (not CompileString)
	prog, err := compiler.Compile("identity.email")
	require.NoError(t, err)

	result, err := dexcel.EvalString(context.Background(), prog, map[string]any{
		"identity": map[string]any{
			"email": "user@example.com",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", result)
}

func TestEvalWithIdentityAndRequest(t *testing.T) {
	vars := append(dexcel.IdentityVariables(), dexcel.RequestVariables()...)
	compiler, err := dexcel.NewCompiler(vars)
	require.NoError(t, err)

	prog, err := compiler.CompileBool(
		`identity.email.endsWith('@example.com') && 'admin' in identity.groups && request.connector_id == 'okta'`,
	)
	require.NoError(t, err)

	identity := dexcel.IdentityFromConnector(connector.Identity{
		UserID:   "123",
		Username: "john",
		Email:    "john@example.com",
		Groups:   []string{"admin", "dev"},
	})
	request := dexcel.RequestFromContext(dexcel.RequestContext{
		ClientID:    "my-app",
		ConnectorID: "okta",
		Scopes:      []string{"openid", "email"},
	})

	result, err := dexcel.EvalBool(context.Background(), prog, map[string]any{
		"identity": identity,
		"request":  request,
	})
	require.NoError(t, err)
	assert.True(t, result)
}

func TestNewCompilerWithVariables(t *testing.T) {
	// Claims variable
	compiler, err := dexcel.NewCompiler(dexcel.ClaimsVariable())
	require.NoError(t, err)

	// claims.email returns dyn from map access, use Compile (not CompileString)
	prog, err := compiler.Compile("claims.email")
	require.NoError(t, err)

	result, err := dexcel.EvalString(context.Background(), prog, map[string]any{
		"claims": map[string]any{
			"email": "test@example.com",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", result)
}
