package client

import (
	"context"

	"github.com/dexidp/dex/storage"
)

// CreateAuthCode saves provided auth code into the database.
func (d *Database) CreateAuthCode(ctx context.Context, code storage.AuthCode) error {
	_, err := d.client.AuthCode.Create().
		SetID(code.ID).
		SetClientID(code.ClientID).
		SetScopes(code.Scopes).
		SetRedirectURI(code.RedirectURI).
		SetNonce(code.Nonce).
		SetClaimsUserID(code.Claims.UserID).
		SetClaimsEmail(code.Claims.Email).
		SetClaimsEmailVerified(code.Claims.EmailVerified).
		SetClaimsUsername(code.Claims.Username).
		SetClaimsPreferredUsername(code.Claims.PreferredUsername).
		SetClaimsGroups(code.Claims.Groups).
		SetCodeChallenge(code.PKCE.CodeChallenge).
		SetCodeChallengeMethod(code.PKCE.CodeChallengeMethod).
		// Save utc time into database because ent doesn't support comparing dates with different timezones
		SetExpiry(code.Expiry.UTC()).
		SetConnectorID(code.ConnectorID).
		SetConnectorData(code.ConnectorData).
		Save(ctx)
	if err != nil {
		return convertDBError("create auth code: %w", err)
	}
	return nil
}

// GetAuthCode extracts an auth code from the database by id.
func (d *Database) GetAuthCode(ctx context.Context, id string) (storage.AuthCode, error) {
	authCode, err := d.client.AuthCode.Get(ctx, id)
	if err != nil {
		return storage.AuthCode{}, convertDBError("get auth code: %w", err)
	}
	return toStorageAuthCode(authCode), nil
}

// DeleteAuthCode deletes an auth code from the database by id.
func (d *Database) DeleteAuthCode(ctx context.Context, id string) error {
	err := d.client.AuthCode.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return convertDBError("delete auth code: %w", err)
	}
	return nil
}
