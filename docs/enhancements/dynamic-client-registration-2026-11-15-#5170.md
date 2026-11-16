# Dex Enhancement Proposal (DEP) 5170 - 2026-11-15 - OAuth 2.0 Dynamic Client Registration (RFC 7591)

## Table of Contents

- [Summary](#summary)
- [Context](#context)
- [Motivation](#motivation)
    - [Goals/Pain](#goalspain)
    - [Non-goals](#non-goals)
- [Proposal](#proposal)
    - [User Experience](#user-experience)
    - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Risks and Mitigations](#risks-and-mitigations)
    - [Alternatives](#alternatives)
- [Future Improvements](#future-improvements)

## Summary

[RFC 7591] lets an OAuth 2.0 client register itself with an
authorization server at runtime instead of being configured by an
administrator in advance.

This DEP proposes a registration endpoint at `/register` that is
disabled by default and, when enabled, always requires an
operator-issued initial access token. A client registered this way is
stored like any other client, but it is marked as dynamically
registered and is held to the metadata it registered: its redirect
URIs, grant types, response types, scopes and token endpoint
authentication method. Clients configured statically or through the
gRPC API keep their current behavior.

[RFC 7591]: https://datatracker.ietf.org/doc/html/rfc7591

## Context

- [#5170] is the discussion for this proposal.
- [#4988] is the implementation this DEP describes. The review of that
  PR asked for a DEP covering the permission model, the client
  lifecycle and the interaction with existing features.
- [#4383] is an earlier implementation of RFC 7591. It was closed in
  favor of #4988, which adds persisted registration policy, its
  enforcement, and configuration.
- [#186] is the original request for OIDC dynamic client registration.
- [#448] and [#4930] ask for wildcard or regular-expression redirect
  URIs. Dynamic registration solves the same problem without loosening
  redirect URI matching.
- [RFC 7592] (registration management) and [OpenID Connect Dynamic
  Client Registration 1.0] are related specifications. This DEP follows
  the OIDC document where it constrains the relationship between
  `grant_types` and `response_types`.

[#5170]: https://github.com/dexidp/dex/discussions/5170
[#4988]: https://github.com/dexidp/dex/pull/4988
[#4383]: https://github.com/dexidp/dex/pull/4383
[#186]: https://github.com/dexidp/dex/issues/186
[#448]: https://github.com/dexidp/dex/issues/448
[#4930]: https://github.com/dexidp/dex/pull/4930
[RFC 7592]: https://datatracker.ietf.org/doc/html/rfc7592
[OpenID Connect Dynamic Client Registration 1.0]: https://openid.net/specs/openid-connect-registration-1_0.html

## Motivation

### Goals/Pain

Every Dex client must be declared ahead of time, with its exact
redirect URIs, in the configuration file or through the gRPC API. That
does not scale when the relying parties are numerous and not known in
advance.

The motivating deployment is a fleet of hardware nodes that each host
their own web UI. Every node is its own relying party with its own
redirect URI, and the software on the node already speaks RFC 7591.
Today the choices are:

1. Maintain one static client per node, or one client listing every
   node's redirect URI, and update it whenever a node is added, renamed
   or removed.
2. Give every provisioning system gRPC API credentials. The gRPC API is
   an administrative API: a caller that can create a client can also
   delete clients, change passwords and manage connectors.
3. Loosen redirect URI matching with wildcards or regular expressions,
   which weakens the property that makes redirect URI validation
   useful.

The goals of this DEP are:

- Let a client that holds an operator-issued credential register itself
  over a standard protocol, with no other access to Dex.
- Constrain a self-registered client to what it registered, so that
  holding the registration credential does not grant more than the
  ability to create ordinary interactive login clients.
- Leave every existing client and every deployment that does not enable
  the feature unchanged.

### Non-goals

- Open (unauthenticated) registration.
- Registration management ([RFC 7592]): reading, updating or deleting a
  client through the registration API.
- Automatic expiry of registered clients.
- Software statements, `jwks`/`jwks_uri`, and the `private_key_jwt` and
  `client_secret_jwt` authentication methods.
- Non-interactive grants for dynamically registered clients: password,
  client credentials, token exchange and device code.
- Rate limiting and caps on the number of registered clients.

## Proposal

### User Experience

#### Enabling registration

Registration is off unless the operator turns it on, and it cannot be
turned on without an initial access token:

```yaml
oauth2:
  dynamicClientRegistration:
    enabled: true
    # Name of an environment variable holding the token, so that it
    # can be injected from a secret and stays out of this file.
    initialAccessTokenEnv: DEX_REGISTRATION_INITIAL_ACCESS_TOKEN
    # Alternatively, inline. Mutually exclusive with
    # initialAccessTokenEnv.
    # initialAccessToken: "..."
```

Dex refuses to start if registration is enabled without a token, if
both token settings are given, or if the named environment variable is
unset or empty.

When enabled, discovery advertises `registration_endpoint`. When
disabled, the route is not mounted and the field is absent.

#### Registering a client

```
POST /register
Authorization: Bearer <initial access token>
Content-Type: application/json

{
  "redirect_uris": ["https://node-17.example.com/oauth2/callback"],
  "client_name": "node-17",
  "scope": "openid profile email"
}
```

```
HTTP/1.1 201 Created
Cache-Control: no-store

{
  "client_id": "...",
  "client_secret": "...",
  "client_id_issued_at": 1700000000,
  "client_secret_expires_at": 0,
  "client_name": "node-17",
  "redirect_uris": ["https://node-17.example.com/oauth2/callback"],
  "token_endpoint_auth_method": "client_secret_basic",
  "grant_types": ["authorization_code"],
  "response_types": ["code"],
  "scope": "openid profile email"
}
```

Dex generates the client ID and secret; a client cannot choose them.
The response echoes the metadata as registered, including defaults, as
RFC 7591 requires. Errors use the RFC 7591 codes `invalid_redirect_uri`
and `invalid_client_metadata`, and `invalid_token` with a
`WWW-Authenticate` challenge when the initial access token is missing
or wrong.

#### Accepted metadata

| Field | Rule |
|---|---|
| `redirect_uris` | Required, at most 20, exact-match, no fragment or userinfo. Confidential clients: `https` only. Public clients: also `http` on loopback and private-use URI schemes. |
| `token_endpoint_auth_method` | `client_secret_basic` (default), `client_secret_post`, or `none` for a public client. |
| `grant_types` | Subset of `authorization_code` (default), `refresh_token`, `implicit`, and only those the server has enabled. Must include `authorization_code` or `implicit`. |
| `response_types` | Default `code`. Only response types the server is configured with. Must be consistent with `grant_types`: `code` needs `authorization_code`; `token` and `id_token` need `implicit`. |
| `scope` | Default `openid`. Subset of `openid`, `offline_access`, `email`, `profile`, `groups`, `federated:id`. `openid` is required; `offline_access` requires the `refresh_token` grant. |
| `client_name` | Optional, at most 256 bytes. |
| `logo_uri` | Optional, `https`, same origin as one of the redirect URIs. |

Unknown metadata is ignored, as RFC 7591 section 2 requires. The
request body is limited to 64 KiB.

#### What a registered client can do

A dynamically registered client behaves like a static client, except
that Dex enforces what it registered:

- An authorization request must use a registered redirect URI, a
  registered response type, and only registered scopes.
- A token request must use a registered grant type and the registered
  authentication method. A client registered for `client_secret_basic`
  cannot authenticate with `client_secret_post`, and vice versa.
- A public client (`none`) must use PKCE with `S256` on every
  authorization code request, whether or not PKCE is enforced globally.
- On the approval page, a notice states that the application registered
  itself and shows its client ID and the origin it will redirect to.

#### Operating it

- **Who can register.** Anyone holding the initial access token. There
  is one token per Dex deployment.
- **Rotating the token.** Change the configured value and restart Dex.
  The old token stops working immediately. Clients registered with the
  old token keep working.
- **Finding clients by token.** Each registered client records an
  identifier of the token that created it (a truncated SHA-256, never
  the token itself) and the time of registration. After a token leak,
  an operator can identify the clients created with the leaked token.
- **Removing a client.** Through the gRPC API (`DeleteClient`) or
  directly in storage. A registered client does not expire and cannot
  remove itself.
- **Changing a client.** Through the gRPC API (`UpdateClient`), for the
  fields it supports. A client that needs different metadata registers
  again and receives a new client ID.
- **Auditing.** Successful registrations are logged at info level with
  the client ID, name, grants, response types, scopes and token
  identifier. Rejections are logged at warning level with the error,
  the remote address, the user agent and, for invalid metadata, the
  submitted metadata. For a rejected token the log says whether the
  header was missing, malformed or mismatched. Tokens and client
  secrets are never logged.

### Implementation Details/Notes/Constraints

#### Storage

`storage.Client` gains the registration policy:

```go
type Client struct {
    // ...existing fields...

    DynamicallyRegistered   bool
    GrantTypes              []string
    ResponseTypes           []string
    AllowedScopes           []string
    TokenEndpointAuthMethod string
    RegistrationTime        int64
    RegistrationTokenID     string
    RegistrationExpiresAt   int64 // reserved, see Future Improvements
}

func (c Client) AllowsGrantType(grantType string) bool
func (c Client) AllowsResponseType(responseTypes []string) bool
func (c Client) AllowsScopes(scopes []string) bool
```

Each `Allows*` method returns true when `DynamicallyRegistered` is
false. This is the single switch that keeps static clients on their
existing behavior: the enforcement points below call these methods
unconditionally.

The change is additive in every storage backend: new columns with
defaults in SQL and ent (one new migration), and new optional fields in
the Kubernetes and etcd representations. Existing clients read back
with `DynamicallyRegistered == false`. The storage conformance tests
cover the new fields.

#### Authentication of the registration request

The endpoint compares the SHA-256 of the presented bearer token with
the SHA-256 of the configured token in constant time. No registration
request is parsed before that check passes.

#### Enforcement points

| Where | Check |
|---|---|
| `/auth` | Redirect URI is registered (existing exact-match check). Requested scopes, response type, and the grant each response type implies are registered. Public clients must send an `S256` challenge. |
| `/token` | The grant type is registered, checked after client authentication. Requested scopes are registered. The authentication method used matches the registered one. |
| `/device/code` | The client registered the `device_code` grant. |
| `/device/callback` | The client registered the `device_code` grant. This is where device flow tokens are minted, so the check does not rely on the earlier ones. |

For the purpose of these checks, a response type that returns a token
or an ID token from the authorization endpoint requires the `implicit`
grant, alone or together with `authorization_code` in a hybrid flow.
The server derives its own supported grant types the same way, so every
response type the server is configured with can be registered. As a
consequence, a server configured with the `id_token` or `code id_token`
response type now lists `implicit` in `grant_types_supported`.

#### Interaction with existing features

- **Static and gRPC-created clients.** Unchanged. They are never marked
  as dynamically registered, and no check applies to them.
- **`skipApprovalScreen`.** Applies to dynamically registered clients
  as it does to all others. When the approval screen is shown, it
  carries the self-registration notice. Operators who enable
  registration and want users to see that notice must not skip the
  approval screen.
- **Device flow.** Not available. `device_code` cannot be registered,
  and the device endpoints reject a client that did not register it,
  including a client that registers Dex's own device callback as its
  redirect URI.
- **Password, client credentials and token exchange grants.** Not
  available. They cannot be registered and are refused at the token
  endpoint.
- **`allowedConnectors`.** Not part of registration metadata, so a
  registered client may use every connector. An operator can restrict a
  registered client afterwards through the gRPC API.
- **MFA.** A registered client has no `mfaChain` of its own, so the
  server's `defaultMFAChain` applies.
- **Trusted peers and cross-client scopes.** A registered client has no
  trusted peers, and `audience:server:client_id:*` scopes are outside
  the registrable scope set, so they are rejected.
- **Sessions and logout.** Post-logout redirect URIs, back-channel
  logout and shared SSO cannot be registered. An operator can set the
  fields the gRPC API supports.
- **Refresh tokens.** Issued only when the client registered both the
  `refresh_token` grant and the `offline_access` scope.
- **Client logo.** `logo_uri` is validated and stored, but Dex does not
  render logos supplied through registration.
- **Discovery.** `registration_endpoint` is present only when the
  feature is enabled. `token_endpoint_auth_methods_supported` gains
  `none`, which Dex already accepts for public clients, and
  `scopes_supported` gains `federated:id`.

### Risks and Mitigations

- **New endpoint that creates persistent state.** The endpoint is not
  mounted unless enabled, and Dex will not start with it enabled and no
  token. Request size, URI lengths and list lengths are bounded.
- **Leaked initial access token.** The holder can create clients but
  cannot read, change or delete anything, and can only create
  interactive login clients for exact redirect URIs. A user still has
  to authenticate with a connector before any token is issued. Rotating
  the token stops further registrations, and the recorded token
  identifier finds the clients it created.
- **Phishing through a self-registered client.** A token holder could
  register a client with a misleading name. The approval screen
  identifies self-registered clients and shows the redirect origin,
  confidential clients are limited to `https` redirect URIs, and logos
  are not rendered. This mitigation is lost when `skipApprovalScreen`
  is set.
- **Unbounded growth.** Registered clients do not expire and there is
  no cap. A token holder can fill storage. The initial access token is
  the only control; deployments that need more should rate-limit
  `/register` at their ingress. Expiry and caps are listed under Future
  Improvements.
- **Policy bypass through another flow.** A client could try to obtain
  tokens through a flow it did not register, as with the device flow.
  Enforcement is placed at each point where an authorization or token
  is issued rather than only at registration, and each is covered by a
  test that drives the real server.
- **Schema change.** Additive, with defaults. No data migration is
  needed and existing clients are unaffected. Downgrading Dex after
  clients have been registered leaves those clients in storage without
  their restrictions; they should be deleted before a downgrade.
- **Behavior change outside the feature.** `grant_types_supported`
  lists `implicit` for servers configured with `id_token` response
  types. No request is accepted or rejected differently for existing
  clients.

This is not a breaking change.

### Alternatives

- **Create clients through the gRPC API.** It works today, but the
  credential it requires grants full administrative access to Dex, and
  the clients this DEP targets implement RFC 7591, not Dex's gRPC API.
- **Wildcard or regular-expression redirect URIs** ([#448], [#4930]).
  One shared client with a pattern is simpler to operate, but every
  node then shares one secret, and a loose pattern lets any matching
  host receive authorization codes.
- **Open registration.** RFC 7591 permits it. It is rejected here
  because it would let any network caller create persistent clients.
- **A separate credential store for registration (multiple tokens,
  scoped or expiring).** More flexible, and more to design and operate.
  A single token with a recorded identifier covers the motivating use
  case and leaves room to add more later.
- **An external provisioning service that calls the gRPC API.** It
  moves the same design questions into a component every deployment
  would have to write.

If this is not implemented, deployments with many dynamically
provisioned relying parties keep maintaining static client lists or run
their own provisioning service against the gRPC API.

## Future Improvements

- Registration management ([RFC 7592]) with per-client registration
  access tokens, so a client can update or delete itself.
- Optional lifetime for registered clients, using the reserved
  `RegistrationExpiresAt` field and garbage collection of expired
  clients.
- More than one initial access token, to allow rotation without a
  restart and separate credentials per provisioning system.
- A cap on the number of dynamically registered clients, and rate
  limiting.
- Exposing the registration fields through the gRPC API, so operators
  can list dynamically registered clients and filter by token
  identifier.
- An option to always show the approval screen for dynamically
  registered clients, independent of `skipApprovalScreen`.
- Per-registration defaults for `allowedConnectors` and `mfaChain`.
- Device flow for dynamically registered clients.
