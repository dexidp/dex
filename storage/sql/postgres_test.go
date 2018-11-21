// +build go1.11

package sql

import (
	"os"
	"testing"
)

func TestPostgresTunables(t *testing.T) {
	host := os.Getenv(testPostgresEnv)
	if host == "" {
		t.Skipf("test environment variable %q not set, skipping", testPostgresEnv)
	}
	baseCfg := &Postgres{
		Database: getenv("DEX_POSTGRES_DATABASE", "postgres"),
		User:     getenv("DEX_POSTGRES_USER", "postgres"),
		Password: getenv("DEX_POSTGRES_PASSWORD", "postgres"),
		Host:     host,
		SSL: PostgresSSL{
			Mode: sslDisable, // Postgres container doesn't support SSL.
		}}

	t.Run("with nothing set, uses defaults", func(t *testing.T) {
		cfg := *baseCfg
		c, err := cfg.open(logger, cfg.createDataSourceName())
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
		c, err := cfg.open(logger, cfg.createDataSourceName())
		if err != nil {
			t.Fatalf("error opening connector: %s", err.Error())
		}
		defer c.db.Close()
		if m := c.db.Stats().MaxOpenConnections; m != 101 {
			t.Errorf("expected MaxOpenConnections to be set to 101, got %d", m)
		}
	})
}
