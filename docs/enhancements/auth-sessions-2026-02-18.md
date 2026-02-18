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
```

**SSO via TrustedPeers**: SSO between clients is controlled by the existing `trustedPeers` configuration on clients. The `trustedPeers` setting defines **which clients can USE this client's session**, not which clients this client can use.

If client B is listed in client A's `trustedPeers`:
1. Client B can issue tokens on behalf of client A (existing behavior)
2. If user logged in via client A, client B can reuse that session (new behavior)

This reuses existing semantics: "if you trust a peer to issue tokens for you, you also trust sharing your authentication state with them."

**Wildcard Support**: `trustedPeers: ["*"]` enables SSO with all clients. This is similar to Keycloak's default behavior where all clients in a realm share sessions.

**SSO Direction**: SSO trust is **unidirectional**. Client A trusting client B does NOT mean client B trusts client A.

```yaml
staticClients:
  # Public app - allows any client to reuse its sessions
  - id: public-app
    name: Public App
    trustedPeers: ["*"]
    # ...

  # Admin app - only specific apps can reuse its sessions
  - id: admin-app
    name: Admin App
    trustedPeers: ["monitoring-app"]  # Only monitoring can SSO from admin sessions
    # ...

  # Secret internal service - NO other clients can reuse its sessions
  - id: secret-service
    name: Secret Service
    trustedPeers: []  # Empty = no SSO allowed from this client's sessions
    # But this client CAN use sessions from other clients that trust it!
    # ...

  # Monitoring app - can SSO from admin-app (because admin-app trusts it)
  - id: monitoring-app
    name: Monitoring App
    trustedPeers: ["admin-app"]  # Bidirectional trust with admin-app
    # ...
```

**Example Scenarios:**

| User logged in via | Accessing | SSO works? | Why |
|-------------------|-----------|------------|-----|
| public-app | admin-app | ✅ Yes | public-app has `trustedPeers: ["*"]` |
| admin-app | public-app | ❌ No | admin-app only trusts monitoring-app |
| admin-app | monitoring-app | ✅ Yes | admin-app trusts monitoring-app |
| secret-service | any client | ❌ No | secret-service has `trustedPeers: []` |
| public-app | secret-service | ✅ Yes | public-app has `trustedPeers: ["*"]` |

**Key Insight**: A "secret" client that doesn't want others to SSO into it simply doesn't list them in `trustedPeers`. But it can still BENEFIT from SSO by being listed in OTHER clients' `trustedPeers`.

**Comparison with Keycloak**: In Keycloak, SSO is realm-wide by default - all clients in a realm share sessions. Dex's approach is more granular: SSO is opt-in per client via `trustedPeers`. Use `["*"]` to achieve Keycloak-like behavior.

**Cookie Security**: The session cookie is always set with secure defaults:
- `HttpOnly: true` - Not accessible via JavaScript
- `Secure: true` - Only sent over HTTPS (automatically disabled for localhost in dev)
- `SameSite: Lax` - CSRF protection
- `Path: /` - Available for all Dex endpoints

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
     │  (remember_me?) │                  │                   │
     ├────────────────>│                  │                   │
     │                 │                  │                   │
     │                 │  If remember_me: │                   │
     │                 │  Create/update   │                   │
     │                 │  UserIdentity    │                   │
     │                 ├─────────────────>│                   │
     │                 │                  │                   │
     │  Set-Cookie     │                  │                   │
     │  (if remember)  │                  │                   │
     │  + redirect to  │                  │                   │
     │  /approval      │                  │                   │
     │<────────────────│                  │                   │
     │                 │                  │                   │
```

**Note**: Session is only created/updated when "Remember Me" is checked. Without it, authentication proceeds as it does today (no persistent session).

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

One new entity is required, designed to work with KV backends (no foreign keys, no JOINs):

##### UserIdentity

