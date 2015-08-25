package repo

import (
	"fmt"

	"github.com/coreos/dex/db"
	"github.com/go-gorp/gorp"
)

func initDB(dsn string) *gorp.DbMap {
	c, err := db.NewConnection(db.Config{DSN: dsn})
	if err != nil {
		panic(fmt.Sprintf("Unable to connect to database: %v", err))
	}

	if err = c.DropTablesIfExists(); err != nil {
		panic(fmt.Sprintf("Unable to drop database tables: %v", err))
	}

	if err = db.DropMigrationsTable(c); err != nil {
		panic(fmt.Sprintf("Unable to drop migration table: %v", err))
	}

	if _, err = db.MigrateToLatest(c); err != nil {
		panic(fmt.Sprintf("Unable to migrate: %v", err))
	}
	return c
}
