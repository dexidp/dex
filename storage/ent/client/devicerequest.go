package client

import (
	"context"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/ent/db/devicerequest"
)

// CreateDeviceRequest saves provided device request into the database.
func (d *Database) CreateDeviceRequest(ctx context.Context, request storage.DeviceRequest) error {
	_, err := d.client.DeviceRequest.Create().
		SetClientID(request.ClientID).
		SetClientSecret(request.ClientSecret).
		SetScopes(request.Scopes).
		SetUserCode(request.UserCode).
		SetDeviceCode(request.DeviceCode).
		// Save utc time into database because ent doesn't support comparing dates with different timezones
		SetExpiry(request.Expiry.UTC()).
		Save(ctx)
	if err != nil {
		return convertDBError("create device request: %w", err)
	}
	return nil
}

// GetDeviceRequest extracts a device request from the database by user code.
func (d *Database) GetDeviceRequest(ctx context.Context, userCode string) (storage.DeviceRequest, error) {
	deviceRequest, err := d.client.DeviceRequest.Query().
		Where(devicerequest.UserCode(userCode)).
		Only(ctx)
	if err != nil {
		return storage.DeviceRequest{}, convertDBError("get device request: %w", err)
	}
	return toStorageDeviceRequest(deviceRequest), nil
}
