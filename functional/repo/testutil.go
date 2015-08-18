package repo

import (
	"fmt"

	"github.com/coopernurse/gorp"

	"github.com/coreos/dex/db"
)

func initDB(dsn string) *gorp.DbMap {
	c, err := db.NewConnection(db.Config{DSN: dsn})
	if err != nil {
		panic(fmt.Sprintf("Unable to connect to database: %v", err))
	}

	if err = c.DropTablesIfExists(); err != nil {
		panic(fmt.Sprintf("Unable to drop database tables: %v", err))
	}

	if err = c.CreateTablesIfNotExists(); err != nil {
		panic(fmt.Sprintf("Unable to create database tables: %v", err))
	}
	return c
}
