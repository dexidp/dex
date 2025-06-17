// Package memory provides an in memory implementation of the storage interface.
package memory

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/dexidp/dex/storage"
)

var _ storage.Storage = (*memStorage)(nil)

// New returns an in memory storage.
func New(logger *slog.Logger) storage.Storage {
	return &memStorage{
		clients:         make(map[string]storage.Client),
		authCodes:       make(map[string]storage.AuthCode),
		refreshTokens:   make(map[string]storage.RefreshToken),
		authReqs:        make(map[string]storage.AuthRequest),
		passwords:       make(map[string]storage.Password),
		offlineSessions: make(map[offlineSessionID]storage.OfflineSessions),
		connectors:      make(map[string]storage.Connector),
		deviceRequests:  make(map[string]storage.DeviceRequest),
		deviceTokens:    make(map[string]storage.DeviceToken),
		logger:          logger,
	}
}

// Config is an implementation of a storage configuration.
//
// TODO(ericchiang): Actually define a storage config interface and have registration.
type Config struct { // The in memory implementation has no config.
}

// Open always returns a new in memory storage.
func (c *Config) Open(logger *slog.Logger) (storage.Storage, error) {
	return New(logger), nil
}

type memStorage struct {
	mu sync.Mutex

	clients         map[string]storage.Client
	authCodes       map[string]storage.AuthCode
	refreshTokens   map[string]storage.RefreshToken
	authReqs        map[string]storage.AuthRequest
	passwords       map[string]storage.Password
	offlineSessions map[offlineSessionID]storage.OfflineSessions
	connectors      map[string]storage.Connector
	deviceRequests  map[string]storage.DeviceRequest
	deviceTokens    map[string]storage.DeviceToken

	keys storage.Keys

	logger *slog.Logger
}

type offlineSessionID struct {
	userID string
	connID string
}

func (s *memStorage) tx(f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f()
}

func (s *memStorage) Close() error { return nil }

func (s *memStorage) GarbageCollect(ctx context.Context, now time.Time) (result storage.GCResult, err error) {
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
		for id, a := range s.deviceRequests {
			if now.After(a.Expiry) {
				delete(s.deviceRequests, id)
				result.DeviceRequests++
			}
		}
		for id, a := range s.deviceTokens {
			if now.After(a.Expiry) {
				delete(s.deviceTokens, id)
				result.DeviceTokens++
			}
		}
	})
	return result, nil
}

func (s *memStorage) CreateClient(ctx context.Context, c storage.Client) (err error) {
	s.tx(func() {
		if _, ok := s.clients[c.ID]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.clients[c.ID] = c
		}
	})
	return
}

func (s *memStorage) CreateAuthCode(ctx context.Context, c storage.AuthCode) (err error) {
	s.tx(func() {
		if _, ok := s.authCodes[c.ID]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.authCodes[c.ID] = c
		}
	})
	return
}

func (s *memStorage) CreateRefresh(ctx context.Context, r storage.RefreshToken) (err error) {
	s.tx(func() {
		if _, ok := s.refreshTokens[r.ID]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.refreshTokens[r.ID] = r
		}
	})
	return
}

func (s *memStorage) CreateAuthRequest(ctx context.Context, a storage.AuthRequest) (err error) {
	s.tx(func() {
		if _, ok := s.authReqs[a.ID]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.authReqs[a.ID] = a
		}
	})
	return
}

func (s *memStorage) CreatePassword(ctx context.Context, p storage.Password) (err error) {
	lowerEmail := strings.ToLower(p.Email)
	s.tx(func() {
		if _, ok := s.passwords[lowerEmail]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.passwords[lowerEmail] = p
		}
	})
	return
}

func (s *memStorage) CreateOfflineSessions(ctx context.Context, o storage.OfflineSessions) (err error) {
	id := offlineSessionID{
		userID: o.UserID,
		connID: o.ConnID,
	}
	s.tx(func() {
		if _, ok := s.offlineSessions[id]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.offlineSessions[id] = o
		}
	})
	return
}

func (s *memStorage) CreateConnector(ctx context.Context, connector storage.Connector) (err error) {
	s.tx(func() {
		if _, ok := s.connectors[connector.ID]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.connectors[connector.ID] = connector
		}
	})
	return
}

