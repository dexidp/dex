package server

import (
	"crypto"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func newTestSessionServer(t *testing.T) *Server {
	t.Helper()

	now := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	issuerURL, err := url.Parse("https://example.com/dex")
	require.NoError(t, err)

	return &Server{
		storage: memory.New(nil),
		logger:  slog.Default(),
		now:     func() time.Time { return now },
		sessionConfig: &SessionConfig{
			CookieName:        "dex_session",
			AbsoluteLifetime:  24 * time.Hour,
			ValidIfNotUsedFor: 1 * time.Hour,
		},
		issuerURL: *issuerURL,
	}
}

func TestSetSessionCookie(t *testing.T) {
	s := newTestSessionServer(t)
	w := httptest.NewRecorder()

	s.setSessionCookie(w, "user1", "conn1", "nonce123", false)

	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)

	c := cookies[0]
	assert.Equal(t, "dex_session", c.Name)
	assert.Equal(t, sessionCookieValue("user1", "conn1", "nonce123", nil), c.Value)
	assert.Equal(t, "/dex", c.Path)
	assert.True(t, c.HttpOnly)
	assert.True(t, c.Secure)
	assert.Equal(t, http.SameSiteLaxMode, c.SameSite)
}

func TestSetSessionCookie_HTTP(t *testing.T) {
	s := newTestSessionServer(t)
	u, _ := url.Parse("http://localhost:5556/dex")
	s.issuerURL = *u
	w := httptest.NewRecorder()

	s.setSessionCookie(w, "user1", "conn1", "nonce123", false)

	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.False(t, cookies[0].Secure)
}

func TestClearSessionCookie(t *testing.T) {
	s := newTestSessionServer(t)
	w := httptest.NewRecorder()

	s.clearSessionCookie(w)

	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, -1, cookies[0].MaxAge)
	assert.Equal(t, "", cookies[0].Value)
}

func TestSessionCookieValueRoundtrip(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		connectorID string
		nonce       string
	}{
		{"simple", "user1", "ldap", "abc123"},
		{"with special chars", "user@example.com", "oidc-provider", "xyz789"},
		{"unicode", "юзер", "коннектор", "nonce"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := sessionCookieValue(tt.userID, tt.connectorID, tt.nonce, nil)
			gotUser, gotConn, gotNonce, err := parseSessionCookie(value, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.userID, gotUser)
			assert.Equal(t, tt.connectorID, gotConn)
			assert.Equal(t, tt.nonce, gotNonce)
		})
	}
}

func TestSessionCookieValueEncryptedRoundtrip(t *testing.T) {
	key := []byte("0123456789abcdef") // 16 bytes = AES-128

	value := sessionCookieValue("user1", "ldap", "nonce1", key)
	// Encrypted value must differ from unencrypted.
	unencrypted := sessionCookieValue("user1", "ldap", "nonce1", nil)
	assert.NotEqual(t, unencrypted, value)

	// Must decrypt correctly.
	gotUser, gotConn, gotNonce, err := parseSessionCookie(value, key)
	require.NoError(t, err)
	assert.Equal(t, "user1", gotUser)
	assert.Equal(t, "ldap", gotConn)
	assert.Equal(t, "nonce1", gotNonce)

	// Wrong key must fail.
	wrongKey := []byte("abcdef0123456789")
	//nolint:dogsled // only for tests
	_, _, _, err = parseSessionCookie(value, wrongKey)
	assert.Error(t, err)

	// No key must fail (encrypted value isn't valid protobuf).
	//nolint:dogsled // only for tests
	_, _, _, err = parseSessionCookie(value, nil)
	assert.Error(t, err)
}

func TestParseSessionCookie_Invalid(t *testing.T) {
	//nolint:dogsled // only for tests
	_, _, _, err := parseSessionCookie("invalid", nil)
	assert.Error(t, err)
	//nolint:dogsled // only for tests
	_, _, _, err = parseSessionCookie("a.b", nil)
	assert.Error(t, err)
}

func TestGetValidAuthSession(t *testing.T) {
	ctx := t.Context()
	authReq := &storage.AuthRequest{ConnectorID: "conn1"}

	t.Run("no session config", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.sessionConfig = nil
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		assert.Nil(t, s.getValidAuthSession(ctx, httptest.NewRecorder(), r, authReq))
	})

	t.Run("no cookie", func(t *testing.T) {
		s := newTestSessionServer(t)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		assert.Nil(t, s.getValidAuthSession(ctx, httptest.NewRecorder(), r, authReq))
	})

	t.Run("invalid cookie format", func(t *testing.T) {
		s := newTestSessionServer(t)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: "invalid-format"})
		w := httptest.NewRecorder()
		assert.Nil(t, s.getValidAuthSession(ctx, w, r, authReq))
		// Cookie should be cleared.
		assert.Equal(t, -1, w.Result().Cookies()[0].MaxAge)
	})

	t.Run("session not found", func(t *testing.T) {
		s := newTestSessionServer(t)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("nouser", "noconn", "nonce", nil)})
		w := httptest.NewRecorder()
		assert.Nil(t, s.getValidAuthSession(ctx, w, r, authReq))
		// Cookie should be cleared.
		assert.Equal(t, -1, w.Result().Cookies()[0].MaxAge)
	})

	t.Run("valid session", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()
		nonce := "test-nonce"

		session := storage.AuthSession{
			UserID:         "user1",
			ConnectorID:    "conn1",
			Nonce:          nonce,
			ClientStates:   map[string]*storage.ClientAuthState{},
			CreatedAt:      now.Add(-30 * time.Minute),
			LastActivity:   now.Add(-5 * time.Minute),
			IPAddress:      "127.0.0.1",
			UserAgent:      "test",
			AbsoluteExpiry: now.Add(24 * time.Hour),
			IdleExpiry:     now.Add(1 * time.Hour),
		}
		require.NoError(t, s.storage.CreateAuthSession(ctx, session))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user1", "conn1", nonce, nil)})

		result := s.getValidAuthSession(ctx, httptest.NewRecorder(), r, authReq)
		require.NotNil(t, result)
		assert.Equal(t, "user1", result.UserID)
		assert.Equal(t, "conn1", result.ConnectorID)
	})

	t.Run("connector mismatch", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()
		nonce := "test-nonce-conn"

		session := storage.AuthSession{
			UserID:         "user1",
			ConnectorID:    "ldap",
			Nonce:          nonce,
			ClientStates:   map[string]*storage.ClientAuthState{},
			CreatedAt:      now.Add(-30 * time.Minute),
			LastActivity:   now.Add(-5 * time.Minute),
			IPAddress:      "127.0.0.1",
			UserAgent:      "test",
			AbsoluteExpiry: now.Add(24 * time.Hour),
			IdleExpiry:     now.Add(1 * time.Hour),
		}
		require.NoError(t, s.storage.CreateAuthSession(ctx, session))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user1", "ldap", nonce, nil)})

		githubReq := &storage.AuthRequest{ConnectorID: "github"}
		assert.Nil(t, s.getValidAuthSession(ctx, httptest.NewRecorder(), r, githubReq))
	})

	t.Run("nonce mismatch", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()

		session := storage.AuthSession{
			UserID:         "user2",
			ConnectorID:    "conn2",
			Nonce:          "correct-nonce",
			ClientStates:   map[string]*storage.ClientAuthState{},
			CreatedAt:      now.Add(-30 * time.Minute),
			LastActivity:   now.Add(-5 * time.Minute),
			IPAddress:      "127.0.0.1",
			UserAgent:      "test",
			AbsoluteExpiry: now.Add(24 * time.Hour),
			IdleExpiry:     now.Add(1 * time.Hour),
		}
		require.NoError(t, s.storage.CreateAuthSession(ctx, session))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user2", "conn2", "wrong-nonce", nil)})

		conn2Req := &storage.AuthRequest{ConnectorID: "conn2"}
		w := httptest.NewRecorder()
		assert.Nil(t, s.getValidAuthSession(ctx, w, r, conn2Req))
		assert.Equal(t, -1, w.Result().Cookies()[0].MaxAge)
	})

	t.Run("expired absolute lifetime", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()
		nonce := "expired-nonce"

		session := storage.AuthSession{
			UserID:         "user3",
			ConnectorID:    "conn3",
			Nonce:          nonce,
			ClientStates:   map[string]*storage.ClientAuthState{},
			CreatedAt:      now.Add(-25 * time.Hour),
			LastActivity:   now.Add(-1 * time.Minute),
			IPAddress:      "127.0.0.1",
			UserAgent:      "test",
			AbsoluteExpiry: now.Add(-1 * time.Hour),
			IdleExpiry:     now.Add(1 * time.Hour),
		}
		require.NoError(t, s.storage.CreateAuthSession(ctx, session))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user3", "conn3", nonce, nil)})

		conn3Req := &storage.AuthRequest{ConnectorID: "conn3"}
		w := httptest.NewRecorder()
		assert.Nil(t, s.getValidAuthSession(ctx, w, r, conn3Req))
		assert.Equal(t, -1, w.Result().Cookies()[0].MaxAge)

		// Session should be deleted.
		_, err := s.storage.GetAuthSession(ctx, "user3", "conn3")
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})

	t.Run("expired idle timeout", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()
		nonce := "idle-nonce"

		session := storage.AuthSession{
			UserID:         "user4",
			ConnectorID:    "conn4",
			Nonce:          nonce,
			ClientStates:   map[string]*storage.ClientAuthState{},
			CreatedAt:      now.Add(-2 * time.Hour),
			LastActivity:   now.Add(-2 * time.Hour),
			IPAddress:      "127.0.0.1",
			UserAgent:      "test",
			AbsoluteExpiry: now.Add(22 * time.Hour),
			IdleExpiry:     now.Add(-1 * time.Hour),
		}
		require.NoError(t, s.storage.CreateAuthSession(ctx, session))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user4", "conn4", nonce, nil)})

		conn4Req := &storage.AuthRequest{ConnectorID: "conn4"}
		w := httptest.NewRecorder()
		assert.Nil(t, s.getValidAuthSession(ctx, w, r, conn4Req))
		assert.Equal(t, -1, w.Result().Cookies()[0].MaxAge)

		// Session should be deleted.
		_, err := s.storage.GetAuthSession(ctx, "user4", "conn4")
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}

