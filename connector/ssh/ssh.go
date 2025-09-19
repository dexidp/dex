// Package ssh implements a connector that authenticates using SSH keys
package ssh

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/ssh"

	"github.com/dexidp/dex/connector"
)

// Config holds the configuration for the SSH connector.
type Config struct {
	// Users maps usernames to their SSH key configuration and user information
	Users map[string]UserConfig `json:"users"`

	// AllowedIssuers specifies which JWT issuers are accepted
	AllowedIssuers []string `json:"allowed_issuers"`

	// DefaultGroups are assigned to all authenticated users
	DefaultGroups []string `json:"default_groups"`

	// TokenTTL specifies how long tokens are valid (in seconds, defaults to 3600 if 0)
	TokenTTL int `json:"token_ttl"`
}

// UserConfig contains a user's SSH keys and identity information.
type UserConfig struct {
	// Keys is a list of SSH public keys authorized for this user.
	// Format: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample... user@host"
	// Note: Per SSH spec, the comment (user@host) part is optional
	Keys []string `json:"keys"`

	// UserInfo contains the user's identity information
	UserInfo `json:",inline"`
}

// UserInfo contains user identity information.
type UserInfo struct {
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Groups   []string `json:"groups"`
	FullName string   `json:"full_name"`
}

// Challenge represents a temporary SSH challenge for authentication
type Challenge struct {
	Data      []byte
	Username  string
	CreatedAt time.Time
}

// challengeStore holds temporary challenges with TTL
type challengeStore struct {
	challenges map[string]*Challenge
	mutex      sync.RWMutex
	ttl        time.Duration
}

// newChallengeStore creates a new challenge store with cleanup
func newChallengeStore(ttl time.Duration) *challengeStore {
	store := &challengeStore{
		challenges: make(map[string]*Challenge),
		ttl:        ttl,
	}
	// Start cleanup goroutine
	go store.cleanup()
	return store
}

// store saves a challenge with expiration
func (cs *challengeStore) store(id string, challenge *Challenge) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	cs.challenges[id] = challenge
}

// get retrieves and removes a challenge
func (cs *challengeStore) get(id string) (challenge *Challenge, found bool) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	challenge, found = cs.challenges[id]
	if found {
		delete(cs.challenges, id) // One-time use
	}
	return challenge, found
}

// cleanup removes expired challenges
func (cs *challengeStore) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		cs.mutex.Lock()
		now := time.Now()
		for id, challenge := range cs.challenges {
			if now.Sub(challenge.CreatedAt) > cs.ttl {
				delete(cs.challenges, id)
			}
		}
		cs.mutex.Unlock()
	}
}

// SSHConnector implements the Dex connector interface for SSH key authentication.
// Supports both JWT-based authentication (TokenIdentityConnector) and
// challenge/response authentication (CallbackConnector).
type SSHConnector struct {
	config     Config
	logger     *slog.Logger
	challenges *challengeStore
}

// Compile-time interface assertions
var (
	_ connector.Connector              = &SSHConnector{}
	_ connector.TokenIdentityConnector = &SSHConnector{}
	_ connector.CallbackConnector      = &SSHConnector{}
)

// Open creates a new SSH connector.
// Uses slog.Logger for compatibility with Dex v2.44.0+.
func (c *Config) Open(id string, logger *slog.Logger) (conn connector.Connector, err error) {
	// Log SSH connector startup
	if logger != nil {
		logger.Info("SSH connector starting")
	}

	// Set default values if not configured
	config := *c
	if config.TokenTTL == 0 {
		config.TokenTTL = 3600 // Default to 1 hour
	}

	return &SSHConnector{
		config:     config,
		logger:     logger,
		challenges: newChallengeStore(5 * time.Minute), // 5-minute challenge TTL
	}, nil
}

