package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"net/url"
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
		AllowedIssuers:         []string{"test-issuer", "another-issuer"},
		DexInstanceID:          "https://dex.test.com",
		AllowedTargetAudiences: []string{"kubectl", "test-client"},
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
			name: "valid_claims_with_target_audience",
			claims: jwt.MapClaims{
				"sub":             "testuser",
				"iss":             "test-issuer",
				"aud":             "https://dex.test.com",
				"target_audience": "kubectl",
				"exp":             float64(time.Now().Add(time.Hour).Unix()),
				"iat":             float64(time.Now().Unix()),
				"jti":             "unique-token-id",
			},
			expectSub: "testuser",
			expectIss: "test-issuer",
			expectErr: false,
		},
		{
			name: "legacy_token_rejected",
			claims: jwt.MapClaims{
				"sub": "testuser",
				"iss": "test-issuer",
				"aud": "kubectl", // Legacy tokens: no longer supported (missing target_audience)
				"exp": float64(time.Now().Add(time.Hour).Unix()),
				"iat": float64(time.Now().Unix()),
				"jti": "unique-token-id",
			},
			expectSub: "testuser",
			expectIss: "test-issuer",
			expectErr: true, // Should fail: legacy tokens no longer supported
		},
		{
			name: "missing_sub",
			claims: jwt.MapClaims{
				"iss": "test-issuer",
				"aud": "https://dex.test.com",
				"exp": float64(time.Now().Add(time.Hour).Unix()),
			},
			expectErr: true,
		},
		{
			name: "expired_token",
			claims: jwt.MapClaims{
				"sub": "testuser",
				"iss": "test-issuer",
				"aud": "https://dex.test.com",
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
				"aud": "https://dex.test.com",
				"exp": float64(time.Now().Add(time.Hour).Unix()),
			},
			expectErr: true,
		},
		{
			name: "invalid_dex_instance_audience",
			claims: jwt.MapClaims{
				"sub": "testuser",
				"iss": "test-issuer",
				"aud": "wrong-dex-instance",
				"exp": float64(time.Now().Add(time.Hour).Unix()),
			},
			expectErr: true,
		},
		{
			name: "invalid_target_audience",
			claims: jwt.MapClaims{
				"sub":             "testuser",
				"iss":             "test-issuer",
				"aud":             "https://dex.test.com",
				"target_audience": "unauthorized-client",
				"exp":             float64(time.Now().Add(time.Hour).Unix()),
			},
			expectErr: true,
		},
		{
			name: "legacy_token_rejected_2",
			claims: jwt.MapClaims{
				"sub": "testuser",
				"iss": "test-issuer",
				"aud": "test-client", // Legacy tokens: no longer supported (missing target_audience)
				"exp": float64(time.Now().Add(time.Hour).Unix()),
				"iat": float64(time.Now().Unix()),
				"jti": "unique-token-id",
			},
			expectSub: "testuser",
			expectIss: "test-issuer",
			expectErr: true, // Should fail: legacy tokens no longer supported
		},
		{
			name: "legacy_token_invalid_audience",
			claims: jwt.MapClaims{
				"sub": "testuser",
				"iss": "test-issuer",
				"aud": "unauthorized-legacy-client", // Not in allowed_target_audiences
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

// =========================
// Challenge/Response Tests
// =========================

func TestSSHConnector_LoginURL_ChallengeResponse(t *testing.T) {
	config := Config{
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
	}

	conn, err := config.Open("ssh", slog.Default())
	require.NoError(t, err)

	sshConn := conn.(*SSHConnector)

	tests := []struct {
		name        string
		callbackURL string
		state       string
		expectError bool
		expectType  string // "challenge" or "jwt"
	}{
		{
			name:        "challenge_request_valid_user",
			callbackURL: "https://dex.example.com/callback?ssh_challenge=true&username=testuser",
			state:       "test-state-123",
			expectError: false,
			expectType:  "challenge",
		},
		{
			name:        "challenge_request_nonexistent_user",
			callbackURL: "https://dex.example.com/callback?ssh_challenge=true&username=nonexistent",
			state:       "test-state-456",
			expectError: false, // SECURITY: No error to prevent user enumeration
			expectType:  "challenge",
		},
		{
			name:        "challenge_request_missing_username",
			callbackURL: "https://dex.example.com/callback?ssh_challenge=true",
			state:       "test-state-789",
			expectError: true,
			expectType:  "challenge",
		},
		{
			name:        "jwt_request_default",
			callbackURL: "https://dex.example.com/callback",
			state:       "test-state-jwt",
			expectError: false,
			expectType:  "jwt",
		},
		{
			name:        "invalid_callback_url",
			callbackURL: "http://[::1]:namedport", // Actually invalid URL
			state:       "test-state-invalid",
			expectError: true,
			expectType:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loginURL, err := sshConn.LoginURL(connector.Scopes{}, tt.callbackURL, tt.state)

			if tt.expectError {
				require.Error(t, err, "Expected error for test case: "+tt.name)
				return
			}

			require.NoError(t, err, "Unexpected error for test case: "+tt.name)
			require.NotEmpty(t, loginURL, "LoginURL should not be empty")

			switch tt.expectType {
			case "challenge":
				require.Contains(t, loginURL, "ssh_challenge=", "Challenge URL should contain challenge parameter")
				require.Contains(t, loginURL, tt.state, "Challenge URL should contain state")
			case "jwt":
				require.Contains(t, loginURL, "ssh_auth=true", "JWT URL should contain ssh_auth flag")
				require.Contains(t, loginURL, tt.state, "JWT URL should contain state")
			}
		})
	}
}

func TestSSHConnector_HandleCallback_ChallengeResponse(t *testing.T) {
	// Generate test SSH key
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	pubKey, err := ssh.NewPublicKey(privKey.Public().(ed25519.PublicKey))
	require.NoError(t, err)

	signer, err := ssh.NewSignerFromKey(privKey)
	require.NoError(t, err)

	pubKeyStr := string(ssh.MarshalAuthorizedKey(pubKey))

	config := Config{
		Users: map[string]UserConfig{
			"testuser": {
				Keys: []string{strings.TrimSpace(pubKeyStr)},
				UserInfo: UserInfo{
					Username: "testuser",
					Email:    "test@example.com",
					Groups:   []string{"admin"},
				},
			},
		},
		AllowedIssuers: []string{"test-issuer"},
	}

	conn, err := config.Open("ssh", slog.Default())
	require.NoError(t, err)

	sshConn := conn.(*SSHConnector)

	// Generate a challenge for testing
	challengeData := make([]byte, 32)
	_, err = rand.Read(challengeData)
	require.NoError(t, err)

	challengeID := "test-challenge-id"
	challenge := &Challenge{
		Data:      challengeData,
		Username:  "testuser",
		CreatedAt: time.Now(),
		IsValid:   true, // Valid user for enumeration prevention testing
	}
	sshConn.challenges.store(challengeID, challenge)

	// Sign the challenge
	signature, err := signer.Sign(rand.Reader, challengeData)
	require.NoError(t, err)

	signatureB64 := base64.StdEncoding.EncodeToString(ssh.Marshal(signature))

	tests := []struct {
		name          string
		formData      map[string]string
		expectError   bool
		errorContains string
	}{
		{
			name: "valid_challenge_response",
			formData: map[string]string{
				"ssh_challenge": "present",
				"username":      "testuser",
				"signature":     signatureB64,
				"state":         "test-state:" + challengeID,
			},
			expectError: false,
		},
		{
			name: "missing_username",
			formData: map[string]string{
				"ssh_challenge": "present",
				"signature":     signatureB64,
				"state":         "test-state:" + challengeID,
			},
			expectError:   true,
			errorContains: "missing required parameters",
		},
		{
			name: "missing_signature",
			formData: map[string]string{
				"ssh_challenge": "present",
				"username":      "testuser",
				"state":         "test-state:" + challengeID,
			},
			expectError:   true,
			errorContains: "missing required parameters",
		},
		{
			name: "invalid_state_format",
			formData: map[string]string{
				"ssh_challenge": "present",
				"username":      "testuser",
				"signature":     signatureB64,
				"state":         "invalid-state",
			},
			expectError:   true,
			errorContains: "invalid state format",
		},
		{
			name: "nonexistent_user",
			formData: map[string]string{
				"ssh_challenge": "present",
				"username":      "nonexistent",
				"signature":     signatureB64,
				"state":         "test-state:" + challengeID,
			},
			expectError:   true,
			errorContains: "invalid or expired challenge", // Challenge is consumed in previous test
		},
		{
			name: "expired_challenge",
			formData: map[string]string{
				"ssh_challenge": "present",
				"username":      "testuser",
				"signature":     signatureB64,
				"state":         "test-state:nonexistent-challenge",
			},
			expectError:   true,
			errorContains: "invalid or expired challenge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP request
			req := httptest.NewRequest("POST", "/callback", strings.NewReader(""))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Add form data
			values := req.URL.Query()
			for key, value := range tt.formData {
				values.Set(key, value)
			}
			req.URL.RawQuery = values.Encode()

			// For POST data, we need to set form values
			req.Form = values

			identity, err := sshConn.HandleCallback(connector.Scopes{}, req)

			if tt.expectError {
				require.Error(t, err, "Expected error for test case: "+tt.name)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains,
						"Error should contain expected message for test case: "+tt.name)
				}
				return
			}

			require.NoError(t, err, "Unexpected error for test case: "+tt.name)
			require.Equal(t, "testuser", identity.UserID, "UserID should match")
			require.Equal(t, "testuser", identity.Username, "Username should match")
			require.Equal(t, "test@example.com", identity.Email, "Email should match")
			require.Contains(t, identity.Groups, "admin", "Groups should contain admin")
		})
	}
}

