package kubernetes

import (
	"time"

	jose "gopkg.in/square/go-jose.v2"

	"github.com/coreos/poke/storage"
	"github.com/coreos/poke/storage/kubernetes/k8sapi"
)

// There will only ever be a single keys resource. Maintain this by setting a
// common name.
const keysName = "openid-connect-keys"

// Client is a mirrored struct from storage with JSON struct tags and
// Kubernetes type metadata.
//
// TODO(ericchiang): Kubernetes has an extremely restricted set of characters it can use for IDs.
// Consider base32ing client IDs.
type Client struct {
	k8sapi.TypeMeta   `json:",inline"`
	k8sapi.ObjectMeta `json:"metadata,omitempty"`

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
			APIVersion: cli.apiVersionForResource(resourceClient),
		},
		ObjectMeta: k8sapi.ObjectMeta{
			Name:      c.ID,
			Namespace: cli.namespace,
		},
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
		ID:           c.ObjectMeta.Name,
		Secret:       c.Secret,
		RedirectURIs: c.RedirectURIs,
		TrustedPeers: c.TrustedPeers,
		Public:       c.Public,
		Name:         c.Name,
		LogoURL:      c.LogoURL,
	}
}

// Identity is a mirrored struct from storage with JSON struct tags.
type Identity struct {
	UserID        string   `json:"userID"`
	Username      string   `json:"username"`
	Email         string   `json:"email"`
	EmailVerified bool     `json:"emailVerified"`
	Groups        []string `json:"groups,omitempty"`

	ConnectorData []byte `json:"connectorData,omitempty"`
}

func fromStorageIdentity(i storage.Identity) Identity {
	return Identity{
		UserID:        i.UserID,
		Username:      i.Username,
		Email:         i.Email,
		EmailVerified: i.EmailVerified,
		Groups:        i.Groups,
		ConnectorData: i.ConnectorData,
	}
}

func toStorageIdentity(i Identity) storage.Identity {
	return storage.Identity{
		UserID:        i.UserID,
		Username:      i.Username,
		Email:         i.Email,
		EmailVerified: i.EmailVerified,
		Groups:        i.Groups,
		ConnectorData: i.ConnectorData,
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

	// The identity of the end user. Generally nil until the user authenticates
	// with a backend.
	Identity *Identity `json:"identity,omitempty"`
	// The connector used to login the user. Set when the user authenticates.
	ConnectorID string `json:"connectorID,omitempty"`

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
		ConnectorID:         req.ConnectorID,
		Expiry:              req.Expiry,
	}
	if req.Identity != nil {
		i := toStorageIdentity(*req.Identity)
		a.Identity = &i
	}
	return a
}

func (cli *client) fromStorageAuthRequest(a storage.AuthRequest) AuthRequest {
	req := AuthRequest{
		TypeMeta: k8sapi.TypeMeta{
			Kind:       kindAuthRequest,
			APIVersion: cli.apiVersionForResource(resourceAuthRequest),
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
		ForceApprovalPrompt: a.ForceApprovalPrompt,
		ConnectorID:         a.ConnectorID,
		Expiry:              a.Expiry,
	}
	if a.Identity != nil {
		i := fromStorageIdentity(*a.Identity)
		req.Identity = &i
	}
	return req
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

	Identity    Identity `json:"identity,omitempty"`
	ConnectorID string   `json:"connectorID,omitempty"`

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
			APIVersion: cli.apiVersionForResource(resourceAuthCode),
		},
		ObjectMeta: k8sapi.ObjectMeta{
			Name:      a.ID,
			Namespace: cli.namespace,
		},
		ClientID:    a.ClientID,
		RedirectURI: a.RedirectURI,
		ConnectorID: a.ConnectorID,
		Nonce:       a.Nonce,
		Scopes:      a.Scopes,
		Identity:    fromStorageIdentity(a.Identity),
		Expiry:      a.Expiry,
	}
}

func toStorageAuthCode(a AuthCode) storage.AuthCode {
	return storage.AuthCode{
		ID:          a.ObjectMeta.Name,
		ClientID:    a.ClientID,
		RedirectURI: a.RedirectURI,
		ConnectorID: a.ConnectorID,
		Nonce:       a.Nonce,
		Scopes:      a.Scopes,
		Identity:    toStorageIdentity(a.Identity),
		Expiry:      a.Expiry,
	}
}

// Refresh is a mirrored struct from storage with JSON struct tags and
// Kubernetes type metadata.
type Refresh struct {
	k8sapi.TypeMeta   `json:",inline"`
	k8sapi.ObjectMeta `json:"metadata,omitempty"`

	ClientID string   `json:"clientID"`
	Scopes   []string `json:"scopes,omitempty"`

	Nonce string `json:"nonce,omitempty"`

	Identity    Identity `json:"identity,omitempty"`
	ConnectorID string   `json:"connectorID,omitempty"`
}

// RefreshList is a list of refresh tokens.
type RefreshList struct {
	k8sapi.TypeMeta `json:",inline"`
	k8sapi.ListMeta `json:"metadata,omitempty"`
	RefreshTokens   []Refresh `json:"items"`
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
			APIVersion: cli.apiVersionForResource(resourceKeys),
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
