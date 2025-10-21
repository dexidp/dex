//go:build cgo
// +build cgo

package sql

import (
	"database/sql"
	"fmt"
	"log/slog"

	sqlite3 "github.com/mattn/go-sqlite3"

	"github.com/dexidp/dex/storage"
)

// SQLite3 options for creating an SQL db.
type SQLite3 struct {
	// File to
	File       string           `json:"file"`
	Encryption EncryptionConfig `json:"encryption" yaml:"encryption"`
}

// Open creates a new storage implementation backed by SQLite3
func (s *SQLite3) Open(logger *slog.Logger) (storage.Storage, error) {
	conn, err := s.open(logger)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (s *SQLite3) open(logger *slog.Logger) (*conn, error) {
	db, err := sql.Open("sqlite3", s.File)
	if err != nil {
		return nil, err
	}

	// always allow only one connection to sqlite3, any other thread/go-routine
	// attempting concurrent access will have to wait
	db.SetMaxOpenConns(1)
	errCheck := func(err error) bool {
		sqlErr, ok := err.(sqlite3.Error)
		if !ok {
			return false
		}
		return sqlErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey
	}

	encryptionSvc, err := setupEncryption(&s.Encryption, logger)
	if err != nil {
		return nil, fmt.Errorf("encryption setup failed: %v", err)
	}

	c := &conn{
		db:                 db,
		flavor:             &flavorSQLite3,
		logger:             logger,
		alreadyExistsCheck: errCheck,
		encryption:         encryptionSvc,
	}

	if _, err := c.migrate(); err != nil {
		return nil, fmt.Errorf("failed to perform migrations: %v", err)
	}

	if encryptionSvc.IsEnabled() {
		logger.Info("checking for unencrypted connectors to migrate")
		if err := c.migrateUnencryptedConnectors(); err != nil {
			logger.Warn("connector encryption migration had errors", "error", err)
			// Don't fail startup - log and continue
		}
	}
	return c, nil
}
