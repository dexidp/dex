package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/dexidp/dex/connector"
)

func TestConfig_Open(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectErr bool
	}{
		{
			name: "valid_config",
			config: Config{
				Users: map[string]UserConfig{
					"testuser": {
						Keys: []string{"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample testuser@example"},
						UserInfo: UserInfo{
							Username: "testuser",
							Email:    "test@example.com",
							Groups:   []string{"admin"},
						},
					},
				},
				AllowedIssuers: []string{"test-issuer"},
			},
			expectErr: false,
		},
		{
			name: "empty_config",
			config: Config{
				Users: map[string]UserConfig{},
			},
			expectErr: false, // Empty config is valid
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn, err := tc.config.Open("ssh", slog.Default())

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, conn)

				// Cast to SSH connector to check internal state
				sshConn, ok := conn.(*SSHConnector)
				require.True(t, ok)
				require.NotNil(t, sshConn.logger)

				// Check that defaults are applied
				require.Equal(t, 3600, sshConn.config.TokenTTL) // Default TTL
			}
		})
	}
}

func TestSSHConnector_LoginURL(t *testing.T) {
	config := Config{}
	conn, err := config.Open("ssh", slog.Default())
	require.NoError(t, err)

	sshConn := conn.(*SSHConnector)

	// LoginURL should return a URL with SSH auth parameters
	loginURL, err := sshConn.LoginURL(connector.Scopes{}, "redirectURI", "state")
	require.NoError(t, err)
	require.Contains(t, loginURL, "ssh_auth=true")
	require.Contains(t, loginURL, "state=state")
}

func TestSSHConnector_HandleCallback(t *testing.T) {
	config := Config{}
	conn, err := config.Open("ssh", slog.Default())
	require.NoError(t, err)

	sshConn := conn.(*SSHConnector)

	// Create a minimal HTTP request to avoid nil pointer
	req := httptest.NewRequest("GET", "/callback", nil)

	identity, err := sshConn.HandleCallback(connector.Scopes{}, req)
	require.Error(t, err)
	require.Equal(t, connector.Identity{}, identity)
	require.Contains(t, err.Error(), "no SSH JWT or authorization code provided")
}

func TestValidateJWTClaims(t *testing.T) {
	config := Config{
		AllowedIssuers: []string{"test-issuer", "another-issuer"},
	}
	conn, err := config.Open("ssh", slog.Default())
	require.NoError(t, err)

	sshConn := conn.(*SSHConnector)

	tests := []struct {
		name      string
		claims    jwt.MapClaims
		expectSub string
		expectIss string
		expectErr bool
	}{
		{
			name: "valid_claims",
			claims: jwt.MapClaims{
				"sub": "testuser",
				"iss": "test-issuer",
				"aud": "kubernetes",
				"exp": float64(time.Now().Add(time.Hour).Unix()),
				"iat": float64(time.Now().Unix()),
				"jti": "unique-token-id",
			},
			expectSub: "testuser",
			expectIss: "test-issuer",
			expectErr: false,
		},
		{
			name: "missing_sub",
			claims: jwt.MapClaims{
				"iss": "test-issuer",
				"aud": "kubernetes",
				"exp": float64(time.Now().Add(time.Hour).Unix()),
			},
			expectErr: true,
		},
		{
			name: "expired_token",
			claims: jwt.MapClaims{
				"sub": "testuser",
				"iss": "test-issuer",
				"aud": "kubernetes",
				"exp": float64(time.Now().Add(-time.Hour).Unix()), // Expired
				"iat": float64(time.Now().Add(-2 * time.Hour).Unix()),
			},
			expectErr: true,
		},
		{
			name: "invalid_issuer",
			claims: jwt.MapClaims{
				"sub": "testuser",
				"iss": "invalid-issuer",
				"aud": "kubernetes",
				"exp": float64(time.Now().Add(time.Hour).Unix()),
			},
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sub, iss, err := sshConn.validateJWTClaims(tc.claims)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectSub, sub)
				require.Equal(t, tc.expectIss, iss)
			}
		})
	}
}

