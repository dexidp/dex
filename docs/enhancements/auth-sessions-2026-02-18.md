# Dex Enhancement Proposal (DEP) - 2026-02-18 - Auth Sessions

## Table of Contents

- [Summary](#summary)
- [Motivation](#motivation)
    - [Goals/Pain](#goalspain)
    - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [User Experience](#user-experience)
    - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Risks and Mitigations](#risks-and-mitigations)
    - [Alternatives](#alternatives)
- [Future Improvements](#future-improvements)

## Summary

This DEP introduces **auth sessions** - a persistent authentication state that enables Dex to track logged-in users across browser sessions. Currently, Dex relies entirely on refresh tokens for session management, which prevents proper implementation of OIDC conformance features like `prompt=none`, `prompt=login`, `id_token_hint`, SSO across clients, and proper logout. User Sessions will be stored server-side with a browser cookie reference, enabling these features while maintaining Dex's simplicity and compatibility with all storage backends (SQL, etcd, Kubernetes CRDs).

## Context

- [OIDC Core 1.0 - Authentication Request](https://openid.net/specs/openid-connect-core-1_0.html#AuthRequest) - `prompt` parameter specification
- [OIDC Core 1.0 - ID Token Hint](https://openid.net/specs/openid-connect-core-1_0.html#IDToken) - `id_token_hint` specification
- [OIDC Session Management 1.0](https://openid.net/specs/openid-connect-session-1_0.html) - Session management specification
- [OIDC RP-Initiated Logout 1.0](https://openid.net/specs/openid-connect-rpinitiated-1_0.html) - Logout specification
- [OIDC Front-Channel Logout 1.0](https://openid.net/specs/openid-connect-frontchannel-1_0.html) - Front-channel logout
- [Keycloak Sessions](https://www.keycloak.org/docs/latest/server_admin/#_sessions) - Reference implementation
- [Ory Hydra Login & Consent Flow](https://www.ory.sh/docs/hydra/concepts/login) - Reference implementation

Current limitations:
- No support for `prompt=none` (silent authentication)
- No support for `prompt=login` (force re-authentication)
- No support for `max_age` parameter
- No support for `id_token_hint` validation
- No SSO between clients (each client requires separate login)
- No proper logout (only refresh token revocation)
- No consent persistence (user must approve every time if not skipped globally)
- No 2FA enrollment storage
- No "Remember Me" functionality

## Motivation

### Goals/Pain

1. **OIDC Conformance** - Enable proper `prompt=none`, `prompt=login`, `max_age`, and `id_token_hint` support
2. **SSO (Single Sign-On)** - Allow users to authenticate once and access multiple clients without re-login
3. **Remember Me** - Allow users to choose persistent vs session-based authentication
4. **Consent Persistence** - Store user consent decisions per client/scope combination within session
5. **Proper Logout** - Enable session termination with optional front-channel logout
6. **Foundation for 2FA** - Enable future TOTP/WebAuthn enrollment storage

### Non-Goals

- **2FA Implementation** - This DEP only provides storage foundation; 2FA flow is a separate DEP
- **Back-Channel Logout** - Server-to-server logout notifications are out of scope
- **Session Clustering/Replication** - Storage backends handle this
- **Admin Session Management UI** - API only, no admin UI
- **Per-connector Session Policies** - Single global session policy initially
- **Identity Refresh During Session** - Deferred to future DEP; initially identity is refreshed only at session termination (like Keycloak)
- **Upstream Connector Logout** - Terminating sessions at upstream IDPs is deferred

## Proposal

### User Experience

#### Configuration

Sessions are controlled by a feature flag and configuration:

```yaml
# Feature flag (environment variable)
# DEX_SESSIONS_ENABLED=true

# config.yaml
sessions:
  # Session cookie name (default: "dex_session")
  # Other cookie settings (Secure, HttpOnly, SameSite=Lax) are not configurable
  # and are set to secure defaults automatically
  cookieName: "dex_session"

  # Session lifetime settings (matches refresh token expiry naming)
  absoluteLifetime: "24h"         # Maximum session lifetime, default: 24h
  validIfNotUsedFor: "1h"         # Session expires if not used, default: 1h

  # Default SSO sharing policy for clients without explicit ssoSharedWith config
  # Options:
  #   "all" - clients without ssoSharedWith share sessions with all other clients (Keycloak-like)
  #   "none" - clients without ssoSharedWith don't share sessions (default)
  ssoSharedWithDefault: "none"

  # Whether "Remember Me" checkbox is checked by default in login/approval forms
  # When true: checkbox is pre-checked, user can uncheck
  # When false: checkbox is unchecked, user must check to persist session (default)
  rememberMeCheckedByDefault: false
```

**ssoSharedWithDefault** controls the default SSO behavior:
- `"none"` (default): Clients without explicit `ssoSharedWith` config don't participate in SSO
- `"all"`: Clients without explicit `ssoSharedWith` config share sessions with all other clients (realm-wide SSO like Keycloak)

Clients with explicit `ssoSharedWith` configuration always use their configured value.

**Note**: The `ssoSharedWith` option is separate from the existing `trustedPeers` option. `trustedPeers` controls which clients can issue tokens on behalf of this client (existing behavior), while `ssoSharedWith` controls which clients can reuse this client's authentication session (new behavior). These can be configured independently based on different security requirements.

**rememberMeCheckedByDefault** controls the initial checkbox state in templates.
This value is passed to templates as `.RememberMeChecked` boolean.

**SSO via ssoSharedWith**: SSO between clients is controlled by the new `ssoSharedWith` configuration on clients. The `ssoSharedWith` setting defines **which clients can USE this client's session**, not which clients this client can use.

If client B is listed in client A's `ssoSharedWith`:
1. If user logged in via client A, client B can reuse that session

This is intentionally separate from `trustedPeers` (which controls token issuance on behalf of another client). Organizations may want different policies for session sharing vs token delegation:
- **ssoSharedWith**: "Can this client's login be reused by another client?"
- **trustedPeers**: "Can another client issue tokens claiming to be this client?"

**Wildcard Support**: `ssoSharedWith: ["*"]` enables SSO with all clients. This is similar to Keycloak's default behavior where all clients in a realm share sessions.

**SSO Direction**: SSO sharing is **unidirectional**. Client A sharing with client B does NOT mean client B shares with client A.

```yaml
staticClients:
  # Public app - allows any client to reuse its sessions
  - id: public-app
    name: Public App
    ssoSharedWith: ["*"]
    # trustedPeers can be configured separately for token delegation
    # ...

  # Admin app - only specific apps can reuse its sessions
  - id: admin-app
    name: Admin App
    ssoSharedWith: ["monitoring-app"]  # Only monitoring can SSO from admin sessions
    # ...

  # Secret internal service - NO other clients can reuse its sessions
  - id: secret-service
    name: Secret Service
    ssoSharedWith: []  # Empty = no SSO allowed from this client's sessions
    # But this client CAN use sessions from other clients that share with it!
    # ...

  # Monitoring app - can SSO from admin-app (because admin-app shares with it)
  - id: monitoring-app
    name: Monitoring App
    ssoSharedWith: ["admin-app"]  # Bidirectional sharing with admin-app
    # ...
```

**Example Scenarios:**

| User logged in via | Accessing | SSO works? | Why |
|-------------------|-----------|------------|-----|
| public-app | admin-app | ✅ Yes | public-app has `ssoSharedWith: ["*"]` |
| admin-app | public-app | ❌ No | admin-app only shares with monitoring-app |
| admin-app | monitoring-app | ✅ Yes | admin-app shares with monitoring-app |
| secret-service | any client | ❌ No | secret-service has `ssoSharedWith: []` |
| public-app | secret-service | ✅ Yes | public-app has `ssoSharedWith: ["*"]` |

**Key Insight**: A "secret" client that doesn't want others to SSO into it simply doesn't list them in `ssoSharedWith`. But it can still BENEFIT from SSO by being listed in OTHER clients' `ssoSharedWith`.

**Comparison with Keycloak**: In Keycloak, SSO is realm-wide by default - all clients in a realm share sessions. Dex's approach is more granular: SSO is opt-in per client via `ssoSharedWith`. Use `["*"]` to achieve Keycloak-like behavior.

**Comparison with trustedPeers**: The `trustedPeers` option continues to control cross-client token issuance (e.g., client B issuing tokens for client A). This is a separate security concern from session sharing. Organizations can configure these independently:
- High SSO sharing, restricted token delegation
- Restricted SSO sharing, high token delegation
- Or any combination based on their security model

**Cookie Security**: The session cookie is always set with secure defaults:
- `HttpOnly: true` - Not accessible via JavaScript
- `Secure: true` - Only sent over HTTPS (automatically disabled for localhost in dev)
- `SameSite: Lax` - CSRF protection
- `Path: <issuerURL.Path>` - Derived from issuer URL (e.g., `/dex` for `https://example.com/dex`)

These settings are not configurable to prevent security misconfigurations.

#### Authentication Flow with Sessions

```
┌─────────┐       ┌─────────┐       ┌───────────┐       ┌───────────┐
│ Browser │       │  Dex    │       │  Storage  │       │ Connector │
└────┬────┘       └────┬────┘       └─────┬─────┘       └─────┬─────┘
     │                 │                  │                   │
     │  GET /auth      │                  │                   │
     │ (no session)    │                  │                   │
     ├────────────────>│                  │                   │
     │                 │                  │                   │
     │                 │  Check session   │                   │
     │                 │  cookie          │                   │
     │                 ├─────────────────>│                   │
     │                 │  (not found)     │                   │
     │                 │<─────────────────│                   │
     │                 │                  │                   │
     │  Redirect to    │                  │                   │
     │  connector      │                  │                   │
     │<────────────────│                  │                   │
     │                 │                  │                   │
     │        ... connector auth flow ... │                   │
     │                 │                  │                   │
     │  Callback with  │                  │                   │
     │  identity       │                  │                   │
     ├────────────────>│                  │                   │
     │                 │                  │                   │
     │                 │  Create/update   │                   │
     │                 │  AuthSession     │                   │
     │                 │  (ALWAYS)        │                   │
     │                 ├─────────────────>│                   │
     │                 │                  │                   │
     │  Set-Cookie:    │                  │                   │
     │  - Session cookie (no MaxAge)      │                   │
     │    if Remember Me unchecked        │                   │
     │  - Persistent cookie (with MaxAge) │                   │
     │    if Remember Me checked          │                   │
     │  + redirect to /approval           │                   │
     │<────────────────│                  │                   │
     │                 │                  │                   │
```

**Key Point**: AuthSession is always created on successful authentication. The "Remember Me" checkbox only controls whether the cookie is a session cookie (deleted on browser close) or a persistent cookie (survives browser restart). This is consistent with Keycloak's behavior.

#### SSO Flow (Returning User)

```
┌─────────┐       ┌─────────┐       ┌───────────┐
│ Browser │       │  Dex    │       │  Storage  │
└────┬────┘       └────┬────┘       └─────┬─────┘
     │                 │                  │
     │  GET /auth      │                  │
     │ (with cookie)   │                  │
     │  client_id=B    │                  │
     ├────────────────>│                  │
     │                 │                  │
     │                 │  Get session     │
     │                 ├─────────────────>│
     │                 │  (valid session) │
     │                 │<─────────────────│
     │                 │                  │
     │                 │  Check SSO       │
     │                 │  policy for      │
     │                 │  client B        │
     │                 │                  │
     │                 │  Check consent   │
     │                 │  for client B    │
     │                 ├─────────────────>│
     │                 │                  │
     │  If consented:  │                  │
     │  redirect with  │                  │
     │  code           │                  │
     │<────────────────│                  │
     │                 │                  │
     │  If not:        │                  │
     │  show approval  │                  │
     │<────────────────│                  │
     │                 │                  │
```

#### prompt=none Flow

```
┌─────────┐       ┌─────────┐       ┌───────────┐
│ Browser │       │  Dex    │       │  Storage  │
└────┬────┘       └────┬────┘       └─────┬─────┘
     │                 │                  │
     │  GET /auth      │                  │
     │  prompt=none    │                  │
     ├────────────────>│                  │
     │                 │                  │
     │                 │  Get session     │
     │                 ├─────────────────>│
     │                 │                  │
     │  If valid session + consent:       │
     │  redirect with code                │
     │<────────────────│                  │
     │                 │                  │
     │  If no session or no consent:      │
     │  redirect with error=login_required│
     │  or error=consent_required         │
     │<────────────────│                  │
     │                 │                  │
```

#### Logout Flow

```
┌─────────┐       ┌─────────┐       ┌───────────┐
│ Browser │       │  Dex    │       │  Storage  │
└────┬────┘       └────┬────┘       └─────┬─────┘
     │                 │                  │
     │  GET /logout    │                  │
     │  id_token_hint= │                  │
     ├────────────────>│                  │
     │                 │                  │
     │                 │  Validate        │
     │                 │  id_token_hint   │
     │                 │                  │
     │                 │  Get identity    │
     │                 │  by session ID   │
     │                 ├─────────────────>│
     │                 │                  │
     │                 │  Deactivate      │
     │                 │  (Active=false)  │
     │                 ├─────────────────>│
     │                 │                  │
     │                 │  Revoke refresh  │
     │                 │  tokens          │
     │                 ├─────────────────>│
     │                 │                  │
     │  Clear cookie + │                  │
     │  redirect or    │                  │
     │  show logout    │                  │
     │  confirmation   │                  │
     │<────────────────│                  │
     │                 │                  │
```

### Implementation Details/Notes/Constraints

#### Feature Flag

```go
// pkg/featureflags/set.go
var (
    // ...existing flags...

    // SessionsEnabled enables user sessions feature
    SessionsEnabled = newFlag("sessions_enabled", false)
)
```

#### New Storage Entities


Two entities are required to properly handle the case where a user might be logged into different clients as different identities in the same browser:

###### AuthSession

```go
// storage/storage.go

// AuthSession represents a browser's authentication state.
// One per browser, referenced by session cookie.
// Key: SessionID (random 32-byte string, stored in cookie)
type AuthSession struct {
    // ID is the session identifier stored in cookie
    ID string

    // ClientStates maps clientID → authentication state for that client
    // Allows different users/identities per client in same browser
    //
    // Design note: This map-based approach is consistent with how OfflineSessions
    // stores refresh tokens per client (OfflineSessions.Refresh map). Given that
    // the number of OAuth clients in a typical deployment is bounded and relatively
    // small (tens to hundreds, not thousands), the serialized size of this map
    // will not exceed practical storage limits for any supported backend.
    ClientStates map[string]*ClientAuthState

    // CreatedAt is when this browser session started
    CreatedAt time.Time

    // LastActivity is when any client was last accessed
    LastActivity time.Time

    // IPAddress at session creation (for audit)
    IPAddress string

    // UserAgent at session creation (for audit)
    UserAgent string
}

// ClientAuthState represents authentication state for a specific client within an auth session.
// Expiration follows OIDC conventions with both absolute and idle timeout:
// - ExpiresAt enforces absolute lifetime (sessions.absoluteLifetime)
// - LastActivity + sessions.validIfNotUsedFor enforces idle timeout
// A client state is considered expired if EITHER condition is met.
type ClientAuthState struct {
    // UserID + ConnectorID identify which UserIdentity is authenticated for this client
    UserID      string
    ConnectorID string

    // Active indicates if authentication is active for this client
    Active bool

    // ExpiresAt is the absolute expiration time for this client session.
    // Set to time.Now() + absoluteLifetime at session creation.
    // Cannot be extended - hard upper bound on session duration.
    ExpiresAt time.Time

    // LastActivity is when this client session was last used (token issued, SSO check, etc.)
    // Used with validIfNotUsedFor to enforce idle timeout.
    // Updated on each request that touches this client state.
    LastActivity time.Time

    // LastTokenIssuedAt is when a token was last issued for this client.
    // Used for logout notifications and audit.
    LastTokenIssuedAt time.Time
}
```

###### UserIdentity

```go
// storage/storage.go

// UserIdentity represents a user's persistent identity data.
// Stores data that persists across sessions:
// - Consent decisions
// - Future: 2FA enrollment
//
// Key: composite of UserID + ConnectorID (one per user per connector)
type UserIdentity struct {
    // UserID is the subject identifier from the connector
    UserID string

    // ConnectorID is the connector that authenticated the user
    ConnectorID string

    // Claims holds the user's identity claims
    // Updated on:
    // 1. Each login (from connector callback)
    // 2. Each refresh token usage (from RefreshConnector.Refresh)
    // This ensures claims stay in sync with OfflineSessions and upstream IDP
    Claims Claims

    // Consents stores user consent per client: map[clientID][]scopes
    // Persists across sessions so user doesn't need to re-consent
    Consents map[string][]string

    // CreatedAt is when this identity was first created
    CreatedAt time.Time

    // LastLogin is when the user last authenticated (used for auth_time claim)
    LastLogin time.Time

    // BlockedUntil is set when user is blocked from logging in
    BlockedUntil time.Time

    // Future: 2FA fields
    // TOTPSecret string
    // WebAuthnCredentials []WebAuthnCredential
}
```

**Two-Entity Design Rationale**

| Entity | Purpose | Lifecycle | Key |
|--------|---------|-----------|-----|
| AuthSession | Browser binding, per-client auth state | Short-lived (session timeout) | SessionID (cookie) |
| UserIdentity | User data, consents, 2FA | Long-lived (persists) | UserID + ConnectorID |

**How It Works: Different Users in Different Clients**

```
Auth Session (cookie: dex_session=abc123)
├── ClientStates["client-A"]:
│   └── UserID: "alice", ConnectorID: "google", Active: true
├── ClientStates["client-B"]:
│   └── UserID: "bob", ConnectorID: "ldap", Active: true
└── ClientStates["client-C"]:
    └── (empty - never authenticated)

UserIdentity (alice + google):
├── Claims: {email: alice@example.com, ...}
├── Consents: {"client-A": ["openid", "email"]}
└── LastLogin: 2024-01-01

UserIdentity (bob + ldap):
├── Claims: {email: bob@corp.com, ...}
├── Consents: {"client-B": ["openid", "groups"]}
└── LastLogin: 2024-01-02
```

**How SSO Works**

When user accesses client-B with existing session:

1. Get `AuthSession` by cookie
2. Check `ClientStates["client-B"]`:
   - If exists and active → user already authenticated for this client
3. If not, check SSO:
   - Find any `ClientStates[X]` where client-X has `ssoSharedWith` containing "client-B"
   - If found → SSO! Copy auth state to `ClientStates["client-B"]`
   - If not found → require authentication

**SSO Session Lookup Algorithm**

```go
// findSSOSession searches for a valid SSO source session for the target client
func (s *Server) findSSOSession(authSession *AuthSession, targetClientID string) (*ClientAuthState, *UserIdentity) {
    targetClient, err := s.storage.GetClient(ctx, targetClientID)
    if err != nil {
        return nil, nil
    }

    // Iterate through all active client states in this browser session
    for sourceClientID, state := range authSession.ClientStates {
        // Skip inactive or expired states
        if !state.Active || time.Now().After(state.ExpiresAt) {
            continue
        }

        // Get the source client configuration
        sourceClient, err := s.storage.GetClient(ctx, sourceClientID)
        if err != nil {
            continue
        }

        // Check if source client shares its session with the target client
        // SSO is allowed if:
        // 1. Source client has ssoSharedWith: ["*"] (shares with everyone)
        // 2. Source client has targetClientID in its ssoSharedWith list
        if !s.clientSharesSessionWith(sourceClient, targetClientID) {
            continue
        }

        // Found a valid SSO source! Get the user identity
        identity, err := s.storage.GetUserIdentity(ctx, state.UserID, state.ConnectorID)
        if err != nil {
            continue
        }

        // Check if user is not blocked
        if identity.BlockedUntil.After(time.Now()) {
            continue
        }

        return state, identity
    }

    return nil, nil
}

// clientSharesSessionWith checks if sourceClient shares its session with targetClientID
func (s *Server) clientSharesSessionWith(sourceClient Client, targetClientID string) bool {
    for _, peer := range sourceClient.SSOSharedWith {
        if peer == "*" || peer == targetClientID {
            return true
        }
    }
    return false
}
```

**SSO Lookup Flow Diagram**

```
User accesses client-B with existing session
                │
                ▼
┌─────────────────────────────────┐
│ Get AuthSession from cookie     │
└─────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────┐
│ Check ClientStates["client-B"]  │
│ exists and active?              │
└─────────────────────────────────┘
        │               │
       Yes              No
        │               │
        ▼               ▼
┌──────────────┐  ┌─────────────────────────────────┐
│ Use existing │  │ For each ClientStates[X]:       │
│ session      │  │   - Is state active?            │
└──────────────┘  │   - Get client-X config         │
                  │   - Does client-X share with B? │
                  │     (X.ssoSharedWith has B or *)│
                  └─────────────────────────────────┘
                          │               │
                    Found match       No match
                          │               │
                          ▼               ▼
                  ┌──────────────┐  ┌──────────────┐
                  │ SSO! Copy    │  │ Require      │
                  │ state to B   │  │ authentication│
                  └──────────────┘  └──────────────┘
```

**Example: SSO Flow**

```
1. User logs into client-A as alice
   AuthSession.ClientStates["client-A"] = {UserID: "alice", Active: true}

2. User accesses client-B
   - client-A.ssoSharedWith includes "client-B" ✓
   - SSO! Copy: ClientStates["client-B"] = {UserID: "alice", Active: true}
   - Issue tokens for alice to client-B
```

**Example: No SSO, Different User**

```
1. User logged into client-A as alice
   AuthSession.ClientStates["client-A"] = {UserID: "alice", Active: true}

2. User accesses client-B (client-A does NOT share with client-B)
   - No SSO available
   - Redirect to connector for authentication

3. User logs in as bob (different account)
   AuthSession.ClientStates["client-B"] = {UserID: "bob", Active: true}

Same browser, two different users, no conflict!
```

**Claims Synchronization with Refresh Tokens**

When a refresh token is used:
1. `RefreshConnector.Refresh()` returns updated claims
2. Update `OfflineSessions.ConnectorData` (existing behavior)
3. **NEW**: Also update `UserIdentity.Claims`:

```go
// In refresh token handler
func (s *Server) handleRefreshToken(...) {
    // ...existing refresh logic...

    newIdentity, err := refreshConn.Refresh(ctx, scopes, oldIdentity)
    if err != nil {
        // Handle refresh failure
    }

    // Update OfflineSessions (existing)
    s.storage.UpdateOfflineSessions(...)

    // Update UserIdentity claims (NEW)
    if s.sessionsEnabled {
        s.storage.UpdateUserIdentity(ctx, newIdentity.UserID, connectorID,
            func(u UserIdentity) (UserIdentity, error) {
                u.Claims = storage.Claims{
                    UserID:    newIdentity.UserID,
                    Username:  newIdentity.Username,
                    Email:     newIdentity.Email,
                    Groups:    newIdentity.Groups,
                    // ...
                }
                return u, nil
            })
    }
}
```

This ensures `UserIdentity.Claims` stays synchronized with:
- Connector's current user data
- `OfflineSessions.ConnectorData`
- Actual refresh token claims

**Why UserIdentity instead of AuthSession?**

The name `UserIdentity` is chosen because this entity stores more than just session state:
1. **Persistent data**: Consent decisions survive session expiration
2. **Future 2FA**: TOTP secrets and WebAuthn credentials will be stored here
3. **One per user/connector**: Unlike sessions which could be per-browser, this is per-identity

**Session ID Regeneration**

The `AuthSession.ID` is regenerated when:
- User logs in from a new browser (new session created)
- Security concern requires new session (e.g., after password change)

Individual `ClientStates` can be invalidated without changing the auth session ID.

**Multiple Users in Same Browser**

With the two-entity design:
- `AuthSession` tracks which user is authenticated for which client
- Different clients can have different users (if no SSO trust)
- Same user can be authenticated for multiple clients (SSO or separate logins)

**SSO and Different Users**

With SSO enabled between clients, the same user is used for all sharing clients:
- User logs in to client-A as "alice@example.com"
- User accesses client-B (client-A shares with client-B) → automatically authenticated as "alice@example.com"
- SSO reuses the identity from the sharing client

If user needs to login as different identity to a sharing client:
- Use `prompt=login` to force re-authentication
- This creates new ClientState for that client with potentially different user

Without SSO, user can be different identities in different clients (see examples above).

#### Storage Interface Extensions

Two new entities require CRUD operations:

```go
// storage/storage.go

type Storage interface {
    // ...existing methods...

    // AuthSession management
    CreateAuthSession(ctx context.Context, s AuthSession) error
    GetAuthSession(ctx context.Context, sessionID string) (AuthSession, error)
    UpdateAuthSession(ctx context.Context, sessionID string, updater func(s AuthSession) (AuthSession, error)) error
    DeleteAuthSession(ctx context.Context, sessionID string) error

    // UserIdentity management
    CreateUserIdentity(ctx context.Context, u UserIdentity) error
    GetUserIdentity(ctx context.Context, userID, connectorID string) (UserIdentity, error)
    UpdateUserIdentity(ctx context.Context, userID, connectorID string, updater func(u UserIdentity) (UserIdentity, error)) error
    DeleteUserIdentity(ctx context.Context, userID, connectorID string) error

    // List for admin API
    ListUserIdentities(ctx context.Context) ([]UserIdentity, error)
}
```

**Garbage Collection**

```go
type GCResult struct {
    // ...existing fields...
    AuthSessions int64  // NEW: expired auth sessions cleaned up
}
```

`AuthSession` objects are garbage collected when:
- `LastActivity + validIfNotUsedFor` exceeded (inactivity)
- All `ClientStates` have expired

`UserIdentity` objects are NOT garbage collected (preserve consents, future 2FA).

#### Session Expiration

**AuthSession expiration:**
- Entire session expires when `LastActivity + validIfNotUsedFor` is reached (idle timeout)
- On expiration, `AuthSession` is deleted by GC
- User must re-authenticate for all clients

**ClientAuthState expiration (per-client within AuthSession):**

Each client state enforces **both** absolute lifetime and idle timeout, consistent with standard OIDC session semantics:

```go
func (s *Server) isClientStateValid(state *ClientAuthState) bool {
    now := time.Now()

    // 1. Check absolute lifetime - hard upper bound, cannot be extended
    if now.After(state.ExpiresAt) {
        return false
    }

    // 2. Check idle timeout - session unused for too long
    if now.After(state.LastActivity.Add(s.sessionsConfig.validIfNotUsedFor)) {
        return false
    }

    // 3. Check explicit deactivation (admin revoked)
    if !state.Active {
        return false
    }

    return true
}
```

When a client state expires:
- Other clients in same auth session remain active
- User must re-authenticate only for the expired client
- On successful re-authentication, a new `ClientAuthState` is created with fresh `ExpiresAt`

**Admin can force re-authentication:**
- Delete `AuthSession` → user must re-auth for all clients
- Set `ClientStates[clientID].Active = false` → user must re-auth for that client only

#### Deletion Risks

**Deleting AuthSession:**
- User must re-authenticate for all clients
- No data loss (consents preserved in UserIdentity)
- Safe operation for logout

**Deleting UserIdentity:**

| What's Lost | Impact |
|-------------|--------|
| Consent decisions | User must re-approve scopes for all clients |
| Future: 2FA enrollment | User must re-enroll TOTP/WebAuthn |

**When to delete UserIdentity:**
- User explicitly requests account deletion (GDPR)
- Admin cleanup of stale identities
- User removed from upstream identity provider

**When NOT to delete (delete AuthSession instead):**
- Regular logout - delete AuthSession or set ClientState.Active = false
- Session expiration - GC handles AuthSession cleanup
- Security concern - delete AuthSession to force re-auth

#### Session Cookie Format

The session cookie contains only the session ID (not the session data):

```
Cookie: dex_session=<session_id>; Path=<issuer_path>; Secure; HttpOnly; SameSite=Lax
```

**Cookie Path**: Derived from the issuer URL path (`issuerURL.Path`). For example:
- Issuer: `https://dex.example.com/` → `Path=/`
- Issuer: `https://example.com/dex` → `Path=/dex`

This is consistent with how Dex already handles routing - all endpoints are prefixed with the issuer path.

**Session Creation vs Cookie Persistence (Keycloak-like behavior)**

Unlike some implementations where "Remember Me" controls session creation, we follow Keycloak's approach:

- **AuthSession is ALWAYS created** on successful authentication
- **"Remember Me" controls cookie persistence**:
  - Unchecked: Session cookie (expires when browser closes)
  - Checked: Persistent cookie (expires at `absoluteLifetime`)

This approach is better because:
1. SSO works within a browser session even without "Remember Me"
2. Consent decisions are preserved during the browser session
3. `prompt=none` works correctly within browser session
4. More intuitive: "Remember Me" = "remember me after I close the browser"

```go
func (s *Server) setSessionCookie(w http.ResponseWriter, sessionID string, rememberMe bool) {
    cookie := &http.Cookie{
        Name:     s.sessionsConfig.CookieName,
        Value:    sessionID,
        Path:     s.issuerURL.Path,
        HttpOnly: true,
        Secure:   s.issuerURL.Scheme == "https",
        SameSite: http.SameSiteLaxMode,
    }

    if rememberMe {
        // Persistent cookie - survives browser restart
        cookie.MaxAge = int(s.sessionsConfig.absoluteLifetime.Seconds())
    }
    // else: Session cookie - no MaxAge, browser deletes on close

    http.SetCookie(w, cookie)
}
```

Session ID generation:
```go
func NewSessionID() string {
    return newSecureID(32) // 256-bit random value
}
```

#### Client Configuration Extension

A new client configuration field is introduced for SSO control:

```go
// storage/storage.go

type Client struct {
    // ...existing fields...

    // TrustedPeers are a list of peers which can issue tokens on this client's behalf.
    // This is used for cross-client token issuance (existing behavior).
    TrustedPeers []string `json:"trustedPeers" yaml:"trustedPeers"`

    // SSOSharedWith defines which other clients can reuse this client's authentication session.
    // When a user is authenticated for this client, clients listed here can skip authentication.
    // This is separate from TrustedPeers - organizations may want different policies for
    // session sharing vs token delegation.
    // Special value "*" means share with all clients (Keycloak-like realm-wide SSO).
    // nil means use ssoSharedWithDefault from sessions config.
    // Empty slice [] means explicitly share with no one.
    SSOSharedWith []string `json:"ssoSharedWith,omitempty" yaml:"ssoSharedWith,omitempty"`
}
```

#### Connector Logout (Future)

Logout URLs should be configured on connectors, not clients. A new connector interface will be added:

```go
// connector/connector.go

// LogoutConnector is an optional interface for connectors that support
// terminating upstream sessions on logout.
type LogoutConnector interface {
    // Logout terminates the user's session at the upstream identity provider.
    // Returns a URL to redirect the user to for upstream logout, or empty string
    // if no redirect is needed.
    Logout(ctx context.Context, connectorData []byte) (logoutURL string, err error)
}
```

Connectors that implement this interface (e.g., OIDC with `end_session_endpoint`, SAML with SLO):
- Are called during Dex logout flow
- Can redirect user to upstream for complete logout
- Implementation details are connector-specific

This is tracked as a future improvement.

#### Server Configuration Extension

```go
// cmd/dex/config.go

type Sessions struct {
    // CookieName is the session cookie name (default: "dex_session")
    CookieName string `json:"cookieName"`

    // AbsoluteLifetime is the maximum session lifetime (default: "24h")
    AbsoluteLifetime string `json:"absoluteLifetime"`

    // ValidIfNotUsedFor is the inactivity timeout (default: "1h")
    ValidIfNotUsedFor string `json:"validIfNotUsedFor"`

    // SSOSharedWithDefault is the default SSO sharing policy
    // "all" = share with all clients, "none" = share with no one (default: "none")
    SSOSharedWithDefault string `json:"ssoSharedWithDefault"`

    // RememberMeCheckedByDefault controls the initial checkbox state in templates
    // true = pre-checked, false = unchecked (default: false)
    RememberMeCheckedByDefault bool `json:"rememberMeCheckedByDefault"`
}
```

**Using ssoSharedWithDefault in SSO logic:**

```go
func (s *Server) clientSharesSessionWith(sourceClient Client, targetClientID string) bool {
    ssoSharedWith := sourceClient.SSOSharedWith

    // If client has no explicit ssoSharedWith, use default
    if ssoSharedWith == nil {
        switch s.sessionsConfig.SSOSharedWithDefault {
        case "all":
            return true // Share with everyone by default
        default: // "none"
            return false // Share with no one by default
        }
    }

    // Explicit configuration: empty slice means explicitly share with no one
    // This is different from nil (not configured)
    if len(ssoSharedWith) == 0 {
        return false
    }

    // Check explicit sharing list
    for _, peer := range ssoSharedWith {
        if peer == "*" || peer == targetClientID {
            return true
        }
    }
    return false
}
```

**Three states for ssoSharedWith:**
1. `nil` (not configured) → use `ssoSharedWithDefault`
2. `[]` (empty slice) → explicitly share with no one
3. `["client-a", ...]` or `["*"]` → explicit sharing list

#### Prompt Parameter Handling

Dex will support the following `prompt` values per OIDC Core specification:
- `none` - Silent authentication, no UI displayed
- `login` - Force re-authentication
- `consent` - Force consent screen
- Empty (default) - Normal flow with session reuse

The `select_account` value is not supported initially (would require account linking feature).

```go
func (s *Server) handleAuthorization(w http.ResponseWriter, r *http.Request) {
    // ...existing parsing...

    prompt := r.Form.Get("prompt")
    maxAge := r.Form.Get("max_age")
    idTokenHint := r.Form.Get("id_token_hint")
    clientID := r.Form.Get("client_id")

    // Get auth session from cookie
    authSession, err := s.getAuthSessionFromCookie(r)

    // Get client auth state for this specific client
    var clientState *ClientAuthState
    var userIdentity *UserIdentity
    if authSession != nil {
        clientState = authSession.ClientStates[clientID]
        if clientState != nil && clientState.Active {
            userIdentity, _ = s.storage.GetUserIdentity(ctx, clientState.UserID, clientState.ConnectorID)
        }
    }

    // Handle max_age parameter (OIDC Core 3.1.2.1)
    if maxAge != "" && userIdentity != nil {
        maxAgeSeconds, err := strconv.Atoi(maxAge)
        if err == nil && maxAgeSeconds >= 0 {
            authAge := time.Since(userIdentity.LastLogin)
            if authAge > time.Duration(maxAgeSeconds)*time.Second {
                // Session is too old, force re-authentication
                clientState = nil
                userIdentity = nil
            }
        }
    }

    switch prompt {
    case "none":
        // Silent authentication - must have valid session and consent
        if clientState == nil || userIdentity == nil {
            s.authErr(w, r, redirectURI, "login_required", state)
            return
        }
        // Check consent in identity
        consentedScopes, hasConsent := userIdentity.Consents[clientID]
        if !hasConsent || !s.scopesCovered(consentedScopes, requestedScopes) {
            s.authErr(w, r, redirectURI, "consent_required", state)
            return
        }
        // Issue tokens without UI

    case "login":
        // Force re-authentication - ignore existing session for this client
        clientState = nil
        userIdentity = nil
        // Continue to connector login

    case "consent":
        // Force consent screen even if previously consented
        // Continue but don't check consent

    default: // "" - normal flow
        // Check for SSO from trusted clients if no direct session
        if clientState == nil && authSession != nil {
            clientState, userIdentity = s.findSSOSession(authSession, clientID)
        }
    }

    // Validate id_token_hint if provided
    if idTokenHint != "" {
        claims, err := s.validateIDTokenHint(idTokenHint)
        if err != nil {
            s.authErr(w, r, redirectURI, "invalid_request", state)
            return
        }
        if userIdentity != nil && userIdentity.UserID != claims.Subject {
            // Identity user doesn't match hint
            if prompt == "none" {
                s.authErr(w, r, redirectURI, "login_required", state)
                return
            }
            // Force re-login for different user
            clientState = nil
            userIdentity = nil
        }
    }

    // ...continue with flow...
}

// findSSOSession looks for a valid SSO session from a sharing client
func (s *Server) findSSOSession(authSession *AuthSession, targetClientID string) (*ClientAuthState, *UserIdentity) {
    for sourceClientID, state := range authSession.ClientStates {
        if !state.Active {
            continue
        }
        sourceClient, _ := s.storage.GetClient(ctx, sourceClientID)
        if sourceClient == nil {
            continue
        }
        // Check if source client shares its session with target client
        if s.clientSharesSessionWith(sourceClient, targetClientID) {
            identity, _ := s.storage.GetUserIdentity(ctx, state.UserID, state.ConnectorID)
            if identity != nil {
                return state, identity
            }
        }
    }
    return nil, nil
}
```

**max_age Parameter**

The `max_age` parameter is supported per OIDC Core specification:
- Specifies the maximum authentication age in seconds
- If the identity's last authentication time (`LastLogin`) exceeds `max_age`, force re-authentication
- When `max_age` is used, the `auth_time` claim MUST be included in the ID token

#### New Endpoints

```
POST /logout
GET  /logout
```

Logout endpoint following RP-Initiated Logout spec ([RFC draft](https://openid.net/specs/openid-connect-rpinitiated-1_0.html)):

```go
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
    idTokenHint := r.FormValue("id_token_hint")
    postLogoutRedirectURI := r.FormValue("post_logout_redirect_uri")
    state := r.FormValue("state")
    clientID := r.FormValue("client_id") // Optional: logout from specific client

    // Get auth session from cookie
    authSession, _ := s.getAuthSessionFromCookie(r)

    // Validate id_token_hint if provided
    var hintUserID, hintConnectorID string
    if idTokenHint != "" {
        claims, err := s.validateIDTokenHint(idTokenHint)
        if err == nil {
            hintUserID = claims.Subject
            // Extract connector from token if possible
        }
    }

    if authSession != nil {
        if clientID != "" {
            // Logout from specific client only
            delete(authSession.ClientStates, clientID)
            s.storage.UpdateAuthSession(ctx, authSession.ID, ...)
        } else {
            // Logout from all clients - delete entire auth session
            s.storage.DeleteAuthSession(ctx, authSession.ID)
        }

        // Revoke refresh tokens for logged-out clients
        // ...
    }

    // Clear cookie and redirect
    s.clearSessionCookie(w)

    // Show logout confirmation or redirect
    if postLogoutRedirectURI != "" && s.isValidPostLogoutURI(postLogoutRedirectURI, idTokenHint) {
        u, _ := url.Parse(postLogoutRedirectURI)
        if state != "" {
            q := u.Query()
            q.Set("state", state)
            u.RawQuery = q.Encode()
        }
        http.Redirect(w, r, u.String(), http.StatusFound)
        return
    }

    // Show logout confirmation page
    s.templates.logout(w, r)
}
```

**Future: Upstream Connector Logout**

For CallbackConnectors (OIDC, OAuth, SAML), the upstream identity provider may also have an active session. Future work should include:
- Implement `LogoutConnector` interface (see above)
- OIDC connectors use `end_session_endpoint` from discovery
- SAML connectors use Single Logout (SLO)
- Redirect user to upstream after Dex logout

This is tracked as a future improvement.

#### Discovery Updates

```go
func (s *Server) constructDiscovery(ctx context.Context) discovery {
    d := discovery{
        // ...existing fields...
    }

    if s.sessionsEnabled {
        d.EndSessionEndpoint = s.absURL("/logout")
    }

    return d
}
```

#### Login Template Updates

When sessions are enabled, add "Remember Me" checkbox to authentication flow.

**Template Data**

The server passes these values to templates:

```go
type templateData struct {
    // ...existing fields...

    // SessionsEnabled indicates if sessions feature is active
    SessionsEnabled bool

    // RememberMeChecked is the default checkbox state
    // Set from config: sessions.rememberMeCheckedByDefault
    RememberMeChecked bool
}
```

**For PasswordConnector (login form exists in Dex):**

```html
<!-- templates/password.html -->
<form method="post">
    <!-- existing fields -->

    {{ if .SessionsEnabled }}
    <div class="remember-me">
        <input type="checkbox" id="remember_me" name="remember_me" value="true"
               {{ if .RememberMeChecked }}checked{{ end }}>
        <label for="remember_me">Remember me</label>
    </div>
    {{ end }}

    <button type="submit">Login</button>
</form>
```

**For CallbackConnector (no login form in Dex):**

For OAuth/OIDC/SAML connectors, the user is redirected to upstream IDP and there's no Dex login form.

**Show on Approval Page** (recommended): Add "Remember Me" checkbox to the approval/consent page. User sees it after returning from upstream IDP, before granting consent.

```html
<!-- templates/approval.html -->
<form method="post">
    <!-- existing scope approval fields -->

    {{ if .SessionsEnabled }}
    <div class="remember-me">
        <input type="checkbox" id="remember_me" name="remember_me" value="true"
               {{ if .RememberMeChecked }}checked{{ end }}>
        <label for="remember_me">Remember me on this device</label>
    </div>
    {{ end }}

    <button type="submit" name="approval" value="approve">Grant Access</button>
</form>
```

**When skipApprovalScreen is true**: If approval screen is skipped, the `rememberMeCheckedByDefault` config determines cookie persistence:
- `false` (default): Session cookie (deleted on browser close)
- `true`: Persistent cookie (survives browser restart)

**Remember Me Behavior** (Keycloak-like):
- **AuthSession is ALWAYS created** on successful authentication regardless of checkbox
- **Checkbox controls cookie persistence only**:
  - **Unchecked**: Session cookie - expires when browser closes. SSO works within browser session.
  - **Checked**: Persistent cookie - survives browser restart until `absoluteLifetime` expires.

#### Connector Type Considerations

**CallbackConnector** (OIDC, OAuth, SAML, GitHub, etc.):
- Session created after successful callback
- Upstream tokens stored in refresh token's ConnectorData (not in session)
- Identity refresh via RefreshConnector when refresh token is used

**PasswordConnector** (LDAP, local passwords):
- Session created after successful password verification
- No upstream tokens
- Identity refresh re-validates against password backend when refresh token is used

Both types work the same way with sessions - the connector type only affects:
1. Initial authentication flow (redirect vs password form)
2. How identity refresh works (via refresh tokens, not sessions)

#### Connector Configuration Changes

Sessions reference a `ConnectorID`, but connector configuration may change after session creation (e.g., OIDC issuer URL changes, LDAP server replaced, connector removed entirely).

**Behavior**: Dex does NOT automatically invalidate sessions when connector configuration changes. This is by design - Dex has no mechanism to detect configuration changes at runtime, and connectors are typically reconfigured during planned maintenance.

**Administrator responsibility**: When connector configuration changes in a way that invalidates existing user identities (e.g., connector removed, upstream IdP replaced), administrators should:
1. Terminate affected sessions via gRPC admin API (future: `DexSessions.TerminateByConnector(connectorID)`)
2. Or wait for sessions to expire naturally
3. Or restart Dex with `DEX_SESSIONS_ENABLED=false` temporarily to force re-authentication

If a session references a connector that no longer exists, the session will fail gracefully at the next use: `GetConnector()` will return an error, and the user will be redirected to authenticate again.

### Risks and Mitigations

#### Security Risks

| Risk | Mitigation |
|------|------------|
| Session hijacking | Secure cookie flags (HttpOnly, Secure, SameSite), short idle timeout |
| Session fixation | Generate new session ID after authentication (see below) |
| CSRF on logout | GET shows confirmation page, POST performs logout |
| Cookie theft | Bind session to fingerprint (IP range, partial user agent) - optional |
| Storage exposure | Session IDs are random 256-bit values, no sensitive data in cookie |

**Session Fixation Protection**

Session fixation attacks occur when an attacker sets a known session ID in a victim's browser before authentication, then hijacks the session after the victim logs in.

References:
- [OWASP Session Fixation](https://owasp.org/www-community/attacks/Session_fixation)
- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)

**Mitigations implemented:**

1. **Regenerate session ID on authentication**: When a user successfully authenticates, ALWAYS generate a new `AuthSession.ID` even if a session already exists. Never reuse a pre-authentication session ID.

```go
// This is not the real method signature, but the implementation example of a specific behavior.
func (s *Server) onSuccessfulAuthentication(w http.ResponseWriter, userID, connectorID, clientID string, rememberMe bool) {
    // ALWAYS generate new session ID - prevents session fixation
    newSessionID := NewSessionID()

    // Create or update AuthSession with NEW ID
    authSession := &AuthSession{
        ID:           newSessionID,  // Always new, never reuse
        ClientStates: make(map[string]*ClientAuthState),
        CreatedAt:    time.Now(),
        // ...
    }

    // Set cookie with new session ID
    s.setSessionCookie(w, newSessionID, rememberMe)
}
```

2. **Don't accept session IDs from URL parameters**: Session IDs are ONLY accepted from cookies, never from query parameters or POST data.

3. **Strict cookie settings**: `HttpOnly`, `Secure`, `SameSite=Lax` prevent common session theft vectors.

4. **Session binding (optional future enhancement)**: Bind session to client characteristics (IP range, user agent) to detect stolen cookies.

**Handling existing sessions during authentication:**

When a user authenticates and an existing `AuthSession` is found:
1. Generate a completely new session ID
2. Copy relevant state from old session to new session (if any)
3. Delete the old `AuthSession` from storage
4. Set cookie with new session ID

This ensures that even if an attacker set a session cookie before authentication, they cannot use it after the victim logs in.

#### Operational Risks

| Risk | Mitigation |
|------|------------|
| Storage growth | AuthSessions are GC'd on inactivity; UserIdentities are per-user like OfflineSessions; admin API allows cleanup |
| Storage performance | Additional read per request to resolve session cookie. Impact depends on backend — see note below |
| Migration complexity | Feature flag allows gradual rollout, no breaking changes |

**Storage Performance Note**

Enabling sessions introduces an additional storage read on each authorization request (to resolve the session cookie to an `AuthSession`). The actual performance impact depends on the storage backend:

- **SQL (Postgres, MySQL, SQLite)**: Session lookup by primary key is a single indexed read — negligible overhead
- **etcd**: Single key-value lookup — negligible overhead
- **Kubernetes CRDs**: GET by resource name — slightly higher latency than SQL/etcd but still within acceptable bounds (may require [priority&fairness](https://kubernetes.io/docs/concepts/cluster-administration/flow-control/) tuning)
- **Memory**: In-process map lookup — no overhead

At this stage, we do not have production metrics to quantify the exact impact. The storage access pattern is identical to existing `OfflineSessions` lookups (single record by key), which are already proven in production. It is recommended to monitor storage latency after enabling sessions and adjusting `validIfNotUsedFor` if the GC frequency needs tuning.

#### Breaking Changes

**None** - Sessions are opt-in via feature flag and configuration. Existing deployments continue to work without changes.

#### Rollback Plan

Sessions are fully controlled by the `DEX_SESSIONS_ENABLED` feature flag. Rollback is straightforward:

1. **Disable feature flag**: Set `DEX_SESSIONS_ENABLED=false` (or remove it)
2. **Immediate effect**: Dex stops creating, reading, and validating sessions. All authorization requests proceed as before sessions were introduced — connector authentication on every request, no SSO, no session cookies
3. **Cookie cleanup**: Existing session cookies in browsers become inert — Dex ignores them when sessions are disabled. They expire naturally per their MaxAge or when the browser is closed
4. **Storage cleanup**: `AuthSession` and `UserIdentity` records remain in storage but are unused. They can be cleaned up manually or left to accumulate no further growth
5. **No downtime required**: Feature flag can be toggled without restart if environment variable reload is supported; otherwise, a rolling restart is sufficient

**Key guarantee**: Disabling the feature flag returns Dex to its pre-sessions behavior with zero side effects. No existing functionality (refresh tokens, connector authentication, token issuance) depends on sessions. Additional tables in the database const nothing and can be deleted later.

#### Migration Path

1. Deploy new Dex version - storage migrations create `AuthSession` and `UserIdentity` tables/resources automatically (no feature flag needed for schema)
2. Enable feature flag `DEX_SESSIONS_ENABLED=true` when ready to use sessions
3. Add `sessions:` configuration block
4. Sessions start being created for all new logins; "Remember Me" controls cookie persistence (session vs persistent cookie)
5. Existing refresh tokens continue to work

**Note**: Storage schema changes (new tables/CRDs) are applied on startup regardless of feature flag. The feature flag only controls whether sessions are actually created and used. This simplifies deployment - you can deploy the new version, then enable sessions later without another deployment.

### Alternatives

#### 1. Stateless Sessions (JWT in Cookie)

**Approach**: Store session data directly in a signed/encrypted JWT cookie.

**Pros**:
- No server-side storage required
- Scales horizontally without shared state

**Cons**:
- Cannot revoke sessions without blocklist
- Cookie size limits (~4KB)
- Cannot store consent history or client tracking for logout
- No server-side session list for logout

**Decision**: Rejected. Server-side sessions are required for proper logout and SSO.

#### 2. Extend OfflineSessions

**Approach**: Add session data to existing OfflineSessions entity.

**Pros**:
- Reuses existing storage
- Simpler migration

**Cons**:
- OfflineSessions are per-connector, not per-browser
- Different lifecycle (refresh token vs browser session)
- Would complicate existing OfflineSessions logic

**Decision**: Rejected. Clean separation is better for maintainability.

#### 3. External Session Store (Redis)

**Approach**: Use Redis for session storage instead of existing backends.

**Pros**:
- Built-in TTL support
- Fast reads/writes
- Proven session store

**Cons**:
- Adds infrastructure dependency
- Against Dex's simplicity philosophy
- Doesn't work with Kubernetes CRD backend

**Decision**: Rejected. Must work with existing storage backends.

#### 4. Do Nothing

**Approach**: Keep using refresh tokens as implicit sessions.

**Cons**:
- Cannot implement OIDC conformance features
- No proper SSO
- No proper logout
- Blocks future features (2FA, etc.)

**Decision**: Rejected. These features are essential for enterprise adoption.

## Future Improvements

1. **Identity Refresh for Long-Lived Sessions**
   - Periodic refresh of user identity from connector during active session
   - Configurable refresh interval
   - Refresh on token request option
   - Handle connector revocation (terminate session)

2. **Upstream Connector Logout**
   - Redirect to upstream IDP logout endpoint after Dex logout
   - Support RP-Initiated Logout towards upstream OIDC providers
   - SAML Single Logout (SLO) support
   - Configurable per-connector logout URLs

3. **Session Introspection Endpoint**
   - Implement session check endpoint similar to [RFC 7662 Token Introspection](https://datatracker.ietf.org/doc/html/rfc7662)
   - Could enable replacing OAuth2 Proxy in some deployments
   - Endpoint: `GET /session/introspect` or similar
   - Returns session validity and user claims
   - Useful for reverse proxies to validate session cookies directly

4. **Front-Channel Logout**
   - Implement [OIDC Front-Channel Logout 1.0](https://openid.net/specs/openid-connect-frontchannel-1_0.html)
   - Notify client applications when user logs out via iframes
   - Requires client `logoutURL` configuration

5. **2FA/MFA Support**
   - Store TOTP secrets in user profile
   - Add MFA enrollment flow
   - Step-up authentication for sensitive operations
   - WebAuthn/Passkey support

6. **Session Management API**
   - List active sessions via gRPC API
   - Revoke sessions via gRPC API
   - Session activity audit log

7. **Back-Channel Logout**
   - Implement [OIDC Back-Channel Logout](https://openid.net/specs/openid-connect-backchannel-1_0.html)
   - Server-to-server logout notifications

8. **Account Linking**
   - Link multiple connector identities to single user
   - Switch between linked identities

9. **Device/Session Fingerprinting**
   - Optional session binding to client characteristics
   - Anomaly detection for session theft

10. **Per-Connector Session Policies**
    - Different session lifetimes per connector
    - Different SSO policies per connector

11. **Session Impersonation for Admin**
    - Admin can impersonate user sessions for debugging
    - Audit logging for impersonation

12. **Consent Management UI**
    - User-facing page to view/revoke consents
    - GDPR compliance features

