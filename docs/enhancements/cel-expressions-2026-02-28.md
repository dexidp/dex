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
    - [Policy Application Flow](#policy-application-flow)
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
[dep-token-exchange]: /docs/enhancements/token-exchange-2023-02-03-%232812.md

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
- **Multi-step logic** — CEL in Dex is scoped to single-expression evaluation. Each expression
  is a standalone, stateless computation with no intermediate variables, chaining, or
  multi-step transformations. If a use case requires sequential logic or conditionally chained
  expressions, it belongs outside Dex (e.g. in an external policy engine or middleware).
  This boundary protects the design from scope creep that pushes CEL beyond what it's good at.

## Proposal

### User Experience

#### Authentication Policy (Phase 2)

Operators can define global and per-client authentication policies in the Dex config:

```yaml
# Global authentication policy — each expression evaluates to bool.
# If true — the request is denied. Evaluated in order; first match wins.
authPolicy:
  - expression: "!identity.email.endsWith('@example.com')"
    message: "'Login restricted to example.com domain'"
  - expression: "!identity.email_verified"
    message: "'Email must be verified'"

staticClients:
  - id: admin-app
    name: Admin Application
    secret: ...
    redirectURIs: [...]
    # Per-client policy — same structure as global
    authPolicy:
      - expression: "!(request.connector_id in ['okta', 'ldap'])"
        message: "'This application requires Okta or LDAP login'"
      - expression: "!('admin' in identity.groups)"
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
    # Add department from upstream claims (only if present)
    - key: "'department'"
      value: "identity.extra['department']"
      condition: "'department' in identity.extra"

staticClients:
  - id: internal-api
    name: Internal API
    secret: ...
    redirectURIs: [...]
    tokenPolicy:
      claims:
        - key: "'custom-claim.company.com/team'"
          value: "identity.extra['team'].orValue('engineering')"
        # Only add on-call claim for ops group members
        - key: "'on_call'"
          value: "true"
          condition: "identity.groups.exists(g, g == 'ops')"
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
    types.go            # CEL type declarations (Identity, Request, etc.)
    cost.go             # Cost estimation and budgeting
    doc.go              # Package documentation
    library/
      email.go          # Email-related CEL functions
      groups.go         # Group-related CEL functions
```

#### Dependencies

```
github.com/google/cel-go v0.27.0
```

The `cel-go` library is the canonical Go implementation maintained by Google, used by Kubernetes
and all major CNCF projects. It follows semantic versioning and provides strong backward
compatibility guarantees.

#### Core API Design

**Public types:**

```go
// CompilationResult holds a compiled CEL program ready for evaluation.
type CompilationResult struct {
    Program    cel.Program
    OutputType *cel.Type
    Expression string
}

// Compiler compiles CEL expressions against a specific environment.
type Compiler struct { /* ... */ }

// CompilerOption configures a Compiler.
type CompilerOption func(*compilerConfig)
```

**Compilation pipeline:**

Each `Compile*` call performs these steps sequentially:
1. Reject expressions exceeding `MaxExpressionLength` (10,240 chars).
2. Compile and type-check the expression via `cel-go`.
3. Validate output type matches the expected type (for typed variants).
4. Estimate cost using `defaultCostEstimator` with size hints — reject if estimated max cost
   exceeds the cost budget.
5. Create an optimized `cel.Program` with runtime cost limit.

Presence tests (`has(field)`, `'key' in map`) have zero cost, matching Kubernetes CEL behavior.

#### Variable Declarations

Variables are declared via `VariableDeclaration{Name, Type}` and registered with `NewCompiler`.
Helper constructors provide pre-defined variable sets:

**`IdentityVariables()`** — the `identity` variable (from `connector.Identity`),
typed as `cel.ObjectType`:

| Field | CEL Type | Source |
|-------|----------|--------|
| `identity.user_id` | `string` | `connector.Identity.UserID` |
| `identity.username` | `string` | `connector.Identity.Username` |
| `identity.preferred_username` | `string` | `connector.Identity.PreferredUsername` |
| `identity.email` | `string` | `connector.Identity.Email` |
| `identity.email_verified` | `bool` | `connector.Identity.EmailVerified` |
| `identity.groups` | `list(string)` | `connector.Identity.Groups` |

**`RequestVariables()`** — the `request` variable (from `RequestContext`),
typed as `cel.ObjectType`:

| Field | CEL Type |
|-------|----------|
| `request.client_id` | `string` |
| `request.connector_id` | `string` |
| `request.scopes` | `list(string)` |
| `request.redirect_uri` | `string` |

**`ClaimsVariable()`** — the `claims` variable for raw upstream claims as `map(string, dyn)`.

**Typing strategy:**

`identity` and `request` use `cel.ObjectType` with explicitly declared fields. This gives
compile-time type checking: a typo like `identity.emial` is rejected at config load time
rather than silently evaluating to null in production — critical for an auth system where a
misconfigured policy could lock users out.

`claims` remains `map(string, dyn)` because its shape is genuinely unknown — it carries
arbitrary upstream IdP data.

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
   func WithVersion(v EnvironmentVersion) CompilerOption
   ```

   This is directly modeled on `k8s.io/apiserver/pkg/cel/environment`.

2. **Library stability** — Custom functions in the `pkg/cel/library` subpackage follow these rules:
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

Like Kubernetes, Dex CEL expressions must be bounded to prevent denial-of-service.

**Constants:**

| Constant | Value | Description |
|----------|-------|-------------|
| `DefaultCostBudget` | `10_000_000` | Max cost units per evaluation (aligned with Kubernetes) |
| `MaxExpressionLength` | `10_240` | Max expression string length in characters |
| `DefaultStringMaxLength` | `256` | Estimated max string size for cost estimation |
| `DefaultListMaxLength` | `100` | Estimated max list size for cost estimation |

**How it works:**

A `defaultCostEstimator` (implementing `checker.CostEstimator`) provides size hints for known
variables (`identity`, `request`, `claims`) so the `cel-go` cost estimator doesn't assume
unbounded sizes. It also provides call cost estimates for custom Dex functions
(`dex.emailDomain`, `dex.emailLocalPart`, `dex.groupMatches`, `dex.groupFilter`).

Expressions are validated at three levels:
1. **Length check** — reject expressions exceeding `MaxExpressionLength`.
2. **Compile-time cost estimation** — reject expressions whose estimated max cost exceeds
   the cost budget.
3. **Runtime cost limit** — abort evaluation if actual cost exceeds the budget.

#### Extension Libraries

The `pkg/cel` environment includes these cel-go standard extensions (same set as Kubernetes):

| Library | Description | Examples |
|---------|-------------|---------|
| `ext.Strings()` | Extended string functions | `"hello".upperAscii()`, `"foo:bar".split(':')`, `s.trim()`, `s.replace('a','b')` |
| `ext.Encoders()` | Base64 encoding/decoding | `base64.encode(bytes)`, `base64.decode(str)` |
| `ext.Lists()` | Extended list functions | `list.slice(1, 3)`, `list.flatten()` |
| `ext.Sets()` | Set operations on lists | `sets.contains(a, b)`, `sets.intersects(a, b)`, `sets.equivalent(a, b)` |
| `ext.Math()` | Math functions | `math.greatest(a, b)`, `math.least(a, b)` |

Plus custom Dex libraries in the `pkg/cel/library` subpackage, each implementing the
`cel.Library` interface:

**`library.Email`** — email-related helpers:

| Function | Signature | Description |
|----------|-----------|-------------|
| `dex.emailDomain` | `(string) -> string` | Returns the domain portion of an email address. `dex.emailDomain("user@example.com") == "example.com"` |
| `dex.emailLocalPart` | `(string) -> string` | Returns the local part of an email address. `dex.emailLocalPart("user@example.com") == "user"` |

**`library.Groups`** — group-related helpers:

| Function | Signature | Description |
|----------|-----------|-------------|
| `dex.groupMatches` | `(list(string), string) -> list(string)` | Returns groups matching a glob pattern. `dex.groupMatches(identity.groups, "team:*")` |
| `dex.groupFilter` | `(list(string), list(string)) -> list(string)` | Returns only groups present in the allowed list. `dex.groupFilter(identity.groups, ["admin", "ops"])` |

#### Example: Compile and Evaluate

```go
// 1. Create a compiler with identity and request variables
compiler, _ := cel.NewCompiler(
    append(cel.IdentityVariables(), cel.RequestVariables()...),
)

// 2. Compile a policy expression (type-checked, cost-estimated)
prog, _ := compiler.CompileBool(
    `identity.email.endsWith('@example.com') && 'admin' in identity.groups`,
)

// 3. Evaluate against real data
result, _ := cel.EvalBool(ctx, prog, map[string]any{
    "identity": cel.IdentityFromConnector(connectorIdentity),
    "request":  cel.RequestFromContext(cel.RequestContext{...}),
})
// result == true
```

### Phase 2: Authentication Policies

**Config Model:**

```go
// AuthPolicy is a list of deny expressions evaluated after a user
// authenticates with a connector. Each expression evaluates to bool.
// If true — the request is denied. Evaluated in order; first match wins.
type AuthPolicy []PolicyExpression

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
         │
         v
connector.HandleCallback() returns Identity
         │
         v
Evaluate global authPolicy (in order)
  - For each expression: evaluate → bool
  - If true → deny with message, HTTP 403
         │
         v
Evaluate per-client authPolicy (in order)
  - Same logic as global
         │
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
    // Condition is an optional CEL expression evaluating to bool.
    // When set, the claim is only included in the token if the condition
    // evaluates to true. If omitted, the claim is always included.
    Condition string `json:"condition,omitempty"`
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

### Policy Application Flow

The following diagram shows the order in which CEL policies are applied.
Each step is optional — if not configured, it is skipped.

```
Connector Authentication
  │
  │ upstream claims → connector.Identity
  │
  v
Authentication Policies
  │
  │  Global authPolicy
  │  Per-client authPolicy
  │
  v
Token Issuance
  │
  │  Global tokenPolicy.filter
  │  Per-client tokenPolicy.filter
  │
  │  Global tokenPolicy.claims
  │  Per-client tokenPolicy.claims
  │
  │  Sign JWT
  │
  v
Token Response
```

| Step | Policy | Scope | Action on match |
|------|--------|-------|-----------------|
| 2 | `authPolicy` (global) | Global | Expression → `true` = DENY login |
| 3 | `authPolicy` (per-client) | Per-client | Expression → `true` = DENY login |
| 4 | `tokenPolicy.filter` (global) | Global | Expression → `false` = DENY token |
| 5 | `tokenPolicy.filter` (per-client) | Per-client | Expression → `false` = DENY token |
| 6 | `tokenPolicy.claims` (global) | Global | Adds/overrides claims (with optional condition) |
| 7 | `tokenPolicy.claims` (per-client) | Per-client | Adds/overrides claims (overrides global) |

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