func TestFindUserByUsernameAndKey(t *testing.T) {
	// Generate test key pair
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	pubKey, err := ssh.NewPublicKey(privKey.Public().(ed25519.PublicKey))
	require.NoError(t, err)

	fingerprint := ssh.FingerprintSHA256(pubKey)
	pubKeyString := string(ssh.MarshalAuthorizedKey(pubKey))

	config := Config{
		Users: map[string]UserConfig{
			"testuser": {
				Keys: []string{
					strings.TrimSpace(pubKeyString), // Full public key format only
				},
				UserInfo: UserInfo{
					Username: "testuser",
					Email:    "test@example.com",
					Groups:   []string{"admin", "developer"},
				},
			},
		},
	}

	conn, err := config.Open("ssh", slog.Default())
	require.NoError(t, err)

	sshConn := conn.(*SSHConnector)

	tests := []struct {
		name        string
		username    string
		fingerprint string
		expectUser  *UserInfo
		expectErr   bool
	}{
		{
			name:        "valid_user_with_public_key",
			username:    "testuser",
			fingerprint: fingerprint,
			expectUser: &UserInfo{
				Username: "testuser",
				Email:    "test@example.com",
				Groups:   []string{"admin", "developer"},
			},
			expectErr: false,
		},
		{
			name:        "user_not_found",
			username:    "nonexistent",
			fingerprint: fingerprint,
			expectErr:   true,
		},
		{
			name:        "key_not_authorized_for_user",
			username:    "testuser",
			fingerprint: "SHA256:unauthorized-key",
			expectErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			userInfo, err := sshConn.findUserByUsernameAndKey(tc.username, tc.fingerprint)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectUser.Username, userInfo.Username)
				require.Equal(t, tc.expectUser.Email, userInfo.Email)
				require.Equal(t, tc.expectUser.Groups, userInfo.Groups)
			}
		})
	}
}

func TestIsKeyMatch(t *testing.T) {
	// Generate test key pair
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	pubKey, err := ssh.NewPublicKey(privKey.Public().(ed25519.PublicKey))
	require.NoError(t, err)

	expectedFingerprint := ssh.FingerprintSHA256(pubKey)
	pubKeyString := string(ssh.MarshalAuthorizedKey(pubKey))

	config := Config{}
	conn, err := config.Open("ssh", slog.Default())
	require.NoError(t, err)

	sshConn := conn.(*SSHConnector)

	tests := []struct {
		name                 string
		authorizedKey        string
		presentedFingerprint string
		expectMatch          bool
	}{
		{
			name:                 "public_key_matches_fingerprint",
			authorizedKey:        strings.TrimSpace(pubKeyString),
			presentedFingerprint: expectedFingerprint,
			expectMatch:          true,
		},
		{
			name:                 "no_match_different_keys",
			authorizedKey:        "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDifferentKeyData",
			presentedFingerprint: expectedFingerprint,
			expectMatch:          false,
		},
		{
			name:                 "invalid_public_key_format",
			authorizedKey:        "invalid-key-format",
			presentedFingerprint: expectedFingerprint,
			expectMatch:          false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := sshConn.isKeyMatch(tc.authorizedKey, tc.presentedFingerprint)
			require.Equal(t, tc.expectMatch, result)
		})
	}
}

