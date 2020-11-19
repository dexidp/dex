// Package memory provides an in memory implementation of the storage interface.
package memory

import (
	"strings"
	"sync"
	"time"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/storage"
)

// New returns an in memory storage.
func New(logger log.Logger) storage.Storage {
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
		user_idps:       make(map[string]storage.UserIdp),
		users:           make(map[string]storage.User),
		acl_tokens:      make(map[string]storage.AclToken),
		client_tokens:   make(map[string]storage.ClientToken),
		logger:          logger,
	}
}

// Config is an implementation of a storage configuration.
//
// TODO(ericchiang): Actually define a storage config interface and have registration.
type Config struct {
	// The in memory implementation has no config.
}

// Open always returns a new in memory storage.
func (c *Config) Open(logger log.Logger) (storage.Storage, error) {
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
	user_idps       map[string]storage.UserIdp
	users           map[string]storage.User
	acl_tokens      map[string]storage.AclToken
	client_tokens   map[string]storage.ClientToken

	keys storage.Keys

	logger log.Logger
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

func (s *memStorage) CreateOfflineSessions(o storage.OfflineSessions) (err error) {
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

func (s *memStorage) CreateConnector(connector storage.Connector) (err error) {
	s.tx(func() {
		if _, ok := s.connectors[connector.ID]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.connectors[connector.ID] = connector
		}
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

func (s *memStorage) GetRefresh(id string) (tok storage.RefreshToken, err error) {
	s.tx(func() {
		var ok bool
		if tok, ok = s.refreshTokens[id]; !ok {
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

func (s *memStorage) GetOfflineSessions(userID string, connID string) (o storage.OfflineSessions, err error) {
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

func (s *memStorage) GetConnector(id string) (connector storage.Connector, err error) {
	s.tx(func() {
		var ok bool
		if connector, ok = s.connectors[id]; !ok {
			err = storage.ErrNotFound
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

func (s *memStorage) ListConnectors() (conns []storage.Connector, err error) {
	s.tx(func() {
		for _, c := range s.connectors {
			conns = append(conns, c)
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

func (s *memStorage) DeleteRefresh(id string) (err error) {
	s.tx(func() {
		if _, ok := s.refreshTokens[id]; !ok {
			err = storage.ErrNotFound
			return
		}
		delete(s.refreshTokens, id)
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

func (s *memStorage) DeleteOfflineSessions(userID string, connID string) (err error) {
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

func (s *memStorage) DeleteConnector(id string) (err error) {
	s.tx(func() {
		if _, ok := s.connectors[id]; !ok {
			err = storage.ErrNotFound
			return
		}
		delete(s.connectors, id)
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

func (s *memStorage) UpdateOfflineSessions(userID string, connID string, updater func(o storage.OfflineSessions) (storage.OfflineSessions, error)) (err error) {
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

func (s *memStorage) UpdateConnector(id string, updater func(c storage.Connector) (storage.Connector, error)) (err error) {
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

func (s *memStorage) CreateDeviceRequest(d storage.DeviceRequest) (err error) {
	s.tx(func() {
		if _, ok := s.deviceRequests[d.UserCode]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.deviceRequests[d.UserCode] = d
		}
	})
	return
}

func (s *memStorage) GetDeviceRequest(userCode string) (req storage.DeviceRequest, err error) {
	s.tx(func() {
		var ok bool
		if req, ok = s.deviceRequests[userCode]; !ok {
			err = storage.ErrNotFound
			return
		}
	})
	return
}

func (s *memStorage) CreateDeviceToken(t storage.DeviceToken) (err error) {
	s.tx(func() {
		if _, ok := s.deviceTokens[t.DeviceCode]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.deviceTokens[t.DeviceCode] = t
		}
	})
	return
}

func (s *memStorage) GetDeviceToken(deviceCode string) (t storage.DeviceToken, err error) {
	s.tx(func() {
		var ok bool
		if t, ok = s.deviceTokens[deviceCode]; !ok {
			err = storage.ErrNotFound
			return
		}
	})
	return
}

func (s *memStorage) UpdateDeviceToken(deviceCode string, updater func(p storage.DeviceToken) (storage.DeviceToken, error)) (err error) {
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

// Functions to manage user_idp

func (s *memStorage) CreateUserIdp(u storage.UserIdp) (err error) {
	s.tx(func() {
		if _, ok := s.user_idps[u.IdpID]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.user_idps[u.IdpID] = u
		}
	})
	return
}

func (s *memStorage) GetUserIdp(id string) (user storage.UserIdp, err error) {
	s.tx(func() {
		var ok bool
		if user, ok = s.user_idps[id]; !ok {
			err = storage.ErrNotFound
		}
	})
	return
}

func (s *memStorage) ListUserIdp() (users []storage.UserIdp, err error) {
	s.tx(func() {
		for _, user := range s.user_idps {
			users = append(users, user)
		}
	})
	return
}

func (s *memStorage) DeleteUserIdp(id string) (err error) {
	s.tx(func() {
		if _, ok := s.user_idps[id]; !ok {
			err = storage.ErrNotFound
			return
		}
		delete(s.user_idps, id)
	})
	return
}

func (s *memStorage) UpdateUserIdp(id string, updater func(old storage.UserIdp) (storage.UserIdp, error)) (err error) {
	s.tx(func() {
		user, ok := s.user_idps[id]
		if !ok {
			err = storage.ErrNotFound
			return
		}
		if user, err = updater(user); err == nil {
			s.user_idps[id] = user
		}
	})
	return
}

// Functions to manage user

func (s *memStorage) CreateUser(u storage.User) (err error) {
	s.tx(func() {
		if _, ok := s.users[u.InternID]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.users[u.InternID] = u
		}
	})
	return
}

func (s *memStorage) GetUser(id string) (user storage.User, err error) {
	s.tx(func() {
		var ok bool
		if user, ok = s.users[id]; !ok {
			err = storage.ErrNotFound
		}
	})
	return
}

func (s *memStorage) ListUser() (users []storage.User, err error) {
	s.tx(func() {
		for _, user := range s.users {
			users = append(users, user)
		}
	})
	return
}

func (s *memStorage) DeleteUser(id string) (err error) {
	s.tx(func() {
		if _, ok := s.users[id]; !ok {
			err = storage.ErrNotFound
			return
		}
		delete(s.users, id)
	})
	return
}

func (s *memStorage) UpdateUser(id string, updater func(old storage.User) (storage.User, error)) (err error) {
	s.tx(func() {
		user, ok := s.users[id]
		if !ok {
			err = storage.ErrNotFound
			return
		}
		if user, err = updater(user); err == nil {
			s.users[id] = user
		}
	})
	return
}

// Functions to manage acl_token

func (s *memStorage) CreateAclToken(t storage.AclToken) (err error) {
	s.tx(func() {
		if _, ok := s.acl_tokens[t.ID]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.acl_tokens[t.ID] = t
		}
	})
	return
}

func (s *memStorage) GetAclToken(id string) (token storage.AclToken, err error) {
	s.tx(func() {
		var ok bool
		if token, ok = s.acl_tokens[id]; !ok {
			err = storage.ErrNotFound
		}
	})
	return
}

func (s *memStorage) ListAclToken() (tokens []storage.AclToken, err error) {
	s.tx(func() {
		for _, token := range s.acl_tokens {
			tokens = append(tokens, token)
		}
	})
	return
}

func (s *memStorage) DeleteAclToken(id string) (err error) {
	s.tx(func() {
		if _, ok := s.acl_tokens[id]; !ok {
			err = storage.ErrNotFound
			return
		}
		delete(s.acl_tokens, id)
	})
	return
}

func (s *memStorage) UpdateAclToken(id string, updater func(old storage.AclToken) (storage.AclToken, error)) (err error) {
	s.tx(func() {
		token, ok := s.acl_tokens[id]
		if !ok {
			err = storage.ErrNotFound
			return
		}
		if token, err = updater(token); err == nil {
			s.acl_tokens[id] = token
		}
	})
	return
}

// Functions to manage client_token

func (s *memStorage) CreateClientToken(t storage.ClientToken) (err error) {
	s.tx(func() {
		if _, ok := s.client_tokens[t.ID]; ok {
			err = storage.ErrAlreadyExists
		} else {
			s.client_tokens[t.ID] = t
		}
	})
	return
}

func (s *memStorage) GetClientToken(id string) (token storage.ClientToken, err error) {
	s.tx(func() {
		var ok bool
		if token, ok = s.client_tokens[id]; !ok {
			err = storage.ErrNotFound
		}
	})
	return
}

func (s *memStorage) ListClientToken() (tokens []storage.ClientToken, err error) {
	s.tx(func() {
		for _, token := range s.client_tokens {
			tokens = append(tokens, token)
		}
	})
	return
}

func (s *memStorage) DeleteClientToken(id string) (err error) {
	s.tx(func() {
		if _, ok := s.client_tokens[id]; !ok {
			err = storage.ErrNotFound
			return
		}
		delete(s.client_tokens, id)
	})
	return
}

func (s *memStorage) UpdateClientToken(id string, updater func(old storage.ClientToken) (storage.ClientToken, error)) (err error) {
	s.tx(func() {
		token, ok := s.client_tokens[id]
		if !ok {
			err = storage.ErrNotFound
			return
		}
		if token, err = updater(token); err == nil {
			s.client_tokens[id] = token
		}
	})
	return
}
