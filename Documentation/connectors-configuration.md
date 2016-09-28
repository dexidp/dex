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
* emailClaim: a `string`. The name of the claim to be treated as an email claim. If empty dex will use a `email` claim.

In order to use the `oidc` connector you must register dex as an OIDC client; this mechanism is different from provider to provider. For Google, follow the instructions at their [developer site](https://developers.google.com/identity/protocols/OpenIDConnect?hl=en). Regardless of your provider, registering your client will also provide you with the client ID and secret.

When registering dex as a client, you need to provide redirect URLs to the provider. dex requires just one:

```
$ISSUER_URL/auth/$CONNECTOR_ID/callback
```

For example runnning a connector with ID `"google"` and an issuer URL of `"https://auth.example.com/foo"` the redirect would be.

```
https://auth.example.com/foo/auth/google/callback
```

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
$ISSUER_URL/auth/$CONNECTOR_ID/callback
```

For example runnning a connector with ID `"github"` and an issuer URL of `"https://auth.example.com/bar"` the redirect would be.

```
https://auth.example.com/bar/auth/github/callback
```

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
$ISSUER_URL/auth/$CONNECTOR_ID/callback
```

For example runnning a connector with ID `"bitbucket"` and an issuer URL of `"https://auth.example.com/spaz"` the redirect would be.

```
https://auth.example.com/spaz/auth/bitbucket/callback
```

Here's an example of a `bitbucket` connector; the clientID and clientSecret should be replaced by values provided by Bitbucket.

```
    {
        "type": "bitbucket",
        "id": "bitbucket",
        "clientID": "$DEX_BITBUCKET_CLIENT_ID",
        "clientSecret": "$DEX_BITBUCKET_CLIENT_SECRET"
    }
```

### `ldap` connector

The `ldap` connector allows email/password based authentication hosted by dex, backed by a LDAP directory. The connector can operate in two primary modes:

1. Binding against a specific directory using the end user's credentials.
2. Searching a directory for a entry using a service account then attempting to bind with the user's credentials.

User entries are expected to have an email attribute (configurable through "emailAttribute"), and optionally a display name attribute (configurable through "nameAttribute").

___NOTE:___ Dex currently requires user registration with the dex system, even if that user already has an account with the upstream LDAP system. Installations that use this connector are recommended to provide the "--enable-automatic-registration" flag.

In addition to `id` and `type`, the `ldap` connector takes the following additional fields:

* host: a `string`. The host and port of the LDAP server in form "host:port".
* useTLS: a `boolean`. Whether the LDAP Connector should issue a StartTLS after successfully connecting to the LDAP Server.
* useSSL: a `boolean`. Whether the LDAP Connector should expect the connection to be encrypted, typically used with ldaps port (636/tcp).
* certFile: a `string`. Optional path to x509 client certificate to present to LDAP server.
* keyFile: a `string`. Key associated with x509 client cert specified in `certFile`.
* caFile: a `string`. Filename for PEM-file containing the set of root certificate authorities that the LDAP client use when verifying the server certificates. Default: use the host's root CA set.
* baseDN: a `string`. Base DN from which Bind DN is built and searches are based.
* nameAttribute: a `string`. Entity attribute to map to display name of users. Default: `cn`
* emailAttribute: a `string`. Required. Attribute to map to Email. Default: `mail`
* searchBeforeAuth: a `boolean`. Perform search for entryDN to be used for bind.
* searchFilter: a `string`. Filter to apply to search. Variable substititions: `%u` User supplied username/e-mail address. `%b` BaseDN. Searches that return multiple entries are considered ambiguous and will return an error.
* searchGroupFilter: a `string`. A filter which should return group entry for a given user. The string is formatted the same as `searchFilter`, execpt `%u` is replaced by the fully qualified user entry. Groups are only searched if the client request the "groups" scope.
* searchScope: a `string`. Scope of the search. `base|one|sub`. Default: `one`
* searchBindDN: a `string`. DN to bind as for search operations.
* searchBindPw: a `string`. Password for bind for search operations.
* bindTemplate: a `string`. Template to build bindDN from user supplied credentials. Variable subtitutions: `%u` User supplied username/e-mail address. `%b` BaseDN. Default: `uid=%u,%b`.

### Example: Authenticating against a specific directory

To authenticate against a specific LDAP directory level, use the "bindTemplate" field. This string describes how to map a username to a LDAP entity.

```
    {
        "type": "ldap",
        "id": "ldap",
        "host": "127.0.0.1:389",
        "baseDN": "cn=users,cn=accounts,dc=auth,dc=example,dc=com",
        "bindTemplate": "uid=%u,%d"
    }
```

"bindTemplate" is a format string. `%d` is replaced by the value of "baseDN" and `%u` is replaced by the username attempting to login. In this case if a user "janedoe" attempts to authenticate, the bindTemplate will be expanded to:

```
uid=janedoe,cn=users,cn=accounts,dc=auth,dc=example,dc=com
```

The connector then attempts to bind as this entry using the password provided by the end user.

### Example: Searching a FreeIPA server with groups

The following configuration will search a FreeIPA directory using an LDAP filter.

```
    {
        "type": "ldap",
        "id": "ldap",
        "host": "127.0.0.1:389",
        "baseDN": "cn=accounts,dc=example,dc=com",

        "searchBeforeAuth": true,
        "searchFilter": "(&(objectClass=person)(uid=%u))",
        "searchGroupFilter": "(&(objectClass=ipausergroup)(member=%u))",
        "searchScope": "sub",

        "searchBindDN": "serviceAccountUser",
        "searchBindPw": "serviceAccountPassword"
    }
```

"searchFilter" is a format string expanded in a similar manner as "bindTemplate". If the user "janedoe" attempts to authenticate, the connector will run the following query using the service account credentials.

```
(&(objectClass=person)(uid=janedoe))
```

If the search finds an entry, it will attempt to use the provided password to bind as that entry. Searches that return multiple entries are considered ambiguous and will return an error.

"searchGroupFilter" is a format string similar to "searchFilter" except `%u` is replaced by the fully qualified user entry returned by "searchFilter". So if the initial search returns "uid=janedoe,cn=users,cn=accounts,dc=example,dc=com", the connector will use the search query:

```
(&(objectClass=ipausergroup)(member=uid=janedoe,cn=users,cn=accounts,dc=example,dc=com))
```

If the client requests the "groups" scope, the names of all returned entries are added to the ID Token "groups" claim.

## Setting the Configuration

To set a connectors configuration in dex, put it in some temporary file, then use the dexctl command to upload it to dex:

```
dexctl --db-url=$DEX_DB_URL set-connector-configs /tmp/dex_connectors.json
```

### `uaa` connector

This connector config lets users authenticate through the
[CloudFoundry User Account and Authentication (UAA) Server](https://github.com/cloudfoundry/uaa). In addition to `id`
and `type`, the `uaa` connector takes the following additional fields:

* clientID: a `string`. The UAA OAuth application client ID.
* clientSecret: a `string`. The UAA OAuth application client secret.
* serverURL: a `string`. The full URL to the UAA service.

To begin, register an OAuth application with UAA. To register dex as a client of your UAA application, there are two
things your OAuth application needs to have configured properly:

* Make sure dex's redirect URL _(`ISSUER_URL/auth/$CONNECTOR_ID/callback`)_ is in the application's `redirect_uri` list
* Make sure the `openid` scope is in the application's `scope` list

Regarding the `redirect_uri` list, as an example if you were running dex at `https://auth.example.com/bar`, the UAA
OAuth application's `redirect_uri` list would need to contain `https://auth.example.com/bar/auth/uaa/callback`.

Here's an example of a `uaa` connector _(The `clientID` and `clientSecret` should be replaced by values provided to UAA
and the `serverURL` should be the fully-qualified URL to your UAA server)_:

```
    {
        "type": "uaa",
        "id": "example-uaa",
        "clientID": "$UAA_OAUTH_APPLICATION_CLIENT_ID",
        "clientSecret": "$UAA_OAUTH_APPLICATION_CLIENT_SECRET",
        "serverURL": "$UAA_SERVER_URL"
    }
```

The `uaa` connector requests only the `openid` scope which allows dex the ability to query the user's identity
information.

### `facebook` connector

This connector config lets users authenticate through [Facebook](https://www.facebook.com/). In addition to `id` and `type`, the `facebook` connector takes the following additional fields:

* clientID: a `string`. The Facebook App ID.
* clientSecret: a `string`. The Facebook App Secret.

To begin, register an App in facebook and configure it according to following steps.

* Go to [https://developers.facebook.com/](https://developers.facebook.com/) and log in using your Facebook credentials.
* If you haven't created developer account follow step 2 in [https://developers.facebook.com/docs/apps/register](https://developers.facebook.com/docs/apps/register).
* Click on `My Apps` and then click `Create a New App`.
* Choose the platform you wish to use. Select `Website` if you are testing dex sample app.
* Enter the name of your new app in the window that appears and click `Create App ID`.
* Enter a `Display Name`, `Contact Email` and select an appropriate `category` from the dropdown. Click `Create App ID`.
* Click on `Dashboard` from the left menu to go to the developer Dashboard. You can find the `App ID` and `App Secret` there. Click Show to view the `App Secret`.
* Click `Settings` on the left menu and navigate to the Basic tab. Add the dex worker domain(if dex is running on localhost, you can add `localhost` as the `App Domain`) and click `Add Platform`.
* Select `Website`  as the platform for the application and enter the dex URL (if dex is running on localhost, you can add `http://localhost:5556`). Click `Save Changes`.
* On the left panel, click `Add Product` and click Get Started for a `Facebook Login` product.
* You can configure the Client OAuth Settings on the window that appears. `Client OAuth Login` should be set to `Yes`. `Web OAuth Login` should be set to `Yes`. `Valid OAuth redirect URIs` should be set to in following format.

```
$ISSUER_URL/auth/$CONNECTOR_ID/callback
```

For example runnning a connector with ID `"facebook"` and an issuer URL of `"https://auth.example.com/spaz"` the redirect would be.

```
https://auth.example.com/spaz/auth/facebook/callback
```

* Scroll down and click the Save Changes button to save the change.


Here's an example of a `facebook` connector configuration; the clientID and clientSecret should be replaced by App ID and App Secret values provided by Facebook.

```
    {
        "type": "facebook",
        "id": "facebook",
        "clientID": "$DEX_FACEBOOK_CLIENT_ID",
        "clientSecret": "$DEX_FACEBOOK_CLIENT_SECRET"
    }
```

