module github.com/dexidp/dex

go 1.15

require (
	github.com/Microsoft/hcsshim v0.8.14 // indirect
	github.com/beevik/etree v1.1.0
	github.com/coreos/go-oidc/v3 v3.0.0
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/dexidp/dex/api/v2 v2.0.0
	github.com/felixge/httpsnoop v1.0.1
	github.com/ghodss/yaml v1.0.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6 // indirect
	github.com/golang/protobuf v1.3.2
	github.com/google/uuid v1.1.1 // indirect
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/kylelemons/godebug v1.1.0
	github.com/lib/pq v1.9.0
	github.com/mattermost/xml-roundtrip-validator v0.0.0-20201204154048-1a8688af4cf1
	github.com/mattn/go-sqlite3 v1.11.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.4.0
	github.com/russellhaering/goxmldsig v1.1.0
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.6.1
	github.com/testcontainers/testcontainers-go v0.0.9
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738
	go.uber.org/atomic v1.4.0 // indirect
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4
	golang.org/x/net v0.0.0-20200505041828-1ed23360d12c
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	google.golang.org/api v0.15.0
	google.golang.org/appengine v1.6.1 // indirect
	google.golang.org/grpc v1.26.0
	gopkg.in/asn1-ber.v1 v1.0.0-20181015200546-f715ec2f112d // indirect
	gopkg.in/ldap.v2 v2.5.1
	gopkg.in/square/go-jose.v2 v2.5.1
	sigs.k8s.io/testing_frameworks v0.1.2
)

replace github.com/dexidp/dex/api/v2 => ./api/v2
