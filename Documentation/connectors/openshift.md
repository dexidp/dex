# Authentication using OpenShift

## Overview

Dex can make use of users and groups defined within OpenShift by querying the platform provided OAuth server.

## Configuration

Create a new OAuth Client by following the steps described in the documentation for [Registering Additional OAuth Clients[(https://docs.openshift.com/container-platform/latest/authentication/configuring-internal-oauth.html#oauth-register-additional-client_configuring-internal-oauth)

This involves creating a resource similar the following

```yaml
kind: OAuthClient
apiVersion: oauth.openshift.io/v1
metadata:
 name: dex
# The value that should be utilized as the `client_secret`
secret: "<clientSecret>" 
# List of valid addresses for the callback. Ensure one of the values that are provided is `(dex issuer)/callback` 
redirectURIs:
 - "https:///<dex_url>/callback" 
grantMethod: prompt
```

The following is an example of a configuration for `examples/config-dev.yaml`:

```yaml
connectors:
  - type: openshift
    # Required field for connector id.
    id: openshift
    # Required field for connector name.
    name: OppenShift
    config:
      # OpenShift API
      baseURL: https://api.mycluster.example.com:6443
      # Credentials can be string literals or pulled from the environment.
      clientID: $OPENSHIFT_OAUTH_CLIENT_ID
      clientSecret: $OPENSHIFT_OAUTH_CLIENT_SECRET
      redirectURI: http://127.0.0.1:5556/dex/
      # Optional: Specify whether to communicate to OpenShift without validating SSL ceertificates
      insecureCA: false
      # Optional: The location of file containing SSL certificates to commmunicate to OpenShift
      rootCA: /etc/ssl/openshift.pem
      # Optional list of required groups a user mmust be a member of
      groups:
        - users

```