```go
// storage/storage.go

// UserIdentity represents a user's persistent identity and authentication state.
// This entity stores:
// - Browser session state (active/inactive, expiration)
// - Consent decisions per client
// - Future: 2FA enrollment (TOTP secrets, WebAuthn credentials)
//
// Key: composite of UserID + ConnectorID (one identity per user per connector)
// The session cookie references this identity, but the ID is NOT the cookie value.
type UserIdentity struct {
    // UserID is the subject identifier from the connector
    UserID string

    // ConnectorID is the connector that authenticated the user
    ConnectorID string

    // SessionID is the current browser session ID (stored in cookie)
    // Regenerated on each new login for security (prevents session fixation)
    SessionID string

    // Claims holds the user's identity claims (refreshed on each login)
    Claims Claims

    // Consents stores user consent per client: map[clientID][]scopes
    // Persists across sessions so user doesn't need to re-consent
    Consents map[string][]string

    // Clients tracks which clients have received tokens from current session.
    // Used for:
    // 1. Logout notifications: Know which clients to notify on logout (future front-channel/back-channel logout)
    // 2. Session audit: Admin can see which apps user accessed in current session
    // 3. Token revocation: On logout, revoke refresh tokens only for clients in this session
    // Map structure: clientID → lastTokenIssuedAt
    // Cleared when session becomes inactive (logout or expiration)
    Clients map[string]time.Time

    // CreatedAt is when this identity was first created
    CreatedAt time.Time

    // LastLogin is when the user last authenticated (used for auth_time claim)
    LastLogin time.Time

    // LastActivity is when the session was last used
    LastActivity time.Time

    // SessionExpiresAt is when the current session expires
    SessionExpiresAt time.Time

    // Active indicates if there is a currently active session
    // false = user must re-authenticate to get a new SessionID
    // Admin can set this to false to force re-authentication
    Active bool

    // BlockedUntil is set when user is blocked from logging.
    // Used for manually blocking users from authenticating in the future.
    BlockedUntil time.Time

    // IPAddress is the client IP at last login (for audit)
    IPAddress string

    // UserAgent is the browser user agent at last login (for audit)
    UserAgent string

    // Future: 2FA fields to stora the data for the second factor
    // TOTPSecret string
    // WebAuthnCredentials []WebAuthnCredential
}
```

**Why UserIdentity instead of AuthSession?**

The name `UserIdentity` is chosen because this entity stores more than just session state:
1. **Persistent data**: Consent decisions survive session expiration
2. **Future 2FA**: TOTP secrets and WebAuthn credentials will be stored here
3. **One per user/connector**: Unlike sessions which could be per-browser, this is per-identity

**Session ID Regeneration**

The `SessionID` is regenerated on each new login:
- Prevents session fixation attacks
- New SessionID is set in cookie on successful authentication
- Old SessionID immediately becomes invalid
- If user logs in from multiple browsers, each login invalidates previous session

**Single Session per Identity**

Each `UserIdentity` has at most one active session (`SessionID`):
- User logs in from Browser A → gets SessionID-1
- User logs in from Browser B → gets SessionID-2, Browser A's session is invalidated
- This is simpler and more secure (no session sprawl)
- **Future:** Could add multi-session support if needed

**SSO and Different Users**

With SSO enabled between clients, the same user is used for all trusted clients:
- User logs in to client-A as "alice@example.com"
- User accesses client-B (trusted by client-A) → automatically authenticated as "alice@example.com"
- User cannot be "alice" in client-A and "bob" in client-B simultaneously in the same browser

If user needs to login as different identity to a trusted client:
- Use `prompt=login` to force re-authentication
- This starts a new session, logging out from previous identity in ALL trusted clients

This is standard OIDC SSO behavior (same as Keycloak, Okta, etc.).

#### Storage Interface Extensions

Standard CRUD operations are added to the storage interface:

```go
// storage/storage.go

type Storage interface {
    // ...existing methods...

    // UserIdentity management
    CreateUserIdentity(ctx context.Context, u UserIdentity) error
    GetUserIdentity(ctx context.Context, userID, connectorID string) (UserIdentity, error)
    GetUserIdentityBySessionID(ctx context.Context, sessionID string) (UserIdentity, error)
    UpdateUserIdentity(ctx context.Context, userID, connectorID string, updater func(u UserIdentity) (UserIdentity, error)) error
    DeleteUserIdentity(ctx context.Context, userID, connectorID string) error
}
```

