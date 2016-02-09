package client

import (
	"errors"
	"net/url"
	"reflect"

	"github.com/coreos/go-oidc/oidc"
)

var (
	ErrorInvalidRedirectURL    = errors.New("not a valid redirect url for the given client")
	ErrorCantChooseRedirectURL = errors.New("must provide a redirect url; client has many")
	ErrorNoValidRedirectURLs   = errors.New("no valid redirect URLs for this client.")
	ErrorNotFound              = errors.New("no data found")
)

type ClientIdentityRepo interface {
	// Metadata returns one matching ClientMetadata if the given client
	// exists, otherwise nil. The returned error will be non-nil only
	// if the repo was unable to determine client existence.
	Metadata(clientID string) (*oidc.ClientMetadata, error)

	// Authenticate asserts that a client with the given ID exists and
	// that the provided secret matches. If either of these assertions
	// fail, (false, nil) will be returned. Only if the repo is unable
	// to make these assertions will a non-nil error be returned.
	Authenticate(creds oidc.ClientCredentials) (bool, error)

	// All returns all registered Client Identities.
	All() ([]oidc.ClientIdentity, error)

	// New registers a ClientIdentity with the repo for the given metadata.
	// An unused ID must be provided. A corresponding secret will be returned
	// in a ClientCredentials struct along with the provided ID.
	New(id string, meta oidc.ClientMetadata) (*oidc.ClientCredentials, error)

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
