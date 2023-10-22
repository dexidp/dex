package storage

import (
	"crypto"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"io"
	"math/big"
	"strings"
	"time"

	jose "gopkg.in/square/go-jose.v2"
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
}

// IsEmpty returns whether the garbage collection result is empty or not.
func (g *GCResult) IsEmpty() bool {
	return g.AuthRequests == 0 &&
		g.AuthCodes == 0 &&
		g.DeviceRequests == 0 &&
		g.DeviceTokens == 0
}

// Storage is the storage interface used by the server. Implementations are
// required to be able to perform atomic compare-and-swap updates and either
// support timezones or standardize on UTC.
type Storage interface {
	Close() error

	// TODO(ericchiang): Let the storages set the IDs of these objects.
	CreateAuthRequest(a AuthRequest) error
	CreateClient(c Client) error
	CreateAuthCode(c AuthCode) error
	CreateRefresh(r RefreshToken) error
	CreatePassword(p Password) error
	CreateOfflineSessions(s OfflineSessions) error
	CreateConnector(c Connector) error
	CreateDeviceRequest(d DeviceRequest) error
	CreateDeviceToken(d DeviceToken) error

	// TODO(ericchiang): return (T, bool, error) so we can indicate not found
	// requests that way instead of using ErrNotFound.
	GetAuthRequest(id string) (AuthRequest, error)
	GetAuthCode(id string) (AuthCode, error)
	GetClient(id string) (Client, error)
	GetKeys() (Keys, error)
	GetRefresh(id string) (RefreshToken, error)
	GetPassword(email string) (Password, error)
	GetOfflineSessions(userID string, connID string) (OfflineSessions, error)
	GetConnector(id string) (Connector, error)
	GetDeviceRequest(userCode string) (DeviceRequest, error)
	GetDeviceToken(deviceCode string) (DeviceToken, error)

	ListClients() ([]Client, error)
	ListRefreshTokens() ([]RefreshToken, error)
	ListPasswords() ([]Password, error)
	ListConnectors() ([]Connector, error)

	// Delete methods MUST be atomic.
	DeleteAuthRequest(id string) error
	DeleteAuthCode(code string) error
	DeleteClient(id string) error
	DeleteRefresh(id string) error
	DeletePassword(email string) error
	DeleteOfflineSessions(userID string, connID string) error
	DeleteConnector(id string) error

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
	UpdateClient(id string, updater func(old Client) (Client, error)) error
	UpdateKeys(updater func(old Keys) (Keys, error)) error
	UpdateAuthRequest(id string, updater func(a AuthRequest) (AuthRequest, error)) error
	UpdateRefreshToken(id string, updater func(r RefreshToken) (RefreshToken, error)) error
	UpdatePassword(email string, updater func(p Password) (Password, error)) error
	UpdateOfflineSessions(userID string, connID string, updater func(s OfflineSessions) (OfflineSessions, error)) error
	UpdateConnector(id string, updater func(c Connector) (Connector, error)) error
	UpdateDeviceToken(deviceCode string, updater func(t DeviceToken) (DeviceToken, error)) error

	// GarbageCollect deletes all expired AuthCodes,
	// AuthRequests, DeviceRequests, and DeviceTokens.
	GarbageCollect(now time.Time) (GCResult, error)
}

// Client represents an OAuth2 client.
//
// For further reading see:
//   - Trusted peers: https://developers.google.com/identity/protocols/CrossClientAuth
//   - Public clients: https://developers.google.com/api-client-library/python/auth/installed-app
type Client struct {
	// Client ID and secret used to identify the client.
	ID        string `json:"id" yaml:"id"`
	IDEnv     string `json:"idEnv" yaml:"idEnv"`
	Secret    string `json:"secret" yaml:"secret"`
	SecretEnv string `json:"secretEnv" yaml:"secretEnv"`

	// A registered set of redirect URIs. When redirecting from dex to the client, the URI
	// requested to redirect to MUST match one of these values, unless the client is "public".
	RedirectURIs []string `json:"redirectURIs" yaml:"redirectURIs"`

	// TrustedPeers are a list of peers which can issue tokens on this client's behalf using
	// the dynamic "oauth2:server:client_id:(client_id)" scope. If a peer makes such a request,
	// this client's ID will appear as the ID Token's audience.
	//
	// Clients inherently trust themselves.
	TrustedPeers []string `json:"trustedPeers" yaml:"trustedPeers"`

	// Public clients must use either use a redirectURL 127.0.0.1:X or "urn:ietf:wg:oauth:2.0:oob"
	Public bool `json:"public" yaml:"public"`

	// Name and LogoURL used when displaying this client to the end user.
	Name    string `json:"name" yaml:"name"`
	LogoURL string `json:"logoURL" yaml:"logoURL"`
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

	// Randomly generated user ID. This is NOT the primary ID of the Password object.
	UserID string `json:"userID"`
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
