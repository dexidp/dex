# Authentication through Microsoft

## Overview

One of the login options for dex uses the Microsoft OAuth2 flow to identify the
end user through their Microsoft account.

When a client redeems a refresh token through dex, dex will re-query Microsoft
to update user information in the ID Token. To do this, __dex stores a readonly
Microsoft access and refresh tokens in its backing datastore.__ Users that
reject dex's access through Microsoft will also revoke all dex clients which
authenticated them through Microsoft.

## Configuration

Register a new application on https://apps.dev.microsoft.com via `Add an app`
ensuring the callback URL is `(dex issuer)/callback`. For example if dex
is listening at the non-root path `https://auth.example.com/dex` the callback
would be `https://auth.example.com/dex/callback`.

The following is an example of a configuration for `examples/config-dev.yaml`:

```yaml
connectors:
  - type: microsoft
    # Required field for connector id.
    id: microsoft
    # Required field for connector name.
    name: Microsoft
    config:
      # Credentials can be string literals or pulled from the environment.
      clientID: $MICROSOFT_APPLICATION_ID
      clientSecret: $MICROSOFT_CLIENT_SECRET
      redirectURI: http://127.0.0.1:5556/dex/callback
```

`tenant` configuration parameter controls what kinds of accounts may be
authenticated in dex. By default, all types of Microsoft accounts (consumers
and organizations) can authenticate in dex via Microsoft. To change this, set
the `tenant` parameter to one of the following:

- `common`- both personal and business/school accounts can authenticate in dex
  via Microsoft (default)
- `consumers` - only personal accounts can authenticate in dex
- `organizations` - only business/school accounts can authenticate in dex
- `<tenant uuid>` or `<tenant name>` - only accounts belonging to specific
  tenant identified by either `<tenant uuid>` or `<tenant name>` can
  authenticate in dex

For example, the following snippet configures dex to only allow business/school
accounts:

```yaml
connectors:
  - type: microsoft
    # Required field for connector id.
    id: microsoft
    # Required field for connector name.
    name: Microsoft
    config:
      # Credentials can be string literals or pulled from the environment.
      clientID: $MICROSOFT_APPLICATION_ID
      clientSecret: $MICROSOFT_CLIENT_SECRET
      redirectURI: http://127.0.0.1:5556/dex/callback
      tenant: organizations
```
