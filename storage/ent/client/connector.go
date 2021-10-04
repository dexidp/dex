package client

import (
	"context"

	"github.com/dexidp/dex/storage"
)

// CreateConnector saves a connector into the database.
func (d *Database) CreateConnector(connector storage.Connector) error {
	_, err := d.client.Connector.Create().
		SetID(connector.ID).
		SetName(connector.Name).
		SetType(connector.Type).
		SetResourceVersion(connector.ResourceVersion).
		SetConfig(connector.Config).
		Save(context.TODO())
	if err != nil {
		return convertDBError("create connector: %w", err)
	}
	return nil
}

// ListConnectors extracts an array of connectors from the database.
func (d *Database) ListConnectors() ([]storage.Connector, error) {
	connectors, err := d.client.Connector.Query().All(context.TODO())
	if err != nil {
		return nil, convertDBError("list connectors: %w", err)
	}

	storageConnectors := make([]storage.Connector, 0, len(connectors))
	for _, c := range connectors {
		storageConnectors = append(storageConnectors, toStorageConnector(c))
	}
	return storageConnectors, nil
}

// GetConnector extracts a connector from the database by id.
func (d *Database) GetConnector(id string) (storage.Connector, error) {
	connector, err := d.client.Connector.Get(context.TODO(), id)
	if err != nil {
		return storage.Connector{}, convertDBError("get connector: %w", err)
	}
	return toStorageConnector(connector), nil
}

// DeleteConnector deletes a connector from the database by id.
func (d *Database) DeleteConnector(id string) error {
	err := d.client.Connector.DeleteOneID(id).Exec(context.TODO())
	if err != nil {
		return convertDBError("delete connector: %w", err)
	}
	return nil
}

// UpdateConnector changes a connector by id using an updater function and saves it to the database.
func (d *Database) UpdateConnector(id string, updater func(old storage.Connector) (storage.Connector, error)) error {
	tx, err := d.BeginTx(context.TODO())
	if err != nil {
		return convertDBError("update connector tx: %w", err)
	}

	connector, err := tx.Connector.Get(context.TODO(), id)
	if err != nil {
		return rollback(tx, "update connector database: %w", err)
	}

	newConnector, err := updater(toStorageConnector(connector))
	if err != nil {
		return rollback(tx, "update connector updating: %w", err)
	}

	_, err = tx.Connector.UpdateOneID(newConnector.ID).
		SetName(newConnector.Name).
		SetType(newConnector.Type).
		SetResourceVersion(newConnector.ResourceVersion).
		SetConfig(newConnector.Config).
		Save(context.TODO())
	if err != nil {
		return rollback(tx, "update connector uploading: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return rollback(tx, "update connector commit: %w", err)
	}

	return nil
}
