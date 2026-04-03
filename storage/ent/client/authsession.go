package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dexidp/dex/storage"
)

// CreateAuthSession saves provided auth session into the database.
func (d *Database) CreateAuthSession(ctx context.Context, session storage.AuthSession) error {
	if session.ClientStates == nil {
		session.ClientStates = make(map[string]*storage.ClientAuthState)
	}
	encodedStates, err := json.Marshal(session.ClientStates)
	if err != nil {
		return fmt.Errorf("encode client states auth session: %w", err)
	}

	id := compositeKeyID(session.UserID, session.ConnectorID, d.hasher)
	_, err = d.client.AuthSession.Create().
		SetID(id).
		SetUserID(session.UserID).
		SetConnectorID(session.ConnectorID).
		SetNonce(session.Nonce).
		SetClientStates(encodedStates).
		SetCreatedAt(session.CreatedAt).
		SetLastActivity(session.LastActivity).
		SetIPAddress(session.IPAddress).
		SetUserAgent(session.UserAgent).
		SetAbsoluteExpiry(session.AbsoluteExpiry.UTC()).
		SetIdleExpiry(session.IdleExpiry.UTC()).
		Save(ctx)
	if err != nil {
		return convertDBError("create auth session: %w", err)
	}
	return nil
}

// GetAuthSession extracts an auth session from the database by user ID and connector ID.
func (d *Database) GetAuthSession(ctx context.Context, userID, connectorID string) (storage.AuthSession, error) {
	id := compositeKeyID(userID, connectorID, d.hasher)
	authSession, err := d.client.AuthSession.Get(ctx, id)
	if err != nil {
		return storage.AuthSession{}, convertDBError("get auth session: %w", err)
	}
	return toStorageAuthSession(authSession), nil
}

// ListAuthSessions extracts all auth sessions from the database.
func (d *Database) ListAuthSessions(ctx context.Context) ([]storage.AuthSession, error) {
	authSessions, err := d.client.AuthSession.Query().All(ctx)
	if err != nil {
		return nil, convertDBError("list auth sessions: %w", err)
	}

	storageAuthSessions := make([]storage.AuthSession, 0, len(authSessions))
	for _, s := range authSessions {
		storageAuthSessions = append(storageAuthSessions, toStorageAuthSession(s))
	}
	return storageAuthSessions, nil
}

// DeleteAuthSession deletes an auth session from the database by user ID and connector ID.
func (d *Database) DeleteAuthSession(ctx context.Context, userID, connectorID string) error {
	id := compositeKeyID(userID, connectorID, d.hasher)
	err := d.client.AuthSession.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return convertDBError("delete auth session: %w", err)
	}
	return nil
}

// UpdateAuthSession changes an auth session using an updater function.
func (d *Database) UpdateAuthSession(ctx context.Context, userID, connectorID string, updater func(s storage.AuthSession) (storage.AuthSession, error)) error {
	id := compositeKeyID(userID, connectorID, d.hasher)
	tx, err := d.BeginTx(ctx)
	if err != nil {
		return convertDBError("update auth session tx: %w", err)
	}

	authSession, err := tx.AuthSession.Get(ctx, id)
	if err != nil {
		return rollback(tx, "update auth session database: %w", err)
	}

	newSession, err := updater(toStorageAuthSession(authSession))
	if err != nil {
		return rollback(tx, "update auth session updating: %w", err)
	}

	if newSession.ClientStates == nil {
		newSession.ClientStates = make(map[string]*storage.ClientAuthState)
	}

	encodedStates, err := json.Marshal(newSession.ClientStates)
	if err != nil {
		return rollback(tx, "encode client states auth session: %w", err)
	}

	_, err = tx.AuthSession.UpdateOneID(id).
		SetClientStates(encodedStates).
		SetLastActivity(newSession.LastActivity).
		SetIPAddress(newSession.IPAddress).
		SetUserAgent(newSession.UserAgent).
		SetAbsoluteExpiry(newSession.AbsoluteExpiry.UTC()).
		SetIdleExpiry(newSession.IdleExpiry.UTC()).
		Save(ctx)
	if err != nil {
		return rollback(tx, "update auth session updating: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return rollback(tx, "update auth session commit: %w", err)
	}

	return nil
}
