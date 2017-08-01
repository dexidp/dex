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
      # Credentials can be string literals or pulled from the environment.  
      clientID: $GITLAB_APPLICATION_ID
      clientSecret: $GITLAB_CLIENT_SECRET
      redirectURI: http://127.0.0.1:5556/dex/callback
      # Optional group, communicate through the "groups" scope.
      #
      # NOTE: This is an EXPERIMENTAL config option and will likely change.
      group: my-group
```

Users can use their GitLab Enterprise account to login to dex. The following configuration can be used to enable a GitLab Enterprise connector on dex:

```yaml
connectors:
- type: gitlab
  # Required field for connector id.
  id: gitlab
  # Required field for connector name.
  name: GitLab
  config:
    # Required fields. Dex must be pre-registered with GitLab Enterprise
    # to get the following values.
    # Credentials can be string literals or pulled from the environment.
    clientID: $GITLAB_CLIENT_ID
    clientSecret: $GITLAB_CLIENT_SECRET
    redirectURI: http://127.0.0.1:5556/dex/callback
    # Optional group, communicate through the "groups" scope.
    #
    # NOTE: This is an EXPERIMENTAL config option and will likely change.
    group: my-group

    # Required ONLY for GitLab Enterprise.
    # This is the Hostname of the GitLab Enterprise account listed on the
    # management console. Ensure this domain is routable on your network.
    hostName: git.example.com
    # ONLY for GitLab Enterprise. Optional field.
    # Used to support self-signed or untrusted CA root certificates.
    rootCA: /etc/dex/ca.crt
```