#### Session Expiration

Sessions become inactive (`Active = false`) when:
- `SessionExpiresAt` (absolute lifetime) is reached
- `LastActivity + validIfNotUsedFor` is reached (inactivity timeout)
- Admin explicitly deactivates via API
- User logs out

The `Active` flag is checked on each request. If expired, it's set to `false` and user must re-authenticate.

**Why Active flag instead of just checking ExpiresAt?**
- Admin can force re-authentication by setting `Active = false`
- Allows "revoke session" without deleting the identity (preserves consents, future 2FA)
- Clear semantic: `Active = false` means "session ended, user must login again"

#### Deletion Risks

Deleting a `UserIdentity` has consequences:

| What's Lost | Impact                                      |
|-------------|---------------------------------------------|
| Consent decisions | User must re-approve scopes for all clients |
| Future: 2FA enrollment | The second factor will be regenerated       |
| Session tracking | No logout notification to clients           |

**When to delete:**
- User explicitly requests account deletion (GDPR)
- Admin cleanup of stale identities
- User removed from upstream identity provider

**When NOT to delete (deactivate instead):**
- Regular logout - set `Active = false`
- Session expiration - set `Active = false`
- Security concern - set `Active = false` to force re-auth or set `BlockedUntil` to block user from logging in for a period of time.

#### Session Cookie Format

The session cookie contains only the session ID (not the session data):

```
Cookie: dex_session=<session_id>; Path=/; Secure; HttpOnly; SameSite=Lax
```

Session ID generation:
```go
func NewSessionID() string {
    return newSecureID(32) // Same as existing NewID but 32 bytes
}
```

#### Client Configuration Extension

No new client configuration fields are required. SSO is controlled by the existing `trustedPeers` field:

