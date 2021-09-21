package ent

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	entSQL "entgo.io/ent/dialect/sql"
	"github.com/go-sql-driver/mysql"

	// Register postgres driver.
	_ "github.com/lib/pq"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/ent/client"
	"github.com/dexidp/dex/storage/ent/db"
)

const (
	// MySQL SSL modes
	mysqlSSLTrue       = "true"
	mysqlSSLFalse      = "false"
	mysqlSSLSkipVerify = "skip-verify"
	mysqlSSLCustom     = "custom"
)

// MySQL options for creating an SQL db.
type MySQL struct {
	NetworkDB

	SSL SSL `json:"ssl"`

	params map[string]string
}

// Open always returns a new in sqlite3 storage.
func (m *MySQL) Open(logger log.Logger) (storage.Storage, error) {
	logger.Debug("experimental ent-based storage driver is enabled")
	drv, err := m.driver()
	if err != nil {
		return nil, err
	}

	databaseClient := client.NewDatabase(
		client.WithClient(db.NewClient(db.Driver(drv))),
		client.WithHasher(sha256.New),
		// Set tx isolation leve for each transaction as dex does for postgres
		client.WithTxIsolationLevel(sql.LevelSerializable),
	)

	if err := databaseClient.Schema().Create(context.TODO()); err != nil {
		return nil, err
	}

	return databaseClient, nil
}

func (m *MySQL) driver() (*entSQL.Driver, error) {
	var tlsConfig string

	switch {
	case m.SSL.CAFile != "" || m.SSL.CertFile != "" || m.SSL.KeyFile != "":
		if err := m.makeTLSConfig(); err != nil {
			return nil, fmt.Errorf("failed to make TLS config: %v", err)
		}
		tlsConfig = mysqlSSLCustom
	case m.SSL.Mode == "":
		tlsConfig = mysqlSSLTrue
	default:
		tlsConfig = m.SSL.Mode
	}

	drv, err := entSQL.Open("mysql", m.dsn(tlsConfig))
	if err != nil {
		return nil, err
	}

	if m.MaxIdleConns == 0 {
		/* Override default behaviour to fix https://github.com/dexidp/dex/issues/1608 */
		drv.DB().SetMaxIdleConns(0)
	} else {
		drv.DB().SetMaxIdleConns(m.MaxIdleConns)
	}

	return drv, nil
}

func (m *MySQL) dsn(tlsConfig string) string {
	cfg := mysql.Config{
		User:                 m.User,
		Passwd:               m.Password,
		DBName:               m.Database,
		AllowNativePasswords: true,

		Timeout: time.Second * time.Duration(m.ConnectionTimeout),

		TLSConfig: tlsConfig,

		ParseTime: true,
		Params:    make(map[string]string),
	}

	if m.Host != "" {
		if m.Host[0] != '/' {
			cfg.Net = "tcp"
			cfg.Addr = m.Host

			if m.Port != 0 {
				cfg.Addr = net.JoinHostPort(m.Host, strconv.Itoa(int(m.Port)))
			}
		} else {
			cfg.Net = "unix"
			cfg.Addr = m.Host
		}
	}

	for k, v := range m.params {
		cfg.Params[k] = v
	}

	return cfg.FormatDSN()
}

func (m *MySQL) makeTLSConfig() error {
	cfg := &tls.Config{}

	if m.SSL.CAFile != "" {
		rootCertPool := x509.NewCertPool()

		pem, err := os.ReadFile(m.SSL.CAFile)
		if err != nil {
			return err
		}

		if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
			return fmt.Errorf("failed to append PEM")
		}
		cfg.RootCAs = rootCertPool
	}

	if m.SSL.CertFile != "" && m.SSL.KeyFile != "" {
		clientCert := make([]tls.Certificate, 0, 1)
		certs, err := tls.LoadX509KeyPair(m.SSL.CertFile, m.SSL.KeyFile)
		if err != nil {
			return err
		}
		clientCert = append(clientCert, certs)
		cfg.Certificates = clientCert
	}

	mysql.RegisterTLSConfig(mysqlSSLCustom, cfg)
	return nil
}
