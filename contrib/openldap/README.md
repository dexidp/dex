# An OpenLDAP container

## Running with rkt



	sudo rkt run --net=default:IP=172.16.28.25 quay.io/coreos/openldap:2.4.44

OpenLDAP will then be available on port 389.

	$ telnet 172.16.28.25 389
	Trying 172.16.28.25...
	Connected to 172.16.28.25.
	Escape character is '^]'.
	^]
	telnet> quit
	Connection closed.

To inspect.

	sudo dnf install -y openldap-clients


## TLS
