package storage

import (
	"context"
	"crypto"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"io"
	"math/big"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
)

var (
	// ErrNotFound is the error returned by storages if a resource cannot be found.
	ErrNotFound = errors.New("not found")

	// ErrAlreadyExists is the error returned by storages if a resource ID is taken during a create.
	ErrAlreadyExists = errors.New("ID already exists")
)

// Kubernetes only allows lower case letters for names.
//
// TODO(ericchiang): refactor ID creation onto the storage.
var encoding = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567")

// Valid characters for user codes
const validUserCharacters = "BCDFGHJKLMNPQRSTVWXZ"

// NewDeviceCode returns a 32 char alphanumeric cryptographically secure string
func NewDeviceCode() string {
	return newSecureID(32)
}

// NewID returns a random string which can be used as an ID for objects.
func NewID() string {
	return newSecureID(16)
}

func newSecureID(len int) string {
	buff := make([]byte, len) // random ID.
	if _, err := io.ReadFull(rand.Reader, buff); err != nil {
		panic(err)
	}
	// Avoid the identifier to begin with number and trim padding
	return string(buff[0]%26+'a') + strings.TrimRight(encoding.EncodeToString(buff[1:]), "=")
}

// NewHMACKey returns a random key which can be used in the computation of an HMAC
func NewHMACKey(h crypto.Hash) []byte {
	return []byte(newSecureID(h.Size()))
}

// GCResult returns the number of objects deleted by garbage collection.
type GCResult struct {
	AuthRequests   int64
	AuthCodes      int64
	DeviceRequests int64
	DeviceTokens   int64
	AuthSessions   int64
}

// IsEmpty returns whether the garbage collection result is empty or not.
func (g *GCResult) IsEmpty() bool {
	return g.AuthRequests == 0 &&
		g.AuthCodes == 0 &&
		g.DeviceRequests == 0 &&
		g.DeviceTokens == 0 &&
		g.AuthSessions == 0
}

// Storage is the storage interface used by the server. Implementations are
// required to be able to perform atomic compare-and-swap updates and either
// support timezones or standardize on UTC.
type Storage interface {
	Close() error

	// TODO(ericchiang): Let the storages set the IDs of these objects.
	CreateAuthRequest(ctx context.Context, a AuthRequest) error
	CreateClient(ctx context.Context, c Client) error
	CreateAuthCode(ctx context.Context, c AuthCode) error
	CreateRefresh(ctx context.Context, r RefreshToken) error
	CreatePassword(ctx context.Context, p Password) error
	CreateOfflineSessions(ctx context.Context, s OfflineSessions) error
	CreateUserIdentity(ctx context.Context, u UserIdentity) error
	CreateAuthSession(ctx context.Context, s AuthSession) error
	CreateConnector(ctx context.Context, c Connector) error
	CreateDeviceRequest(ctx context.Context, d DeviceRequest) error
	CreateDeviceToken(ctx context.Context, d DeviceToken) error

	// TODO(ericchiang): return (T, bool, error) so we can indicate not found
	// requests that way instead of using ErrNotFound.
	GetAuthRequest(ctx context.Context, id string) (AuthRequest, error)
	GetAuthCode(ctx context.Context, id string) (AuthCode, error)
	GetClient(ctx context.Context, id string) (Client, error)
	GetKeys(ctx context.Context) (Keys, error)
	GetRefresh(ctx context.Context, id string) (RefreshToken, error)
	GetPassword(ctx context.Context, email string) (Password, error)
	GetOfflineSessions(ctx context.Context, userID string, connID string) (OfflineSessions, error)
	GetUserIdentity(ctx context.Context, userID, connectorID string) (UserIdentity, error)
	GetAuthSession(ctx context.Context, userID, connectorID string) (AuthSession, error)
	GetConnector(ctx context.Context, id string) (Connector, error)
	GetDeviceRequest(ctx context.Context, userCode string) (DeviceRequest, error)
	GetDeviceToken(ctx context.Context, deviceCode string) (DeviceToken, error)

	ListClients(ctx context.Context) ([]Client, error)
	ListRefreshTokens(ctx context.Context) ([]RefreshToken, error)
	ListPasswords(ctx context.Context) ([]Password, error)
	ListConnectors(ctx context.Context) ([]Connector, error)
	ListUserIdentities(ctx context.Context) ([]UserIdentity, error)
	ListAuthSessions(ctx context.Context) ([]AuthSession, error)

	// Delete methods MUST be atomic.
	DeleteAuthRequest(ctx context.Context, id string) error
	DeleteAuthCode(ctx context.Context, code string) error
	DeleteClient(ctx context.Context, id string) error
	DeleteRefresh(ctx context.Context, id string) error
	DeletePassword(ctx context.Context, email string) error
	DeleteOfflineSessions(ctx context.Context, userID string, connID string) error
	DeleteUserIdentity(ctx context.Context, userID, connectorID string) error
	DeleteAuthSession(ctx context.Context, userID, connectorID string) error
	DeleteConnector(ctx context.Context, id string) error

	// Update methods take a function for updating an object then performs that update within
	// a transaction. "updater" functions may be called multiple times by a single update call.
	//
	// Because new fields may be added to resources, updaters should only modify existing
	// fields on the old object rather then creating new structs. For example:
	//
	//		updater := func(old storage.Client) (storage.Client, error) {
	//			old.Secret = newSecret
	//			return old, nil
	//		}
	//		if err := s.UpdateClient(clientID, updater); err != nil {
	//			// update failed, handle error
	//		}
	//
	UpdateClient(ctx context.Context, id string, updater func(old Client) (Client, error)) error
	UpdateKeys(ctx context.Context, updater func(old Keys) (Keys, error)) error
	UpdateAuthRequest(ctx context.Context, id string, updater func(a AuthRequest) (AuthRequest, error)) error
	UpdateRefreshToken(ctx context.Context, id string, updater func(r RefreshToken) (RefreshToken, error)) error
	UpdatePassword(ctx context.Context, email string, updater func(p Password) (Password, error)) error
	UpdateOfflineSessions(ctx context.Context, userID string, connID string, updater func(s OfflineSessions) (OfflineSessions, error)) error
	UpdateUserIdentity(ctx context.Context, userID, connectorID string, updater func(u UserIdentity) (UserIdentity, error)) error
	UpdateAuthSession(ctx context.Context, userID, connectorID string, updater func(s AuthSession) (AuthSession, error)) error
	UpdateConnector(ctx context.Context, id string, updater func(c Connector) (Connector, error)) error
	UpdateDeviceToken(ctx context.Context, deviceCode string, updater func(t DeviceToken) (DeviceToken, error)) error

	// GarbageCollect deletes all expired AuthCodes,
	// AuthRequests, DeviceRequests, and DeviceTokens.
	GarbageCollect(ctx context.Context, now time.Time) (GCResult, error)
}

