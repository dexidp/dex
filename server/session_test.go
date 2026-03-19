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
	assert.Equal(t, sessionCookieValue("user1", "conn1", "nonce123"), c.Value)
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
			value := sessionCookieValue(tt.userID, tt.connectorID, tt.nonce)
			gotUser, gotConn, gotNonce, err := parseSessionCookie(value)
			require.NoError(t, err)
			assert.Equal(t, tt.userID, gotUser)
			assert.Equal(t, tt.connectorID, gotConn)
			assert.Equal(t, tt.nonce, gotNonce)
		})
	}
}

func TestParseSessionCookie_Invalid(t *testing.T) {
	//nolint:dogsled // only for tests
	_, _, _, err := parseSessionCookie("invalid")
	assert.Error(t, err)
	//nolint:dogsled // only for tests
	_, _, _, err = parseSessionCookie("a.b")
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
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("nouser", "noconn", "nonce")})
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
			UserID:       "user1",
			ConnectorID:  "conn1",
			Nonce:        nonce,
			ClientStates: map[string]*storage.ClientAuthState{},
			CreatedAt:    now.Add(-30 * time.Minute),
			LastActivity: now.Add(-5 * time.Minute),
			IPAddress:    "127.0.0.1",
			UserAgent:    "test",
		}
		require.NoError(t, s.storage.CreateAuthSession(ctx, session))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user1", "conn1", nonce)})

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
			UserID:       "user1",
			ConnectorID:  "ldap",
			Nonce:        nonce,
			ClientStates: map[string]*storage.ClientAuthState{},
			CreatedAt:    now.Add(-30 * time.Minute),
			LastActivity: now.Add(-5 * time.Minute),
			IPAddress:    "127.0.0.1",
			UserAgent:    "test",
		}
		require.NoError(t, s.storage.CreateAuthSession(ctx, session))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user1", "ldap", nonce)})

		githubReq := &storage.AuthRequest{ConnectorID: "github"}
		assert.Nil(t, s.getValidAuthSession(ctx, httptest.NewRecorder(), r, githubReq))
	})

	t.Run("nonce mismatch", func(t *testing.T) {
		s := newTestSessionServer(t)
		now := s.now()

		session := storage.AuthSession{
			UserID:       "user2",
			ConnectorID:  "conn2",
			Nonce:        "correct-nonce",
			ClientStates: map[string]*storage.ClientAuthState{},
			CreatedAt:    now.Add(-30 * time.Minute),
			LastActivity: now.Add(-5 * time.Minute),
			IPAddress:    "127.0.0.1",
			UserAgent:    "test",
		}
		require.NoError(t, s.storage.CreateAuthSession(ctx, session))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user2", "conn2", "wrong-nonce")})

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
			UserID:       "user3",
			ConnectorID:  "conn3",
			Nonce:        nonce,
			ClientStates: map[string]*storage.ClientAuthState{},
			CreatedAt:    now.Add(-25 * time.Hour),
			LastActivity: now.Add(-1 * time.Minute),
			IPAddress:    "127.0.0.1",
			UserAgent:    "test",
		}
		require.NoError(t, s.storage.CreateAuthSession(ctx, session))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user3", "conn3", nonce)})

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
			UserID:       "user4",
			ConnectorID:  "conn4",
			Nonce:        nonce,
			ClientStates: map[string]*storage.ClientAuthState{},
			CreatedAt:    now.Add(-2 * time.Hour),
			LastActivity: now.Add(-2 * time.Hour),
			IPAddress:    "127.0.0.1",
			UserAgent:    "test",
		}
		require.NoError(t, s.storage.CreateAuthSession(ctx, session))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user4", "conn4", nonce)})

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

		userID, connectorID, nonce, err := parseSessionCookie(cookies[0].Value)
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
			CreatedAt:    now.Add(-30 * time.Minute),
			LastActivity: now.Add(-10 * time.Minute),
			IPAddress:    "127.0.0.1",
			UserAgent:    "test",
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
		_, _, gotNonce, err := parseSessionCookie(cookies[0].Value)
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
		CreatedAt:    now.Add(-30 * time.Minute),
		LastActivity: now.Add(-1 * time.Minute),
		IPAddress:    "127.0.0.1",
		UserAgent:    "test",
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
	r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue(userID, connectorID, nonce)})
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
			CreatedAt:    now.Add(-2 * time.Hour),
			LastActivity: now.Add(-1 * time.Minute),
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
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user-1", "mock", "test-nonce")})
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
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user-1", "mock", "test-nonce")})
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
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user-1", "mock", "test-nonce")})
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
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user-1", "mock", "test-nonce")})
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
		r.AddCookie(&http.Cookie{Name: "dex_session", Value: sessionCookieValue("user-1", "mock", "test-nonce")})
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
