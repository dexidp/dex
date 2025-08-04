# dex - A federated OpenID Connect provider

![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/dexidp/dex/ci.yaml?style=flat-square&branch=master)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/dexidp/dex/badge?style=flat-square)](https://api.securityscorecards.dev/projects/github.com/dexidp/dex)
[![Go Report Card](https://goreportcard.com/badge/github.com/dexidp/dex?style=flat-square)](https://goreportcard.com/report/github.com/dexidp/dex)
[![Gitpod ready-to-code](https://img.shields.io/badge/Gitpod-ready--to--code-blue?logo=gitpod&style=flat-square)](https://gitpod.io/#https://github.com/dexidp/dex)

![logo](docs/logos/dex-horizontal-color.png)

Dex is an identity service that uses [OpenID Connect][openid-connect] to drive authentication for other apps.

Dex acts as a portal to other identity providers through ["connectors."](#connectors) This lets dex defer authentication to LDAP servers, SAML providers, or established identity providers like GitHub, Google, and Active Directory. Clients write their authentication logic once to talk to dex, then dex handles the protocols for a given backend.

## ID Tokens

ID Tokens are an OAuth2 extension introduced by OpenID Connect and dex's primary feature. ID Tokens are [JSON Web Tokens][jwt-io] (JWTs) signed by dex and returned as part of the OAuth2 response that attests to the end user's identity. An example JWT might look like:

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

## Kubernetes and Dex

Dex runs natively on top of any Kubernetes cluster using Custom Resource Definitions and can drive API server authentication through the OpenID Connect plugin. Clients, such as the [`kubernetes-dashboard`](https://github.com/kubernetes/dashboard) and `kubectl`, can act on behalf of users who can login to the cluster through any identity provider dex supports.

* More docs for running dex as a Kubernetes authenticator can be found [here](https://dexidp.io/docs/guides/kubernetes/).
* You can find more about companies and projects which use dex, [here](./ADOPTERS.md).

## Connectors

When a user logs in through dex, the user's identity is usually stored in another user-management system: a LDAP directory, a GitHub org, etc. Dex acts as a shim between a client app and the upstream identity provider. The client only needs to understand OpenID Connect to query dex, while dex implements an array of protocols for querying other user-management systems.

![](docs/img/dex-flow.png)

A "connector" is a strategy used by dex for authenticating a user against another identity provider. Dex implements connectors that target specific platforms such as GitHub, LinkedIn, and Microsoft as well as established protocols like LDAP and SAML.

Depending on the connectors limitations in protocols can prevent dex from issuing [refresh tokens][scopes] or returning [group membership][scopes] claims. For example, because SAML doesn't provide a non-interactive way to refresh assertions, if a user logs in through the SAML connector dex won't issue a refresh token to its client. Refresh token support is required for clients that require offline access, such as `kubectl`.

Dex implements the following connectors:

| Name | supports refresh tokens | supports groups claim | supports preferred_username claim | status | notes |
| ---- | ----------------------- | --------------------- | --------------------------------- | ------ | ----- |
| [LDAP](https://dexidp.io/docs/connectors/ldap/) | yes | yes | yes | stable | |
| [GitHub](https://dexidp.io/docs/connectors/github/) | yes | yes | yes | stable | |
| [SAML 2.0](https://dexidp.io/docs/connectors/saml/) | no | yes | no | stable | WARNING: Unmaintained and likely vulnerable to auth bypasses ([#1884](https://github.com/dexidp/dex/discussions/1884)) |
| [GitLab](https://dexidp.io/docs/connectors/gitlab/) | yes | yes | yes | beta | |
| [OpenID Connect](https://dexidp.io/docs/connectors/oidc/) | yes | yes | yes | beta | Includes Salesforce, Azure, etc. |
| [OAuth 2.0](https://dexidp.io/docs/connectors/oauth/) | no | yes | yes | alpha | |
| [Google](https://dexidp.io/docs/connectors/google/) | yes | yes | yes | alpha | |
| [LinkedIn](https://dexidp.io/docs/connectors/linkedin/) | yes | no | no | beta | |
| [Microsoft](https://dexidp.io/docs/connectors/microsoft/) | yes | yes | no | beta | |
| [AuthProxy](https://dexidp.io/docs/connectors/authproxy/) | no | yes | no | alpha | Authentication proxies such as Apache2 mod_auth, etc. |
| [Bitbucket Cloud](https://dexidp.io/docs/connectors/bitbucketcloud/) | yes | yes | no | alpha | |
| [OpenShift](https://dexidp.io/docs/connectors/openshift/) | yes | yes | no | alpha | |
| [Atlassian Crowd](https://dexidp.io/docs/connectors/atlassian-crowd/) | yes | yes | yes * | beta | preferred_username claim must be configured through config |
| [Gitea](https://dexidp.io/docs/connectors/gitea/) | yes | no | yes | beta | |
| [OpenStack Keystone](https://dexidp.io/docs/connectors/keystone/) | yes | yes | no | alpha | |

Stable, beta, and alpha are defined as:

* Stable: well tested, in active use, and will not change in backward incompatible ways.
* Beta: tested and unlikely to change in backward incompatible ways.
* Alpha: may be untested by core maintainers and is subject to change in backward incompatible ways.

All changes or deprecations of connector features will be announced in the [release notes][release-notes].

## Documentation

* [Getting started](https://dexidp.io/docs/getting-started/)
* [Intro to OpenID Connect](https://dexidp.io/docs/openid-connect/)
* [Writing apps that use dex][using-dex]
* [What's new in v2](https://dexidp.io/docs/archive/v2/)
* [Custom scopes, claims, and client features](https://dexidp.io/docs/custom-scopes-claims-clients/)
* [Storage options](https://dexidp.io/docs/storage/)
* [gRPC API](https://dexidp.io/docs/api/)
* [Using Kubernetes with dex](https://dexidp.io/docs/kubernetes/)
* Client libraries
  * [Go][go-oidc]

## Reporting a vulnerability

Please see our [security policy](.github/SECURITY.md) for details about reporting vulnerabilities.

## Getting help

- For feature requests and bugs, file an [issue](https://github.com/dexidp/dex/issues).
- For general discussion about both using and developing Dex:
    - join the [#dexidp](https://cloud-native.slack.com/messages/dexidp) on the CNCF Slack
    - open a new [discussion](https://github.com/dexidp/dex/discussions)
    - join the [dex-dev](https://groups.google.com/forum/#!forum/dex-dev) mailing list

[openid-connect]: https://openid.net/connect/
[standard-claims]: https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims
[scopes]: https://dexidp.io/docs/custom-scopes-claims-clients/#scopes
[using-dex]: https://dexidp.io/docs/using-dex/
[jwt-io]: https://jwt.io/
[kubernetes]: https://kubernetes.io/docs/reference/access-authn-authz/authentication/#openid-connect-tokens
[aws-sts]: https://docs.aws.amazon.com/STS/latest/APIReference/Welcome.html
[go-oidc]: https://github.com/coreos/go-oidc
[issue-1065]: https://github.com/dexidp/dex/issues/1065
[release-notes]: https://github.com/dexidp/dex/releases

## Development

When all coding and testing is done, please run the test suite:

```shell
make testall
```

For the best developer experience, install [Nix](https://builtwithnix.org/) and [direnv](https://direnv.net/).

Alternatively, install Go and Docker manually or using a package manager. Install the rest of the dependencies by running `make deps`.

For release process, please read the [release documentation](https://dexidp.io/docs/development/releases/).

## License

The project is licensed under the [Apache License, Version 2.0](LICENSE).
