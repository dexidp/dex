# Example app

An OpenID Connect client for trying dex out. It runs the flows dex implements,
shows what comes back, and — given `--grpc-addr` — talks to dex's management API.

```
go run ./examples/example-app
```

It listens on `http://127.0.0.1:5555` and expects dex at `http://127.0.0.1:5556/dex`.

## What it does

**Browser flows.** Authorization code with PKCE, and device code. Each
authorization gets its own `state`, `nonce` and PKCE verifier, kept against the
browser's session — that is what the callback is checked against.

**Direct grants.** Refresh, client credentials, password and token exchange,
each from a form on the front page.

**Token tools.** Introspection, local signature verification, and UserInfo.
The first two answer different questions: introspection asks dex whether it is
still honouring a token, verification checks the signature and lifetime the way
a resource server would. A revoked token passes the second and fails the first.

**Sessions.** The app keeps a session per browser and, by default, re-checks
every 30s with a `prompt=none` request that dex still has one. Sign out of dex
in another tab and this app stops showing you as signed in.

## Configuring dex for it

`examples/config-dev.yaml` is set up for all of this already — sessions, the
device callback, the password connector and the gRPC API. What each part is for,
if you are writing your own config:

```yaml
oauth2:
  # Grants are refused unless listed. The default list is authorization_code
  # and refresh_token.
  grantTypes:
    - authorization_code
    - refresh_token
    - urn:ietf:params:oauth:grant-type:device_code
    - client_credentials
    - password
    - urn:ietf:params:oauth:grant-type:token-exchange
  # Without this the password grant is refused: dex needs to know which
  # connector verifies the password.
  passwordConnector: local

# Sessions are what make single sign-on, the home page and prompt=none session
# checks work. Also needs DEX_SESSIONS_ENABLED=true.
sessions:
  cookieName: dex_session

staticClients:
  - id: example-app
    secret: ZXhhbXBsZS1hcHAtc2VjcmV0
    name: Example App
    redirectURIs:
      - http://127.0.0.1:5555/callback
      # The device flow redirects through dex itself, so this has to be
      # registered too — with the issuer's path prefix.
      - /dex/device/callback

# Only needed for the gRPC API page.
grpc:
  addr: 127.0.0.1:5557
```

Token exchange also needs a connector that can verify the token you bring, which
in dex means one implementing `TokenIdentityConnector`.

## The gRPC API page

```
--grpc-addr 127.0.0.1:5557
```

The page is in sections: clients, local passwords, connectors, the identities
dex has recorded, and a user's sessions and refresh tokens. Without
`--grpc-addr` the page still exists and says what to pass to connect it.

The connection is plaintext by default, which is how dex's example config
exposes the API locally. For anything else, pass certificates — see
`examples/grpc-client/cert-gen` for generating a set:

```
--grpc-addr 127.0.0.1:5557
--grpc-ca examples/grpc-client/ca.crt
--grpc-client-cert examples/grpc-client/client.crt
--grpc-client-key examples/grpc-client/client.key
```

Revoking a refresh token is worth trying against your own session: revoke it,
then let the app refresh, and the sign-in ends.

## Flags

Run `--help` for the full list. The ones worth knowing:

| Flag | Default | |
|---|---|---|
| `--issuer` | `http://127.0.0.1:5556/dex` | Where dex is. |
| `--listen` | `http://127.0.0.1:5555` | Where this app serves. |
| `--redirect-uri` | `http://127.0.0.1:5555/callback` | Also decides the path the callback is served on. |
| `--pkce` | `true` | Send a PKCE challenge. |
| `--session-check-interval` | `30s` | How stale the app lets its idea of dex's session get. `0` stops checking, and then it will show a user who has signed out. |
| `--grpc-addr` | — | Enables the API page. |
| `--debug` | `false` | Log every request to and from dex. |
