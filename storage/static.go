package storage

import (
	"context"
	"errors"
	"log/slog"
	"strings"
)

// Tests for this code are in the "memory" package, since this package doesn't
// define a concrete storage implementation.

// staticClientsStorage is a storage that only allow read-only actions on clients.
// All read actions return from the list of clients stored in memory, not the
// underlying
type staticClientsStorage struct {
	Storage

	// A read-only set of clients.
	clients     []Client
	clientsByID map[string]Client
}

// WithStaticClients adds a read-only set of clients to the underlying storages.
func WithStaticClients(s Storage, staticClients []Client) Storage {
	clientsByID := make(map[string]Client, len(staticClients))
	for _, client := range staticClients {
		clientsByID[client.ID] = client
	}

	return staticClientsStorage{s, staticClients, clientsByID}
}

func (s staticClientsStorage) GetClient(ctx context.Context, id string) (Client, error) {
	if client, ok := s.clientsByID[id]; ok {
		return client, nil
	}
	return s.Storage.GetClient(ctx, id)
}

func (s staticClientsStorage) isStatic(id string) bool {
	_, ok := s.clientsByID[id]
	return ok
}

func (s staticClientsStorage) ListClients(ctx context.Context) ([]Client, error) {
	clients, err := s.Storage.ListClients(ctx)
	if err != nil {
		return nil, err
	}
	n := 0
	for _, client := range clients {
		// If a client in the backing storage has the same ID as a static client
		// prefer the static client.
		if !s.isStatic(client.ID) {
			clients[n] = client
			n++
		}
	}
	return append(clients[:n], s.clients...), nil
}

func (s staticClientsStorage) CreateClient(ctx context.Context, c Client) error {
	if s.isStatic(c.ID) {
		return errors.New("static clients: read-only cannot create client")
	}
	return s.Storage.CreateClient(ctx, c)
}

func (s staticClientsStorage) DeleteClient(ctx context.Context, id string) error {
	if s.isStatic(id) {
		return errors.New("static clients: read-only cannot delete client")
	}
	return s.Storage.DeleteClient(ctx, id)
}

func (s staticClientsStorage) UpdateClient(ctx context.Context, id string, updater func(old Client) (Client, error)) error {
	if s.isStatic(id) {
		return errors.New("static clients: read-only cannot update client")
	}
	return s.Storage.UpdateClient(ctx, id, updater)
}

type staticPasswordsStorage struct {
	Storage

	// A read-only set of passwords.
	passwords []Password
	// A map of passwords that is indexed by lower-case email ids
	passwordsByEmail map[string]Password

	logger *slog.Logger
}

// WithStaticPasswords returns a storage with a read-only set of passwords.
func WithStaticPasswords(s Storage, staticPasswords []Password, logger *slog.Logger) Storage {
	passwordsByEmail := make(map[string]Password, len(staticPasswords))
	for _, p := range staticPasswords {
		// Enable case insensitive email comparison.
		lowerEmail := strings.ToLower(p.Email)
		if _, ok := passwordsByEmail[lowerEmail]; ok {
			logger.Error("attempting to create StaticPasswords with the same email id", "email", p.Email)
		}
		passwordsByEmail[lowerEmail] = p
	}

	return staticPasswordsStorage{s, staticPasswords, passwordsByEmail, logger}
}

func (s staticPasswordsStorage) isStatic(email string) bool {
	_, ok := s.passwordsByEmail[strings.ToLower(email)]
	return ok
}

func (s staticPasswordsStorage) GetPassword(ctx context.Context, email string) (Password, error) {
	// TODO(ericchiang): BLAH. We really need to figure out how to handle
	// lower cased emails better.
	email = strings.ToLower(email)
	if password, ok := s.passwordsByEmail[email]; ok {
		return password, nil
	}
	return s.Storage.GetPassword(ctx, email)
}

func (s staticPasswordsStorage) ListPasswords(ctx context.Context) ([]Password, error) {
	passwords, err := s.Storage.ListPasswords(ctx)
	if err != nil {
		return nil, err
	}

	n := 0
	for _, password := range passwords {
		// If an entry has the same email as those provided in the static
		// values, prefer the static value.
		if !s.isStatic(password.Email) {
			passwords[n] = password
			n++
		}
	}
	return append(passwords[:n], s.passwords...), nil
}

func (s staticPasswordsStorage) CreatePassword(ctx context.Context, p Password) error {
	if s.isStatic(p.Email) {
		return errors.New("static passwords: read-only cannot create password")
	}
	return s.Storage.CreatePassword(ctx, p)
}

func (s staticPasswordsStorage) DeletePassword(ctx context.Context, email string) error {
	if s.isStatic(email) {
		return errors.New("static passwords: read-only cannot delete password")
	}
	return s.Storage.DeletePassword(ctx, email)
}

func (s staticPasswordsStorage) UpdatePassword(ctx context.Context, email string, updater func(old Password) (Password, error)) error {
	if s.isStatic(email) {
		return errors.New("static passwords: read-only cannot update password")
	}
	return s.Storage.UpdatePassword(ctx, email, updater)
}

// staticConnectorsStorage represents a storage with read-only set of connectors.
type staticConnectorsStorage struct {
	Storage

	// A read-only set of connectors.
	connectors     []Connector
	connectorsByID map[string]Connector
}

// WithStaticConnectors returns a storage with a read-only set of Connectors. Write actions,
// such as updating existing Connectors, will fail.
func WithStaticConnectors(s Storage, staticConnectors []Connector) Storage {
	connectorsByID := make(map[string]Connector, len(staticConnectors))
	for _, c := range staticConnectors {
		connectorsByID[c.ID] = c
	}
	return staticConnectorsStorage{s, staticConnectors, connectorsByID}
}

func (s staticConnectorsStorage) isStatic(id string) bool {
	_, ok := s.connectorsByID[id]
	return ok
}

func (s staticConnectorsStorage) GetConnector(ctx context.Context, id string) (Connector, error) {
	if connector, ok := s.connectorsByID[id]; ok {
		return connector, nil
	}
	return s.Storage.GetConnector(ctx, id)
}

func (s staticConnectorsStorage) ListConnectors(ctx context.Context) ([]Connector, error) {
	connectors, err := s.Storage.ListConnectors(ctx)
	if err != nil {
		return nil, err
	}

	n := 0
	for _, connector := range connectors {
		// If an entry has the same id as those provided in the static
		// values, prefer the static value.
		if !s.isStatic(connector.ID) {
			connectors[n] = connector
			n++
		}
	}
	return append(connectors[:n], s.connectors...), nil
}

func (s staticConnectorsStorage) CreateConnector(ctx context.Context, c Connector) error {
	if s.isStatic(c.ID) {
		return errors.New("static connectors: read-only cannot create connector")
	}
	return s.Storage.CreateConnector(ctx, c)
}

func (s staticConnectorsStorage) DeleteConnector(ctx context.Context, id string) error {
	if s.isStatic(id) {
		return errors.New("static connectors: read-only cannot delete connector")
	}
	return s.Storage.DeleteConnector(ctx, id)
}

func (s staticConnectorsStorage) UpdateConnector(ctx context.Context, id string, updater func(old Connector) (Connector, error)) error {
	if s.isStatic(id) {
		return errors.New("static connectors: read-only cannot update connector")
	}
	return s.Storage.UpdateConnector(ctx, id, updater)
}
