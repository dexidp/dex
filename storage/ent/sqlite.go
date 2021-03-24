package ent

import (
	"context"
	"crypto/sha256"
	"strings"

	"entgo.io/ent/dialect/sql"

	// Register sqlite driver.
	_ "github.com/mattn/go-sqlite3"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/ent/client"
	"github.com/dexidp/dex/storage/ent/db"
)

// SQLite3 options for creating an SQL db.
type SQLite3 struct {
	File string `json:"file"`
}

// Open always returns a new in sqlite3 storage.
func (s *SQLite3) Open(logger log.Logger) (storage.Storage, error) {
	logger.Debug("experimental ent-based storage driver is enabled")

	// Implicitly set foreign_keys pragma to "on" because it is required by ent
	s.File = addFK(s.File)

	drv, err := sql.Open("sqlite3", s.File)
	if err != nil {
		return nil, err
	}

	pool := drv.DB()
	if s.File == ":memory:" {
		// sqlite3 uses file locks to coordinate concurrent access. In memory
		// doesn't support this, so limit the number of connections to 1.
		pool.SetMaxOpenConns(1)
	}

	databaseClient := client.NewDatabase(
		client.WithClient(db.NewClient(db.Driver(drv))),
		client.WithHasher(sha256.New),
	)

	if err := databaseClient.Schema().Create(context.TODO()); err != nil {
		return nil, err
	}

	return databaseClient, nil
}

func addFK(dsn string) string {
	if strings.Contains(dsn, "_fk") {
		return dsn
	}

	delim := "?"
	if strings.Contains(dsn, "?") {
		delim = "&"
	}
	return dsn + delim + "_fk=1"
}
