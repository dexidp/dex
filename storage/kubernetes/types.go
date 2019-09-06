package kubernetes

import (
	"strings"
	"time"

	jose "gopkg.in/square/go-jose.v2"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/kubernetes/k8sapi"
)

const (
	// Kinds
	kindAuthCode        = "AuthCode"
	kindAuthRequest     = "AuthRequest"
	kindClient          = "OAuth2Client"
	kindRefreshToken    = "RefreshToken"
	kindKeys            = "SigningKey"
	kindPassword        = "Password"
	kindOfflineSessions = "OfflineSessions"
	kindConnector       = "Connector"

	// Resources
	resourceAuthCode        = "authcodes"
	resourceAuthRequest     = "authrequests"
	resourceClient          = "oauth2clients"
	resourceRefreshToken    = "refreshtokens"
	resourceKeys            = "signingkeies" // Kubernetes attempts to pluralize.
	resourcePassword        = "passwords"
	resourceOfflineSessions = "offlinesessionses" // Again attempts to pluralize.
	resourceConnector       = "connectors"

	// Singular Resources for CRD references
	resourceAuthCodeSingular        = "authcode"
	resourceAuthRequestSingular     = "authrequest"
	resourceClientSingular          = "oauth2client"
	resourceRefreshTokenSingular    = "refreshtoken"
	resourceKeysSingular            = "signingkey"
	resourcePasswordSingular        = "password"
	resourceOfflineSessionsSingular = "offlinesessions"
	resourceConnectorSingular       = "connector"

	// APIs
	customResourceDefinitionAPI       = "apiextensions.k8s.io/v1beta1"
	customResourceDefinitionNamespace = ""
	customResourceDefinitionResource  = "customresourcedefinitions"

	// Dex k8s-specific API
	apiGroup    = "dex.coreos.com"
	apiFullPath = apiGroup + "/v1"

	// CRD names
	crdNameAuthCodes         = "authcodes." + apiGroup
	crdNameAuthRequests      = "authrequests." + apiGroup
	crdNameOAuth2Clients     = "oauth2clients." + apiGroup
	crdNameRefreshTokens     = "refreshtokens." + apiGroup
	crdNameSigningKeies      = "signingkeies." + apiGroup
	crdNamePasswords         = "passwords." + apiGroup
	crdNameOfflineSessionses = "offlinesessionses." + apiGroup
	crdNameConnectors        = "connectors." + apiGroup

	// CRD Versions
	crdVersionAuthCodes           = "v1"
	crdVersionAuthRequests        = "v1"
	crdVersionOAuth2Clients       = "v1"
	crdVersionRefreshTokenClients = "v1"
	crdVersionSigningKeies        = "v1"
	crdVersionPasswords           = "v1"
	crdVersionOfflineSessionses   = "v1"
	crdVersionConnectors          = "v1"

	// There will only ever be a single keys resource. Maintain this by setting a
	// common name.
	keysName = "openid-connect-keys"
)

var crdMeta = k8sapi.TypeMeta{
	APIVersion: "apiextensions.k8s.io/v1beta1",
	Kind:       "CustomResourceDefinition",
}

