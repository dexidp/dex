# Dex Enhancement Proposal (DEP) - 2026-02-24 - External Connectors

## Table of Contents

- [Summary](#summary)
- [Motivation](#motivation)
    - [Goals/Pain](#goals)
    - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [User Experience](#user-experience)
    - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Risks and Mitigations](#risks-and-mitigations)
    - [Alternatives](#alternatives)
- [Future Improvements](#future-improvements)

## Summary

This DEP proposes splitting Dex connectors into **core** (bundled) and **external** categories,
and providing an official, supported path for users to build custom Dex distributions
with their own connectors compiled in — similar to how the OpenTelemetry Collector
and Kubernetes Scheduler handle extensibility.

Core connectors (LDAP, OIDC, OAuth2, SAML, AuthProxy) remain in the main repository.
All provider-specific connectors (GitHub, GitLab, Google, Microsoft, etc.) are moved
to a separate monorepo `dexidp/dex-connectors` and maintained as independent Go modules.
Dex provides a **builder tool** (`dexbuilder`) and a **registration API** that makes it
trivial to compile a custom `dex` binary with any combination of connectors.

## Context

- Dex currently ships ~15 connectors in the main repo, each pulling in provider-specific dependencies, which increases binary size, maintenance burden, and the attack surface.
- Connector bugs or dependency updates block Dex core releases, and vice-versa.
- Community members who want to add new connectors must submit PRs to the core repo, where maintainers become responsible for code they may not have expertise in.
- The OpenTelemetry Collector solved an identical problem with [ocb (OpenTelemetry Collector Builder)](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder) — users declare desired components in a manifest and the tool generates a `main.go` that compiles them in.
- Kubernetes Scheduler uses a similar approach with its [component config and plugin registration pattern](https://kubernetes.io/docs/reference/scheduling/config/).

## Motivation

### Goals/Pain

- **Reduce core maintenance burden**: Maintainers should only be responsible for protocol-level connectors (LDAP, OIDC, OAuth2, SAML, AuthProxy) that are generic and widely used.
- **Enable community ownership**: Provider-specific connector maintainers can release, test, and iterate independently.
- **Provide an official extensibility path**: Users with private or niche identity providers can build and maintain their own connectors without forking Dex.
- **Reduce dependency bloat**: The core binary only includes dependencies for core connectors; external connectors bring their own dependencies at build time.
- **Faster release cycles**: Connector updates no longer block core releases, and connector authors can release at their own pace.

### Non-goals

- Runtime plugin loading (Go plugins, WASM, gRPC) — explicitly not chosen (see [Alternatives](#alternatives)).
- Changing connector interfaces — the existing `connector.Connector`, `CallbackConnector`, `PasswordConnector`, `SAMLConnector`, `RefreshConnector`, and `TokenIdentityConnector` interfaces remain unchanged.
- Moving storage backends out of the main repo — out of scope, but the same pattern could be applied later.
- Abandoning existing external connectors — they will continue to be maintained under `dexidp`, just in a separate repository.

## Proposal

### User Experience

#### For users deploying Dex with default connectors

Nothing changes. The default `dex` binary and Docker image will continue to ship with all
current connectors (core + dexidp-maintained external connectors).
This is the **"batteries-included"** distribution.

#### For users who want a custom set of connectors

Users create a **builder manifest** file (`builder.yaml`):

```yaml
dex:
  version: v2.43.0

connectors:
  # Core connectors (ldap, oidc, oauth, saml, authproxy) are always included.
  # List additional connector Go modules below.
  # Each entry is a Go module path + version.
  - gomod: github.com/dexidp/dex-connectors/github v0.3.0
  - gomod: github.com/dexidp/dex-connectors/gitlab v0.2.1
  - gomod: github.com/myorg/dex-connector-okta v1.0.0

# Optional: exclude specific core connectors you don't need.
# By default all core connectors are included.
excludeCoreConnectors:
  - authproxy
```

Then run the builder tool:

```bash
# Install the builder
go install github.com/dexidp/dex/cmd/dexbuilder@latest

# Build custom dex binary
dexbuilder --config builder.yaml --output ./dex
```

**How it works**:

- The `connectors` list contains **Go module paths**. This is the only supported format —
  each connector is a Go module that can be fetched by `go mod download`.
- Each connector library defines its **type name** (e.g., `"github"`, `"okta"`) internally
  via `init()` registration. The type name is what users put in the Dex config file
  under `connectors[].type`. Users do **not** choose or specify the type name in `builder.yaml` —
  it comes from the connector library itself.
- **Name collisions**: If two connector modules register the same type name, the build
  will fail with a panic at init time: `connector type "xxx" already registered`.
  This is intentional — type names must be globally unique. External connector authors
  must choose names that don't collide with core connector type names
  (`ldap`, `oidc`, `oauth`, `saml`, `authproxy`) or other well-known connectors.
- **Excluding core connectors**: The `excludeCoreConnectors` field allows removing
  specific core connectors from the build. This is useful for minimizing binary size or
  reducing attack surface when certain core connectors are not needed. The builder
  simply omits the blank import for excluded connectors in the generated `main.go`.

#### For connector developers

Connector developers create a standard Go module that:

1. Implements the `connector.Connector` interface (and any of the sub-interfaces: `CallbackConnector`, `PasswordConnector`, etc.)
2. Implements the `connector.Config` interface (see below — this is the unexported-field config interface moved to the `connector` package)
3. Calls `connector.Register()` in an `init()` function to register its type name

The **type name** is defined by the connector author in their `init()` call and becomes the
value users use in `connectors[].type` in the Dex configuration file.

Example for a hypothetical Okta connector:

```go
package okta

import (
    "log/slog"

    "github.com/dexidp/dex/connector"
)

func init() {
    // "okta" is the type name users will use in dex config: type: okta
    connector.Register("okta", func() connector.Config {
        return new(OktaConfig)
    })
}

type OktaConfig struct {
    Issuer       string `json:"issuer"`
    ClientID     string `json:"clientID"`
    ClientSecret string `json:"clientSecret"`
}

func (c *OktaConfig) Open(id string, logger *slog.Logger) (connector.Connector, error) {
    return &oktaConnector{
        issuer:   c.Issuer,
        clientID: c.ClientID,
        // ...
    }, nil
}

type oktaConnector struct {
    // ...
}

// Implement connector.CallbackConnector, connector.RefreshConnector, etc.
```

### Implementation Details/Notes/Constraints

#### Phase 1: Connector Registration API

Move the `ConnectorConfig` interface to the `connector` package (renamed to `connector.Config`)
and make the registration registry live there. This avoids circular imports — connector
packages depend on `connector` package, and `server` package also depends on `connector` package.

```go
// connector/registry.go

// Config is a configuration that can open a connector.
type Config interface {
    Open(id string, logger *slog.Logger) (Connector, error)
}

// registry is the internal map of registered connector types.
// It is not exported — access is only through Register() and Get().
var registry = map[string]func() Config{}

// Register registers a connector type that can be used in Dex configuration.
// It is intended to be called from init() functions of connector packages.
// Calling Register with an already registered type name will panic.
func Register(typeName string, factory func() Config) {
    if _, exists := registry[typeName]; exists {
        panic(fmt.Sprintf("connector type %q already registered", typeName))
    }
    registry[typeName] = factory
}

// Get returns the config factory for the given connector type name.
// Returns nil if the type is not registered.
func Get(typeName string) func() Config {
    return registry[typeName]
}

// RegisteredTypes returns a list of all registered connector type names.
func RegisteredTypes() []string {
    types := make([]string, 0, len(registry))
    for t := range registry {
        types = append(types, t)
    }
    return types
}
```

Key design decisions:
- The `registry` map is **unexported**. External code cannot modify it directly — only
  through `Register()`, which enforces uniqueness via panic.
- `connector.Config` replaces `server.ConnectorConfig`. The `server` package uses
  `connector.Get()` to look up connector types instead of its own `ConnectorsConfig` map.

Core connectors register themselves via `init()` in their own packages:

```go
// connector/ldap/ldap.go
func init() {
    connector.Register("ldap", func() connector.Config {
        return new(Config)
    })
}
```

The existing `ConnectorsConfig` variable and hardcoded map in `server/server.go` is removed.
The `openConnector` function is updated to use `connector.Get()`:

```go
// server/server.go

func openConnector(logger *slog.Logger, conn storage.Connector) (connector.Connector, error) {
    factory := connector.Get(conn.Type)
    if factory == nil {
        return nil, fmt.Errorf("unknown connector type %q", conn.Type)
    }

    connConfig := factory()
    if len(conn.Config) != 0 {
        if err := json.Unmarshal(conn.Config, connConfig); err != nil {
            return nil, fmt.Errorf("parse connector config: %v", err)
        }
    }

    c, err := connConfig.Open(conn.ID, logger)
    if err != nil {
        return nil, fmt.Errorf("failed to create connector %s: %v", conn.ID, err)
    }
    return c, nil
}
```

Core connectors are included via blank imports. In the default distribution, all connectors
are imported in `cmd/dex/connectors.go`:

```go
// cmd/dex/connectors.go
package main

import (
    // Core connectors (always included in default distribution)
    _ "github.com/dexidp/dex/connector/ldap"
    _ "github.com/dexidp/dex/connector/oidc"
    _ "github.com/dexidp/dex/connector/oauth"
    _ "github.com/dexidp/dex/connector/saml"
    _ "github.com/dexidp/dex/connector/authproxy"
    _ "github.com/dexidp/dex/connector/mock"

    // External connectors (included in default distribution for backwards compatibility)
    _ "github.com/dexidp/dex-connectors/github"
    _ "github.com/dexidp/dex-connectors/gitlab"
    _ "github.com/dexidp/dex-connectors/google"
    _ "github.com/dexidp/dex-connectors/microsoft"
    _ "github.com/dexidp/dex-connectors/linkedin"
    _ "github.com/dexidp/dex-connectors/bitbucketcloud"
    _ "github.com/dexidp/dex-connectors/openshift"
    _ "github.com/dexidp/dex-connectors/gitea"
    _ "github.com/dexidp/dex-connectors/atlassiancrowd"
    _ "github.com/dexidp/dex-connectors/keystone"
)
```

#### Phase 2: Move external connectors to `dexidp/dex-connectors`

All external connectors are moved to a single monorepo `github.com/dexidp/dex-connectors`
with one Go module per connector:

```
dexidp/dex-connectors/
├── github/
│   ├── go.mod          # module github.com/dexidp/dex-connectors/github
│   ├── go.sum
│   ├── github.go
│   └── github_test.go
├── gitlab/
│   ├── go.mod          # module github.com/dexidp/dex-connectors/gitlab
│   └── ...
├── google/
│   ├── go.mod          # module github.com/dexidp/dex-connectors/google
│   └── ...
├── microsoft/
│   ├── go.mod          # module github.com/dexidp/dex-connectors/microsoft
│   └── ...
├── linkedin/
│   ├── go.mod          # module github.com/dexidp/dex-connectors/linkedin
│   └── ...
├── bitbucket/
│   ├── go.mod          # module github.com/dexidp/dex-connectors/bitbucket
│   └── ...
├── openshift/
│   ├── go.mod          # module github.com/dexidp/dex-connectors/openshift
│   └── ...
├── gitea/
│   ├── go.mod          # module github.com/dexidp/dex-connectors/gitea
│   └── ...
├── atlassiancrowd/
│   ├── go.mod          # module github.com/dexidp/dex-connectors/atlassiancrowd
│   └── ...
└── keystone/
    ├── go.mod          # module github.com/dexidp/dex-connectors/keystone
    └── ...
```

**Why a monorepo with multiple modules (not separate repos)?**
- Single place for CI, issue tracking, and contributor guidelines.
- Shared tooling (linters, Makefile, release scripts).
- Easier to discover all dexidp-maintained connectors.
- Each connector is still an independent Go module with its own `go.mod`,
  so it can be versioned and released independently.
- A `dependabot.yml` at the root keeps all connectors' dependencies up to date.
- This is the same pattern used by `open-telemetry/opentelemetry-collector-contrib`.

Each module:
- Has its own `go.mod` depending on `github.com/dexidp/dex` (for the `connector` package interfaces)
- Can be versioned independently via Git tags (`github/v0.3.0`, `gitlab/v0.2.1`, etc.)
- Has a `CODEOWNERS` entry for community maintainers

| Connector        | Module Path                                            | Type Key           |
|------------------|--------------------------------------------------------|--------------------|
| GitHub           | `github.com/dexidp/dex-connectors/github`              | `github`           |
| GitLab           | `github.com/dexidp/dex-connectors/gitlab`              | `gitlab`           |
| Google           | `github.com/dexidp/dex-connectors/google`              | `google`           |
| Microsoft        | `github.com/dexidp/dex-connectors/microsoft`           | `microsoft`        |
| LinkedIn         | `github.com/dexidp/dex-connectors/linkedin`            | `linkedin`         |
| Bitbucket Cloud  | `github.com/dexidp/dex-connectors/bitbucket`           | `bitbucket-cloud`  |
| OpenShift        | `github.com/dexidp/dex-connectors/openshift`           | `openshift`        |
| Gitea            | `github.com/dexidp/dex-connectors/gitea`               | `gitea`            |
| Atlassian Crowd  | `github.com/dexidp/dex-connectors/atlassiancrowd`      | `atlassian-crowd`  |
| Keystone         | `github.com/dexidp/dex-connectors/keystone`            | `keystone`         |

#### Phase 3: Builder tool (`dexbuilder`)

Create `cmd/dexbuilder/` that:

1. Reads a `builder.yaml` manifest
2. Validates that `excludeCoreConnectors` only lists known core connector names
3. Creates a temporary Go module directory
4. Generates a `main.go` that imports:
   - Core connectors (minus any excluded ones) via blank imports
   - All requested external connector Go modules via blank imports
5. Generates `go.mod` with the specified Dex version and connector module dependencies
6. Runs `go build` to produce the final binary

The generated `main.go` looks like:

```go
// Code generated by dexbuilder. DO NOT EDIT.
package main

import (
    dex "github.com/dexidp/dex/cmd/dex"

    // Core connectors
    _ "github.com/dexidp/dex/connector/ldap"
    _ "github.com/dexidp/dex/connector/oidc"
    _ "github.com/dexidp/dex/connector/oauth"
    _ "github.com/dexidp/dex/connector/saml"
    // authproxy excluded by user config

    // External connectors
    _ "github.com/dexidp/dex-connectors/github"
    _ "github.com/myorg/dex-connector-okta"
)

func main() {
    dex.Main()
}
```

If two imported connector modules register the same type name, the binary will panic
at startup during `init()` execution with a clear error message:
`panic: connector type "github" already registered`.

This requires extracting the core CLI logic from `cmd/dex/main.go` into an exported `Main()` function:

```go
// cmd/dex/run.go

// Main is the entry point for the dex server.
// It is exported so that custom distributions can call it from their own main().
func Main() {
    if err := commandRoot().Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err.Error())
        os.Exit(2)
    }
}
```

#### Phase 4: Docker images

The main `dexidp/dex` repo CI produces two Docker images:

- `ghcr.io/dexidp/dex` — all dexidp-maintained connectors (current behavior). Uses a `builder.yaml` in the repo that pulls all connectors from `dexidp/dex-connectors`.
- `ghcr.io/dexidp/dex-core` — core connectors only (lean image). Uses `cmd/dex/` with only
  core connector imports.

During a transition period, the default `ghcr.io/dexidp/dex` image continues to ship all
connectors.

#### Required changes to `dex` repository

1. **Move `ConnectorConfig` to `connector` package** as `connector.Config`:
   - Add `Register()`, `Get()`, `RegisteredTypes()` functions
   - Make the internal `registry` map unexported
   - Remove `server.ConnectorsConfig` map and `server.ConnectorConfig` interface

2. **Add `init()` registration** to each existing connector package.

3. **Update `server/server.go`**:
   - Remove all connector imports
   - Remove `ConnectorsConfig` map
   - Update `openConnector()` to use `connector.Get()`
   - Update `cmd/dex/config.go` unmarshalling to use `connector.Get()`

4. **Create `cmd/dex/connectors.go`** with blank imports of all connectors
   (for the default distribution).

5. **Export `Main()` function** from `cmd/dex/` for custom distribution entry points.

6. **Create `cmd/dexbuilder/`** — the builder tool.

#### Migration path

1. **v2.x (Phase 1)**: Add `connector.Register()` API. All connectors still in the main repo,
   but each registers itself via `init()`. The hardcoded `ConnectorsConfig` map is removed.
   **Fully backwards compatible** — same binary, same behavior.

2. **v2.x+1 (Phase 2-3)**: Introduce `dexbuilder`. Copy connectors to `dexidp/dex-connectors`
   (keep originals in main repo with deprecation notices). Default distribution still
   includes everything.

3. **v3.0 (Phase 4)**: Remove external connectors from the main module. Along with the `dex` image we ship `dex-core` image.
   Breaking change in import paths only — Dex config file format remains unchanged.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| **Breaking change for users importing connector packages** | Maintain Go module redirects (`Deprecated` comments + module retraction). Provide migration guide. Phase over 2-3 minor releases. |
| **Interface versioning — connector interface changes break all external connectors** | Connector interfaces are already stable. Introduce a `connector.Version` constant that connectors can check. Major interface changes require a new interface (e.g., `CallbackConnectorV2`) with adapter fallback. |
| **Discovery — users won't know which connectors exist** | Maintain a registry page in docs and a `connectors.yaml` index in the main repo listing all known connectors with their module paths and status. |
| **Quality control for third-party connectors** | Provide `connector/conformance` test suite (already partially exists in `storage/conformance`). Third-party connectors can run conformance tests in their CI. Official `dexidp/` connectors are held to the same standards as core. |
| **Build complexity for users** | `dexbuilder` makes it a single command. Provide example `Dockerfile` and GitHub Actions workflow for building custom images. |
| **Type name collisions between connectors** | `connector.Register()` panics on duplicate names. Core connector names are reserved and documented. External connector authors must check the registry before choosing a name. |
| **Security: malicious third-party connectors** | Third-party connectors run with full process privileges — they are compiled in, not sandboxed. This is clearly documented. Users are responsible for auditing third-party connector code. Official `dexidp/` connectors go through code review. The `dexbuilder` tool prints a warning when building with non-`dexidp` modules. |

### Alternatives

#### 1. WebAssembly (WASM) plugins

WASM has significant limitations for Go (large binary size, limited stdlib support,
no network access without WASI). Compiling connectors to WASM is complex, and
runtime overhead is non-trivial for auth-critical paths.

**Security concerns**: WASM provides sandboxing, which is a benefit. However, connectors
inherently need network access (to talk to upstream identity providers) and access to
configuration secrets (client IDs, client secrets). Granting these capabilities through
WASI largely negates the sandboxing benefit. Additionally, the WASM-to-host boundary
becomes a new attack surface that needs careful specification and auditing.

#### 2. Go plugins (`plugin` package)

Go plugins require exact Go version and dependency version matching between the plugin
and the host binary. This makes distribution extremely fragile. Plugins are not
supported on all platforms (notably Windows). The Go team has effectively deprioritized
the feature.

**Security concerns**: Go plugins run in the same process with full access to the host
memory space, same as compiled-in code, but without the ability to audit them at build
time. A malicious `.so` plugin loaded at runtime can do anything the host process can —
read secrets, exfiltrate data, modify behavior of other connectors. With compile-time
inclusion, the source code is at least visible and auditable in `go.sum`.

#### 3. gRPC-based plugins (HashiCorp go-plugin style)

Adds operational complexity (sidecar processes, health checking, deployment of additional
binaries). Performance overhead for every auth request (serialization + network hop).
Significantly more complex to develop, test, and debug connectors.

**Security concerns**: gRPC plugins communicate over local sockets or TCP, introducing
a new IPC attack surface. The host must authenticate plugin connections to prevent
impersonation. Plugins run as separate processes and could be replaced or intercepted
by a local attacker. TLS between host and plugin adds complexity. Configuration secrets
must be transmitted to the plugin process, creating additional exposure points. If
plugins are fetched and started automatically, supply chain attacks become a concern —
a compromised plugin binary runs with whatever OS-level privileges the Dex process has.

#### 4. Do nothing

Dex continues to accumulate connectors in the main repo. Maintenance burden grows.
Community contributions slow down due to high review requirements for code maintainers
don't use. Dependency conflicts become more frequent.

## Future Improvements

- **Connector conformance test suite**: Expand `connector/conformance` to provide a standard set of tests any connector can run (interface compliance, error handling, context cancellation, etc.)
- **`dexbuilder` Docker integration**: `dexbuilder --docker` to directly produce a Docker image.
- **Connector catalog website**: A searchable catalog of community connectors, similar to the Terraform Registry or OTel Collector contrib components.
- **Storage backends**: Apply the same external pattern to storage backends (ent, etcd, kubernetes, SQL). However, for the storage we can take similar approach in the future.
- **Hot-reload of connector configuration**: While connectors are still compiled in, their configuration could be reloaded without restart.
- **Versioned connector interfaces**: If connector interfaces need breaking changes in the future, provide versioned interfaces (`CallbackConnectorV2`) with automatic adapter wrapping for backward compatibility.

