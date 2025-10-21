//go:build cgo
// +build cgo

package sql

import (
	"testing"
)

func TestSQLite3(t *testing.T) {
	testDB(t, &SQLite3{
		File: ":memory:",
		// EncryptionConfig has zero-value, disabled
	}, false)
}
