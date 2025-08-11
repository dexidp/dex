//go:build !cgo
// +build !cgo

// This is a stub for the no CGO compilation (CGO_ENABLED=0)

package sql

import (
	"fmt"
	"log/slog"

	"github.com/dexidp/dex/storage"
)

type SQLite3 struct{}

func (s *SQLite3) Open(logger *slog.Logger) (storage.Storage, error) {
	return nil, fmt.Errorf("Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work")
}
