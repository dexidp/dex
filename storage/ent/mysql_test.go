package ent

import (
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

	MySQL8EntHostEnv     = "DEX_MYSQL8_ENT_HOST"
	MySQL8EntPortEnv     = "DEX_MYSQL8_ENT_PORT"
	MySQL8EntDatabaseEnv = "DEX_MYSQL8_ENT_DATABASE"
	MySQL8EntUserEnv     = "DEX_MYSQL8_ENT_USER"
	MySQL8EntPasswordEnv = "DEX_MYSQL8_ENT_PASSWORD"
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

func mysql8TestConfig(host string, port uint64) *MySQL {
	return &MySQL{
		NetworkDB: NetworkDB{
			Database: getenv(MySQL8EntDatabaseEnv, "mysql"),
			User:     getenv(MySQL8EntUserEnv, "mysql"),
			Password: getenv(MySQL8EntPasswordEnv, "mysql"),
			Host:     host,
			Port:     uint16(port),
		},
		SSL: SSL{
			Mode: mysqlSSLFalse,
		},
		params: map[string]string{
			"innodb_lock_wait_timeout": "1",
		},
	}
}

func newMySQL8Storage(t *testing.T, host string, port uint64) storage.Storage {
	logger := slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := mysql8TestConfig(host, port)
	s, err := cfg.Open(logger)
	if err != nil {
		panic(err)
	}
	return s
}

func newMySQLStorage(t *testing.T, host string, port uint64) storage.Storage {
	logger := slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug}))

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

	newStorage := func(t *testing.T) storage.Storage {
		return newMySQLStorage(t, host, port)
	}
	conformance.RunTests(t, newStorage)
	conformance.RunTransactionTests(t, newStorage)

	// TODO(nabokihms): ent MySQL does not retry on deadlocks (Error 1213, SQLSTATE 40001:
	// Deadlock found when trying to get lock; try restarting transaction).
	// Under high contention most updates fail.
	// conformance.RunConcurrencyTests(t, newStorage)
}

func TestMySQL8(t *testing.T) {
	host := os.Getenv(MySQL8EntHostEnv)
	if host == "" {
		t.Skipf("test environment variable %s not set, skipping", MySQL8EntHostEnv)
	}

	port := uint64(3306)
	if rawPort := os.Getenv(MySQL8EntPortEnv); rawPort != "" {
		var err error

		port, err = strconv.ParseUint(rawPort, 10, 32)
		require.NoError(t, err, "invalid mysql port %q: %s", rawPort, err)
	}

	newStorage := func(t *testing.T) storage.Storage {
		return newMySQL8Storage(t, host, port)
	}
	conformance.RunTests(t, newStorage)
	conformance.RunTransactionTests(t, newStorage)

	// TODO(nabokihms): ent MySQL 8 does not retry on deadlocks (Error 1213, SQLSTATE 40001:
	// Deadlock found when trying to get lock; try restarting transaction).
	// Under high contention most updates fail.
	// conformance.RunConcurrencyTests(t, newStorage)
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
