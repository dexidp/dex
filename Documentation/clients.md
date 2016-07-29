# Clients (\*aka Relying Parties)

## Configuration

Clients can be created in two different ways:

1. Through the [bootstrap API.](https://github.com/coreos/dex/tree/master/schema/adminschema)
1. Through the [Dynamic Registration API.](https://openid.net/specs/openid-connect-registration-1_0.html) That endpoint is hosted at `/registration`


## Dex Features

Dex contains some client features that are not in any OIDC spec, but can be very useful.

## Cross Client Authorization

Inspired by Google's [Cross-Client Identity](https://developers.google.com/identity/protocols/CrossClientAuth), dex also has a way of having one client mint tokens for other ones, called Cross-client authorization.

A client can only mint JWTs for another client if it is a *trusted peer* of that other client. Currently the only way to set trusted peers is the [bootstrap API](https://github.com/coreos/dex/tree/master/schema/adminschema).

To initiate cross-client authentication, add one more scopes of the following form to the initial auth request:

```
audience:server:client_id:$OTHER_CLIENT_ID
```

OTHER\_CLIENT\_ID is the ID of the client for whom you want a token. You can have multiple such scopes in your request, one for each client whom you want the token to be valid for.

After proceeding as normal with the rest of the auth flow, the resulting ID token will have an `aud` field of only the client ID(s) specified by the scope(s). Note that this means this JWT will not have the initiating client's ID in the `aud`; if you want the client's own ID in the `aud`, you must explicitly request it. A client is always implicitly a trusted client of itself.

## Public Clients

There are times when the confidentiality of the client secret cannot be guaranteed; native mobile clients and command-line tools are common examples.

For these cases, *Public Clients* exist, which have certain restrictions:

1. `http://localhost:$PORT` and `urn:ietf:wg:oauth:2.0:oob` are the only valid redirect URIs.

1. A native client cannot obtain *client credentials* from the `/token` endpoint.

These restrictions are aimed at mitigating certain attacks that can arise as the result of having a non-confidenital client secert.

### Creating a public client.

The only way to create a public client is through the [bootstrap API.](https://github.com/coreos/dex/tree/master/schema/adminschema) There are also special requirements for creating a public client:

* A public client must have a client name specified. This is because client name is used in the creation of the client ID for public clients - in confidential clients, the name is dervied from a redirect URI, which public clients do not have.

* Redirect URIs must not be specified; they are implicit.

## Out-Of-Band Auth Flow

For situations in which an app does not have access to a browser, the out-of-band (oob) flow exists. If you specify "urn:ietf:wg:oauth:2.0:oob" as a redirect URI, after authentication, instead of being redirected to the client site, the user is presented with the auth code in a text field, which they must copy and paste ("out of band" as it were) into their app.


\* In OpenID Connect a client is called a "Relying Party", but "client" seems to
be the more common ter, has been around longer and is present in paramter names
like "client_id" so we prefer it over "Relying Party" usually.

## Groups

Connectors that support groups (currently only the LDAP connector) can embed the groups a user belongs to in the ID Token. Using the scope "groups" during the initial redirect with a connector that supports groups will return an JWT with the following field.

```
{
  "groups": [
    "cn=ipausers,cn=groups,cn=accounts,dc=example,dc=com,
    "cn=team-engineering,cn=groups,cn=accounts,dc=example,dc=com"
  ],
  ...
}
```

If the client has also requested a refresh token, the groups field is updated during each refresh request.
