package rememberme

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

// setupTestEnvironment creates a realistic test environment following dex patterns
func setupTestEnvironment(t *testing.T) (storage.Storage, storage.ActiveSessionStorage, *slog.Logger) {
	logger := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelError}))

	// Use dex's standard in-memory storage
	store := memory.New(logger)
	sessionStore := memory.NewSessionStore(logger)

	// Initialize with real keys like dex does
	ctx := context.Background()
	err := store.UpdateKeys(ctx, func(old storage.Keys) (storage.Keys, error) {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		signingKey := &jose.JSONWebKey{Key: key}
		signingKeyPub := &jose.JSONWebKey{Key: &key.PublicKey}

		return storage.Keys{
			SigningKey:    signingKey,
			SigningKeyPub: signingKeyPub,
		}, nil
	})
	require.NoError(t, err)

	return store, sessionStore, logger
}

// createTestRequest creates an HTTP request with optional cookie
func createTestRequest(connectorName string, cookieValue string) *http.Request {
	req := httptest.NewRequest("GET", "/auth", nil)
	if cookieValue != "" {
		cookieName := connector_cookie_name(connectorName)
		req.AddCookie(&http.Cookie{Name: cookieName, Value: cookieValue})
	}
	return req
}

// createTestIdentity creates a sample identity for testing
func createTestIdentity() connector.Identity {
	return connector.Identity{
		UserID:            "user123",
		Username:          "testuser",
		PreferredUsername: "testuser",
		Email:             "test@example.com",
		EmailVerified:     true,
		Groups:            []string{"group1", "group2"},
	}
}

