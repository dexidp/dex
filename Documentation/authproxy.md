# Authenticating proxy

NOTE: This connector is experimental and may change in the future.

## Overview

The `authproxy` connector returns identities based on authentication which your
front-end web server performs. Dex consumes the `X-Remote-User` header set by
the proxy, which is then used as the user's email address.

__The proxy MUST remove any `X-Remote-*` headers set by the client, for any URL
path, before the request is forwarded to dex.__

The connector does not support refresh tokens or groups.

## Configuration

The `authproxy` connector is used by proxies to implement login strategies not
supported by dex. For example, a proxy could handle a different OAuth2 strategy
such as Slack. The connector takes no configuration other than a `name` and `id`:

```yaml
connectors:
# Slack login implemented by an authenticating proxy, not by dex.
- type: authproxy
  id: slack
  name: Slack 
```

The proxy only needs to authenticate the user when they attempt to visit the
callback URL path:

```
( dex issuer URL )/callback/( connector id )?( url query )
```

For example, if dex is running at `https://auth.example.com/dex` and the connector
ID is `slack`, the callback URL would look like:

```
https://auth.example.com/dex/callback/slack?state=xdg3z6quhrhwaueo5iysvliqf
``` 

The proxy should login the user then return them to the exact URL (inlucing the
query), setting `X-Remote-User` to the user's email before proxying the request
to dex.

## Configuration example - Apache 2

The following is an example config file that can be used by the external
connector to authenticate a user.

```yaml
connectors:
- type: authproxy
  id: myBasicAuth
  name: HTTP Basic Auth
```

The authproxy connector assumes that you configured your front-end web server
such that it performs authentication for the `/dex/callback/myBasicAuth`
location and provides the result in the X-Remote-User HTTP header. The following
configuration will work for Apache 2.4.10+:

```
<Location /dex/>
    ProxyPass "http://localhost:5556/dex/"
    ProxyPassReverse "http://localhost:5556/dex/"

    # Strip the X-Remote-User header from all requests except for the ones
    # where we override it.
    RequestHeader unset X-Remote-User
</Location>

<Location /dex/callback/myBasicAuth>
    AuthType Basic
    AuthName "db.debian.org webPassword"
    AuthBasicProvider file
    AuthUserFile "/etc/apache2/debian-web-pw.htpasswd"
    Require valid-user

    # Defense in depth: clear the Authorization header so that
    # Debian Web Passwords never even reach dex.
    RequestHeader unset Authorization

    # Requires Apache 2.4.10+
    RequestHeader set X-Remote-User expr=%{REMOTE_USER}@debian.org

    ProxyPass "http://localhost:5556/dex/callback/myBasicAuth"
    ProxyPassReverse "http://localhost:5556/dex/callback/myBasicAuth"
</Location>
```

## Full Apache2 setup

After installing your Linux distribution’s Apache2 package, place the following
virtual host configuration in e.g. `/etc/apache2/sites-available/sso.conf`:

```
<VirtualHost sso.example.net>
    ServerName sso.example.net

    ServerAdmin webmaster@localhost
    DocumentRoot /var/www/html

    ErrorLog ${APACHE_LOG_DIR}/error.log
    CustomLog ${APACHE_LOG_DIR}/access.log combined

    <Location /dex/>
        ProxyPass "http://localhost:5556/dex/"
        ProxyPassReverse "http://localhost:5556/dex/"

        # Strip the X-Remote-User header from all requests except for the ones
        # where we override it.
        RequestHeader unset X-Remote-User
    </Location>

    <Location /dex/callback/myBasicAuth>
        AuthType Basic
        AuthName "db.debian.org webPassword"
        AuthBasicProvider file
        AuthUserFile "/etc/apache2/debian-web-pw.htpasswd"
        Require valid-user

        # Defense in depth: clear the Authorization header so that
        # Debian Web Passwords never even reach dex.
        RequestHeader unset Authorization

        # Requires Apache 2.4.10+
        RequestHeader set X-Remote-User expr=%{REMOTE_USER}@debian.org

        ProxyPass "http://localhost:5556/dex/callback/myBasicAuth"
        ProxyPassReverse "http://localhost:5556/dex/callback/myBasicAuth"
    </Location>
</VirtualHost>
```

Then, enable it using `a2ensite sso.conf`, followed by a restart of Apache2.
