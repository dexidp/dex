package etcd

import (
	"time"

	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/namespace"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/storage"
)

var defaultDialTimeout = 2 * time.Second

// SSL represents SSL options for etcd databases.
type SSL struct {
	ServerName string `json:"serverName" yaml:"serverName"`
	CAFile     string `json:"caFile" yaml:"caFile"`
	KeyFile    string `json:"keyFile" yaml:"keyFile"`
	CertFile   string `json:"certFile" yaml:"certFile"`
}

// Etcd options for connecting to etcd databases.
// If you are using a shared etcd cluster for storage, it might be useful to
// configure an etcd namespace either via Namespace field or using `etcd grpc-proxy
// --namespace=<prefix>`
type Etcd struct {
	Endpoints []string `json:"endpoints" yaml:"endpoints"`
	Namespace string   `json:"namespace" yaml:"namespace"`
	Username  string   `json:"username" yaml:"username"`
	Password  string   `json:"password" yaml:"password"`
	SSL       SSL      `json:"ssl" yaml:"ssl"`
}

// Open creates a new storage implementation backed by Etcd
func (p *Etcd) Open(logger log.Logger) (storage.Storage, error) {
	return p.open(logger)
}

func (p *Etcd) open(logger log.Logger) (*conn, error) {
	cfg := clientv3.Config{
		Endpoints:   p.Endpoints,
		DialTimeout: defaultDialTimeout,
		Username:    p.Username,
		Password:    p.Password,
	}

	var cfgtls *transport.TLSInfo
	tlsinfo := transport.TLSInfo{}
	if p.SSL.CertFile != "" {
		tlsinfo.CertFile = p.SSL.CertFile
		cfgtls = &tlsinfo
	}

	if p.SSL.KeyFile != "" {
		tlsinfo.KeyFile = p.SSL.KeyFile
		cfgtls = &tlsinfo
	}

	if p.SSL.CAFile != "" {
		tlsinfo.TrustedCAFile = p.SSL.CAFile
		cfgtls = &tlsinfo
	}

	if p.SSL.ServerName != "" {
		tlsinfo.ServerName = p.SSL.ServerName
		cfgtls = &tlsinfo
	}

	if cfgtls != nil {
		clientTLS, err := cfgtls.ClientConfig()
		if err != nil {
			return nil, err
		}
		cfg.TLS = clientTLS
	}

	db, err := clientv3.New(cfg)
	if err != nil {
		return nil, err
	}
	if len(p.Namespace) > 0 {
		db.KV = namespace.NewKV(db.KV, p.Namespace)
	}
	c := &conn{
		db:     db,
		logger: logger,
	}
	return c, nil
}