func (s *memStorage) GetAuthCode(ctx context.Context, id string) (c storage.AuthCode, err error) {
	s.tx(func() {
		var ok bool
		if c, ok = s.authCodes[id]; !ok {
			err = storage.ErrNotFound
			return
		}
	})
	return
}

func (s *memStorage) GetPassword(ctx context.Context, email string) (p storage.Password, err error) {
	email = strings.ToLower(email)
	s.tx(func() {
		var ok bool
		if p, ok = s.passwords[email]; !ok {
			err = storage.ErrNotFound
		}
	})
	return
}

func (s *memStorage) GetClient(ctx context.Context, id string) (client storage.Client, err error) {
	s.tx(func() {
		var ok bool
		if client, ok = s.clients[id]; !ok {
			err = storage.ErrNotFound
		}
	})
	return
}

func (s *memStorage) GetKeys(ctx context.Context) (keys storage.Keys, err error) {
	s.tx(func() { keys = s.keys })
	return
}

func (s *memStorage) GetRefresh(ctx context.Context, id string) (tok storage.RefreshToken, err error) {
	s.tx(func() {
		var ok bool
		if tok, ok = s.refreshTokens[id]; !ok {
			err = storage.ErrNotFound
			return
		}
	})
	return
}

func (s *memStorage) GetAuthRequest(ctx context.Context, id string) (req storage.AuthRequest, err error) {
	s.tx(func() {
		var ok bool
		if req, ok = s.authReqs[id]; !ok {
			err = storage.ErrNotFound
			return
		}
	})
	return
}

func (s *memStorage) GetOfflineSessions(ctx context.Context, userID string, connID string) (o storage.OfflineSessions, err error) {
	id := offlineSessionID{
		userID: userID,
		connID: connID,
	}
	s.tx(func() {
		var ok bool
		if o, ok = s.offlineSessions[id]; !ok {
			err = storage.ErrNotFound
			return
		}
	})
	return
}

func (s *memStorage) GetConnector(ctx context.Context, id string) (connector storage.Connector, err error) {
	s.tx(func() {
		var ok bool
		if connector, ok = s.connectors[id]; !ok {
			err = storage.ErrNotFound
		}
	})
	return
}

func (s *memStorage) ListClients(ctx context.Context) (clients []storage.Client, err error) {
	s.tx(func() {
		for _, client := range s.clients {
			clients = append(clients, client)
		}
	})
	return
}

func (s *memStorage) ListRefreshTokens(ctx context.Context) (tokens []storage.RefreshToken, err error) {
	s.tx(func() {
		for _, refresh := range s.refreshTokens {
			tokens = append(tokens, refresh)
		}
	})
	return
}

func (s *memStorage) ListPasswords(ctx context.Context) (passwords []storage.Password, err error) {
	s.tx(func() {
		for _, password := range s.passwords {
			passwords = append(passwords, password)
		}
	})
	return
}

func (s *memStorage) ListConnectors(ctx context.Context) (conns []storage.Connector, err error) {
	s.tx(func() {
		for _, c := range s.connectors {
			conns = append(conns, c)
		}
	})
	return
}

