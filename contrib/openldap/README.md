# An OpenLDAP container

## Running with rkt

First be sure to clean any existing containers and turn SELinux to Permissive (this is due to a known issue in rkt).

    sudo setenforce Permissive
    sudo rkt gc --grace-period=0s

Run the OpenLDAP container at a predefined IP, this will set some initial values.

    sudo rkt run --net=default:IP=172.16.28.25 quay.io/coreos/openldap:2.4.44

OpenLDAP will then be available on port 389. To work with the container's examples install the openldap client programs on your host.

    sudo dnf install -y openldap-clients

`ldapadd` can be used to add new entries to the directory.

    ldapadd \
      -h 172.16.28.25 \
      -D "cn=Manager,dc=example,dc=com" \
      -w "secret" \
      -f examples/example.ldif

The created entries can be searched with the `ldapsearch` command.

    ldapsearch \
      -h 172.16.28.25 \
      -D "cn=Manager,dc=example,dc=com" \
      -w "secret" \
      -b "dc=example,dc=com" \
      '(objectClass=*)'

## Customizing the created directory

The container uses environment variables defined in the `scripts/entrypoint.sh` bash file for initial configuration. Overriding these values will cause the 

    sudo rkt run \
      --set-env=LDAP_DOMAIN="dc=dex,dc=coreos,dc=com" \
      --set-env=LDAP_ROOT_CN="cn=admin" \
      --set-env=LDAP_ROOT_PW="password" \
      --net=default:IP=172.16.28.25 \
      quay.io/coreos/openldap:2.4.44

## Development

The `Makefile` can be used to build the container using Docker. This will download OpenLDAP, compile it in a container, then add the entrypoint script.

    make

General development looks like.

    vim scripts/entrypoint.sh
    make
    sudo docker run -it --rm --entrypoint=/bin/sh quay.io/coreos/openldap:2.4.44
    # poke around or run /entrypoint.sh manually

## TODO

* TLS support.
* Seed with initial data through mounted volume.
* Better `objectClass` schemas that match other LDAP deployments.