func TestCreateOrUpdateAuthSession(t *testing.T) {
	ctx := t.Context()

	t.Run("create new session", func(t *testing.T) {
		s := newTestSessionServer(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		authReq := storage.AuthRequest{
			ID:          "auth-1",
			ClientID:    "client-1",
			Claims:      storage.Claims{UserID: "user-1"},
			ConnectorID: "mock",
		}

		err := s.createOrUpdateAuthSession(ctx, r, w, authReq, false)
		require.NoError(t, err)

		// Cookie should be set.
		cookies := w.Result().Cookies()
		require.Len(t, cookies, 1)

		userID, connectorID, nonce, err := parseSessionCookie(cookies[0].Value, nil)
		require.NoError(t, err)
		assert.Equal(t, "user-1", userID)
		assert.Equal(t, "mock", connectorID)
		assert.NotEmpty(t, nonce)

		// Session should exist in storage.
		session, err := s.storage.GetAuthSession(ctx, "user-1", "mock")
		require.NoError(t, err)
		assert.Equal(t, "user-1", session.UserID)
		assert.Equal(t, "mock", session.ConnectorID)
		require.Contains(t, session.ClientStates, "client-1")
		assert.True(t, session.ClientStates["client-1"].Active)
	})

	t.Run("update existing session", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()
		nonce := "existing-nonce"

		existingSession := storage.AuthSession{
			UserID:      "user-1",
			ConnectorID: "mock",
			Nonce:       nonce,
			ClientStates: map[string]*storage.ClientAuthState{
				"client-1": {
					Active:       true,
					ExpiresAt:    now.Add(24 * time.Hour),
					LastActivity: now.Add(-10 * time.Minute),
				},
			},
			CreatedAt:      now.Add(-30 * time.Minute),
			LastActivity:   now.Add(-10 * time.Minute),
			IPAddress:      "127.0.0.1",
			UserAgent:      "test",
			AbsoluteExpiry: now.Add(24 * time.Hour),
			IdleExpiry:     now.Add(50 * time.Minute),
		}
		require.NoError(t, s.storage.CreateAuthSession(ctx, existingSession))

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		authReq := storage.AuthRequest{
			ID:          "auth-2",
			ClientID:    "client-2",
			Claims:      storage.Claims{UserID: "user-1"},
			ConnectorID: "mock",
		}

		err := s.createOrUpdateAuthSession(ctx, r, w, authReq, false)
		require.NoError(t, err)

		// Cookie should be set with existing nonce.
		cookies := w.Result().Cookies()
		require.Len(t, cookies, 1)
		_, _, gotNonce, err := parseSessionCookie(cookies[0].Value, nil)
		require.NoError(t, err)
		assert.Equal(t, nonce, gotNonce)

		// Session should have both clients.
		session, err := s.storage.GetAuthSession(ctx, "user-1", "mock")
		require.NoError(t, err)
		assert.Len(t, session.ClientStates, 2)
		assert.Contains(t, session.ClientStates, "client-1")
		assert.Contains(t, session.ClientStates, "client-2")
	})

	t.Run("nil session config", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.sessionConfig = nil
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		err := s.createOrUpdateAuthSession(ctx, r, w, storage.AuthRequest{}, false)
		assert.NoError(t, err)
		assert.Empty(t, w.Result().Cookies())
	})
}

// setupSessionLoginFixture creates the necessary storage objects for trySessionLogin tests.
func setupSessionLoginFixture(t *testing.T, s *Server) storage.AuthRequest {
	t.Helper()
	ctx := t.Context()
	now := s.now()

	require.NoError(t, s.storage.CreateAuthSession(ctx, storage.AuthSession{
		UserID:      "user-1",
		ConnectorID: "mock",
		Nonce:       "test-nonce",
		ClientStates: map[string]*storage.ClientAuthState{
			"client-1": {
				Active:       true,
				ExpiresAt:    now.Add(24 * time.Hour),
				LastActivity: now.Add(-1 * time.Minute),
			},
		},
		CreatedAt:      now.Add(-30 * time.Minute),
		LastActivity:   now.Add(-1 * time.Minute),
		IPAddress:      "127.0.0.1",
		UserAgent:      "test",
		AbsoluteExpiry: now.Add(24 * time.Hour),
		IdleExpiry:     now.Add(59 * time.Minute),
	}))

	require.NoError(t, s.storage.CreateUserIdentity(ctx, storage.UserIdentity{
		UserID:      "user-1",
		ConnectorID: "mock",
		Claims: storage.Claims{
			UserID:   "user-1",
			Username: "testuser",
			Email:    "test@example.com",
		},
		Consents:  map[string][]string{"client-1": {"openid", "email"}},
		CreatedAt: now.Add(-1 * time.Hour),
		LastLogin: now.Add(-30 * time.Minute),
	}))

	authReq := storage.AuthRequest{
		ID:          storage.NewID(),
		ClientID:    "client-1",
		ConnectorID: "mock",
		Scopes:      []string{"openid", "email"},
		RedirectURI: "http://localhost/callback",
		MaxAge:      -1,
		HMACKey:     storage.NewHMACKey(crypto.SHA256),
		Expiry:      now.Add(10 * time.Minute),
	}
	require.NoError(t, s.storage.CreateAuthRequest(ctx, authReq))
	return authReq
}

func sessionCookieRequest(userID, connectorID, nonce string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue(userID, connectorID, nonce, nil)})
	return r
}

