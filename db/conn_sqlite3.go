// +build cgo

package db

// Register the sqlite3 driver.

import "github.com/mattn/go-sqlite3"

func init() {
	registerAlreadyExistsChecker(func(err error) bool {
		sqlErr, ok := err.(sqlite3.Error)
		if !ok {
			return false
		}
		return sqlErr.ExtendedCode == sqlite3.ErrConstraintUnique
	})
}
