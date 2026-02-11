package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/ent"
	"github.com/dexidp/dex/storage/memory"
)

// sshSigningMethodTest implements jwt.SigningMethod for creating test SSH-signed JWTs.
type sshSigningMethodTest struct{}

func (m *sshSigningMethodTest) Alg() (algorithm string) {
	algorithm = "SSH"
	return algorithm
}

func (m *sshSigningMethodTest) Verify(signingString string, signature []byte, key interface{}) (err error) {
	err = fmt.Errorf("verify not used in test signing")
	return err
}

func (m *sshSigningMethodTest) Sign(signingString string, key interface{}) (signature []byte, err error) {
	signer, ok := key.(ssh.Signer)
	if !ok {
		err = fmt.Errorf("expected ssh.Signer, got %T", key)
		return signature, err
	}

	sig, signErr := signer.Sign(rand.Reader, []byte(signingString))
	if signErr != nil {
		err = fmt.Errorf("SSH signing failed: %w", signErr)
		return signature, err
	}

	// Encode just the blob as base64 — the server reconstructs the ssh.Signature
	encoded := base64.StdEncoding.EncodeToString(sig.Blob)
	signature = []byte(encoded)
	return signature, err
}

// generateTestSSHJWT creates a JWT signed with an SSH private key for testing.
// The JWT uses the dual-audience model: aud=dexInstanceID, target_audience=final audience.
func generateTestSSHJWT(t *testing.T, signer ssh.Signer, username, issuer, dexInstanceID, targetAudience string) (tokenString string) {
	t.Helper()

	signingMethod := &sshSigningMethodTest{}
	jwt.RegisterSigningMethod("SSH", func() (m jwt.SigningMethod) {
		m = signingMethod
		return m
	})

	now := time.Now()
	claims := jwt.MapClaims{
		"sub":             username,
		"iss":             issuer,
		"aud":             dexInstanceID,
		"target_audience": targetAudience,
		"exp":             now.Add(time.Hour).Unix(),
		"iat":             now.Unix(),
		"nbf":             now.Add(-time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(signingMethod, claims)
	var err error
	tokenString, err = token.SignedString(signer)
	require.NoError(t, err, "failed to sign test JWT")
	return tokenString
}

// sshConnectorJSON returns JSON config for the SSH connector with the given public key.
func sshConnectorJSON(t *testing.T, pubKeyStr, serverURL string) (configJSON []byte) {
	t.Helper()

	config := map[string]interface{}{
		"users": map[string]interface{}{
			"testuser": map[string]interface{}{
				"keys":     []string{strings.TrimSpace(pubKeyStr)},
				"username": "testuser",
				"email":    "testuser@example.com",
				"groups":   []string{"developers", "ssh-users"},
			},
		},
		"allowed_issuers":          []string{"test-ssh-client"},
		"dex_instance_id":          serverURL,
		"allowed_target_audiences": []string{"ssh-test-client", "kubectl"},
		"default_groups":           []string{"authenticated"},
		"token_ttl":                3600,
		"challenge_ttl":            300,
	}

	var err error
	configJSON, err = json.Marshal(config)
	require.NoError(t, err, "failed to marshal SSH connector config")
	return configJSON
}

// newTestServerWithStorage creates a test server using the provided storage backend.
// It registers an SSH connector and an OAuth2 client for token exchange testing.
func newTestServerWithStorage(
	t *testing.T,
	s storage.Storage,
	pubKeyStr string,
) (httpServer *httptest.Server, server *Server) {
	t.Helper()

	var srv *Server
	httpServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.ServeHTTP(w, r)
	}))

	logger := slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := t.Context()

	config := Config{
		Issuer:  httpServer.URL,
		Storage: s,
		Web: WebConfig{
			Dir: "../web",
		},
		Logger:             logger,
		PrometheusRegistry: prometheus.NewRegistry(),
		HealthChecker:      gosundheit.New(),
		SkipApprovalScreen: true,
		AllowedGrantTypes: []string{
			grantTypeAuthorizationCode,
			grantTypeRefreshToken,
			grantTypeTokenExchange,
		},
	}

	// Create SSH connector in storage
	connectorConfig := sshConnectorJSON(t, pubKeyStr, httpServer.URL)
	sshConn := storage.Connector{
		ID:              "ssh",
		Type:            "ssh",
		Name:            "SSH",
		ResourceVersion: "1",
		Config:          connectorConfig,
	}
	err := s.CreateConnector(ctx, sshConn)
	require.NoError(t, err, "failed to create SSH connector in storage")

	// Create OAuth2 client for token exchange
	err = s.CreateClient(ctx, storage.Client{
		ID:      "ssh-test-client",
		Secret:  "ssh-test-secret",
		Name:    "SSH Test Client",
		LogoURL: "https://example.com/logo.png",
	})
	require.NoError(t, err, "failed to create test client")

	srv, err = newServer(ctx, config, staticRotationStrategy(testKey))
	require.NoError(t, err, "failed to create server")

	srv.refreshTokenPolicy, err = NewRefreshTokenPolicy(logger, false, "", "", "")
	require.NoError(t, err, "failed to create refresh token policy")
	srv.refreshTokenPolicy.now = time.Now

	server = srv
	return httpServer, server
}