func TestTrySessionLogin(t *testing.T) {
	ctx := t.Context()

	t.Run("no session", func(t *testing.T) {
		s := newTestSessionServer(t)
		authReq := storage.AuthRequest{ConnectorID: "mock"}
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		_, ok := s.trySessionLogin(ctx, r, w, &authReq)
		assert.False(t, ok)
	})

	t.Run("successful login with skipApproval", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = true
		authReq := setupSessionLoginFixture(t, s)

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		_, ok := s.trySessionLogin(ctx, r, w, &authReq)
		assert.True(t, ok)
	})

	t.Run("successful login redirects to approval", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = false
		authReq := setupSessionLoginFixture(t, s)
		authReq.ForceApprovalPrompt = true

		require.NoError(t, s.storage.UpdateAuthRequest(ctx, authReq.ID, func(a storage.AuthRequest) (storage.AuthRequest, error) {
			a.ForceApprovalPrompt = true
			return a, nil
		}))

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		redirectURL, ok := s.trySessionLogin(ctx, r, w, &authReq)
		assert.True(t, ok)
		assert.Contains(t, redirectURL, "/approval")
		assert.Contains(t, redirectURL, "req="+authReq.ID)
	})

	t.Run("skips approval when consent already given", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = false
		authReq := setupSessionLoginFixture(t, s)

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		_, ok := s.trySessionLogin(ctx, r, w, &authReq)
		assert.True(t, ok)
	})

	t.Run("connector mismatch returns false", func(t *testing.T) {
		s := newTestSessionServer(t)
		authReq := setupSessionLoginFixture(t, s)
		authReq.ConnectorID = "github"

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		_, ok := s.trySessionLogin(ctx, r, w, &authReq)
		assert.False(t, ok)
	})

	t.Run("no client state for requested client", func(t *testing.T) {
		s := newTestSessionServer(t)
		authReq := setupSessionLoginFixture(t, s)
		authReq.ClientID = "unknown-client"

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		_, ok := s.trySessionLogin(ctx, r, w, &authReq)
		assert.False(t, ok)
	})

	t.Run("expired client state returns false", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()

		require.NoError(t, s.storage.CreateAuthSession(t.Context(), storage.AuthSession{
			UserID:      "user-exp",
			ConnectorID: "mock",
			Nonce:       "nonce-exp",
			ClientStates: map[string]*storage.ClientAuthState{
				"client-1": {
					Active:    true,
					ExpiresAt: now.Add(-1 * time.Hour),
				},
			},
			CreatedAt:      now.Add(-2 * time.Hour),
			LastActivity:   now.Add(-1 * time.Minute),
			AbsoluteExpiry: now.Add(22 * time.Hour),
			IdleExpiry:     now.Add(59 * time.Minute),
		}))

		require.NoError(t, s.storage.CreateUserIdentity(t.Context(), storage.UserIdentity{
			UserID:      "user-exp",
			ConnectorID: "mock",
			Claims:      storage.Claims{UserID: "user-exp"},
			Consents:    make(map[string][]string),
			CreatedAt:   now,
			LastLogin:   now,
		}))

		authReq := storage.AuthRequest{
			ID:          storage.NewID(),
			ClientID:    "client-1",
			ConnectorID: "mock",
			MaxAge:      -1,
			HMACKey:     storage.NewHMACKey(crypto.SHA256),
			Expiry:      now.Add(10 * time.Minute),
		}
		require.NoError(t, s.storage.CreateAuthRequest(t.Context(), authReq))

		r := sessionCookieRequest("user-exp", "mock", "nonce-exp")
		w := httptest.NewRecorder()

		_, ok := s.trySessionLogin(ctx, r, w, &authReq)
		assert.False(t, ok)
	})

	t.Run("updates session activity", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = true
		authReq := setupSessionLoginFixture(t, s)

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		_, ok := s.trySessionLogin(ctx, r, w, &authReq)
		require.True(t, ok)

		session, err := s.storage.GetAuthSession(ctx, "user-1", "mock")
		require.NoError(t, err)
		assert.Equal(t, s.now(), session.LastActivity)
	})
}

// setupSessionWithIdentity creates an AuthSession, UserIdentity, and AuthRequest in storage
// for use in trySessionLogin tests. Returns the authReq.
func setupSessionWithIdentity(t *testing.T, s *Server, now time.Time, lastLogin time.Time) storage.AuthRequest {
	t.Helper()
	ctx := t.Context()
	nonce := "test-nonce"

	session := storage.AuthSession{
		UserID:      "user-1",
		ConnectorID: "mock",
		Nonce:       nonce,
		ClientStates: map[string]*storage.ClientAuthState{
			"client-1": {
				Active:       true,
				ExpiresAt:    now.Add(24 * time.Hour),
				LastActivity: now.Add(-1 * time.Minute),
			},
		},
		CreatedAt:    now.Add(-30 * time.Minute),
		LastActivity: now.Add(-1 * time.Minute),
		IPAddress:    "127.0.0.1",
		UserAgent:    "test",
	}
	require.NoError(t, s.storage.CreateAuthSession(ctx, session))

	ui := storage.UserIdentity{
		UserID:      "user-1",
		ConnectorID: "mock",
		Claims: storage.Claims{
			UserID:   "user-1",
			Username: "testuser",
			Email:    "test@example.com",
		},
		Consents:  make(map[string][]string),
		CreatedAt: now.Add(-1 * time.Hour),
		LastLogin: lastLogin,
	}
	require.NoError(t, s.storage.CreateUserIdentity(ctx, ui))

	authReq := storage.AuthRequest{
		ID:          storage.NewID(),
		ClientID:    "client-1",
		ConnectorID: "mock",
		Scopes:      []string{"openid"},
		RedirectURI: "http://localhost/callback",
		MaxAge:      -1,
		HMACKey:     storage.NewHMACKey(crypto.SHA256),
		Expiry:      now.Add(10 * time.Minute),
	}
	require.NoError(t, s.storage.CreateAuthRequest(ctx, authReq))

	return authReq
}

func TestTrySessionLogin_MaxAge(t *testing.T) {
	ctx := t.Context()

	t.Run("max_age not specified, session reused", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()

		authReq := setupSessionWithIdentity(t, s, now, now.Add(-2*time.Hour))
		authReq.MaxAge = -1 // not specified

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user-1", "mock", "test-nonce", nil)})
		w := httptest.NewRecorder()

		_, ok := s.trySessionLogin(ctx, r, w, &authReq)
		assert.True(t, ok, "session should be reused when max_age is not specified")
	})

	t.Run("max_age satisfied, session reused", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()

		// User logged in 10 minutes ago, max_age=3600 (1 hour)
		authReq := setupSessionWithIdentity(t, s, now, now.Add(-10*time.Minute))
		authReq.MaxAge = 3600

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user-1", "mock", "test-nonce", nil)})
		w := httptest.NewRecorder()

		_, ok := s.trySessionLogin(ctx, r, w, &authReq)
		assert.True(t, ok, "session should be reused when max_age is satisfied")
	})

	t.Run("max_age exceeded, force re-auth", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()

		// User logged in 2 hours ago, max_age=3600 (1 hour)
		authReq := setupSessionWithIdentity(t, s, now, now.Add(-2*time.Hour))
		authReq.MaxAge = 3600

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user-1", "mock", "test-nonce", nil)})
		w := httptest.NewRecorder()

		_, ok := s.trySessionLogin(ctx, r, w, &authReq)
		assert.False(t, ok, "session should NOT be reused when max_age is exceeded")
	})

	t.Run("max_age=0, always force re-auth", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()

		// User logged in 1 second ago, max_age=0
		authReq := setupSessionWithIdentity(t, s, now, now.Add(-1*time.Second))
		authReq.MaxAge = 0

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user-1", "mock", "test-nonce", nil)})
		w := httptest.NewRecorder()

		_, ok := s.trySessionLogin(ctx, r, w, &authReq)
		assert.False(t, ok, "max_age=0 should always force re-authentication")
	})

	t.Run("auth_time is set from UserIdentity.LastLogin", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = false
		now := s.now()
		lastLogin := now.Add(-10 * time.Minute)

		authReq := setupSessionWithIdentity(t, s, now, lastLogin)
		authReq.ForceApprovalPrompt = true // force approval so AuthRequest is not deleted

		require.NoError(t, s.storage.UpdateAuthRequest(ctx, authReq.ID, func(a storage.AuthRequest) (storage.AuthRequest, error) {
			a.ForceApprovalPrompt = true
			return a, nil
		}))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user-1", "mock", "test-nonce", nil)})
		w := httptest.NewRecorder()

		redirectURL, ok := s.trySessionLogin(ctx, r, w, &authReq)
		require.True(t, ok)
		assert.Contains(t, redirectURL, "/approval")

		// Verify AuthTime was set on the auth request.
		updated, err := s.storage.GetAuthRequest(ctx, authReq.ID)
		require.NoError(t, err)
		assert.Equal(t, lastLogin.Unix(), updated.AuthTime.Unix())
	})
}

func TestTrySessionLoginWithSession_IDTokenHint(t *testing.T) {
	ctx := t.Context()

	// genSubject("user-1", "mock") produces a deterministic subject string.
	hintSubjectForUser1Mock, err := genSubject("user-1", "mock")
	require.NoError(t, err)

	hintSubjectOther, err := genSubject("other-user", "mock")
	require.NoError(t, err)

	t.Run("hint matches session user - session login succeeds", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = true
		authReq := setupSessionLoginFixture(t, s)

		session := s.getValidAuthSession(ctx, httptest.NewRecorder(), sessionCookieRequest("user-1", "mock", "test-nonce"), &authReq)
		require.NotNil(t, session)

		// Verify hint matches.
		assert.True(t, sessionMatchesHint(session, hintSubjectForUser1Mock))

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		_, ok := s.trySessionLoginWithSession(ctx, r, w, &authReq, session)
		assert.True(t, ok)
	})

	t.Run("hint does not match session user - session invalidated", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = true
		authReq := setupSessionLoginFixture(t, s)

		session := s.getValidAuthSession(ctx, httptest.NewRecorder(), sessionCookieRequest("user-1", "mock", "test-nonce"), &authReq)
		require.NotNil(t, session)

		// Verify hint does NOT match.
		assert.False(t, sessionMatchesHint(session, hintSubjectOther))

		// Simulating the hint mismatch logic from handleConnectorLogin:
		// when hint doesn't match and prompt is not none, session is set to nil.
		var nilSession *storage.AuthSession
		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		_, ok := s.trySessionLoginWithSession(ctx, r, w, &authReq, nilSession)
		assert.False(t, ok, "session login should fail when session is invalidated due to hint mismatch")
	})

	t.Run("hint with no session - trySessionLoginWithSession returns false", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = true
		authReq := setupSessionLoginFixture(t, s)

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		_, ok := s.trySessionLoginWithSession(ctx, r, w, &authReq, nil)
		assert.False(t, ok)
	})

	t.Run("no hint - unchanged behavior", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = true
		authReq := setupSessionLoginFixture(t, s)

		session := s.getValidAuthSession(ctx, httptest.NewRecorder(), sessionCookieRequest("user-1", "mock", "test-nonce"), &authReq)
		require.NotNil(t, session)

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		_, ok := s.trySessionLoginWithSession(ctx, r, w, &authReq, session)
		assert.True(t, ok)
	})
}

