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
| `offline_access` | Token response should include a refresh token. |
| `audience:server:client_id:( client-id )` | Dynamic scope indicating that the ID token should be issued on behalf of another client. See the _"Cross-client trust and authorized party"_ section below. |

## Custom claims

Beyond the [required OpenID Connect claims][core-claims], and a handful of [standard claims][standard-claims], dex implements the following non-standard claims.

| Name | Description |
| ---- | ------------|
| `groups` | A list of strings representing the groups a user is a member of. |
| `dex.coreos.com/login` | The strategy used to authenticate the user. Examples include "ldap", "github", "saml", etc. |

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

[core-claims]: https://openid.net/specs/openid-connect-core-1_0.html#IDToken
[standard-claims]: https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims
