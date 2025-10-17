package rememberme

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha3"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/internal/jwt"
	"github.com/dexidp/dex/storage"
	"github.com/go-jose/go-jose/v4"
)

var _ storage.Storage = (*mockStorage)(nil)
var _ storage.ActiveSessionStorage = (*mockStorage)(nil)

// mockStorage implements storage.Storage for testing key retrieval.
type mockStorage struct {
	keys storage.Keys
	err  error
}

// CreateSession implements storage.ActiveSessionStorage.
func (m *mockStorage) CreateSession(ctx context.Context, identifier string, data storage.ActiveSession) error {
	panic("unimplemented")
}

// GetSession implements storage.ActiveSessionStorage.
func (m *mockStorage) GetSession(ctx context.Context, identifier string) (storage.ActiveSession, error) {
	panic("unimplemented")
}

// CreateAuthCode implements storage.Storage.
func (m *mockStorage) CreateAuthCode(ctx context.Context, c storage.AuthCode) error {
	panic("unimplemented")
}

// CreateAuthRequest implements storage.Storage.
func (m *mockStorage) CreateAuthRequest(ctx context.Context, a storage.AuthRequest) error {
	panic("unimplemented")
}

// CreateClient implements storage.Storage.
func (m *mockStorage) CreateClient(ctx context.Context, c storage.Client) error {
	panic("unimplemented")
}

// CreateConnector implements storage.Storage.
func (m *mockStorage) CreateConnector(ctx context.Context, c storage.Connector) error {
	panic("unimplemented")
}

// CreateDeviceRequest implements storage.Storage.
func (m *mockStorage) CreateDeviceRequest(ctx context.Context, d storage.DeviceRequest) error {
	panic("unimplemented")
}

// CreateDeviceToken implements storage.Storage.
func (m *mockStorage) CreateDeviceToken(ctx context.Context, d storage.DeviceToken) error {
	panic("unimplemented")
}

// CreateOfflineSessions implements storage.Storage.
func (m *mockStorage) CreateOfflineSessions(ctx context.Context, s storage.OfflineSessions) error {
	panic("unimplemented")
}

// CreatePassword implements storage.Storage.
func (m *mockStorage) CreatePassword(ctx context.Context, p storage.Password) error {
	panic("unimplemented")
}

// CreateRefresh implements storage.Storage.
func (m *mockStorage) CreateRefresh(ctx context.Context, r storage.RefreshToken) error {
	panic("unimplemented")
}

// DeleteAuthCode implements storage.Storage.
func (m *mockStorage) DeleteAuthCode(ctx context.Context, code string) error {
	panic("unimplemented")
}

// DeleteAuthRequest implements storage.Storage.
func (m *mockStorage) DeleteAuthRequest(ctx context.Context, id string) error {
	panic("unimplemented")
}

// DeleteClient implements storage.Storage.
func (m *mockStorage) DeleteClient(ctx context.Context, id string) error {
	panic("unimplemented")
}

// DeleteConnector implements storage.Storage.
func (m *mockStorage) DeleteConnector(ctx context.Context, id string) error {
	panic("unimplemented")
}

// DeleteOfflineSessions implements storage.Storage.
func (m *mockStorage) DeleteOfflineSessions(ctx context.Context, userID string, connID string) error {
	panic("unimplemented")
}

// DeletePassword implements storage.Storage.
func (m *mockStorage) DeletePassword(ctx context.Context, email string) error {
	panic("unimplemented")
}

// DeleteRefresh implements storage.Storage.
func (m *mockStorage) DeleteRefresh(ctx context.Context, id string) error {
	panic("unimplemented")
}

// GarbageCollect implements storage.Storage.
func (m *mockStorage) GarbageCollect(ctx context.Context, now time.Time) (storage.GCResult, error) {
	panic("unimplemented")
}

// GetAuthCode implements storage.Storage.
func (m *mockStorage) GetAuthCode(ctx context.Context, id string) (storage.AuthCode, error) {
	panic("unimplemented")
}

// GetAuthRequest implements storage.Storage.
func (m *mockStorage) GetAuthRequest(ctx context.Context, id string) (storage.AuthRequest, error) {
	panic("unimplemented")
}

// GetClient implements storage.Storage.
func (m *mockStorage) GetClient(ctx context.Context, id string) (storage.Client, error) {
	panic("unimplemented")
}

// GetConnector implements storage.Storage.
func (m *mockStorage) GetConnector(ctx context.Context, id string) (storage.Connector, error) {
	panic("unimplemented")
}

// GetDeviceRequest implements storage.Storage.
func (m *mockStorage) GetDeviceRequest(ctx context.Context, userCode string) (storage.DeviceRequest, error) {
	panic("unimplemented")
}

// GetDeviceToken implements storage.Storage.
func (m *mockStorage) GetDeviceToken(ctx context.Context, deviceCode string) (storage.DeviceToken, error) {
	panic("unimplemented")
}

// GetOfflineSessions implements storage.Storage.
func (m *mockStorage) GetOfflineSessions(ctx context.Context, userID string, connID string) (storage.OfflineSessions, error) {
	panic("unimplemented")
}

// GetPassword implements storage.Storage.
func (m *mockStorage) GetPassword(ctx context.Context, email string) (storage.Password, error) {
	panic("unimplemented")
}

// GetRefresh implements storage.Storage.
func (m *mockStorage) GetRefresh(ctx context.Context, id string) (storage.RefreshToken, error) {
	panic("unimplemented")
}

// ListClients implements storage.Storage.
func (m *mockStorage) ListClients(ctx context.Context) ([]storage.Client, error) {
	panic("unimplemented")
}

// ListConnectors implements storage.Storage.
func (m *mockStorage) ListConnectors(ctx context.Context) ([]storage.Connector, error) {
	panic("unimplemented")
}

// ListPasswords implements storage.Storage.
func (m *mockStorage) ListPasswords(ctx context.Context) ([]storage.Password, error) {
	panic("unimplemented")
}

// ListRefreshTokens implements storage.Storage.
func (m *mockStorage) ListRefreshTokens(ctx context.Context) ([]storage.RefreshToken, error) {
	panic("unimplemented")
}

// UpdateAuthRequest implements storage.Storage.
func (m *mockStorage) UpdateAuthRequest(ctx context.Context, id string, updater func(a storage.AuthRequest) (storage.AuthRequest, error)) error {
	panic("unimplemented")
}

// UpdateClient implements storage.Storage.
func (m *mockStorage) UpdateClient(ctx context.Context, id string, updater func(old storage.Client) (storage.Client, error)) error {
	panic("unimplemented")
}

// UpdateConnector implements storage.Storage.
func (m *mockStorage) UpdateConnector(ctx context.Context, id string, updater func(c storage.Connector) (storage.Connector, error)) error {
	panic("unimplemented")
}

// UpdateDeviceToken implements storage.Storage.
func (m *mockStorage) UpdateDeviceToken(ctx context.Context, deviceCode string, updater func(t storage.DeviceToken) (storage.DeviceToken, error)) error {
	panic("unimplemented")
}

// UpdateKeys implements storage.Storage.
func (m *mockStorage) UpdateKeys(ctx context.Context, updater func(old storage.Keys) (storage.Keys, error)) error {
	panic("unimplemented")
}

// UpdateOfflineSessions implements storage.Storage.
func (m *mockStorage) UpdateOfflineSessions(ctx context.Context, userID string, connID string, updater func(s storage.OfflineSessions) (storage.OfflineSessions, error)) error {
	panic("unimplemented")
}

// UpdatePassword implements storage.Storage.
func (m *mockStorage) UpdatePassword(ctx context.Context, email string, updater func(p storage.Password) (storage.Password, error)) error {
	panic("unimplemented")
}

// UpdateRefreshToken implements storage.Storage.
func (m *mockStorage) UpdateRefreshToken(ctx context.Context, id string, updater func(r storage.RefreshToken) (storage.RefreshToken, error)) error {
	panic("unimplemented")
}

func (m *mockStorage) GetKeys(ctx context.Context) (storage.Keys, error) {
	return m.keys, m.err
}

func (m *mockStorage) Close() error { return nil }

// mockSessionStorage implements storage.ActiveSessionStorage for testing.
type mockSessionStorage struct {
	sessions  map[string]storage.ActiveSession
	getErr    error
	createErr error
	gcErr     error
}

// GarbageCollect implements storage.ActiveSessionStorage.
func (m *mockSessionStorage) GarbageCollect(ctx context.Context, now time.Time) (storage.GCResult, error) {
	panic("unimplemented")
}

func (m *mockSessionStorage) GetSession(ctx context.Context, id string) (storage.ActiveSession, error) {
	if m.getErr != nil {
		return storage.ActiveSession{}, m.getErr
	}
	session, ok := m.sessions[id]
	if !ok {
		return storage.ActiveSession{}, storage.ErrNotFound
	}
	return session, nil
}

func (m *mockSessionStorage) CreateSession(ctx context.Context, id string, session storage.ActiveSession) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.sessions[id] = session
	return nil
}

// noOpLogger is a silent logger for tests (level higher than Error to suppress output).
var noOpLogger = slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.Level(slog.LevelError) + 1}))

// fixedNow returns a fixed time for deterministic testing.
func fixedNow() time.Time {
	return time.Date(2025, 10, 17, 18, 0, 0, 0, time.UTC)
}