```go
// storage/storage.go

type Client struct {
    // ...existing fields...

    // TrustedPeers are a list of peers which can issue tokens on this client's behalf.
    // When sessions are enabled, sessions can also be shared between trusted peers.
    // Special value "*" means trust all clients (Keycloak-like realm-wide SSO).
    TrustedPeers []string `json:"trustedPeers" yaml:"trustedPeers"`
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
}
```

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

    // Get identity from session cookie
    identity, err := s.getIdentityFromCookie(r)

    // Handle max_age parameter (OIDC Core 3.1.2.1)
    // max_age specifies maximum authentication age in seconds
    if maxAge != "" && identity != nil && identity.Active {
        maxAgeSeconds, err := strconv.Atoi(maxAge)
        if err == nil && maxAgeSeconds >= 0 {
            authAge := time.Since(identity.LastLogin)
            if authAge > time.Duration(maxAgeSeconds)*time.Second {
                // Session is too old, force re-authentication
                identity = nil
            }
        }
    }

    switch prompt {
    case "none":
        // Silent authentication - must have valid session and consent
        if err != nil || identity == nil || !identity.Active {
            s.authErr(w, r, redirectURI, "login_required", state)
            return
        }
        // Check consent in identity
        consentedScopes, hasConsent := identity.Consents[clientID]
        if !hasConsent || !s.scopesCovered(consentedScopes, requestedScopes) {
            s.authErr(w, r, redirectURI, "consent_required", state)
            return
        }
        // Issue tokens without UI

    case "login":
        // Force re-authentication - ignore existing session
        identity = nil
        // Continue to connector login

    case "consent":
        // Force consent screen even if previously consented
        // Continue but don't check session consent

    default: // "" - normal flow
        // Use session if available and active
    }

    // Validate id_token_hint if provided
    if idTokenHint != "" {
        claims, err := s.validateIDTokenHint(idTokenHint)
        if err != nil {
            s.authErr(w, r, redirectURI, "invalid_request", state)
            return
        }
        if identity != nil && identity.UserID != claims.Subject {
            // Identity user doesn't match hint
            if prompt == "none" {
                s.authErr(w, r, redirectURI, "login_required", state)
                return
            }
            // Force re-login for different user
            identity = nil
        }
    }

    // ...continue with flow...
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

    // Get identity from session cookie
    identity, _ := s.getIdentityFromCookie(r)

    // Validate id_token_hint and match to identity
    if idTokenHint != "" {
        claims, err := s.validateIDTokenHint(idTokenHint)
        if err == nil && identity != nil && identity.UserID == claims.Subject {
            // Valid hint matching identity
        }
    }

    if identity != nil {
        // Deactivate session (preserves consents, future 2FA enrollment)
        s.storage.UpdateUserIdentity(ctx, identity.UserID, identity.ConnectorID,
            func(u UserIdentity) (UserIdentity, error) {
                u.Active = false
                u.SessionID = "" // Invalidate session ID
                u.Clients = nil  // Clear client tracking
                return u, nil
            })

        // Revoke all refresh tokens for this user/connector
        // This ensures complete logout

        // Future: Call LogoutConnector.Logout() for upstream logout
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

**For PasswordConnector (login form exists in Dex):**

```html
<!-- templates/password.html -->
<form method="post">
    <!-- existing fields -->

    {{ if .SessionsEnabled }}
    <div class="remember-me">
        <input type="checkbox" id="remember_me" name="remember_me" value="true">
        <label for="remember_me">Remember me</label>
    </div>
    {{ end }}

    <button type="submit">Login</button>
</form>
```

**For CallbackConnector (no login form in Dex):**

For OAuth/OIDC/SAML connectors, the user is redirected to upstream IDP and there's no Dex login form. Options:

1. **Show on Approval Page** (recommended): Add "Remember Me" checkbox to the approval/consent page. User sees it after returning from upstream IDP, before granting consent.

```html
<!-- templates/approval.html -->
<form method="post">
    <!-- existing scope approval fields -->

    {{ if .SessionsEnabled }}
    <div class="remember-me">
        <input type="checkbox" id="remember_me" name="remember_me" value="true">
        <label for="remember_me">Remember me on this device</label>
    </div>
    {{ end }}

    <button type="submit" name="approval" value="approve">Grant Access</button>
</form>
```

2. **Configurable Default**: If `skipApprovalScreen: true`, user never sees approval page. In this case, use a configurable default:

```yaml
sessions:
  # ...
  # Default remember_me value when approval screen is skipped
  # Options: "always", "never", "prompt" (show intermediate page)
  rememberMeDefault: "never"  # default: "never" for backwards compatibility
```

**Recommendation**: Start with option 1 (approval page). If `skipApprovalScreen: true`, default to no session (current behavior). This maintains backwards compatibility and gives users explicit control.

**Remember Me Behavior**:
- **Unchecked (default)**: No browser session is created. User must re-authenticate on each browser session. This preserves current Dex behavior.
- **Checked**: A persistent session is created. Session cookie persists across browser restarts until `absoluteLifetime` or `validIfNotUsedFor` expires.

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

### Risks and Mitigations

#### Security Risks

| Risk | Mitigation |
|------|------------|
| Session hijacking | Secure cookie flags (HttpOnly, Secure, SameSite), short idle timeout |
| Session fixation | Generate new session ID after authentication |
| CSRF on logout | Require id_token_hint or confirmation page |
| Cookie theft | Bind session to fingerprint (IP range, partial user agent) - optional |
| Storage exposure | Session IDs are random 256-bit values, no sensitive data in cookie |

#### Operational Risks

| Risk | Mitigation |
|------|------------|
| Storage growth | Sessions are per-user like OfflineSessions; admin API allows cleanup |
| Migration complexity | Feature flag allows gradual rollout, no breaking changes |

#### Breaking Changes

**None** - Sessions are opt-in via feature flag and configuration. Existing deployments continue to work without changes.

#### Migration Path

1. Deploy new Dex version - storage migrations create `UserIdentity` table/resource automatically (no feature flag needed for schema)
2. Enable feature flag `DEX_SESSIONS_ENABLED=true` when ready to use sessions
3. Add `sessions:` configuration block
4. Sessions start being created for new logins when "Remember Me" is checked
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

