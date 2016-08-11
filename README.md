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

## Getting started

dex requires a Go installation and a GOPATH configured. Clone it down the
correct place, and simply type `make` to compile dex.

```
git clone https://github.com/coreos/dex.git $GOPATH/src/github.com/coreos/dex
cd $GOPATH/src/github.com/coreos/dex
git checkout dev
make
```

dex is a single, scalable binary that pulls all configuration from a config
file (no command line flags at the moment). Use one of the config files defined
in the `examples` folder to start up dex with an in-memory data store.

```
./bin/dex serve examples/config-dev.yaml
```

dex allows OAuth2 clients to be defined statically through the config file. In
another window, run the `example-app` (an OAuth2 client). By default this is
configured to use the client ID and secret defined in the config file.

```
./bin/example-app
```

Then to interact with dex, like any other OAuth2 provider, you must first visit
a client app, then be prompted to login through dex. This can be achieved using
the following steps:

NOTE: The UIs are extremely bare bones at the moment.

1. Navigate to http://localhost:5555/ in your browser.
2. Hit "login" on the example app to be redirected to dex.
3. Choose the "mock" option to login as a predefined user.
4. Approve the example app's request.
5. See the resulting token the example app claims from dex.