// Client represents an OAuth2 client.
//
// For further reading see:
//   - Trusted peers: https://developers.google.com/identity/protocols/CrossClientAuth
//   - Public clients: https://developers.google.com/api-client-library/python/auth/installed-app
type Client struct {
	// Client ID and secret used to identify the client.
	ID        string `json:"id"`
	IDEnv     string `json:"idEnv"`
	Secret    string `json:"secret"`
	SecretEnv string `json:"secretEnv"`

	// A registered set of redirect URIs. When redirecting from dex to the client, the URI
	// requested to redirect to MUST match one of these values, unless the client is "public".
	RedirectURIs []string `json:"redirectURIs"`

	// PostLogoutRedirectURIs is a registered set of URIs that the client can redirect to
	// after logout. Per OIDC RP-Initiated Logout Section 2, the post_logout_redirect_uri
	// parameter MUST match one of these values.
	PostLogoutRedirectURIs []string `json:"postLogoutRedirectURIs"`

	// TrustedPeers are a list of peers which can issue tokens on this client's behalf using
	// the dynamic "oauth2:server:client_id:(client_id)" scope. If a peer makes such a request,
	// this client's ID will appear as the ID Token's audience.
	//
	// Clients inherently trust themselves.
	TrustedPeers []string `json:"trustedPeers"`

	// Public clients must use either use a redirectURL 127.0.0.1:X or "urn:ietf:wg:oauth:2.0:oob"
	Public bool `json:"public"`

	// Name and LogoURL used when displaying this client to the end user.
	Name    string `json:"name"`
	LogoURL string `json:"logoURL"`

	// AllowedConnectors is a list of connector IDs that the client is allowed to use for authentication.
	// If empty, all connectors are allowed.
	AllowedConnectors []string `json:"allowedConnectors"`

	// MFAChain is an ordered list of MFA authenticator IDs that a user must complete
	// during login. Empty means no MFA required.
	MFAChain []string `json:"mfaChain"`

	// SSOSharedWith defines which other clients can reuse this client's authentication session.
	// When a user is authenticated for this client, clients listed here can skip authentication.
	// Special value "*" means share with all clients (Keycloak-like realm-wide SSO).
	// nil means use ssoSharedWithDefault from sessions config.
	// Empty slice [] means explicitly share with no one.
	SSOSharedWith []string `json:"ssoSharedWith" yaml:"ssoSharedWith"`

	// ClientCredentialsClaims holds identity claims used when issuing tokens via the
	// client_credentials grant. Kept separate from core Client fields to avoid mixing
	// application identity (ID, secret, redirect URIs) with user-like identity attributes.
	ClientCredentialsClaims *ClientCredentialsClaims `json:"clientCredentialsClaims,omitempty"`
}