func (s *memStorage) DeletePassword(ctx context.Context, email string) (err error) {
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

func (s *memStorage) DeleteClient(ctx context.Context, id string) (err error) {
	s.tx(func() {
		if _, ok := s.clients[id]; !ok {
			err = storage.ErrNotFound
			return
		}
		delete(s.clients, id)
	})
	return
}

func (s *memStorage) DeleteRefresh(ctx context.Context, id string) (err error) {
	s.tx(func() {
		if _, ok := s.refreshTokens[id]; !ok {
			err = storage.ErrNotFound
			return
		}
		delete(s.refreshTokens, id)
	})
	return
}

func (s *memStorage) DeleteAuthCode(ctx context.Context, id string) (err error) {
	s.tx(func() {
		if _, ok := s.authCodes[id]; !ok {
			err = storage.ErrNotFound
			return
		}
		delete(s.authCodes, id)
	})
	return
}

func (s *memStorage) DeleteAuthRequest(ctx context.Context, id string) (err error) {
	s.tx(func() {
		if _, ok := s.authReqs[id]; !ok {
			err = storage.ErrNotFound
			return
		}
		delete(s.authReqs, id)
	})
	return
}

func (s *memStorage) DeleteOfflineSessions(ctx context.Context, userID string, connID string) (err error) {
	id := offlineSessionID{
		userID: userID,
		connID: connID,
	}
	s.tx(func() {
		if _, ok := s.offlineSessions[id]; !ok {
			err = storage.ErrNotFound
			return
		}
		delete(s.offlineSessions, id)
	})
	return
}

func (s *memStorage) DeleteConnector(ctx context.Context, id string) (err error) {
	s.tx(func() {
		if _, ok := s.connectors[id]; !ok {
			err = storage.ErrNotFound
			return
		}
		delete(s.connectors, id)
	})
	return
}

func (s *memStorage) UpdateClient(ctx context.Context, id string, updater func(old storage.Client) (storage.Client, error)) (err error) {
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

func (s *memStorage) UpdateKeys(ctx context.Context, updater func(old storage.Keys) (storage.Keys, error)) (err error) {
	s.tx(func() {
		var keys storage.Keys
		if keys, err = updater(s.keys); err == nil {
			s.keys = keys
		}
	})
	return
}

func (s *memStorage) UpdateAuthRequest(ctx context.Context, id string, updater func(old storage.AuthRequest) (storage.AuthRequest, error)) (err error) {
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

func (s *memStorage) UpdatePassword(ctx context.Context, email string, updater func(p storage.Password) (storage.Password, error)) (err error) {
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

func (s *memStorage) UpdateRefreshToken(ctx context.Context, id string, updater func(p storage.RefreshToken) (storage.RefreshToken, error)) (err error) {
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

func (s *memStorage) UpdateOfflineSessions(ctx context.Context, userID string, connID string, updater func(o storage.OfflineSessions) (storage.OfflineSessions, error)) (err error) {
	id := offlineSessionID{
		userID: userID,
		connID: connID,
	}
	s.tx(func() {
		r, ok := s.offlineSessions[id]
		if !ok {
			err = storage.ErrNotFound
			return
		}
		if r, err = updater(r); err == nil {
			s.offlineSessions[id] = r
		}
	})
	return
}

func (s *memStorage) UpdateConnector(ctx context.Context, id string, updater func(c storage.Connector) (storage.Connector, error)) (err error) {
	s.tx(func() {
		r, ok := s.connectors[id]
		if !ok {
			err = storage.ErrNotFound
			return
		}
		if r, err = updater(r); err == nil {
			s.connectors[id] = r
		}
	})
	return
}

func (s *memStorage) CreateDeviceRequest(ctx context.Context, d storage.DeviceRequest) (err error) {
	s.tx(func() {
		if _, ok := s.deviceRequests[d.UserCode]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.deviceRequests[d.UserCode] = d
		}
	})
	return
}

func (s *memStorage) GetDeviceRequest(ctx context.Context, userCode string) (req storage.DeviceRequest, err error) {
	s.tx(func() {
		var ok bool
		if req, ok = s.deviceRequests[userCode]; !ok {
			err = storage.ErrNotFound
			return
		}
	})
	return
}

func (s *memStorage) CreateDeviceToken(ctx context.Context, t storage.DeviceToken) (err error) {
	s.tx(func() {
		if _, ok := s.deviceTokens[t.DeviceCode]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.deviceTokens[t.DeviceCode] = t
		}
	})
	return
}

func (s *memStorage) GetDeviceToken(ctx context.Context, deviceCode string) (t storage.DeviceToken, err error) {
	s.tx(func() {
		var ok bool
		if t, ok = s.deviceTokens[deviceCode]; !ok {
			err = storage.ErrNotFound
			return
		}
	})
	return
}

func (s *memStorage) UpdateDeviceToken(ctx context.Context, deviceCode string, updater func(p storage.DeviceToken) (storage.DeviceToken, error)) (err error) {
	s.tx(func() {
		r, ok := s.deviceTokens[deviceCode]
		if !ok {
			err = storage.ErrNotFound
			return
		}
		if r, err = updater(r); err == nil {
			s.deviceTokens[deviceCode] = r
		}
	})
	return
}