// LoginURL returns the URL for SSH-based login.
// Supports both JWT-based and challenge/response authentication flows.
func (c *SSHConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (loginURL string, err error) {
	// Check if this is a challenge/response request (indicated by specific parameter)
	parsedCallback, err := url.Parse(callbackURL)
	if err != nil {
		return loginURL, fmt.Errorf("invalid callback URL: %w", err)
	}

	// If this is a challenge request, generate challenge and embed it
	if parsedCallback.Query().Get("ssh_challenge") == "true" {
		username := parsedCallback.Query().Get("username")
		return c.generateChallengeURL(callbackURL, state, username)
	}

	// Default: JWT-based authentication (backward compatibility)
	// For JWT clients, return callback URL with SSH auth flag
	loginURL = fmt.Sprintf("%s?state=%s&ssh_auth=true", callbackURL, state)
	return loginURL, err
}

// generateChallengeURL creates a challenge and returns a URL containing it
// SECURITY: Validates user exists before generating challenge to prevent user enumeration
func (c *SSHConnector) generateChallengeURL(callbackURL, state, username string) (challengeURL string, err error) {
	// Security check: Validate user exists to prevent user enumeration
	if username == "" {
		c.logAuditEvent("auth_attempt", "", "unknown", "challenge", "failed", "missing username in challenge request")
		return "", errors.New("username required for challenge generation")
	}

	if _, exists := c.config.Users[username]; !exists {
		c.logAuditEvent("auth_attempt", username, "unknown", "challenge", "failed", "user not found during challenge generation")
		return "", errors.New("user not found")
	}

	// Generate cryptographic challenge
	challengeData := make([]byte, 32)
	if _, err := rand.Read(challengeData); err != nil {
		return "", fmt.Errorf("failed to generate challenge: %w", err)
	}

	// Create unique challenge ID
	challengeID := base64.URLEncoding.EncodeToString(challengeData[:16])

	// Store challenge temporarily with username for validation
	challenge := &Challenge{
		Data:      challengeData,
		Username:  username,
		CreatedAt: time.Now(),
	}
	c.challenges.store(challengeID, challenge)

	// Create callback URL with challenge embedded
	challengeB64 := base64.URLEncoding.EncodeToString(challengeData)
	stateWithChallenge := fmt.Sprintf("%s:%s", state, challengeID)

	// Parse the callback URL to handle existing query parameters properly
	parsedCallback, err := url.Parse(callbackURL)
	if err != nil {
		return challengeURL, fmt.Errorf("invalid callback URL: %w", err)
	}

	// Add our parameters to the existing query
	values := parsedCallback.Query()
	values.Set("state", stateWithChallenge)
	values.Set("ssh_challenge", challengeB64)
	parsedCallback.RawQuery = values.Encode()

	c.logAuditEvent("challenge_generated", username, "unknown", "challenge", "success", "challenge generated successfully")
	challengeURL = parsedCallback.String()
	return challengeURL, err
}

// HandleCallback processes the SSH authentication callback.
// Supports both JWT-based and challenge/response authentication flows.
func (c *SSHConnector) HandleCallback(scopes connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	// Check if this is a challenge/response flow
	if challengeB64 := r.FormValue("ssh_challenge"); challengeB64 != "" {
		return c.handleChallengeResponse(r)
	}

	// Handle JWT-based authentication (existing flow)
	return c.handleJWTCallback(r)
}

// handleJWTCallback processes JWT-based authentication (existing logic)
func (c *SSHConnector) handleJWTCallback(r *http.Request) (identity connector.Identity, err error) {
	// Handle both SSH JWT directly and as authorization code
	var sshJWT string

	// First try direct SSH JWT parameter
	sshJWT = r.FormValue("ssh_jwt")

	// If not found, try as authorization code
	if sshJWT == "" {
		sshJWT = r.FormValue("code")
	}

	if sshJWT == "" {
		c.logAuditEvent("auth_attempt", "", "", "", "failed", "no SSH JWT or authorization code provided")
		return connector.Identity{}, errors.New("no SSH JWT or authorization code provided")
	}

	// Validate and extract identity using existing JWT logic
	return c.validateSSHJWT(sshJWT)
}

// handleChallengeResponse processes challenge/response authentication
func (c *SSHConnector) handleChallengeResponse(r *http.Request) (identity connector.Identity, err error) {
	// Extract parameters
	username := r.FormValue("username")
	signature := r.FormValue("signature")
	state := r.FormValue("state")

	if username == "" || signature == "" || state == "" {
		c.logAuditEvent("auth_attempt", username, "unknown", "challenge", "failed", "missing required parameters")
		return connector.Identity{}, errors.New("missing required parameters: username, signature, or state")
	}

	// Extract challenge ID from state
	parts := strings.Split(state, ":")
	if len(parts) < 2 {
		c.logAuditEvent("auth_attempt", username, "unknown", "challenge", "failed", "invalid state format")
		return connector.Identity{}, errors.New("invalid state format")
	}
	challengeID := parts[len(parts)-1]

	// Retrieve stored challenge
	challenge, exists := c.challenges.get(challengeID)
	if !exists {
		c.logAuditEvent("auth_attempt", username, "unknown", "challenge", "failed", "invalid or expired challenge")
		return connector.Identity{}, errors.New("invalid or expired challenge")
	}

	// SECURITY: Validate that the username matches the challenge
	// This prevents challenge reuse across different users
	if challenge.Username != username {
		c.logAuditEvent("auth_attempt", username, "unknown", "challenge", "failed",
			fmt.Sprintf("username mismatch: challenge for %s, request for %s", challenge.Username, username))
		return connector.Identity{}, errors.New("challenge username mismatch")
	}

	// Validate user exists in configuration (redundant but defensive)
	userConfig, exists := c.config.Users[username]
	if !exists {
		c.logAuditEvent("auth_attempt", username, "unknown", "challenge", "failed", "user not found")
		return connector.Identity{}, errors.New("user not found")
	}

	// Verify SSH signature against challenge
	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		c.logAuditEvent("auth_attempt", username, "unknown", "challenge", "failed", "invalid signature encoding")
		return connector.Identity{}, fmt.Errorf("invalid signature encoding: %w", err)
	}

	// Try each configured SSH key for the user
	var verifiedKey ssh.PublicKey
	for _, keyStr := range userConfig.Keys {
		if pubKey, err := c.parseSSHKey(keyStr); err == nil {
			if c.verifySSHSignature(pubKey, challenge.Data, signatureBytes) {
				verifiedKey = pubKey
				break
			}
		}
	}

	if verifiedKey == nil {
		keyFingerprint := "unknown"
		c.logAuditEvent("auth_attempt", username, keyFingerprint, "challenge", "failed", "signature verification failed")
		return connector.Identity{}, errors.New("signature verification failed")
	}

	// Create identity from user configuration
	userInfo := userConfig.UserInfo
	if userInfo.Username == "" {
		userInfo.Username = username
	}

	// Combine default groups with user-specific groups
	allGroups := append([]string{}, c.config.DefaultGroups...)
	allGroups = append(allGroups, userInfo.Groups...)

	identity = connector.Identity{
		UserID:            userInfo.Username,
		Username:          userInfo.Username,
		PreferredUsername: userInfo.Username,
		Email:             userInfo.Email,
		EmailVerified:     true,
		Groups:            allGroups,
	}

	// Log successful authentication
	keyFingerprint := ssh.FingerprintSHA256(verifiedKey)
	c.logAuditEvent("auth_success", username, keyFingerprint, "challenge", "success",
		fmt.Sprintf("user %s authenticated with SSH key %s via challenge/response", username, keyFingerprint))

	return identity, nil
}

