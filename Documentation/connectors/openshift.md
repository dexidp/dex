# Authentication using OpenShift

## Overview

Dex can make use of users and groups defined within OpenShift by querying the platform provided OAuth server.

## Configuration


### Creating an OAuth Client

Two forms of OAuth Clients can be utilized:

* [Using a Service Account as an OAuth Client](https://docs.openshift.com/container-platform/latest/authentication/using-service-accounts-as-oauth-client.html) (Recommended)
* [Registering An Additional OAuth Client](https://docs.openshift.com/container-platform/latest/authentication/configuring-internal-oauth.html#oauth-register-additional-client_configuring-internal-oauth)

#### Using a Service Account as an OAuth Client

OpenShift Service Accounts can be used as a constrained form of OAuth client. Making use of a Service Account to represent an OAuth Client is the recommended option as it does not require elevated privileged within the OpenShift cluster. Create a new Service Account or make use of an existing Service Account.

Patch the Service Account to add an annotation for location of the Redirect URI

```
oc patch serviceaccount <name> --type='json' -p='[{"op": "add", "path": "/metadata/annotations/serviceaccounts.openshift.io~1oauth-redirecturi.dex", "value":"https:///<dex_url>/callback"}]'
```

The Client ID for a Service Account representing an OAuth Client takes the form `

The Client Secret for a Service Account representing an OAuth Client is the long lived OAuth Token that is configued for the Service Account. Execute the following command to retrieve the OAuth Token.

```
oc serviceaccounts get-token <name>
```

#### Registering An Additional OAuth Client

Instead of using a constrained form of Service Account to represent an OAuth Client, an additional OAuthClient resource can be created.

Create a new OAuthClient resource similar to the following:

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

### Dex Configuration

The following is an example of a configuration for `examples/config-dev.yaml`:

```yaml
connectors:
  - type: openshift
    # Required field for connector id.
    id: openshift
    # Required field for connector name.
    name: OpenShift
    config:
      # OpenShift API
      issuer: https://api.mycluster.example.com:6443
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