// doTokenExchange performs an RFC 8693 token exchange request against the server.
func doTokenExchange(
	t *testing.T,
	server *Server,
	serverURL string,
	subjectToken string,
	connectorID string,
	clientID string,
	clientSecret string,
	subjectTokenType string,
	requestedTokenType string,
	scope string,
	audience string,
) (rr *httptest.ResponseRecorder) {
	t.Helper()

	vals := make(url.Values)
	vals.Set("grant_type", grantTypeTokenExchange)
	setNonEmpty(vals, "connector_id", connectorID)
	setNonEmpty(vals, "scope", scope)
	setNonEmpty(vals, "requested_token_type", requestedTokenType)
	setNonEmpty(vals, "subject_token_type", subjectTokenType)
	setNonEmpty(vals, "subject_token", subjectToken)
	setNonEmpty(vals, "client_id", clientID)
	setNonEmpty(vals, "client_secret", clientSecret)
	setNonEmpty(vals, "audience", audience)

	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, serverURL+"/token", strings.NewReader(vals.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	server.handleToken(rr, req)
	return rr
}

// generateTestSSHKeyPair creates an ed25519 SSH key pair for testing.
func generateTestSSHKeyPair(t *testing.T) (pubKeyStr string, signer ssh.Signer) {
	t.Helper()

	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err, "failed to generate ed25519 key")

	pubKey, err := ssh.NewPublicKey(privKey.Public().(ed25519.PublicKey))
	require.NoError(t, err, "failed to create SSH public key")

	signer, err = ssh.NewSignerFromKey(privKey)
	require.NoError(t, err, "failed to create SSH signer")

	pubKeyStr = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pubKey)))
	return pubKeyStr, signer
}

// tokenExchangeSubtest defines a table-driven subtest for token exchange.
type tokenExchangeSubtest struct {
	name               string
	subjectTokenType   string
	requestedTokenType string
	scope              string
	audience           string
	connectorID        string
	useValidToken      bool
	useBadSignature    bool
	omitSubjectToken   bool
	expectedCode       int
	expectedTokenType  string
}

