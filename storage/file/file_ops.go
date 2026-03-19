package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dexidp/dex/storage"
)

// AuthRequest operations

func (s *fileStorage) CreateAuthRequest(ctx context.Context, a storage.AuthRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s.json", a.ID)
	if s.fileExists("auth-requests", filename) {
		return storage.ErrAlreadyExists
	}

	return s.writeFile("auth-requests", filename, a)
}

func (s *fileStorage) GetAuthRequest(ctx context.Context, id string) (storage.AuthRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var a storage.AuthRequest
	filename := fmt.Sprintf("%s.json", id)
	if err := s.readFile("auth-requests", filename, &a); err != nil {
		return storage.AuthRequest{}, err
	}

	return a, nil
}

func (s *fileStorage) UpdateAuthRequest(ctx context.Context, id string, updater func(a storage.AuthRequest) (storage.AuthRequest, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var a storage.AuthRequest
	filename := fmt.Sprintf("%s.json", id)
	if err := s.readFile("auth-requests", filename, &a); err != nil {
		return err
	}

	updated, err := updater(a)
	if err != nil {
		return err
	}

	return s.writeFile("auth-requests", filename, updated)
}

func (s *fileStorage) DeleteAuthRequest(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s.json", id)
	return s.deleteFile("auth-requests", filename)
}

// AuthCode operations

func (s *fileStorage) CreateAuthCode(ctx context.Context, c storage.AuthCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s.json", c.ID)
	if s.fileExists("auth-codes", filename) {
		return storage.ErrAlreadyExists
	}

	return s.writeFile("auth-codes", filename, c)
}

func (s *fileStorage) GetAuthCode(ctx context.Context, id string) (storage.AuthCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var c storage.AuthCode
	filename := fmt.Sprintf("%s.json", id)
	if err := s.readFile("auth-codes", filename, &c); err != nil {
		return storage.AuthCode{}, err
	}

	return c, nil
}

func (s *fileStorage) DeleteAuthCode(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s.json", id)
	return s.deleteFile("auth-codes", filename)
}

// RefreshToken operations

func (s *fileStorage) CreateRefresh(ctx context.Context, r storage.RefreshToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s.json", r.ID)
	if s.fileExists("refresh-tokens", filename) {
		return storage.ErrAlreadyExists
	}

	return s.writeFile("refresh-tokens", filename, r)
}

func (s *fileStorage) GetRefresh(ctx context.Context, id string) (storage.RefreshToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var r storage.RefreshToken
	filename := fmt.Sprintf("%s.json", id)
	if err := s.readFile("refresh-tokens", filename, &r); err != nil {
		return storage.RefreshToken{}, err
	}

	return r, nil
}

func (s *fileStorage) UpdateRefreshToken(ctx context.Context, id string, updater func(r storage.RefreshToken) (storage.RefreshToken, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var r storage.RefreshToken
	filename := fmt.Sprintf("%s.json", id)
	if err := s.readFile("refresh-tokens", filename, &r); err != nil {
		return err
	}

	updated, err := updater(r)
	if err != nil {
		return err
	}

	return s.writeFile("refresh-tokens", filename, updated)
}

func (s *fileStorage) DeleteRefresh(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s.json", id)
	return s.deleteFile("refresh-tokens", filename)
}

func (s *fileStorage) ListRefreshTokens(ctx context.Context) ([]storage.RefreshToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := os.ReadDir(filepath.Join(s.dataDir, "refresh-tokens"))
	if err != nil {
		return nil, err
	}

	var tokens []storage.RefreshToken
	for _, file := range files {
		var r storage.RefreshToken
		if err := s.readFile("refresh-tokens", file.Name(), &r); err != nil {
			s.logger.Warn("failed to read refresh token file", "file", file.Name(), "error", err)
			continue
		}
		tokens = append(tokens, r)
	}

	return tokens, nil
}

// OfflineSessions operations

func (s *fileStorage) CreateOfflineSessions(ctx context.Context, o storage.OfflineSessions) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s-%s.json", o.UserID, o.ConnID)
	if s.fileExists("offline-sessions", filename) {
		return storage.ErrAlreadyExists
	}

	return s.writeFile("offline-sessions", filename, o)
}

func (s *fileStorage) GetOfflineSessions(ctx context.Context, userID string, connID string) (storage.OfflineSessions, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var o storage.OfflineSessions
	filename := fmt.Sprintf("%s-%s.json", userID, connID)
	if err := s.readFile("offline-sessions", filename, &o); err != nil {
		return storage.OfflineSessions{}, err
	}

	return o, nil
}

func (s *fileStorage) UpdateOfflineSessions(ctx context.Context, userID string, connID string, updater func(s storage.OfflineSessions) (storage.OfflineSessions, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var o storage.OfflineSessions
	filename := fmt.Sprintf("%s-%s.json", userID, connID)
	if err := s.readFile("offline-sessions", filename, &o); err != nil {
		return err
	}

	updated, err := updater(o)
	if err != nil {
		return err
	}

	return s.writeFile("offline-sessions", filename, updated)
}

func (s *fileStorage) DeleteOfflineSessions(ctx context.Context, userID string, connID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s-%s.json", userID, connID)
	return s.deleteFile("offline-sessions", filename)
}

// Connector operations

func (s *fileStorage) CreateConnector(ctx context.Context, c storage.Connector) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s.json", c.ID)
	if s.fileExists("connectors", filename) {
		return storage.ErrAlreadyExists
	}

	return s.writeFile("connectors", filename, c)
}

func (s *fileStorage) GetConnector(ctx context.Context, id string) (storage.Connector, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var c storage.Connector
	filename := fmt.Sprintf("%s.json", id)
	if err := s.readFile("connectors", filename, &c); err != nil {
		return storage.Connector{}, err
	}

	return c, nil
}

func (s *fileStorage) UpdateConnector(ctx context.Context, id string, updater func(c storage.Connector) (storage.Connector, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var c storage.Connector
	filename := fmt.Sprintf("%s.json", id)
	if err := s.readFile("connectors", filename, &c); err != nil {
		return err
	}

	updated, err := updater(c)
	if err != nil {
		return err
	}

	return s.writeFile("connectors", filename, updated)
}

func (s *fileStorage) DeleteConnector(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s.json", id)
	return s.deleteFile("connectors", filename)
}

func (s *fileStorage) ListConnectors(ctx context.Context) ([]storage.Connector, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := os.ReadDir(filepath.Join(s.dataDir, "connectors"))
	if err != nil {
		return nil, err
	}

	var connectors []storage.Connector
	for _, file := range files {
		var c storage.Connector
		if err := s.readFile("connectors", file.Name(), &c); err != nil {
			s.logger.Warn("failed to read connector file", "file", file.Name(), "error", err)
			continue
		}
		connectors = append(connectors, c)
	}

	return connectors, nil
}

// DeviceRequest operations

func (s *fileStorage) CreateDeviceRequest(ctx context.Context, d storage.DeviceRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s.json", d.UserCode)
	if s.fileExists("device-requests", filename) {
		return storage.ErrAlreadyExists
	}

	return s.writeFile("device-requests", filename, d)
}

func (s *fileStorage) GetDeviceRequest(ctx context.Context, userCode string) (storage.DeviceRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var d storage.DeviceRequest
	filename := fmt.Sprintf("%s.json", userCode)
	if err := s.readFile("device-requests", filename, &d); err != nil {
		return storage.DeviceRequest{}, err
	}

	return d, nil
}

// DeviceToken operations

func (s *fileStorage) CreateDeviceToken(ctx context.Context, d storage.DeviceToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s.json", d.DeviceCode)
	if s.fileExists("device-tokens", filename) {
		return storage.ErrAlreadyExists
	}

	return s.writeFile("device-tokens", filename, d)
}

func (s *fileStorage) GetDeviceToken(ctx context.Context, deviceCode string) (storage.DeviceToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var d storage.DeviceToken
	filename := fmt.Sprintf("%s.json", deviceCode)
	if err := s.readFile("device-tokens", filename, &d); err != nil {
		return storage.DeviceToken{}, err
	}

	return d, nil
}

func (s *fileStorage) UpdateDeviceToken(ctx context.Context, deviceCode string, updater func(t storage.DeviceToken) (storage.DeviceToken, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var d storage.DeviceToken
	filename := fmt.Sprintf("%s.json", deviceCode)
	if err := s.readFile("device-tokens", filename, &d); err != nil {
		return err
	}

	updated, err := updater(d)
	if err != nil {
		return err
	}

	return s.writeFile("device-tokens", filename, updated)
}

// Keys operations

func (s *fileStorage) GetKeys(ctx context.Context) (storage.Keys, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var keys storage.Keys
	filename := "signing-keys.json"
	if err := s.readFile("keys", filename, &keys); err != nil {
		if err == storage.ErrNotFound {
			return storage.Keys{}, nil
		}
		return storage.Keys{}, err
	}

	return keys, nil
}

func (s *fileStorage) UpdateKeys(ctx context.Context, updater func(old storage.Keys) (storage.Keys, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var keys storage.Keys
	filename := "signing-keys.json"
	if err := s.readFile("keys", filename, &keys); err != nil {
		if err != storage.ErrNotFound {
			return err
		}
	}

	updated, err := updater(keys)
	if err != nil {
		return err
	}

	return s.writeFile("keys", filename, updated)
}
