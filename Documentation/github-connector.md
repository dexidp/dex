# Authentication through GitHub

## Overview

One of the login options for dex uses the GitHub OAuth2 flow to identify the end user through their GitHub account.

When a client redeems a refresh token through dex, dex will re-query GitHub to update user information in the ID Token. To do this, __dex stores a readonly GitHub access token in its backing datastore.__ Users that reject dex's access through GitHub will also revoke all dex clients which authenticated them through GitHub.

## Configuration

Register a new application with [GitHub][github-oauth2] ensuring the callback URL is `(dex issuer)/callback`. For example if dex is listening at the non-root path `https://auth.example.com/dex` the callback would be `https://auth.example.com/dex/callback`.

The following is an example of a configuration for `examples/config-dev.yaml`:

```yaml
connectors:
- type: github
  # Required field for connector id.
  id: github
  # Required field for connector name.
  name: GitHub
  config:
    # Credentials can be string literals or pulled from the environment.
    clientID: $GITHUB_CLIENT_ID
    clientSecret: $GITHUB_CLIENT_SECRET
    redirectURI: http://127.0.0.1:5556/dex/callback
    # Optional organization to pull teams from, communicate through the
    # "groups" scope.
    #
    # NOTE: This is an EXPERIMENTAL config option and will likely change.
    org: my-oranization
```

[github-oauth2]: https://github.com/settings/applications/new
