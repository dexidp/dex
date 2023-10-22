package client

import (
	"fmt"
	"hash"

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

// compose hashed id from user and connection id to use it as primary key
// ent doesn't support multi-key primary yet
// https://github.com/facebook/ent/issues/400
func offlineSessionID(userID string, connID string, hasher func() hash.Hash) string {
	h := hasher()

	h.Write([]byte(userID))
	h.Write([]byte(connID))
	return fmt.Sprintf("%x", h.Sum(nil))
}