// ClientCredentialsClaims contains claims that are included in tokens issued via
// the client_credentials grant. This is scoped to client_credentials to keep the
// Client struct focused on application-level concerns.
type ClientCredentialsClaims struct {
	Groups []string `json:"groups,omitempty"`
}

// Claims represents the ID Token claims supported by the server.
type Claims struct {
	UserID            string
	Username          string
	PreferredUsername string
	Email             string
	EmailVerified     bool

	Groups []string
}

// PKCE is a container for the data needed to perform Proof Key for Code Exchange (RFC 7636) auth flow
type PKCE struct {
	CodeChallenge       string
	CodeChallengeMethod string
}

// AuthRequest represents a OAuth2 client authorization request. It holds the state
// of a single auth flow up to the point that the user authorizes the client.
type AuthRequest struct {
	// ID used to identify the authorization request.
	ID string

	// ID of the client requesting authorization from a user.
	ClientID string

	// Values parsed from the initial request. These describe the resources the client is
	// requesting as well as values describing the form of the response.
	ResponseTypes []string
	Scopes        []string
	RedirectURI   string
	Nonce         string
	State         string

	// The client has indicated that the end user must be shown an approval prompt
	// on all requests. The server cannot cache their initial action for subsequent
	// attempts.
	ForceApprovalPrompt bool

	// OIDC prompt parameter. Controls authentication and consent UI behavior.
	// Values: "none", "login", "consent", "select_account".
	Prompt string

	// MaxAge is the OIDC max_age parameter — maximum allowable elapsed time
	// in seconds since the user last actively authenticated.
	// -1 means not specified.
	MaxAge int

	// AuthTime is when the user last actively authenticated (entered credentials).
	// Set during finalizeLogin (= now) or trySessionLogin (= UserIdentity.LastLogin).
	// Used in ID token as "auth_time" claim.
	AuthTime time.Time

	Expiry time.Time

	// Has the user proved their identity through a backing identity provider?
	//
	// If false, the following fields are invalid.
	LoggedIn bool

	// The identity of the end user. Generally nil until the user authenticates
	// with a backend.
	Claims Claims

	// The connector used to login the user and any data the connector wishes to persists.
	// Set when the user authenticates.
	ConnectorID   string
	ConnectorData []byte

	// PKCE CodeChallenge and CodeChallengeMethod
	PKCE PKCE

	// HMACKey is used when generating an AuthRequest-specific HMAC
	HMACKey []byte

	// MFAValidated is set to true if the user has completed multi-factor authentication.
	MFAValidated bool

	// WebAuthnSessionData stores temporary WebAuthn ceremony data (challenge, etc.)
	// between Begin and Finish calls. JSON-encoded webauthn.SessionData.
	WebAuthnSessionData []byte
}

// AuthCode represents a code which can be exchanged for an OAuth2 token response.
//
// This value is created once an end user has authorized a client, the server has
// redirect the end user back to the client, but the client hasn't exchanged the
// code for an access_token and id_token.
type AuthCode struct {
	// Actual string returned as the "code" value.
	ID string

	// The client this code value is valid for. When exchanging the code for a
	// token response, the client must use its client_secret to authenticate.
	ClientID string

	// As part of the OAuth2 spec when a client makes a token request it MUST
	// present the same redirect_uri as the initial redirect. This values is saved
	// to make this check.
	//
	// https://tools.ietf.org/html/rfc6749#section-4.1.3
	RedirectURI string

	// If provided by the client in the initial request, the provider MUST create
	// a ID Token with this nonce in the JWT payload.
	Nonce string

	// Scopes authorized by the end user for the client.
	Scopes []string

	// Authentication data provided by an upstream source.
	ConnectorID   string
	ConnectorData []byte
	Claims        Claims

	Expiry time.Time

	// AuthTime is when the user last actively authenticated.
	// Carried over from AuthRequest to include in ID tokens.
	AuthTime time.Time

	// PKCE CodeChallenge and CodeChallengeMethod
	PKCE PKCE
}

