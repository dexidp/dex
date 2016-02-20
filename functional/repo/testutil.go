package repo

import (
	"os"
	"testing"

	"github.com/go-gorp/gorp"

	"github.com/coreos/dex/db"
)

func connect(t *testing.T) *gorp.DbMap {
	dsn := os.Getenv("DEX_TEST_DSN")
	if dsn == "" {
		t.Fatal("DEX_TEST_DSN environment variable not set")
	}
	c, err := db.NewConnection(db.Config{DSN: dsn})
	if err != nil {
		t.Fatalf("Unable to connect to database: %v", err)
	}
	if err = c.DropTablesIfExists(); err != nil {
		t.Fatalf("Unable to drop database tables: %v", err)
	}

	if err = db.DropMigrationsTable(c); err != nil {
		t.Fatalf("Unable to drop migration table: %v", err)
	}

	n, err := db.MigrateToLatest(c)
	if err != nil {
		t.Fatalf("Unable to migrate: %v", err)
	}
	if n == 0 {
		t.Fatalf("No migrations performed")
	}

	return c
}
