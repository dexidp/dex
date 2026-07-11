package server

import (
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/connector"
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
func TestFinalizeLoginBlockedAccount(t *testing.T) {
	t.Setenv("DEX_SESSIONS_ENABLED", "true")

	httpServer, server := newTestServer(t, func(c *Config) {
		c.SessionConfig = &SessionConfig{AbsoluteLifetime: time.Hour, ValidIfNotUsedFor: time.Hour}
	})
	defer httpServer.Close()

	ctx := t.Context()

	ident := connector.Identity{UserID: "user-1", Email: "user@example.com"}
	authReq := storage.AuthRequest{
		ID:          "login-req",
		ClientID:    "example-app",
		Expiry:      time.Now().Add(time.Hour),
		ConnectorID: "mock",
	}
	require.NoError(t, server.storage.CreateAuthRequest(ctx, authReq))
	require.NoError(t, server.storage.CreateUserIdentity(ctx, storage.UserIdentity{
		UserID:              "user-1",
		ConnectorID:         "mock",
		Claims:              storage.Claims{UserID: "user-1", Email: "user@example.com"},
		Consents:            map[string][]string{},
		MFASecrets:          map[string]*storage.MFASecret{},
		WebAuthnCredentials: map[string][]storage.WebAuthnCredential{},
		CreatedAt:           time.Now(),
		LastLogin:           time.Now(),
		BlockedUntil:        time.Now().Add(time.Hour),
	}))

	// Blocked: finalizeLogin must reject without marking the request logged in.
	_, _, err := server.finalizeLogin(ctx, ident, authReq, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "locked")

	updated, err := server.storage.GetAuthRequest(ctx, authReq.ID)
	require.NoError(t, err)
	require.False(t, updated.LoggedIn, "blocked account must not be logged in")

	// Clear the block: login should now proceed.
	require.NoError(t, server.storage.UpdateUserIdentity(ctx, "user-1", "mock", func(u storage.UserIdentity) (storage.UserIdentity, error) {
		u.BlockedUntil = time.Time{}
		return u, nil
	}))
	_, _, err = server.finalizeLogin(ctx, ident, authReq, nil)
	require.NoError(t, err)
}
