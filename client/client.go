package client

import (
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"reflect"

	"github.com/coreos/dex/repo"
	"github.com/coreos/go-oidc/oidc"
)

var (
	ErrorInvalidRedirectURL    = errors.New("not a valid redirect url for the given client")
	ErrorCantChooseRedirectURL = errors.New("must provide a redirect url; client has many")
	ErrorNoValidRedirectURLs   = errors.New("no valid redirect URLs for this client.")
	ErrorNotFound              = errors.New("no data found")
)

type Client struct {
	Credentials oidc.ClientCredentials
	Metadata    oidc.ClientMetadata
	Admin       bool
}

type ClientRepo interface {
	Get(tx repo.Transaction, clientID string) (Client, error)

	// Metadata returns one matching ClientMetadata if the given client
	// exists, otherwise nil. The returned error will be non-nil only
	// if the repo was unable to determine client existence.
	Metadata(tx repo.Transaction, clientID string) (*oidc.ClientMetadata, error)

	// Authenticate asserts that a client with the given ID exists and
	// that the provided secret matches. If either of these assertions
	// fail, (false, nil) will be returned. Only if the repo is unable
	// to make these assertions will a non-nil error be returned.
	Authenticate(tx repo.Transaction, creds oidc.ClientCredentials) (bool, error)

	// All returns all registered Clients
	All(tx repo.Transaction) ([]Client, error)

	// New registers a Client with the repo.
	// An unused ID must be provided. A corresponding secret will be returned
	// in a ClientCredentials struct along with the provided ID.
	New(tx repo.Transaction, client Client) (*oidc.ClientCredentials, error)

	SetDexAdmin(clientID string, isAdmin bool) error

	IsDexAdmin(clientID string) (bool, error)
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

func ClientsFromReader(r io.Reader) ([]Client, error) {
	var c []struct {
		ID           string   `json:"id"`
		Secret       string   `json:"secret"`
		RedirectURLs []string `json:"redirectURLs"`
	}
	if err := json.NewDecoder(r).Decode(&c); err != nil {
		return nil, err
	}
	clients := make([]Client, len(c))
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

		clients[i] = Client{
			Credentials: oidc.ClientCredentials{
				ID:     client.ID,
				Secret: client.Secret,
			},
			Metadata: oidc.ClientMetadata{
				RedirectURIs: redirectURIs,
			},
		}
	}
	return clients, nil
}
