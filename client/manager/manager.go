package manager

import (
	"encoding/base64"
	"net/url"

	"errors"

	"github.com/coreos/dex/client"
	pcrypto "github.com/coreos/dex/pkg/crypto"
	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/repo"
	"github.com/coreos/go-oidc/oidc"
	"golang.org/x/crypto/bcrypt"
)

const (
	// Blowfish, the algorithm underlying bcrypt, has a maximum
	// password length of 72. We explicitly track and check this
	// since the bcrypt library will silently ignore portions of
	// a password past the first 72 characters.
	maxSecretLength = 72
)

var (
	localHostRedirectURL = mustParseURL("http://localhost:0")
)

type ClientOptions struct {
	TrustedPeers []string
}

type SecretGenerator func() ([]byte, error)

func DefaultSecretGenerator() ([]byte, error) {
	return pcrypto.RandBytes(maxSecretLength)
}

func CompareHashAndPassword(hashedPassword, password []byte) error {
	if len(password) > maxSecretLength {
		return errors.New("password length greater than max secret length")
	}
	return bcrypt.CompareHashAndPassword(hashedPassword, password)
}

// ClientManager performs client-related "business-logic" functions on client and related objects.
// This is in contrast to the Repos which perform little more than CRUD operations.
type ClientManager struct {
	clientRepo        client.ClientRepo
	begin             repo.TransactionFactory
	secretGenerator   SecretGenerator
	clientIDGenerator func(string) (string, error)
}

type ManagerOptions struct {
	SecretGenerator   func() ([]byte, error)
	ClientIDGenerator func(string) (string, error)
}

func NewClientManager(clientRepo client.ClientRepo, txnFactory repo.TransactionFactory, options ManagerOptions) *ClientManager {
	if options.SecretGenerator == nil {
		options.SecretGenerator = DefaultSecretGenerator
	}
	if options.ClientIDGenerator == nil {
		options.ClientIDGenerator = oidc.GenClientID
	}
	return &ClientManager{
		clientRepo:        clientRepo,
		begin:             txnFactory,
		secretGenerator:   options.SecretGenerator,
		clientIDGenerator: options.ClientIDGenerator,
	}
}

// New creates and persists a new client with the given options, returning the generated credentials.
// Any Credenials provided with the client are ignored and overwritten by the generated  ID and Secret.
// "Normal" (i.e. non-Public) clients must have at least one valid RedirectURI in their Metadata.
// Public clients must not have any RedirectURIs and must have a client name.
func (m *ClientManager) New(cli client.Client, options *ClientOptions) (*oidc.ClientCredentials, error) {
	tx, err := m.begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := validateClient(cli); err != nil {
		return nil, err
	}

	err = m.addClientCredentials(&cli)
	if err != nil {
		return nil, err
	}
	creds := cli.Credentials

	// Save Client
	_, err = m.clientRepo.New(tx, cli)
	if err != nil {
		return nil, err
	}

	if options != nil && len(options.TrustedPeers) > 0 {
		err = m.clientRepo.SetTrustedPeers(tx, creds.ID, options.TrustedPeers)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	// Returns creds with unhashed secret
	return &creds, nil
}

func (m *ClientManager) Get(id string) (client.Client, error) {
	return m.clientRepo.Get(nil, id)
}

func (m *ClientManager) All() ([]client.Client, error) {
	return m.clientRepo.All(nil)
}

func (m *ClientManager) Metadata(clientID string) (*oidc.ClientMetadata, error) {
	c, err := m.clientRepo.Get(nil, clientID)
	if err != nil {
		return nil, err
	}

	return &c.Metadata, nil
}

func (m *ClientManager) IsDexAdmin(clientID string) (bool, error) {
	c, err := m.clientRepo.Get(nil, clientID)
	if err != nil {
		return false, err
	}

	return c.Admin, nil
}

func (m *ClientManager) SetDexAdmin(clientID string, isAdmin bool) error {
	tx, err := m.begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	c, err := m.clientRepo.Get(tx, clientID)
	if err != nil {
		return err
	}

	c.Admin = isAdmin
	err = m.clientRepo.Update(tx, c)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (m *ClientManager) Authenticate(creds oidc.ClientCredentials) (bool, error) {
	clientSecret, err := m.clientRepo.GetSecret(nil, creds.ID)
	if err != nil {
		log.Errorf("error getting secret for client ID: %v: err: %v", creds.ID, err)
		return false, nil
	}

	if clientSecret == nil {
		log.Errorf("no secret found for client ID: %v", creds.ID)
		return false, nil
	}

	dec, err := base64.URLEncoding.DecodeString(creds.Secret)
	if err != nil {
		log.Errorf("error Decoding client creds: %v", err)
		return false, nil
	}

	ok := CompareHashAndPassword(clientSecret, dec) == nil
	return ok, nil
}

func (m *ClientManager) addClientCredentials(cli *client.Client) error {
	var seed string
	if cli.Public {
		seed = cli.Metadata.ClientName
	} else {
		seed = cli.Metadata.RedirectURIs[0].Host
	}

	var err error
	var clientID string
	if cli.Credentials.ID != "" {
		clientID = cli.Credentials.ID
	} else {
		// Generate Client ID
		clientID, err = m.clientIDGenerator(seed)
		if err != nil {
			return err
		}
	}

	var clientSecret string
	if cli.Credentials.Secret != "" {
		clientSecret = cli.Credentials.Secret
	} else {
		// Generate Secret
		secret, err := m.secretGenerator()
		if err != nil {
			return err
		}
		clientSecret = base64.URLEncoding.EncodeToString(secret)
	}

	cli.Credentials = oidc.ClientCredentials{
		ID:     clientID,
		Secret: clientSecret,
	}
	return nil
}

func validateClient(cli client.Client) error {
	// NOTE: please be careful changing the errors returned here; they are used
	// downstream (eg. in the admin API) to determine the http errors returned.
	if cli.Public {
		if len(cli.Metadata.RedirectURIs) > 0 {
			return client.ErrorPublicClientRedirectURIs
		}
		if cli.Metadata.ClientName == "" {
			return client.ErrorPublicClientMissingName
		}
		cli.Metadata.RedirectURIs = []url.URL{
			localHostRedirectURL,
		}
	} else {
		if len(cli.Metadata.RedirectURIs) < 1 {
			return client.ErrorMissingRedirectURI
		}
	}

	err := cli.Metadata.Valid()
	if err != nil {
		return client.ValidationError{Err: err}
	}
	return nil
}

func mustParseURL(s string) url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return *u
}
