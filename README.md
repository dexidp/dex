# dex - A federated OpenID Connect provider

[![GoDoc](https://godoc.org/github.com/coreos/dex?status.svg)](https://godoc.org/github.com/coreos/dex)

![logo](Documentation/logos/dex-horizontal-color.png)

Dex is an OpenID Connect server that allows users to login through upstream identity providers. Clients use a standards-based OAuth2 flow to login users, while the actual authentication is performed by established user management systems such as Google, GitHub, FreeIPA, etc.

[OpenID Connect][openid-connect] is a flavor of OAuth that builds on top of OAuth2 using the JOSE standards. This allows dex to provide:

* Short-lived, signed tokens with standard fields (such as email) issued on behalf of users.
* "well-known" discovery of OAuth2 endpoints.
* OAuth2 mechanisms such as refresh tokens and revocation for long term access.
* Automatic signing key rotation.

Standards-based token responses allows applications to interact with any OpenID Connect server instead of writing backend specific "access_token" dances. Systems that can already consume ID Tokens issued by dex include:

* [Kubernetes][kubernetes]
* [Amazon STS][amazon-sts]

## Documentation

* [Getting started](Documentation/getting-started.md)
* [Storage options](Documentation/storage.md)
* [Intro to OpenID Connect](Documentation/openid-connect.md)
* [gRPC API](Documentation/api.md)
* Identity provider logins (coming soon!)
* Client libraries (coming soon!)

[openid-connect]: https://openid.net/connect/
[kubernetes]: http://kubernetes.io/docs/admin/authentication/#openid-connect-tokens
[amazon-sts]: https://docs.aws.amazon.com/STS/latest/APIReference/Welcome.html
