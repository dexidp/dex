# Authentication through Gitlab

## Overview

GitLab is a web-based Git repository manager with wiki and issue tracking features, using an open source license, developed by GitLab Inc. One of the login options for dex uses the GitLab OAuth2 flow to identify the end user through their GitLab account. You can use this option with [gitlab.com](gitlab.com), GitLab community or enterprise edition.

When a client redeems a refresh token through dex, dex will re-query GitLab to update user information in the ID Token. To do this, __dex stores a readonly GitLab access token in its backing datastore.__ Users that reject dex's access through GitLab will also revoke all dex clients which authenticated them through GitLab.

## Configuration

Register a new application via `User Settings -> Applications` ensuring the callback URL is `(dex issuer)/callback`. For example if dex is listening at the non-root path `https://auth.example.com/dex` the callback would be `https://auth.example.com/dex/callback`.

The application requires the user to grant the `read_user` and `openid` scopes. The latter is required only if group membership is a desired claim.

The following is an example of a configuration for `examples/config-dev.yaml`:

```yaml
connectors:
  - type: gitlab
    # Required field for connector id.
    id: gitlab
    # Required field for connector name.
    name: GitLab
    config:
      # optional, default = https://gitlab.com
      baseURL: https://gitlab.com
      # Credentials can be string literals or pulled from the environment.
      clientID: $GITLAB_APPLICATION_ID
      clientSecret: $GITLAB_CLIENT_SECRET
      redirectURI: http://127.0.0.1:5556/dex/callback
      # Optional groups whitelist, communicated through the "groups" scope.
      # If `groups` is omitted, all of the user's GitLab groups are returned when the groups scope is present.
      # If `groups` is provided, this acts as a whitelist - only the user's GitLab groups that are in the configured `groups` below will go into the groups claim.  Conversely, if the user is not in any of the configured `groups`, the user will not be authenticated.
      groups:
      - my-group
      # flag which will switch from using the internal GitLab id to the users handle (@mention) as the user id.
      # It is possible for a user to change their own user name but it is very rare for them to do so
      useLoginAsID: false
```
