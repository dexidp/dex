# Getting started

## Building the dex binary

Dex requires a Go installation and a GOPATH configured. For setting up a Go workspace, refer to the [official documentation][go-setup]. Clone it down the correct place, and simply type `make` to compile the dex binary.

```
$ go get github.com/coreos/dex
$ cd $GOPATH/src/github.com/coreos/dex
$ make
```

## Configuration

Dex exclusively pulls configuration options from a config file. Use the [example config][example-config] file found in the `examples/` directory to start an instance of dex with an in-memory data store and a set of predefined OAuth2 clients.

```
./bin/dex serve examples/config-dev.yaml
```

The [example config][example-config] file documents many of the configuration options through inline comments. For extra config options, look at that file.

## Running a client

Dex operates like most other OAuth2 providers. Users are redirected from a client app to dex to login. Dex ships with an example client app (also built with the `make` command), for testing and demos.

By default, the example client is configured with the same OAuth2 credentials defined in `examples/config-dev.yaml` to talk to dex. Running the example app will cause it to query dex's [discovery endpoint][oidc-discovery] and determine the OAuth2 endpoints.

```
./bin/example-app
```

Login to dex through the example app using the following steps.

1. Navigate to the example app in your browser at http://localhost:5555/ in your browser.
2. Hit "login" on the example app to be redirected to dex.
3. Choose the "Login with Email" and enter "admin@example.com" and "password"
4. Approve the example app's request.
5. See the resulting token the example app claims from dex.

## Further reading

Check out the Documentation directory for further reading on setting up different storages, interacting with the dex API, intros for OpenID Connect, and logging in through other identity providers such as Google, GitHub, or LDAP.

[go-setup]: https://golang.org/doc/install
[example-config]: ../examples/config-dev.yaml
[oidc-discovery]: https://openid.net/specs/openid-connect-discovery-1_0-17.html#ProviderMetadata
