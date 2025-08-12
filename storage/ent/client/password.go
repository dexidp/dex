package client

import (
	"context"
	"strings"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/ent/db/password"
)

// CreatePassword saves provided password into the database.
func (d *Database) CreatePassword(ctx context.Context, password storage.Password) error {
	_, err := d.client.Password.Create().
		SetEmail(password.Email).
		SetHash(password.Hash).
		SetUsername(password.Username).
		SetUserID(password.UserID).
		Save(ctx)
	if err != nil {
		return convertDBError("create password: %w", err)
	}
	return nil
}

// ListPasswords extracts an array of passwords from the database.
func (d *Database) ListPasswords(ctx context.Context) ([]storage.Password, error) {
	passwords, err := d.client.Password.Query().All(ctx)
	if err != nil {
		return nil, convertDBError("list passwords: %w", err)
	}

	storagePasswords := make([]storage.Password, 0, len(passwords))
	for _, p := range passwords {
		storagePasswords = append(storagePasswords, toStoragePassword(p))
	}
	return storagePasswords, nil
}

// GetPassword extracts a password from the database by email.
func (d *Database) GetPassword(ctx context.Context, email string) (storage.Password, error) {
	email = strings.ToLower(email)
	passwordFromStorage, err := d.client.Password.Query().
		Where(password.Email(email)).
		Only(ctx)
	if err != nil {
		return storage.Password{}, convertDBError("get password: %w", err)
	}
	return toStoragePassword(passwordFromStorage), nil
}

// DeletePassword deletes a password from the database by email.
func (d *Database) DeletePassword(ctx context.Context, email string) error {
	email = strings.ToLower(email)
	_, err := d.client.Password.Delete().
		Where(password.Email(email)).
		Exec(ctx)
	if err != nil {
		return convertDBError("delete password: %w", err)
	}
	return nil
}

// UpdatePassword changes a password by email using an updater function and saves it to the database.
func (d *Database) UpdatePassword(ctx context.Context, email string, updater func(old storage.Password) (storage.Password, error)) error {
	email = strings.ToLower(email)

	tx, err := d.BeginTx(ctx)
	if err != nil {
		return convertDBError("update connector tx: %w", err)
	}

	passwordToUpdate, err := tx.Password.Query().
		Where(password.Email(email)).
		Only(ctx)
	if err != nil {
		return rollback(tx, "update password database: %w", err)
	}

	newPassword, err := updater(toStoragePassword(passwordToUpdate))
	if err != nil {
		return rollback(tx, "update password updating: %w", err)
	}

	_, err = tx.Password.Update().
		Where(password.Email(newPassword.Email)).
		SetEmail(newPassword.Email).
		SetHash(newPassword.Hash).
		SetUsername(newPassword.Username).
		SetUserID(newPassword.UserID).
		Save(ctx)
	if err != nil {
		return rollback(tx, "update password uploading: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return rollback(tx, "update password commit: %w", err)
	}

	return nil
}
