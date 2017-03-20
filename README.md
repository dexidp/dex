# dex - A federated OpenID Connect provider

[![Travis](https://api.travis-ci.org/coreos/dex.svg)](https://travis-ci.org/coreos/dex)
[![GoDoc](https://godoc.org/github.com/coreos/dex?status.svg)](https://godoc.org/github.com/coreos/dex)
[![Go Report Card](https://goreportcard.com/badge/github.com/coreos/dex)](https://goreportcard.com/report/github.com/coreos/dex)

![logo](Documentation/logos/dex-horizontal-color.png)

Dex is an OpenID Connect server that connects to other identity providers. Clients use a standards-based OAuth2 flow to login users, while the actual authentication is performed by established user management systems such as Google, GitHub, FreeIPA, etc.

[OpenID Connect][openid-connect] is a flavor of OAuth that builds on top of OAuth2 using the JOSE standards. This allows dex to provide:

* Short-lived, signed tokens with standard fields (such as email) issued on behalf of users.
* "well-known" discovery of OAuth2 endpoints.
* OAuth2 mechanisms such as refresh tokens and revocation for long term access.
* Automatic signing key rotation.

Standards-based token responses allows applications to interact with any OpenID Connect server instead of writing backend specific "access_token" dances. Systems that can already consume ID Tokens issued by dex include:

* [Kubernetes][kubernetes]
* [AWS STS][aws-sts]

## Kubernetes + dex

Dex's main production use is as an auth-N addon in CoreOS's enterprise Kubernetes solution, [Tectonic][tectonic]. Dex runs natively on top of any Kubernetes cluster using Third Party Resources and can drive API server authentication through the OpenID Connect plugin. Clients, such as the [Tectonic Console][tectonic-console] and `kubectl`, can act on behalf users who can login to the cluster through any identity provider dex supports.

More docs for running dex as a Kubernetes authenticator can be found [here](Documentation/kubernetes.md).

## Documentation

* [Getting started](Documentation/getting-started.md)
* [What's new in v2](Documentation/v2.md)
* [Storage options](Documentation/storage.md)
* [Intro to OpenID Connect](Documentation/openid-connect.md)
* [gRPC API](Documentation/api.md)
* [Using Kubernetes with dex](Documentation/kubernetes.md)
* Identity provider logins
  * [LDAP](Documentation/ldap-connector.md)
  * [GitHub](Documentation/github-connector.md)
  * [SAML 2.0 (experimental)](Documentation/saml-connector.md)
  * [OpenID Connect](Documentation/oidc-connector.md) (includes Google, Salesforce, Azure, etc.)
* Client libraries
  * [Go][go-oidc]

## Getting help

* For bugs and feature requests (including documentation!), file an [issue][issues].
* For general discussion about both using and developing dex, join the [dex-dev][dex-dev] mailing list.
* For more details on dex development plans, check out the GitHub [milestones][milestones].

[openid-connect]: https://openid.net/connect/
[kubernetes]: http://kubernetes.io/docs/admin/authentication/#openid-connect-tokens
[aws-sts]: https://docs.aws.amazon.com/STS/latest/APIReference/Welcome.html
[tectonic]: https://tectonic.com/
[tectonic-console]: https://tectonic.com/enterprise/docs/latest/usage/index.html#tectonic-console
[go-oidc]: https://github.com/coreos/go-oidc
[issues]: https://github.com/coreos/dex/issues
[dex-dev]: https://groups.google.com/forum/#!forum/dex-dev
[milestones]: https://github.com/coreos/dex/milestones
