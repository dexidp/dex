# oidc-proxy

The OpenID Connect proxy is an authentication proxy that can be used with any OpenID Connect provider, including dex. It acts as a reverse proxy while requiring end users to first authenticate with a provider before accessing the backend.

## Setup

`oidc-proxy` is deployed in front of a backend service, alongside dex. When end users make a request to the proxy, if they haven't authenticated with the dex, they're redirect and asked to login.

![](img/oidc-proxy.png)

## Authorization

`oidc-proxy` supports simple authorization based on a user's email, either through an email or domain whitelist. For example, to allow only users with an `@example.com` email address, provide the following flag:

```
--allow-domains=example.com
```

Or to whitelist `admin@example.com` and `webmaster@example.com`, use:

```
--allow-emails=admin@example.com,webmaster@example.com
```

If neither of these flags are provided, any user who can authenticate through the provider is allowed to login to the service.

In the future, we hope to allow more pluggable authorization policies, either by providing webhooks or allowing `oidc-proxy` to be importable as a package.

## Example

First, compile the `dex` and `oidc-proxy` binaries.

```
make
```

In one terminal, start a dex server:

```
./bin/dex serve examples/config-dev.yaml
```

In another run a backend service:

```
python3 -m http.server
```

Finally run `oidc-proxy`, authenticating against dex and proxying the backend service:

```
./bin/oidc-proxy \
  --allow-emails=jane@example.com \
  --backend-url=http://localhost:8000 \
  --client-id=oidc-proxy \
  --client-secret=oidc-proxy-secret \
  --listen-http=127.0.0.1:5558 \
  --redirect-uri=http://127.0.0.1:5558/callback \
  --issuer-url=http://127.0.0.1:5556/dex
```

Visiting [http://127.0.0.0:5558](http://127.0.0.1:5558) will take you to the proxy. Seeing you haven't logged in, it will immediately redirect you to dex.

Logging in with username `jane@example.com` and password `password`, the proxy will let you through to the backend.


## Session secrets

When running multiple instances of `oidc-proxy`, users must ensure the session secrets is the same for the each instance. The secret must be 32 random bytes base64 encoded.

```bash
$ head -c 32 /dev/urandom | base64
UaTLQOxo6hUjUWD6EWdFC0fRySsTN/A4Ti3YUhc+qe4=
```

This can be passed to each instance of oidc-proxy using the following flag:

```
--session-secret=UaTLQOxo6hUjUWD6EWdFC0fRySsTN/A4Ti3YUhc+qe4=
```
