# Configuring Connectors

Connectors connect dex to authentication providers. dex needs to have at least one connector configured so that users can log in.

## Configuration Format

The dex connector configuration format is a JSON array of objects, each with an ID and type, in addition to whatever other configuration is required, like so:

```
[
    {
        "id": "local",
        "type": "local"
    }, {
        "id": "Google",
        "type": "oidc",
        ...<<more config>>...
    } 
]
```

The additional configuration is dependent on the specific type of connector.

### `local` connector

The `local` connector allows email/password based authentication hosted by dex itself. It is special in several ways:

* There can only be one `local` connector in your configuration.
* The id must be `local`
* No other configuration is required

When the `local` connector is present, users can authenticate with the "Log in With Email" button on the authentication screen.

The configuration for the local connector is always the same; it looks like this:

```
    {
        "id": "local",
        "type": "local"
    }
```

### `oidc` connector

This connector config lets users authenticate with other OIDC providers. In addition to `id` and `type`, the `oidc` connector takes the following additional fields:

* issuerURL: a `string`. The base URL for the OIDC provider. Should be a URL with an `https` scheme.

* clientID: a `string`. The OIDC client ID.

* clientSecret: a `string`. The OIDC client secret.

* trustedEmailProvider: a `boolean`. If true dex will trust the email address claims from this provider and not require that users verify their emails.

In order to use the `oidc` connector you must register dex as an OIDC client; this mechanism is different from provider to provider. For Google, follow the instructions at their [developer site](https://developers.google.com/identity/protocols/OpenIDConnect?hl=en). Regardless of your provider, registering your client will also provide you with the client ID and secret.

When registering dex as a client, you need to provide redirect URLs to the provider. dex requires just one:

```
https://$DEX_HOST:$DEX_PORT/auth/$CONNECTOR_ID/callback
```

`$DEX_HOST` and `$DEX_PORT` are the host and port of your dex installation. `$CONNECTOR_ID` is the `id` field of the connector for this OIDC provider.

Here's what a `oidc` connector looks like configured for authenticating with Google; the clientID and clientSecret shown are not usable. We consider Google a trusted email provider because the email address that is present in claims is for a Google provisioned email account (eg. an `@gmail.com` address)

```
    {
        "type": "oidc",
        "id": "google",
        "issuerURL": "https://accounts.google.com",
        "clientID": "$DEX_GOOGLE_CLIENT_ID",
        "clientSecret": "$DEX_GOOGLE_CLIENT_SECRET",
        "trustedEmailProvider": true
    }
```

### `github` connector

This connector config lets users authenticate through [GitHub](https://github.com/). In addition to `id` and `type`, the `github` connector takes the following additional fields:

* clientID: a `string`. The GitHub OAuth application client ID.

* clientSecret: a `string`. The GitHub OAuth application client secret.

To begin, register an OAuth application with GitHub through your, or your organization's [account settings](ttps://github.com/settings/applications/new). To register dex as a client of your GitHub application, enter dex's redirect URL under 'Authorization callback URL':

```
https://$DEX_HOST:$DEX_PORT/auth/$CONNECTOR_ID/callback
```

`$DEX_HOST` and `$DEX_PORT` are the host and port of your dex installation. `$CONNECTOR_ID` is the `id` field of the connector.

Here's an example of a `github` connector; the clientID and clientSecret should be replaced by values provided by GitHub.

```
    {
        "type": "github",
        "id": "github",
        "clientID": "$DEX_GITHUB_CLIENT_ID",
        "clientSecret": "$DEX_GITHUB_CLIENT_SECRET"
    }
```

The `github` connector requests read only access to user's email through the [`user:email` scope](https://developer.github.com/v3/oauth/#scopes).

### `bitbucket` connector

This connector config lets users authenticate through [Bitbucket](https://bitbucket.org/). In addition to `id` and `type`, the `bitbucket` connector takes the following additional fields:

* clientID: a `string`. The Bitbucket OAuth consumer client ID.

* clientSecret: a `string`. The Bitbucket OAuth consumer client secret.

To begin, register an OAuth consumer with Bitbucket through your, or your teams's management page. Follow the documentation at their [developer site](https://confluence.atlassian.com/bitbucket/oauth-on-bitbucket-cloud-238027431.html). 
__NOTE:__ When configuring a consumer through Bitbucket you _must_ configure read email permissions.

To register dex as a client of your Bitbucket consumer, enter dex's redirect URL under 'Callback URL':

```
https://$DEX_HOST:$DEX_PORT/auth/$CONNECTOR_ID/callback
```

`$DEX_HOST` and `$DEX_PORT` are the host and port of your dex installation. `$CONNECTOR_ID` is the `id` field of the connector.

Here's an example of a `bitbucket` connector; the clientID and clientSecret should be replaced by values provided by Bitbucket.

```
    {
        "type": "bitbucket",
        "id": "bitbucket",
        "clientID": "$DEX_BITBUCKET_CLIENT_ID",
        "clientSecret": "$DEX_BITBUCKET_CLIENT_SECRET"
    }
```

## Setting the Configuration

To set a connectors configuration in dex, put it in some temporary file, then use the dexctl command to upload it to dex:

```
dexctl -db-url=$DEX_DB_URL set-connector-configs /tmp/dex_connectors.json
```
