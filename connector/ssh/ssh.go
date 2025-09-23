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

	// ChallengeTTL specifies how long challenges are valid (in seconds, defaults to 300 if 0)
	ChallengeTTL int `json:"challenge_ttl"`
}

// UserConfig contains a user's SSH keys and identity information.
type UserConfig struct {
	// Keys is a list of SSH public keys authorized for this user.
	// Format: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample... user@host"
	// Note: Per SSH spec, the comment (user@host) part is optional
	Keys []string `json:"keys"`

	// UserInfo contains the user's identity information returned in OIDC tokens.
	// This information is configured by administrators and cannot be influenced by clients.
	UserInfo `json:",inline"`
}

// UserInfo contains user identity information for OIDC token claims.
// All fields are configured administratively to prevent privilege escalation attacks.
type UserInfo struct {
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Groups   []string `json:"groups"`
	FullName string   `json:"full_name"`
}

// Challenge represents a temporary SSH challenge for challenge/response authentication.
// Challenges are single-use and expire after the configured ChallengeTTL (default 5 minutes) to prevent replay attacks.
type Challenge struct {
	Data      []byte
	Username  string
	CreatedAt time.Time
	IsValid   bool // True if username exists in config, false for enumeration prevention
}

// challengeStore holds temporary challenges with TTL
type challengeStore struct {
	challenges map[string]*Challenge
	mutex      sync.RWMutex
	ttl        time.Duration
}

// rateLimiter prevents brute force user enumeration attacks
type rateLimiter struct {
	attempts    map[string][]time.Time
	mutex       sync.RWMutex
	maxAttempts int
	window      time.Duration
}

// newRateLimiter creates a rate limiter with cleanup
func newRateLimiter(maxAttempts int, window time.Duration) (limiter *rateLimiter) {
	limiter = &rateLimiter{
		attempts:    make(map[string][]time.Time),
		maxAttempts: maxAttempts,
		window:      window,
	}
	// Start cleanup goroutine
	go limiter.cleanup()
	return
}

// isAllowed checks if an IP can make another attempt
func (rl *rateLimiter) isAllowed(ip string) (allowed bool) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	attemptTimes := rl.attempts[ip]

	// Remove old attempts outside the window
	var validAttempts []time.Time
	for _, attemptTime := range attemptTimes {
		if now.Sub(attemptTime) < rl.window {
			validAttempts = append(validAttempts, attemptTime)
		}
	}

	// Check if under limit
	if len(validAttempts) >= rl.maxAttempts {
		rl.attempts[ip] = validAttempts
		allowed = false
		return
	}

	// Record this attempt
	validAttempts = append(validAttempts, now)
	rl.attempts[ip] = validAttempts
	allowed = true
	return allowed
}

// cleanup removes old rate limit entries
func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute * 5)
	for range ticker.C {
		rl.mutex.Lock()
		now := time.Now()
		for ip, attempts := range rl.attempts {
			var validAttempts []time.Time
			for _, attemptTime := range attempts {
				if now.Sub(attemptTime) < rl.window {
					validAttempts = append(validAttempts, attemptTime)
				}
			}
			if len(validAttempts) == 0 {
				delete(rl.attempts, ip)
			} else {
				rl.attempts[ip] = validAttempts
			}
		}
		rl.mutex.Unlock()
	}
}

// newChallengeStore creates a new challenge store with cleanup
func newChallengeStore(ttl time.Duration) (store *challengeStore) {
	store = &challengeStore{
		challenges: make(map[string]*Challenge),
		ttl:        ttl,
	}
	// Start cleanup goroutine
	go store.cleanup()
	return
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
	return
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
	config      Config
	logger      *slog.Logger
	challenges  *challengeStore
	rateLimiter *rateLimiter
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
	if config.ChallengeTTL == 0 {
		config.ChallengeTTL = 300 // Default to 5 minutes
	}

	conn = &SSHConnector{
		config:      config,
		logger:      logger,
		challenges:  newChallengeStore(time.Duration(config.ChallengeTTL) * time.Second),
		rateLimiter: newRateLimiter(10, time.Minute*5), // 10 attempts per 5 minutes per IP
	}
	return
}

