module github.com/dexidp/dex

go 1.15

require (
	github.com/Microsoft/hcsshim v0.8.14 // indirect
	github.com/beevik/etree v1.1.0
	github.com/coreos/go-oidc/v3 v3.0.0
	github.com/dexidp/dex/api/v2 v2.0.0
	github.com/felixge/httpsnoop v1.0.1
	github.com/ghodss/yaml v1.0.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/protobuf v1.4.3
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/kylelemons/godebug v1.1.0
	github.com/lib/pq v1.9.0
	github.com/mattermost/xml-roundtrip-validator v0.0.0-20201219040909-8fd2afad43d1
	github.com/mattn/go-sqlite3 v1.14.6
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.4.0
	github.com/russellhaering/goxmldsig v1.1.0
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.7.0
	github.com/testcontainers/testcontainers-go v0.0.9
	go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b
	golang.org/x/oauth2 v0.0.0-20201208152858-08078c50e5b5
	google.golang.org/api v0.37.0
	google.golang.org/grpc v1.34.0
	google.golang.org/grpc/examples v0.0.0-20210122012134-2c42474aca0c // indirect
	gopkg.in/asn1-ber.v1 v1.0.0-20181015200546-f715ec2f112d // indirect
	gopkg.in/ldap.v2 v2.5.1
	gopkg.in/square/go-jose.v2 v2.5.1
	sigs.k8s.io/testing_frameworks v0.1.2
)

replace github.com/dexidp/dex/api/v2 => ./api/v2
