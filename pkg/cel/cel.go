package cel

import (
	"context"
	"fmt"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/ext"

	"github.com/dexidp/dex/pkg/cel/library"
)

// EnvironmentVersion represents the version of the CEL environment.
// New variables, functions, or libraries are introduced in new versions.
type EnvironmentVersion uint32

const (
	// EnvironmentV1 is the initial CEL environment.
	EnvironmentV1 EnvironmentVersion = 1
)

// CompilationResult holds a compiled CEL program ready for evaluation.
type CompilationResult struct {
	Program    cel.Program
	OutputType *cel.Type
	Expression string

	ast *cel.Ast
}

// CompilerOption configures a Compiler.
type CompilerOption func(*compilerConfig)

type compilerConfig struct {
	costBudget uint64
	version    EnvironmentVersion
}

func defaultCompilerConfig() *compilerConfig {
	return &compilerConfig{
		costBudget: DefaultCostBudget,
		version:    EnvironmentV1,
	}
}

// WithCostBudget sets a custom cost budget for expression evaluation.
func WithCostBudget(budget uint64) CompilerOption {
	return func(cfg *compilerConfig) {
		cfg.costBudget = budget
	}
}

// WithVersion sets the target environment version for the compiler.
// Defaults to the latest version. Specifying an older version ensures
// that only functions/types available at that version are used.
func WithVersion(v EnvironmentVersion) CompilerOption {
	return func(cfg *compilerConfig) {
		cfg.version = v
	}
}

// Compiler compiles CEL expressions against a specific environment.
type Compiler struct {
	env *cel.Env
	cfg *compilerConfig
}

// NewCompiler creates a new CEL compiler with the specified variable
// declarations and options.
//
// All custom Dex libraries are automatically included.
// The environment is configured with cost limits and safe defaults.
func NewCompiler(variables []VariableDeclaration, opts ...CompilerOption) (*Compiler, error) {
	cfg := defaultCompilerConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	envOpts := make([]cel.EnvOption, 0, 8+len(variables))
	envOpts = append(envOpts,
		cel.DefaultUTCTimeZone(true),

		// Standard extension libraries (same set as Kubernetes)
		ext.Strings(),
		ext.Encoders(),
		ext.Lists(),
		ext.Sets(),
		ext.Math(),

		// Native Go types for typed variable access.
		// This gives compile-time field checking: identity.emial → error at config load.
		ext.NativeTypes(
			ext.ParseStructTags(true),
			reflect.TypeOf(IdentityVal{}),
			reflect.TypeOf(RequestVal{}),
		),

		// Custom Dex libraries
		cel.Lib(&library.Email{}),
		cel.Lib(&library.Groups{}),

		// Presence tests like has(field) and 'key' in map are O(1) hash
		// lookups on map(string, dyn) variables, so they should not count
		// toward the cost budget. Without this, expressions with multiple
		// 'in' checks (e.g. "'admin' in identity.groups") would accumulate
		// inflated cost estimates. This matches Kubernetes CEL behavior
		// where presence tests are free for CRD validation rules.
		cel.CostEstimatorOptions(
			checker.PresenceTestHasCost(false),
		),
	)

	for _, v := range variables {
		envOpts = append(envOpts, cel.Variable(v.Name, v.Type))
	}

	env, err := cel.NewEnv(envOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &Compiler{env: env, cfg: cfg}, nil
}

// CompileBool compiles a CEL expression that must evaluate to bool.
func (c *Compiler) CompileBool(expression string) (*CompilationResult, error) {
	return c.compile(expression, cel.BoolType)
}

// CompileString compiles a CEL expression that must evaluate to string.
func (c *Compiler) CompileString(expression string) (*CompilationResult, error) {
	return c.compile(expression, cel.StringType)
}

// CompileStringList compiles a CEL expression that must evaluate to list(string).
func (c *Compiler) CompileStringList(expression string) (*CompilationResult, error) {
	return c.compile(expression, cel.ListType(cel.StringType))
}

// Compile compiles a CEL expression with any output type.
func (c *Compiler) Compile(expression string) (*CompilationResult, error) {
	return c.compile(expression, nil)
}

func (c *Compiler) compile(expression string, expectedType *cel.Type) (*CompilationResult, error) {
	if len(expression) > MaxExpressionLength {
		return nil, fmt.Errorf("expression exceeds maximum length of %d characters", MaxExpressionLength)
	}

	ast, issues := c.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("CEL compilation failed: %w", issues.Err())
	}

	if expectedType != nil && !ast.OutputType().IsEquivalentType(expectedType) {
		return nil, fmt.Errorf(
			"expected expression output type %s, got %s",
			expectedType, ast.OutputType(),
		)
	}

	// Estimate cost at compile time and reject expressions that are too expensive.
	costEst, err := c.env.EstimateCost(ast, &defaultCostEstimator{})
	if err != nil {
		return nil, fmt.Errorf("CEL cost estimation failed: %w", err)
	}

	if costEst.Max > c.cfg.costBudget {
		return nil, fmt.Errorf(
			"CEL expression estimated cost %d exceeds budget %d",
			costEst.Max, c.cfg.costBudget,
		)
	}

	prog, err := c.env.Program(ast,
		cel.EvalOptions(cel.OptOptimize),
		cel.CostLimit(c.cfg.costBudget),
	)
	if err != nil {
		return nil, fmt.Errorf("CEL program creation failed: %w", err)
	}

	return &CompilationResult{
		Program:    prog,
		OutputType: ast.OutputType(),
		Expression: expression,
		ast:        ast,
	}, nil
}

// Eval evaluates a compiled program against the given variables.
func Eval(ctx context.Context, result *CompilationResult, variables map[string]any) (ref.Val, error) {
	out, _, err := result.Program.ContextEval(ctx, variables)
	if err != nil {
		return nil, fmt.Errorf("CEL evaluation failed: %w", err)
	}

	return out, nil
}

// EvalBool is a convenience function that evaluates and asserts bool output.
func EvalBool(ctx context.Context, result *CompilationResult, variables map[string]any) (bool, error) {
	out, err := Eval(ctx, result, variables)
	if err != nil {
		return false, err
	}

	v, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("expected bool result, got %T", out.Value())
	}

	return v, nil
}

// EvalString is a convenience function that evaluates and asserts string output.
func EvalString(ctx context.Context, result *CompilationResult, variables map[string]any) (string, error) {
	out, err := Eval(ctx, result, variables)
	if err != nil {
		return "", err
	}

	v, ok := out.Value().(string)
	if !ok {
		return "", fmt.Errorf("expected string result, got %T", out.Value())
	}

	return v, nil
}