func TestParseAuthRequest_PromptAndMaxAge(t *testing.T) {
	t.Run("prompt=consent sets ForceApprovalPrompt", func(t *testing.T) {
		authReq := storage.AuthRequest{
			Prompt:              "consent",
			ForceApprovalPrompt: true,
		}
		assert.True(t, authReq.ForceApprovalPrompt)
		assert.Equal(t, "consent", authReq.Prompt)
	})

	t.Run("max_age default is -1", func(t *testing.T) {
		authReq := storage.AuthRequest{
			MaxAge: -1,
		}
		assert.Equal(t, -1, authReq.MaxAge)
	})
}

func TestClientSharesSessionWith(t *testing.T) {
	tests := []struct {
		name           string
		ssoSharedWith  []string
		defaultPolicy  string
		targetClientID string
		want           bool
	}{
		{
			name:           "nil uses default none",
			ssoSharedWith:  nil,
			defaultPolicy:  "none",
			targetClientID: "client-b",
			want:           false,
		},
		{
			name:           "nil uses default all",
			ssoSharedWith:  nil,
			defaultPolicy:  "all",
			targetClientID: "client-b",
			want:           true,
		},
		{
			name:           "nil with empty default",
			ssoSharedWith:  nil,
			defaultPolicy:  "",
			targetClientID: "client-b",
			want:           false,
		},
		{
			name:           "empty slice means no sharing",
			ssoSharedWith:  []string{},
			defaultPolicy:  "all",
			targetClientID: "client-b",
			want:           false,
		},
		{
			name:           "wildcard shares with everyone",
			ssoSharedWith:  []string{"*"},
			defaultPolicy:  "none",
			targetClientID: "any-client",
			want:           true,
		},
		{
			name:           "explicit match",
			ssoSharedWith:  []string{"client-b", "client-c"},
			defaultPolicy:  "none",
			targetClientID: "client-b",
			want:           true,
		},
		{
			name:           "no match in list",
			ssoSharedWith:  []string{"client-b", "client-c"},
			defaultPolicy:  "none",
			targetClientID: "client-d",
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestSessionServer(t)
			s.sessionConfig.SSOSharedWithDefault = tt.defaultPolicy

			client := storage.Client{
				ID:            "source-client",
				SSOSharedWith: tt.ssoSharedWith,
			}
			got := s.clientSharesSessionWith(client, tt.targetClientID)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFindSSOSession(t *testing.T) {
	ctx := t.Context()

	t.Run("finds SSO session from sharing client", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()

		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID:            "client-a",
			Secret:        "secret",
			Name:          "Client A",
			SSOSharedWith: []string{"client-b"},
		}))

		session := &storage.AuthSession{
			UserID:      "user-1",
			ConnectorID: "mock",
			ClientStates: map[string]*storage.ClientAuthState{
				"client-a": {
					Active:       true,
					ExpiresAt:    now.Add(24 * time.Hour),
					LastActivity: now.Add(-5 * time.Minute),
				},
			},
		}

		assert.NotNil(t, s.findSSOSession(ctx, session, "client-b"))
	})

	t.Run("no SSO when client does not share", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()

		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID:            "client-a",
			Secret:        "secret",
			Name:          "Client A",
			SSOSharedWith: []string{"client-c"}, // Does not share with client-b
		}))

		session := &storage.AuthSession{
			UserID:      "user-1",
			ConnectorID: "mock",
			ClientStates: map[string]*storage.ClientAuthState{
				"client-a": {
					Active:       true,
					ExpiresAt:    now.Add(24 * time.Hour),
					LastActivity: now.Add(-5 * time.Minute),
				},
			},
		}

		assert.Nil(t, s.findSSOSession(ctx, session, "client-b"))
	})

	t.Run("skips inactive client states", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()

		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID:            "client-a",
			Secret:        "secret",
			Name:          "Client A",
			SSOSharedWith: []string{"*"},
		}))

		session := &storage.AuthSession{
			UserID:      "user-1",
			ConnectorID: "mock",
			ClientStates: map[string]*storage.ClientAuthState{
				"client-a": {
					Active:       false, // Inactive
					ExpiresAt:    now.Add(24 * time.Hour),
					LastActivity: now.Add(-5 * time.Minute),
				},
			},
		}

		assert.Nil(t, s.findSSOSession(ctx, session, "client-b"))
	})

	t.Run("skips expired client states", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()

		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID:            "client-a",
			Secret:        "secret",
			Name:          "Client A",
			SSOSharedWith: []string{"*"},
		}))

		session := &storage.AuthSession{
			UserID:      "user-1",
			ConnectorID: "mock",
			ClientStates: map[string]*storage.ClientAuthState{
				"client-a": {
					Active:       true,
					ExpiresAt:    now.Add(-1 * time.Hour), // Expired
					LastActivity: now.Add(-5 * time.Minute),
				},
			},
		}

		assert.Nil(t, s.findSSOSession(ctx, session, "client-b"))
	})

	t.Run("wildcard SSO with default all", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.sessionConfig.SSOSharedWithDefault = "all"
		now := s.now()

		// Client with nil SSOSharedWith — uses default "all"
		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID:     "client-a",
			Secret: "secret",
			Name:   "Client A",
			// SSOSharedWith is nil → uses ssoSharedWithDefault="all"
		}))

		session := &storage.AuthSession{
			UserID:      "user-1",
			ConnectorID: "mock",
			ClientStates: map[string]*storage.ClientAuthState{
				"client-a": {
					Active:       true,
					ExpiresAt:    now.Add(24 * time.Hour),
					LastActivity: now.Add(-5 * time.Minute),
				},
			},
		}

		assert.NotNil(t, s.findSSOSession(ctx, session, "client-b"))
	})
}