func TestIsAllowedIssuer(t *testing.T) {
	config := Config{
		AllowedIssuers: []string{"allowed-issuer-1", "allowed-issuer-2"},
	}

	conn, err := config.Open("ssh", slog.Default())
	require.NoError(t, err)

	sshConn := conn.(*SSHConnector)

	tests := []struct {
		name     string
		issuer   string
		expected bool
	}{
		{
			name:     "allowed_issuer_1",
			issuer:   "allowed-issuer-1",
			expected: true,
		},
		{
			name:     "not_allowed_issuer",
			issuer:   "not-allowed-issuer",
			expected: false,
		},
		{
			name:     "empty_issuer",
			issuer:   "",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := sshConn.isAllowedIssuer(tc.issuer)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestTokenIdentity_Integration(t *testing.T) {
	t.Skip("Skipping complex integration test - requires real SSH JWT from kubectl-ssh-oidc client")

	// This integration test would require a real SSH JWT token created by kubectl-ssh-oidc
	// which involves SSH agent interaction and proper JWT signing with SSH keys.
	// For unit testing purposes, we test the individual components instead.
}

// TestSecurityFix_RejectsUnauthorizedKeys verifies that the security vulnerability is fixed.
// Previously, anyone could create a JWT with any public key in the claims and have it accepted.
// Now, only keys configured in Dex are accepted for verification.
func TestSecurityFix_RejectsUnauthorizedKeys(t *testing.T) {
	// Generate an authorized key for the test
	_, authorizedPrivKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	authorizedPubKey, err := ssh.NewPublicKey(authorizedPrivKey.Public().(ed25519.PublicKey))
	require.NoError(t, err)

	authorizedKeyStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(authorizedPubKey)))

	config := Config{
		Users: map[string]UserConfig{
			"testuser": {
				Keys: []string{authorizedKeyStr}, // Only the authorized key is configured
				UserInfo: UserInfo{
					Username: "testuser",
					Email:    "test@example.com",
				},
			},
		},
		AllowedIssuers: []string{"test-issuer"},
	}

	conn, err := config.Open("ssh", slog.Default())
	require.NoError(t, err)

	sshConn := conn.(*SSHConnector)

	// Test with a malicious JWT - this simulates an attacker trying to bypass auth
	// In the old vulnerable code, they could embed their own public key in the JWT claims
	maliciousJWT := "invalid.jwt.token"

	// Attempt authentication with unauthorized JWT should fail
	_, err = sshConn.validateSSHJWT(maliciousJWT)
	require.Error(t, err, "Authentication should fail with invalid JWT")

	// The error should indicate parsing failed, not that an embedded key was accepted
	require.Contains(t, err.Error(), "failed to parse JWT structure",
		"Error should indicate JWT parsing failed (no embedded keys accepted)")

	t.Log("✓ Security fix verified: malformed JWTs are rejected")

	// Test with a well-formed but unauthorized JWT (no valid signature from configured keys)
	maliciousJWT2 := "eyJhbGciOiJTU0giLCJ0eXAiOiJKV1QifQ.eyJzdWIiOiJ0ZXN0dXNlciIsImlzcyI6InRlc3QtaXNzdWVyIiwiYXVkIjoia3ViZXJuZXRlcyIsImV4cCI6OTk5OTk5OTk5OSwiaWF0IjoxNjAwMDAwMDAwLCJuYmYiOjE2MDAwMDAwMDB9.fake-signature"

	_, err = sshConn.validateSSHJWT(maliciousJWT2)
	require.Error(t, err, "Authentication should fail with unauthorized signature")
	require.Contains(t, err.Error(), "no configured key could verify",
		"Error should indicate no configured key could verify the JWT")

	t.Log("✓ Security fix verified: only configured keys can verify JWTs")
}

// Benchmark tests
func BenchmarkFindUserByUsernameAndKey(b *testing.B) {
	// Generate test keys
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(b, err)

	pubKey, err := ssh.NewPublicKey(privKey.Public().(ed25519.PublicKey))
	require.NoError(b, err)

	fingerprint := ssh.FingerprintSHA256(pubKey)

	// Create config with many users
	config := Config{
		Users: make(map[string]UserConfig),
	}

	for i := 0; i < 100; i++ {
		username := "user" + string(rune('0'+i%10)) + string(rune('0'+i/10))
		config.Users[username] = UserConfig{
			Keys: []string{"SHA256:key" + string(rune('0'+i%10)) + string(rune('0'+i/10))},
			UserInfo: UserInfo{
				Username: username,
				Email:    username + "@example.com",
			},
		}
	}

	// Add our test user
	config.Users["testuser"] = UserConfig{
		Keys: []string{fingerprint},
		UserInfo: UserInfo{
			Username: "testuser",
			Email:    "test@example.com",
		},
	}

	conn, err := config.Open("ssh", slog.Default())
	require.NoError(b, err)

	sshConn := conn.(*SSHConnector)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := sshConn.findUserByUsernameAndKey("testuser", fingerprint)
		if err != nil {
			b.Fatal(err)
		}
	}
}