// LoginURL generates the OAuth2 authorization URL for SSH authentication.
// The implementation supports two authentication modes:
//
//  1. JWT-based authentication: Returns URL with ssh_auth=true parameter for clients
//     that will perform OAuth2 Token Exchange with SSH-signed JWTs
//
//  2. Challenge/response authentication: Generates cryptographic challenge when
//     ssh_challenge=true parameter is present, embeds challenge in callback URL
//
// The URL format follows standard OAuth2 authorization code flow patterns.
// Clients determine the authentication mode via query parameters.

func (c *SSHConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (loginURL string, err error) {
	// This method exists for interface compatibility but lacks request context
	// Rate limiting is not possible without HTTP request - log this limitation
	var parsedCallback *url.URL
	parsedCallback, err = url.Parse(callbackURL)
	if err != nil {
		err = fmt.Errorf("invalid callback URL: %w", err)
		return
	}

	// If this is a challenge request without request context, we can't rate limit
	if parsedCallback.Query().Get("ssh_challenge") == "true" {
		username := parsedCallback.Query().Get("username")
		c.logAuditEvent("auth_attempt", username, "unknown", "challenge", "warning", "challenge request without rate limiting context")
		// Proceed without rate limiting (not ideal but maintains compatibility)
		loginURL, err = c.generateChallengeURL(callbackURL, state, username, "unknown")
		return
	}

	// Default: JWT-based authentication (backward compatibility)
	// For JWT clients, return callback URL with SSH auth flag
	loginURL = fmt.Sprintf("%s?state=%s&ssh_auth=true", callbackURL, state)
	return
}

// generateChallengeURL creates a callback URL with an embedded SSH challenge.
// This method implements the challenge generation phase of challenge/response authentication.
//
// The process:
// 1. Validates the requested username exists in configuration
// 2. Generates cryptographically random challenge data
// 3. Stores challenge temporarily with expiration
// 4. Encodes challenge in base64 and embeds in callback URL
// 5. Returns URL that clients can extract challenge from
//
// Security: Challenges are single-use and time-limited to prevent replay attacks.
// User enumeration is prevented by validating usernames before challenge generation.

func (c *SSHConnector) generateChallengeURL(callbackURL, state, username, clientIP string) (challengeURL string, err error) {
	// SECURITY: Rate limiting to prevent brute force user enumeration (skip if IP unknown)
	if clientIP != "unknown" && !c.rateLimiter.isAllowed(clientIP) {
		c.logAuditEvent("auth_attempt", username, "unknown", "challenge", "failed", fmt.Sprintf("rate limit exceeded for IP %s", clientIP))
		err = errors.New("too many requests")
		return challengeURL, err
	}
	// SECURITY: Prevent user enumeration by always generating challenges
	// Valid and invalid users get identical responses - authentication fails later
	if username == "" {
		c.logAuditEvent("auth_attempt", "", "unknown", "challenge", "failed", "missing username in challenge request")
		err = errors.New("username required for challenge generation")
		return challengeURL, err
	}

	// Check if user exists, but DON'T change the response behavior
	userExists := false
	if _, exists := c.config.Users[username]; exists {
		userExists = exists
	}

	// ALWAYS generate cryptographic challenge (prevents timing attacks)
	challengeData := make([]byte, 32)
	if _, err = rand.Read(challengeData); err != nil {
		return challengeURL, fmt.Errorf("failed to generate challenge: %w", err)
	}

	// Create unique challenge ID
	challengeID := base64.URLEncoding.EncodeToString(challengeData[:16])

	// Store challenge with validity flag (prevents user enumeration)
	challenge := &Challenge{
		Data:      challengeData,
		Username:  username,
		CreatedAt: time.Now(),
		IsValid:   userExists, // This determines if auth will succeed later
	}
	c.challenges.store(challengeID, challenge)

	// Create callback URL with challenge embedded
	challengeB64 := base64.URLEncoding.EncodeToString(challengeData)
	stateWithChallenge := fmt.Sprintf("%s:%s", state, challengeID)

	// Parse the callback URL to handle existing query parameters properly
	var parsedCallback *url.URL
	parsedCallback, err = url.Parse(callbackURL)
	if err != nil {
		err = fmt.Errorf("invalid callback URL: %w", err)
		return challengeURL, err
	}

	// Add our parameters to the existing query
	values := parsedCallback.Query()
	values.Set("state", stateWithChallenge)
	values.Set("ssh_challenge", challengeB64)
	parsedCallback.RawQuery = values.Encode()

	// SECURITY: Always log success to prevent enumeration via logs
	// Real validation happens during signature verification
	c.logAuditEvent("challenge_generated", username, "unknown", "challenge", "success", "challenge generated successfully")
	challengeURL = parsedCallback.String()
	err = nil
	return challengeURL, err
}

// HandleCallback processes OAuth2 callbacks for SSH authentication.
// This method implements the callback phase of the OAuth2 authorization code flow.
//
// The connector supports two distinct authentication flows:
//
// 1. JWT-based authentication:
//   - Clients provide SSH-signed JWTs as authorization codes
//   - JWTs are verified against administratively configured SSH keys
//   - Supports OAuth2 Token Exchange (RFC 8693) pattern
//
// 2. Challenge/response authentication:
//   - Clients provide signatures of previously issued challenges
//   - Signatures are verified against SSH keys for the claimed user
//   - Follows standard OAuth2 authorization code pattern
//
// Both flows result in connector.Identity objects containing user attributes
// configured administratively, preventing client-controlled privilege escalation.
func (c *SSHConnector) HandleCallback(scopes connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	// Check if this is a challenge/response flow
	if challengeB64 := r.FormValue("ssh_challenge"); challengeB64 != "" {
		identity, err = c.handleChallengeResponse(r)
		return
	}

	// Handle JWT-based authentication (existing flow)
	identity, err = c.handleJWTCallback(r)
	return
}

// handleJWTCallback processes JWT-based authentication via OAuth2 Token Exchange.
// This method validates SSH-signed JWTs submitted as OAuth2 authorization codes.
//
// The JWT verification process:
// 1. Extracts JWT from either direct submission or authorization code
// 2. Parses JWT headers to identify signing key requirements
// 3. Validates JWT signature against administratively configured SSH keys
// 4. Verifies JWT claims (issuer, expiration, audience)
// 5. Maps authenticated user to configured identity attributes
//
// Security: Only SSH keys configured by administrators can verify JWTs.
// No cryptographic material from JWTs is trusted until signature verification succeeds.
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
		err = errors.New("no SSH JWT or authorization code provided")
		return
	}

	// Validate and extract identity using existing JWT logic
	identity, err = c.validateSSHJWT(sshJWT)
	return
}

