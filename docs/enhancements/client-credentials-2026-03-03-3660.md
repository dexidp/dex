# Dex Enhancement Proposal (DEP) 3660 - 2026-03-03 - Client Credentials Grant

## Table of Contents

- [Summary](#summary)
- [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [User Experience](#user-experience)
    - [Implementation Details](#implementation-details)
    - [Risks and Mitigations](#risks-and-mitigations)
- [Future Improvements](#future-improvements)

## Summary

[RFC 6749 Section 4.4] defines the `client_credentials` grant type for service-to-service
authentication where no end-user is involved. This DEP proposes implementing this grant in Dex,
gated behind an opt-in configuration flag, so that machine clients can obtain tokens directly
without requiring a browser-based redirect flow.

[RFC 6749 Section 4.4]: https://datatracker.ietf.org/doc/html/rfc6749#section-4.4

## Context

This has been a long-standing community request:

- [#3660 Support client_credentials grant type] is the canonical issue tracking this request
- [#926 Support resource owner password credentials grant] highlights the broader need for
  non-interactive flows in automated environments
- [#2657 Get OIDC token issued by Dex using a token issued by one of the connectors] solved
  a related problem via token exchange (RFC 8693), but does not cover the pure M2M case
  where no upstream user identity exists

Common use cases reported by the community:

- CI/CD pipelines authenticating against ArgoCD or other Dex-protected APIs
- Kubernetes operators and controllers making authenticated API calls
- Service meshes and microservices authenticating without a human in the loop

[#3660 Support client_credentials grant type]: https://github.com/dexidp/dex/issues/3660
[#926 Support resource owner password credentials grant]: https://github.com/dexidp/dex/issues/926
[#2657 Get OIDC token issued by Dex using a token issued by one of the connectors]: https://github.com/dexidp/dex/issues/2657

## Motivation

### Goals

- Allow confidential machine clients to authenticate directly against Dex's `/token` endpoint
  using their client ID and secret, without any user interaction
- Ensure no behavior change for existing deployments by defaulting the feature to disabled
- Support standard scopes (`openid`, `email`, `profile`, `groups`) so that downstream
  applications can use the resulting token with their existing authorization logic

### Non-Goals

- Support for public clients: the `client_credentials` grant requires a confidential client
  with a non-empty secret
- Refresh token issuance: M2M clients are expected to re-authenticate rather than hold
  long-lived refresh tokens
- Custom claim sources: claims are derived from the static client configuration, not from
  a connector backend

## Proposal

### User Experience

Enable the grant by adding `client_credentials` to the `oauth2.grantTypes` list. When
`grantTypes` is omitted, Dex uses a default list that does not include `client_credentials`,
so the entry must be explicit:

```yaml
oauth2:
  grantTypes:
    - authorization_code
    - implicit
    - password
    - refresh_token
    - urn:ietf:params:oauth:grant-type:device_code
    - urn:ietf:params:oauth:grant-type:token-exchange
    - client_credentials
```

Register a confidential static client (non-empty `secret`, no `public: true`):

```yaml
staticClients:
  - id: my-service
    secret: my-service-secret
    name: My Service
```

Request a token via HTTP Basic authentication:

```bash
curl -X POST https://dex.example.com/token \
  -u "my-service:my-service-secret" \
  -d "grant_type=client_credentials"
```

To receive an ID token in addition to the access token, include the `openid` scope:

```bash
curl -X POST https://dex.example.com/token \
  -u "my-service:my-service-secret" \
  -d "grant_type=client_credentials&scope=openid+profile"
```

The response follows the standard OAuth2 token response format:

```json
{
  "access_token": "...",
  "token_type": "bearer",
  "expires_in": 86400
}
```

With `scope=openid`, an `id_token` is included in the response as well.

**Token claims** are derived from the client itself:

| Claim | Value |
|---|---|
| `sub` | base64-encoded protobuf of (client ID, empty connector ID) |
| `aud` | client ID |
| `name` / `preferred_username` | client name (requires `profile` scope) |
| `groups` | groups from `clientCredentialsClaims.groups` on the client (requires `groups` scope) |

**Rejected scopes:** `offline_access` and `federated:id` are not supported; requesting them
returns an error.

#### ArgoCD example

When using Dex as the OIDC provider for ArgoCD, CI/CD pipelines can authenticate
programmatically without a user session:

```yaml
# dex config
oauth2:
  grantTypes:
    - authorization_code
    - implicit
    - password
    - refresh_token
    - urn:ietf:params:oauth:grant-type:device_code
    - urn:ietf:params:oauth:grant-type:token-exchange
    - client_credentials

staticClients:
  - id: argocd-pipeline
    secret: pipeline-secret
    name: ArgoCD Pipeline Client
```

```bash
# Obtain a token from Dex
TOKEN=$(curl -s -X POST https://dex.example.com/token \
  -u "argocd-pipeline:pipeline-secret" \
  -d "grant_type=client_credentials&scope=openid" \
  | jq -r .id_token)

# Use the token with the ArgoCD API
argocd app list --auth-token "$TOKEN" --server argocd.example.com
```

### Implementation Details

The grant is gated behind `clientCredentialsEnabled` in the `oauth2` config block,
following the same pattern as `passwordConnector` for the `password` grant.
When the flag is `false` (the default), the grant type is filtered out in `newServer()`
and never advertised in the discovery document.

The token endpoint handler authenticates the client via HTTP Basic auth, verifies the
client is confidential, and issues a signed token. No connector lookup or user session
is involved.

Implemented in [#4583].

[#4583]: https://github.com/dexidp/dex/pull/4583

### Risks and Mitigations

- **Credential exposure:** client secrets used for `client_credentials` must be treated with
  the same care as any long-lived credential. Rotate them regularly and store them in a
  secrets manager (Kubernetes Secret, Vault, etc.).
- **Over-privileged clients:** because claims come from the client configuration rather than
  a user identity, downstream applications should validate the `sub` claim carefully. Grant
  only the minimum required permissions to each client.
- **Opt-in only:** the grant is never active unless `client_credentials` is listed in
  `oauth2.grantTypes` explicitly, so existing deployments are unaffected.

## Future Improvements

- Allow per-client scope restrictions to limit what a machine client can request