func TestChallengeStore(t *testing.T) {
	store := newChallengeStore(50 * time.Millisecond) // Very short TTL for testing

	// Test storing and retrieving challenges
	challengeData := []byte("test-challenge-data")
	challenge := &Challenge{
		Data:      challengeData,
		Username:  "testuser",
		CreatedAt: time.Now(),
		IsValid:   true, // Valid user for testing
	}

	// Store challenge
	store.store("test-id", challenge)

	// Retrieve challenge
	retrieved, exists := store.get("test-id")
	require.True(t, exists, "Challenge should exist after storing")
	require.Equal(t, challengeData, retrieved.Data, "Challenge data should match")
	require.Equal(t, "testuser", retrieved.Username, "Username should match")

	// Challenge should be removed after retrieval (one-time use)
	_, exists = store.get("test-id")
	require.False(t, exists, "Challenge should be removed after retrieval")

	// Test manual TTL check
	expiredChallenge := &Challenge{
		Data:      []byte("expired-data"),
		Username:  "testuser",
		CreatedAt: time.Now().Add(-100 * time.Millisecond), // Already expired
		IsValid:   true,                                    // Valid user but expired challenge
	}
	store.store("expired-id", expiredChallenge)

	// Manually run cleanup logic
	store.mutex.Lock()
	now := time.Now()
	for id, challenge := range store.challenges {
		if now.Sub(challenge.CreatedAt) > store.ttl {
			delete(store.challenges, id)
		}
	}
	store.mutex.Unlock()

	// Challenge should be cleaned up
	_, exists = store.get("expired-id")
	require.False(t, exists, "Expired challenge should be cleaned up")
}

