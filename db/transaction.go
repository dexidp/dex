package db

import (
	"github.com/go-gorp/gorp"

	"github.com/coreos/dex/db/translate"
	"github.com/coreos/dex/repo"
)

func executor(dbMap *gorp.DbMap, tx repo.Transaction) gorp.SqlExecutor {
	var exec gorp.SqlExecutor
	if tx == nil {
		exec = dbMap
	} else {
		gorpTx, ok := tx.(*gorp.Transaction)
		if !ok {
			panic("wrong kind of transaction passed to a DB repo")
		}

		// Check if the underlying value of the pointer is nil.
		// This is not caught by the initial comparison (tx == nil).
		if gorpTx == nil {
			exec = dbMap
		} else {
			exec = gorpTx
		}
	}

	if _, ok := dbMap.Dialect.(gorp.SqliteDialect); ok {
		exec = translate.NewExecutor(exec, translate.PostgresToSQLite)
	}
	return exec
}
