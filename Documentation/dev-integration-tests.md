# Running integration tests

## Kubernetes

Kubernetes tests will only run if the `DEX_KUBECONFIG` environment variable is set.

```
$ export DEX_KUBECONFIG=~/.kube/config
$ go test -v -i ./storage/kubernetes
$ go test -v ./storage/kubernetes
```

Because third party resources creation isn't synchronized it's expected that the tests fail the first time. Fear not, and just run them again.

## Postgres

Running database tests locally require:

* A systemd based Linux distro.
* A recent version of [rkt](https://github.com/coreos/rkt) installed.

The `standup.sh` script in the SQL directory is used to run databases in containers with systemd daemonizing the process.

```
$ sudo ./storage/sql/standup.sh create postgres
Starting postgres. To view progress run

  journalctl -fu dex-postgres

Running as unit dex-postgres.service.
To run tests export the following environment variables:

  export DEX_POSTGRES_DATABASE=postgres; export DEX_POSTGRES_USER=postgres; export DEX_POSTGRES_PASSWORD=postgres; export DEX_POSTGRES_HOST=172.16.28.3:5432

```

Exporting the variables will cause the database tests to be run, rather than skipped.

```
$ # sqlite3 takes forever to compile, be sure to install test dependencies
$ go test -v -i ./storage/sql
$ go test -v ./storage/sql
```

When you're done, tear down the unit using the `standup.sh` script.

```
$ sudo ./storage/sql/standup.sh destroy postgres
```

## LDAP

To run LDAP tests locally, you require a container running OpenLDAP.

Run OpenLDAP docker image:

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