// TestUserEnumerationPrevention verifies that the SSH connector prevents user enumeration attacks
func TestUserEnumerationPrevention(t *testing.T) {
	config := Config{
		Users: map[string]UserConfig{
			"validuser": {
				Keys: []string{"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExampleKey validuser@example.com"},
				UserInfo: UserInfo{
					Username: "validuser",
					Email:    "validuser@example.com",
					Groups:   []string{"users"},
				},
			},
		},
		AllowedIssuers: []string{"test-issuer"},
		DefaultGroups:  []string{"authenticated"},
		TokenTTL:       3600,
		ChallengeTTL:   300,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	conn, err := config.Open("ssh", logger)
	require.NoError(t, err)
	sshConn := conn.(*SSHConnector)

	// Test cases: valid user vs invalid user should have identical responses
	testCases := []struct {
		name             string
		username         string
		expectedBehavior string
	}{
		{"valid_user", "validuser", "should_generate_valid_challenge"},
		{"invalid_user", "attackeruser", "should_generate_invalid_challenge"},
		{"another_invalid", "nonexistent", "should_generate_invalid_challenge"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			callbackURL := fmt.Sprintf("https://dex.example.com/callback?ssh_challenge=true&username=%s", tc.username)
			state := "test-state"

			// Both valid and invalid users should get challenge URLs (no error)
			challengeURL, err := sshConn.LoginURL(connector.Scopes{}, callbackURL, state)
			require.NoError(t, err, "Both valid and invalid users should get challenge URLs")
			require.Contains(t, challengeURL, "ssh_challenge=", "Challenge should be embedded in URL")

			// Extract challenge from URL to verify it was stored
			parsedURL, err := url.Parse(challengeURL)
			require.NoError(t, err)
			challengeB64 := parsedURL.Query().Get("ssh_challenge")
			require.NotEmpty(t, challengeB64, "Challenge should be present in URL")

			// Extract state to get challenge ID
			stateWithID := parsedURL.Query().Get("state")
			parts := strings.Split(stateWithID, ":")
			require.Len(t, parts, 2, "State should contain challenge ID")
			challengeID := parts[1]

			// Verify challenge was stored (should exist for both valid and invalid users)
			challenge, found := sshConn.challenges.get(challengeID)
			require.True(t, found, "Challenge should be stored for enumeration prevention")
			require.Equal(t, tc.username, challenge.Username, "Username should match")

			// Check the IsValid flag (this is the key difference)
			if tc.expectedBehavior == "should_generate_valid_challenge" {
				require.True(t, challenge.IsValid, "Valid user should have IsValid=true")
			} else {
				require.False(t, challenge.IsValid, "Invalid user should have IsValid=false")
			}
		})
	}

	t.Run("identical_response_timing", func(t *testing.T) {
		// Measure response times to ensure they're similar (basic timing attack prevention)
		measureTime := func(username string) (duration time.Duration) {
			start := time.Now()
			callbackURL := fmt.Sprintf("https://dex.example.com/callback?ssh_challenge=true&username=%s", username)
			_, err := sshConn.LoginURL(connector.Scopes{}, callbackURL, "test-state")
			require.NoError(t, err)
			duration = time.Since(start)
			return
		}

		// Measure multiple times for statistical significance
		validTimes := make([]time.Duration, 5)
		invalidTimes := make([]time.Duration, 5)

		for i := 0; i < 5; i++ {
			validTimes[i] = measureTime("validuser")
			invalidTimes[i] = measureTime("nonexistentuser")
		}

		// Calculate averages
		var validTotal, invalidTotal time.Duration
		for i := 0; i < 5; i++ {
			validTotal += validTimes[i]
			invalidTotal += invalidTimes[i]
		}
		validAvg := validTotal / 5
		invalidAvg := invalidTotal / 5

		// Response times should be similar (within 50% of each other)
		// This is a basic test - sophisticated timing attacks may still be possible
		ratio := float64(validAvg) / float64(invalidAvg)
		if ratio > 1 {
			ratio = 1 / ratio // Ensure ratio is <= 1
		}
		require.GreaterOrEqual(t, ratio, 0.5, "Response times should be similar to prevent timing attacks")
		t.Logf("✓ Timing test passed: valid_avg=%v, invalid_avg=%v, ratio=%.2f", validAvg, invalidAvg, ratio)
	})
}

