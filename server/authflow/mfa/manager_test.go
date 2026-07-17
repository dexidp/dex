package mfa

import (
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
)

func totpOpts() totp.ValidateOpts {
	return totp.ValidateOpts{Period: totpPeriod, Skew: totpSkew, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1}
}

// TestValidateTOTPCode covers replay protection: a code is single-use per
// time-step, while a code from a later step is still accepted.
func TestValidateTOTPCode(t *testing.T) {
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "dex", AccountName: "user@example.com"})
	require.NoError(t, err)
	secret := key.Secret()

	now := time.Unix(1700000000, 0)
	code, err := totp.GenerateCodeCustom(secret, now, totpOpts())
	require.NoError(t, err)

	// First use is accepted and reports the matched counter.
	ok, counter := validateTOTPCode(secret, code, now, 0)
	require.True(t, ok)
	require.Equal(t, now.Unix()/totpPeriod, counter)

	// Replaying the same code with that counter recorded is rejected.
	ok, _ = validateTOTPCode(secret, code, now, counter)
	require.False(t, ok, "replayed code must be rejected")

	// A wrong code is rejected.
	ok, _ = validateTOTPCode(secret, "000000", now, 0)
	require.False(t, ok)

	// A code from the next step is accepted and advances the counter.
	next := now.Add(totpPeriod * time.Second)
	nextCode, err := totp.GenerateCodeCustom(secret, next, totpOpts())
	require.NoError(t, err)
	ok, nextCounter := validateTOTPCode(secret, nextCode, next, counter)
	require.True(t, ok)
	require.Greater(t, nextCounter, counter)
}

// TestBuildWebAuthnUserDropsCloneWarning verifies the stored CloneWarning flag
// is not fed back into the credential, so a credential that once tripped clone
// detection is not permanently locked out (go-webauthn never clears the flag).
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

// TestFinalizeLoginBlockedAccount verifies that login finalization is refused
// while the user identity's BlockedUntil is in the future, and proceeds once it
// has elapsed.
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
