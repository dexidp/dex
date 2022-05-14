module github.com/dexidp/dex

go 1.17

require (
	entgo.io/ent v0.10.1
	github.com/AppsFlyer/go-sundheit v0.5.0
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/beevik/etree v1.1.0
	github.com/coreos/go-oidc/v3 v3.1.0
	github.com/dexidp/dex/api/v2 v2.1.0
	github.com/felixge/httpsnoop v1.0.2
	github.com/ghodss/yaml v1.0.0
	github.com/go-ldap/ldap/v3 v3.4.2
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/kylelemons/godebug v1.1.0
	github.com/lib/pq v1.10.4
	github.com/mattermost/xml-roundtrip-validator v0.1.0
	github.com/mattn/go-sqlite3 v1.14.11
	github.com/oklog/run v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.1
	github.com/russellhaering/goxmldsig v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.4.0
	github.com/stretchr/testify v1.7.1
	go.etcd.io/etcd/client/pkg/v3 v3.5.2
	go.etcd.io/etcd/client/v3 v3.5.2
	golang.org/x/crypto v0.0.0-20220208050332-20e1d8d225ab
	golang.org/x/net v0.0.0-20220325170049-de3da57026de
	golang.org/x/oauth2 v0.0.0-20220309155454-6242fa91716a
	google.golang.org/api v0.74.0
	google.golang.org/grpc v1.45.0
	google.golang.org/protobuf v1.28.0
	gopkg.in/square/go-jose.v2 v2.6.0
)

replace github.com/dexidp/dex/api/v2 => ./api/v2
