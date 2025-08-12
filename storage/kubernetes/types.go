package kubernetes

import (
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/kubernetes/k8sapi"
)

const (
	apiGroup = "dex.coreos.com"

	legacyCRDAPIVersion = "apiextensions.k8s.io/v1beta1"
	crdAPIVersion       = "apiextensions.k8s.io/v1"
)

// The set of custom resource definitions required by the storage. These are managed by
// the storage so it can migrate itself by creating new resources.
func customResourceDefinitions(apiVersion string) []k8sapi.CustomResourceDefinition {
	crdMeta := k8sapi.TypeMeta{
		APIVersion: apiVersion,
		Kind:       "CustomResourceDefinition",
	}

	var version string
	var scope k8sapi.ResourceScope
	var versions []k8sapi.CustomResourceDefinitionVersion

	switch apiVersion {
	case crdAPIVersion:
		preserveUnknownFields := true
		versions = []k8sapi.CustomResourceDefinitionVersion{
			{
				Name:    "v1",
				Served:  true,
				Storage: true,
				Schema: &k8sapi.CustomResourceValidation{
					OpenAPIV3Schema: &k8sapi.JSONSchemaProps{
						Type:                   "object",
						XPreserveUnknownFields: &preserveUnknownFields,
					},
				},
			},
		}
		scope = k8sapi.NamespaceScoped
	case legacyCRDAPIVersion:
		version = "v1"
	default:
		panic("unknown apiVersion " + apiVersion)
	}

	return []k8sapi.CustomResourceDefinition{
		{
			ObjectMeta: k8sapi.ObjectMeta{
				Name: "authcodes.dex.coreos.com",
			},
			TypeMeta: crdMeta,
			Spec: k8sapi.CustomResourceDefinitionSpec{
				Group:    apiGroup,
				Version:  version,
				Versions: versions,
				Scope:    scope,
				Names: k8sapi.CustomResourceDefinitionNames{
					Plural:   "authcodes",
					Singular: "authcode",
					Kind:     "AuthCode",
				},
			},
		},
		{
			ObjectMeta: k8sapi.ObjectMeta{
				Name: "authrequests.dex.coreos.com",
			},
			TypeMeta: crdMeta,
			Spec: k8sapi.CustomResourceDefinitionSpec{
				Group:    apiGroup,
				Version:  version,
				Versions: versions,
				Scope:    scope,
				Names: k8sapi.CustomResourceDefinitionNames{
					Plural:   "authrequests",
					Singular: "authrequest",
					Kind:     "AuthRequest",
				},
			},
		},
		{
			ObjectMeta: k8sapi.ObjectMeta{
				Name: "oauth2clients.dex.coreos.com",
			},
			TypeMeta: crdMeta,
			Spec: k8sapi.CustomResourceDefinitionSpec{
				Group:    apiGroup,
				Version:  version,
				Versions: versions,
				Scope:    scope,
				Names: k8sapi.CustomResourceDefinitionNames{
					Plural:   "oauth2clients",
					Singular: "oauth2client",
					Kind:     "OAuth2Client",
				},
			},
		},
		{
			ObjectMeta: k8sapi.ObjectMeta{
				Name: "signingkeies.dex.coreos.com",
			},
			TypeMeta: crdMeta,
			Spec: k8sapi.CustomResourceDefinitionSpec{
				Group:    apiGroup,
				Version:  version,
				Versions: versions,
				Scope:    scope,
				Names: k8sapi.CustomResourceDefinitionNames{
					// `signingkeies` is an artifact from the old TPR pluralization.
					// Users don't directly interact with this value, hence leaving it
					// as is.
					Plural:   "signingkeies",
					Singular: "signingkey",
					Kind:     "SigningKey",
				},
			},
		},
		{
			ObjectMeta: k8sapi.ObjectMeta{
				Name: "refreshtokens.dex.coreos.com",
			},
			TypeMeta: crdMeta,
			Spec: k8sapi.CustomResourceDefinitionSpec{
				Group:    apiGroup,
				Version:  version,
				Versions: versions,
				Scope:    scope,
				Names: k8sapi.CustomResourceDefinitionNames{
					Plural:   "refreshtokens",
					Singular: "refreshtoken",
					Kind:     "RefreshToken",
				},
			},
		},
		{
			ObjectMeta: k8sapi.ObjectMeta{
				Name: "passwords.dex.coreos.com",
			},
			TypeMeta: crdMeta,
			Spec: k8sapi.CustomResourceDefinitionSpec{
				Group:    apiGroup,
				Version:  version,
				Versions: versions,
				Scope:    scope,
				Names: k8sapi.CustomResourceDefinitionNames{
					Plural:   "passwords",
					Singular: "password",
					Kind:     "Password",
				},
			},
		},
		{
			ObjectMeta: k8sapi.ObjectMeta{
				Name: "offlinesessionses.dex.coreos.com",
			},
			TypeMeta: crdMeta,
			Spec: k8sapi.CustomResourceDefinitionSpec{
				Group:    apiGroup,
				Version:  version,
				Versions: versions,
				Scope:    scope,
				Names: k8sapi.CustomResourceDefinitionNames{
					Plural:   "offlinesessionses",
					Singular: "offlinesessions",
					Kind:     "OfflineSessions",
				},
			},
		},
		{
			ObjectMeta: k8sapi.ObjectMeta{
				Name: "connectors.dex.coreos.com",
			},
			TypeMeta: crdMeta,
			Spec: k8sapi.CustomResourceDefinitionSpec{
				Group:    apiGroup,
				Version:  version,
				Versions: versions,
				Scope:    scope,
				Names: k8sapi.CustomResourceDefinitionNames{
					Plural:   "connectors",
					Singular: "connector",
					Kind:     "Connector",
				},
			},
		},
		{
			ObjectMeta: k8sapi.ObjectMeta{
				Name: "devicerequests.dex.coreos.com",
			},
			TypeMeta: crdMeta,
			Spec: k8sapi.CustomResourceDefinitionSpec{
				Group:    apiGroup,
				Version:  version,
				Versions: versions,
				Scope:    scope,
				Names: k8sapi.CustomResourceDefinitionNames{
					Plural:   "devicerequests",
					Singular: "devicerequest",
					Kind:     "DeviceRequest",
				},
			},
		},
		{
			ObjectMeta: k8sapi.ObjectMeta{
				Name: "devicetokens.dex.coreos.com",
			},
			TypeMeta: crdMeta,
			Spec: k8sapi.CustomResourceDefinitionSpec{
				Group:    apiGroup,
				Version:  version,
				Versions: versions,
				Scope:    scope,
				Names: k8sapi.CustomResourceDefinitionNames{
					Plural:   "devicetokens",
					Singular: "devicetoken",
					Kind:     "DeviceToken",
				},
			},
		},
	}
}