// RefreshToken is an OAuth2 refresh token which allows a client to request new
// tokens on the end user's behalf.
type RefreshToken struct {
	ID string

	// A single token that's rotated every time the refresh token is refreshed.
	//
	// May be empty.
	Token         string
	ObsoleteToken string

	CreatedAt time.Time
	LastUsed  time.Time

	// Client this refresh token is valid for.
	ClientID string

	// Authentication data provided by an upstream source.
	ConnectorID   string
	ConnectorData []byte
	Claims        Claims

	// Scopes present in the initial request. Refresh requests may specify a set
	// of scopes different from the initial request when refreshing a token,
	// however those scopes must be encompassed by this set.
	Scopes []string

	// Nonce value supplied during the initial redirect. This is required to be part
	// of the claims of any future id_token generated by the client.
	Nonce string
}

// RefreshTokenRef is a reference object that contains metadata about refresh tokens.
type RefreshTokenRef struct {
	ID string

	// Client the refresh token is valid for.
	ClientID string

	CreatedAt time.Time
	LastUsed  time.Time
}

// MFASecret stores the enrollment state and secret for an MFA authenticator.
// Note: Secret is stored without encryption. Encrypting secrets at rest is the
// responsibility of the storage backend (e.g., encrypted etcd, disk encryption).
type MFASecret struct {
	AuthenticatorID string    `json:"authenticatorID"`
	Type            string    `json:"type"`
	Secret          string    `json:"secret"`
	Confirmed       bool      `json:"confirmed"`
	CreatedAt       time.Time `json:"createdAt"`
}

// WebAuthnCredential stores a registered WebAuthn credential for a user.
type WebAuthnCredential struct {
	CredentialID    []byte    `json:"credentialID"`
	PublicKey       []byte    `json:"publicKey"`
	AttestationType string    `json:"attestationType"`
	AAGUID          []byte    `json:"aaguid"`
	SignCount       uint32    `json:"signCount"`
	CloneWarning    bool      `json:"cloneWarning"`
	Transport       []string  `json:"transport"`
	BackupEligible  bool      `json:"backupEligible"`
	BackupState     bool      `json:"backupState"`
	DisplayName     string    `json:"displayName"`
	CreatedAt       time.Time `json:"createdAt"`
}

// UserIdentity represents persistent per-user identity data.
type UserIdentity struct {
	UserID              string
	ConnectorID         string
	Claims              Claims
	Consents            map[string][]string             // clientID -> approved scopes
	MFASecrets          map[string]*MFASecret           // authenticatorID -> secret
	WebAuthnCredentials map[string][]WebAuthnCredential // authenticatorID -> credentials
	CreatedAt           time.Time
	LastLogin           time.Time
	BlockedUntil        time.Time
}

// ClientAuthState represents authentication state for a specific client within an auth session.
type ClientAuthState struct {
	Active            bool
	ExpiresAt         time.Time
	LastActivity      time.Time
	LastTokenIssuedAt time.Time
}

// LogoutState holds RP parameters saved in the auth session during logout.
// These are written before the upstream logout redirect and read back in the callback.
type LogoutState struct {
	PostLogoutRedirectURI string
	State                 string // RP's opaque state parameter
	ClientID              string
	ConnectorID           string
}

// AuthSession represents a user's authentication session from a specific connector.
// Keyed by composite (UserID, ConnectorID), similar to OfflineSessions.
// The Nonce field is a random value included in the session cookie to prevent forgery.
//
// TODO(nabokihms): support multiple sessions in one browser by storing multiple
// session references in the cookie (e.g. "ref1|ref2") so that different users
// can maintain independent sessions in the same browser.
type AuthSession struct {
	UserID       string
	ConnectorID  string
	Nonce        string                      // random, included in cookie for verification
	ClientStates map[string]*ClientAuthState // clientID -> auth state
	CreatedAt    time.Time
	LastActivity time.Time
	IPAddress    string
	UserAgent    string

	// AbsoluteExpiry is CreatedAt + AbsoluteLifetime, set once at creation.
	AbsoluteExpiry time.Time
	// IdleExpiry is LastActivity + ValidIfNotUsedFor, updated on every activity.
	IdleExpiry time.Time

	// LogoutState is set during RP-Initiated Logout before redirecting to the
	// upstream provider. The callback handler reads it back to complete the flow.
	// Nil when no logout is in progress.
	LogoutState *LogoutState
}

// OfflineSessions objects are sessions pertaining to users with refresh tokens.
type OfflineSessions struct {
	// UserID of an end user who has logged into the server.
	UserID string

	// The ID of the connector used to login the user.
	ConnID string

	// Refresh is a hash table of refresh token reference objects
	// indexed by the ClientID of the refresh token.
	Refresh map[string]*RefreshTokenRef

	// Authentication data provided by an upstream source.
	ConnectorData []byte
}

