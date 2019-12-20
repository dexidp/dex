# dex - A federated OpenID Connect provider

[![Travis](https://api.travis-ci.org/dexidp/dex.svg)](https://travis-ci.org/dexidp/dex)
[![GoDoc](https://godoc.org/github.com/dexidp/dex?status.svg)](https://godoc.org/github.com/dexidp/dex)
[![Go Report Card](https://goreportcard.com/badge/github.com/dexidp/dex)](https://goreportcard.com/report/github.com/dexidp/dex)

![logo](Documentation/logos/dex-horizontal-color.png)

Dex is an identity service that uses [OpenID Connect][openid-connect] to drive authentication for other apps.

Dex acts as a portal to other identity providers through ["connectors."](#connectors) This lets dex defer authentication to LDAP servers, SAML providers, or established identity providers like GitHub, Google, and Active Directory. Clients write their authentication logic once to talk to dex, then dex handles the protocols for a given backend.

## ID Tokens

ID Tokens are an OAuth2 extension introduced by OpenID Connect and dex's primary feature. ID Tokens are [JSON Web Tokens][jwt-io] (JWTs) signed by dex and returned as part of the OAuth2 response that attest to the end user's identity. An example JWT might look like:

```
eyJhbGciOiJSUzI1NiIsImtpZCI6IjlkNDQ3NDFmNzczYjkzOGNmNjVkZDMyNjY4NWI4NjE4MGMzMjRkOTkifQ.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjU1NTYvZGV4Iiwic3ViIjoiQ2djeU16UXlOelE1RWdabmFYUm9kV0kiLCJhdWQiOiJleGFtcGxlLWFwcCIsImV4cCI6MTQ5Mjg4MjA0MiwiaWF0IjoxNDkyNzk1NjQyLCJhdF9oYXNoIjoiYmk5NmdPWFpTaHZsV1l0YWw5RXFpdyIsImVtYWlsIjoiZXJpYy5jaGlhbmdAY29yZW9zLmNvbSIsImVtYWlsX3ZlcmlmaWVkIjp0cnVlLCJncm91cHMiOlsiYWRtaW5zIiwiZGV2ZWxvcGVycyJdLCJuYW1lIjoiRXJpYyBDaGlhbmcifQ.OhROPq_0eP-zsQRjg87KZ4wGkjiQGnTi5QuG877AdJDb3R2ZCOk2Vkf5SdP8cPyb3VMqL32G4hLDayniiv8f1_ZXAde0sKrayfQ10XAXFgZl_P1yilkLdknxn6nbhDRVllpWcB12ki9vmAxklAr0B1C4kr5nI3-BZLrFcUR5sQbxwJj4oW1OuG6jJCNGHXGNTBTNEaM28eD-9nhfBeuBTzzO7BKwPsojjj4C9ogU4JQhGvm_l4yfVi0boSx8c0FX3JsiB0yLa1ZdJVWVl9m90XmbWRSD85pNDQHcWZP9hR6CMgbvGkZsgjG32qeRwUL_eNkNowSBNWLrGNPoON1gMg
```

ID Tokens contains standard claims assert which client app logged the user in, when the token expires, and the identity of the user.

```json
{
  "iss": "http://127.0.0.1:5556/dex",
  "sub": "CgcyMzQyNzQ5EgZnaXRodWI",
  "aud": "example-app",
  "exp": 1492882042,
  "iat": 1492795642,
  "at_hash": "bi96gOXZShvlWYtal9Eqiw",
  "email": "jane.doe@coreos.com",
  "email_verified": true,
  "groups": [
    "admins",
    "developers"
  ],
  "name": "Jane Doe"
}
```

Because these tokens are signed by dex and [contain standard-based claims][standard-claims] other services can consume them as service-to-service credentials. Systems that can already consume OpenID Connect ID Tokens issued by dex include:

* [Kubernetes][kubernetes]
* [AWS STS][aws-sts]

For details on how to request or validate an ID Token, see [_"Writing apps that use dex"_][using-dex].

## Kubernetes + dex

Dex's main production use is as an auth-N addon in CoreOS's enterprise Kubernetes solution, [Tectonic][tectonic]. Dex runs natively on top of any Kubernetes cluster using Third Party Resources and can drive API server authentication through the OpenID Connect plugin. Clients, such as the [Tectonic Console][tectonic-console] and `kubectl`, can act on behalf users who can login to the cluster through any identity provider dex supports.

More docs for running dex as a Kubernetes authenticator can be found [here](Documentation/kubernetes.md).

## Connectors

When a user logs in through dex, the user's identity is usually stored in another user-management system: a LDAP directory, a GitHub org, etc. Dex acts as a shim between a client app and the upstream identity provider. The client only needs to understand OpenID Connect to query dex, while dex implements an array of protocols for querying other user-management systems.

![](Documentation/img/dex-flow.png)

A "connector" is a strategy used by dex for authenticating a user against another identity provider. Dex implements connectors that target specific platforms such as GitHub, LinkedIn, and Microsoft as well as established protocols like LDAP and SAML.

Depending on the connectors limitations in protocols can prevent dex from issuing [refresh tokens][scopes] or returning [group membership][scopes] claims. For example, because SAML doesn't provide a non-interactive way to refresh assertions, if a user logs in through the SAML connector dex won't issue a refresh token to its client. Refresh token support is required for clients that require offline access, such as `kubectl`.

Dex implements the following connectors:

| Name | supports refresh tokens | supports groups claim | supports preferred_username claim | status | notes |
| ---- | ----------------------- | --------------------- | --------------------------------- | ------ | ----- |
| [LDAP](Documentation/connectors/ldap.md) | yes | yes | yes | stable | |
| [GitHub](Documentation/connectors/github.md) | yes | yes | yes | stable | |
| [SAML 2.0](Documentation/connectors/saml.md) | no | yes | no | stable |
| [GitLab](Documentation/connectors/gitlab.md) | yes | yes | yes | beta | |
| [OpenID Connect](Documentation/connectors/oidc.md) | yes | no ([#1065][issue-1065]) | no | beta | Includes Google, Salesforce, Azure, etc. |
| [LinkedIn](Documentation/connectors/linkedin.md) | yes | no | no | beta | |
| [Microsoft](Documentation/connectors/microsoft.md) | yes | yes | no | beta | |
| [AuthProxy](Documentation/connectors/authproxy.md) | no | no | no | alpha | Authentication proxies such as Apache2 mod_auth, etc. |
| [Bitbucket Cloud](Documentation/connectors/bitbucketcloud.md) | yes | yes | no | alpha | |

Stable, beta, and alpha are defined as:

* Stable: well tested, in active use, and will not change in backward incompatible ways.
* Beta: tested and unlikely to change in backward incompatible ways.
* Alpha: may be untested by core maintainers and is subject to change in backward incompatible ways.

All changes or deprecations of connector features will be announced in the [release notes][release-notes].

## Documentation

* [Getting started](Documentation/getting-started.md)
* [Intro to OpenID Connect](Documentation/openid-connect.md)
* [Writing apps that use dex][using-dex]
* [What's new in v2](Documentation/v2.md)
* [Custom scopes, claims, and client features](Documentation/custom-scopes-claims-clients.md)
* [Storage options](Documentation/storage.md)
* [gRPC API](Documentation/api.md)
* [Using Kubernetes with dex](Documentation/kubernetes.md)
* Client libraries
  * [Go][go-oidc]

## Reporting a security vulnerability

Due to their public nature, GitHub and mailing lists are NOT appropriate places for reporting vulnerabilities. Please refer to CoreOS's [security disclosure][disclosure] process when reporting issues that may be security related.

## Getting help

* For feature requests and bugs, file an [issue][issues].
* For general discussion about both using and developing dex, join the [dex-dev][dex-dev] mailing list.

[openid-connect]: https://openid.net/connect/
[standard-claims]: https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims
[scopes]: Documentation/custom-scopes-claims-clients.md#scopes
[using-dex]: Documentation/using-dex.md
[jwt-io]: https://jwt.io/
[kubernetes]: http://kubernetes.io/docs/admin/authentication/#openid-connect-tokens
[aws-sts]: https://docs.aws.amazon.com/STS/latest/APIReference/Welcome.html
[tectonic]: https://tectonic.com/
[tectonic-console]: https://tectonic.com/enterprise/docs/latest/usage/index.html#tectonic-console
[go-oidc]: https://github.com/coreos/go-oidc
[issue-1065]: https://github.com/dexidp/dex/issues/1065
[release-notes]: https://github.com/dexidp/dex/releases
[issues]: https://github.com/dexidp/dex/issues
[dex-dev]: https://groups.google.com/forum/#!forum/dex-dev
[disclosure]: https://coreos.com/security/disclosure/