func TestTrySessionLogin_SSO(t *testing.T) {
	ctx := t.Context()

	t.Run("SSO login from sharing client", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = true
		now := s.now()

		// Create source client that shares with target
		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID:            "client-a",
			Secret:        "secret",
			Name:          "Client A",
			SSOSharedWith: []string{"client-b"},
		}))

		// Create session with client-a authenticated
		require.NoError(t, s.storage.CreateAuthSession(ctx, storage.AuthSession{
			UserID:      "user-1",
			ConnectorID: "mock",
			Nonce:       "test-nonce",
			ClientStates: map[string]*storage.ClientAuthState{
				"client-a": {
					Active:       true,
					ExpiresAt:    now.Add(24 * time.Hour),
					LastActivity: now.Add(-1 * time.Minute),
				},
			},
			CreatedAt:      now.Add(-30 * time.Minute),
			LastActivity:   now.Add(-1 * time.Minute),
			IPAddress:      "127.0.0.1",
			UserAgent:      "test",
			AbsoluteExpiry: now.Add(24 * time.Hour),
			IdleExpiry:     now.Add(59 * time.Minute),
		}))

		require.NoError(t, s.storage.CreateUserIdentity(ctx, storage.UserIdentity{
			UserID:      "user-1",
			ConnectorID: "mock",
			Claims: storage.Claims{
				UserID:   "user-1",
				Username: "testuser",
				Email:    "test@example.com",
			},
			Consents:  map[string][]string{"client-b": {"openid", "email"}},
			CreatedAt: now.Add(-1 * time.Hour),
			LastLogin: now.Add(-30 * time.Minute),
		}))

		// Auth request for client-b (not directly in session)
		authReq := storage.AuthRequest{
			ID:          storage.NewID(),
			ClientID:    "client-b",
			ConnectorID: "mock",
			Scopes:      []string{"openid", "email"},
			RedirectURI: "http://localhost/callback",
			MaxAge:      -1,
			HMACKey:     storage.NewHMACKey(crypto.SHA256),
			Expiry:      now.Add(10 * time.Minute),
		}
		require.NoError(t, s.storage.CreateAuthRequest(ctx, authReq))

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		session := s.getValidAuthSession(ctx, w, r, &authReq)
		require.NotNil(t, session)

		_, ok := s.trySessionLoginWithSession(ctx, r, w, &authReq, session)
		assert.True(t, ok, "SSO login should succeed")

		// Verify client-b state was created in session
		updated, err := s.storage.GetAuthSession(ctx, "user-1", "mock")
		require.NoError(t, err)
		assert.Contains(t, updated.ClientStates, "client-b")
		assert.True(t, updated.ClientStates["client-b"].Active)
	})

	t.Run("SSO derived state capped by source expiry", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = true
		now := s.now()

		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID:            "client-a",
			Secret:        "secret",
			Name:          "Client A",
			SSOSharedWith: []string{"client-b"},
		}))

		// Source state expires in 1 hour — less than AbsoluteLifetime (24h).
		sourceExpiry := now.Add(1 * time.Hour)
		require.NoError(t, s.storage.CreateAuthSession(ctx, storage.AuthSession{
			UserID:      "user-1",
			ConnectorID: "mock",
			Nonce:       "test-nonce",
			ClientStates: map[string]*storage.ClientAuthState{
				"client-a": {
					Active:       true,
					ExpiresAt:    sourceExpiry,
					LastActivity: now.Add(-1 * time.Minute),
				},
			},
			CreatedAt:      now.Add(-30 * time.Minute),
			LastActivity:   now.Add(-1 * time.Minute),
			IPAddress:      "127.0.0.1",
			UserAgent:      "test",
			AbsoluteExpiry: now.Add(24 * time.Hour),
			IdleExpiry:     now.Add(59 * time.Minute),
		}))

		require.NoError(t, s.storage.CreateUserIdentity(ctx, storage.UserIdentity{
			UserID:      "user-1",
			ConnectorID: "mock",
			Claims: storage.Claims{
				UserID:   "user-1",
				Username: "testuser",
				Email:    "test@example.com",
			},
			Consents:  map[string][]string{"client-b": {"openid", "email"}},
			CreatedAt: now.Add(-1 * time.Hour),
			LastLogin: now.Add(-30 * time.Minute),
		}))

		authReq := storage.AuthRequest{
			ID:          storage.NewID(),
			ClientID:    "client-b",
			ConnectorID: "mock",
			Scopes:      []string{"openid", "email"},
			RedirectURI: "http://localhost/callback",
			MaxAge:      -1,
			HMACKey:     storage.NewHMACKey(crypto.SHA256),
			Expiry:      now.Add(10 * time.Minute),
		}
		require.NoError(t, s.storage.CreateAuthRequest(ctx, authReq))

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		session := s.getValidAuthSession(ctx, w, r, &authReq)
		require.NotNil(t, session)

		_, ok := s.trySessionLoginWithSession(ctx, r, w, &authReq, session)
		assert.True(t, ok, "SSO login should succeed")

		updated, err := s.storage.GetAuthSession(ctx, "user-1", "mock")
		require.NoError(t, err)
		require.Contains(t, updated.ClientStates, "client-b")
		assert.Equal(t, sourceExpiry, updated.ClientStates["client-b"].ExpiresAt,
			"derived state expiry should be capped at source state expiry")
	})

	t.Run("SSO derived state uses configured lifetime when source expires later", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = true
		now := s.now()

		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID:            "client-a",
			Secret:        "secret",
			Name:          "Client A",
			SSOSharedWith: []string{"client-b"},
		}))

		// Source state expires in 48 hours — more than AbsoluteLifetime (24h).
		require.NoError(t, s.storage.CreateAuthSession(ctx, storage.AuthSession{
			UserID:      "user-1",
			ConnectorID: "mock",
			Nonce:       "test-nonce",
			ClientStates: map[string]*storage.ClientAuthState{
				"client-a": {
					Active:       true,
					ExpiresAt:    now.Add(48 * time.Hour),
					LastActivity: now.Add(-1 * time.Minute),
				},
			},
			CreatedAt:      now.Add(-30 * time.Minute),
			LastActivity:   now.Add(-1 * time.Minute),
			IPAddress:      "127.0.0.1",
			UserAgent:      "test",
			AbsoluteExpiry: now.Add(24 * time.Hour),
			IdleExpiry:     now.Add(59 * time.Minute),
		}))

		require.NoError(t, s.storage.CreateUserIdentity(ctx, storage.UserIdentity{
			UserID:      "user-1",
			ConnectorID: "mock",
			Claims: storage.Claims{
				UserID:   "user-1",
				Username: "testuser",
				Email:    "test@example.com",
			},
			Consents:  map[string][]string{"client-b": {"openid", "email"}},
			CreatedAt: now.Add(-1 * time.Hour),
			LastLogin: now.Add(-30 * time.Minute),
		}))

		authReq := storage.AuthRequest{
			ID:          storage.NewID(),
			ClientID:    "client-b",
			ConnectorID: "mock",
			Scopes:      []string{"openid", "email"},
			RedirectURI: "http://localhost/callback",
			MaxAge:      -1,
			HMACKey:     storage.NewHMACKey(crypto.SHA256),
			Expiry:      now.Add(10 * time.Minute),
		}
		require.NoError(t, s.storage.CreateAuthRequest(ctx, authReq))

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		session := s.getValidAuthSession(ctx, w, r, &authReq)
		require.NotNil(t, session)

		_, ok := s.trySessionLoginWithSession(ctx, r, w, &authReq, session)
		assert.True(t, ok, "SSO login should succeed")

		updated, err := s.storage.GetAuthSession(ctx, "user-1", "mock")
		require.NoError(t, err)
		require.Contains(t, updated.ClientStates, "client-b")
		assert.Equal(t, now.Add(s.sessionConfig.AbsoluteLifetime), updated.ClientStates["client-b"].ExpiresAt,
			"derived state expiry should use configured AbsoluteLifetime when source expires later")
	})

	t.Run("no SSO when client does not share", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()

		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID:            "client-a",
			Secret:        "secret",
			Name:          "Client A",
			SSOSharedWith: []string{}, // Shares with nobody
		}))

		require.NoError(t, s.storage.CreateAuthSession(ctx, storage.AuthSession{
			UserID:      "user-1",
			ConnectorID: "mock",
			Nonce:       "test-nonce",
			ClientStates: map[string]*storage.ClientAuthState{
				"client-a": {
					Active:       true,
					ExpiresAt:    now.Add(24 * time.Hour),
					LastActivity: now.Add(-1 * time.Minute),
				},
			},
			CreatedAt:      now.Add(-30 * time.Minute),
			LastActivity:   now.Add(-1 * time.Minute),
			IPAddress:      "127.0.0.1",
			UserAgent:      "test",
			AbsoluteExpiry: now.Add(24 * time.Hour),
			IdleExpiry:     now.Add(59 * time.Minute),
		}))

		authReq := storage.AuthRequest{
			ID:          storage.NewID(),
			ClientID:    "client-b",
			ConnectorID: "mock",
			MaxAge:      -1,
			HMACKey:     storage.NewHMACKey(crypto.SHA256),
			Expiry:      now.Add(10 * time.Minute),
		}
		require.NoError(t, s.storage.CreateAuthRequest(ctx, authReq))

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		session := s.getValidAuthSession(ctx, w, r, &authReq)
		require.NotNil(t, session)

		_, ok := s.trySessionLoginWithSession(ctx, r, w, &authReq, session)
		assert.False(t, ok, "SSO login should fail when client does not share")
	})
}

func TestFinishSessionLogin_MFA(t *testing.T) {
	ctx := t.Context()

	setupMFAFixture := func(t *testing.T, mfaProviders map[string]MFAProvider, clientMFAChain []string) (*Server, storage.AuthRequest) {
		t.Helper()
		s := newTestSessionServer(t)
		s.skipApproval = true
		s.mfaProviders = mfaProviders

		// Create connector in storage and register it in the connectors map.
		require.NoError(t, s.storage.CreateConnector(ctx, storage.Connector{
			ID:              "mock",
			Type:            "ldap",
			Name:            "Mock LDAP",
			ResourceVersion: "1",
		}))
		s.mu.Lock()
		s.connectors = map[string]Connector{
			"mock": {Type: "ldap", ResourceVersion: "1"},
		}
		s.mu.Unlock()

		// Create client with MFA chain.
		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID:       "client-1",
			Secret:   "secret",
			Name:     "Test Client",
			MFAChain: clientMFAChain,
		}))

		authReq := setupSessionLoginFixture(t, s)
		return s, authReq
	}

	t.Run("MFA required redirects to MFA page", func(t *testing.T) {
		s, authReq := setupMFAFixture(t, map[string]MFAProvider{
			"totp": NewTOTPProvider("test-issuer", nil), // nil connectorTypes = enabled for all
		}, []string{"totp"})

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		redirectURL, ok := s.trySessionLogin(ctx, r, w, &authReq)
		require.True(t, ok)
		assert.Contains(t, redirectURL, "/mfa/totp", "should redirect to MFA page")
		assert.Contains(t, redirectURL, "req="+authReq.ID, "redirect should include auth request ID")
		assert.Contains(t, redirectURL, "authenticator=totp", "redirect should include authenticator ID")

		// MFAValidated should NOT be set.
		updated, err := s.storage.GetAuthRequest(ctx, authReq.ID)
		require.NoError(t, err)
		assert.False(t, updated.MFAValidated, "MFAValidated should be false when MFA is required")
		// LoggedIn should still be set even though MFA is pending.
		assert.True(t, updated.LoggedIn, "LoggedIn should be true even when MFA is pending")
	})

	t.Run("MFA provider not enabled for connector type skips MFA", func(t *testing.T) {
		// TOTP provider only enabled for "oidc" connectors, but our connector is "ldap".
		s, authReq := setupMFAFixture(t, map[string]MFAProvider{
			"totp": NewTOTPProvider("test-issuer", []string{"oidc"}),
		}, []string{"totp"})
		require.NoError(t, s.storage.UpdateAuthRequest(ctx, authReq.ID, func(a storage.AuthRequest) (storage.AuthRequest, error) {
			a.ForceApprovalPrompt = true
			return a, nil
		}))
		authReq.ForceApprovalPrompt = true

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		redirectURL, ok := s.trySessionLogin(ctx, r, w, &authReq)
		require.True(t, ok)
		assert.Contains(t, redirectURL, "/approval")
	})
}