// The set of custom resource definitions required by the storage. These are managed by
// the storage so it can migrate itself by creating new resources.
var customResourceDefinitions = []k8sapi.CustomResourceDefinition{
	{
		ObjectMeta: k8sapi.ObjectMeta{
			Name: crdNameAuthCodes,
		},
		TypeMeta: crdMeta,
		Spec: k8sapi.CustomResourceDefinitionSpec{
			Group:   apiGroup,
			Version: crdVersionAuthCodes,
			Names: k8sapi.CustomResourceDefinitionNames{
				Plural:   resourceAuthCode,
				Singular: resourceAuthCodeSingular,
				Kind:     kindAuthCode,
			},
		},
	},
	{
		ObjectMeta: k8sapi.ObjectMeta{
			Name: crdNameAuthRequests,
		},
		TypeMeta: crdMeta,
		Spec: k8sapi.CustomResourceDefinitionSpec{
			Group:   apiGroup,
			Version: crdVersionAuthRequests,
			Names: k8sapi.CustomResourceDefinitionNames{
				Plural:   resourceAuthRequest,
				Singular: resourceAuthRequestSingular,
				Kind:     kindAuthRequest,
			},
		},
	},
	{
		ObjectMeta: k8sapi.ObjectMeta{
			Name: crdNameOAuth2Clients,
		},
		TypeMeta: crdMeta,
		Spec: k8sapi.CustomResourceDefinitionSpec{
			Group:   apiGroup,
			Version: crdVersionOAuth2Clients,
			Names: k8sapi.CustomResourceDefinitionNames{
				Plural:   resourceClient,
				Singular: resourceClientSingular,
				Kind:     kindClient,
			},
		},
	},
	{
		ObjectMeta: k8sapi.ObjectMeta{
			Name: crdNameRefreshTokens,
		},
		TypeMeta: crdMeta,
		Spec: k8sapi.CustomResourceDefinitionSpec{
			Group:   apiGroup,
			Version: crdVersionRefreshTokenClients,
			Names: k8sapi.CustomResourceDefinitionNames{
				Plural:   resourceRefreshToken,
				Singular: resourceRefreshTokenSingular,
				Kind:     kindRefreshToken,
			},
		},
	},
	{
		ObjectMeta: k8sapi.ObjectMeta{
			Name: crdNameSigningKeies,
		},
		TypeMeta: crdMeta,
		Spec: k8sapi.CustomResourceDefinitionSpec{
			Group:   apiGroup,
			Version: crdVersionSigningKeies,
			Names: k8sapi.CustomResourceDefinitionNames{
				// `signingkeies` is an artifact from the old TPR pluralization.
				// Users don't directly interact with this value, hence leaving it
				// as is.
				Plural:   resourceKeys,
				Singular: resourceKeysSingular,
				Kind:     kindKeys,
			},
		},
	},
	{
		ObjectMeta: k8sapi.ObjectMeta{
			Name: crdNamePasswords,
		},
		TypeMeta: crdMeta,
		Spec: k8sapi.CustomResourceDefinitionSpec{
			Group:   apiGroup,
			Version: crdVersionPasswords,
			Names: k8sapi.CustomResourceDefinitionNames{
				Plural:   resourcePassword,
				Singular: resourcePasswordSingular,
				Kind:     kindPassword,
			},
		},
	},
	{
		ObjectMeta: k8sapi.ObjectMeta{
			Name: crdNameOfflineSessionses,
		},
		TypeMeta: crdMeta,
		Spec: k8sapi.CustomResourceDefinitionSpec{
			Group:   apiGroup,
			Version: crdVersionOfflineSessionses,
			Names: k8sapi.CustomResourceDefinitionNames{
				Plural:   resourceOfflineSessions,
				Singular: resourceOfflineSessionsSingular,
				Kind:     kindOfflineSessions,
			},
		},
	},
	{
		ObjectMeta: k8sapi.ObjectMeta{
			Name: crdNameConnectors,
		},
		TypeMeta: crdMeta,
		Spec: k8sapi.CustomResourceDefinitionSpec{
			Group:   apiGroup,
			Version: crdVersionConnectors,
			Names: k8sapi.CustomResourceDefinitionNames{
				Plural:   resourceConnector,
				Singular: resourceConnectorSingular,
				Kind:     kindConnector,
			},
		},
	},
}

// Client is a mirrored struct from storage with JSON struct tags and
// Kubernetes type metadata.
type Client struct {
	// Name is a hash of the ID.
	k8sapi.TypeMeta   `json:",inline"`
	k8sapi.ObjectMeta `json:"metadata,omitempty"`

	// ID is immutable, since it's a primary key and should not be changed.
	ID string `json:"id,omitempty"`

	Secret       string   `json:"secret,omitempty"`
	RedirectURIs []string `json:"redirectURIs,omitempty"`
	TrustedPeers []string `json:"trustedPeers,omitempty"`

	Public bool `json:"public"`

	Name    string `json:"name,omitempty"`
	LogoURL string `json:"logoURL,omitempty"`
}

