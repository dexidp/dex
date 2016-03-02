package db

// Register the postgres driver.

import "github.com/lib/pq"

func init() {
	registerAlreadyExistsChecker(func(err error) bool {
		sqlErr, ok := err.(*pq.Error)
		if !ok {
			return false
		}
		return sqlErr.Code == pgErrorCodeUniqueViolation
	})
}
