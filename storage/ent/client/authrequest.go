package client

import (
	"context"
	"fmt"

	"github.com/dexidp/dex/storage"
)

// CreateAuthRequest saves provided auth request into the database.
func (d *Database) CreateAuthRequest(ctx context.Context, authRequest storage.AuthRequest) error {
	_, err := d.client.AuthRequest.Create().
		SetID(authRequest.ID).
		SetClientID(authRequest.ClientID).
		SetScopes(authRequest.Scopes).
		SetResponseTypes(authRequest.ResponseTypes).
		SetRedirectURI(authRequest.RedirectURI).
		SetState(authRequest.State).
		SetNonce(authRequest.Nonce).
		SetForceApprovalPrompt(authRequest.ForceApprovalPrompt).
		SetLoggedIn(authRequest.LoggedIn).
		SetClaimsUserID(authRequest.Claims.UserID).
		SetClaimsEmail(authRequest.Claims.Email).
		SetClaimsEmailVerified(authRequest.Claims.EmailVerified).
		SetClaimsUsername(authRequest.Claims.Username).
		SetClaimsPreferredUsername(authRequest.Claims.PreferredUsername).
		SetClaimsGroups(authRequest.Claims.Groups).
		SetCodeChallenge(authRequest.PKCE.CodeChallenge).
		SetCodeChallengeMethod(authRequest.PKCE.CodeChallengeMethod).
		// Save utc time into database because ent doesn't support comparing dates with different timezones
		SetExpiry(authRequest.Expiry.UTC()).
		SetConnectorID(authRequest.ConnectorID).
		SetConnectorData(authRequest.ConnectorData).
		SetHmacKey(authRequest.HMACKey).
		Save(ctx)
	if err != nil {
		return convertDBError("create auth request: %w", err)
	}
	return nil
}

// GetAuthRequest extracts an auth request from the database by id.
func (d *Database) GetAuthRequest(ctx context.Context, id string) (storage.AuthRequest, error) {
	authRequest, err := d.client.AuthRequest.Get(ctx, id)
	if err != nil {
		return storage.AuthRequest{}, convertDBError("get auth request: %w", err)
	}
	return toStorageAuthRequest(authRequest), nil
}

// DeleteAuthRequest deletes an auth request from the database by id.
func (d *Database) DeleteAuthRequest(ctx context.Context, id string) error {
	err := d.client.AuthRequest.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return convertDBError("delete auth request: %w", err)
	}
	return nil
}

// UpdateAuthRequest changes an auth request by id using an updater function and saves it to the database.
func (d *Database) UpdateAuthRequest(ctx context.Context, id string, updater func(old storage.AuthRequest) (storage.AuthRequest, error)) error {
	tx, err := d.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("update auth request tx: %w", err)
	}

	authRequest, err := tx.AuthRequest.Get(context.TODO(), id)
	if err != nil {
		return rollback(tx, "update auth request database: %w", err)
	}

	newAuthRequest, err := updater(toStorageAuthRequest(authRequest))
	if err != nil {
		return rollback(tx, "update auth request updating: %w", err)
	}

	_, err = tx.AuthRequest.UpdateOneID(newAuthRequest.ID).
		SetClientID(newAuthRequest.ClientID).
		SetScopes(newAuthRequest.Scopes).
		SetResponseTypes(newAuthRequest.ResponseTypes).
		SetRedirectURI(newAuthRequest.RedirectURI).
		SetState(newAuthRequest.State).
		SetNonce(newAuthRequest.Nonce).
		SetForceApprovalPrompt(newAuthRequest.ForceApprovalPrompt).
		SetLoggedIn(newAuthRequest.LoggedIn).
		SetClaimsUserID(newAuthRequest.Claims.UserID).
		SetClaimsEmail(newAuthRequest.Claims.Email).
		SetClaimsEmailVerified(newAuthRequest.Claims.EmailVerified).
		SetClaimsUsername(newAuthRequest.Claims.Username).
		SetClaimsPreferredUsername(newAuthRequest.Claims.PreferredUsername).
		SetClaimsGroups(newAuthRequest.Claims.Groups).
		SetCodeChallenge(newAuthRequest.PKCE.CodeChallenge).
		SetCodeChallengeMethod(newAuthRequest.PKCE.CodeChallengeMethod).
		// Save utc time into database because ent doesn't support comparing dates with different timezones
		SetExpiry(newAuthRequest.Expiry.UTC()).
		SetConnectorID(newAuthRequest.ConnectorID).
		SetConnectorData(newAuthRequest.ConnectorData).
		SetHmacKey(newAuthRequest.HMACKey).
		Save(context.TODO())
	if err != nil {
		return rollback(tx, "update auth request uploading: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return rollback(tx, "update auth request commit: %w", err)
	}

	return nil
}
