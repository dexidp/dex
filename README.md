# dex - A federated OpenID Connect provider

![Caution image](Documentation/img/caution.png)

__This is an experimental version of dex that is likely to change in
incompatible ways.__

dex is an OAuth2 server that presents clients with a low overhead framework for
identifying users while leveraging existing identity services such as Google
Accounts, FreeIPA, GitHub, etc, for actual authentication. dex sits between your
applications and an identity service, providing a backend agnostic flavor of
OAuth2 called [OpenID Connect](https://openid.net/connect/), a spec will allows
dex to support:

* Short-lived, signed tokens with predefined fields (such as email) issued on
behalf of users.
* Well known discovery of OAuth2 endpoints.
* OAuth2 mechanisms such as refresh tokens and revocation for long term access.
* Automatic signing key rotation.

Any system which can query dex can cryptographically verify a users identity
based on these tokens, allowing authentication events to be passed between
backend services.

One such application that consumes OpenID Connect tokens is the [Kubernetes](
http://kubernetes.io/) API server, allowing dex to provide identity for any
Kubernetes clusters.
