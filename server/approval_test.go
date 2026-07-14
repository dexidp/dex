package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
)

func TestSkipApprovalWithExistingConsent(t *testing.T) {
	ctx := t.Context()
	setSessionsEnabled(t, true)

	connID := "mock"
	authReqID := "test-consent-skip"
	expiry := time.Now().Add(100 * time.Second)

	tests := []struct {
		name        string
		consents    map[string][]string
		scopes      []string
		clientID    string
		forcePrompt bool
		wantPath    string
	}{
		{
			name:     "Existing consent covers requested scopes",
			consents: map[string][]string{"test": {"email", "profile"}},
			scopes:   []string{"openid", "email", "profile"},
			clientID: "test",
			wantPath: "/callback/cb",
		},
		{
			name:     "Existing consent missing a scope",
			consents: map[string][]string{"test": {"email"}},
			scopes:   []string{"openid", "email", "profile"},
			clientID: "test",
			wantPath: "/approval",
		},
		{
			name:        "Force approval overrides consent",
			consents:    map[string][]string{"test": {"email", "profile"}},
			scopes:      []string{"openid", "email", "profile"},
			clientID:    "test",
			forcePrompt: true,
			wantPath:    "/approval",
		},
		{
			name:     "No consent for this client",
			consents: map[string][]string{"other-client": {"email"}},
			scopes:   []string{"openid", "email"},
			clientID: "test",
			wantPath: "/approval",
		},
		{
			name:     "Only openid scope - skip with empty consent",
			consents: map[string][]string{"test": {}},
			scopes:   []string{"openid"},
			clientID: "test",
			wantPath: "/callback/cb",
		},
		{
			name:     "offline_access requires consent",
			consents: map[string][]string{"test": {}},
			scopes:   []string{"openid", "offline_access"},
			clientID: "test",
			wantPath: "/approval",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpServer, s := newTestServer(t, func(c *Config) {
				c.SkipApprovalScreen = false
				c.Now = time.Now
			})
			defer httpServer.Close()

			// Pre-create UserIdentity with consents
			require.NoError(t, s.storage.CreateUserIdentity(ctx, storage.UserIdentity{
				UserID:      "0-385-28089-0",
				ConnectorID: connID,
				Claims: storage.Claims{
					UserID:        "0-385-28089-0",
					Username:      "Kilgore Trout",
					Email:         "kilgore@kilgore.trout",
					EmailVerified: true,
				},
				Consents:  tc.consents,
				CreatedAt: time.Now(),
				LastLogin: time.Now(),
			}))

			authReq := storage.AuthRequest{
				ID:                  authReqID,
				ConnectorID:         connID,
				ClientID:            tc.clientID,
				RedirectURI:         "cb",
				Expiry:              expiry,
				ResponseTypes:       []string{responseTypeCode},
				Scopes:              tc.scopes,
				ForceApprovalPrompt: tc.forcePrompt,
			}
			require.NoError(t, s.storage.CreateAuthRequest(ctx, authReq))

			rr := httptest.NewRecorder()
			reqPath := fmt.Sprintf("/callback/%s?state=%s", connID, authReqID)
			s.handleConnectorCallback(rr, httptest.NewRequest("GET", reqPath, nil))

			require.Equal(t, 303, rr.Code)
			cb, err := url.Parse(rr.Result().Header.Get("Location"))
			require.NoError(t, err)
			require.Equal(t, tc.wantPath, cb.Path)
		})
	}
}

