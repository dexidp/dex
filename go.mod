module github.com/dexidp/dex

go 1.16

require (
	entgo.io/ent v0.8.0
	github.com/AppsFlyer/go-sundheit v0.4.0
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/beevik/etree v1.1.0
	github.com/coreos/go-oidc/v3 v3.0.0
	github.com/dexidp/dex/api/v2 v2.0.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-ldap/ldap/v3 v3.3.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/kylelemons/godebug v1.1.0
	github.com/lib/pq v1.10.2
	github.com/mattermost/xml-roundtrip-validator v0.1.0
	github.com/mattn/go-sqlite3 v1.14.7
	github.com/oklog/run v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/russellhaering/goxmldsig v1.1.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3
	github.com/stretchr/testify v1.7.0
	go.etcd.io/etcd/client/pkg/v3 v3.5.0
	go.etcd.io/etcd/client/v3 v3.5.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.21.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.21.0
	go.opentelemetry.io/contrib/instrumentation/runtime v0.21.0
	go.opentelemetry.io/otel v1.0.0-RC1
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.21.0
	go.opentelemetry.io/otel/sdk v1.0.0-RC1
	go.opentelemetry.io/otel/sdk/metric v0.21.0
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/net v0.0.0-20210503060351-7fd8e65b6420
	golang.org/x/oauth2 v0.0.0-20210615190721-d04028783cf1
	google.golang.org/api v0.49.0
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/square/go-jose.v2 v2.6.0
)

replace github.com/dexidp/dex/api/v2 => ./api/v2
