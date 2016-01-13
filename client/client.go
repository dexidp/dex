package client

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/url"
	"reflect"
	"sort"

	pcrypto "github.com/coreos/dex/pkg/crypto"
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

func NewClientIdentityRepo(cs []oidc.ClientIdentity) ClientIdentityRepo {
	cr := memClientIdentityRepo{
		idents: make(map[string]oidc.ClientIdentity, len(cs)),
		admins: make(map[string]bool),
	}

	for _, c := range cs {
		c := c
		cr.idents[c.Credentials.ID] = c
	}

	return &cr
}

type memClientIdentityRepo struct {
	idents map[string]oidc.ClientIdentity
	admins map[string]bool
}

func (cr *memClientIdentityRepo) New(id string, meta oidc.ClientMetadata) (*oidc.ClientCredentials, error) {
	if _, ok := cr.idents[id]; ok {
		return nil, errors.New("client ID already exists")
	}

	secret, err := pcrypto.RandBytes(32)
	if err != nil {
		return nil, err
	}

	cc := oidc.ClientCredentials{
		ID:     id,
		Secret: base64.URLEncoding.EncodeToString(secret),
	}

	cr.idents[id] = oidc.ClientIdentity{
		Metadata:    meta,
		Credentials: cc,
	}

	return &cc, nil
}

func (cr *memClientIdentityRepo) Metadata(clientID string) (*oidc.ClientMetadata, error) {
	ci, ok := cr.idents[clientID]
	if !ok {
		return nil, ErrorNotFound
	}
	return &ci.Metadata, nil
}

func (cr *memClientIdentityRepo) Authenticate(creds oidc.ClientCredentials) (bool, error) {
	ci, ok := cr.idents[creds.ID]
	ok = ok && ci.Credentials.Secret == creds.Secret
	return ok, nil
}

func (cr *memClientIdentityRepo) All() ([]oidc.ClientIdentity, error) {
	cs := make(sortableClientIdentities, 0, len(cr.idents))
	for _, ci := range cr.idents {
		ci := ci
		cs = append(cs, ci)
	}
	sort.Sort(cs)
	return cs, nil
}

func (cr *memClientIdentityRepo) SetDexAdmin(clientID string, isAdmin bool) error {
	cr.admins[clientID] = isAdmin
	return nil
}

func (cr *memClientIdentityRepo) IsDexAdmin(clientID string) (bool, error) {
	return cr.admins[clientID], nil
}

type sortableClientIdentities []oidc.ClientIdentity

func (s sortableClientIdentities) Len() int {
	return len([]oidc.ClientIdentity(s))
}

func (s sortableClientIdentities) Less(i, j int) bool {
	return s[i].Credentials.ID < s[j].Credentials.ID
}

func (s sortableClientIdentities) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func NewClientIdentityRepoFromReader(r io.Reader) (ClientIdentityRepo, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var cs []clientIdentity
	if err = json.Unmarshal(b, &cs); err != nil {
		return nil, err
	}

	ocs := make([]oidc.ClientIdentity, len(cs))
	for i, c := range cs {
		ocs[i] = oidc.ClientIdentity(c)
	}

	return NewClientIdentityRepo(ocs), nil
}

type clientIdentity oidc.ClientIdentity

func (ci *clientIdentity) UnmarshalJSON(data []byte) error {
	c := struct {
		ID           string   `json:"id"`
		Secret       string   `json:"secret"`
		RedirectURLs []string `json:"redirectURLs"`
	}{}

	if err := json.Unmarshal(data, &c); err != nil {
		return err
	}

	ci.Credentials = oidc.ClientCredentials{
		ID:     c.ID,
		Secret: c.Secret,
	}
	ci.Metadata = oidc.ClientMetadata{
		RedirectURIs: make([]url.URL, len(c.RedirectURLs)),
	}

	for i, us := range c.RedirectURLs {
		up, err := url.Parse(us)
		if err != nil {
			return err
		}
		ci.Metadata.RedirectURIs[i] = *up
	}

	return nil
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