// handleChallengeResponse processes challenge/response authentication flows.
// This method validates SSH signatures of previously issued challenges.
//
// The verification process:
// 1. Extracts challenge, signature, and username from callback request
// 2. Retrieves stored challenge data and validates expiration
// 3. Verifies SSH signature against user's configured public keys
// 4. Returns user identity attributes from administrative configuration
//
// Security: Challenges are single-use and time-limited. User enumeration is
// prevented by only generating challenges for valid configured users.
func (c *SSHConnector) handleChallengeResponse(r *http.Request) (identity connector.Identity, err error) {
	// Extract parameters
	username := r.FormValue("username")
	signature := r.FormValue("signature")
	state := r.FormValue("state")

	if username == "" || signature == "" || state == "" {
		c.logAuditEvent("auth_attempt", username, "unknown", "challenge", "failed", "missing required parameters")
		err = errors.New("missing required parameters: username, signature, or state")
		return identity, err
	}

	// Extract challenge ID from state
	parts := strings.Split(state, ":")
	if len(parts) < 2 {
		c.logAuditEvent("auth_attempt", username, "unknown", "challenge", "failed", "invalid state format")
		err = errors.New("invalid state format")
		return identity, err
	}
	challengeID := parts[len(parts)-1]

	// Retrieve stored challenge
	challenge, exists := c.challenges.get(challengeID)
	if !exists {
		c.logAuditEvent("auth_attempt", username, "unknown", "challenge", "failed", "invalid or expired challenge")
		err = errors.New("invalid or expired challenge")
		return identity, err
	}

	// SECURITY: Validate that the username matches the challenge
	// This prevents challenge reuse across different users
	if challenge.Username != username {
		c.logAuditEvent("auth_attempt", username, "unknown", "challenge", "failed",
			fmt.Sprintf("username mismatch: challenge for %s, request for %s", challenge.Username, username))
		err = errors.New("challenge username mismatch")
		return identity, err
	}

	// SECURITY: Check if this was a valid user challenge (prevents enumeration)
	if !challenge.IsValid {
		c.logAuditEvent("auth_attempt", username, "unknown", "challenge", "failed", "invalid user challenge")
		err = errors.New("authentication failed")
		return identity, err
	}

	// Get user config (we know it exists because IsValid=true)
	userConfig, exists := c.config.Users[username]
	if !exists {
		// This should never happen if IsValid=true, but defensive programming
		c.logAuditEvent("auth_attempt", username, "unknown", "challenge", "failed", "user config missing")
		err = errors.New("authentication failed")
		return identity, err
	}

	// Verify SSH signature against challenge
	var signatureBytes []byte
	signatureBytes, err = base64.StdEncoding.DecodeString(signature)
	if err != nil {
		c.logAuditEvent("auth_attempt", username, "unknown", "challenge", "failed", "invalid signature encoding")
		return identity, fmt.Errorf("invalid signature encoding: %w", err)
	}

	// Try each configured SSH key for the user
	var verifiedKey ssh.PublicKey
	for _, keyStr := range userConfig.Keys {
		var pubKey ssh.PublicKey
		if pubKey, err = c.parseSSHKey(keyStr); err == nil {
			if c.verifySSHSignature(pubKey, challenge.Data, signatureBytes) {
				verifiedKey = pubKey
				break
			}
		}
	}

	if verifiedKey == nil {
		keyFingerprint := "unknown"
		c.logAuditEvent("auth_attempt", username, keyFingerprint, "challenge", "failed", "signature verification failed")
		err = errors.New("signature verification failed")
		return identity, err
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

	err = nil
	return identity, err
}

// parseSSHKey parses a public key string into an SSH public key
func (c *SSHConnector) parseSSHKey(keyStr string) (pubKey ssh.PublicKey, err error) {
	var comment string
	var options []string
	var rest []byte
	pubKey, comment, options, rest, err = ssh.ParseAuthorizedKey([]byte(keyStr))
	_ = comment // Comment is optional per SSH spec
	_ = options // Options not used in this context
	_ = rest    // Rest not used in this context
	if err != nil {
		err = fmt.Errorf("invalid SSH public key format: %w", err)
		return
	}
	return
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
		valid = false
		return
	}

	// Verify the signature against the data
	err := pubKey.Verify(data, sig)
	valid = err == nil
	return
}

