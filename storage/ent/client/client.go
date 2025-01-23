package client

import (
	"context"

	"github.com/dexidp/dex/storage"
)

// CreateClient saves provided oauth2 client settings into the database.
func (d *Database) CreateClient(ctx context.Context, client storage.Client) error {
	_, err := d.client.OAuth2Client.Create().
		SetID(client.ID).
		SetName(client.Name).
		SetSecret(client.Secret).
		SetPublic(client.Public).
		SetLogoURL(client.LogoURL).
		SetRedirectUris(client.RedirectURIs).
		SetTrustedPeers(client.TrustedPeers).
		Save(ctx)
	if err != nil {
		return convertDBError("create oauth2 client: %w", err)
	}
	return nil
}

// ListClients extracts an array of oauth2 clients from the database.
func (d *Database) ListClients(ctx context.Context) ([]storage.Client, error) {
	clients, err := d.client.OAuth2Client.Query().All(ctx)
	if err != nil {
		return nil, convertDBError("list clients: %w", err)
	}

	storageClients := make([]storage.Client, 0, len(clients))
	for _, c := range clients {
		storageClients = append(storageClients, toStorageClient(c))
	}
	return storageClients, nil
}

// GetClient extracts an oauth2 client from the database by id.
func (d *Database) GetClient(ctx context.Context, id string) (storage.Client, error) {
	client, err := d.client.OAuth2Client.Get(ctx, id)
	if err != nil {
		return storage.Client{}, convertDBError("get client: %w", err)
	}
	return toStorageClient(client), nil
}

// DeleteClient deletes an oauth2 client from the database by id.
func (d *Database) DeleteClient(ctx context.Context, id string) error {
	err := d.client.OAuth2Client.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return convertDBError("delete client: %w", err)
	}
	return nil
}

// UpdateClient changes an oauth2 client by id using an updater function and saves it to the database.
func (d *Database) UpdateClient(ctx context.Context, id string, updater func(old storage.Client) (storage.Client, error)) error {
	tx, err := d.BeginTx(ctx)
	if err != nil {
		return convertDBError("update client tx: %w", err)
	}

	client, err := tx.OAuth2Client.Get(ctx, id)
	if err != nil {
		return rollback(tx, "update client database: %w", err)
	}

	newClient, err := updater(toStorageClient(client))
	if err != nil {
		return rollback(tx, "update client updating: %w", err)
	}

	_, err = tx.OAuth2Client.UpdateOneID(newClient.ID).
		SetName(newClient.Name).
		SetSecret(newClient.Secret).
		SetPublic(newClient.Public).
		SetLogoURL(newClient.LogoURL).
		SetRedirectUris(newClient.RedirectURIs).
		SetTrustedPeers(newClient.TrustedPeers).
		Save(ctx)
	if err != nil {
		return rollback(tx, "update client uploading: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return rollback(tx, "update auth request commit: %w", err)
	}

	return nil
}