// There will only ever be a single keys resource. Maintain this by setting a
// common name.
const keysName = "openid-connect-keys"

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
	UserID            string   `json:"userID"`
	Username          string   `json:"username"`
	PreferredUsername string   `json:"preferredUsername"`
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"emailVerified"`
	Groups            []string `json:"groups,omitempty"`
}

func fromStorageClaims(i storage.Claims) Claims {
	return Claims{
		UserID:            i.UserID,
		Username:          i.Username,
		PreferredUsername: i.PreferredUsername,
		Email:             i.Email,
		EmailVerified:     i.EmailVerified,
		Groups:            i.Groups,
	}
}

func toStorageClaims(i Claims) storage.Claims {
	return storage.Claims{
		UserID:            i.UserID,
		Username:          i.Username,
		PreferredUsername: i.PreferredUsername,
		Email:             i.Email,
		EmailVerified:     i.EmailVerified,
		Groups:            i.Groups,
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

	CodeChallenge       string `json:"code_challenge,omitempty"`
	CodeChallengeMethod string `json:"code_challenge_method,omitempty"`

	HMACKey []byte `json:"hmac_key"`
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
		PKCE: storage.PKCE{
			CodeChallenge:       req.CodeChallenge,
			CodeChallengeMethod: req.CodeChallengeMethod,
		},
		HMACKey: req.HMACKey,
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
		CodeChallenge:       a.PKCE.CodeChallenge,
		CodeChallengeMethod: a.PKCE.CodeChallengeMethod,
		HMACKey:             a.HMACKey,
	}
	return req
}

// Password is a mirrored struct from the storage with JSON struct tags and
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

	CodeChallenge       string `json:"code_challenge,omitempty"`
	CodeChallengeMethod string `json:"code_challenge_method,omitempty"`
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
		ClientID:            a.ClientID,
		RedirectURI:         a.RedirectURI,
		ConnectorID:         a.ConnectorID,
		ConnectorData:       a.ConnectorData,
		Nonce:               a.Nonce,
		Scopes:              a.Scopes,
		Claims:              fromStorageClaims(a.Claims),
		Expiry:              a.Expiry,
		CodeChallenge:       a.PKCE.CodeChallenge,
		CodeChallengeMethod: a.PKCE.CodeChallengeMethod,
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
		PKCE: storage.PKCE{
			CodeChallenge:       a.CodeChallenge,
			CodeChallengeMethod: a.CodeChallengeMethod,
		},
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

	Token         string `json:"token,omitempty"`
	ObsoleteToken string `json:"obsoleteToken,omitempty"`

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
		ObsoleteToken: r.ObsoleteToken,
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
		ObsoleteToken: r.ObsoleteToken,
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

	UserID        string                              `json:"userID,omitempty"`
	ConnID        string                              `json:"connID,omitempty"`
	Refresh       map[string]*storage.RefreshTokenRef `json:"refresh,omitempty"`
	ConnectorData []byte                              `json:"connectorData,omitempty"`
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
		UserID:        o.UserID,
		ConnID:        o.ConnID,
		Refresh:       o.Refresh,
		ConnectorData: o.ConnectorData,
	}
}

func toStorageOfflineSessions(o OfflineSessions) storage.OfflineSessions {
	s := storage.OfflineSessions{
		UserID:        o.UserID,
		ConnID:        o.ConnID,
		Refresh:       o.Refresh,
		ConnectorData: o.ConnectorData,
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

	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
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
		ID:     c.ID,
		Type:   c.Type,
		Name:   c.Name,
		Config: c.Config,
	}
}

func toStorageConnector(c Connector) storage.Connector {
	return storage.Connector{
		ID:              c.ID,
		Type:            c.Type,
		Name:            c.Name,
		ResourceVersion: c.ObjectMeta.ResourceVersion,
		Config:          c.Config,
	}
}

// ConnectorList is a list of Connectors.
type ConnectorList struct {
	k8sapi.TypeMeta `json:",inline"`
	k8sapi.ListMeta `json:"metadata,omitempty"`
	Connectors      []Connector `json:"items"`
}

// DeviceRequest is a mirrored struct from storage with JSON struct tags and
// Kubernetes type metadata.
type DeviceRequest struct {
	k8sapi.TypeMeta   `json:",inline"`
	k8sapi.ObjectMeta `json:"metadata,omitempty"`

	DeviceCode   string    `json:"device_code,omitempty"`
	ClientID     string    `json:"client_id,omitempty"`
	ClientSecret string    `json:"client_secret,omitempty"`
	Scopes       []string  `json:"scopes,omitempty"`
	Expiry       time.Time `json:"expiry"`
}

// DeviceRequestList is a list of DeviceRequests.
type DeviceRequestList struct {
	k8sapi.TypeMeta `json:",inline"`
	k8sapi.ListMeta `json:"metadata,omitempty"`
	DeviceRequests  []DeviceRequest `json:"items"`
}

func (cli *client) fromStorageDeviceRequest(a storage.DeviceRequest) DeviceRequest {
	req := DeviceRequest{
		TypeMeta: k8sapi.TypeMeta{
			Kind:       kindDeviceRequest,
			APIVersion: cli.apiVersion,
		},
		ObjectMeta: k8sapi.ObjectMeta{
			Name:      strings.ToLower(a.UserCode),
			Namespace: cli.namespace,
		},
		DeviceCode:   a.DeviceCode,
		ClientID:     a.ClientID,
		ClientSecret: a.ClientSecret,
		Scopes:       a.Scopes,
		Expiry:       a.Expiry,
	}
	return req
}

func toStorageDeviceRequest(req DeviceRequest) storage.DeviceRequest {
	return storage.DeviceRequest{
		UserCode:     strings.ToUpper(req.ObjectMeta.Name),
		DeviceCode:   req.DeviceCode,
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		Scopes:       req.Scopes,
		Expiry:       req.Expiry,
	}
}

// DeviceToken is a mirrored struct from storage with JSON struct tags and
// Kubernetes type metadata.
type DeviceToken struct {
	k8sapi.TypeMeta   `json:",inline"`
	k8sapi.ObjectMeta `json:"metadata,omitempty"`

	Status              string    `json:"status,omitempty"`
	Token               string    `json:"token,omitempty"`
	Expiry              time.Time `json:"expiry"`
	LastRequestTime     time.Time `json:"last_request"`
	PollIntervalSeconds int       `json:"poll_interval"`
	CodeChallenge       string    `json:"code_challenge,omitempty"`
	CodeChallengeMethod string    `json:"code_challenge_method,omitempty"`
}

// DeviceTokenList is a list of DeviceTokens.
type DeviceTokenList struct {
	k8sapi.TypeMeta `json:",inline"`
	k8sapi.ListMeta `json:"metadata,omitempty"`
	DeviceTokens    []DeviceToken `json:"items"`
}

func (cli *client) fromStorageDeviceToken(t storage.DeviceToken) DeviceToken {
	req := DeviceToken{
		TypeMeta: k8sapi.TypeMeta{
			Kind:       kindDeviceToken,
			APIVersion: cli.apiVersion,
		},
		ObjectMeta: k8sapi.ObjectMeta{
			Name:      t.DeviceCode,
			Namespace: cli.namespace,
		},
		Status:              t.Status,
		Token:               t.Token,
		Expiry:              t.Expiry,
		LastRequestTime:     t.LastRequestTime,
		PollIntervalSeconds: t.PollIntervalSeconds,
		CodeChallenge:       t.PKCE.CodeChallenge,
		CodeChallengeMethod: t.PKCE.CodeChallengeMethod,
	}
	return req
}

func toStorageDeviceToken(t DeviceToken) storage.DeviceToken {
	return storage.DeviceToken{
		DeviceCode:          t.ObjectMeta.Name,
		Status:              t.Status,
		Token:               t.Token,
		Expiry:              t.Expiry,
		LastRequestTime:     t.LastRequestTime,
		PollIntervalSeconds: t.PollIntervalSeconds,
		PKCE: storage.PKCE{
			CodeChallenge:       t.CodeChallenge,
			CodeChallengeMethod: t.CodeChallengeMethod,
		},
	}
}
