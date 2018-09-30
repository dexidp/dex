# Authentication through Bitbucket Cloud

## Overview

One of the login options for dex uses the Bitbucket OAuth2 flow to identify the end user through their Bitbucket account.

When a client redeems a refresh token through dex, dex will re-query Bitbucket to update user information in the ID Token. To do this, __dex stores a readonly Bitbucket access token in its backing datastore.__ Users that reject dex's access through Bitbucket will also revoke all dex clients which authenticated them through Bitbucket.

## Configuration

Register a new OAuth consumer with [Bitbucket](https://confluence.atlassian.com/bitbucket/oauth-on-bitbucket-cloud-238027431.html) ensuring the callback URL is `(dex issuer)/callback`. For example if dex is listening at the non-root path `https://auth.example.com/dex` the callback would be `https://auth.example.com/dex/callback`.

The following is an example of a configuration for `examples/config-dev.yaml`:

```yaml
connectors:
- type: bitbucket
  # Required field for connector id.
  id: bitbucket
  # Required field for connector name.
  name: Bitbucket
  config:
    # Credentials can be string literals or pulled from the environment.
    clientID: $BITBUCKET_CLIENT_ID
    clientSecret: BITBUCKET_CLIENT_SECRET
    redirectURI: http://127.0.0.1:5556/dex/callback
    # Optional teams, communicated through the "groups" scope.
    teams:
    - my-team
```
