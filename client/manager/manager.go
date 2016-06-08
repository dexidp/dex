package manager

import (
	"encoding/base64"

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

func (m *ClientManager) New(cli client.Client) (*oidc.ClientCredentials, error) {
	tx, err := m.begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

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
	if err != nil || clientSecret == nil {
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
	// Generate Client ID
	if len(cli.Metadata.RedirectURIs) < 1 {
		return errors.New("no client redirect url given")
	}
	clientID, err := m.clientIDGenerator(cli.Metadata.RedirectURIs[0].Host)
	if err != nil {
		return err
	}

	// Generate Secret
	secret, err := m.secretGenerator()
	if err != nil {
		return err
	}
	clientSecret := base64.URLEncoding.EncodeToString(secret)
	cli.Credentials = oidc.ClientCredentials{
		ID:     clientID,
		Secret: clientSecret,
	}
	return nil
}
