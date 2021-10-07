package client

import (
	"context"
	"errors"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/ent/db"
)

func getKeys(client *db.KeysClient) (storage.Keys, error) {
	rawKeys, err := client.Get(context.TODO(), keysRowID)
	if err != nil {
		return storage.Keys{}, convertDBError("get keys: %w", err)
	}

	return toStorageKeys(rawKeys), nil
}

// GetKeys returns signing keys, public keys and verification keys from the database.
func (d *Database) GetKeys() (storage.Keys, error) {
	return getKeys(d.client.Keys)
}

// UpdateKeys rotates keys using updater function.
func (d *Database) UpdateKeys(updater func(old storage.Keys) (storage.Keys, error)) error {
	firstUpdate := false

	tx, err := d.BeginTx(context.TODO())
	if err != nil {
		return convertDBError("update keys tx: %w", err)
	}

	storageKeys, err := getKeys(tx.Keys)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return rollback(tx, "update keys get: %w", err)
		}
		firstUpdate = true
	}

	newKeys, err := updater(storageKeys)
	if err != nil {
		return rollback(tx, "update keys updating: %w", err)
	}

	// ent doesn't have an upsert support yet
	// https://github.com/facebook/ent/issues/139
	if firstUpdate {
		_, err = tx.Keys.Create().
			SetID(keysRowID).
			SetNextRotation(newKeys.NextRotation).
			SetSigningKey(*newKeys.SigningKey).
			SetSigningKeyPub(*newKeys.SigningKeyPub).
			SetVerificationKeys(newKeys.VerificationKeys).
			Save(context.TODO())
		if err != nil {
			return rollback(tx, "create keys: %w", err)
		}
		if err = tx.Commit(); err != nil {
			return rollback(tx, "update keys commit: %w", err)
		}
		return nil
	}

	err = tx.Keys.UpdateOneID(keysRowID).
		SetNextRotation(newKeys.NextRotation.UTC()).
		SetSigningKey(*newKeys.SigningKey).
		SetSigningKeyPub(*newKeys.SigningKeyPub).
		SetVerificationKeys(newKeys.VerificationKeys).
		Exec(context.TODO())
	if err != nil {
		return rollback(tx, "update keys uploading: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return rollback(tx, "update keys commit: %w", err)
	}

	return nil
}