// Password is an email to password mapping managed by the storage.
type Password struct {
	// Email and identifying name of the password. Emails are assumed to be valid and
	// determining that an end-user controls the address is left to an outside application.
	//
	// Emails are case insensitive and should be standardized by the storage.
	//
	// Storages that don't support an extended character set for IDs, such as '.' and '@'
	// (cough cough, kubernetes), must map this value appropriately.
	Email string `json:"email"`

	// Bcrypt encoded hash of the password. This package enforces a min cost value of 10
	Hash []byte `json:"hash"`

	// Bcrypt encoded hash of the password set in environment variable of this name.
	HashFromEnv string `json:"hashFromEnv"`

	// Optional username to display. NOT used during login.
	Username string `json:"username"`

	// Optional full name for OIDC "name" claim.
	// Defaults to Username when empty.
	Name string `json:"name"`

	// Optional preferred username for OIDC "preferred_username" claim.
	PreferredUsername string `json:"preferredUsername"`

	// Optional value for OIDC "email_verified" claim.
	// Defaults to true for backwards compatibility when nil.
	EmailVerified *bool `json:"emailVerified,omitempty"`

	// Randomly generated user ID. This is NOT the primary ID of the Password object.
	UserID string `json:"userID"`

	// Groups assigned to the user
	Groups []string `json:"groups"`
}

// Connector is an object that contains the metadata about connectors used to login to Dex.
type Connector struct {
	// ID that will uniquely identify the connector object.
	ID string `json:"id"`
	// The Type of the connector. E.g. 'oidc' or 'ldap'
	Type string `json:"type"`
	// The Name of the connector that is used when displaying it to the end user.
	Name string `json:"name"`
	// ResourceVersion is the static versioning used to keep track of dynamic configuration
	// changes to the connector object made by the API calls.
	ResourceVersion string `json:"resourceVersion"`
	// Config holds all the configuration information specific to the connector type. Since there
	// no generic struct we can use for this purpose, it is stored as a byte stream.
	//
	// NOTE: This is a bug. The JSON tag should be `config`.
	// However, fixing this requires migrating Kubernetes objects for all previously created connectors,
	// or making Dex reading both tags and act accordingly.
	Config []byte `json:"email"`

	// GrantTypes is a list of grant types that this connector is allowed to be used with.
	// If empty, all grant types are allowed.
	GrantTypes []string `json:"grantTypes,omitempty"`
}

// VerificationKey is a rotated signing key which can still be used to verify
// signatures.
type VerificationKey struct {
	PublicKey *jose.JSONWebKey `json:"publicKey"`
	Expiry    time.Time        `json:"expiry"`
}

// Keys hold encryption and signing keys.
type Keys struct {
	// Key for creating and verifying signatures. These may be nil.
	SigningKey    *jose.JSONWebKey
	SigningKeyPub *jose.JSONWebKey

	// Old signing keys which have been rotated but can still be used to validate
	// existing signatures.
	VerificationKeys []VerificationKey

	// The next time the signing key will rotate.
	//
	// For caching purposes, implementations MUST NOT update keys before this time.
	NextRotation time.Time
}

// NewUserCode returns a randomized 8 character user code for the device flow.
// No vowels are included to prevent accidental generation of words
func NewUserCode() string {
	code := randomString(8)
	return code[:4] + "-" + code[4:]
}

func randomString(n int) string {
	v := big.NewInt(int64(len(validUserCharacters)))
	bytes := make([]byte, n)
	for i := 0; i < n; i++ {
		c, _ := rand.Int(rand.Reader, v)
		bytes[i] = validUserCharacters[c.Int64()]
	}
	return string(bytes)
}

// DeviceRequest represents an OIDC device authorization request. It holds the state of a device request until the user
// authenticates using their user code or the expiry time passes.
type DeviceRequest struct {
	// The code the user will enter in a browser
	UserCode string
	// The unique device code for device authentication
	DeviceCode string
	// The client ID the code is for
	ClientID string
	// The Client Secret
	ClientSecret string
	// The scopes the device requests
	Scopes []string
	// The expire time
	Expiry time.Time
}

// DeviceToken is a structure which represents the actual token of an authorized device and its rotation parameters
type DeviceToken struct {
	DeviceCode          string
	Status              string
	Token               string
	Expiry              time.Time
	LastRequestTime     time.Time
	PollIntervalSeconds int
	PKCE                PKCE
}