// ClientList is a list of Clients.
type ClientList struct {
	k8sapi.TypeMeta `json:",inline"`
	k8sapi.ListMeta `json:"metadata,omitempty"`
	Clients         []Client `json:"items"`
}

func (cli *client) fromStorageClient(c storage.Client) Client {
	return Client{
		TypeMeta: k8sapi.TypeMeta{
			Kind:       kindClient,
			APIVersion: cli.apiVersion,
		},
		ObjectMeta: k8sapi.ObjectMeta{
			Name:      cli.idToName(c.ID),
			Namespace: cli.namespace,
		},
		ID:           c.ID,
		Secret:       c.Secret,
		RedirectURIs: c.RedirectURIs,
		TrustedPeers: c.TrustedPeers,
		Public:       c.Public,
		Name:         c.Name,
		LogoURL:      c.LogoURL,
	}
}

func toStorageClient(c Client) storage.Client {
	return storage.Client{
		ID:           c.ID,
		Secret:       c.Secret,
		RedirectURIs: c.RedirectURIs,
		TrustedPeers: c.TrustedPeers,
		Public:       c.Public,
		Name:         c.Name,
		LogoURL:      c.LogoURL,
	}
}

// Claims is a mirrored struct from storage with JSON struct tags.
type Claims struct {
	UserID        string   `json:"userID"`
	Username      string   `json:"username"`
	Email         string   `json:"email"`
	EmailVerified bool     `json:"emailVerified"`
	Groups        []string `json:"groups,omitempty"`
}

func fromStorageClaims(i storage.Claims) Claims {
	return Claims{
		UserID:        i.UserID,
		Username:      i.Username,
		Email:         i.Email,
		EmailVerified: i.EmailVerified,
		Groups:        i.Groups,
	}
}

func toStorageClaims(i Claims) storage.Claims {
	return storage.Claims{
		UserID:        i.UserID,
		Username:      i.Username,
		Email:         i.Email,
		EmailVerified: i.EmailVerified,
		Groups:        i.Groups,
	}
}

// AuthRequest is a mirrored struct from storage with JSON struct tags and
// Kubernetes type metadata.
type AuthRequest struct {
	k8sapi.TypeMeta   `json:",inline"`
	k8sapi.ObjectMeta `json:"metadata,omitempty"`

	ClientID      string   `json:"clientID"`
	ResponseTypes []string `json:"responseTypes,omitempty"`
	Scopes        []string `json:"scopes,omitempty"`
	RedirectURI   string   `json:"redirectURI"`

	Nonce string `json:"nonce,omitempty"`
	State string `json:"state,omitempty"`

	// The client has indicated that the end user must be shown an approval prompt
	// on all requests. The server cannot cache their initial action for subsequent
	// attempts.
	ForceApprovalPrompt bool `json:"forceApprovalPrompt,omitempty"`

	LoggedIn bool `json:"loggedIn"`

	// The identity of the end user. Generally nil until the user authenticates
	// with a backend.
	Claims Claims `json:"claims,omitempty"`
	// The connector used to login the user. Set when the user authenticates.
	ConnectorID   string `json:"connectorID,omitempty"`
	ConnectorData []byte `json:"connectorData,omitempty"`

	Expiry time.Time `json:"expiry"`
}

// AuthRequestList is a list of AuthRequests.
type AuthRequestList struct {
	k8sapi.TypeMeta `json:",inline"`
	k8sapi.ListMeta `json:"metadata,omitempty"`
	AuthRequests    []AuthRequest `json:"items"`
}

func toStorageAuthRequest(req AuthRequest) storage.AuthRequest {
	a := storage.AuthRequest{
		ID:                  req.ObjectMeta.Name,
		ClientID:            req.ClientID,
		ResponseTypes:       req.ResponseTypes,
		Scopes:              req.Scopes,
		RedirectURI:         req.RedirectURI,
		Nonce:               req.Nonce,
		State:               req.State,
		ForceApprovalPrompt: req.ForceApprovalPrompt,
		LoggedIn:            req.LoggedIn,
		ConnectorID:         req.ConnectorID,
		ConnectorData:       req.ConnectorData,
		Expiry:              req.Expiry,
		Claims:              toStorageClaims(req.Claims),
	}
	return a
}

