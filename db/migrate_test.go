package db

import (
	"fmt"
	"os"
	"testing"

	"github.com/coopernurse/gorp"
)

func initDB(dsn string) *gorp.DbMap {
	c, err := NewConnection(Config{DSN: dsn})
	if err != nil {
		panic(fmt.Sprintf("error making db connection: %q", err))
	}
	if err = c.DropTablesIfExists(); err != nil {
		panic(fmt.Sprintf("Unable to drop database tables: %v", err))
	}

	return c
}

// TestGetPlannedMigrations is a sanity check, ensuring that at least one
// migration can be found.
func TestGetPlannedMigrations(t *testing.T) {
	dsn := os.Getenv("DEX_TEST_DSN")
	if dsn == "" {
		t.Logf("Test will not run without DEX_TEST_DSN environment variable.")
		return
	}
	dbMap := initDB(dsn)
	ms, err := GetPlannedMigrations(dbMap)
	if err != nil {
		pwd, err := os.Getwd()
		t.Logf("pwd: %v", pwd)
		t.Fatalf("unexpected err: %q", err)
	}

	if len(ms) == 0 {
		t.Fatalf("expected non-empty migrations")
	}
}