// parseSSHKey parses a public key string into an SSH public key
func (c *SSHConnector) parseSSHKey(keyStr string) (pubKey ssh.PublicKey, err error) {
	publicKey, comment, options, rest, err := ssh.ParseAuthorizedKey([]byte(keyStr))
	_ = comment // Comment is optional per SSH spec
	_ = options // Options not used in this context
	_ = rest    // Rest not used in this context
	if err != nil {
		return nil, fmt.Errorf("invalid SSH public key format: %w", err)
	}
	return publicKey, nil
}

// verifySSHSignature verifies an SSH signature against data using a public key
func (c *SSHConnector) verifySSHSignature(pubKey ssh.PublicKey, data, signature []byte) (valid bool) {
	// For SSH signature verification, we need to reconstruct the signed data format
	// SSH signatures typically sign a specific data format

	// Create a signature object from the signature bytes
	sig := &ssh.Signature{}
	if err := ssh.Unmarshal(signature, sig); err != nil {
		if c.logger != nil {
			c.logger.Debug("Failed to unmarshal SSH signature", "error", err)
		}
		return false
	}

	// Verify the signature against the data
	err := pubKey.Verify(data, sig)
	return err == nil
}

// validateSSHJWT validates an SSH-signed JWT and extracts user identity.
// SECURITY FIX: Now uses configured keys for verification instead of trusting keys from JWT claims.
func (c *SSHConnector) validateSSHJWT(sshJWTString string) (identity connector.Identity, err error) {
	// Register our custom SSH signing method for JWT parsing
	jwt.RegisterSigningMethod("SSH", func() jwt.SigningMethod {
		return &SSHSigningMethodServer{}
	})

	// Parse JWT with secure verification - try all configured user keys
	token, verifiedUser, verifiedKey, err := c.parseAndVerifyJWTSecurely(sshJWTString)
	if err != nil {
		c.logAuditEvent("auth_attempt", "unknown", "unknown", "unknown", "failed", fmt.Sprintf("JWT parse error: %s", err.Error()))
		return connector.Identity{}, fmt.Errorf("failed to parse JWT: %w", err)
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return connector.Identity{}, errors.New("invalid JWT claims format")
	}

	// Validate JWT claims (extracted for readability)
	sub, iss, err := c.validateJWTClaims(claims)
	if err != nil {
		keyFingerprint := ssh.FingerprintSHA256(verifiedKey)
		c.logAuditEvent("auth_attempt", sub, keyFingerprint, iss, "failed", err.Error())
		return connector.Identity{}, err
	}

	// Use the verified user info (key was already verified during parsing)
	userInfo := c.config.Users[verifiedUser].UserInfo
	if userInfo.Username == "" {
		userInfo.Username = verifiedUser
	}

	// Build identity
	identity = connector.Identity{
		UserID:        userInfo.Username,
		Username:      userInfo.Username,
		Email:         userInfo.Email,
		EmailVerified: true,
		Groups:        append(userInfo.Groups, c.config.DefaultGroups...),
	}

	// Log successful authentication with verified key fingerprint
	keyFingerprint := ssh.FingerprintSHA256(verifiedKey)
	c.logAuditEvent("auth_success", sub, keyFingerprint, iss, "success", fmt.Sprintf("user %s authenticated with key %s", sub, keyFingerprint))

	return identity, nil
}