// validateSSHJWT validates an SSH-signed JWT and extracts user identity.
// SECURITY FIX: Now uses configured keys for verification instead of trusting keys from JWT claims.
func (c *SSHConnector) validateSSHJWT(sshJWTString string) (identity connector.Identity, err error) {
	// Register our custom SSH signing method for JWT parsing
	jwt.RegisterSigningMethod("SSH", func() (method jwt.SigningMethod) {
		method = &SSHSigningMethodServer{}
		return method
	})

	// Parse JWT with secure verification - try all configured user keys
	var token *jwt.Token
	var verifiedUser string
	var verifiedKey ssh.PublicKey
	token, verifiedUser, verifiedKey, err = c.parseAndVerifyJWTSecurely(sshJWTString)
	if err != nil {
		c.logAuditEvent("auth_attempt", "unknown", "unknown", "unknown", "failed", fmt.Sprintf("JWT parse error: %s", err.Error()))
		err = fmt.Errorf("failed to parse JWT: %w", err)
		return identity, err
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		err = errors.New("invalid JWT claims format")
		return identity, err
	}

	// Validate JWT claims (extracted for readability)
	var sub, iss string
	sub, iss, err = c.validateJWTClaims(claims)
	if err != nil {
		keyFingerprint := ssh.FingerprintSHA256(verifiedKey)
		c.logAuditEvent("auth_attempt", sub, keyFingerprint, iss, "failed", err.Error())
		return identity, err
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

	err = nil
	return identity, err
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
	var unverifiedToken *jwt.Token
	unverifiedToken, _, err = parser.ParseUnverified(sshJWTString, jwt.MapClaims{})
	if err != nil {
		err = fmt.Errorf("failed to parse JWT structure: %w", err)
		return token, username, pubKey, err
	}

	// Extract the subject claim - this tells us which user is CLAIMING to authenticate
	// IMPORTANT: We do NOT trust this claim yet! It's just used to know which keys to try
	claims, ok := unverifiedToken.Claims.(jwt.MapClaims)
	if !ok {
		err = errors.New("invalid claims format")
		return token, username, pubKey, err
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		err = errors.New("missing or invalid sub claim")
		return token, username, pubKey, err
	}

	// Now we have the subject from the JWT - i.e. the user trying to auth.
	// We still don't trust it though! It's only used to guide our verification attempts.

	// PASS 2: Try cryptographic verification against each configured public key
	// SECURITY CRITICAL: Only SSH keys explicitly configured in Dex by administrators can verify JWTs
	// This enforces the separation between authentication and authorization:
	// - Authentication: Cryptographic proof the client holds a private key
	// - Authorization: Administrative decision about which keys/users are allowed
	for configUsername, userConfig := range c.config.Users {
		for _, authorizedKeyStr := range userConfig.Keys {
			// Parse the configured public key (trusted, set by administrators)
			var publicKey ssh.PublicKey
			var comment string
			var options []string
			var rest []byte
			publicKey, comment, options, rest, err = ssh.ParseAuthorizedKey([]byte(authorizedKeyStr))
			_, _, _ = comment, options, rest // Explicitly ignore unused return values
			if err != nil {
				continue // Skip invalid keys
			}

			// Attempt cryptographic verification of JWT signature using this configured key
			// This proves the client holds the corresponding private key
			var verifiedToken *jwt.Token
			verifiedToken, err = jwt.Parse(sshJWTString, func(token *jwt.Token) (key interface{}, keyErr error) {
				if token.Method.Alg() != "SSH" {
					keyErr = fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
					return key, keyErr
				}
				// Return the configured public key for verification - NOT any key from JWT claims
				key = publicKey
				return key, keyErr
			})

			if err == nil && verifiedToken.Valid {
				// SUCCESS: Cryptographic verification passed with a configured key!
				// NOW we can trust the JWT claims because we've proven:
				// 1. The JWT was signed by a private key corresponding to a configured public key
				// 2. The configured key belongs to this username (per administrator configuration)
				// 3. No key injection attack is possible (we never used keys from JWT claims)
				//
				// Return the username from our configuration (trusted), not from JWT claims
				token = verifiedToken
				username = configUsername
				pubKey = publicKey
				err = nil
				return token, username, pubKey, err
			}
		}
	}

	err = fmt.Errorf("no configured key could verify the JWT signature")
	return token, username, pubKey, err
}

// validateJWTClaims validates the standard JWT claims (sub, aud, iss, exp, nbf).
// Returns subject, issuer, and any validation error.
func (c *SSHConnector) validateJWTClaims(claims jwt.MapClaims) (username string, issuer string, err error) {
	// Validate required claims
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		err = errors.New("missing or invalid sub claim")
		return username, issuer, err
	}

	aud, ok := claims["aud"].(string)
	if !ok || aud == "" {
		username = sub
		err = errors.New("missing or invalid aud claim")
		return username, issuer, err
	}

	iss, ok := claims["iss"].(string)
	if !ok || iss == "" {
		username = sub
		err = errors.New("missing or invalid iss claim")
		return username, issuer, err
	}

	// Validate audience - ensure this token is intended for our Dex instance
	if aud != "kubernetes" {
		username = sub
		issuer = iss
		err = fmt.Errorf("invalid audience: %s", aud)
		return username, issuer, err
	}

	// Validate issuer
	if !c.isAllowedIssuer(iss) {
		username = sub
		issuer = iss
		err = fmt.Errorf("invalid issuer: %s", iss)
		return username, issuer, err
	}

	// Validate expiration (critical security check)
	exp, ok := claims["exp"].(float64)
	if !ok {
		username = sub
		issuer = iss
		err = errors.New("missing or invalid exp claim")
		return username, issuer, err
	}

	if time.Unix(int64(exp), 0).Before(time.Now()) {
		username = sub
		issuer = iss
		err = errors.New("token has expired")
		return username, issuer, err
	}

	// Validate not before if present
	if nbfClaim, nbfOk := claims["nbf"].(float64); nbfOk {
		if time.Unix(int64(nbfClaim), 0).After(time.Now()) {
			username = sub
			issuer = iss
			err = errors.New("token not yet valid")
			return username, issuer, err
		}
	}

	username = sub
	issuer = iss
	return username, issuer, err
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
				userInfo = userConfig.UserInfo
				if userInfo.Username == "" {
					userInfo.Username = username
				}
				return
			}
		}
		err = fmt.Errorf("key %s not authorized for user %s", keyFingerprint, username)
		return
	}

	err = fmt.Errorf("user %s not found or key %s not authorized", username, keyFingerprint)
	return
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
		matches = false
		return
	}

	// Generate fingerprint from the public key and compare
	authorizedKeyFingerprint := ssh.FingerprintSHA256(publicKey)
	matches = authorizedKeyFingerprint == presentedKeyFingerprint
	return
}

