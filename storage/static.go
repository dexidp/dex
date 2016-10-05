package storage

import (
	"errors"
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

type staticPasswordsStorage struct {
	Storage

	passwordsByEmail map[string]Password
}

// WithStaticPasswords returns a storage with a read-only set of passwords. Write actions,
// such as creating other passwords, will fail.
func WithStaticPasswords(s Storage, staticPasswords []Password) Storage {
	passwordsByEmail := make(map[string]Password, len(staticPasswords))
	for _, p := range staticPasswords {
		p.Email = strings.ToLower(p.Email)
		passwordsByEmail[p.Email] = p
	}
	return staticPasswordsStorage{s, passwordsByEmail}
}

func (s staticPasswordsStorage) GetPassword(email string) (Password, error) {
	if password, ok := s.passwordsByEmail[strings.ToLower(email)]; ok {
		return password, nil
	}
	return Password{}, ErrNotFound
}

func (s staticPasswordsStorage) CreatePassword(p Password) error {
	return errors.New("static passwords: read-only cannot create password")
}

func (s staticPasswordsStorage) DeletePassword(id string) error {
	return errors.New("static passwords: read-only cannot create password")
}

func (s staticPasswordsStorage) UpdatePassword(id string, updater func(old Password) (Password, error)) error {
	return errors.New("static passwords: read-only cannot update password")
}
