# Authentication through Gitlab

## Overview

GitLab is a web-based Git repository manager with wiki and issue tracking features, using an open source license, developed by GitLab Inc. One of the login options for dex uses the GitLab OAuth2 flow to identify the end user through their GitLab account. You can use this option with [gitlab.com](gitlab.com), GitLab community or enterprise edition.

When a client redeems a refresh token through dex, dex will re-query GitLab to update user information in the ID Token. To do this, __dex stores a readonly GitLab access token in its backing datastore.__ Users that reject dex's access through GitLab will also revoke all dex clients which authenticated them through GitLab.

## Configuration

Register a new application via `User Settings -> Applications` ensuring the callback URL is `(dex issuer)/callback`. For example if dex is listening at the non-root path `https://auth.example.com/dex` the callback would be `https://auth.example.com/dex/callback`.

The following is an example of a configuration for `examples/config-dev.yaml`:

```yaml
connectors:
  - type: gitlab
    # Required field for connector id.
    id: gitlab
    # Required field for connector name.
    name: GitLab
    config:
      # optional, default = https://www.gitlab.com 
      baseURL: https://www.gitlab.com
      # Credentials can be string literals or pulled from the environment.  
      clientID: $GITLAB_APPLICATION_ID 
      clientSecret: $GITLAB_CLIENT_SECRET
      redirectURI: http://127.0.0.1:5556/dex/callback
```