func TestHandleRememberMe_Integration(t *testing.T) {
	store, sessionStore, logger := setupTestEnvironment(t)
	ctx := context.Background()
	connectorName := "test-connector"
	expiryDuration := 24 * time.Hour
	identity := createTestIdentity()

	tests := []struct {
		name  string
		setup func() (*http.Request, AuthContext)
		want  func(t *testing.T, result *RememberMeCtx, err error)
	}{
		{
			name: "no cookie with anonymous context returns ErrNotFound",
			setup: func() (*http.Request, AuthContext) {
				req := createTestRequest(connectorName, "")
				authCtx := NewAnonymousAuthContext(connectorName, expiryDuration)
				return req, authCtx
			},
			want: func(t *testing.T, result *RememberMeCtx, err error) {
				require.Error(t, err)
				require.True(t, errors.Is(err, storage.ErrNotFound))
				require.Nil(t, result)
			},
		},
		{
			name: "no cookie with identity creates new session and sets cookie",
			setup: func() (*http.Request, AuthContext) {
				req := createTestRequest(connectorName, "")
				authCtx := NewAuthContextWithIdentity(connectorName, identity, expiryDuration)
				return req, authCtx
			},
			want: func(t *testing.T, result *RememberMeCtx, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.True(t, result.IsValid())

				// Verify session details
				require.Equal(t, identity.UserID, result.Session.Identity.UserID)
				require.Equal(t, identity.Email, result.Session.Identity.Email)
				require.Equal(t, identity.Groups, result.Session.Identity.Groups)
				require.True(t, result.Session.Expiry.After(time.Now()))

				// Verify cookie is set
				require.False(t, result.Cookie.Empty())
				cookie, unset := result.Cookie.Get()
				require.False(t, unset)
				require.Equal(t, connector_cookie_name(connectorName), cookie.Name)
				require.NotEmpty(t, cookie.Value)
				require.True(t, cookie.Secure)
				require.True(t, cookie.HttpOnly)
				require.Equal(t, http.SameSiteStrictMode, cookie.SameSite)

				// Verify session was stored
				storedSession, err := sessionStore.GetSession(ctx, cookie.Value)
				require.NoError(t, err)
				require.Equal(t, identity.UserID, storedSession.Identity.UserID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, authCtx := tt.setup()
			result, err := HandleRememberMe(ctx, logger, req, authCtx, store, sessionStore)
			tt.want(t, result, err)
		})
	}
}

func TestHandleRememberMe_WithExistingSessions(t *testing.T) {
	store, sessionStore, logger := setupTestEnvironment(t)
	ctx := context.Background()
	connectorName := "test-connector"
	expiryDuration := 24 * time.Hour
	identity := createTestIdentity()

	// First create a session to test retrieval
	req := createTestRequest(connectorName, "")
	authCtx := NewAuthContextWithIdentity(connectorName, identity, expiryDuration)
	initialResult, err := HandleRememberMe(ctx, logger, req, authCtx, store, sessionStore)
	require.NoError(t, err)
	require.NotNil(t, initialResult)

	cookie, _ := initialResult.Cookie.Get()
	cookieValue := cookie.Value

	t.Run("valid cookie with active session returns session without new cookie", func(t *testing.T) {
		req := createTestRequest(connectorName, cookieValue)
		authCtx := NewAnonymousAuthContext(connectorName, expiryDuration)

		result, err := HandleRememberMe(ctx, logger, req, authCtx, store, sessionStore)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.True(t, result.IsValid())
		require.Equal(t, identity.UserID, result.Session.Identity.UserID)
		require.True(t, result.Cookie.Empty()) // No cookie change needed
	})

	t.Run("expired session unsets cookie", func(t *testing.T) {
		// Create a fresh session store to avoid ID conflicts
		freshStore, freshSessionStore, freshLogger := setupTestEnvironment(t)

		expiredIdentity := connector.Identity{
			UserID:   "expired-user",
			Username: "expireduser",
			Email:    "expired@example.com",
			Groups:   []string{"expired-group"},
		}

		// Create an expired session directly with a known identifier
		expiredSession := storage.ActiveSession{
			Identity: expiredIdentity,
			Expiry:   time.Now().Add(-time.Hour),
		}

		// Create the session using a predictable signed identifier
		// First get a signed identifier by creating a valid session
		tempReq := createTestRequest(connectorName, "")
		tempAuthCtx := NewAuthContextWithIdentity(connectorName, expiredIdentity, time.Hour) // short duration
		tempResult, err := HandleRememberMe(ctx, freshLogger, tempReq, tempAuthCtx, freshStore, freshSessionStore)
		require.NoError(t, err)

		tempCookie, _ := tempResult.Cookie.Get()
		sessionID := tempCookie.Value

		// Wait a moment and directly update the session in storage to be expired
		// by creating a new session store and directly setting expired session
		testSessionStore := memory.NewSessionStore(freshLogger)
		err = testSessionStore.CreateSession(ctx, sessionID, expiredSession)
		require.NoError(t, err)

		// Test with the expired session
		req := createTestRequest(connectorName, sessionID)
		authCtx := NewAnonymousAuthContext(connectorName, expiryDuration)
		result, err := HandleRememberMe(ctx, freshLogger, req, authCtx, freshStore, testSessionStore)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.False(t, result.IsValid())

		// Cookie should be unset
		require.False(t, result.Cookie.Empty())
		resultCookie, unset := result.Cookie.Get()
		require.True(t, unset)
		require.Equal(t, -1, resultCookie.MaxAge)
	})

	t.Run("invalid cookie signature unsets cookie", func(t *testing.T) {
		// Use an invalid JWT format
		invalidCookie := "invalid.jwt.signature"
		req := createTestRequest(connectorName, invalidCookie)
		authCtx := NewAnonymousAuthContext(connectorName, expiryDuration)

		result, err := HandleRememberMe(ctx, logger, req, authCtx, store, sessionStore)
		require.Error(t, err)
		require.NotNil(t, result)
		require.False(t, result.IsValid())

		// Cookie should be unset
		require.False(t, result.Cookie.Empty())
		cookie, unset := result.Cookie.Get()
		require.True(t, unset)
		require.Equal(t, connector_cookie_name(connectorName), cookie.Name)
	})
}

func TestHandleRememberMe_EndToEndWorkflow(t *testing.T) {
	store, sessionStore, logger := setupTestEnvironment(t)
	ctx := context.Background()
	connectorName := "test-connector"
	expiryDuration := 24 * time.Hour
	identity := createTestIdentity()

	t.Run("complete login workflow", func(t *testing.T) {
		// Step 1: Initial login - no cookie present
		req1 := createTestRequest(connectorName, "")
		authCtx1 := NewAuthContextWithIdentity(connectorName, identity, expiryDuration)

		result1, err := HandleRememberMe(ctx, logger, req1, authCtx1, store, sessionStore)
		require.NoError(t, err)
		require.True(t, result1.IsValid())

		cookie1, unset1 := result1.Cookie.Get()
		require.False(t, unset1)
		require.NotEmpty(t, cookie1.Value)

		// Step 2: Return visit with cookie - should recognize user
		req2 := createTestRequest(connectorName, cookie1.Value)
		authCtx2 := NewAnonymousAuthContext(connectorName, expiryDuration)

		result2, err := HandleRememberMe(ctx, logger, req2, authCtx2, store, sessionStore)
		require.NoError(t, err)
		require.True(t, result2.IsValid())
		require.Equal(t, identity.UserID, result2.Session.Identity.UserID)
		require.True(t, result2.Cookie.Empty()) // No cookie change needed

		// Step 3: Verify session consistency
		require.Equal(t, result1.Session.Identity.UserID, result2.Session.Identity.UserID)
		require.Equal(t, result1.Session.Identity.Email, result2.Session.Identity.Email)
	})

	t.Run("multiple connectors isolation", func(t *testing.T) {
		connector1 := "connector-1"
		connector2 := "connector-2"

		// Create session for connector1
		req1 := createTestRequest(connector1, "")
		authCtx1 := NewAuthContextWithIdentity(connector1, identity, expiryDuration)
		result1, err := HandleRememberMe(ctx, logger, req1, authCtx1, store, sessionStore)
		require.NoError(t, err)

		cookie1, _ := result1.Cookie.Get()

		// Create a request for connector2 but with connector1's cookie
		// This simulates having both cookies in the browser
		req2 := httptest.NewRequest("GET", "/auth", nil)
		req2.AddCookie(cookie1) // connector1's cookie

		// Since connector2 looks for its own cookie name, it won't find connector1's cookie
		authCtx2 := NewAnonymousAuthContext(connector2, expiryDuration)
		result2, err := HandleRememberMe(ctx, logger, req2, authCtx2, store, sessionStore)
		require.Error(t, err)
		require.True(t, errors.Is(err, storage.ErrNotFound))
		require.Nil(t, result2)
	})
}

func TestHandleRememberMe_ErrorHandling(t *testing.T) {
	store, sessionStore, logger := setupTestEnvironment(t)
	ctx := context.Background()
	connectorName := "test-connector"
	expiryDuration := 24 * time.Hour
	identity := createTestIdentity()

	t.Run("session not found after valid signature verification", func(t *testing.T) {
		// Create a session first to get a valid signed cookie
		req := createTestRequest(connectorName, "")
		authCtx := NewAuthContextWithIdentity(connectorName, identity, expiryDuration)
		result, err := HandleRememberMe(ctx, logger, req, authCtx, store, sessionStore)
		require.NoError(t, err)

		cookie, _ := result.Cookie.Get()

		// Create a different session store that doesn't have this session
		emptySessionStore := memory.NewSessionStore(logger)

		// Try to retrieve with valid cookie but empty session store
		req2 := createTestRequest(connectorName, cookie.Value)
		authCtx2 := NewAnonymousAuthContext(connectorName, expiryDuration)
		result2, err := HandleRememberMe(ctx, logger, req2, authCtx2, store, emptySessionStore)

		require.NoError(t, err)
		require.NotNil(t, result2)
		require.False(t, result2.IsValid())

		// Should unset cookie when session not found
		require.False(t, result2.Cookie.Empty())
		unsetCookie, unset := result2.Cookie.Get()
		require.True(t, unset)
		require.Equal(t, connector_cookie_name(connectorName), unsetCookie.Name)
	})
}

func TestExtractCookie(t *testing.T) {
	connectorName := "test-connector"

	t.Run("cookie present", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/auth", nil)
		req.AddCookie(&http.Cookie{Name: connector_cookie_name(connectorName), Value: "test-value"})
		req.AddCookie(&http.Cookie{Name: "other-cookie", Value: "ignored"})

		value, found := extractCookie(req, connectorName)
		require.True(t, found)
		require.Equal(t, "test-value", value)
	})

	t.Run("cookie not present", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/auth", nil)
		value, found := extractCookie(req, connectorName)
		require.False(t, found)
		require.Equal(t, "", value)
	})
}

func TestConnectorCookieName(t *testing.T) {
	tests := []struct {
		connector string
		expected  string
	}{
		{"test", "dex_active_session_cookie_test"},
		{"google", "dex_active_session_cookie_google"},
		{"ldap-local", "dex_active_session_cookie_ldap-local"},
	}

	for _, tt := range tests {
		t.Run(tt.connector, func(t *testing.T) {
			result := connector_cookie_name(tt.connector)
			require.Equal(t, tt.expected, result)
		})
	}
}
