package client

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"reflect"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/coreos/dex/repo"
	"github.com/coreos/go-oidc/oidc"
)

var (
	ErrorInvalidClientID = errors.New("not a valid client ID")

	ErrorInvalidClientSecret = errors.New("not a valid client Secret")

	ErrorInvalidRedirectURL    = errors.New("not a valid redirect url for the given client")
	ErrorCantChooseRedirectURL = errors.New("must provide a redirect url; client has many")
	ErrorNoValidRedirectURLs   = errors.New("no valid redirect URLs for this client.")

	ErrorPublicClientRedirectURIs = errors.New("public clients cannot have redirect URIs")
	ErrorPublicClientMissingName  = errors.New("public clients must have a name")

	ErrorMissingRedirectURI = errors.New("no client redirect url given")

	ErrorNotFound = errors.New("no data found")
)

type ValidationError struct {
	Err error
}

func (v ValidationError) Error() string {
	return v.Err.Error()
}

const (
	bcryptHashCost = 10

	OOBRedirectURI = "urn:ietf:wg:oauth:2.0:oob"
)

func HashSecret(creds oidc.ClientCredentials) ([]byte, error) {
	secretBytes, err := base64.URLEncoding.DecodeString(creds.Secret)
	if err != nil {
		return nil, ErrorInvalidClientSecret
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(
		secretBytes),
		bcryptHashCost)
	if err != nil {
		return nil, err
	}
	return hashed, nil
}

type Client struct {
	Credentials oidc.ClientCredentials
	Metadata    oidc.ClientMetadata
	Admin       bool
	Public      bool
}

func (c Client) ValidRedirectURL(u *url.URL) (url.URL, error) {
	if c.Public {
		if u == nil {
			return url.URL{}, ErrorInvalidRedirectURL
		}
		if u.String() == OOBRedirectURI {
			return *u, nil
		}

		if u.Scheme != "http" {
			return url.URL{}, ErrorInvalidRedirectURL
		}

		hostPort := strings.Split(u.Host, ":")
		if len(hostPort) != 2 {
			return url.URL{}, ErrorInvalidRedirectURL
		}

		if hostPort[0] != "localhost" || u.Path != "" || u.RawPath != "" || u.RawQuery != "" || u.Fragment != "" {
			return url.URL{}, ErrorInvalidRedirectURL
		}

		return *u, nil
	}

	return ValidRedirectURL(u, c.Metadata.RedirectURIs)
}

type ClientRepo interface {
	Get(tx repo.Transaction, clientID string) (Client, error)

	// GetSecret returns the (base64 encoded) hashed client secret
	GetSecret(tx repo.Transaction, clientID string) ([]byte, error)

	// All returns all registered Clients
	All(tx repo.Transaction) ([]Client, error)

	// New registers a Client with the repo.
	// An unused ID must be provided. A corresponding secret will be returned
	// in a ClientCredentials struct along with the provided ID.
	New(tx repo.Transaction, client Client) (*oidc.ClientCredentials, error)

	Update(tx repo.Transaction, client Client) error

	// GetTrustedPeers returns the list of clients authorized to mint ID token for the given client.
	GetTrustedPeers(tx repo.Transaction, clientID string) ([]string, error)

	// SetTrustedPeers sets the list of clients authorized to mint ID token for the given client.
	SetTrustedPeers(tx repo.Transaction, clientID string, clientIDs []string) error
}

// ValidRedirectURL returns the passed in URL if it is present in the redirectURLs list, and returns an error otherwise.
// If nil is passed in as the rURL and there is only one URL in redirectURLs,
// that URL will be returned. If nil is passed but theres >1 URL in the slice,
// then an error is returned.
func ValidRedirectURL(rURL *url.URL, redirectURLs []url.URL) (url.URL, error) {
	if len(redirectURLs) == 0 {
		return url.URL{}, ErrorNoValidRedirectURLs
	}

	if rURL == nil {
		if len(redirectURLs) > 1 {
			return url.URL{}, ErrorCantChooseRedirectURL
		}

		return redirectURLs[0], nil
	}

	for _, ru := range redirectURLs {
		if reflect.DeepEqual(ru, *rURL) {
			return ru, nil
		}
	}
	return url.URL{}, ErrorInvalidRedirectURL
}

// LoadableClient contains sufficient information for creating a Client and its related entities.
type LoadableClient struct {
	Client       Client
	TrustedPeers []string
}

func ClientsFromReader(r io.Reader) ([]LoadableClient, error) {
	var c []struct {
		ID           string   `json:"id"`
		Secret       string   `json:"secret"`
		RedirectURLs []string `json:"redirectURLs"`
		Admin        bool     `json:"admin"`
		Public       bool     `json:"public"`
		TrustedPeers []string `json:"trustedPeers"`
	}
	if err := json.NewDecoder(r).Decode(&c); err != nil {
		return nil, err
	}
	clients := make([]LoadableClient, len(c))
	for i, client := range c {
		if client.ID == "" {
			return nil, errors.New("clients must have an ID")
		}
		if len(client.Secret) == 0 {
			return nil, errors.New("clients must have a Secret")
		}
		redirectURIs := make([]url.URL, len(client.RedirectURLs))
		for j, u := range client.RedirectURLs {
			uri, err := url.Parse(u)
			if err != nil {
				return nil, err
			}
			redirectURIs[j] = *uri
		}

		clients[i] = LoadableClient{
			Client: Client{
				Credentials: oidc.ClientCredentials{
					ID:     client.ID,
					Secret: client.Secret,
				},
				Metadata: oidc.ClientMetadata{
					RedirectURIs: redirectURIs,
				},
				Admin:  client.Admin,
				Public: client.Public,
			},
			TrustedPeers: client.TrustedPeers,
		}
	}
	return clients, nil
}
