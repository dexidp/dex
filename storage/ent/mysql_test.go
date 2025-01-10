package ent

import (
	"io"
	"log/slog"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/conformance"
)

const (
	MySQLEntHostEnv     = "DEX_MYSQL_ENT_HOST"
	MySQLEntPortEnv     = "DEX_MYSQL_ENT_PORT"
	MySQLEntDatabaseEnv = "DEX_MYSQL_ENT_DATABASE"
	MySQLEntUserEnv     = "DEX_MYSQL_ENT_USER"
	MySQLEntPasswordEnv = "DEX_MYSQL_ENT_PASSWORD"
)

func mysqlTestConfig(host string, port uint64) *MySQL {
	return &MySQL{
		NetworkDB: NetworkDB{
			Database: getenv(MySQLEntDatabaseEnv, "mysql"),
			User:     getenv(MySQLEntUserEnv, "mysql"),
			Password: getenv(MySQLEntPasswordEnv, "mysql"),
			Host:     host,
			Port:     uint16(port),
		},
		SSL: SSL{
			// This was originally mysqlSSLSkipVerify. It lead to handshake errors.
			// See https://github.com/go-sql-driver/mysql/issues/1635 for more details.
			Mode: mysqlSSLFalse,
		},
		params: map[string]string{
			"innodb_lock_wait_timeout": "1",
		},
	}
}

func newMySQLStorage(host string, port uint64) storage.Storage {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))

	cfg := mysqlTestConfig(host, port)
	s, err := cfg.Open(logger)
	if err != nil {
		panic(err)
	}
	return s
}

func TestMySQL(t *testing.T) {
	host := os.Getenv(MySQLEntHostEnv)
	if host == "" {
		t.Skipf("test environment variable %s not set, skipping", MySQLEntHostEnv)
	}

	port := uint64(3306)
	if rawPort := os.Getenv(MySQLEntPortEnv); rawPort != "" {
		var err error

		port, err = strconv.ParseUint(rawPort, 10, 32)
		require.NoError(t, err, "invalid mysql port %q: %s", rawPort, err)
	}

	newStorage := func() storage.Storage {
		return newMySQLStorage(host, port)
	}
	conformance.RunTests(t, newStorage)
	conformance.RunTransactionTests(t, newStorage)
}

func TestMySQLDSN(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *MySQL
		desiredDSN string
	}{
		{
			name: "Host port",
			cfg: &MySQL{
				NetworkDB: NetworkDB{
					Host: "localhost",
					Port: uint16(3306),
				},
			},
			desiredDSN: "tcp(localhost:3306)/?checkConnLiveness=false&parseTime=true&tls=false&maxAllowedPacket=0",
		},
		{
			name: "Host with port",
			cfg: &MySQL{
				NetworkDB: NetworkDB{
					Host: "localhost:3306",
				},
			},
			desiredDSN: "tcp(localhost:3306)/?checkConnLiveness=false&parseTime=true&tls=false&maxAllowedPacket=0",
		},
		{
			name: "Host ipv6 with port",
			cfg: &MySQL{
				NetworkDB: NetworkDB{
					Host: "[a:b:c:d]:3306",
				},
			},
			desiredDSN: "tcp([a:b:c:d]:3306)/?checkConnLiveness=false&parseTime=true&tls=false&maxAllowedPacket=0",
		},
		{
			name: "Credentials and timeout",
			cfg: &MySQL{
				NetworkDB: NetworkDB{
					Database:          "test",
					User:              "test",
					Password:          "test",
					ConnectionTimeout: 5,
				},
			},
			desiredDSN: "test:test@/test?checkConnLiveness=false&parseTime=true&timeout=5s&tls=false&maxAllowedPacket=0",
		},
		{
			name: "SSL",
			cfg: &MySQL{
				SSL: SSL{
					CAFile:   "/ca.crt",
					KeyFile:  "/cert.crt",
					CertFile: "/cert.key",
				},
			},
			desiredDSN: "/?checkConnLiveness=false&parseTime=true&tls=false&maxAllowedPacket=0",
		},
		{
			name: "With Params",
			cfg: &MySQL{
				params: map[string]string{
					"innodb_lock_wait_timeout": "1",
				},
			},
			desiredDSN: "/?checkConnLiveness=false&parseTime=true&tls=false&maxAllowedPacket=0&innodb_lock_wait_timeout=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.desiredDSN, tt.cfg.dsn(mysqlSSLFalse))
		})
	}
}