// TestPromptNone tests the prompt=none silent authentication scenarios.
// These verify the code paths in handleConnectorLogin (handlers.go:444-457)
// where prompt=none requires session-based login without any UI.
func TestPromptNone(t *testing.T) {
	ctx := t.Context()

	t.Run("valid session with consent issues code silently", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = false
		authReq := setupSessionLoginFixture(t, s)
		// Fixture already sets up Consents: {"client-1": {"openid", "email"}}
		// and authReq.Scopes = {"openid", "email"} — consent is satisfied.

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		session := s.getValidAuthSession(ctx, w, r, &authReq)
		require.NotNil(t, session)

		redirectURL, ok := s.trySessionLoginWithSession(ctx, r, w, &authReq, session)
		require.True(t, ok, "session login should succeed")
		assert.Empty(t, redirectURL, "should return empty URL when code is issued directly (silent auth)")
	})

	t.Run("valid session without consent returns approval URL", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = false
		now := s.now()

		require.NoError(t, s.storage.CreateAuthSession(ctx, storage.AuthSession{
			UserID:      "user-1",
			ConnectorID: "mock",
			Nonce:       "test-nonce",
			ClientStates: map[string]*storage.ClientAuthState{
				"client-1": {Active: true, ExpiresAt: now.Add(24 * time.Hour), LastActivity: now.Add(-1 * time.Minute)},
			},
			CreatedAt:      now.Add(-30 * time.Minute),
			LastActivity:   now.Add(-1 * time.Minute),
			AbsoluteExpiry: now.Add(24 * time.Hour),
			IdleExpiry:     now.Add(59 * time.Minute),
		}))
		require.NoError(t, s.storage.CreateUserIdentity(ctx, storage.UserIdentity{
			UserID:      "user-1",
			ConnectorID: "mock",
			Claims:      storage.Claims{UserID: "user-1", Username: "testuser", Email: "test@example.com"},
			Consents:    map[string][]string{}, // No consent for any client.
			CreatedAt:   now.Add(-1 * time.Hour),
			LastLogin:   now.Add(-30 * time.Minute),
		}))

		authReq := storage.AuthRequest{
			ID:          storage.NewID(),
			ClientID:    "client-1",
			ConnectorID: "mock",
			Scopes:      []string{"openid", "email"},
			RedirectURI: "http://localhost/callback",
			MaxAge:      -1,
			HMACKey:     storage.NewHMACKey(crypto.SHA256),
			Expiry:      now.Add(10 * time.Minute),
		}
		require.NoError(t, s.storage.CreateAuthRequest(ctx, authReq))

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		session := s.getValidAuthSession(ctx, w, r, &authReq)
		require.NotNil(t, session)

		// In handleConnectorLogin, a non-empty redirectURL with prompt=none
		// triggers errInteractionRequired ("Consent required").
		redirectURL, ok := s.trySessionLoginWithSession(ctx, r, w, &authReq, session)
		require.True(t, ok, "session login should succeed (user is authenticated)")
		assert.Contains(t, redirectURL, "/approval", "should return approval URL when consent is missing")
	})

	t.Run("no session returns false", func(t *testing.T) {
		s := newTestSessionServer(t)
		authReq := storage.AuthRequest{ConnectorID: "mock"}
		r := httptest.NewRequest(http.MethodGet, "/", nil) // No cookie.
		w := httptest.NewRecorder()

		// In handleConnectorLogin, this triggers errLoginRequired.
		_, ok := s.trySessionLogin(ctx, r, w, &authReq)
		assert.False(t, ok, "should fail without session")
	})

	t.Run("SSO available issues code silently", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = true
		now := s.now()

		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID: "client-a", Secret: "secret", Name: "A", SSOSharedWith: []string{"client-b"},
		}))

		require.NoError(t, s.storage.CreateAuthSession(ctx, storage.AuthSession{
			UserID: "user-1", ConnectorID: "mock", Nonce: "test-nonce",
			ClientStates: map[string]*storage.ClientAuthState{
				"client-a": {Active: true, ExpiresAt: now.Add(24 * time.Hour), LastActivity: now.Add(-1 * time.Minute)},
			},
			CreatedAt: now.Add(-30 * time.Minute), LastActivity: now.Add(-1 * time.Minute),
			AbsoluteExpiry: now.Add(24 * time.Hour), IdleExpiry: now.Add(59 * time.Minute),
		}))
		require.NoError(t, s.storage.CreateUserIdentity(ctx, storage.UserIdentity{
			UserID: "user-1", ConnectorID: "mock",
			Claims:    storage.Claims{UserID: "user-1", Username: "testuser", Email: "test@example.com"},
			Consents:  map[string][]string{},
			CreatedAt: now.Add(-1 * time.Hour), LastLogin: now.Add(-30 * time.Minute),
		}))

		authReq := storage.AuthRequest{
			ID: storage.NewID(), ClientID: "client-b", ConnectorID: "mock",
			Scopes: []string{"openid"}, RedirectURI: "http://localhost/callback",
			MaxAge: -1, HMACKey: storage.NewHMACKey(crypto.SHA256), Expiry: now.Add(10 * time.Minute),
		}
		require.NoError(t, s.storage.CreateAuthRequest(ctx, authReq))

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		session := s.getValidAuthSession(ctx, w, r, &authReq)
		require.NotNil(t, session)

		redirectURL, ok := s.trySessionLoginWithSession(ctx, r, w, &authReq, session)
		require.True(t, ok, "SSO silent login should succeed")
		assert.Empty(t, redirectURL, "should issue code silently via SSO (skipApproval=true, openid-only)")

		// Verify SSO created a new client state.
		updated, err := s.storage.GetAuthSession(ctx, "user-1", "mock")
		require.NoError(t, err)
		assert.Contains(t, updated.ClientStates, "client-b", "SSO should create client state for target")
	})

	t.Run("MFA required returns redirect not silent", func(t *testing.T) {
		// This is the prompt=none + MFA case: finishSessionLogin returns MFA redirect URL.
		// In handleConnectorLogin, this is a successful (ok=true) redirect, not errLoginRequired.
		s := newTestSessionServer(t)
		s.skipApproval = true
		s.mfaProviders = map[string]MFAProvider{
			"totp": NewTOTPProvider("test-issuer", nil),
		}

		require.NoError(t, s.storage.CreateConnector(ctx, storage.Connector{
			ID: "mock", Type: "ldap", Name: "Mock", ResourceVersion: "1",
		}))
		s.mu.Lock()
		s.connectors = map[string]Connector{"mock": {Type: "ldap", ResourceVersion: "1"}}
		s.mu.Unlock()
		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID: "client-1", Secret: "secret", Name: "Test", MFAChain: []string{"totp"},
		}))

		authReq := setupSessionLoginFixture(t, s)

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		redirectURL, ok := s.trySessionLogin(ctx, r, w, &authReq)
		require.True(t, ok)
		assert.Contains(t, redirectURL, "/mfa/totp", "prompt=none with MFA should redirect to MFA page")
	})
}

