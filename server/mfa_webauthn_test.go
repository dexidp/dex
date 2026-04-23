package server

import (
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
)

func TestNewWebAuthnProvider(t *testing.T) {
	tests := []struct {
		name        string
		rpDisplay   string
		rpID        string
		rpOrigins   []string
		timeout     string
		issuerURL   string
		wantErr     bool
		errContains string
	}{
		{
			name:      "derives rpID and origin from issuer",
			rpDisplay: "Test App",
			issuerURL: "https://auth.example.com/dex",
		},
		{
			name:      "explicit rpID and origins",
			rpDisplay: "Test App",
			rpID:      "example.com",
			rpOrigins: []string{"https://auth.example.com"},
			issuerURL: "https://auth.example.com/dex",
		},
		{
			name:      "defaults rpDisplayName from hostname",
			issuerURL: "https://auth.example.com",
		},
		{
			name:      "invalid timeout",
			rpDisplay: "Test",
			timeout:   "not-a-duration",
			issuerURL: "https://auth.example.com",
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			provider, err := NewWebAuthnProvider(
				tc.rpDisplay, tc.rpID, tc.rpOrigins,
				"", tc.timeout, tc.issuerURL,
				nil,
			)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "WebAuthn", provider.Type())
		})
	}
}

func TestWebAuthnProviderConnectorTypeFiltering(t *testing.T) {
	provider, err := NewWebAuthnProvider("Test", "", nil, "", "",
		"https://example.com", []string{"ldap", "oidc"})
	require.NoError(t, err)

	require.True(t, provider.EnabledForConnectorType("ldap"))
	require.True(t, provider.EnabledForConnectorType("oidc"))
	require.False(t, provider.EnabledForConnectorType("saml"))

	// No filter — all types allowed.
	providerAll, err := NewWebAuthnProvider("Test", "", nil, "", "",
		"https://example.com", nil)
	require.NoError(t, err)
	require.True(t, providerAll.EnabledForConnectorType("anything"))
}

func TestBuildWebAuthnUser(t *testing.T) {
	identity := storage.UserIdentity{
		UserID:      "user-123",
		ConnectorID: "conn-1",
		Claims: storage.Claims{
			Email:             "user@example.com",
			PreferredUsername: "User Name",
		},
		WebAuthnCredentials: map[string][]storage.WebAuthnCredential{
			"webauthn-1": {
				{
					CredentialID:    []byte("cred-1"),
					PublicKey:       []byte("pk-1"),
					AttestationType: "none",
					AAGUID:          []byte("aaguid-1"),
					SignCount:       5,
					Transport:       []string{"usb"},
				},
			},
			"webauthn-2": {
				{
					CredentialID: []byte("cred-2"),
					PublicKey:    []byte("pk-2"),
				},
			},
		},
	}

	user := buildWebAuthnUser(identity, "webauthn-1")
	require.Equal(t, []byte("user-123|conn-1"), user.WebAuthnID())
	require.Equal(t, "user@example.com", user.WebAuthnName())
	require.Equal(t, "User Name", user.WebAuthnDisplayName())
	require.Len(t, user.WebAuthnCredentials(), 1)
	require.Equal(t, []byte("cred-1"), user.WebAuthnCredentials()[0].ID)
	require.Equal(t, uint32(5), user.WebAuthnCredentials()[0].Authenticator.SignCount)

	// Different authenticator — should get the other credential.
	user2 := buildWebAuthnUser(identity, "webauthn-2")
	require.Len(t, user2.WebAuthnCredentials(), 1)
	require.Equal(t, []byte("cred-2"), user2.WebAuthnCredentials()[0].ID)

	// Unknown authenticator — no credentials.
	user3 := buildWebAuthnUser(identity, "unknown")
	require.Empty(t, user3.WebAuthnCredentials())
}

func TestCompleteMFAStep(t *testing.T) {
	httpServer, server := newTestServer(t, func(c *Config) {
		c.SessionConfig = &SessionConfig{AbsoluteLifetime: time.Hour, ValidIfNotUsedFor: time.Hour}

		provider, err := NewWebAuthnProvider("Test", "", nil, "", "",
			"http://127.0.0.1", nil)
		require.NoError(t, err)

		c.MFAProviders = map[string]MFAProvider{
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
	redirectURL, err := server.completeMFAStep(ctx, authReq, "webauthn-1")
	require.NoError(t, err)
	require.Contains(t, redirectURL, "/mfa/webauthn")
	require.Contains(t, redirectURL, "authenticator=webauthn-2")

	// Completing second (last) step should redirect to approval.
	redirectURL, err = server.completeMFAStep(ctx, authReq, "webauthn-2")
	require.NoError(t, err)
	require.Contains(t, redirectURL, "/approval")

	// Verify MFAValidated was set.
	updated, err := server.storage.GetAuthRequest(ctx, authReq.ID)
	require.NoError(t, err)
	require.True(t, updated.MFAValidated)
}

func TestWebAuthnHandlersMissingHMAC(t *testing.T) {
	httpServer, server := newTestServer(t, func(c *Config) {
		c.SessionConfig = &SessionConfig{AbsoluteLifetime: time.Hour, ValidIfNotUsedFor: time.Hour}
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
	httpServer, server := newTestServer(t, func(c *Config) {
		c.SessionConfig = &SessionConfig{AbsoluteLifetime: time.Hour, ValidIfNotUsedFor: time.Hour}

		provider, err := NewWebAuthnProvider("Test", "", nil, "", "",
			"http://127.0.0.1", nil)
		require.NoError(t, err)

		c.MFAProviders = map[string]MFAProvider{
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
	hmacVal := computeHMAC(hmacKey, authReq.ID, "webauthn-1")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/mfa/webauthn?req="+authReq.ID+"&hmac="+hmacVal+"&authenticator=webauthn-1", nil)
	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, "Register security key")
	require.Contains(t, body, "startWebAuthn")
}
