 Authentication through Atlassian Crowd

## Overview

Atlassian Crowd is a centralized identity management solution providing single sign-on and user identity.

Current connector uses request to [Crowd REST API](https://developer.atlassian.com/server/crowd/json-requests-and-responses/) endpoints:
* `/user` - to get user-info
* `/session` - to authenticate the user

Offline Access scope support provided with a new request to user authentication and user info endpoints. 

## Configuration
To start using the Atlassian Crowd connector, firstly you need to register an application in your Crowd like specified in the [docs](https://confluence.atlassian.com/crowd/adding-an-application-18579591.html).

The following is an example of a configuration for dex `examples/config-dev.yaml`:

```yaml
connectors:
- type: atlassian-crowd
  # Required field for connector id.
  id: crowd
  # Required field for connector name.
  name: Crowd
  config:
    # Required field to connect to Crowd.
    baseURL: https://crowd.example.com/crowd
    # Credentials can be string literals or pulled from the environment.
    clientID: $ATLASSIAN_CROWD_APPLICATION_ID
    clientSecret: $ATLASSIAN_CROWD_CLIENT_SECRET
    # Optional groups whitelist, communicated through the "groups" scope.
    # If `groups` is omitted, all of the user's Crowd groups are returned when the groups scope is present.
    # If `groups` is provided, this acts as a whitelist - only the user's Crowd groups that are in the configured `groups` below will go into the groups claim.  
    # Conversely, if the user is not in any of the configured `groups`, the user will not be authenticated.
    groups:
    - my-group
    # Prompt for username field.
    usernamePrompt: Login
    # Optionally set preferred_username claim.
    # If `preferredUsernameField` is omitted or contains an invalid option, the `preferred_username` claim will be empty.
    # If `preferredUsernameField` is set, the `preferred_username` claim will be set to the chosen Crowd user attribute value.
    # Possible choices are: "key", "name", "email"
    preferredUsernameField: name
```
