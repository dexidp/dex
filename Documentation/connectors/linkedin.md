# Authentication through LinkedIn

## Overview

One of the login options for dex uses the LinkedIn OAuth2 flow to identify the end user through their LinkedIn account.

When a client redeems a refresh token through dex, dex will re-query LinkedIn to update user information in the ID Token. To do this, __dex stores a readonly LinkedIn access token in its backing datastore.__ Users that reject dex's access through LinkedIn will also revoke all dex clients which authenticated them through LinkedIn.

## Configuration

Register a new application via `My Apps -> Create Application` ensuring the callback URL is `(dex issuer)/callback`. For example if dex is listening at the non-root path `https://auth.example.com/dex` the callback would be `https://auth.example.com/dex/callback`.

The following is an example of a configuration for `examples/config-dev.yaml`:

```yaml
connectors:
  - type: linkedin
    # Required field for connector id.
    id: linkedin
    # Required field for connector name.
    name: LinkedIn
    config:
      # Credentials can be string literals or pulled from the environment.
      clientID: $LINKEDIN_APPLICATION_ID
      clientSecret: $LINKEDIN_CLIENT_SECRET
      redirectURI: http://127.0.0.1:5556/dex/callback
```
