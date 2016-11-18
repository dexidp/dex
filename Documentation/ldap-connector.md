# Authentication through LDAP

## Overview

The LDAP connector allows email/password based authentication, backed by a LDAP directory.

The connector executes two primary queries:

1. Finding the user based on the end user's credentials.
2. Searching for groups using the user entry.

## Configuration

User entries are expected to have an email attribute (configurable through `emailAttr`), and a display name attribute (configurable through `nameAttr`). `*Attr` attributes could be set to "DN" in situations where it is needed but not available elsewhere, and if "DN" attribute does not exist in the record.

The following is an example config file that can be used by the LDAP connector to authenticate a user.

```yaml

connectors:
- type: ldap
  id: ldap
  config:
    # Host and optional port of the LDAP server in the form "host:port".
    # If the port is not supplied, it will be guessed based on the TLS config.
    host: ldap.example.com:636
    # Following field is required if the LDAP host is not using TLS (port 389).
    # insecureNoSSL: true
    # Path to a trusted root certificate file. Default: use the host's root CA.
    rootCA: /etc/dex/ldap.ca
    # The DN and password for an application service account. The connector uses
    # these credentials to search for users and groups. Not required if the LDAP
    # server provides access for anonymous auth.
    bindDN: uid=seviceaccount,cn=users,dc=example,dc=com
    bindPW: password
    # User entry search configuration.
    userSearch:
      # BaseDN to start the search from. It will translate to the query
      # "(&(objectClass=person)(uid=<username>))".
      baseDN: cn=users,dc=example,dc=com
      # Optional filter to apply when searching the directory.
      filter: "(objectClass=person)"
      # username attribute used for comparing user entries. This will be translated
      # and combined with the other filter as "(<attr>=<username>)".
      username: uid
      # The following three fields are direct mappings of attributes on the user entry.
      # String representation of the user.
      idAttr: uid
      # Required. Attribute to map to Email.
      emailAttr: mail
      # Maps to display name of users. No default value.
      nameAttr: name
    # Group search configuration.
    groupSearch:
      # BaseDN to start the search from. It will translate to the query
      # "(&(objectClass=group)(member=<user uid>))".
      baseDN: cn=groups,dc=freeipa,dc=example,dc=com
      # Optional filter to apply when searching the directory.
      filter: "(objectClass=group)"
      # Following two fields are used to match a user to a group. It adds an additional
      # requirement to the filter that an attribute in the group must match the user's
      # attribute value.
      userAttr: uid
      groupAttr: member
      # Represents group name.
      nameAttr: name
```

The LDAP connector first initializes a connection to the LDAP directory using the `bindDN` and `bindPW`. It then tries to search for the given `username` and bind as that user to verify their password.
Searches that return multiple entries are considered ambiguous and will return an error.

## Example: Searching a FreeIPA server with groups

The following configuration will allow the LDAP connector to search a FreeIPA directory using an LDAP filter.

```yaml

connectors:
- type: ldap
  id: ldap
  config:
    # host and port of the LDAP server in form "host:port".
    host: freeipa.example.com:636
    # freeIPA server's CA
    rootCA: ca.crt
    userSearch:
      # Would translate to the query "(&(objectClass=person)(uid=<username>))".
      baseDN: cn=users,dc=freeipa,dc=example,dc=com
      filter: "(objectClass=posixAccount)"
      username: uid
      idAttr: uid
      # Required. Attribute to map to Email.
      emailAttr: mail
      # Entity attribute to map to display name of users.
    groupSearch:
      # Would translate to the query "(&(objectClass=group)(member=<user uid>))".
      baseDN: cn=groups,dc=freeipa,dc=example,dc=com
      filter: "(objectClass=group)"
      userAttr: uid
      groupAttr: member
      nameAttr: name
```

If the search finds an entry, it will attempt to use the provided password to bind as that user entry.
