dex
=====

[![Docker Repository on Quay.io](https://quay.io/repository/coreos/dex/status?token=2e772caf-ea17-45d5-8455-8fcf39dae6e1 "Docker Repository on Quay.io")](https://quay.io/repository/coreos/dex)

dex is a federated identity management service. It provides OpenID Connect (OIDC) to users, and can proxy to multiple remote identity providers (IdP) to drive actual authentication, as well as managing local username/password credentials.

We named the project 'dex' beceause it is a central index of users that other pieces of software can authenticate against.


## Architecture

dex consists of multiple components:

- **dex-worker** is the primary server component of dex
    - host a user-facing API that drives the OIDC protocol
	- proxy to remote identity providers via "connectors"
    - provides an API for administrators to manage users.
- **dex-overlord** is an auxiliary process responsible for two things:
	- rotation of keys used by the workers to sign identity tokens
	- garbage collection of stale data in the database
    - provides an API for bootstrapping the system.
- **dexctl** is CLI tool used to manage an dex deployment
	- configure identity provider connectors
	- administer OIDC client identities
- **database**; a database is used to for persistent storage for keys, users,
  OAuth sessions and other data. Currently Postgres is the only supported
  database.

A typical dex deployment consists of N dex-workers behind a load balanacer, and one dex-overlord.
The dex-workers directly handle user requests, so the loss of all workers can result in service downtime.
The single dex-overlord runs its tasks periodically, so it does not need to maintain 100% uptime.

## Who Should Use Dex?

    **TODO**

## Similar Software

    **TODO**

## Connectors

Remote IdPs could implement any auth-N protocol. *Connectors* contain protocol-specific logic and are used to communicate with remote IdPs. Possible examples of connectors could be: OIDC, LDAP, Local credentials, Basic Auth, etc.

dex ships with an OIDC connector, useful for authenticating with services like Google and Salesforce (or even other dex instances!) and a "local" connector, in which dex itself presents a UI for users to authenticate via dex-stored credentials.

Future connectors can be developed and added as future interoperability requirements emerge.

## Relevant Specifications

These specs are referenced and implemented to some degree in the `jose` package of this project.

- [JWK](https://tools.ietf.org/html/draft-ietf-jose-json-web-key-36)
- [JWT](https://tools.ietf.org/html/draft-ietf-oauth-json-web-token-30)
- [JWS](https://tools.ietf.org/html/draft-jones-json-web-signature-04)

OpenID Connect (OIDC) is broken up into several specifications. The following (amongst others) are relevant:

- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- [OpenID Connect Discovery 1.0](https://openid.net/specs/openid-connect-discovery-1_0.html)
- [OAuth 2.0 RFC](https://tools.ietf.org/html/rfc6749)

## Example OIDC Discovery Endpoints

- https://accounts.google.com/.well-known/openid-configuration
- https://login.salesforce.com/.well-known/openid-configuration

# Next steps:

If you want to try out dex quickly with a single process and no database (do *not* run this way in production!) take a look at the [dev guide][dev-guide].

For running the full stack check out the [getting started guide][getting-started].

[getting-started]: https://github.com/coreos/dex/blob/master/Documentation/getting-started.md
[dev-guide]: https://github.com/coreos/dex/blob/master/Documentation/dev-guide.md

# Coming Soon

- Multiple backing Identity Providers
- Identity Management
- Authorization
