dex
=====

[![Docker Repository on Quay.io](https://quay.io/repository/coreos/dex/status?token=5a9732e4-53d6-4419-b56b-9f784f7f9233 "Docker Repository on Quay.io")](https://quay.io/repository/coreos/dex)

dex is a federated identity management service.
It provides OpenID Connect (OIDC) to users, while it proxies to multiple remote identity providers (IdP) to drive actual authentication.
We named the project 'dex' beceause it is a central index of users that other pieces of software can authenticate against.

## Architecture

dex consists of multiple components:

- **dex-worker** is the primary server component of dex
	- host a user-facing API that drives the OIDC protocol
	- proxy to remote identity providers via "connectors"
- **dex-overlord** is an auxiliary process responsible for two things:
	- rotation of keys used by the workers to sign identity tokens
	- garbage collection of stale data in the database
- **dexctl** is CLI tool used to manage an dex deployment
	- configure identity provider connectors
	- administer OIDC client identities

A typical dex deployment consists of N dex-workers behind a load balanacer, and one dex-overlord.
The dex-workers directly handle user requests, so the loss of all workers can result in service downtime.
The single dex-overlord runs its tasks periodically, so it does not need to maintain 100% uptime.

## Connectors

Remote IdPs could implement any auth-N protocol.
*connectors* contain protocol-specific logic and are used to communicate with remote IdPs.
Possible examples of connectors could be: OIDC, LDAP, Local Memory, Basic Auth, etc.
dex ships with an OIDC connector, and a basic "local" connector for in-memory testing purposes.
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

# Building

## With Host Go Environment

`./build`

## With Docker

`./go-docker ./build`

## Docker Build and Push

Binaries must be compiled first.
Builds a docker image and pushes it to the quay repo.
The image is tagged with the git sha and 'latest'.

```
export QUAY_USER=xxx
export QUAY_PASSWORD=yyy
./build-docker-push
```

## Rebuild API from JSON schema

Go API bindings are generated from a JSON Discovery file.
To regenerate run:

```
./schema/generator
```

For updating generator dependencies see docs in: `schema/generator_import.go`.

## Runing Tests

Run all tests: `./test`

Single package only: `PKG=<pkgname> ./test`

Functional tests: `./test-functional`

Run with docker:

```
./go-docker ./test
./go-docker ./test-functional
```

# Running

Run the main dex server:

After building, run `./bin/dex` and provider the required arguments.
Additionally start `./bin/dex-overlord` for key rotation and database garbage collection.

# Deploying

Generate systemd unit files by injecting secrets into the unit file templates located in: `./static/...`.

```
source <path-to-secure>/prod/dex.env.txt
./build-units
```

Resulting unit files are output to: `./deploy`

# Registering Clients

Like all OAuth2 servers clients must be registered with a callback url.
New clients can be registered with the dexctl CLI tool:
```
dexctl --db-url=postgres://localhost/auth?sslmode=disable new-client http://example.com/auth/callback
```

The tool will print the `client-id` and `client-secret` to stdout; you must save these for use in your client application. The output of this command is "KEY=VALUE" format, so If you `eval` it in your shell, the relevant variables are available to use.

Note that for the initial invocation of `dexctl` you need to provide a DSN URL to create a new-client. Once you have created this initial client, you can use its client-id and client-secret as credentials to dexctl, and make requests via the HTTP API instead of the DB:

```
dexctl --endpoint=http://your-issuer-url --client-id=your_client_id --client-secret=your_client_secret new-client
```

or, if you want to go the eval route:
```
eval "$(dexctl --endpoint=http://your-issuer-url --client-id=your_client_id --client-secret=your_client_secret new-client)"
```

The latter form makes the variables `DEX_APP_CLIENT_ID`, `DEX_APP_CLIENT_SECRET` and `DEX_APP_REDIRECTURL_0` available to your shell.

This will allow you to create new clients from machines that cannot hit the database.

# Standup Dev Script

A script which will create a database, create a client, start an overlord and a worker and start the example app exists at `contrib/standup-db.sh`.

# Coming Soon

- Multiple backing Identity Providers
- Identity Management
- Authorization
