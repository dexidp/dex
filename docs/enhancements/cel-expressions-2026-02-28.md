# Dex Enhancement Proposal (DEP) - 2026-02-28 - CEL (Common Expression Language) Integration

## Table of Contents

- [Summary](#summary)
- [Context](#context)
- [Motivation](#motivation)
    - [Goals/Pain](#goalspain)
    - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [User Experience](#user-experience)
    - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
        - [Phase 1: pkg/cel - Core CEL Library](#phase-1-pkgcel---core-cel-library)
        - [Phase 2: Authentication Policies](#phase-2-authentication-policies)
        - [Phase 3: Token Policies](#phase-3-token-policies)
        - [Phase 4: OIDC Connector Claim Mapping](#phase-4-oidc-connector-claim-mapping)
    - [Risks and Mitigations](#risks-and-mitigations)
    - [Alternatives](#alternatives)
- [Future Improvements](#future-improvements)

## Summary

This DEP proposes integrating [CEL (Common Expression Language)][cel-spec] into Dex as a first-class
expression engine for policy evaluation, claim mapping, and token customization. A new reusable
`pkg/cel` package will provide a safe, sandboxed CEL environment with Kubernetes-grade compatibility
guarantees, cost budgets, and a curated set of extension libraries. Subsequent phases will leverage
this package to implement authentication policies, token policies, advanced claim mapping in
connectors, and per-client/global access rules — replacing the need for ad-hoc configuration fields
and external policy engines.

[cel-spec]: https://github.com/google/cel-spec

## Context

- [#1583 Add allowedGroups option for clients config][#1583] — a long-standing request for a
  configuration option to allow a client to specify a list of allowed groups.
- [#1635 Connector Middleware][#1635] — long-standing request for a policy/middleware layer between
  connectors and the server for claim transformations and access control.
- [#1052 Allow restricting connectors per client][#1052] — frequently requested feature to restrict
  which connectors are available to specific OAuth2 clients.
- [#2178 Custom claims in ID tokens][#2178] — requests for including additional payload in issued tokens.
- [#2812 Token Exchange DEP][dep-token-exchange] — mentions CEL/Rego as future improvement for
  policy-based assertions on exchanged tokens.
- The OIDC connector already has a growing set of ad-hoc claim mutation options
  (`ClaimMapping`, `ClaimMutations.NewGroupFromClaims`, `FilterGroupClaims`, `ModifyGroupNames`)
  that would benefit from a unified expression language.
- Previous community discussions explored OPA/Rego and JMESPath, but CEL offers a better fit
  (see [Alternatives](#alternatives)).

[#1583]: https://github.com/dexidp/dex/pull/1583
[#1635]: https://github.com/dexidp/dex/issues/1635
[#1052]: https://github.com/dexidp/dex/issues/1052
[#2178]: https://github.com/dexidp/dex/issues/2178
[dep-token-exchange]: /docs/enhancements/token-exchange-2023-02-03-#2812.md

## Motivation

### Goals/Pain

1. **Complex query/filter capabilities** — Dex needs a way to express complex validations and
   mutations in multiple places (authentication flow, token issuance, claim mapping). Today each
   feature requires new Go code, new config fields, and a new release cycle. CEL allows operators
   to express these rules declaratively without code changes.

2. **Authentication policies** — Operators want to control _who_ can log in based on rich
   conditions: restrict specific connectors to specific clients, require group membership for
   certain clients, deny login based on email domain, enforce MFA claims, etc. Currently there is
   no unified mechanism; users rely on downstream applications or external proxies.

3. **Token policies** — Operators want to customize issued tokens: add extra claims to ID tokens,
   restrict scopes per client, modify `aud` claims, include upstream connector metadata, etc.
   Today this requires forking Dex or using a reverse proxy.

4. **Claim mapping in OIDC connector** — The OIDC connector has accumulated multiple ad-hoc config
   options for claim mapping and group mutations (`ClaimMapping`, `NewGroupFromClaims`,
   `FilterGroupClaims`, `ModifyGroupNames`). A single CEL expression field would replace all of
   these with a more powerful and composable approach.

5. **Per-client and global policies** — One of the most frequent requests is allowing different
   connectors for different clients and restricting group-based access per client. CEL policies at
   the global and per-client level address this cleanly.

6. **CNCF ecosystem alignment** — CEL has massive adoption across the CNCF ecosystem:

   | Project                   | CEL Usage                                                                                                                                                  | Evidence |
   |---------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|
   | **Kubernetes**            | ValidatingAdmissionPolicy, CRD validation rules (`x-kubernetes-validations`), AuthorizationPolicy, field selectors, CEL-based match conditions in webhooks | [KEP-3488][k8s-cel-kep], [CRD Validation Rules][k8s-crd-cel], [AuthorizationPolicy KEP-3221][k8s-authz-cel] |
   | **Kyverno**               | CEL expressions in validation/mutation policies (v1.12+), preconditions                                                                                    | [Kyverno CEL docs][kyverno-cel] |
   | **OPA Gatekeeper**        | Partially added support for CEL in constraint templates                                                                                                    | [Gatekeeper CEL][gatekeeper-cel] |
   | **Istio**                 | AuthorizationPolicy conditions, request routing, telemetry                                                                                                 | [Istio CEL docs][istio-cel] |
   | **Envoy / Envoy Gateway** | RBAC filter, ext_authz, rate limiting, route matching, access logging                                                                                      | [Envoy CEL docs][envoy-cel] |
   | **Tekton**                | Pipeline when expressions, CEL custom tasks                                                                                                                | [Tekton CEL Interceptor][tekton-cel] |
   | **Knative**               | Trigger filters using CEL expressions                                                                                                                      | [Knative CEL filters][knative-cel] |
   | **Google Cloud**          | IAM Conditions, Cloud Deploy, Security Command Center                                                                                                      | [Google IAM CEL][gcp-cel] |
   | **Cert-Manager**          | CertificateRequestPolicy approval using CEL                                                                                                                | [cert-manager approver-policy CEL][cert-manager-cel] |
   | **Cilium**                | Hubble CEL filter logic                                                                                                                                    | [Cilium CEL docs][cilium-cel] |
   | **Crossplane**            | Composition functions with CEL-based patch transforms                                                                                                      | [Crossplane CEL transforms][crossplane-cel] |
   | **Kube-OVN**              | Network policy extensions using CEL                                                                                                                        | [Kube-OVN CEL][kube-ovn-cel] |

   [k8s-cel-kep]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/3488-cel-admission-control
   [k8s-crd-cel]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation-rules
   [k8s-authz-cel]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-auth/3221-structured-authorization-configuration
   [kyverno-cel]: https://kyverno.io/docs/writing-policies/cel/
   [gatekeeper-cel]: https://open-policy-agent.github.io/gatekeeper/website/docs/validating-admission-policy/#policy-updates-to-add-vap-cel
   [istio-cel]: https://istio.io/latest/docs/reference/config/security/conditions/
   [envoy-cel]: https://www.envoyproxy.io/docs/envoy/latest/xds/type/v3/cel.proto
   [tekton-cel]: https://tekton.dev/docs/triggers/cel_expressions/
   [knative-cel]: https://github.com/knative/eventing/blob/main/docs/broker/filtering.md#add-cel-expression-filter
   [gcp-cel]: https://cloud.google.com/iam/docs/conditions-overview
   [cert-manager-cel]: https://cert-manager.io/docs/policy/approval/approver-policy/#validations
   [cilium-cel]: https://docs.cilium.io/en/stable/_api/v1/flow/README/#flowfilter-experimental
   [crossplane-cel]: https://github.com/crossplane-contrib/function-cel-filter
   [kube-ovn-cel]: https://kubeovn.github.io/docs/stable/en/advance/cel-expression/

   By choosing CEL, Dex operators who already use Kubernetes or other CNCF tools can reuse their
   existing knowledge of the expression language.

### Non-Goals

- **Full policy engine** — This DEP does not aim to replace dedicated external policy engines
  (OPA, Kyverno). CEL in Dex is scoped to identity and token operations.
- **Breaking changes to existing configuration** — All existing config fields (`ClaimMapping`,
  `ClaimMutations`, etc.) will continue to work. CEL expressions are additive/opt-in.
- **Authorization (beyond Dex scope)** — Dex is an identity provider; downstream authorization
  decisions remain the responsibility of relying parties. CEL policies in Dex are limited to
  authentication and token issuance concerns.
- **Multi-phase CEL in a single DEP** — Only Phase 1 (`pkg/cel` package) is targeted for
  immediate implementation. Phases 2-4 are included here for design context and will have their
  own implementation PRs.

## Proposal

### User Experience

#### Authentication Policy (Phase 2)

Operators can define global and per-client authentication policies in the Dex config:

```yaml
# Global authentication policy
authPolicy:
  rules:
    - deny:
        # Deny login if user email is not from allowed domain
        expression: "!identity.email.endsWith('@example.com')"
        message: "'Login restricted to example.com domain'"
    - deny:
        expression: "!identity.email_verified"
        message: "'Email must be verified'"

staticClients:
  - id: admin-app
    name: Admin Application
    secret: ...
    redirectURIs: [...]
    # Per-client policy
    authPolicy:
      rules:
        # Only allow specific connectors for this client
        - deny:
            expression: "!(request.connector_id in ['okta', 'ldap'])"
            message: "'This application requires Okta or LDAP login'"
        # Require admin group
        - deny:
            expression: "!('admin' in identity.groups)"
            message: "'Admin group membership required'"
```

#### Token Policy (Phase 3)

Operators can add extra claims or mutate token contents:

```yaml
tokenPolicy:
  # Global mutations applied to all ID tokens
  claims:
    # Add a custom claim based on group membership
    - key: "'role'"
      value: "identity.groups.exists(g, g == 'admin') ? 'admin' : 'user'"
    # Include connector ID as a claim
    - key: "'idp'"
      value: "request.connector_id"
    # Add department from upstream claims
    - key: "'department'"
      value: "identity.extra['department'].orValue('unknown')"

staticClients:
  - id: internal-api
    name: Internal API
    secret: ...
    redirectURIs: [...]
    tokenPolicy:
      claims:
        - key: "'custom-claim.company.com/team'"
          value: "identity.extra['team'].orValue('engineering')"
      # Restrict scopes
      filter:
        expression: "request.scopes.all(s, s in ['openid', 'email', 'profile'])"
        message: "'Unsupported scope requested'"
```

#### OIDC Connector Claim Mapping (Phase 4)

Replace ad-hoc claim mapping with CEL:

```yaml
connectors:
  - type: oidc
    id: corporate-idp
    name: Corporate IdP
    config:
      issuer: https://idp.example.com
      clientID: dex-client
      clientSecret: ...
      # CEL-based claim mapping — replaces claimMapping and claimModifications
      claimMappingExpressions:
        username: "claims.preferred_username.orValue(claims.email)"
        email: "claims.email"
        groups: >
          claims.groups
            .filter(g, g.startsWith('dex:'))
            .map(g, g.trimPrefix('dex:'))
        emailVerified: "claims.email_verified.orValue(true)"
        # Extra claims to pass through to token policies
        extra:
          department: "claims.department.orValue('unknown')"
          cost_center: "claims.cost_center.orValue('')"
```

### Implementation Details/Notes/Constraints

### Phase 1: `pkg/cel` — Core CEL Library

This is the foundation that all subsequent phases build upon. The package provides a safe,
reusable CEL environment with Kubernetes-grade guarantees.

#### Package Structure

```
pkg/
  cel/
    cel.go              # Core Environment, compilation, evaluation
    cel_test.go         # Tests
    library.go          # Custom Dex CEL function library
    library_test.go     # Library tests
    types.go            # CEL type declarations (Identity, Request, etc.)
    cost.go             # Cost estimation and budgeting
    cost_test.go        # Cost estimation tests
    doc.go              # Package documentation
```

#### Dependencies

```
github.com/google/cel-go v0.24+
```

The `cel-go` library is the canonical Go implementation maintained by Google, used by Kubernetes
and all major CNCF projects. It follows semantic versioning and provides strong backward
compatibility guarantees.

#### Core API Design

```go
package cel

import (
    "context"
    "fmt"

    "github.com/google/cel-go/cel"
    "github.com/google/cel-go/checker"
    "github.com/google/cel-go/common/types/ref"
    "github.com/google/cel-go/ext"
)

// CompilationResult holds a compiled CEL program ready for evaluation.
type CompilationResult struct {
    Program    cel.Program
    OutputType *cel.Type
    Expression string
}

// Compiler compiles and caches CEL expressions against a specific environment.
type Compiler struct {
    env *cel.Env
}

// CompilerOption configures a Compiler.
type CompilerOption func(*compilerConfig)

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

    envOpts := []cel.EnvOption{
        cel.DefaultUTCTimeZone(true),

        // Standard extension libraries (same set as Kubernetes)
        ext.Strings(),
        ext.Encoders(),
        ext.Lists(),
        ext.Sets(),
        ext.Math(),

        // Custom Dex library
        cel.Lib(&dexLib{}),

        // Cost limit
        cel.CostEstimatorOptions(checker.CostEstimatorOptions{
            SizeEstimateOptions: []checker.SizeEstimateOption{
                checker.PresenceTestHasCost(false),
            },
        }),
    }

    // Register declared variables
    for _, v := range variables {
        envOpts = append(envOpts, cel.Variable(v.Name, v.Type))
    }

    env, err := cel.NewEnv(envOpts...)
    if err != nil {
        return nil, fmt.Errorf("failed to create CEL environment: %w", err)
    }

    return &Compiler{env: env}, nil
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

    prog, err := c.env.Program(ast,
        cel.EvalOptions(cel.OptOptimize),
        cel.CostLimit(cfg.costBudget),
    )
    if err != nil {
        return nil, fmt.Errorf("CEL program creation failed: %w", err)
    }

    return &CompilationResult{
        Program:    prog,
        OutputType: ast.OutputType(),
        Expression: expression,
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
```

#### Variable Declarations

```go
package cel

// VariableDeclaration declares a named variable and its CEL type
// that will be available in expressions.
type VariableDeclaration struct {
    Name string
    Type *cel.Type
}

// IdentityVariables provides the 'identity' variable with user claims.
//
//   identity.user_id        — string
//   identity.username       — string
//   identity.email          — string
//   identity.email_verified — bool
//   identity.groups         — list(string)
//   identity.extra          — map(string, dyn)
func IdentityVariables() []VariableDeclaration {
    return []VariableDeclaration{
        {Name: "identity", Type: cel.MapType(cel.StringType, cel.DynType)},
    }
}

// RequestVariables provides the 'request' variable with request context.
//
//   request.client_id     — string
//   request.connector_id  — string
//   request.scopes        — list(string)
//   request.redirect_uri  — string
func RequestVariables() []VariableDeclaration {
    return []VariableDeclaration{
        {Name: "request", Type: cel.MapType(cel.StringType, cel.DynType)},
    }
}

// ClaimsVariable provides a 'claims' map for raw upstream claims.
//
//   claims — map(string, dyn)
func ClaimsVariable() []VariableDeclaration {
    return []VariableDeclaration{
        {Name: "claims", Type: cel.MapType(cel.StringType, cel.DynType)},
    }
}
```

#### Compatibility Guarantees

Following the Kubernetes CEL compatibility model
([KEP-3488: CEL for Admission Control][kep-3488], [Kubernetes CEL Migration Guide][k8s-cel-compat]):

1. **Environment versioning** — The CEL environment is versioned. When new functions or variables
   are added, they are introduced under a new environment version. Existing expressions compiled
   against an older version continue to work.

   ```go
   // EnvironmentVersion represents the version of the CEL environment.
   // New variables, functions, or libraries are introduced in new versions.
   type EnvironmentVersion uint32

   const (
       // EnvironmentV1 is the initial CEL environment.
       EnvironmentV1 EnvironmentVersion = 1
   )

   // WithVersion sets the target environment version for the compiler.
   // Defaults to the latest version. Specifying an older version ensures
   // that only functions/types available at that version are used.
   func WithVersion(v EnvironmentVersion) CompilerOption {
       return func(cfg *compilerConfig) {
           cfg.version = v
       }
   }
   ```

   This is directly modeled on how Kubernetes versions CEL environments in
   `k8s.io/apiserver/pkg/cel/environment` — each Kubernetes version introduces a new
   environment version that may include new CEL libraries, while older expressions compiled
   against an older version remain valid.

2. **Library stability** — Custom functions added via `library.go` follow these rules:
   - Functions MUST NOT be removed once released.
   - Function signatures MUST NOT change once released.
   - New functions MUST be added under a new `EnvironmentVersion`.
   - If a function needs to be replaced, the old one is deprecated but kept forever.

3. **Type stability** — CEL types (`Identity`, `Request`, `Claims`) follow the same rules:
   - Fields MUST NOT be removed.
   - Field types MUST NOT change.
   - New fields are added in a new `EnvironmentVersion`.

4. **Semantic versioning of `cel-go`** — The `cel-go` dependency follows semver. Dex pins to a
   minor version range and updates are tested for behavioral changes. This is exactly the approach
   Kubernetes takes: `k8s.io/apiextensions-apiserver` pins `cel-go` and gates new features behind
   environment versions.

5. **Feature gates** — New CEL-powered features are gated behind Dex feature flags (using the
   existing `pkg/featureflags` mechanism) during their alpha phase.

[kep-3488]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/3488-cel-admission-control
[k8s-cel-compat]: https://kubernetes.io/docs/reference/using-api/cel/

#### Cost Estimation and Budgets

Like Kubernetes, Dex CEL expressions must be bounded to prevent denial-of-service:

```go
package cel

// DefaultCostBudget is the default cost budget for a single expression evaluation.
// Aligned with Kubernetes defaults: enough for typical identity operations
// but prevents runaway expressions.
const DefaultCostBudget = 10_000_000

// MaxExpressionLength is the maximum length of a CEL expression string.
const MaxExpressionLength = 10_240

// EstimateCost returns the estimated cost range for a compiled expression.
// This is computed statically at compile time without evaluating the expression.
func EstimateCost(result *CompilationResult) (min, max uint64) {
    // Uses cel-go's built-in cost estimator (checker.CostEstimator).
    // ...
}

// WithCostBudget sets a custom cost budget for expression evaluation.
func WithCostBudget(budget uint64) CompilerOption {
    return func(cfg *compilerConfig) {
        cfg.costBudget = budget
    }
}
```

Kubernetes uses `checker.CostEstimator` at admission time to reject CRDs with validation rules
that exceed cost limits. Dex will similarly validate expressions at config load time:
- Reject expressions that exceed `MaxExpressionLength`.
- Estimate cost at compile time and warn if estimated max cost exceeds `DefaultCostBudget`.
- Enforce runtime cost budget during evaluation and abort expressions that exceed the budget.

#### Extension Libraries

The `pkg/cel` environment includes these cel-go standard extensions (same set as Kubernetes):

| Library | Description | Examples |
|---------|-------------|---------|
| `ext.Strings()` | Extended string functions | `"hello".upperAscii()`, `"foo:bar".split(':')`, `s.trim()`, `s.replace('a','b')` |
| `ext.Encoders()` | Base64 encoding/decoding | `base64.encode(bytes)`, `base64.decode(str)` |
| `ext.Lists()` | Extended list functions | `list.slice(1, 3)`, `list.flatten()` |
| `ext.Sets()` | Set operations on lists | `sets.contains(a, b)`, `sets.intersects(a, b)`, `sets.equivalent(a, b)` |
| `ext.Math()` | Math functions | `math.greatest(a, b)`, `math.least(a, b)` |

Plus a custom `dex` library with identity-specific helpers:

```go
package cel

// dexLib is the custom Dex CEL function library.
// All functions here are subject to the compatibility guarantees above.
type dexLib struct{}

// CompileOptions returns the CEL environment options for the Dex library.
func (dexLib) CompileOptions() []cel.EnvOption {
    return []cel.EnvOption{
        cel.Function("dex.emailDomain",
            cel.Overload("dex_email_domain_string",
                []*cel.Type{cel.StringType},
                cel.StringType,
                cel.UnaryBinding(emailDomainImpl),
            ),
        ),
        cel.Function("dex.emailLocalPart",
            cel.Overload("dex_email_local_part_string",
                []*cel.Type{cel.StringType},
                cel.StringType,
                cel.UnaryBinding(emailLocalPartImpl),
            ),
        ),
        cel.Function("dex.groupMatches",
            cel.Overload("dex_group_matches_list_string",
                []*cel.Type{cel.ListType(cel.StringType), cel.StringType},
                cel.ListType(cel.StringType),
                cel.BinaryBinding(groupMatchesImpl),
            ),
        ),
    }
}

// ProgramOptions returns the CEL program options for the Dex library.
func (dexLib) ProgramOptions() []cel.ProgramOption {
    return nil
}

// Functions provided by dexLib (V1):
//
//   dex.emailDomain(email: string) -> string
//     Returns the domain portion of an email address.
//     Example: dex.emailDomain("user@example.com") == "example.com"
//
//   dex.emailLocalPart(email: string) -> string
//     Returns the local part of an email address.
//     Example: dex.emailLocalPart("user@example.com") == "user"
//
//   dex.groupMatches(groups: list(string), pattern: string) -> list(string)
//     Returns groups matching a glob pattern.
//     Example: dex.groupMatches(identity.groups, "team:*")
```

#### Activation Data Mapping

Internal Go types are mapped to CEL variables before evaluation:

```go
package cel

import "github.com/dexidp/dex/connector"

// IdentityFromConnector converts a connector.Identity to a CEL-compatible map.
func IdentityFromConnector(id connector.Identity) map[string]any {
    return map[string]any{
        "user_id":            id.UserID,
        "username":           id.Username,
        "preferred_username": id.PreferredUsername,
        "email":              id.Email,
        "email_verified":     id.EmailVerified,
        "groups":             id.Groups,
    }
}

// RequestContext represents the authentication/token request context
// available as the 'request' variable in CEL expressions.
type RequestContext struct {
    ClientID    string
    ConnectorID string
    Scopes      []string
    RedirectURI string
}

// RequestFromContext converts a RequestContext to a CEL-compatible map.
func RequestFromContext(rc RequestContext) map[string]any {
    return map[string]any{
        "client_id":    rc.ClientID,
        "connector_id": rc.ConnectorID,
        "scopes":       rc.Scopes,
        "redirect_uri": rc.RedirectURI,
    }
}
```

#### Example: Compile and Evaluate

```go
package main

import (
    "context"
    "fmt"

    "github.com/dexidp/dex/connector"
    dexcel "github.com/dexidp/dex/pkg/cel"
)

func main() {
    // Create a compiler with identity and request variables
    compiler, err := dexcel.NewCompiler(
        append(dexcel.IdentityVariables(), dexcel.RequestVariables()...),
    )
    if err != nil {
        panic(err)
    }

    // Compile a policy expression
    prog, err := compiler.CompileBool(
        `identity.email.endsWith('@example.com') && 'admin' in identity.groups`,
    )
    if err != nil {
        panic(err)
    }

    // Evaluate against real data
    result, err := dexcel.EvalBool(context.Background(), prog, map[string]any{
        "identity": dexcel.IdentityFromConnector(connector.Identity{
            UserID:   "123",
            Username: "john",
            Email:    "john@example.com",
            Groups:   []string{"admin", "dev"},
        }),
        "request": dexcel.RequestFromContext(dexcel.RequestContext{
            ClientID:    "my-app",
            ConnectorID: "okta",
            Scopes:      []string{"openid", "email"},
        }),
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(result) // true
}
```

### Phase 2: Authentication Policies

**Config Model:**

```go
// AuthPolicy defines authentication policies evaluated after a user
// successfully authenticates with a connector.
type AuthPolicy struct {
    Rules []AuthPolicyRule `json:"rules"`
}

type AuthPolicyRule struct {
    // Deny defines a condition that, when true, denies the authentication.
    Deny *PolicyExpression `json:"deny,omitempty"`
}

// PolicyExpression is a CEL expression with an optional human-readable message.
type PolicyExpression struct {
    // Expression is a CEL expression that evaluates to bool.
    Expression string `json:"expression"`
    // Message is a CEL expression that evaluates to string (displayed to the user on deny).
    // If empty, a generic message is shown.
    Message string `json:"message,omitempty"`
}
```

**Evaluation point:** After `connector.CallbackConnector.HandleCallback()` or
`connector.PasswordConnector.Login()` returns an identity, and before the auth request is
finalized. Implemented in `server/handlers.go` at `handleConnectorCallback`.

**Available CEL variables:** `identity` (from connector), `request` (client_id, connector_id,
scopes, redirect_uri).

**Compilation:** All policy expressions are compiled once at config load time (in
`cmd/dex/serve.go`) and stored in the `Server` struct. This ensures:
- Syntax/type errors are caught at startup, not at runtime.
- No compilation overhead per request.
- Cost estimation can warn operators about expensive expressions at startup.

**Evaluation flow:**

```
User authenticates via connector
         |
         v
connector.HandleCallback() returns Identity
         |
         v
Evaluate global authPolicy rules (in order)
  - For each rule with deny: evaluate expression
  - If expression returns true → deny with message, HTTP 403
         |
         v
Look up per-client authPolicy for request.client_id
Evaluate per-client authPolicy rules (in order)
  - Same deny logic as global
         |
         v
Continue normal flow (approval screen or redirect)
```

### Phase 3: Token Policies

**Config Model:**

```go
// TokenPolicy defines policies for token issuance.
type TokenPolicy struct {
    // Claims adds or overrides claims in the issued ID token.
    Claims []ClaimExpression `json:"claims,omitempty"`
    // Filter validates the token request. If expression evaluates to false,
    // the request is denied.
    Filter *PolicyExpression `json:"filter,omitempty"`
}

type ClaimExpression struct {
    // Key is a CEL expression evaluating to string — the claim name.
    Key string `json:"key"`
    // Value is a CEL expression evaluating to dyn — the claim value.
    Value string `json:"value"`
}
```

**Evaluation point:** In `server/oauth2.go` during ID token construction, after standard
claims are built but before JWT signing.

**Available CEL variables:** `identity`, `request`, `existing_claims` (the standard claims already
computed as `map(string, dyn)`).

**Claim merge order:**
1. Standard Dex claims (sub, iss, aud, email, groups, etc.)
2. Global `tokenPolicy.claims` evaluated and merged
3. Per-client `tokenPolicy.claims` evaluated and merged (overrides global)

**Reserved (forbidden) claim names:**

Certain claim names are reserved and MUST NOT be set or overridden by CEL token policy
expressions. Attempting to use a reserved claim key will result in a config validation error at
startup. This prevents operators from accidentally breaking the OIDC/OAuth2 contract or
undermining Dex's security guarantees.

```go
// ReservedClaimNames is the set of claim names that CEL token policy
// expressions are forbidden from setting. These are core OIDC/OAuth2 claims
// managed exclusively by Dex.
var ReservedClaimNames = map[string]struct{}{
    "iss":       {},  // Issuer — always set by Dex to its own issuer URL
    "sub":       {},  // Subject — derived from connector identity, must not be spoofed
    "aud":       {},  // Audience — determined by the OAuth2 client, not policy
    "exp":       {},  // Expiration — controlled by Dex token TTL configuration
    "iat":       {},  // Issued At — set by Dex at signing time
    "nbf":       {},  // Not Before — set by Dex at signing time
    "jti":       {},  // JWT ID — generated by Dex for token revocation/uniqueness
    "auth_time": {},  // Authentication Time — set by Dex from the auth session
    "nonce":     {},  // Nonce — echoed from the client's authorization request
    "at_hash":   {},  // Access Token Hash — computed by Dex from the access token
    "c_hash":    {},  // Code Hash — computed by Dex from the authorization code
}
```

The reserved list is enforced in two places:
1. **Config load time** — When compiling token policy `ClaimExpression` entries, Dex statically
   evaluates the `Key` expression (which must be a string literal or constant-foldable) and rejects
   it if the result is in `ReservedClaimNames`.
2. **Runtime (defense in depth)** — Before merging evaluated claims into the ID token, Dex checks
   each key against `ReservedClaimNames` and logs a warning + skips the claim if it matches. This
   guards against dynamic key expressions that couldn't be statically checked.

### Phase 4: OIDC Connector Claim Mapping

**Config Model:**

In `connector/oidc/oidc.go`:

```go
type Config struct {
    // ... existing fields ...

    // ClaimMappingExpressions provides CEL-based claim mapping.
    // When set, these take precedence over ClaimMapping and ClaimMutations.
    ClaimMappingExpressions *ClaimMappingExpression `json:"claimMappingExpressions,omitempty"`
}

type ClaimMappingExpression struct {
    // Username is a CEL expression evaluating to string.
    // Available variable: 'claims' (map of upstream claims).
    Username string `json:"username,omitempty"`
    // Email is a CEL expression evaluating to string.
    Email string `json:"email,omitempty"`
    // Groups is a CEL expression evaluating to list(string).
    Groups string `json:"groups,omitempty"`
    // EmailVerified is a CEL expression evaluating to bool.
    EmailVerified string `json:"emailVerified,omitempty"`
    // Extra is a map of claim names to CEL expressions evaluating to dyn.
    // These are carried through to token policies.
    Extra map[string]string `json:"extra,omitempty"`
}
```

**Available CEL variable:** `claims` — `map(string, dyn)` containing all raw upstream claims from
the ID token and/or UserInfo endpoint.

This replaces the need for `ClaimMapping`, `NewGroupFromClaims`, `FilterGroupClaims`, and
`ModifyGroupNames` with a single, more powerful mechanism.

**Backward compatibility:** When `claimMappingExpressions` is nil, the existing `ClaimMapping` and
`ClaimMutations` logic is used unchanged. When `claimMappingExpressions` is set, a startup warning is
logged if legacy mapping fields are also configured.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| **CEL expression complexity / DoS** | Cost budgets with configurable limits (default aligned with Kubernetes). Expressions are validated at config load time. Runtime evaluation is aborted if cost exceeds budget. |
| **Learning curve for operators** | CEL has excellent documentation, playground ([cel.dev](https://cel.dev)), and massive CNCF adoption. Dex docs will include a dedicated CEL guide with examples. Most operators already know CEL from Kubernetes. |
| **`cel-go` dependency size** | `cel-go` adds ~5MB to binary. This is acceptable for the functionality provided. Kubernetes, Istio, Envoy all accept this trade-off. |
| **Breaking changes in `cel-go`** | Pin to semver minor range. Environment versioning ensures existing expressions continue to work across upgrades. |
| **Security: CEL expression injection** | CEL expressions are defined by operators in the server config, not by end users. No CEL expression is ever constructed from user input at runtime. |
| **Config migration** | Old config fields (`ClaimMapping`, `ClaimMutations`) continue to work. CEL expressions are opt-in. If both are specified, CEL takes precedence with a config-time warning. |
| **Error messages exposing internals** | CEL deny `message` expressions are controlled by the operator. Default messages are generic. Evaluation errors are logged server-side, not exposed to end users. |
| **Performance** | Expressions are compiled once at startup. Evaluation is sub-millisecond for typical identity operations. Cost budgets prevent pathological cases. Benchmarks will be included in `pkg/cel` tests. |

### Alternatives

#### OPA/Rego

OPA was previously considered ([#1635], token exchange DEP). While powerful, it has significant
drawbacks for Dex:

- **Separate daemon** — OPA typically runs as a sidecar or daemon; adds operational complexity.
  Even the embedded Go library (`github.com/open-policy-agent/opa/rego`) is significantly
  heavier than `cel-go`.
- **Rego learning curve** — Rego is a Datalog-derived language unfamiliar to most developers.
  CEL syntax is closer to C/Java/Go and is immediately readable.
- **Overkill** — Dex needs simple expression evaluation, not a full policy engine with data
  loading, bundles, and partial evaluation.
- **No inline expressions** — Rego policies are typically separate files, not inline config
  expressions. This makes the config harder to understand and deploy.
- **Smaller CNCF footprint for embedding** — While OPA is a graduated CNCF project, CEL has
  broader adoption as an _embedded_ language (Kubernetes, Istio, Envoy, Kyverno, etc.).

#### JMESPath

JMESPath was proposed for claim mapping. Drawbacks:

- **Query-only** — JMESPath is a JSON query language. It cannot express boolean conditions,
  mutations, or string operations naturally.
- **Limited type system** — No type checking at compile time. Errors are only caught at runtime.
- **Small ecosystem** — Limited adoption compared to CEL. No CNCF projects use JMESPath for
  policy evaluation.
- **No cost estimation** — No way to bound execution time.

#### Hardcoded Go Logic

The current approach: each feature requires new Go structs, config fields, and code. This is
unsustainable:
- `ClaimMapping`, `NewGroupFromClaims`, `FilterGroupClaims`, `ModifyGroupNames` are each separate
  features that could be one CEL expression.
- Every new policy need requires a Dex code change and release.
- Combinatorial explosion of config options.

#### No Change

Without CEL or an equivalent:
- Operators continue to request per-client connector restrictions, custom claims, claim
  transformations, and access policies — issues remain open indefinitely.
- Dex accumulates more ad-hoc config fields, increasing maintenance burden.
- Complex use cases require external reverse proxies, forking Dex, or middleware.

## Future Improvements

- **CEL in other connectors** — Extend CEL claim mapping beyond OIDC to LDAP (attribute mapping),
  SAML (assertion mapping), and other connectors with complex attribute mapping needs.
- **Policy testing framework** — Unit test framework for operators to validate their CEL
  expressions against fixture data before deployment.
- **Connector selection via CEL** — Replace the static connector-per-client mapping with a CEL
  expression that dynamically determines which connectors to show based on request attributes.