func (cli *client) fromStorageAuthRequest(a storage.AuthRequest) AuthRequest {
	req := AuthRequest{
		TypeMeta: k8sapi.TypeMeta{
			Kind:       kindAuthRequest,
			APIVersion: cli.apiVersion,
		},
		ObjectMeta: k8sapi.ObjectMeta{
			Name:      a.ID,
			Namespace: cli.namespace,
		},
		ClientID:            a.ClientID,
		ResponseTypes:       a.ResponseTypes,
		Scopes:              a.Scopes,
		RedirectURI:         a.RedirectURI,
		Nonce:               a.Nonce,
		State:               a.State,
		LoggedIn:            a.LoggedIn,
		ForceApprovalPrompt: a.ForceApprovalPrompt,
		ConnectorID:         a.ConnectorID,
		ConnectorData:       a.ConnectorData,
		Expiry:              a.Expiry,
		Claims:              fromStorageClaims(a.Claims),
	}
	return req
}

// Password is a mirrored struct from the stroage with JSON struct tags and
// Kubernetes type metadata.
type Password struct {
	k8sapi.TypeMeta   `json:",inline"`
	k8sapi.ObjectMeta `json:"metadata,omitempty"`

	// The Kubernetes name is actually an encoded version of this value.
	//
	// This field is IMMUTABLE. Do not change.
	Email string `json:"email,omitempty"`

	Hash     []byte `json:"hash,omitempty"`
	Username string `json:"username,omitempty"`
	UserID   string `json:"userID,omitempty"`
}

// PasswordList is a list of Passwords.
type PasswordList struct {
	k8sapi.TypeMeta `json:",inline"`
	k8sapi.ListMeta `json:"metadata,omitempty"`
	Passwords       []Password `json:"items"`
}

func (cli *client) fromStoragePassword(p storage.Password) Password {
	email := strings.ToLower(p.Email)
	return Password{
		TypeMeta: k8sapi.TypeMeta{
			Kind:       kindPassword,
			APIVersion: cli.apiVersion,
		},
		ObjectMeta: k8sapi.ObjectMeta{
			Name:      cli.idToName(email),
			Namespace: cli.namespace,
		},
		Email:    email,
		Hash:     p.Hash,
		Username: p.Username,
		UserID:   p.UserID,
	}
}

func toStoragePassword(p Password) storage.Password {
	return storage.Password{
		Email:    p.Email,
		Hash:     p.Hash,
		Username: p.Username,
		UserID:   p.UserID,
	}
}

// AuthCode is a mirrored struct from storage with JSON struct tags and
// Kubernetes type metadata.
type AuthCode struct {
	k8sapi.TypeMeta   `json:",inline"`
	k8sapi.ObjectMeta `json:"metadata,omitempty"`

	ClientID    string   `json:"clientID"`
	Scopes      []string `json:"scopes,omitempty"`
	RedirectURI string   `json:"redirectURI"`

	Nonce string `json:"nonce,omitempty"`
	State string `json:"state,omitempty"`

	Claims Claims `json:"claims,omitempty"`

	ConnectorID   string `json:"connectorID,omitempty"`
	ConnectorData []byte `json:"connectorData,omitempty"`

	Expiry time.Time `json:"expiry"`
}

// AuthCodeList is a list of AuthCodes.
type AuthCodeList struct {
	k8sapi.TypeMeta `json:",inline"`
	k8sapi.ListMeta `json:"metadata,omitempty"`
	AuthCodes       []AuthCode `json:"items"`
}

