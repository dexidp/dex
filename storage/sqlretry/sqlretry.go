// Package sqlretry provides a shared retry policy for transient SQL transaction
// failures (serialization failures / deadlocks under SERIALIZABLE isolation).
//
// It is used by both SQL-backed storages — storage/sql (raw database/sql) and
// storage/ent (the ent ORM) — which both run refresh-token rotation in a
// SERIALIZABLE transaction and must be prepared to retry transactions the
// database aborts under concurrency.
package sqlretry

import (
	"errors"
	"math/rand"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
)

// MaxRetries is the maximum number of retries; the initial attempt is not
// counted. Postgres requires applications to be prepared to retry transactions
// aborted with SQLSTATE 40001; see
// https://www.postgresql.org/docs/current/transaction-iso.html.
//
// 8 retries comfortably absorbs realistic refresh-token contention; combined
// with jittered backoff — which de-synchronizes the retrying transactions so
// effective concurrency at any instant stays low — it also survives pathological
// high-contention conformance tests, while bounding worst-case latency.
const MaxRetries = 8

// backoffMs is the per-attempt base backoff in milliseconds. The last element is
// reused for any attempt beyond its length; a random jitter of up to the same
// magnitude is added on top to de-synchronize retrying transactions (avoid a
// thundering herd).
var backoffMs = []int{5, 10, 25, 50, 100}

const (
	pgErrSerializationFailure = "40001" // serialization_failure
	pgErrDeadlockDetected     = "40P01" // deadlock_detected
	mysqlErrLockDeadlock      = 1213    // ER_LOCK_DEADLOCK
	mysqlErrLockWaitTimeout   = 1205    // ER_LOCK_WAIT_TIMEOUT
)

// IsSerializationFailure reports whether err is a transient transaction failure
// (serialization failure or deadlock) that is safe to retry by re-running the
// whole transaction. It understands both the lib/pq and go-sql-driver/mysql
// error types, unwrapping the error chain with errors.As.
func IsSerializationFailure(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == pgErrSerializationFailure || pqErr.Code == pgErrDeadlockDetected
	}

	var myErr *mysql.MySQLError
	if errors.As(err, &myErr) {
		return myErr.Number == mysqlErrLockDeadlock || myErr.Number == mysqlErrLockWaitTimeout
	}

	return false
}

// Do runs fn, retrying the whole operation on transient serialization/deadlock
// failures with bounded, jittered backoff. Retrying is safe only when fn opens a
// fresh transaction and re-reads current state on each attempt.
//
// isRetryable classifies an error as transient; if nil, no retries are performed
// (used by backends, such as SQLite, that don't surface serialization failures).
// onRetry, if non-nil, is called before each backoff sleep with the upcoming
// (1-based) attempt number and the error that triggered the retry — e.g. for
// logging.
func Do(fn func() error, isRetryable func(error) bool, onRetry func(attempt int, err error)) error {
	for attempt := 0; ; attempt++ {
		err := fn()
		if err == nil || isRetryable == nil || !isRetryable(err) || attempt >= MaxRetries {
			return err
		}

		if onRetry != nil {
			onRetry(attempt+1, err)
		}

		backoff := backoffMs[min(attempt, len(backoffMs)-1)]
		time.Sleep(time.Duration(backoff+rand.Intn(backoff+1)) * time.Millisecond)
	}
}