// standardTokenExchangeSubtests returns the common set of subtests run against each storage backend.
func standardTokenExchangeSubtests() (subtests []tokenExchangeSubtest) {
	subtests = []tokenExchangeSubtest{
		{
			name:               "access-token-exchange",
			subjectTokenType:   tokenTypeAccess,
			requestedTokenType: tokenTypeAccess,
			scope:              "openid",
			connectorID:        "ssh",
			useValidToken:      true,
			expectedCode:       http.StatusOK,
			expectedTokenType:  tokenTypeAccess,
		},
		{
			name:               "id-token-exchange",
			subjectTokenType:   tokenTypeID,
			requestedTokenType: tokenTypeID,
			scope:              "openid",
			connectorID:        "ssh",
			useValidToken:      true,
			expectedCode:       http.StatusOK,
			expectedTokenType:  tokenTypeID,
		},
		{
			name:               "default-token-type",
			subjectTokenType:   tokenTypeAccess,
			requestedTokenType: "",
			scope:              "openid",
			connectorID:        "ssh",
			useValidToken:      true,
			expectedCode:       http.StatusOK,
			expectedTokenType:  tokenTypeAccess,
		},
		{
			name:               "with-audience",
			subjectTokenType:   tokenTypeAccess,
			requestedTokenType: tokenTypeAccess,
			scope:              "openid",
			audience:           "kubectl",
			connectorID:        "ssh",
			useValidToken:      true,
			expectedCode:       http.StatusOK,
			expectedTokenType:  tokenTypeAccess,
		},
		{
			name:               "missing-subject-token",
			subjectTokenType:   tokenTypeAccess,
			requestedTokenType: tokenTypeAccess,
			scope:              "openid",
			connectorID:        "ssh",
			omitSubjectToken:   true,
			expectedCode:       http.StatusBadRequest,
		},
		{
			name:               "invalid-connector",
			subjectTokenType:   tokenTypeAccess,
			requestedTokenType: tokenTypeAccess,
			scope:              "openid",
			connectorID:        "nonexistent",
			useValidToken:      true,
			expectedCode:       http.StatusBadRequest,
		},
		{
			name:               "invalid-signature",
			subjectTokenType:   tokenTypeAccess,
			requestedTokenType: tokenTypeAccess,
			scope:              "openid",
			connectorID:        "ssh",
			useBadSignature:    true,
			expectedCode:       http.StatusUnauthorized,
		},
	}
	return subtests
}

// runTokenExchangeSubtests runs the standard set of token exchange subtests
// against a server backed by the given storage.
func runTokenExchangeSubtests(
	t *testing.T,
	s storage.Storage,
	pubKeyStr string,
	validSigner ssh.Signer,
	badSigner ssh.Signer,
) {
	t.Helper()

	httpServer, server := newTestServerWithStorage(t, s, pubKeyStr)
	defer httpServer.Close()

	for _, tc := range standardTokenExchangeSubtests() {
		t.Run(tc.name, func(t *testing.T) {
			var subjectToken string
			switch {
			case tc.omitSubjectToken:
				subjectToken = ""
			case tc.useBadSignature:
				subjectToken = generateTestSSHJWT(t, badSigner, "testuser", "test-ssh-client", httpServer.URL, "ssh-test-client")
			case tc.useValidToken:
				subjectToken = generateTestSSHJWT(t, validSigner, "testuser", "test-ssh-client", httpServer.URL, "ssh-test-client")
			}

			rr := doTokenExchange(
				t, server, httpServer.URL,
				subjectToken, tc.connectorID,
				"ssh-test-client", "ssh-test-secret",
				tc.subjectTokenType, tc.requestedTokenType,
				tc.scope, tc.audience,
			)

			require.Equal(t, tc.expectedCode, rr.Code, "unexpected status code: %s", rr.Body.String())
			require.Equal(t, "application/json", rr.Result().Header.Get("Content-Type"))

			if tc.expectedCode == http.StatusOK {
				var res accessTokenResponse
				err := json.NewDecoder(rr.Result().Body).Decode(&res)
				require.NoError(t, err, "failed to decode response")
				require.Equal(t, tc.expectedTokenType, res.IssuedTokenType)
				require.NotEmpty(t, res.AccessToken, "access_token should not be empty")
				require.Equal(t, "bearer", res.TokenType)
				require.Greater(t, res.ExpiresIn, 0, "expires_in should be positive")
			}
		})
	}
}

// TestTokenExchangeSSH_SQLite tests the full SSH token exchange flow using SQLite in-memory storage.
// This test always runs (no env vars required).
func TestTokenExchangeSSH_SQLite(t *testing.T) {
	pubKeyStr, validSigner := generateTestSSHKeyPair(t)
	_, badSigner := generateTestSSHKeyPair(t)

	logger := slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := ent.SQLite3{File: ":memory:"}
	s, err := cfg.Open(logger)
	require.NoError(t, err, "failed to open SQLite storage")

	runTokenExchangeSubtests(t, s, pubKeyStr, validSigner, badSigner)
}

