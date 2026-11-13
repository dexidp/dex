//go:build cgo
// +build cgo

package sql

import (
	"database/sql"
	"errors"
	"log/slog"
	"testing"

	"github.com/dexidp/dex/storage/sqlretry"
)

// errRetryable is a sentinel used to drive the txRetryCheck in tests.
var errRetryable = errors.New("retryable transient failure")

func newRetryTestConn(t *testing.T) *conn {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	return &conn{
		db:                 db,
		flavor:             &flavorSQLite3,
		logger:             slog.New(slog.DiscardHandler),
		alreadyExistsCheck: func(error) bool { return false },
		txRetryCheck:       func(err error) bool { return errors.Is(err, errRetryable) },
	}
}

func TestExecTxWithRetryRetriesUntilSuccess(t *testing.T) {
	c := newRetryTestConn(t)

	attempts := 0
	err := c.execTxWithRetry(func(*trans) error {
		attempts++
		if attempts <= 2 {
			return errRetryable
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts (2 retries), got %d", attempts)
	}
}

func TestExecTxWithRetryGivesUpAfterMaxRetries(t *testing.T) {
	c := newRetryTestConn(t)

	attempts := 0
	err := c.execTxWithRetry(func(*trans) error {
		attempts++
		return errRetryable
	})
	if !errors.Is(err, errRetryable) {
		t.Fatalf("expected retryable error to be returned, got: %v", err)
	}
	// 1 initial attempt + sqlretry.MaxRetries retries.
	if want := sqlretry.MaxRetries + 1; attempts != want {
		t.Fatalf("expected %d attempts, got %d", want, attempts)
	}
}

func TestExecTxWithRetryDoesNotRetryNonRetryableError(t *testing.T) {
	c := newRetryTestConn(t)

	sentinel := errors.New("permanent failure")
	attempts := 0
	err := c.execTxWithRetry(func(*trans) error {
		attempts++
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error to be returned, got: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("expected exactly 1 attempt for non-retryable error, got %d", attempts)
	}
}

func TestExecTxWithRetryNilRetryCheckDoesNotRetry(t *testing.T) {
	c := newRetryTestConn(t)
	c.txRetryCheck = nil

	attempts := 0
	err := c.execTxWithRetry(func(*trans) error {
		attempts++
		return errRetryable
	})
	if !errors.Is(err, errRetryable) {
		t.Fatalf("expected error to be returned, got: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("expected exactly 1 attempt when txRetryCheck is nil, got %d", attempts)
	}
}
