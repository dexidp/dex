# Authentication through GitHub

## Overview

One of the login options for dex uses the GitHub OAuth2 flow to identify the end user through their GitHub account.

When a client redeems a refresh token through dex, dex will re-query GitHub to update user information in the ID Token. To do this, __dex stores a readonly GitHub access token in its backing datastore.__ Users that reject dex's access through GitHub will also revoke all dex clients which authenticated them through GitHub.

## Caveats

* A user must explicitly [request][github-request-org-access] an [organization][github-orgs] give dex [resource access][github-approve-org-access]. Dex will not have the correct permissions to determine if the user is in that organization otherwise, and the user will not be able to log in. This request mechanism is a feature of the GitHub API.

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

    # Optional organizations and teams, communicated through the "groups" scope.
    #
    # NOTE: This is an EXPERIMENTAL config option and will likely change.
    #
    # Legacy 'org' field. 'org' and 'orgs' cannot be used simultaneously. A user
    # MUST be a member of the following org to authenticate with dex.
    # org: my-organization
    #
    # Dex queries the following organizations for group information if the
    # "groups" scope is provided. Group claims are formatted as "(org):(team)".
    # For example if a user is part of the "engineering" team of the "coreos"
    # org, the group claim would include "coreos:engineering".
    #
    # If orgs are specified in the config then user MUST be a member of at least one of the specified orgs to
    # authenticate with dex.
    #
    # If neither 'org' nor 'orgs' are specified in the config and 'loadAllGroups' setting set to true then user
    # authenticate with ALL user's Github groups. Typical use case for this setup:
    # provide read-only access to everyone and give full permissions if user has 'my-organization:admins-team' group claim.  
    orgs:
    - name: my-organization
      # Include all teams as claims.
    - name: my-organization-with-teams
      # A white list of teams. Only include group claims for these teams.
      teams:
      - red-team
      - blue-team
    # Flag which indicates that all user groups and teams should be loaded.
    loadAllGroups: false

    # Optional choice between 'name' (default), 'slug', or 'both'.
    #
    # As an example, group claims for member of 'Site Reliability Engineers' in
    # Acme organization would yield:
    #   - ['acme:Site Reliability Engineers'] for 'name'
    #   - ['acme:site-reliability-engineers'] for 'slug'
    #   - ['acme:Site Reliability Engineers', 'acme:site-reliability-engineers'] for 'both'
    teamNameField: slug
    # flag which will switch from using the internal GitHub id to the users handle (@mention) as the user id.
    # It is possible for a user to change their own user name but it is very rare for them to do so
    useLoginAsID: false
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
    # Optional organizations and teams, communicated through the "groups" scope.
    #
    # NOTE: This is an EXPERIMENTAL config option and will likely change.
    #
    # Legacy 'org' field. 'org' and 'orgs' cannot be used simultaneously. A user
    # MUST be a member of the following org to authenticate with dex.
    # org: my-organization
    #
    # Dex queries the following organizations for group information if the
    # "groups" scope is provided. Group claims are formatted as "(org):(team)".
    # For example if a user is part of the "engineering" team of the "coreos"
    # org, the group claim would include "coreos:engineering".
    #
    # A user MUST be a member of at least one of the following orgs to
    # authenticate with dex.
    orgs:
    - name: my-organization
      # Include all teams as claims.
    - name: my-organization-with-teams
      # A white list of teams. Only include group claims for these teams.
      teams:
      - red-team
      - blue-team
    # Required ONLY for GitHub Enterprise.
    # This is the Hostname of the GitHub Enterprise account listed on the
    # management console. Ensure this domain is routable on your network.
    hostName: git.example.com
    # ONLY for GitHub Enterprise. Optional field.
    # Used to support self-signed or untrusted CA root certificates.
    rootCA: /etc/dex/ca.crt
```

[github-oauth2]: https://github.com/settings/applications/new
[github-orgs]: https://developer.github.com/v3/orgs/
[github-request-org-access]: https://help.github.com/articles/requesting-organization-approval-for-oauth-apps/
[github-approve-org-access]: https://help.github.com/articles/approving-oauth-apps-for-your-organization/