// TestTokenExchangeSSH_Postgres tests the full SSH token exchange flow using PostgreSQL storage.
// Gated by DEX_POSTGRES_ENT_HOST environment variable.
func TestTokenExchangeSSH_Postgres(t *testing.T) {
	host := os.Getenv("DEX_POSTGRES_ENT_HOST")
	if host == "" {
		t.Skipf("test environment variable DEX_POSTGRES_ENT_HOST not set, skipping")
	}

	port := uint64(5432)
	if rawPort := os.Getenv("DEX_POSTGRES_ENT_PORT"); rawPort != "" {
		var parseErr error
		port, parseErr = strconv.ParseUint(rawPort, 10, 32)
		require.NoError(t, parseErr, "invalid postgres port %q", rawPort)
	}

	logger := slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := ent.Postgres{
		NetworkDB: ent.NetworkDB{
			Database: envOrDefault("DEX_POSTGRES_ENT_DATABASE", "postgres"),
			User:     envOrDefault("DEX_POSTGRES_ENT_USER", "postgres"),
			Password: envOrDefault("DEX_POSTGRES_ENT_PASSWORD", "postgres"),
			Host:     host,
			Port:     uint16(port),
		},
		SSL: ent.SSL{Mode: "disable"},
	}
	s, err := cfg.Open(logger)
	require.NoError(t, err, "failed to open Postgres storage")

	pubKeyStr, validSigner := generateTestSSHKeyPair(t)
	_, badSigner := generateTestSSHKeyPair(t)

	runTokenExchangeSubtests(t, s, pubKeyStr, validSigner, badSigner)
}

// TestTokenExchangeSSH_MySQL tests the full SSH token exchange flow using MySQL storage.
// Gated by DEX_MYSQL_ENT_HOST environment variable.
func TestTokenExchangeSSH_MySQL(t *testing.T) {
	host := os.Getenv("DEX_MYSQL_ENT_HOST")
	if host == "" {
		t.Skipf("test environment variable DEX_MYSQL_ENT_HOST not set, skipping")
	}

	port := uint64(3306)
	if rawPort := os.Getenv("DEX_MYSQL_ENT_PORT"); rawPort != "" {
		var parseErr error
		port, parseErr = strconv.ParseUint(rawPort, 10, 32)
		require.NoError(t, parseErr, "invalid mysql port %q", rawPort)
	}

	logger := slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := ent.MySQL{
		NetworkDB: ent.NetworkDB{
			Database: envOrDefault("DEX_MYSQL_ENT_DATABASE", "mysql"),
			User:     envOrDefault("DEX_MYSQL_ENT_USER", "mysql"),
			Password: envOrDefault("DEX_MYSQL_ENT_PASSWORD", "mysql"),
			Host:     host,
			Port:     uint16(port),
		},
		SSL: ent.SSL{Mode: "false"},
	}
	s, err := cfg.Open(logger)
	require.NoError(t, err, "failed to open MySQL storage")

	pubKeyStr, validSigner := generateTestSSHKeyPair(t)
	_, badSigner := generateTestSSHKeyPair(t)

	runTokenExchangeSubtests(t, s, pubKeyStr, validSigner, badSigner)
}

// TestTokenExchangeSSH_InMemory tests the full SSH token exchange flow using in-memory storage.
// This verifies the SSH connector works through the full server stack with the default storage.
func TestTokenExchangeSSH_InMemory(t *testing.T) {
	pubKeyStr, validSigner := generateTestSSHKeyPair(t)
	_, badSigner := generateTestSSHKeyPair(t)

	logger := slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug}))
	s := memory.New(logger)

	runTokenExchangeSubtests(t, s, pubKeyStr, validSigner, badSigner)
}