// TestPromptConsent tests that prompt=consent forces the approval screen
// even when consent is already given.
func TestPromptConsent(t *testing.T) {
	ctx := t.Context()

	t.Run("ForceApprovalPrompt overrides existing consent in session login", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = false
		authReq := setupSessionLoginFixture(t, s)

		// Set ForceApprovalPrompt (set by prompt=consent in parseAuthorizationRequest).
		require.NoError(t, s.storage.UpdateAuthRequest(ctx, authReq.ID, func(a storage.AuthRequest) (storage.AuthRequest, error) {
			a.ForceApprovalPrompt = true
			return a, nil
		}))
		authReq.ForceApprovalPrompt = true

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		redirectURL, ok := s.trySessionLogin(ctx, r, w, &authReq)
		require.True(t, ok)
		assert.Contains(t, redirectURL, "/approval", "should show approval even though consent exists")
	})

	t.Run("login+consent parsed correctly", func(t *testing.T) {
		prompt, err := ParsePrompt("login consent")
		require.NoError(t, err)
		assert.True(t, prompt.Login(), "login flag should be set")
		assert.True(t, prompt.Consent(), "consent flag should be set")
	})
}

// TestSSO_ConsentAndMFA tests SSO interactions with consent and MFA.
func TestSSO_ConsentAndMFA(t *testing.T) {
	ctx := t.Context()

	// setupSSOFixture creates a two-client SSO scenario where client-a shares with client-b.
	setupSSOFixture := func(t *testing.T, s *Server, consentsForB []string) storage.AuthRequest {
		t.Helper()
		now := s.now()

		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID: "client-a", Secret: "secret", Name: "A", SSOSharedWith: []string{"client-b"},
		}))

		require.NoError(t, s.storage.CreateAuthSession(ctx, storage.AuthSession{
			UserID: "user-1", ConnectorID: "mock", Nonce: "test-nonce",
			ClientStates: map[string]*storage.ClientAuthState{
				"client-a": {Active: true, ExpiresAt: now.Add(24 * time.Hour), LastActivity: now.Add(-1 * time.Minute)},
			},
			CreatedAt: now.Add(-30 * time.Minute), LastActivity: now.Add(-1 * time.Minute),
			AbsoluteExpiry: now.Add(24 * time.Hour), IdleExpiry: now.Add(59 * time.Minute),
		}))

		consents := map[string][]string{}
		if len(consentsForB) > 0 {
			consents["client-b"] = consentsForB
		}
		require.NoError(t, s.storage.CreateUserIdentity(ctx, storage.UserIdentity{
			UserID: "user-1", ConnectorID: "mock",
			Claims:    storage.Claims{UserID: "user-1", Username: "testuser", Email: "test@example.com"},
			Consents:  consents,
			CreatedAt: now.Add(-1 * time.Hour), LastLogin: now.Add(-30 * time.Minute),
		}))

		authReq := storage.AuthRequest{
			ID: storage.NewID(), ClientID: "client-b", ConnectorID: "mock",
			Scopes: []string{"openid", "email"}, RedirectURI: "http://localhost/callback",
			MaxAge: -1, HMACKey: storage.NewHMACKey(crypto.SHA256), Expiry: now.Add(10 * time.Minute),
		}
		require.NoError(t, s.storage.CreateAuthRequest(ctx, authReq))
		return authReq
	}

	t.Run("SSO without consent for target shows approval", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = false
		authReq := setupSSOFixture(t, s, nil) // No consent for client-b.

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		session := s.getValidAuthSession(ctx, w, r, &authReq)
		require.NotNil(t, session)

		redirectURL, ok := s.trySessionLoginWithSession(ctx, r, w, &authReq, session)
		require.True(t, ok, "SSO login should succeed")
		assert.Contains(t, redirectURL, "/approval", "should show approval when target client has no consent")
	})

	t.Run("SSO with consent for target skips approval", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = false
		authReq := setupSSOFixture(t, s, []string{"openid", "email"})

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		session := s.getValidAuthSession(ctx, w, r, &authReq)
		require.NotNil(t, session)

		redirectURL, ok := s.trySessionLoginWithSession(ctx, r, w, &authReq, session)
		require.True(t, ok, "SSO login should succeed")
		assert.Empty(t, redirectURL, "should skip approval when consent exists for target client")
	})

	t.Run("SSO with MFA required on target client redirects to MFA", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = true
		s.mfaProviders = map[string]MFAProvider{
			"totp": NewTOTPProvider("test-issuer", nil),
		}

		require.NoError(t, s.storage.CreateConnector(ctx, storage.Connector{
			ID: "mock", Type: "ldap", Name: "Mock", ResourceVersion: "1",
		}))
		s.mu.Lock()
		s.connectors = map[string]Connector{"mock": {Type: "ldap", ResourceVersion: "1"}}
		s.mu.Unlock()

		// client-b requires MFA.
		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID: "client-b", Secret: "secret", Name: "B", MFAChain: []string{"totp"},
		}))

		authReq := setupSSOFixture(t, s, []string{"openid", "email"})

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		session := s.getValidAuthSession(ctx, w, r, &authReq)
		require.NotNil(t, session)

		redirectURL, ok := s.trySessionLoginWithSession(ctx, r, w, &authReq, session)
		require.True(t, ok)
		assert.Contains(t, redirectURL, "/mfa/totp", "SSO to MFA-requiring client should redirect to MFA")
	})

	t.Run("SSO source without MFA target with MFA enforces MFA", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = true
		s.mfaProviders = map[string]MFAProvider{
			"totp": NewTOTPProvider("test-issuer", nil),
		}

		require.NoError(t, s.storage.CreateConnector(ctx, storage.Connector{
			ID: "mock", Type: "ldap", Name: "Mock", ResourceVersion: "1",
		}))
		s.mu.Lock()
		s.connectors = map[string]Connector{"mock": {Type: "ldap", ResourceVersion: "1"}}
		s.mu.Unlock()

		// client-a has NO MFA, client-b requires MFA.
		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID: "client-a", Secret: "secret", Name: "A", SSOSharedWith: []string{"client-b"},
			MFAChain: []string{}, // Explicitly no MFA.
		}))
		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID: "client-b", Secret: "secret", Name: "B",
			MFAChain: []string{"totp"},
		}))

		now := s.now()
		require.NoError(t, s.storage.CreateAuthSession(ctx, storage.AuthSession{
			UserID: "user-1", ConnectorID: "mock", Nonce: "test-nonce",
			ClientStates: map[string]*storage.ClientAuthState{
				"client-a": {Active: true, ExpiresAt: now.Add(24 * time.Hour), LastActivity: now.Add(-1 * time.Minute)},
			},
			CreatedAt: now.Add(-30 * time.Minute), LastActivity: now.Add(-1 * time.Minute),
			AbsoluteExpiry: now.Add(24 * time.Hour), IdleExpiry: now.Add(59 * time.Minute),
		}))
		require.NoError(t, s.storage.CreateUserIdentity(ctx, storage.UserIdentity{
			UserID: "user-1", ConnectorID: "mock",
			Claims:    storage.Claims{UserID: "user-1", Username: "testuser", Email: "test@example.com"},
			Consents:  map[string][]string{},
			CreatedAt: now.Add(-1 * time.Hour), LastLogin: now.Add(-30 * time.Minute),
		}))

		authReq := storage.AuthRequest{
			ID: storage.NewID(), ClientID: "client-b", ConnectorID: "mock",
			Scopes: []string{"openid"}, RedirectURI: "http://localhost/callback",
			MaxAge: -1, HMACKey: storage.NewHMACKey(crypto.SHA256), Expiry: now.Add(10 * time.Minute),
		}
		require.NoError(t, s.storage.CreateAuthRequest(ctx, authReq))

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		session := s.getValidAuthSession(ctx, w, r, &authReq)
		require.NotNil(t, session)

		redirectURL, ok := s.trySessionLoginWithSession(ctx, r, w, &authReq, session)
		require.True(t, ok)
		assert.Contains(t, redirectURL, "/mfa/totp",
			"SSO from no-MFA source to MFA-requiring target must enforce MFA")
	})
}