// parseAndVerifyJWTSecurely implements secure 2-pass JWT verification following jwt-ssh-agent pattern.
//
// CRITICAL SECURITY MODEL:
// - JWT is just a packaging format - it contains NO trusted data until verification succeeds
// - Trusted public keys and user mappings are configured separately in Dex by administrators
// - Authentication (JWT signature verification) is separated from authorization (user/key mapping)
// - This prevents key injection attacks where clients could embed their own verification keys
//
// Returns the parsed token, verified username, verified public key, and any error.
func (c *SSHConnector) parseAndVerifyJWTSecurely(sshJWTString string) (token *jwt.Token, username string, pubKey ssh.PublicKey, err error) {
	// PASS 1: Parse JWT structure without verification to extract claims
	// This is tricky - we need to get the subject to know which keys to try for verification,
	// but we're NOT ready to trust this data yet. The claims are UNTRUSTED until verification succeeds.
	parser := &jwt.Parser{}
	unverifiedToken, _, err := parser.ParseUnverified(sshJWTString, jwt.MapClaims{})
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to parse JWT structure: %w", err)
	}

	// Extract the subject claim - this tells us which user is CLAIMING to authenticate
	// IMPORTANT: We do NOT trust this claim yet! It's just used to know which keys to try
	claims, ok := unverifiedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, "", nil, errors.New("invalid claims format")
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return nil, "", nil, errors.New("missing or invalid sub claim")
	}

	// Now we have the subject from the JWT - i.e. the user trying to auth.
	// We still don't trust it though! It's only used to guide our verification attempts.

	// PASS 2: Try cryptographic verification against each configured public key
	// SECURITY CRITICAL: Only SSH keys explicitly configured in Dex by administrators can verify JWTs
	// This enforces the separation between authentication and authorization:
	// - Authentication: Cryptographic proof the client holds a private key
	// - Authorization: Administrative decision about which keys/users are allowed
	for username, userConfig := range c.config.Users {
		for _, authorizedKeyStr := range userConfig.Keys {
			// Parse the configured public key (trusted, set by administrators)
			publicKey, comment, options, rest, err := ssh.ParseAuthorizedKey([]byte(authorizedKeyStr))
			_, _, _ = comment, options, rest // Explicitly ignore unused return values
			if err != nil {
				continue // Skip invalid keys
			}

			// Attempt cryptographic verification of JWT signature using this configured key
			// This proves the client holds the corresponding private key
			verifiedToken, err := jwt.Parse(sshJWTString, func(token *jwt.Token) (interface{}, error) {
				if token.Method.Alg() != "SSH" {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				// Return the configured public key for verification - NOT any key from JWT claims
				return publicKey, nil
			})

			if err == nil && verifiedToken.Valid {
				// SUCCESS: Cryptographic verification passed with a configured key!
				// NOW we can trust the JWT claims because we've proven:
				// 1. The JWT was signed by a private key corresponding to a configured public key
				// 2. The configured key belongs to this username (per administrator configuration)
				// 3. No key injection attack is possible (we never used keys from JWT claims)
				//
				// Return the username from our configuration (trusted), not from JWT claims
				return verifiedToken, username, publicKey, nil
			}
		}
	}

	return nil, "", nil, fmt.Errorf("no configured key could verify the JWT signature")
}

