package authflow

import (
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/mfa"
	"github.com/dexidp/dex/server/session"
	"github.com/dexidp/dex/storage"
)

func TestCompleteMFAStep(t *testing.T) {
	httpServer, server := newTestHandler(t, func(c *testFlowConfig) {
		c.SessionConfig = &session.Config{AbsoluteLifetime: time.Hour, ValidIfNotUsedFor: time.Hour}

		provider, err := mfa.NewWebAuthnProvider("Test", "", nil, "", "",
			"http://127.0.0.1", nil)
		require.NoError(t, err)

		c.MFAProviders = map[string]mfa.Provider{
			"webauthn-1": provider,
			"webauthn-2": provider,
		}
		c.DefaultMFAChain = []string{"webauthn-1", "webauthn-2"}
	})
	defer httpServer.Close()

	ctx := t.Context()

	hmacKey := make([]byte, 32)
	_, err := rand.Read(hmacKey)
	require.NoError(t, err)

	authReq := storage.AuthRequest{
		ID:       "test-req-chain",
		ClientID: "example-app",
		Expiry:   time.Now().Add(time.Hour),
		HMACKey:  hmacKey,
		LoggedIn: true,
		Claims: storage.Claims{
			UserID: "user-1",
			Email:  "user@example.com",
		},
		ConnectorID: "mock",
	}
	require.NoError(t, server.storage.CreateAuthRequest(ctx, authReq))
	require.NoError(t, server.storage.CreateClient(ctx, storage.Client{
		ID:     "example-app",
		Secret: "secret",
	}))

	// Completing first step should redirect to second.
	redirectURL, err := server.mfa.CompleteStep(ctx, authReq, "webauthn-1")
	require.NoError(t, err)
	require.Contains(t, redirectURL, "/mfa/webauthn")
	require.Contains(t, redirectURL, "authenticator=webauthn-2")

	// Completing the last step should redirect to the flow dispatcher.
	redirectURL, err = server.mfa.CompleteStep(ctx, authReq, "webauthn-2")
	require.NoError(t, err)
	require.Contains(t, redirectURL, "/auth?")

	// Verify MFAValidated was set.
	updated, err := server.storage.GetAuthRequest(ctx, authReq.ID)
	require.NoError(t, err)
	require.True(t, updated.MFAValidated)
}

func TestWebAuthnHandlersMissingHMAC(t *testing.T) {
	httpServer, server := newTestHandler(t, func(c *testFlowConfig) {
		c.SessionConfig = &session.Config{AbsoluteLifetime: time.Hour, ValidIfNotUsedFor: time.Hour}

		provider, err := mfa.NewWebAuthnProvider("Test", "", nil, "", "",
			"http://127.0.0.1", nil)
		require.NoError(t, err)

		c.MFAProviders = map[string]mfa.Provider{"webauthn-1": provider}
		c.DefaultMFAChain = []string{"webauthn-1"}
	})
	defer httpServer.Close()

	endpoints := []string{
		"/mfa/webauthn/register/begin",
		"/mfa/webauthn/register/finish",
		"/mfa/webauthn/login/begin",
		"/mfa/webauthn/login/finish",
	}

	for _, ep := range endpoints {
		t.Run(ep, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, ep, nil)
			server.ServeHTTP(rr, req)
			// Should fail with unauthorized (no hmac).
			require.Equal(t, http.StatusUnauthorized, rr.Code)
		})
	}
}

func TestWebAuthnVerifyPageRender(t *testing.T) {
	httpServer, server := newTestHandler(t, func(c *testFlowConfig) {
		c.SessionConfig = &session.Config{AbsoluteLifetime: time.Hour, ValidIfNotUsedFor: time.Hour}

		provider, err := mfa.NewWebAuthnProvider("Test", "", nil, "", "",
			"http://127.0.0.1", nil)
		require.NoError(t, err)

		c.MFAProviders = map[string]mfa.Provider{
			"webauthn-1": provider,
		}
		c.DefaultMFAChain = []string{"webauthn-1"}
	})
	defer httpServer.Close()

	ctx := t.Context()

	hmacKey := make([]byte, 32)
	_, err := rand.Read(hmacKey)
	require.NoError(t, err)

	authReq := storage.AuthRequest{
		ID:       "test-webauthn-verify",
		ClientID: "example-app",
		Expiry:   time.Now().Add(time.Hour),
		HMACKey:  hmacKey,
		LoggedIn: true,
		Claims: storage.Claims{
			UserID: "user-1",
			Email:  "user@example.com",
		},
		ConnectorID: "mock",
	}
	require.NoError(t, server.storage.CreateAuthRequest(ctx, authReq))

	// Create user identity without WebAuthn credentials (enrollment mode).
	require.NoError(t, server.storage.CreateUserIdentity(ctx, storage.UserIdentity{
		UserID:              "user-1",
		ConnectorID:         "mock",
		Claims:              authReq.Claims,
		Consents:            map[string][]string{},
		MFASecrets:          map[string]*storage.MFASecret{},
		WebAuthnCredentials: map[string][]storage.WebAuthnCredential{},
		CreatedAt:           time.Now(),
		LastLogin:           time.Now(),
	}))

	// Generate HMAC for the request.
	hmacVal := internal.ComputeHMAC(hmacKey, authReq.ID, "webauthn-1")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/mfa/webauthn?req="+authReq.ID+"&hmac="+hmacVal+"&authenticator=webauthn-1", nil)
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, "Register security key")
	require.Contains(t, body, "startWebAuthn")
}
