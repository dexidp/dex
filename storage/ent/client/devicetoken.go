package client

import (
	"context"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/ent/db/devicetoken"
)

// CreateDeviceToken saves provided token into the database.
func (d *Database) CreateDeviceToken(ctx context.Context, token storage.DeviceToken) error {
	_, err := d.client.DeviceToken.Create().
		SetDeviceCode(token.DeviceCode).
		SetToken([]byte(token.Token)).
		SetPollInterval(token.PollIntervalSeconds).
		// Save utc time into database because ent doesn't support comparing dates with different timezones
		SetExpiry(token.Expiry.UTC()).
		SetLastRequest(token.LastRequestTime.UTC()).
		SetStatus(token.Status).
		SetCodeChallenge(token.PKCE.CodeChallenge).
		SetCodeChallengeMethod(token.PKCE.CodeChallengeMethod).
		Save(ctx)
	if err != nil {
		return convertDBError("create device token: %w", err)
	}
	return nil
}

// GetDeviceToken extracts a token from the database by device code.
func (d *Database) GetDeviceToken(ctx context.Context, deviceCode string) (storage.DeviceToken, error) {
	deviceToken, err := d.client.DeviceToken.Query().
		Where(devicetoken.DeviceCode(deviceCode)).
		Only(ctx)
	if err != nil {
		return storage.DeviceToken{}, convertDBError("get device token: %w", err)
	}
	return toStorageDeviceToken(deviceToken), nil
}

// UpdateDeviceToken changes a token by device code using an updater function and saves it to the database.
func (d *Database) UpdateDeviceToken(ctx context.Context, deviceCode string, updater func(old storage.DeviceToken) (storage.DeviceToken, error)) error {
	tx, err := d.BeginTx(ctx)
	if err != nil {
		return convertDBError("update device token tx: %w", err)
	}

	token, err := tx.DeviceToken.Query().
		Where(devicetoken.DeviceCode(deviceCode)).
		Only(ctx)
	if err != nil {
		return rollback(tx, "update device token database: %w", err)
	}

	newToken, err := updater(toStorageDeviceToken(token))
	if err != nil {
		return rollback(tx, "update device token updating: %w", err)
	}

	_, err = tx.DeviceToken.Update().
		Where(devicetoken.DeviceCode(newToken.DeviceCode)).
		SetDeviceCode(newToken.DeviceCode).
		SetToken([]byte(newToken.Token)).
		SetPollInterval(newToken.PollIntervalSeconds).
		// Save utc time into database because ent doesn't support comparing dates with different timezones
		SetExpiry(newToken.Expiry.UTC()).
		SetLastRequest(newToken.LastRequestTime.UTC()).
		SetStatus(newToken.Status).
		SetCodeChallenge(newToken.PKCE.CodeChallenge).
		SetCodeChallengeMethod(newToken.PKCE.CodeChallengeMethod).
		Save(ctx)
	if err != nil {
		return rollback(tx, "update device token uploading: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return rollback(tx, "update device token commit: %w", err)
	}

	return nil
}
