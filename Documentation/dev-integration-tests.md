# Running integration tests

## Postgres

Running database tests locally requires:

* Docker

To run the database integration tests:

- start a postgres container:

  `docker run --name dex-postgres -e POSTGRES_USER=postgres -e POSTGRES_DB=dex -p 5432:5432 -d postgres:11`
- export the required environment variables:

  `export DEX_POSTGRES_DATABASE=dex DEX_POSTGRES_USER=postgres DEX_POSTGRES_PASSWORD=postgres DEX_POSTGRES_HOST=127.0.0.1:5432`

- run the storage/sql tests:

  ```
  $ # sqlite3 takes forever to compile, be sure to install test dependencies
  $ go test -v -i ./storage/sql
  $ go test -v ./storage/sql
  ```

- clean up the postgres container: `docker rm -f dex-postgres`

## Etcd

These tests can also be executed using docker:

- start the container (where `NODE1` is set to the host IP address):

  ```
  $ export NODE1=0.0.0.0
  $ docker run --name dex-etcd -p 2379:2379 -p 2380:2380 gcr.io/etcd-development/etcd:v3.3.10 \
    /usr/local/bin/etcd --name node1 \
    --initial-advertise-peer-urls http://${NODE1}:2380 --listen-peer-urls http://${NODE1}:2380 \
    --advertise-client-urls http://${NODE1}:2379 --listen-client-urls http://${NODE1}:2379 \
    --initial-cluster node1=http://${NODE1}:2380
  ```

- run the tests, passing the correct endpoint for this etcd instance in `DEX_ETCD_ENDPOINTS`:

  `DEX_ETCD_ENDPOINTS=http://localhost:2379 go test -v ./storage/etcd`
- clean up the etcd container: `docker rm -f dex-etcd`

## LDAP

The LDAP integration tests require [OpenLDAP][openldap] installed on the host machine. To run them, use `go test`:

```
export DEX_LDAP_TESTS=1
go test -v ./connector/ldap/
```

To quickly stand up a LDAP server for development, see the LDAP [_"Getting started"_][ldap-getting-started] example. This also requires OpenLDAP installed on the host.

To stand up a containerized LDAP server run the OpenLDAP docker image:

```
$ sudo docker run --hostname ldap.example.org --name openldap-container --detach osixia/openldap:1.1.6
```

By default TLS is enabled and a certificate is created with the container hostname, which in this case is "ldap.example.org". It will create an empty LDAP for the company Example Inc. and the domain example.org. By default the admin has the password admin.

Add new users and groups (sample .ldif file included at the end):

```
$ sudo docker exec openldap-container ldapadd -x -D "cn=admin,dc=example,dc=org" -w admin -f <path to .ldif> -h ldap.example.org -ZZ
```

Verify that the added entries are in your directory with ldapsearch :

```
$ sudo docker exec openldap-container ldapsearch -x -h localhost -b dc=example,dc=org -D "cn=admin,dc=example,dc=org" -w admin
```
The .ldif file should contain seed data. Example file contents:

```
dn: cn=Test1,dc=example,dc=org
objectClass: organizationalRole
cn: Test1

dn: cn=Test2,dc=example,dc=org
objectClass: organizationalRole
cn: Test2

dn: ou=groups,dc=example,dc=org
ou: groups
objectClass: top
objectClass: organizationalUnit

dn: cn=tstgrp,ou=groups,dc=example,dc=org
objectClass: top
objectClass: groupOfNames
member: cn=Test1,dc=example,dc=org
cn: tstgrp
```

## SAML

### Okta

The Okta identity provider supports free accounts for developers to test their implementation against. This document describes configuring an Okta application to test dex's SAML connector.

First, [sign up for a developer account][okta-sign-up]. Then, to create a SAML application:

* Go to the admin screen.
* Click "Add application"
* Click "Create New App"
* Choose "SAML 2.0" and press "Create"
* Configure SAML
  * Enter `http://127.0.0.1:5556/dex/callback` for "Single sign on URL"
  * Enter `http://127.0.0.1:5556/dex/callback` for "Audience URI (SP Entity ID)"
  * Under "ATTRIBUTE STATEMENTS (OPTIONAL)" add an "email" and "name" attribute. The values should be something like `user:email` and `user:firstName`, respectively.
  * Under "GROUP ATTRIBUTE STATEMENTS (OPTIONAL)" add a "groups" attribute. Use the "Regexp" filter `.*`.

After the application's created, assign yourself to the app.

* "Applications" > "Applications"
* Click on your application then under the "People" tab press the "Assign to People" button and add yourself.

At the app, go to the "Sign On" tab and then click "View Setup Instructions". Use those values to fill out the following connector in `examples/config-dev.yaml`.

```yaml
connectors:
- type: saml
  id: saml
  name: Okta
  config:
    ssoURL: ( "Identity Provider Single Sign-On URL" )
    caData: ( base64'd value of "X.509 Certificate" )
    redirectURI: http://127.0.0.1:5556/dex/callback
    usernameAttr: name
    emailAttr: email
    groupsAttr: groups
```

Start both dex and the example app, and try logging in (requires not requesting a refresh token).

[okta-sign-up]: https://www.okta.com/developer/signup/
[openldap]: https://www.openldap.org/
[ldap-getting-started]: ldap-connector.md#getting-started
