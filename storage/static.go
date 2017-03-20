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

// WithStaticClients adds a read-only set of clients to the underlying storages.
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
	return s.Storage.GetClient(id)
}

func (s staticClientsStorage) isStatic(id string) bool {
	_, ok := s.clientsByID[id]
	return ok
}

func (s staticClientsStorage) ListClients() ([]Client, error) {
	clients, err := s.Storage.ListClients()
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

func (s staticClientsStorage) CreateClient(c Client) error {
	if s.isStatic(c.ID) {
		return errors.New("static clients: read-only cannot create client")
	}
	return s.Storage.CreateClient(c)
}

func (s staticClientsStorage) DeleteClient(id string) error {
	if s.isStatic(id) {
		return errors.New("static clients: read-only cannot delete client")
	}
	return s.Storage.DeleteClient(id)
}

func (s staticClientsStorage) UpdateClient(id string, updater func(old Client) (Client, error)) error {
	if s.isStatic(id) {
		return errors.New("static clients: read-only cannot update client")
	}
	return s.Storage.UpdateClient(id, updater)
}

type staticPasswordsStorage struct {
	Storage

	// A read-only set of passwords.
	passwords        []Password
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
	return staticPasswordsStorage{s, staticPasswords, passwordsByEmail}
}

func (s staticPasswordsStorage) isStatic(email string) bool {
	_, ok := s.passwordsByEmail[strings.ToLower(email)]
	return ok
}

func (s staticPasswordsStorage) GetPassword(email string) (Password, error) {
	// TODO(ericchiang): BLAH. We really need to figure out how to handle
	// lower cased emails better.
	email = strings.ToLower(email)
	if password, ok := s.passwordsByEmail[email]; ok {
		return password, nil
	}
	return s.Storage.GetPassword(email)
}

func (s staticPasswordsStorage) ListPasswords() ([]Password, error) {
	passwords, err := s.Storage.ListPasswords()
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

func (s staticPasswordsStorage) CreatePassword(p Password) error {
	if s.isStatic(p.Email) {
		return errors.New("static passwords: read-only cannot create password")
	}
	return s.Storage.CreatePassword(p)
}

func (s staticPasswordsStorage) DeletePassword(email string) error {
	if s.isStatic(email) {
		return errors.New("static passwords: read-only cannot create password")
	}
	return s.Storage.DeletePassword(email)
}

func (s staticPasswordsStorage) UpdatePassword(email string, updater func(old Password) (Password, error)) error {
	if s.isStatic(email) {
		return errors.New("static passwords: read-only cannot update password")
	}
	return s.Storage.UpdatePassword(email, updater)
}
