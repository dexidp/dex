package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dexidp/dex/storage"
)

// CreateUserIdentity saves provided user identity into the database.
func (d *Database) CreateUserIdentity(ctx context.Context, identity storage.UserIdentity) error {
	if identity.Consents == nil {
		identity.Consents = make(map[string][]string)
	}
	encodedConsents, err := json.Marshal(identity.Consents)
	if err != nil {
		return fmt.Errorf("encode consents user identity: %w", err)
	}

	id := compositeKeyID(identity.UserID, identity.ConnectorID, d.hasher)
	_, err = d.client.UserIdentity.Create().
		SetID(id).
		SetUserID(identity.UserID).
		SetConnectorID(identity.ConnectorID).
		SetClaimsUserID(identity.Claims.UserID).
		SetClaimsUsername(identity.Claims.Username).
		SetClaimsPreferredUsername(identity.Claims.PreferredUsername).
		SetClaimsEmail(identity.Claims.Email).
		SetClaimsEmailVerified(identity.Claims.EmailVerified).
		SetClaimsGroups(identity.Claims.Groups).
		SetConsents(encodedConsents).
		SetCreatedAt(identity.CreatedAt).
		SetLastLogin(identity.LastLogin).
		SetBlockedUntil(identity.BlockedUntil).
		Save(ctx)
	if err != nil {
		return convertDBError("create user identity: %w", err)
	}
	return nil
}

// GetUserIdentity extracts a user identity from the database by user id and connector id.
func (d *Database) GetUserIdentity(ctx context.Context, userID, connectorID string) (storage.UserIdentity, error) {
	id := compositeKeyID(userID, connectorID, d.hasher)

	userIdentity, err := d.client.UserIdentity.Get(ctx, id)
	if err != nil {
		return storage.UserIdentity{}, convertDBError("get user identity: %w", err)
	}
	return toStorageUserIdentity(userIdentity), nil
}

// DeleteUserIdentity deletes a user identity from the database by user id and connector id.
func (d *Database) DeleteUserIdentity(ctx context.Context, userID, connectorID string) error {
	id := compositeKeyID(userID, connectorID, d.hasher)

	err := d.client.UserIdentity.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return convertDBError("delete user identity: %w", err)
	}
	return nil
}

// UpdateUserIdentity changes a user identity by user id and connector id using an updater function.
func (d *Database) UpdateUserIdentity(ctx context.Context, userID string, connectorID string, updater func(u storage.UserIdentity) (storage.UserIdentity, error)) error {
	id := compositeKeyID(userID, connectorID, d.hasher)

	tx, err := d.BeginTx(ctx)
	if err != nil {
		return convertDBError("update user identity tx: %w", err)
	}

	userIdentity, err := tx.UserIdentity.Get(ctx, id)
	if err != nil {
		return rollback(tx, "update user identity database: %w", err)
	}

	newUserIdentity, err := updater(toStorageUserIdentity(userIdentity))
	if err != nil {
		return rollback(tx, "update user identity updating: %w", err)
	}

	if newUserIdentity.Consents == nil {
		newUserIdentity.Consents = make(map[string][]string)
	}

	encodedConsents, err := json.Marshal(newUserIdentity.Consents)
	if err != nil {
		return rollback(tx, "encode consents user identity: %w", err)
	}

	_, err = tx.UserIdentity.UpdateOneID(id).
		SetUserID(newUserIdentity.UserID).
		SetConnectorID(newUserIdentity.ConnectorID).
		SetClaimsUserID(newUserIdentity.Claims.UserID).
		SetClaimsUsername(newUserIdentity.Claims.Username).
		SetClaimsPreferredUsername(newUserIdentity.Claims.PreferredUsername).
		SetClaimsEmail(newUserIdentity.Claims.Email).
		SetClaimsEmailVerified(newUserIdentity.Claims.EmailVerified).
		SetClaimsGroups(newUserIdentity.Claims.Groups).
		SetConsents(encodedConsents).
		SetCreatedAt(newUserIdentity.CreatedAt).
		SetLastLogin(newUserIdentity.LastLogin).
		SetBlockedUntil(newUserIdentity.BlockedUntil).
		Save(ctx)
	if err != nil {
		return rollback(tx, "update user identity uploading: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return rollback(tx, "update user identity commit: %w", err)
	}

	return nil
}

// ListUserIdentities lists all user identities in the database.
func (d *Database) ListUserIdentities(ctx context.Context) ([]storage.UserIdentity, error) {
	userIdentities, err := d.client.UserIdentity.Query().All(ctx)
	if err != nil {
		return nil, convertDBError("list user identities: %w", err)
	}

	storageUserIdentities := make([]storage.UserIdentity, 0, len(userIdentities))
	for _, u := range userIdentities {
		storageUserIdentities = append(storageUserIdentities, toStorageUserIdentity(u))
	}
	return storageUserIdentities, nil
}
