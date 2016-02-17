// Package db provides SQL implementations of dex's storage interfaces.
package db

import (
	"github.com/go-gorp/gorp"

	"github.com/coreos/dex/db/translate"
	"github.com/coreos/dex/repo"
)

// db is the connection type passed to repos.
//
// TODO(ericchiang): Eventually just return this instead of gorp.DbMap during Conn.
// All actions should go through this type instead of dbMap.
type db struct {
	dbMap *gorp.DbMap
}

// executor returns a driver agnostic SQL executor.
//
// The expected flavor of all queries is the flavor used by github.com/lib/pq. All bind
// parameters must be unique and in sequential order (e.g. $1, $2, ...).
//
// See github.com/coreos/dex/db/translate for details on the translation.
//
// If tx is nil, a non-transaction context is provided.
func (conn *db) executor(tx repo.Transaction) gorp.SqlExecutor {
	var exec gorp.SqlExecutor
	if tx == nil {
		exec = conn.dbMap
	} else {
		gorpTx, ok := tx.(*gorp.Transaction)
		if !ok {
			panic("wrong kind of transaction passed to a DB repo")
		}

		// Check if the underlying value of the pointer is nil.
		// This is not caught by the initial comparison (tx == nil).
		if gorpTx == nil {
			exec = conn.dbMap
		} else {
			exec = gorpTx
		}
	}

	if _, ok := conn.dbMap.Dialect.(gorp.SqliteDialect); ok {
		exec = translate.NewTranslatingExecutor(exec, translate.PostgresToSQLite)
	}
	return exec
}

// quote escapes a table name for a driver specific SQL query. quote uses the
// gorp's package underlying quote logic and should NOT be used on untrusted input.
func (conn *db) quote(tableName string) string {
	return conn.dbMap.Dialect.QuotedTableForQuery("", tableName)
}

func (conn *db) begin() (repo.Transaction, error) {
	return conn.dbMap.Begin()
}