func TestConsentPersistedOnApproval(t *testing.T) {
	ctx := t.Context()
	setSessionsEnabled(t, true)

	httpServer, s := newTestServer(t, nil)
	defer httpServer.Close()

	userID := "test-user"
	connectorID := "mock"
	clientID := "test"

	// Pre-create UserIdentity (would have been created during login)
	require.NoError(t, s.storage.CreateUserIdentity(ctx, storage.UserIdentity{
		UserID:      userID,
		ConnectorID: connectorID,
		Claims:      storage.Claims{UserID: userID},
		Consents:    make(map[string][]string),
		CreatedAt:   time.Now(),
		LastLogin:   time.Now(),
	}))

	authReq := storage.AuthRequest{
		ID:            "approval-consent-test",
		ClientID:      clientID,
		ConnectorID:   connectorID,
		ResponseTypes: []string{responseTypeCode},
		RedirectURI:   "https://client.example/callback",
		Expiry:        time.Now().Add(time.Minute),
		LoggedIn:      true,
		Claims:        storage.Claims{UserID: userID},
		Scopes:        []string{"openid", "email", "profile"},
		HMACKey:       []byte("consent-test-key"),
	}
	require.NoError(t, s.storage.CreateAuthRequest(ctx, authReq))

	mac := computeHMAC(authReq.HMACKey, authReq.ID, "")

	form := url.Values{
		"approval": {"approve"},
		"req":      {authReq.ID},
		"hmac":     {mac},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/approval", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s.ServeHTTP(rr, req)

	require.Equal(t, http.StatusSeeOther, rr.Code, "approval should redirect")

	ui, err := s.storage.GetUserIdentity(ctx, userID, connectorID)
	require.NoError(t, err, "UserIdentity should exist")
	require.Equal(t, []string{"openid", "email", "profile"}, ui.Consents[clientID], "approved scopes should be persisted")
}

func TestScopesCoveredByConsent(t *testing.T) {
	tests := []struct {
		name      string
		approved  []string
		requested []string
		want      bool
	}{
		{
			name:      "All scopes covered",
			approved:  []string{"email", "profile"},
			requested: []string{"openid", "email", "profile"},
			want:      true,
		},
		{
			name:      "Missing scope",
			approved:  []string{"email"},
			requested: []string{"openid", "email", "groups"},
			want:      false,
		},
		{
			name:      "Only openid scope skipped",
			approved:  []string{},
			requested: []string{"openid"},
			want:      true,
		},
		{
			name:      "offline_access requires consent",
			approved:  []string{},
			requested: []string{"openid", "offline_access"},
			want:      false,
		},
		{
			name:      "offline_access covered by consent",
			approved:  []string{"offline_access"},
			requested: []string{"openid", "offline_access"},
			want:      true,
		},
		{
			name:      "Nil approved",
			approved:  nil,
			requested: []string{"email"},
			want:      false,
		},
		{
			name:      "Empty requested",
			approved:  []string{"email"},
			requested: []string{},
			want:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := scopesCoveredByConsent(tc.approved, tc.requested)
			require.Equal(t, tc.want, got)
		})
	}
}

// TestConsentSurvivesSessionDeletion verifies that UserIdentity.Consents
// persists independently from AuthSession lifecycle (logout should not
// clear consent decisions).
func TestConsentSurvivesSessionDeletion(t *testing.T) {
	ctx := t.Context()

	httpServer, s := newTestServerWithSessions(t, nil)
	defer httpServer.Close()

	userID := "test-user"
	connectorID := "mock"
	clientID := "test-client"

	// Create UserIdentity with existing consents.
	require.NoError(t, s.storage.CreateUserIdentity(ctx, storage.UserIdentity{
		UserID:      userID,
		ConnectorID: connectorID,
		Claims:      storage.Claims{UserID: userID, Username: "testuser"},
		Consents:    map[string][]string{clientID: {"openid", "email", "profile"}},
		CreatedAt:   time.Now(),
		LastLogin:   time.Now(),
	}))

	// Create and then delete the session (simulating logout).
	require.NoError(t, s.storage.CreateAuthSession(ctx, storage.AuthSession{
		UserID: userID, ConnectorID: connectorID, Nonce: "nonce",
		CreatedAt: time.Now(), LastActivity: time.Now(),
	}))
	require.NoError(t, s.storage.DeleteAuthSession(ctx, userID, connectorID))

	// Session is gone.
	_, err := s.storage.GetAuthSession(ctx, userID, connectorID)
	require.ErrorIs(t, err, storage.ErrNotFound)

	// Consent survives.
	ui, err := s.storage.GetUserIdentity(ctx, userID, connectorID)
	require.NoError(t, err)
	require.Equal(t, []string{"openid", "email", "profile"}, ui.Consents[clientID],
		"consent should survive session deletion")
}

