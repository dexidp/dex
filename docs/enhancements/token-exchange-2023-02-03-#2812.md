# Dex Enhancement Proposal (DEP) 2812 - 2023-02-03 - Token Exchange

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

[RFC 8693] specifies a new OAuth2 `grant_type` of `urn:ietf:params:oauth:grant-type:token-exchange`.
Using this grant type, when clients start an authentication flow with Dex,
in lieu of being redirected to their upstream IDP for authentication on demand,
clients can present an independently obtained, valid token from their IDP to Dex.
This is primarily useful in fully automated environments with job/machine identities,
where there is no human in the loop to handle browser-based login flows.
This DEP proposes to implement the new grant type for Dex.

[RFC 8693]: https://www.rfc-editor.org/rfc/rfc8693.html

## Context

- [#1668 Question: non-web based clients?]
  was closed with no real resolution
- [#1484 Token exchange for external tokens]
  mentions that Keycloak has a similar capability
- [#2657 Get OIDC token issued by Dex using a token issued by one of the connectors] 
  is similar to the previous issue, but this time links to the new (January 2020) [RFC 8693].

I believe the context for all of these are similar:
a downstream project using Dex as its only IDP wants to grant access to programmatic clients
without issuing long lived API tokens.

Examples of downstream issues:

- [argoproj/argo-cd#11632 ArgoCD SSO login via Azure AD Auth using OIDC not work for cli sso login]

Other related Dex issues:

- [#2450 Non-OIDC JWT Connector] is a functionally similar request, but expanded to arbitrary JWTs
- [#1225 GitHub Non-Web application flow support] also asks for an exchange, but for an opaque GitHub PAT

More broadly, this fits into recent movements to issue machine identities:

- [GCP Service Identity](https://cloud.google.com/run/docs/securing/service-identity)
- [AWS Execution Role](https://docs.aws.amazon.com/lambda/latest/dg/lambda-intro-execution-role.html)
- [GitHub Actions OIDC](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect)
- [CircleCI OIDC](https://circleci.com/docs/openid-connect-tokens/)
- [Kubernetes Service Accounts](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/)
- [SPIFFE](https://spiffe.io/)

and granting access to resources based on trusting federated identities:

- [GCP Workload Identity Federation](https://cloud.google.com/iam/docs/workload-identity-federation)
- [AWS STS AssumeRoleWithWebIdentity](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRoleWithWebIdentity.html)

[#1484 Token exchange for external tokens]: https://github.com/dexidp/dex/issues/1484
[#1668 Question: non-web based clients?]: https://github.com/dexidp/dex/issues/1668
[#2657 Get OIDC token issued by Dex using a token issued by one of the connectors]: https://github.com/dexidp/dex/issues/2657
[argoproj/argo-cd#11632 ArgoCD SSO login via Azure AD Auth using OIDC not work for cli sso login]: https://github.com/argoproj/argo-cd/issues/11632
[#2450 Non-OIDC JWT Connector]: https://github.com/dexidp/dex/issues/2450
[#1225 GitHub Non-Web application flow support]: https://github.com/dexidp/dex/issues/1225

An initial attempt is at [#2806](https://github.com/dexidp/dex/pull/2806)

## Motivation

### Goals/Pain

The goal is to allow programmatic access to Dex-protected resources 
without the use of static/long-lived secret tokens (API keys, username/password)
or web-based redirect flows.
Such scenarios are common in CI/CD workflows,
and in general automation of common tasks.

### Non-goals

- Work will be scoped to just the OIDC connector
- [RFC 8693 Section 2.1.1. Relationship between Resource, Audience, and Scope]
  details more complex authorization checks based on targeted resources.
  This is considered out of scope.

[RFC 8693 Section 2.1.1. Relationship between Resource, Audience, and Scope]: https://www.rfc-editor.org/rfc/rfc8693.html#name-relationship-between-resour

## Proposal

### User Experience

Clients can make `POST` requests with `application/x-www-form-urlencoded` 
parameters as specified by [RFC 8693] to Dex's `/token` endpoint.
If successful, an access token will be returned,
allowing direct authentication with Dex.
No refresh tokens will be issued,
perform a new exchange (possibly with refreshed upstream tokens) to obtain a new access token.

The request parameters from [RFC 8693 Section 2.1](https://www.rfc-editor.org/rfc/rfc8693.html#name-request):

- `grant_type`: REQUIRED - `urn:ietf:params:oauth:grant-type:token-exchange`
- `resource`: OPTIONAL - the `audience` in the issued Dex token
- `audience`: REQUIRED (RFC OPTIONAL) - the connector to verify the provided token against
- `scope`: OPTIONAL - the `scope` in the issued Dex token
- `requested_token_type`: OPTIONAL - one of `urn:ietf:params:oauth:token-type:access_token` or `urn:ietf:params:oauth:token-type:id_token`, defaulting to access token
- `subject_token`: REQUIRED - the token issued by the upstream IDP
- `subject_token_type`: REQUIRED - `urn:ietf:params:oauth:token-type:id_token` or `urn:ietf:params:oauth:token-type:access_token` if `getUserInfo` is `true`.
- `actor_token`: OPTIONAL - unused
- `actor_token_type`: OPTIONAL - unused

The response parameters from [RFC 8693 Section 2.2](https://www.rfc-editor.org/rfc/rfc8693.html#name-response):

- `access_token`: the issued token, the field is called `access_token` for legacy reasons
- `issued_token_type`: the actual type of the issued token
- `token_type`: the value `Bearer`
- `expires_in`: validity lifetime in seconds
- `scope`: the requested scope
- `refresh_token`: unused

The connector only needs to be configured with an issuer,
no client ID / client secrets are necessary

```yaml
connectors:
- type: oidc
  id: my-platform
  name: My Platform
  config:
    issuer: https://oidc.my-platform.example/
```

We expose a global and connector setting, 
`allowedGrantTypes: []string` defaulting to all implemented types.

### Implementation Details/Notes/Constraints

- Connectors expose a new interface `TokenIdentity` that will verify the given token and return the associated identity.
  A Dex access/id token is then minted for the given identity.

- `actor_token` and `actor_token_type` are "MUST ... if the actor token is present, 
  also perform the appropriate validation procedures for its indicated token type".
  We will ignore these fields for the initial implementation.


### Risks and Mitigations

With token exchanges (sometimes known as identity impersonation), 
is they allow for easier lateral movement if an attacker gains access to an upstream token.
We limit the potential impact by not issuing refresh tokens, preventing persistent access.
Combined with short token lifetimes, it should limit the period of time between authentication to upstream IDPs.
Additionally, a new `allowedGrantTypes` would allow for disabling exchanges if the functionality isn't needed.

### Alternatives

- Continue to use static keys - 
  this is a secret management nightmare 
  and quite painful when client storage of keys is [breached](https://circleci.com/blog/january-4-2023-security-alert/)

## Future Improvements

- Other connectors may wish to implement the same capability under Oauth
- The password connector could be switch to support this new endpoint, submitting passwords as access tokens,
  allowing for multiple password connectors to be configured
- The `audience` field could be made optional if there is a single connector or the id token is inspected for issuer url
- The `actor_token` and `actor_token_type` can be checked / validated if a suitable use case is determined.
- A policy language like [cel] or [rego] as mentioned on [#1635 Connector Middleware] 
  would allow for stronger assertions of the provided identity against requested resource access.

[cel]: https://github.com/google/cel-go
[rego]: https://www.openpolicyagent.org/docs/latest/policy-language/
[#1635 Connector Middleware]: https://github.com/dexidp/dex/issues/1635
