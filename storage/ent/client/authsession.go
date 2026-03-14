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

	_, err = d.client.AuthSession.Create().
		SetID(session.ID).
		SetClientStates(encodedStates).
		SetCreatedAt(session.CreatedAt).
		SetLastActivity(session.LastActivity).
		SetIPAddress(session.IPAddress).
		SetUserAgent(session.UserAgent).
		Save(ctx)
	if err != nil {
		return convertDBError("create auth session: %w", err)
	}
	return nil
}

// GetAuthSession extracts an auth session from the database by session ID.
func (d *Database) GetAuthSession(ctx context.Context, sessionID string) (storage.AuthSession, error) {
	authSession, err := d.client.AuthSession.Get(ctx, sessionID)
	if err != nil {
		return storage.AuthSession{}, convertDBError("get auth session: %w", err)
	}
	return toStorageAuthSession(authSession), nil
}

// DeleteAuthSession deletes an auth session from the database by session ID.
func (d *Database) DeleteAuthSession(ctx context.Context, sessionID string) error {
	err := d.client.AuthSession.DeleteOneID(sessionID).Exec(ctx)
	if err != nil {
		return convertDBError("delete auth session: %w", err)
	}
	return nil
}

// UpdateAuthSession changes an auth session using an updater function.
func (d *Database) UpdateAuthSession(ctx context.Context, sessionID string, updater func(s storage.AuthSession) (storage.AuthSession, error)) error {
	tx, err := d.BeginTx(ctx)
	if err != nil {
		return convertDBError("update auth session tx: %w", err)
	}

	authSession, err := tx.AuthSession.Get(ctx, sessionID)
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

	_, err = tx.AuthSession.UpdateOneID(sessionID).
		SetClientStates(encodedStates).
		SetCreatedAt(newSession.CreatedAt).
		SetLastActivity(newSession.LastActivity).
		SetIPAddress(newSession.IPAddress).
		SetUserAgent(newSession.UserAgent).
		Save(ctx)
	if err != nil {
		return rollback(tx, "update auth session uploading: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return rollback(tx, "update auth session commit: %w", err)
	}

	return nil
}
