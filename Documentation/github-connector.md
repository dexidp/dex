# Authentication through GitHub

## Overview

One of the login options for dex uses the GitHub OAuth2 flow to identify the end user through their GitHub account.

When a client redeems a refresh token through dex, dex will re-query GitHub to update user information in the ID Token. To do this, __dex stores a readonly GitHub access token in its backing datastore.__ Users that reject dex's access through GitHub will also revoke all dex clients which authenticated them through GitHub.

## Caveats

* Please note that in order for a user to be authenticated via GitHub, the user needs to mark their email id as public on GitHub. This will enable the API to return the user's email to Dex.
* Currently, authentication via GitHub allows users outside of the `Org` specified in the connector to login. This is being tracked by [issue #920][issue-920].

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

## GitHub Enterprise

Users can use their GitHub Enterprise account to login to dex. The following configuration can be used to enable a GitHub Enterprise connector on dex:

```yaml
connectors:
- type: github
  # Required field for connector id.
  id: github
  # Required field for connector name.
  name: GitHub
  config:
    # Required fields. Dex must be pre-registered with GitHub Enterprise
    # to get the following values.
    # Credentials can be string literals or pulled from the environment.
    clientID: $GITHUB_CLIENT_ID
    clientSecret: $GITHUB_CLIENT_SECRET
    redirectURI: http://127.0.0.1:5556/dex/callback
    # Optional organization to pull teams from, communicate through the
    # "groups" scope.
    #
    # NOTE: This is an EXPERIMENTAL config option and will likely change.
    org: my-oranization

    # Required ONLY for GitHub Enterprise.
    # This is the Hostname of the GitHub Enterprise account listed on the
    # management console. Ensure this domain is routable on your network.
    hostName: git.example.com
    # ONLY for GitHub Enterprise. Optional field.
    # Used to support self-signed or untrusted CA root certificates.
    rootCA: /etc/dex/ca.crt
```

[github-oauth2]: https://github.com/settings/applications/new
[issue-920]: https://github.com/coreos/dex/issues/920
