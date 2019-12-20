// +build go1.11

package sql

import (
	"os"
	"strconv"
	"testing"
)

func TestPostgresTunables(t *testing.T) {
	host := os.Getenv(testPostgresEnv)
	if host == "" {
		t.Skipf("test environment variable %q not set, skipping", testPostgresEnv)
	}

	port := uint64(5432)
	if rawPort := os.Getenv("DEX_POSTGRES_PORT"); rawPort != "" {
		var err error

		port, err = strconv.ParseUint(rawPort, 10, 32)
		if err != nil {
			t.Fatalf("invalid postgres port %q: %s", rawPort, err)
		}
	}

	baseCfg := &Postgres{
		NetworkDB: NetworkDB{
			Database: getenv("DEX_POSTGRES_DATABASE", "postgres"),
			User:     getenv("DEX_POSTGRES_USER", "postgres"),
			Password: getenv("DEX_POSTGRES_PASSWORD", "postgres"),
			Host:     host,
			Port:     uint16(port),
		},
		SSL: SSL{
			Mode: pgSSLDisable, // Postgres container doesn't support SSL.
		}}

	t.Run("with nothing set, uses defaults", func(t *testing.T) {
		cfg := *baseCfg
		c, err := cfg.open(logger)
		if err != nil {
			t.Fatalf("error opening connector: %s", err.Error())
		}
		defer c.db.Close()
		if m := c.db.Stats().MaxOpenConnections; m != 5 {
			t.Errorf("expected MaxOpenConnections to have its default (5), got %d", m)
		}
	})

	t.Run("with something set, uses that", func(t *testing.T) {
		cfg := *baseCfg
		cfg.MaxOpenConns = 101
		c, err := cfg.open(logger)
		if err != nil {
			t.Fatalf("error opening connector: %s", err.Error())
		}
		defer c.db.Close()
		if m := c.db.Stats().MaxOpenConnections; m != 101 {
			t.Errorf("expected MaxOpenConnections to be set to 101, got %d", m)
		}
	})
}
