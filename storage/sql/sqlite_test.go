// +build cgo

package sql

import (
	"testing"
)

func TestSQLite3(t *testing.T) {
	testDB(t, &SQLite3{":memory:"}, false)
}
