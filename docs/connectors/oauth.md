# Authentication using Generic OAuth 2.0 provider

## Overview

Dex users can make use of this connector to work with standards-compliant [OAuth 2.0](https://oauth.net/2/) authorization provider, in case of that authorization provider is not in the Dex connectors list.

## Configuration

The following is an example of a configuration for using OAuth connector with Reddit.

```yaml
connectors:
- type: oauth
  # ID of OAuth 2.0 provider
  id: reddit 
  # Name of OAuth 2.0 provider
  name: reddit
  config:
    # Connector config values starting with a "$" will read from the environment.
    clientID: $REDDIT_CLIENT_ID
    clientSecret: $REDDIT_CLIENT_SECRET
    redirectURI: http://127.0.0.1:5556/callback

    tokenURL: https://www.reddit.com/api/v1/access_token
    authorizationURL: https://www.reddit.com/api/v1/authorize
    userInfoURL: https: https://www.reddit.com/api/v1/me
 
    # Optional: Specify whether to communicate to Auth provider without validating SSL certificates
    # insecureSkipVerify: false

    # Optional: The location of file containing SSL certificates to commmunicate to Auth provider 
    # rootCAs: /etc/ssl/reddit.pem

    # Optional: List of scopes to request Auth provider for access user account
    # scopes:
    #  - identity

    # Optional: Configurable keys for user ID look up
    # Default: id
    # userIDKey:

    # Auth roviders return non-standard user identity profile
    # Use claimMapping to map those user infomations to standard claims:
    claimMapping:
      # Optional: Configurable keys for user name look up
      # Default: user_name
      # userNameKey:

      # Optional: Configurable keys for preferred username look up
      # Default: preferred_username
      # preferredUsernameKey:

      # Optional: Configurable keys for user groups look up
      # Default: groups
      # groupsKey:

      # Optional: Configurable keys for email look up
      # Default: email
      # emailKey:

      # Optional: Configurable keys for email verified look up
      # Default: email_verified
      # emailVerifiedKey:
```
