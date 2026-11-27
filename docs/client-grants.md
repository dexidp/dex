# Client grant restrictions

`staticClients[].grantTypes` restricts the OAuth grant types a client may use.
An omitted or empty list preserves the server's existing grant policy. A client
list never enables a grant disabled by `oauth2.grantTypes`.

```yaml
oauth2:
  grantTypes: [authorization_code, refresh_token, client_credentials]
staticClients:
  - id: application
    secretEnv: APPLICATION_CLIENT_SECRET
    grantTypes: [client_credentials]
  - id: interactive
    secretEnv: INTERACTIVE_CLIENT_SECRET
    redirectURIs: [https://app.example.com/callback]
    grantTypes: [authorization_code, refresh_token]
```

The token endpoint authenticates the client before applying its grant restriction
and returns HTTP 400 `unauthorized_client` before processing a disallowed grant.
Authorization requests are also restricted: `code` needs `authorization_code`,
implicit responses need `implicit`, and hybrid responses need both. Device login
uses `urn:ietf:params:oauth:grant-type:device_code`, including its browser callback.
Device polling rechecks the owning client's current restriction. Device codes
created before client ownership was stored retain their existing polling behavior
until they expire; they cannot be associated with a client retroactively.

The gRPC API exposes `grant_types` on `Client` and `ClientInfo`. On
`UpdateClientReq`, omit `grant_types` to keep the current restriction, send
`{grant_types: [...]}` to replace it, or `{}` to clear it. Client restrictions
are independent of connector `grantTypes` and client `allowedConnectors`.

`client_credentials` identifies an application. It does not authenticate an end
user or authorize the application to act as a person. Resource permissions still
come from the resource server's RBAC policy.
