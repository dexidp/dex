package client

import (
	"context"

	"github.com/dexidp/dex/storage"
)

// CreateRefresh saves provided refresh token into the database.
func (d *Database) CreateRefresh(ctx context.Context, refresh storage.RefreshToken) error {
	_, err := d.client.RefreshToken.Create().
		SetID(refresh.ID).
		SetClientID(refresh.ClientID).
		SetScopes(refresh.Scopes).
		SetNonce(refresh.Nonce).
		SetClaimsUserID(refresh.Claims.UserID).
		SetClaimsEmail(refresh.Claims.Email).
		SetClaimsEmailVerified(refresh.Claims.EmailVerified).
		SetClaimsUsername(refresh.Claims.Username).
		SetClaimsPreferredUsername(refresh.Claims.PreferredUsername).
		SetClaimsGroups(refresh.Claims.Groups).
		SetConnectorID(refresh.ConnectorID).
		SetConnectorData(refresh.ConnectorData).
		SetToken(refresh.Token).
		SetObsoleteToken(refresh.ObsoleteToken).
		// Save utc time into database because ent doesn't support comparing dates with different timezones
		SetLastUsed(refresh.LastUsed.UTC()).
		SetCreatedAt(refresh.CreatedAt.UTC()).
		Save(ctx)
	if err != nil {
		return convertDBError("create refresh token: %w", err)
	}
	return nil
}

// ListRefreshTokens extracts an array of refresh tokens from the database.
func (d *Database) ListRefreshTokens(ctx context.Context) ([]storage.RefreshToken, error) {
	refreshTokens, err := d.client.RefreshToken.Query().All(ctx)
	if err != nil {
		return nil, convertDBError("list refresh tokens: %w", err)
	}

	storageRefreshTokens := make([]storage.RefreshToken, 0, len(refreshTokens))
	for _, r := range refreshTokens {
		storageRefreshTokens = append(storageRefreshTokens, toStorageRefreshToken(r))
	}
	return storageRefreshTokens, nil
}

// GetRefresh extracts a refresh token from the database by id.
func (d *Database) GetRefresh(ctx context.Context, id string) (storage.RefreshToken, error) {
	refreshToken, err := d.client.RefreshToken.Get(ctx, id)
	if err != nil {
		return storage.RefreshToken{}, convertDBError("get refresh token: %w", err)
	}
	return toStorageRefreshToken(refreshToken), nil
}

// DeleteRefresh deletes a refresh token from the database by id.
func (d *Database) DeleteRefresh(ctx context.Context, id string) error {
	err := d.client.RefreshToken.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return convertDBError("delete refresh token: %w", err)
	}
	return nil
}

// UpdateRefreshToken changes a refresh token by id using an updater function and saves it to the database.
func (d *Database) UpdateRefreshToken(ctx context.Context, id string, updater func(old storage.RefreshToken) (storage.RefreshToken, error)) error {
	tx, err := d.BeginTx(ctx)
	if err != nil {
		return convertDBError("update refresh token tx: %w", err)
	}

	token, err := tx.RefreshToken.Get(ctx, id)
	if err != nil {
		return rollback(tx, "update refresh token database: %w", err)
	}

	newtToken, err := updater(toStorageRefreshToken(token))
	if err != nil {
		return rollback(tx, "update refresh token updating: %w", err)
	}

	_, err = tx.RefreshToken.UpdateOneID(newtToken.ID).
		SetClientID(newtToken.ClientID).
		SetScopes(newtToken.Scopes).
		SetNonce(newtToken.Nonce).
		SetClaimsUserID(newtToken.Claims.UserID).
		SetClaimsEmail(newtToken.Claims.Email).
		SetClaimsEmailVerified(newtToken.Claims.EmailVerified).
		SetClaimsUsername(newtToken.Claims.Username).
		SetClaimsPreferredUsername(newtToken.Claims.PreferredUsername).
		SetClaimsGroups(newtToken.Claims.Groups).
		SetConnectorID(newtToken.ConnectorID).
		SetConnectorData(newtToken.ConnectorData).
		SetToken(newtToken.Token).
		SetObsoleteToken(newtToken.ObsoleteToken).
		// Save utc time into database because ent doesn't support comparing dates with different timezones
		SetLastUsed(newtToken.LastUsed.UTC()).
		SetCreatedAt(newtToken.CreatedAt.UTC()).
		Save(ctx)
	if err != nil {
		return rollback(tx, "update refresh token uploading: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return rollback(tx, "update refresh token commit: %w", err)
	}
	return nil
}
