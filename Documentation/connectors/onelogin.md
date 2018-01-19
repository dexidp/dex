# Authentication through Onelogin OIDC provider

## Overview

Dex is able to use Onelogin OpenID Connect provider as an authentication source.
When logging in, dex will redirect to the upstream provider and perform the
necessary OAuth2 flows to determine the end users email, username, etc. Further,
roles for the user will be retrieved from the Onelonin API and inserted into the
Groups claim. More details on the OpenID Connect protocol can be found in
[_An overview of OpenID Connect_][oidc-doc].

## Caveats

This work is based on the OIDC connector. It has the same issues with refresh
tokens and not propogating upstream changes until the user logs in again.

## Configuration

```yaml
connectors:
- type: onelogin
  id: onelogin
  name: Onelogin
  config:
    # Subdomain for your Onelogin account.
    # eg: example translates to https://example.onelogin.com/oidc
    subdomain: example

    # Connector config values starting with a "$" will read from the environment.

    # Read only API credentials are required for retrieving user roles.
    # Credentials may be generated [here](https://example.onelogin.com/api_credentials)
    apiId: $ONELOGIN_API_ID
    apiSecret: $ONELOGIN_API_SECRET

    # Client credentials
    clientID: $ONELOGIN_CLIENT_ID
    clientSecret: $ONELOGIN_CLIENT_SECRET

    # Connector config values starting with a "$" will read from the environment.
    clientID: $GOOGLE_CLIENT_ID
    clientSecret: $GOOGLE_CLIENT_SECRET

    # Dex's issuer URL + "/callback"
    redirectURI: http://127.0.0.1:5556/callback

    # To filter groups and trim prefix from upstream group
    # eg. "group_admin" will be returned as "admin"
    rolesPrefix: group_
```

[oidc-doc]: openid-connect.md
