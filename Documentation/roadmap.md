# dex Roadmap

Here's some of the things that are priorities for the folks working on dex here at CoreOS.

## OpenID Connect Client Self-Registation (Issue #186, PR #267)

Having clients be able to [register themselves](https://openid.net/specs/openid-connect-registration-1_0.html) and manage their own secrets and metadata will be extremely helpful in bootstrapping situations.

## Refresh Tokens (Issue #261)

We currently have refresh tokens implemented as per the OpenID Connect core spec, but we have no way to revoke them. We will probably implement the [OAuth2 token revocation spec](https://tools.ietf.org/html/rfc7009) and/or a UI for revocation.

## Groups (Issue #175)

We want to add support to dex for managing and querying groups of users. The idea is that this will serve as the building blocks for creating authorization systems which use dex. [The proposal](https://docs.google.com/document/d/1OCKW-8rBCngBFWMMrSGokKqWt-a8lg3WvfrejcETBMA/edit#heading=h.9kkruegwavaf) is mostly settled but still should be considered a Work in Progress.


