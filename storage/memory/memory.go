// Package memory provides an in memory implementation of the storage interface.
package memory

import (
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/dex/storage"
)

// New returns an in memory storage.
func New(logger logrus.FieldLogger) storage.Storage {
	return &memStorage{
		clients:       make(map[string]storage.Client),
		authCodes:     make(map[string]storage.AuthCode),
		refreshTokens: make(map[string]storage.RefreshToken),
		authReqs:      make(map[string]storage.AuthRequest),
		passwords:     make(map[string]storage.Password),
		logger:        logger,
	}
}

// Config is an implementation of a storage configuration.
//
// TODO(ericchiang): Actually define a storage config interface and have registration.
type Config struct {
	// The in memory implementation has no config.
}

// Open always returns a new in memory storage.
func (c *Config) Open(logger logrus.FieldLogger) (storage.Storage, error) {
	return New(logger), nil
}

type memStorage struct {
	mu sync.Mutex

	clients       map[string]storage.Client
	authCodes     map[string]storage.AuthCode
	refreshTokens map[string]storage.RefreshToken
	authReqs      map[string]storage.AuthRequest
	passwords     map[string]storage.Password

	keys storage.Keys

	logger logrus.FieldLogger
}

func (s *memStorage) tx(f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f()
}

func (s *memStorage) Close() error { return nil }

func (s *memStorage) GarbageCollect(now time.Time) (result storage.GCResult, err error) {
	s.tx(func() {
		for id, a := range s.authCodes {
			if now.After(a.Expiry) {
				delete(s.authCodes, id)
				result.AuthCodes++
			}
		}
		for id, a := range s.authReqs {
			if now.After(a.Expiry) {
				delete(s.authReqs, id)
				result.AuthRequests++
			}
		}
	})
	return result, nil
}

func (s *memStorage) CreateClient(c storage.Client) (err error) {
	s.tx(func() {
		if _, ok := s.clients[c.ID]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.clients[c.ID] = c
		}
	})
	return
}

func (s *memStorage) CreateAuthCode(c storage.AuthCode) (err error) {
	s.tx(func() {
		if _, ok := s.authCodes[c.ID]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.authCodes[c.ID] = c
		}
	})
	return
}

func (s *memStorage) CreateRefresh(r storage.RefreshToken) (err error) {
	s.tx(func() {
		if _, ok := s.refreshTokens[r.ID]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.refreshTokens[r.ID] = r
		}
	})
	return
}

func (s *memStorage) CreateAuthRequest(a storage.AuthRequest) (err error) {
	s.tx(func() {
		if _, ok := s.authReqs[a.ID]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.authReqs[a.ID] = a
		}
	})
	return
}

func (s *memStorage) CreatePassword(p storage.Password) (err error) {
	p.Email = strings.ToLower(p.Email)
	s.tx(func() {
		if _, ok := s.passwords[p.Email]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.passwords[p.Email] = p
		}
	})
	return
}

func (s *memStorage) GetPassword(email string) (p storage.Password, err error) {
	email = strings.ToLower(email)
	s.tx(func() {
		var ok bool
		if p, ok = s.passwords[email]; !ok {
			err = storage.ErrNotFound
		}
	})
	return
}

func (s *memStorage) GetClient(id string) (client storage.Client, err error) {
	s.tx(func() {
		var ok bool
		if client, ok = s.clients[id]; !ok {
			err = storage.ErrNotFound
		}
	})
	return
}

func (s *memStorage) GetKeys() (keys storage.Keys, err error) {
	s.tx(func() { keys = s.keys })
	return
}

func (s *memStorage) GetRefresh(token string) (tok storage.RefreshToken, err error) {
	s.tx(func() {
		var ok bool
		if tok, ok = s.refreshTokens[token]; !ok {
			err = storage.ErrNotFound
			return
		}
	})
	return
}

func (s *memStorage) GetAuthRequest(id string) (req storage.AuthRequest, err error) {
	s.tx(func() {
		var ok bool
		if req, ok = s.authReqs[id]; !ok {
			err = storage.ErrNotFound
			return
		}
	})
	return
}

