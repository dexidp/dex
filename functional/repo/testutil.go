package repo

import (
	"fmt"

	"os"

	"github.com/coreos/dex/db"
	_ "github.com/coreos/dex/db/postgresql"
)

func initDB(dsn string) db.Driver {
	storage := "postgresql"
	ts := os.Getenv("DEX_TEST_STORAGE")
	if ts != "" {
		storage = ts
	}

	rd := db.GetDriver(storage)
	if rd == nil {
		panic("storage driver not found:" + storage)
	}

	s, err := rd.NewWithMap(map[string]interface{}{"url": dsn})
	if err != nil {
		panic(fmt.Sprintf("Unable to connect to database: %v", err))
	}

	if err = s.DropTablesIfExists(); err != nil {
		panic(fmt.Sprintf("Unable to drop database tables: %v", err))
	}

	if err = s.DropMigrationsTable(); err != nil {
		panic(fmt.Sprintf("Unable to drop migration table: %v", err))
	}

	if _, err = s.MigrateToLatest(); err != nil {
		panic(fmt.Sprintf("Unable to migrate: %v", err))
	}
	return s
}
