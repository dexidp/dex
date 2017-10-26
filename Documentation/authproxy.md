# External authentication

## Overview

The authproxy connector returns identities based on authentication which your
front-end web server performs.

The connector does not support refresh tokens or groups at this point.

## Configuration

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