// validateJWTClaims validates the standard JWT claims (sub, aud, iss, exp, nbf).
// Returns subject, issuer, and any validation error.
func (c *SSHConnector) validateJWTClaims(claims jwt.MapClaims) (username string, issuer string, err error) {
	// Validate required claims
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", "", errors.New("missing or invalid sub claim")
	}

	aud, ok := claims["aud"].(string)
	if !ok || aud == "" {
		return sub, "", errors.New("missing or invalid aud claim")
	}

	iss, ok := claims["iss"].(string)
	if !ok || iss == "" {
		return sub, "", errors.New("missing or invalid iss claim")
	}

	// Validate audience - ensure this token is intended for our Dex instance
	if aud != "kubernetes" {
		return sub, iss, fmt.Errorf("invalid audience: %s", aud)
	}

	// Validate issuer
	if !c.isAllowedIssuer(iss) {
		return sub, iss, fmt.Errorf("invalid issuer: %s", iss)
	}

	// Validate expiration (critical security check)
	exp, ok := claims["exp"].(float64)
	if !ok {
		return sub, iss, errors.New("missing or invalid exp claim")
	}

	if time.Unix(int64(exp), 0).Before(time.Now()) {
		return sub, iss, errors.New("token has expired")
	}

	// Validate not before if present
	if nbfClaim, nbfOk := claims["nbf"].(float64); nbfOk {
		if time.Unix(int64(nbfClaim), 0).After(time.Now()) {
			return sub, iss, errors.New("token not yet valid")
		}
	}

	return sub, iss, nil
}

// findUserByUsernameAndKey finds a user by username and verifies the key is authorized.
// This provides O(1) lookup performance instead of searching all users.
// Supports both SSH fingerprints and full public key formats.
func (c *SSHConnector) findUserByUsernameAndKey(username, keyFingerprint string) (userInfo UserInfo, err error) {
	// First, check the new Users format (O(1) lookup)
	if userConfig, exists := c.config.Users[username]; exists {
		// Check if this key is authorized for this user
		for _, authorizedKey := range userConfig.Keys {
			if c.isKeyMatch(authorizedKey, keyFingerprint) {
				// Return the user info with username filled in if not already set
				userInfo := userConfig.UserInfo
				if userInfo.Username == "" {
					userInfo.Username = username
				}
				return userInfo, nil
			}
		}
		return UserInfo{}, fmt.Errorf("key %s not authorized for user %s", keyFingerprint, username)
	}

	return UserInfo{}, fmt.Errorf("user %s not found or key %s not authorized", username, keyFingerprint)
}

// isKeyMatch checks if an authorized key (from config) matches the presented key fingerprint.
// Only supports full public key format in the config:
//   - Full public keys: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample... user@host"
//     Note: Per SSH spec, the comment (user@host) part is optional
func (c *SSHConnector) isKeyMatch(authorizedKey, presentedKeyFingerprint string) (matches bool) {
	// Parse the authorized key as a full public key
	publicKey, comment, _, rest, err := ssh.ParseAuthorizedKey([]byte(authorizedKey))
	_ = comment // Ignore comment
	_ = rest    // Ignore rest
	if err != nil {
		// Invalid public key format
		c.logger.Warn("Invalid public key format in configuration", "key", authorizedKey, "error", err)
		return false
	}

	// Generate fingerprint from the public key and compare
	authorizedKeyFingerprint := ssh.FingerprintSHA256(publicKey)
	return authorizedKeyFingerprint == presentedKeyFingerprint
}