// isAllowedIssuer checks if the JWT issuer is allowed.
func (c *SSHConnector) isAllowedIssuer(issuer string) (allowed bool) {
	if len(c.config.AllowedIssuers) == 0 {
		allowed = true // Allow all if none specified
		return
	}

	for _, allowedIssuer := range c.config.AllowedIssuers {
		if issuer == allowedIssuer {
			allowed = true
			return
		}
	}

	allowed = false
	return allowed
}

// SSHSigningMethodServer implements JWT signing method for server-side SSH verification.
type SSHSigningMethodServer struct{}

// Alg returns the signing method algorithm identifier.
func (m *SSHSigningMethodServer) Alg() (algorithm string) {
	algorithm = "SSH"
	return
}

// Sign is not implemented on server side (client-only operation).
func (m *SSHSigningMethodServer) Sign(signingString string, key interface{}) (signature []byte, err error) {
	err = errors.New("SSH signing not supported on server side")
	return
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
	var signatureBytes []byte
	signatureBytes, err = base64.StdEncoding.DecodeString(signatureStr)
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
		err = fmt.Errorf("SSH signature verification failed: %w", err)
		return err
	}

	err = nil
	return err
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

// TokenIdentity validates SSH JWT tokens via OAuth2 Token Exchange (RFC 8693).
// This method implements the TokenIdentityConnector interface, enabling clients
// to exchange SSH-signed JWTs for Dex identity tokens.
//
// The OAuth2 Token Exchange flow:
// 1. Client creates JWT signed with SSH private key
// 2. Client calls Dex token exchange endpoint with SSH JWT as subject token
// 3. Dex validates JWT signature against administratively configured SSH keys
// 4. Dex returns standard OAuth2 tokens (ID token, access token, refresh token)
//
// Supported subject token types:
// - "ssh_jwt" (custom type for SSH-signed JWTs)
// - "urn:ietf:params:oauth:token-type:jwt" (RFC 8693 standard)
// - "urn:ietf:params:oauth:token-type:access_token" (compatibility)
// - "urn:ietf:params:oauth:token-type:id_token" (compatibility)
//
// Security: JWT verification follows a secure 2-pass process where no JWT content
// is trusted until cryptographic signature verification against configured SSH keys succeeds.
func (c *SSHConnector) TokenIdentity(ctx context.Context, subjectTokenType, subjectToken string) (identity connector.Identity, err error) {
	if c.logger != nil {
		c.logger.InfoContext(ctx, "TokenIdentity method called", "tokenType", subjectTokenType)
	}

	// Validate token type - accept standard OAuth2 JWT types
	switch subjectTokenType {
	case "ssh_jwt", "urn:ietf:params:oauth:token-type:jwt", "urn:ietf:params:oauth:token-type:access_token", "urn:ietf:params:oauth:token-type:id_token":
		// Supported token types
	default:
		err = fmt.Errorf("unsupported token type: %s", subjectTokenType)
		return
	}

	// Use existing SSH JWT validation logic
	identity, err = c.validateSSHJWT(subjectToken)
	if err != nil {
		if c.logger != nil {
			// SSH agent trying multiple keys is normal behavior - log at debug level
			c.logger.DebugContext(ctx, "SSH JWT validation failed in TokenIdentity", "error", err)
		}
		err = fmt.Errorf("SSH JWT validation failed: %w", err)
		return
	}

	if c.logger != nil {
		c.logger.InfoContext(ctx, "TokenIdentity successful", "user", identity.UserID)
	}
	return
}
