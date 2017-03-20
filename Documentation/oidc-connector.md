# Authentication through an OpenID Connect provider

## Overview

Dex is able to use another OpenID Connect provider as an authentication source. When logging in, dex will redirect to the upstream provider and perform the necessary OAuth2 flows to determine the end users email, username, etc. More details on the OpenID Connect protocol can be found in [_An overview of OpenID Connect_][oidc-doc].

Prominent examples of OpenID Connect providers include Google Accounts, Salesforce, and Azure AD v2 ([not v1][azure-ad-v1]).

## Caveats

Many OpenID Connect providers implement different restrictions on refresh tokens. For example, Google will only issue the first login attempt a refresh token, then not return one after. Because of this, this connector does not refresh the id_token claims when a client of dex redeems a refresh token, which can result in stale user info.

It's generally recommended to avoid using refresh tokens with the `oidc` connector.

Progress on this caveat can be tracked in [issue #863][google-refreshing].

## Configuration

```yaml
connectors:
- type: oidc
  id: google
  name: Google
  config:
    # Canonical URL of the provider, also used for configuration discovery.
    # This value MUST match the value returned in the provider config discovery.
    #
    # See: https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderConfig
    issuer: https://accounts.google.com

    # Connector config values starting with a "$" will read from the environment.
    clientID: $GOOGLE_CLIENT_ID
    clientSecret: $GOOGLE_CLIENT_SECRET

    # Dex's issuer URL + "/callback"
    redirectURI: http://127.0.0.1:5556/callback


    # Some providers require passing client_secret via POST parameters instead
    # of basic auth, despite the OAuth2 RFC discouraging it. Many of these
    # cases are caught internally, but some may need to uncommented the
    # following field.
    #
    # basicAuthUnsupported: true
```

[oidc-doc]: openid-connect.md
[google-refreshing]: https://github.com/coreos/dex/issues/863
[azure-ad-v1]: https://github.com/coreos/go-oidc/issues/133
