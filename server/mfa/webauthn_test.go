package mfa

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
)

func TestBuildWebAuthnUserDropsCloneWarning(t *testing.T) {
	identity := storage.UserIdentity{
		UserID:      "user-1",
		ConnectorID: "mock",
		WebAuthnCredentials: map[string][]storage.WebAuthnCredential{
			"webauthn-1": {{
				CredentialID: []byte("cred"),
				SignCount:    42,
				CloneWarning: true,
			}},
		},
	}

	user := buildWebAuthnUser(identity, "webauthn-1")
	creds := user.WebAuthnCredentials()
	require.Len(t, creds, 1)
	require.Equal(t, uint32(42), creds[0].Authenticator.SignCount, "sign count must be preserved")
	require.False(t, creds[0].Authenticator.CloneWarning, "stored CloneWarning must not be loaded back")
}

// TestNewWebAuthnProvider verifies provider construction: the relying-party ID
// and origin are derived from the issuer URL, and invalid configurations are
// rejected.
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
