# Custom scopes, claims and client features

This document describes the set of OAuth2 and OpenID Connect features implemented by dex.

## Scopes

The following is the exhaustive list of scopes supported by dex:

| Name | Description |
| ---- | ------------|
| `openid` | Required scope for all login requests. |
| `email` | ID token claims should include the end user's email and if that email was verified by an upstream provider. |
| `profile` | ID token claims should include the username of the end user. |
| `groups` | ID token claims should include a list of groups the end user is a member of. |
| `federated:id` | ID token claims should include information from the ID provider. The token will contain the connector ID and the user ID assigned at the provider. |
| `offline_access` | Token response should include a refresh token. Doesn't work in combinations with some connectors, notability the [SAML connector][saml-connector] ignores this scope. |
| `audience:server:client_id:( client-id )` | Dynamic scope indicating that the ID token should be issued on behalf of another client. See the _"Cross-client trust and authorized party"_ section below. |

## Custom claims

Beyond the [required OpenID Connect claims][core-claims], and a handful of [standard claims][standard-claims], dex implements the following non-standard claims.

| Name | Description |
| ---- | ------------|
| `groups` | A list of strings representing the groups a user is a member of. |
| `federated_claims` | The connector ID and the user ID assigned to the user at the provider. |
| `email` | The email of the user. |
| `email_verified` | If the upstream provider has verified the email. |
| `name` | User's display name. |

The `federated_claims` claim has the following format:

```json
"federated_claims": {
  "connector_id": "github",
  "user_id": "110272483197731336751"
}
```

## Cross-client trust and authorized party

Dex has the ability to issue ID tokens to clients on behalf of other clients. In OpenID Connect terms, this means the ID token's `aud` (audience) claim being a different client ID than the client that performed the login.

For example, this feature could be used to allow a web app to generate an ID token on behalf of a command line tool:

```yaml
staticClients:
- id: web-app
  redirectURIs:
  - 'https://web-app.example.com/callback'
  name: 'Web app'
  secret: web-app-secret

- id: cli-app
  redirectURIs:
  - 'https://cli-app.example.com/callback'
  name: 'Command line tool'
  secret: cli-app-secret
  # The command line tool lets the web app issue ID tokens on its behalf.
  trustedPeers:
  - web-app
```

Note that the command line tool must explicitly trust the web app using the `trustedPeers` field. The web app can then use the following scope to request an ID token that's issued for the command line tool.

```
audience:server:client_id:cli-app
```

The ID token claims will then include the following audience and authorized party:

```
{
    "aud": "cli-app",
    "azp": "web-app",
    "email": "foo@bar.com",
    // other claims...
}
``` 

## Public clients

Public clients are inspired by Google's [_"Installed Applications"_][installed-apps] and are meant to impose restrictions on applications that don't intend to keep their client secret private. Clients can be declared as public using the `public` config option.

```yaml
staticClients:
- id: cli-app
  public: true
  name: 'CLI app'
  secret: cli-app-secret
```

Instead of traditional redirect URIs, public clients are limited to either redirects that begin with "http://localhost" or a special "out-of-browser" URL "urn:ietf:wg:oauth:2.0:oob". The latter triggers dex to display the OAuth2 code in the browser, prompting the end user to manually copy it to their app. It's the client's responsibility to either create a screen or a prompt to receive the code, then perform a code exchange for a token response.

When using the "out-of-browser" flow, an ID Token nonce is strongly recommended.

[saml-connector]: saml-connector.md
[core-claims]: https://openid.net/specs/openid-connect-core-1_0.html#IDToken
[standard-claims]: https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims
[installed-apps]: https://developers.google.com/api-client-library/python/auth/installed-app
