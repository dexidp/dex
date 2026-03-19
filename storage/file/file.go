// Package file provides a file-based JSON implementation of the storage interface.
package file

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dexidp/dex/storage"
)

var _ storage.Storage = (*fileStorage)(nil)

// Config holds the configuration for file-based storage.
type Config struct {
	DataDir string `json:"dataDir"`
}

// Open creates a new file-based storage backend.
func (c *Config) Open(logger *slog.Logger) (storage.Storage, error) {
	if c.DataDir == "" {
		return nil, errors.New("dataDir must be specified")
	}

	if err := os.MkdirAll(c.DataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	s := &fileStorage{
		dataDir: c.DataDir,
		logger:  logger,
	}

	subdirs := []string{
		"passwords",
		"clients",
		"auth-requests",
		"auth-codes",
		"refresh-tokens",
		"offline-sessions",
		"connectors",
		"device-requests",
		"device-tokens",
		"keys",
	}

	for _, subdir := range subdirs {
		path := filepath.Join(c.DataDir, subdir)
		if err := os.MkdirAll(path, 0700); err != nil {
			return nil, fmt.Errorf("failed to create %s directory: %w", subdir, err)
		}
	}

	logger.Info("file storage initialized", "dataDir", c.DataDir)

	return s, nil
}

type fileStorage struct {
	dataDir string
	mu      sync.RWMutex
	logger  *slog.Logger
}

// Helper functions

func (s *fileStorage) writeFile(dir, filename string, data interface{}) error {
	path := filepath.Join(s.dataDir, dir, filename)

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(path, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (s *fileStorage) readFile(dir, filename string, data interface{}) error {
	path := filepath.Join(s.dataDir, dir, filename)

	jsonData, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return storage.ErrNotFound
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	if err := json.Unmarshal(jsonData, data); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

func (s *fileStorage) deleteFile(dir, filename string) error {
	path := filepath.Join(s.dataDir, dir, filename)

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return storage.ErrNotFound
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (s *fileStorage) fileExists(dir, filename string) bool {
	path := filepath.Join(s.dataDir, dir, filename)
	_, err := os.Stat(path)
	return err == nil
}

// Close implements storage.Storage.
func (s *fileStorage) Close() error {
	return nil
}

// GarbageCollect implements storage.Storage.
func (s *fileStorage) GarbageCollect(ctx context.Context, now time.Time) (result storage.GCResult, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clean up expired auth codes
	if files, err := os.ReadDir(filepath.Join(s.dataDir, "auth-codes")); err == nil {
		for _, file := range files {
			var authCode storage.AuthCode
			if err := s.readFile("auth-codes", file.Name(), &authCode); err == nil {
				if now.After(authCode.Expiry) {
					s.deleteFile("auth-codes", file.Name())
					result.AuthCodes++
				}
			}
		}
	}

	// Clean up expired auth requests
	if files, err := os.ReadDir(filepath.Join(s.dataDir, "auth-requests")); err == nil {
		for _, file := range files {
			var authReq storage.AuthRequest
			if err := s.readFile("auth-requests", file.Name(), &authReq); err == nil {
				if now.After(authReq.Expiry) {
					s.deleteFile("auth-requests", file.Name())
					result.AuthRequests++
				}
			}
		}
	}

	// Clean up expired device requests
	if files, err := os.ReadDir(filepath.Join(s.dataDir, "device-requests")); err == nil {
		for _, file := range files {
			var deviceReq storage.DeviceRequest
			if err := s.readFile("device-requests", file.Name(), &deviceReq); err == nil {
				if now.After(deviceReq.Expiry) {
					s.deleteFile("device-requests", file.Name())
					result.DeviceRequests++
				}
			}
		}
	}

	// Clean up expired device tokens
	if files, err := os.ReadDir(filepath.Join(s.dataDir, "device-tokens")); err == nil {
		for _, file := range files {
			var deviceToken storage.DeviceToken
			if err := s.readFile("device-tokens", file.Name(), &deviceToken); err == nil {
				if now.After(deviceToken.Expiry) {
					s.deleteFile("device-tokens", file.Name())
					result.DeviceTokens++
				}
			}
		}
	}

	return result, nil
}

// Password operations

func (s *fileStorage) CreatePassword(ctx context.Context, p storage.Password) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s.json", p.UserID)

	if s.fileExists("passwords", filename) {
		return storage.ErrAlreadyExists
	}

	return s.writeFile("passwords", filename, p)
}

func (s *fileStorage) GetPassword(ctx context.Context, email string) (storage.Password, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.getPasswordLocked(email)
}

func (s *fileStorage) getPasswordLocked(email string) (storage.Password, error) {
	lowerEmail := strings.ToLower(email)

	files, err := os.ReadDir(filepath.Join(s.dataDir, "passwords"))
	if err != nil {
		return storage.Password{}, storage.ErrNotFound
	}

	for _, file := range files {
		var p storage.Password
		if err := s.readFile("passwords", file.Name(), &p); err != nil {
			continue
		}
		if strings.ToLower(p.Email) == lowerEmail {
			return p, nil
		}
	}

	return storage.Password{}, storage.ErrNotFound
}

func (s *fileStorage) UpdatePassword(ctx context.Context, email string, updater func(p storage.Password) (storage.Password, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := s.getPasswordWriteLocked(email)
	if err != nil {
		return err
	}

	updated, err := updater(p)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s.json", updated.UserID)
	return s.writeFile("passwords", filename, updated)
}

func (s *fileStorage) getPasswordWriteLocked(email string) (storage.Password, error) {
	lowerEmail := strings.ToLower(email)
	files, err := os.ReadDir(filepath.Join(s.dataDir, "passwords"))
	if err != nil {
		return storage.Password{}, storage.ErrNotFound
	}

	for _, file := range files {
		var p storage.Password
		if err := s.readFile("passwords", file.Name(), &p); err != nil {
			continue
		}
		if strings.ToLower(p.Email) == lowerEmail {
			return p, nil
		}
	}

	return storage.Password{}, storage.ErrNotFound
}

func (s *fileStorage) DeletePassword(ctx context.Context, email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := s.getPasswordWriteLocked(email)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s.json", p.UserID)
	return s.deleteFile("passwords", filename)
}

func (s *fileStorage) ListPasswords(ctx context.Context) ([]storage.Password, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := os.ReadDir(filepath.Join(s.dataDir, "passwords"))
	if err != nil {
		return nil, err
	}

	var passwords []storage.Password
	for _, file := range files {
		var p storage.Password
		if err := s.readFile("passwords", file.Name(), &p); err != nil {
			s.logger.Warn("failed to read password file", "file", file.Name(), "error", err)
			continue
		}
		passwords = append(passwords, p)
	}

	return passwords, nil
}

// Client operations

func (s *fileStorage) CreateClient(ctx context.Context, c storage.Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s.json", c.ID)
	if s.fileExists("clients", filename) {
		return storage.ErrAlreadyExists
	}

	return s.writeFile("clients", filename, c)
}

func (s *fileStorage) GetClient(ctx context.Context, id string) (storage.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var c storage.Client
	filename := fmt.Sprintf("%s.json", id)
	if err := s.readFile("clients", filename, &c); err != nil {
		return storage.Client{}, err
	}

	return c, nil
}

func (s *fileStorage) UpdateClient(ctx context.Context, id string, updater func(old storage.Client) (storage.Client, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var c storage.Client
	filename := fmt.Sprintf("%s.json", id)
	if err := s.readFile("clients", filename, &c); err != nil {
		return err
	}

	updated, err := updater(c)
	if err != nil {
		return err
	}

	return s.writeFile("clients", filename, updated)
}

func (s *fileStorage) DeleteClient(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s.json", id)
	return s.deleteFile("clients", filename)
}

func (s *fileStorage) ListClients(ctx context.Context) ([]storage.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := os.ReadDir(filepath.Join(s.dataDir, "clients"))
	if err != nil {
		return nil, err
	}

	var clients []storage.Client
	for _, file := range files {
		var c storage.Client
		if err := s.readFile("clients", file.Name(), &c); err != nil {
			s.logger.Warn("failed to read client file", "file", file.Name(), "error", err)
			continue
		}
		clients = append(clients, c)
	}

	return clients, nil
}