func (cli *client) fromStorageAuthCode(a storage.AuthCode) AuthCode {
	return AuthCode{
		TypeMeta: k8sapi.TypeMeta{
			Kind:       kindAuthCode,
			APIVersion: cli.apiVersion,
		},
		ObjectMeta: k8sapi.ObjectMeta{
			Name:      a.ID,
			Namespace: cli.namespace,
		},
		ClientID:      a.ClientID,
		RedirectURI:   a.RedirectURI,
		ConnectorID:   a.ConnectorID,
		ConnectorData: a.ConnectorData,
		Nonce:         a.Nonce,
		Scopes:        a.Scopes,
		Claims:        fromStorageClaims(a.Claims),
		Expiry:        a.Expiry,
	}
}

func toStorageAuthCode(a AuthCode) storage.AuthCode {
	return storage.AuthCode{
		ID:            a.ObjectMeta.Name,
		ClientID:      a.ClientID,
		RedirectURI:   a.RedirectURI,
		ConnectorID:   a.ConnectorID,
		ConnectorData: a.ConnectorData,
		Nonce:         a.Nonce,
		Scopes:        a.Scopes,
		Claims:        toStorageClaims(a.Claims),
		Expiry:        a.Expiry,
	}
}

// RefreshToken is a mirrored struct from storage with JSON struct tags and
// Kubernetes type metadata.
type RefreshToken struct {
	k8sapi.TypeMeta   `json:",inline"`
	k8sapi.ObjectMeta `json:"metadata,omitempty"`

	CreatedAt time.Time
	LastUsed  time.Time

	ClientID string   `json:"clientID"`
	Scopes   []string `json:"scopes,omitempty"`

	Token string `json:"token,omitempty"`

	Nonce string `json:"nonce,omitempty"`

	Claims        Claims `json:"claims,omitempty"`
	ConnectorID   string `json:"connectorID,omitempty"`
	ConnectorData []byte `json:"connectorData,omitempty"`
}

// RefreshList is a list of refresh tokens.
type RefreshList struct {
	k8sapi.TypeMeta `json:",inline"`
	k8sapi.ListMeta `json:"metadata,omitempty"`
	RefreshTokens   []RefreshToken `json:"items"`
}

func toStorageRefreshToken(r RefreshToken) storage.RefreshToken {
	return storage.RefreshToken{
		ID:            r.ObjectMeta.Name,
		Token:         r.Token,
		CreatedAt:     r.CreatedAt,
		LastUsed:      r.LastUsed,
		ClientID:      r.ClientID,
		ConnectorID:   r.ConnectorID,
		ConnectorData: r.ConnectorData,
		Scopes:        r.Scopes,
		Nonce:         r.Nonce,
		Claims:        toStorageClaims(r.Claims),
	}
}

func (cli *client) fromStorageRefreshToken(r storage.RefreshToken) RefreshToken {
	return RefreshToken{
		TypeMeta: k8sapi.TypeMeta{
			Kind:       kindRefreshToken,
			APIVersion: cli.apiVersion,
		},
		ObjectMeta: k8sapi.ObjectMeta{
			Name:      r.ID,
			Namespace: cli.namespace,
		},
		Token:         r.Token,
		CreatedAt:     r.CreatedAt,
		LastUsed:      r.LastUsed,
		ClientID:      r.ClientID,
		ConnectorID:   r.ConnectorID,
		ConnectorData: r.ConnectorData,
		Scopes:        r.Scopes,
		Nonce:         r.Nonce,
		Claims:        fromStorageClaims(r.Claims),
	}
}

// Keys is a mirrored struct from storage with JSON struct tags and Kubernetes
// type metadata.
type Keys struct {
	k8sapi.TypeMeta   `json:",inline"`
	k8sapi.ObjectMeta `json:"metadata,omitempty"`

	// Key for creating and verifying signatures. These may be nil.
	SigningKey    *jose.JSONWebKey `json:"signingKey,omitempty"`
	SigningKeyPub *jose.JSONWebKey `json:"signingKeyPub,omitempty"`
	// Old signing keys which have been rotated but can still be used to validate
	// existing signatures.
	VerificationKeys []storage.VerificationKey `json:"verificationKeys,omitempty"`

	// The next time the signing key will rotate.
	//
	// For caching purposes, implementations MUST NOT update keys before this time.
	NextRotation time.Time `json:"nextRotation"`
}