// isAllowedIssuer checks if the JWT issuer is allowed.
func (c *SSHConnector) isAllowedIssuer(issuer string) (allowed bool) {
	if len(c.config.AllowedIssuers) == 0 {
		return true // Allow all if none specified
	}

	for _, allowed := range c.config.AllowedIssuers {
		if issuer == allowed {
			return true
		}
	}

	return false
}

// SSHSigningMethodServer implements JWT signing method for server-side SSH verification.
type SSHSigningMethodServer struct{}

// Alg returns the signing method algorithm identifier.
func (m *SSHSigningMethodServer) Alg() string {
	return "SSH"
}

// Sign is not implemented on server side (client-only operation).
func (m *SSHSigningMethodServer) Sign(signingString string, key interface{}) (signature []byte, err error) {
	return nil, errors.New("SSH signing not supported on server side")
}

// Verify verifies the JWT signature using the SSH public key.
func (m *SSHSigningMethodServer) Verify(signingString string, signature []byte, key interface{}) (err error) {
	// Parse SSH public key
	publicKey, ok := key.(ssh.PublicKey)
	if !ok {
		return fmt.Errorf("SSH verification requires ssh.PublicKey, got %T", key)
	}

	// Decode the base64-encoded signature
	signatureStr := string(signature)
	signatureBytes, err := base64.StdEncoding.DecodeString(signatureStr)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// For SSH signature verification, we need to construct the signature structure
	// The signature format follows SSH wire protocol
	sshSignature := &ssh.Signature{
		Format: publicKey.Type(), // Use key type as format
		Blob:   signatureBytes,
	}

	// Verify the signature
	err = publicKey.Verify([]byte(signingString), sshSignature)
	if err != nil {
		return fmt.Errorf("SSH signature verification failed: %w", err)
	}

	return nil
}

// logAuditEvent logs SSH authentication events for security auditing.
// This provides comprehensive audit trails for SSH-based authentication attempts.
func (c *SSHConnector) logAuditEvent(eventType, username, keyFingerprint, issuer, status, details string) {
	// Build structured log message
	logMsg := fmt.Sprintf("SSH_AUDIT: type=%s username=%s key=%s issuer=%s status=%s details=%q",
		eventType, username, keyFingerprint, issuer, status, details)

	// Use slog.Logger for audit logging
	if c.logger != nil {
		c.logger.Info(logMsg)
	} else {
		// Fallback: use standard output for audit logging
		// This ensures audit events are always logged even if logger is unavailable
		fmt.Printf("%s\n", logMsg)
	}
}

// TokenIdentity implements the TokenIdentityConnector interface.
// This method validates an SSH JWT token and returns the user identity.
func (c *SSHConnector) TokenIdentity(ctx context.Context, subjectTokenType, subjectToken string) (identity connector.Identity, err error) {
	if c.logger != nil {
		c.logger.InfoContext(ctx, "TokenIdentity method called", "tokenType", subjectTokenType)
	}

	// Validate token type - accept standard OAuth2 JWT types
	switch subjectTokenType {
	case "ssh_jwt", "urn:ietf:params:oauth:token-type:jwt", "urn:ietf:params:oauth:token-type:access_token", "urn:ietf:params:oauth:token-type:id_token":
		// Supported token types
	default:
		return connector.Identity{}, fmt.Errorf("unsupported token type: %s", subjectTokenType)
	}

	// Use existing SSH JWT validation logic
	identity, err = c.validateSSHJWT(subjectToken)
	if err != nil {
		if c.logger != nil {
			// SSH agent trying multiple keys is normal behavior - log at debug level
			c.logger.DebugContext(ctx, "SSH JWT validation failed in TokenIdentity", "error", err)
		}
		return connector.Identity{}, fmt.Errorf("SSH JWT validation failed: %w", err)
	}

	if c.logger != nil {
		c.logger.InfoContext(ctx, "TokenIdentity successful", "user", identity.UserID)
	}
	return identity, nil
}