// TestConsentIsolatedBetweenClients verifies that consent given for
// client-A does not satisfy scope check for client-B.
func TestConsentIsolatedBetweenClients(t *testing.T) {
	approvedForA := map[string][]string{"client-a": {"openid", "email"}}

	// client-b should not have consent.
	require.False(t, scopesCoveredByConsent(approvedForA["client-b"], []string{"openid", "email"}),
		"consent for client-a should not cover client-b")

	// client-a should have consent.
	require.True(t, scopesCoveredByConsent(approvedForA["client-a"], []string{"openid", "email"}),
		"consent for client-a should cover client-a's requested scopes")
}

type getAuthRequestErrorStorage struct {
	storage.Storage
	err error
}

func (s *getAuthRequestErrorStorage) GetAuthRequest(context.Context, string) (storage.AuthRequest, error) {
	return storage.AuthRequest{}, s.err
}

func TestHandleApprovalGetAuthRequestErrorGET(t *testing.T) {
	httpServer, server := newTestServer(t, func(c *Config) {
		c.Storage = &getAuthRequestErrorStorage{Storage: c.Storage, err: errors.New("storage unavailable")}
	})
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/approval?req=any&hmac=AQ", nil)

	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	require.Contains(t, rr.Body.String(), "Database error.")
}

func TestHandleApprovalGetAuthRequestNotFoundGET(t *testing.T) {
	httpServer, server := newTestServer(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/approval?req=does-not-exist&hmac=AQ", nil)

	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), "User session error.")
	require.NotContains(t, rr.Body.String(), "Database error.")
}

func TestHandleApprovalGetAuthRequestNotFoundPOST(t *testing.T) {
	httpServer, server := newTestServer(t, nil)
	defer httpServer.Close()

	body := strings.NewReader("approval=approve&req=does-not-exist&hmac=AQ")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/approval", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), "User session error.")
	require.NotContains(t, rr.Body.String(), "Database error.")
}

func TestHandleApprovalDoubleSubmitPOST(t *testing.T) {
	ctx := t.Context()
	httpServer, server := newTestServer(t, nil)
	defer httpServer.Close()

	authReq := storage.AuthRequest{
		ID:            "approval-double-submit",
		ClientID:      "test",
		ResponseTypes: []string{responseTypeCode},
		RedirectURI:   "https://client.example/callback",
		Expiry:        time.Now().Add(time.Minute),
		LoggedIn:      true,
		MFAValidated:  true,
		HMACKey:       []byte("approval-double-submit-key"),
	}
	require.NoError(t, server.storage.CreateAuthRequest(ctx, authReq))

	mac := computeHMAC(authReq.HMACKey, authReq.ID, "")

	form := url.Values{
		"approval": {"approve"},
		"req":      {authReq.ID},
		"hmac":     {mac},
	}

	firstRR := httptest.NewRecorder()
	firstReq := httptest.NewRequest(http.MethodPost, "/approval", strings.NewReader(form.Encode()))
	firstReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.ServeHTTP(firstRR, firstReq)

	require.Equal(t, http.StatusSeeOther, firstRR.Code)
	require.Contains(t, firstRR.Header().Get("Location"), "https://client.example/callback")

	secondRR := httptest.NewRecorder()
	secondReq := httptest.NewRequest(http.MethodPost, "/approval", strings.NewReader(form.Encode()))
	secondReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.ServeHTTP(secondRR, secondReq)

	require.Equal(t, http.StatusBadRequest, secondRR.Code)
	require.Contains(t, secondRR.Body.String(), "User session error.")
	require.NotContains(t, secondRR.Body.String(), "Database error.")
}