// generateTestKey creates a test ECDSA signing key.
func generateTestKey() *ecdsa.PrivateKey {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	return key
}

// generateSignedHash creates a signed hash for a given identity.
func generateSignedHash(identity connector.Identity, signingKey *ecdsa.PrivateKey) (string, error) {
	h := sha3.New512()
	h.Write([]byte(identity.Email))
	for _, g := range identity.Groups {
		h.Write([]byte(g))
	}
	h.Write([]byte(identity.UserID))
	h.Write([]byte(identity.Username))
	h.Write([]byte(identity.PreferredUsername))
	hash := fmt.Sprintf("%x", h.Sum(nil))

	signAlg, _ := jwt.SignatureAlgorithm(&jose.JSONWebKey{Key: signingKey})
	signedBytes, err := jwt.SignPayload(&jose.JSONWebKey{Key: signingKey}, signAlg, []byte(hash))
	if err != nil {
		return "", err
	}
	// Explicitly encode as []byte.
	return base64.RawURLEncoding.EncodeToString([]byte(signedBytes)), nil
}

func TestHandleRememberMe(t *testing.T) {
	// Common test fixtures.
	ctx := context.Background()
	connectorName := "test-connector"
	expiryDuration := 24 * time.Hour
	testKey := generateTestKey()
	mockStore := &mockStorage{
		keys: storage.Keys{
			SigningKey: &jose.JSONWebKey{Key: testKey},
		},
	}
	mockSessionStore := &mockSessionStorage{
		sessions: make(map[string]storage.ActiveSession),
	}

	// Helper to create a request with optional cookie.
	newRequest := func(cookieValue string) *http.Request {
		req := httptest.NewRequest("GET", "/auth", nil)
		if cookieValue != "" {
			cookieName := connector_cookie_name(connectorName)
			req.AddCookie(&http.Cookie{Name: cookieName, Value: cookieValue})
		}
		return req
	}

	// Sample identity.
	identity := connector.Identity{
		UserID:            "user123",
		Username:          "testuser",
		PreferredUsername: "testuser",
		Email:             "test@example.com",
		EmailVerified:     true,
		Groups:            []string{"group1"},
	}

	t.Run("No cookie, anonymous context", func(t *testing.T) {
		req := newRequest("") // No cookie.
		data := NewAnonymousAuthContext(connectorName, expiryDuration)
		ctx, err := HandleRememberMe(ctx, noOpLogger, req, data, mockStore, mockSessionStore)
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Expected ErrNotFound, got: %v", err)
		}
		if ctx != nil {
			t.Error("Expected nil context")
		}
	})

	t.Run("No cookie, with identity: Create new session and set cookie", func(t *testing.T) {
		req := newRequest("") // No cookie.
		data := NewAuthContextWithIdentity(connectorName, identity, expiryDuration)
		rmCtx, err := HandleRememberMe(ctx, noOpLogger, req, data, mockStore, mockSessionStore)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !rmCtx.IsValid() {
			t.Error("Expected valid session")
		}
		if rmCtx.Session.Identity.UserID != identity.UserID {
			t.Errorf("Expected UserID %s, got %s", identity.UserID, rmCtx.Session.Identity.UserID)
		}
		if rmCtx.Cookie.Empty() || rmCtx.Cookie.unset {
			t.Error("Expected cookie to be set")
		}
		cookie, _ := rmCtx.Cookie.Get()
		if cookie.Name != connector_cookie_name(connectorName) {
			t.Errorf("Unexpected cookie name: %s", cookie.Name)
		}
		if cookie.Value == "" {
			t.Error("Expected non-empty cookie value")
		}
		if cookie.Expires.Before(fixedNow().Add(expiryDuration - time.Minute)) { // Allow some leeway.
			t.Errorf("Unexpected expiry: %v", cookie.Expires)
		}
		// Verify session was stored.
		storedSession, err := mockSessionStore.GetSession(ctx, cookie.Value)
		if err != nil || storedSession.Identity.UserID != identity.UserID {
			t.Error("Session not stored correctly")
		}
	})

	t.Run("Valid cookie with active session", func(t *testing.T) {
		// Pre-populate a session.
		signedHash, err := generateSignedHash(identity, testKey)
		if err != nil {
			t.Fatal(err)
		}
		session := storage.ActiveSession{
			Identity: identity,
			Expiry:   fixedNow().Add(expiryDuration),
		}
		mockSessionStore.sessions[signedHash] = session

		req := newRequest(signedHash)
		data := NewAnonymousAuthContext(connectorName, expiryDuration)
		rmCtx, err := HandleRememberMe(ctx, noOpLogger, req, data, mockStore, mockSessionStore)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !rmCtx.IsValid() {
			t.Error("Expected valid session")
		}
		if rmCtx.Session.Identity.UserID != identity.UserID {
			t.Errorf("Expected UserID %s, got %s", identity.UserID, rmCtx.Session.Identity.UserID)
		}
		if !rmCtx.Cookie.Empty() {
			t.Error("Expected no cookie change (empty GetOrUnsetCookie)")
		}
	})

	t.Run("Cookie present, expired session: Unset cookie", func(t *testing.T) {
		// Pre-populate an expired session.
		signedHash, err := generateSignedHash(identity, testKey)
		if err != nil {
			t.Fatal(err)
		}
		session := storage.ActiveSession{
			Identity: identity,
			Expiry:   fixedNow().Add(-time.Hour), // Expired.
		}
		mockSessionStore.sessions[signedHash] = session

		req := newRequest(signedHash)
		data := NewAnonymousAuthContext(connectorName, expiryDuration)
		rmCtx, err := HandleRememberMe(ctx, noOpLogger, req, data, mockStore, mockSessionStore)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if rmCtx.IsValid() {
			t.Error("Expected invalid (expired) session")
		}
		if rmCtx.Cookie.Empty() || !rmCtx.Cookie.unset {
			t.Error("Expected cookie to be unset")
		}
		cookie, unset := rmCtx.Cookie.Get()
		if !unset || cookie.Name != connector_cookie_name(connectorName) || cookie.MaxAge != -1 {
			t.Error("Unexpected unset cookie")
		}
	})

	t.Run("Cookie present, session not found: Unset cookie", func(t *testing.T) {
		signedHash := "invalid-hash"
		req := newRequest(signedHash)
		data := NewAnonymousAuthContext(connectorName, expiryDuration)
		rmCtx, err := HandleRememberMe(ctx, noOpLogger, req, data, mockStore, mockSessionStore)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if rmCtx.IsValid() {
			t.Error("Expected invalid session")
		}
		if rmCtx.Cookie.Empty() || !rmCtx.Cookie.unset {
			t.Error("Expected cookie to be unset")
		}
	})

	t.Run("Error: Failed to get keys", func(t *testing.T) {
		mockStore.err = errors.New("key error")
		req := newRequest("")
		data := NewAuthContextWithIdentity(connectorName, identity, expiryDuration)
		_, err := HandleRememberMe(ctx, noOpLogger, req, data, mockStore, mockSessionStore)
		if err == nil || !errors.Is(err, mockStore.err) {
			t.Errorf("Expected key error, got: %v", err)
		}
		mockStore.err = nil // Reset.
	})

	t.Run("Error: Failed to create session", func(t *testing.T) {
		mockSessionStore.createErr = errors.New("create error")
		req := newRequest("")
		data := NewAuthContextWithIdentity(connectorName, identity, expiryDuration)
		_, err := HandleRememberMe(ctx, noOpLogger, req, data, mockStore, mockSessionStore)
		if err == nil || !errors.Is(err, mockSessionStore.createErr) {
			t.Errorf("Expected create error, got: %v", err)
		}
		mockSessionStore.createErr = nil // Reset.
	})

	t.Run("Error: Failed to get session", func(t *testing.T) {
		mockSessionStore.getErr = errors.New("get error")
		signedHash, _ := generateSignedHash(identity, testKey)
		req := newRequest(signedHash)
		data := NewAnonymousAuthContext(connectorName, expiryDuration)
		_, err := HandleRememberMe(ctx, noOpLogger, req, data, mockStore, mockSessionStore)
		if err == nil || !errors.Is(err, mockSessionStore.getErr) {
			t.Errorf("Expected get error, got: %v", err)
		}
		mockSessionStore.getErr = nil // Reset.
	})
}

func TestExtractCookie(t *testing.T) {
	connectorName := "test-connector"
	req := httptest.NewRequest("GET", "/auth", nil)
	req.AddCookie(&http.Cookie{Name: connector_cookie_name(connectorName), Value: "test-value"})
	req.AddCookie(&http.Cookie{Name: "other-cookie", Value: "ignored"})

	value, found := extractCookie(req, connectorName)
	if !found || value != "test-value" {
		t.Errorf("Expected value 'test-value', got: %s (found: %v)", value, found)
	}

	// No cookie.
	req = httptest.NewRequest("GET", "/auth", nil)
	value, found = extractCookie(req, connectorName)
	if found || value != "" {
		t.Error("Expected not found")
	}
}

func TestConnectorCookieName(t *testing.T) {
	if name := connector_cookie_name("test"); name != "dex_active_session_cookie_test" {
		t.Errorf("Unexpected name: %s", name)
	}
}
