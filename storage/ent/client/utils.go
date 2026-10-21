package client

import (
	"fmt"
	"hash"
	"time"

	"github.com/pkg/errors"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/ent/db"
)

func rollback(tx *db.Tx, t string, err error) error {
	rerr := tx.Rollback()
	err = convertDBError(t, err)

	if rerr == nil {
		return err
	}
	return errors.Wrapf(err, "rolling back transaction: %v", rerr)
}

func convertDBError(t string, err error) error {
	if db.IsNotFound(err) {
		return storage.ErrNotFound
	}

	if db.IsConstraintError(err) {
		return storage.ErrAlreadyExists
	}

	return fmt.Errorf(t, err)
}

// compositeKeyID composes a hashed id from two key parts to use as primary key.
// ent doesn't support multi-key primary yet
// https://github.com/facebook/ent/issues/400
func compositeKeyID(first string, second string, hasher func() hash.Hash) string {
	h := hasher()

	h.Write([]byte(first))
	h.Write([]byte(second))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// nonZeroTime returns nil for the zero time.Time, so an unset optional time
// field is stored as NULL. MySQL rejects the zero value ('0000-00-00') under
// its default sql_mode.
func nonZeroTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
