# Authentication through an OpenID Connect provider

## Overview

Dex is able to use another OpenID Connect provider as an authentication source. When logging in, dex will redirect to the upstream provider and perform the necessary OAuth2 flows to determine the end users email, username, etc. More details on the OpenID Connect protocol can be found in [_An overview of OpenID Connect_][oidc-doc].

Prominent examples of OpenID Connect providers include Google Accounts, Salesforce, and Azure AD v2 ([not v1][azure-ad-v1]).

## Caveats

This connector does not support the "groups" claim. Progress for this is tracked in [issue #1065][issue-1065].

When using refresh tokens, changes to the upstream claims aren't propegated to the id_token returned by dex. If a user's email changes, the "email" claim returned by dex won't change unless the user logs in again. Progress for this is tracked in [issue #863][issue-863].

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
    
    # Google supports whitelisting allowed domains when using G Suite
    # (Google Apps). The following field can be set to a list of domains
    # that can log in:
    #
    # hostedDomains:
    #  - example.com
```

[oidc-doc]: openid-connect.md
[issue-863]: https://github.com/coreos/dex/issues/863
[issue-1065]: https://github.com/coreos/dex/issues/1065
[azure-ad-v1]: https://github.com/coreos/go-oidc/issues/133
