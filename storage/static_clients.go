package storage

import "errors"

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

// WithStaticClients returns a storage with a read-only set of clients. Write actions,
// such as creating other clients, will fail.
//
// In the future the returned storage may allow creating and storing additional clients
// in the underlying storage.
func WithStaticClients(s Storage, staticClients []Client) Storage {
	clientsByID := make(map[string]Client, len(staticClients))
	for _, client := range staticClients {
		clientsByID[client.ID] = client
	}
	return staticClientsStorage{s, staticClients, clientsByID}
}

func (s staticClientsStorage) GetClient(id string) (Client, error) {
	if client, ok := s.clientsByID[id]; ok {
		return client, nil
	}
	return Client{}, ErrNotFound
}

func (s staticClientsStorage) ListClients() ([]Client, error) {
	clients := make([]Client, len(s.clients))
	copy(clients, s.clients)
	return clients, nil
}

func (s staticClientsStorage) CreateClient(c Client) error {
	return errors.New("static clients: read-only cannot create client")
}

func (s staticClientsStorage) DeleteClient(id string) error {
	return errors.New("static clients: read-only cannot delete client")
}

func (s staticClientsStorage) UpdateClient(id string, updater func(old Client) (Client, error)) error {
	return errors.New("static clients: read-only cannot update client")
}
