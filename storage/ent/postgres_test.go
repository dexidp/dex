package ent

import (
	"os"
	"strconv"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/conformance"
)

const (
	PostgresEntHostEnv     = "DEX_POSTGRES_ENT_HOST"
	PostgresEntPortEnv     = "DEX_POSTGRES_ENT_PORT"
	PostgresEntDatabaseEnv = "DEX_POSTGRES_ENT_DATABASE"
	PostgresEntUserEnv     = "DEX_POSTGRES_ENT_USER"
	PostgresEntPasswordEnv = "DEX_POSTGRES_ENT_PASSWORD"
)

func postgresTestConfig(host string, port uint64) *Postgres {
	return &Postgres{
		NetworkDB: NetworkDB{
			Database: getenv(PostgresEntDatabaseEnv, "postgres"),
			User:     getenv(PostgresEntUserEnv, "postgres"),
			Password: getenv(PostgresEntPasswordEnv, "postgres"),
			Host:     host,
			Port:     uint16(port),
		},
		SSL: SSL{
			Mode: pgSSLDisable, // Postgres container doesn't support SSL.
		},
	}
}

func newPostgresStorage(host string, port uint64) storage.Storage {
	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: &logrus.TextFormatter{DisableColors: true},
		Level:     logrus.DebugLevel,
	}

	cfg := postgresTestConfig(host, port)
	s, err := cfg.Open(logger)
	if err != nil {
		panic(err)
	}
	return s
}

func TestPostgres(t *testing.T) {
	host := os.Getenv(PostgresEntHostEnv)
	if host == "" {
		t.Skipf("test environment variable %s not set, skipping", PostgresEntHostEnv)
	}

	port := uint64(5432)
	if rawPort := os.Getenv(PostgresEntPortEnv); rawPort != "" {
		var err error

		port, err = strconv.ParseUint(rawPort, 10, 32)
		require.NoError(t, err, "invalid postgres port %q: %s", rawPort, err)
	}

	newStorage := func() storage.Storage {
		return newPostgresStorage(host, port)
	}
	conformance.RunTests(t, newStorage)
	conformance.RunTransactionTests(t, newStorage)
}

func TestPostgresDSN(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *Postgres
		desiredDSN string
	}{
		{
			name: "Host port",
			cfg: &Postgres{
				NetworkDB: NetworkDB{
					Host: "localhost",
					Port: uint16(5432),
				},
			},
			desiredDSN: "connect_timeout=0 host='localhost' port=5432 sslmode='verify-full'",
		},
		{
			name: "Host with port",
			cfg: &Postgres{
				NetworkDB: NetworkDB{
					Host: "localhost:5432",
				},
			},
			desiredDSN: "connect_timeout=0 host='localhost' port=5432 sslmode='verify-full'",
		},
		{
			name: "Host ipv6 with port",
			cfg: &Postgres{
				NetworkDB: NetworkDB{
					Host: "[a:b:c:d]:5432",
				},
			},
			desiredDSN: "connect_timeout=0 host='a:b:c:d' port=5432 sslmode='verify-full'",
		},
		{
			name: "Credentials and timeout",
			cfg: &Postgres{
				NetworkDB: NetworkDB{
					Database:          "test",
					User:              "test",
					Password:          "test",
					ConnectionTimeout: 5,
				},
			},
			desiredDSN: "connect_timeout=5 user='test' password='test' dbname='test' sslmode='verify-full'",
		},
		{
			name: "SSL",
			cfg: &Postgres{
				SSL: SSL{
					Mode:     pgSSLRequire,
					CAFile:   "/ca.crt",
					KeyFile:  "/cert.crt",
					CertFile: "/cert.key",
				},
			},
			desiredDSN: "connect_timeout=0 sslmode='require' sslrootcert='/ca.crt' sslcert='/cert.key' sslkey='/cert.crt'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.desiredDSN, tt.cfg.dsn())
		})
	}
}

func TestPostgresDriver(t *testing.T) {
	host := os.Getenv(PostgresEntHostEnv)
	if host == "" {
		t.Skipf("test environment variable %s not set, skipping", PostgresEntHostEnv)
	}

	port := uint64(5432)
	if rawPort := os.Getenv(PostgresEntPortEnv); rawPort != "" {
		var err error

		port, err = strconv.ParseUint(rawPort, 10, 32)
		require.NoError(t, err, "invalid postgres port %q: %s", rawPort, err)
	}

	tests := []struct {
		name         string
		cfg          func() *Postgres
		desiredConns int
	}{
		{
			name:         "Defaults",
			cfg:          func() *Postgres { return postgresTestConfig(host, port) },
			desiredConns: 5,
		},
		{
			name: "Tune",
			cfg: func() *Postgres {
				cfg := postgresTestConfig(host, port)
				cfg.MaxOpenConns = 101
				return cfg
			},
			desiredConns: 101,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			drv, err := tt.cfg().driver()
			require.NoError(t, err)

			require.Equal(t, tt.desiredConns, drv.DB().Stats().MaxOpenConnections)
		})
	}
}