// TestUpdateSessionTokenIssuedAt tests session activity tracking
// when tokens are issued via sendCodeResponse (handlers.go:1016).
func TestUpdateSessionTokenIssuedAt(t *testing.T) {
	ctx := t.Context()

	t.Run("updates session fields for correct client", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()

		require.NoError(t, s.storage.CreateAuthSession(ctx, storage.AuthSession{
			UserID: "user-1", ConnectorID: "mock", Nonce: "test-nonce",
			ClientStates: map[string]*storage.ClientAuthState{
				"client-1": {Active: true, ExpiresAt: now.Add(24 * time.Hour), LastActivity: now.Add(-10 * time.Minute)},
				"client-2": {Active: true, ExpiresAt: now.Add(24 * time.Hour), LastActivity: now.Add(-10 * time.Minute)},
			},
			CreatedAt: now.Add(-1 * time.Hour), LastActivity: now.Add(-10 * time.Minute),
			AbsoluteExpiry: now.Add(24 * time.Hour), IdleExpiry: now.Add(50 * time.Minute),
		}))

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		s.updateSessionTokenIssuedAt(r, "client-1")

		session, err := s.storage.GetAuthSession(ctx, "user-1", "mock")
		require.NoError(t, err)

		assert.Equal(t, now, session.LastActivity, "session LastActivity should be updated")
		assert.Equal(t, now.Add(s.sessionConfig.ValidIfNotUsedFor), session.IdleExpiry, "IdleExpiry should be extended")
		assert.Equal(t, now, session.ClientStates["client-1"].LastTokenIssuedAt, "client-1 LastTokenIssuedAt should be set")
		assert.Equal(t, now, session.ClientStates["client-1"].LastActivity, "client-1 LastActivity should be updated")
		// client-2 should be untouched.
		assert.Equal(t, now.Add(-10*time.Minute), session.ClientStates["client-2"].LastActivity,
			"client-2 should not be affected")
	})

	t.Run("noop when sessions disabled", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.sessionConfig = nil

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		// Should not panic.
		s.updateSessionTokenIssuedAt(r, "any-client")
	})
}

// TestIdleExpiryExtension verifies that session activity pushes
// IdleExpiry forward, preventing premature session expiration.
func TestIdleExpiryExtension(t *testing.T) {
	ctx := t.Context()

	t.Run("createOrUpdateAuthSession extends IdleExpiry", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()

		// Create an existing session with IdleExpiry close to now.
		require.NoError(t, s.storage.CreateAuthSession(ctx, storage.AuthSession{
			UserID: "user-1", ConnectorID: "mock", Nonce: "test-nonce",
			ClientStates:   map[string]*storage.ClientAuthState{},
			CreatedAt:      now.Add(-50 * time.Minute),
			LastActivity:   now.Add(-50 * time.Minute),
			AbsoluteExpiry: now.Add(24 * time.Hour),
			IdleExpiry:     now.Add(10 * time.Minute), // Only 10 minutes left.
		}))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		authReq := storage.AuthRequest{ClientID: "client-1", ConnectorID: "mock",
			Claims: storage.Claims{UserID: "user-1"}}

		err := s.createOrUpdateAuthSession(ctx, r, w, authReq, false)
		require.NoError(t, err)

		session, err := s.storage.GetAuthSession(ctx, "user-1", "mock")
		require.NoError(t, err)
		assert.Equal(t, now.Add(s.sessionConfig.ValidIfNotUsedFor), session.IdleExpiry,
			"IdleExpiry should be reset to now + ValidIfNotUsedFor")
	})

	t.Run("finishSessionLogin extends IdleExpiry", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = true
		now := s.now()

		require.NoError(t, s.storage.CreateAuthSession(ctx, storage.AuthSession{
			UserID: "user-1", ConnectorID: "mock", Nonce: "test-nonce",
			ClientStates: map[string]*storage.ClientAuthState{
				"client-1": {Active: true, ExpiresAt: now.Add(24 * time.Hour), LastActivity: now.Add(-50 * time.Minute)},
			},
			CreatedAt: now.Add(-50 * time.Minute), LastActivity: now.Add(-50 * time.Minute),
			AbsoluteExpiry: now.Add(24 * time.Hour),
			IdleExpiry:     now.Add(10 * time.Minute), // About to expire.
		}))
		require.NoError(t, s.storage.CreateUserIdentity(ctx, storage.UserIdentity{
			UserID: "user-1", ConnectorID: "mock",
			Claims:    storage.Claims{UserID: "user-1", Username: "testuser", Email: "test@example.com"},
			Consents:  map[string][]string{},
			CreatedAt: now.Add(-1 * time.Hour), LastLogin: now.Add(-50 * time.Minute),
		}))

		authReq := storage.AuthRequest{
			ID: storage.NewID(), ClientID: "client-1", ConnectorID: "mock",
			Scopes: []string{"openid"}, RedirectURI: "http://localhost/callback",
			MaxAge: -1, HMACKey: storage.NewHMACKey(crypto.SHA256), Expiry: now.Add(10 * time.Minute),
		}
		require.NoError(t, s.storage.CreateAuthRequest(ctx, authReq))

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()

		session := s.getValidAuthSession(ctx, w, r, &authReq)
		require.NotNil(t, session)

		_, ok := s.trySessionLoginWithSession(ctx, r, w, &authReq, session)
		require.True(t, ok)

		updated, err := s.storage.GetAuthSession(ctx, "user-1", "mock")
		require.NoError(t, err)
		assert.Equal(t, now.Add(s.sessionConfig.ValidIfNotUsedFor), updated.IdleExpiry,
			"IdleExpiry should be extended after session login")
	})
}

// TestSSO_Unidirectional verifies that SSO sharing is one-way:
// A sharing with B does NOT mean B shares with A.
func TestSSO_Unidirectional(t *testing.T) {
	ctx := t.Context()

	setup := func(t *testing.T, s *Server, loginClient, targetClient string) (storage.AuthRequest, *storage.AuthSession) {
		t.Helper()
		now := s.now()

		require.NoError(t, s.storage.CreateAuthSession(ctx, storage.AuthSession{
			UserID: "user-1", ConnectorID: "mock", Nonce: "test-nonce",
			ClientStates: map[string]*storage.ClientAuthState{
				loginClient: {Active: true, ExpiresAt: now.Add(24 * time.Hour), LastActivity: now.Add(-1 * time.Minute)},
			},
			CreatedAt: now.Add(-30 * time.Minute), LastActivity: now.Add(-1 * time.Minute),
			AbsoluteExpiry: now.Add(24 * time.Hour), IdleExpiry: now.Add(59 * time.Minute),
		}))
		require.NoError(t, s.storage.CreateUserIdentity(ctx, storage.UserIdentity{
			UserID: "user-1", ConnectorID: "mock",
			Claims:    storage.Claims{UserID: "user-1", Username: "testuser", Email: "test@example.com"},
			Consents:  map[string][]string{},
			CreatedAt: now.Add(-1 * time.Hour), LastLogin: now.Add(-30 * time.Minute),
		}))

		authReq := storage.AuthRequest{
			ID: storage.NewID(), ClientID: targetClient, ConnectorID: "mock",
			Scopes: []string{"openid"}, RedirectURI: "http://localhost/callback",
			MaxAge: -1, HMACKey: storage.NewHMACKey(crypto.SHA256), Expiry: now.Add(10 * time.Minute),
		}
		require.NoError(t, s.storage.CreateAuthRequest(ctx, authReq))

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()
		session := s.getValidAuthSession(ctx, w, r, &authReq)
		return authReq, session
	}

	t.Run("A shares with B, login A request B succeeds", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = true

		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID: "client-a", Secret: "s", Name: "A", SSOSharedWith: []string{"client-b"},
		}))
		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID: "client-b", Secret: "s", Name: "B", SSOSharedWith: []string{}, // Does NOT share back.
		}))

		authReq, session := setup(t, s, "client-a", "client-b")
		require.NotNil(t, session)

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()
		_, ok := s.trySessionLoginWithSession(ctx, r, w, &authReq, session)
		assert.True(t, ok, "A→B SSO should succeed")
	})

	t.Run("B does not share with A, login B request A fails", func(t *testing.T) {
		s := newTestSessionServer(t)
		s.skipApproval = true

		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID: "client-a", Secret: "s", Name: "A", SSOSharedWith: []string{"client-b"},
		}))
		require.NoError(t, s.storage.CreateClient(ctx, storage.Client{
			ID: "client-b", Secret: "s", Name: "B", SSOSharedWith: []string{}, // Does NOT share.
		}))

		authReq, session := setup(t, s, "client-b", "client-a")
		require.NotNil(t, session)

		r := sessionCookieRequest("user-1", "mock", "test-nonce")
		w := httptest.NewRecorder()
		_, ok := s.trySessionLoginWithSession(ctx, r, w, &authReq, session)
		assert.False(t, ok, "B→A SSO should fail because B does not share with A")
	})
}

// TestRememberMeDefault tests that the rememberMeDefault helper
// returns the correct value based on session configuration.
func TestRememberMeDefault(t *testing.T) {
	t.Run("sessions disabled returns nil", func(t *testing.T) {
		s := &Server{sessionConfig: nil}
		assert.Nil(t, s.rememberMeDefault())
	})

	t.Run("default false", func(t *testing.T) {
		s := &Server{sessionConfig: &SessionConfig{RememberMeCheckedByDefault: false}}
		v := s.rememberMeDefault()
		require.NotNil(t, v)
		assert.False(t, *v)
	})

	t.Run("default true", func(t *testing.T) {
		s := &Server{sessionConfig: &SessionConfig{RememberMeCheckedByDefault: true}}
		v := s.rememberMeDefault()
		require.NotNil(t, v)
		assert.True(t, *v)
	})
}