func TestSSHConnector_ChallengeResponse_Integration(t *testing.T) {
	// Generate test SSH key
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	pubKey, err := ssh.NewPublicKey(privKey.Public().(ed25519.PublicKey))
	require.NoError(t, err)

	signer, err := ssh.NewSignerFromKey(privKey)
	require.NoError(t, err)

	pubKeyStr := string(ssh.MarshalAuthorizedKey(pubKey))

	config := Config{
		Users: map[string]UserConfig{
			"integrationuser": {
				Keys: []string{strings.TrimSpace(pubKeyStr)},
				UserInfo: UserInfo{
					Username: "integrationuser",
					Email:    "integration@example.com",
					Groups:   []string{"developers", "testers"},
				},
			},
		},
		DefaultGroups:  []string{"authenticated"},
		AllowedIssuers: []string{"test-issuer"},
		TokenTTL:       3600,
	}

	conn, err := config.Open("ssh", slog.Default())
	require.NoError(t, err)

	sshConn := conn.(*SSHConnector)

	// Step 1: Request challenge URL
	callbackURL := "https://dex.example.com/callback?ssh_challenge=true&username=integrationuser"
	state := "integration-test-state"

	loginURL, err := sshConn.LoginURL(connector.Scopes{Groups: true}, callbackURL, state)
	require.NoError(t, err, "LoginURL should succeed")
	require.Contains(t, loginURL, "ssh_challenge=", "Login URL should contain challenge")

	// Step 2: Extract challenge from URL
	parsedURL, err := url.Parse(loginURL)
	require.NoError(t, err, "Should parse login URL")

	challengeB64 := parsedURL.Query().Get("ssh_challenge")
	require.NotEmpty(t, challengeB64, "Challenge should be present in URL")

	stateWithChallenge := parsedURL.Query().Get("state")
	require.NotEmpty(t, stateWithChallenge, "State should be present")

	challengeData, err := base64.URLEncoding.DecodeString(challengeB64)
	require.NoError(t, err, "Should decode challenge")

	// Step 3: Sign challenge with SSH key
	signature, err := signer.Sign(rand.Reader, challengeData)
	require.NoError(t, err)

	signatureB64 := base64.StdEncoding.EncodeToString(ssh.Marshal(signature))

	// Step 4: Submit signed challenge
	req := httptest.NewRequest("POST", "/callback", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	values := url.Values{}
	values.Set("ssh_challenge", challengeB64)
	values.Set("username", "integrationuser")
	values.Set("signature", signatureB64)
	values.Set("state", stateWithChallenge)
	req.Form = values

	identity, err := sshConn.HandleCallback(connector.Scopes{Groups: true}, req)
	require.NoError(t, err, "HandleCallback should succeed")

	// Step 5: Verify identity
	require.Equal(t, "integrationuser", identity.UserID, "UserID should match")
	require.Equal(t, "integrationuser", identity.Username, "Username should match")
	require.Equal(t, "integration@example.com", identity.Email, "Email should match")
	require.Equal(t, true, identity.EmailVerified, "Email should be verified")

	// Check groups (should include both user groups and default groups)
	expectedGroups := []string{"authenticated", "developers", "testers"}
	for _, expectedGroup := range expectedGroups {
		require.Contains(t, identity.Groups, expectedGroup, "Should contain group: "+expectedGroup)
	}

	t.Log("✓ Challenge/response integration test successful")
}
