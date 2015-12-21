# dex


[![Build Status](https://travis-ci.org/coreos/dex.png?branch=master)](https://travis-ci.org/coreos/dex)
[![Docker Repository on Quay.io](https://quay.io/repository/coreos/dex/status?token=2e772caf-ea17-45d5-8455-8fcf39dae6e1 "Docker Repository on Quay.io")](https://quay.io/repository/coreos/dex)
[![GoDoc](https://godoc.org/github.com/coreos/dex?status.svg)](https://godoc.org/github.com/coreos/dex)

dex is a federated identity management service. It provides OpenID Connect (OIDC) and OAuth 2.0 to users, and can proxy to multiple remote identity providers (IdP) to drive actual authentication, as well as managing local username/password credentials.

We named the project 'dex' because it is a central index of users that other pieces of software can authenticate against.


## Architecture

dex consists of multiple components:

- **dex-worker** is the primary server component of dex
	- host a user-facing API that drives the OIDC protocol
	- proxy to remote identity providers via "connectors"
	- provides an API for administrators to manage users.
- **dex-overlord** is an auxiliary process responsible for various administrative tasks:
	- rotation of keys used by the workers to sign identity tokens
	- garbage collection of stale data in the database
	- provides an API for bootstrapping the system.
- **dexctl** is a CLI tool used to manage a dex deployment
	- configure identity provider connectors
	- administer OIDC client identities
- **database**; a database is used to for persistent storage for keys, users,
  OAuth sessions and other data. Currently Postgres is the only supported
  database.

A typical dex deployment consists of N dex-workers behind a load balanacer, and one dex-overlord.
The dex-workers directly handle user requests, so the loss of all workers can result in service downtime.
The single dex-overlord runs its tasks periodically, so it does not need to maintain 100% uptime.

## Who Should Use Dex?

A non-exhaustive list of those who would benefit from using dex:

- Those who want a language/framework-agnostic way to manage authentication.
- Those who want to federate authentication from mutiple providers of differing types.
- Those who want to manage user credentials (eg. username and password) and perform authentication locally
- Those who want to create an OIDC Identity Provider for multiple clients to authenticate against.
- Those who want any or all of the above in a Free and Open Source project.

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

## Similar Software

### [Auth0](https://auth0.com)

Auth0 is a commercial product which implements the OpenID Connect protocol and [JWT](http://jwt.io). It comes with built-in support for 30+ social providers (and provide extenibility points to add customs); enterprise providers like ADFS, SiteMinder, Ping, Tivoli, or any SAML provider; LDAP/AD connectors that can be run behind firewalls via [an open source agent/connector](https://github.com/auth0/ad-ldap-connector); built-in user/password stores with email and phone verification; legacy user/password stores running Mongo, PG, MySQL, SQL Server among others; multi-factor auth; passwordless support; custom extensibility of the auth pipeline through node.js and many other things.

You could chain dex with Auth0, dex as RP and Auth0 as OpenId Connect Provider, and bring to dex all the providers that comes in Auth0 plus the user management capabilities. 

### [CloudFoundry UAA](https://github.com/cloudfoundry/uaa)

>The UAA is a multi tenant identity management service, used in Cloud Foundry, but also available as a stand alone OAuth2 server.

### [OmniAuth](https://github.com/intridea/omniauth)

OmniAuth provides authentication federation at the language (Ruby) level, with a [wide range of integrations](https://github.com/intridea/omniauth/wiki/List-of-Strategies) available.

### [Okta](http://developer.okta.com/product/)
Okta is a commercial product which is similar to dex in that for it too, identity federation is a key feature. It connects to many more authentication providers than dex, and also does the federation in the oppposite direction - it can be used as a SSO to other identity providers.

### [Shibboleth](https://shibboleth.net/)

Shibboleth is an open source system implementing the [SAML](https://www.oasis-open.org/standards#samlv2.0) standard, and can federate from a variety of backends, most notably LDAP. 