func (cli *client) fromStorageKeys(keys storage.Keys) Keys {
	return Keys{
		TypeMeta: k8sapi.TypeMeta{
			Kind:       kindKeys,
			APIVersion: cli.apiVersion,
		},
		ObjectMeta: k8sapi.ObjectMeta{
			Name:      keysName,
			Namespace: cli.namespace,
		},
		SigningKey:       keys.SigningKey,
		SigningKeyPub:    keys.SigningKeyPub,
		VerificationKeys: keys.VerificationKeys,
		NextRotation:     keys.NextRotation,
	}
}

func toStorageKeys(keys Keys) storage.Keys {
	return storage.Keys{
		SigningKey:       keys.SigningKey,
		SigningKeyPub:    keys.SigningKeyPub,
		VerificationKeys: keys.VerificationKeys,
		NextRotation:     keys.NextRotation,
	}
}

// OfflineSessions is a mirrored struct from storage with JSON struct tags and Kubernetes
// type metadata.
type OfflineSessions struct {
	k8sapi.TypeMeta   `json:",inline"`
	k8sapi.ObjectMeta `json:"metadata,omitempty"`

	UserID  string                              `json:"userID,omitempty"`
	ConnID  string                              `json:"connID,omitempty"`
	Refresh map[string]*storage.RefreshTokenRef `json:"refresh,omitempty"`
}

func (cli *client) fromStorageOfflineSessions(o storage.OfflineSessions) OfflineSessions {
	return OfflineSessions{
		TypeMeta: k8sapi.TypeMeta{
			Kind:       kindOfflineSessions,
			APIVersion: cli.apiVersion,
		},
		ObjectMeta: k8sapi.ObjectMeta{
			Name:      cli.offlineTokenName(o.UserID, o.ConnID),
			Namespace: cli.namespace,
		},
		UserID:  o.UserID,
		ConnID:  o.ConnID,
		Refresh: o.Refresh,
	}
}

func toStorageOfflineSessions(o OfflineSessions) storage.OfflineSessions {
	s := storage.OfflineSessions{
		UserID:  o.UserID,
		ConnID:  o.ConnID,
		Refresh: o.Refresh,
	}
	if s.Refresh == nil {
		// Server code assumes this will be non-nil.
		s.Refresh = make(map[string]*storage.RefreshTokenRef)
	}
	return s
}

// Connector is a mirrored struct from storage with JSON struct tags and Kubernetes
// type metadata.
type Connector struct {
	k8sapi.TypeMeta   `json:",inline"`
	k8sapi.ObjectMeta `json:"metadata,omitempty"`

	ID              string `json:"id,omitempty"`
	Type            string `json:"type,omitempty"`
	Name            string `json:"name,omitempty"`
	ResourceVersion string `json:"resourceVersion,omitempty"`
	// Config holds connector specific configuration information
	Config []byte `json:"config,omitempty"`
}

func (cli *client) fromStorageConnector(c storage.Connector) Connector {
	return Connector{
		TypeMeta: k8sapi.TypeMeta{
			Kind:       kindConnector,
			APIVersion: cli.apiVersion,
		},
		ObjectMeta: k8sapi.ObjectMeta{
			Name:      c.ID,
			Namespace: cli.namespace,
		},
		ID:              c.ID,
		Type:            c.Type,
		Name:            c.Name,
		ResourceVersion: c.ResourceVersion,
		Config:          c.Config,
	}
}

func toStorageConnector(c Connector) storage.Connector {
	return storage.Connector{
		ID:              c.ID,
		Type:            c.Type,
		Name:            c.Name,
		ResourceVersion: c.ResourceVersion,
		Config:          c.Config,
	}
}

// ConnectorList is a list of Connectors.
type ConnectorList struct {
	k8sapi.TypeMeta `json:",inline"`
	k8sapi.ListMeta `json:"metadata,omitempty"`
	Connectors      []Connector `json:"items"`
}