func (s *memStorage) ListClients() (clients []storage.Client, err error) {
	s.tx(func() {
		for _, client := range s.clients {
			clients = append(clients, client)
		}
	})
	return
}

func (s *memStorage) ListRefreshTokens() (tokens []storage.RefreshToken, err error) {
	s.tx(func() {
		for _, refresh := range s.refreshTokens {
			tokens = append(tokens, refresh)
		}
	})
	return
}

func (s *memStorage) ListPasswords() (passwords []storage.Password, err error) {
	s.tx(func() {
		for _, password := range s.passwords {
			passwords = append(passwords, password)
		}
	})
	return
}

func (s *memStorage) DeletePassword(email string) (err error) {
	email = strings.ToLower(email)
	s.tx(func() {
		if _, ok := s.passwords[email]; !ok {
			err = storage.ErrNotFound
			return
		}
		delete(s.passwords, email)
	})
	return
}

func (s *memStorage) DeleteClient(id string) (err error) {
	s.tx(func() {
		if _, ok := s.clients[id]; !ok {
			err = storage.ErrNotFound
			return
		}
		delete(s.clients, id)
	})
	return
}

func (s *memStorage) DeleteRefresh(token string) (err error) {
	s.tx(func() {
		if _, ok := s.refreshTokens[token]; !ok {
			err = storage.ErrNotFound
			return
		}
		delete(s.refreshTokens, token)
	})
	return
}

func (s *memStorage) DeleteAuthCode(id string) (err error) {
	s.tx(func() {
		if _, ok := s.authCodes[id]; !ok {
			err = storage.ErrNotFound
			return
		}
		delete(s.authCodes, id)
	})
	return
}

func (s *memStorage) DeleteAuthRequest(id string) (err error) {
	s.tx(func() {
		if _, ok := s.authReqs[id]; !ok {
			err = storage.ErrNotFound
			return
		}
		delete(s.authReqs, id)
	})
	return
}

func (s *memStorage) GetAuthCode(id string) (c storage.AuthCode, err error) {
	s.tx(func() {
		var ok bool
		if c, ok = s.authCodes[id]; !ok {
			err = storage.ErrNotFound
			return
		}
	})
	return
}

func (s *memStorage) UpdateClient(id string, updater func(old storage.Client) (storage.Client, error)) (err error) {
	s.tx(func() {
		client, ok := s.clients[id]
		if !ok {
			err = storage.ErrNotFound
			return
		}
		if client, err = updater(client); err == nil {
			s.clients[id] = client
		}
	})
	return
}

func (s *memStorage) UpdateKeys(updater func(old storage.Keys) (storage.Keys, error)) (err error) {
	s.tx(func() {
		var keys storage.Keys
		if keys, err = updater(s.keys); err == nil {
			s.keys = keys
		}
	})
	return
}

func (s *memStorage) UpdateAuthRequest(id string, updater func(old storage.AuthRequest) (storage.AuthRequest, error)) (err error) {
	s.tx(func() {
		req, ok := s.authReqs[id]
		if !ok {
			err = storage.ErrNotFound
			return
		}
		if req, err = updater(req); err == nil {
			s.authReqs[id] = req
		}
	})
	return
}

func (s *memStorage) UpdatePassword(email string, updater func(p storage.Password) (storage.Password, error)) (err error) {
	email = strings.ToLower(email)
	s.tx(func() {
		req, ok := s.passwords[email]
		if !ok {
			err = storage.ErrNotFound
			return
		}
		if req, err = updater(req); err == nil {
			s.passwords[email] = req
		}
	})
	return
}

func (s *memStorage) UpdateRefreshToken(id string, updater func(p storage.RefreshToken) (storage.RefreshToken, error)) (err error) {
	s.tx(func() {
		r, ok := s.refreshTokens[id]
		if !ok {
			err = storage.ErrNotFound
			return
		}
		if r, err = updater(r); err == nil {
			s.refreshTokens[id] = r
		}
	})
	return
}
