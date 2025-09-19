// Package ssh implements a connector that authenticates using SSH keys
package ssh

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
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

// SSHConnector implements the Dex connector interface for SSH key authentication.
type SSHConnector struct {
	config Config
	logger *slog.Logger
}

// Compile-time interface assertion to ensure SSHConnector implements Connector interface
var _ connector.Connector = &SSHConnector{}

// Open creates a new SSH connector.
// Uses slog.Logger for compatibility with Dex v2.44.0+.
func (c *Config) Open(id string, logger *slog.Logger) (connector.Connector, error) {
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
		config: config,
		logger: logger,
	}, nil
}

// LoginURL returns the URL for SSH-based login.
func (c *SSHConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, error) {
	// For SSH authentication, we don't use a traditional login URL
	// Instead, clients directly present SSH-signed JWTs
	return fmt.Sprintf("%s?state=%s&ssh_auth=true", callbackURL, state), nil
}

// HandleCallback processes the SSH authentication callback.
func (c *SSHConnector) HandleCallback(scopes connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
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
		return identity, errors.New("no SSH JWT or authorization code provided")
	}

	// Validate and extract identity - this will now work with Dex's standard token generation
	return c.validateSSHJWT(sshJWT)
}

// validateSSHJWT validates an SSH-signed JWT and extracts user identity.
// SECURITY FIX: Now uses configured keys for verification instead of trusting keys from JWT claims.
func (c *SSHConnector) validateSSHJWT(sshJWTString string) (connector.Identity, error) {
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
	identity := connector.Identity{
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
func (c *SSHConnector) parseAndVerifyJWTSecurely(sshJWTString string) (*jwt.Token, string, ssh.PublicKey, error) {
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
func (c *SSHConnector) validateJWTClaims(claims jwt.MapClaims) (string, string, error) {
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
func (c *SSHConnector) findUserByUsernameAndKey(username, keyFingerprint string) (UserInfo, error) {
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
func (c *SSHConnector) isKeyMatch(authorizedKey, presentedKeyFingerprint string) bool {
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
func (c *SSHConnector) isAllowedIssuer(issuer string) bool {
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
func (m *SSHSigningMethodServer) Sign(signingString string, key interface{}) ([]byte, error) {
	return nil, errors.New("SSH signing not supported on server side")
}

// Verify verifies the JWT signature using the SSH public key.
func (m *SSHSigningMethodServer) Verify(signingString string, signature []byte, key interface{}) error {
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
func (c *SSHConnector) TokenIdentity(ctx context.Context, subjectTokenType, subjectToken string) (connector.Identity, error) {
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
	identity, err := c.validateSSHJWT(subjectToken)
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