// TestTokenExchangeSSH_LDAPCoexistence tests that the SSH connector works correctly
// when an LDAP connector is also registered. This verifies that connector routing
// dispatches token exchange requests to the correct connector.
func TestTokenExchangeSSH_LDAPCoexistence(t *testing.T) {
	pubKeyStr, validSigner := generateTestSSHKeyPair(t)

	logger := slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug}))
	s := memory.New(logger)

	var srv *Server
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.ServeHTTP(w, r)
	}))
	defer httpServer.Close()

	ctx := t.Context()

	config := Config{
		Issuer:  httpServer.URL,
		Storage: s,
		Web: WebConfig{
			Dir: "../web",
		},
		Logger:             logger,
		PrometheusRegistry: prometheus.NewRegistry(),
		HealthChecker:      gosundheit.New(),
		SkipApprovalScreen: true,
		AllowedGrantTypes: []string{
			grantTypeAuthorizationCode,
			grantTypeRefreshToken,
			grantTypeTokenExchange,
		},
	}

	// Register SSH connector
	sshConfig := sshConnectorJSON(t, pubKeyStr, httpServer.URL)
	err := s.CreateConnector(ctx, storage.Connector{
		ID:              "ssh",
		Type:            "ssh",
		Name:            "SSH",
		ResourceVersion: "1",
		Config:          sshConfig,
	})
	require.NoError(t, err, "failed to create SSH connector")

	// Register LDAP connector (minimal config — just needs to exist in storage for routing tests)
	ldapConfig, err := json.Marshal(map[string]interface{}{
		"host":          "ldap.example.com:389",
		"insecureNoSSL": true,
		"bindDN":        "cn=admin,dc=example,dc=org",
		"bindPW":        "admin",
		"userSearch": map[string]interface{}{
			"baseDN":    "ou=People,dc=example,dc=org",
			"username":  "cn",
			"idAttr":    "DN",
			"emailAttr": "mail",
			"nameAttr":  "cn",
		},
	})
	require.NoError(t, err, "failed to marshal LDAP config")

	err = s.CreateConnector(ctx, storage.Connector{
		ID:              "ldap",
		Type:            "ldap",
		Name:            "LDAP",
		ResourceVersion: "1",
		Config:          ldapConfig,
	})
	require.NoError(t, err, "failed to create LDAP connector")

	// Create OAuth2 client
	err = s.CreateClient(ctx, storage.Client{
		ID:     "ssh-test-client",
		Secret: "ssh-test-secret",
		Name:   "SSH Test Client",
	})
	require.NoError(t, err, "failed to create test client")

	srv, err = newServer(ctx, config, staticRotationStrategy(testKey))
	require.NoError(t, err, "failed to create server")

	srv.refreshTokenPolicy, err = NewRefreshTokenPolicy(logger, false, "", "", "")
	require.NoError(t, err, "failed to create refresh token policy")
	srv.refreshTokenPolicy.now = time.Now

	t.Run("ssh-connector-routes-correctly", func(t *testing.T) {
		subjectToken := generateTestSSHJWT(t, validSigner, "testuser", "test-ssh-client", httpServer.URL, "ssh-test-client")
		rr := doTokenExchange(
			t, srv, httpServer.URL,
			subjectToken, "ssh",
			"ssh-test-client", "ssh-test-secret",
			tokenTypeAccess, tokenTypeAccess,
			"openid", "",
		)
		require.Equal(t, http.StatusOK, rr.Code, "SSH token exchange should succeed: %s", rr.Body.String())

		var res accessTokenResponse
		err := json.NewDecoder(rr.Result().Body).Decode(&res)
		require.NoError(t, err)
		require.NotEmpty(t, res.AccessToken)
		require.Equal(t, tokenTypeAccess, res.IssuedTokenType)
	})

	t.Run("ldap-connector-rejects-token-exchange", func(t *testing.T) {
		// LDAP connector does not implement TokenIdentityConnector, so token exchange should fail
		subjectToken := generateTestSSHJWT(t, validSigner, "testuser", "test-ssh-client", httpServer.URL, "ssh-test-client")
		rr := doTokenExchange(
			t, srv, httpServer.URL,
			subjectToken, "ldap",
			"ssh-test-client", "ssh-test-secret",
			tokenTypeAccess, tokenTypeAccess,
			"openid", "",
		)
		require.Equal(t, http.StatusBadRequest, rr.Code, "LDAP connector should reject token exchange")
	})

	t.Run("nonexistent-connector-returns-error", func(t *testing.T) {
		subjectToken := generateTestSSHJWT(t, validSigner, "testuser", "test-ssh-client", httpServer.URL, "ssh-test-client")
		rr := doTokenExchange(
			t, srv, httpServer.URL,
			subjectToken, "nonexistent",
			"ssh-test-client", "ssh-test-secret",
			tokenTypeAccess, tokenTypeAccess,
			"openid", "",
		)
		require.Equal(t, http.StatusBadRequest, rr.Code, "nonexistent connector should return error")
	})
}

// envOrDefault returns the environment variable value or a default.
func envOrDefault(key, defaultVal string) (val string) {
	val = os.Getenv(key)
	if val == "" {
		val = defaultVal
	}
	return val
}
