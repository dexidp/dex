package db

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/coreos/go-oidc/oidc"
	"github.com/go-gorp/gorp"
	"golang.org/x/crypto/bcrypt"

	"github.com/coreos/dex/client"
	pcrypto "github.com/coreos/dex/pkg/crypto"
	"github.com/coreos/dex/pkg/log"
)

const (
	clientTableName = "client_identity"

	bcryptHashCost = 10

	// Blowfish, the algorithm underlying bcrypt, has a maximum
	// password length of 72. We explicitly track and check this
	// since the bcrypt library will silently ignore portions of
	// a password past the first 72 characters.
	maxSecretLength = 72

	// postgres error codes
	pgErrorCodeUniqueViolation = "23505" // unique_violation
)

func init() {
	register(table{
		name:    clientTableName,
		model:   clientModel{},
		autoinc: false,
		pkey:    []string{"id"},
	})
}

func newClientModel(cli client.Client) (*clientModel, error) {
	secretBytes, err := base64.URLEncoding.DecodeString(cli.Credentials.Secret)
	if err != nil {
		return nil, err
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(
		secretBytes),
		bcryptHashCost)
	if err != nil {
		return nil, err
	}

	bmeta, err := json.Marshal(&cli.Metadata)
	if err != nil {
		return nil, err
	}

	cim := clientModel{
		ID:       cli.Credentials.ID,
		Secret:   hashed,
		Metadata: string(bmeta),
		DexAdmin: cli.Admin,
	}

	return &cim, nil
}

type clientModel struct {
	ID       string `db:"id"`
	Secret   []byte `db:"secret"`
	Metadata string `db:"metadata"`
	DexAdmin bool   `db:"dex_admin"`
}

func (m *clientModel) Client() (*client.Client, error) {
	ci := client.Client{
		Credentials: oidc.ClientCredentials{
			ID: m.ID,
		},
		Admin: m.DexAdmin,
	}

	if err := json.Unmarshal([]byte(m.Metadata), &ci.Metadata); err != nil {
		return nil, err
	}

	return &ci, nil
}

func NewClientRepo(dbm *gorp.DbMap) client.ClientRepo {
	return newClientRepo(dbm)
}

func newClientRepo(dbm *gorp.DbMap) *clientRepo {
	return &clientRepo{db: &db{dbm}}
}

func NewClientRepoFromClients(dbm *gorp.DbMap, clients []client.Client) (client.ClientRepo, error) {
	repo := newClientRepo(dbm)
	tx, err := repo.begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	exec := repo.executor(tx)
	for _, c := range clients {
		if c.Credentials.Secret == "" {
			return nil, fmt.Errorf("client %q has no secret", c.Credentials.ID)
		}
		cm, err := newClientModel(c)
		if err != nil {
			return nil, err
		}
		err = exec.Insert(cm)
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return repo, nil
}

type clientRepo struct {
	*db
}

func (r *clientRepo) Get(clientID string) (client.Client, error) {
	m, err := r.executor(nil).Get(clientModel{}, clientID)
	if err == sql.ErrNoRows || m == nil {
		return client.Client{}, client.ErrorNotFound
	}
	if err != nil {
		return client.Client{}, err
	}

	cim, ok := m.(*clientModel)
	if !ok {
		log.Errorf("expected clientModel but found %v", reflect.TypeOf(m))
		return client.Client{}, errors.New("unrecognized model")
	}

	ci, err := cim.Client()
	if err != nil {
		return client.Client{}, err
	}

	return *ci, nil
}

func (r *clientRepo) Metadata(clientID string) (*oidc.ClientMetadata, error) {
	c, err := r.Get(clientID)
	if err != nil {
		return nil, err
	}

	return &c.Metadata, nil
}

func (r *clientRepo) IsDexAdmin(clientID string) (bool, error) {
	m, err := r.executor(nil).Get(clientModel{}, clientID)
	if m == nil || err != nil {
		return false, err
	}

	cim, ok := m.(*clientModel)
	if !ok {
		log.Errorf("expected clientModel but found %v", reflect.TypeOf(m))
		return false, errors.New("unrecognized model")
	}

	return cim.DexAdmin, nil
}

func (r *clientRepo) SetDexAdmin(clientID string, isAdmin bool) error {
	tx, err := r.begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	exec := r.executor(tx)

	m, err := exec.Get(clientModel{}, clientID)
	if m == nil || err != nil {
		return err
	}

	cim, ok := m.(*clientModel)
	if !ok {
		log.Errorf("expected clientModel but found %v", reflect.TypeOf(m))
		return errors.New("unrecognized model")
	}

	cim.DexAdmin = isAdmin
	_, err = exec.Update(cim)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *clientRepo) Authenticate(creds oidc.ClientCredentials) (bool, error) {
	m, err := r.executor(nil).Get(clientModel{}, creds.ID)
	if m == nil || err != nil {
		return false, err
	}

	cim, ok := m.(*clientModel)
	if !ok {
		log.Errorf("expected clientModel but found %v", reflect.TypeOf(m))
		return false, errors.New("unrecognized model")
	}

	dec, err := base64.URLEncoding.DecodeString(creds.Secret)
	if err != nil {
		log.Errorf("error Decoding client creds: %v", err)
		return false, nil
	}

	if len(dec) > maxSecretLength {
		return false, nil
	}

	ok = bcrypt.CompareHashAndPassword(cim.Secret, dec) == nil
	return ok, nil
}

var alreadyExistsCheckers []func(err error) bool

func registerAlreadyExistsChecker(f func(err error) bool) {
	alreadyExistsCheckers = append(alreadyExistsCheckers, f)
}

// isAlreadyExistsErr detects database error codes for failing a unique constraint.
//
// Because database drivers are optionally compiled, use registerAlreadyExistsChecker to
// register driver specific implementations.
func isAlreadyExistsErr(err error) bool {
	for _, checker := range alreadyExistsCheckers {
		if checker(err) {
			return true
		}
	}
	return false
}

func (r *clientRepo) New(cli client.Client) (*oidc.ClientCredentials, error) {
	secret, err := pcrypto.RandBytes(maxSecretLength)
	if err != nil {
		return nil, err
	}

	cli.Credentials.Secret = base64.URLEncoding.EncodeToString(secret)
	cim, err := newClientModel(cli)

	if err != nil {
		return nil, err
	}

	if err := r.executor(nil).Insert(cim); err != nil {
		if isAlreadyExistsErr(err) {
			err = errors.New("client ID already exists")
		}
		return nil, err
	}

	cc := oidc.ClientCredentials{
		ID:     cli.Credentials.ID,
		Secret: cli.Credentials.Secret,
	}

	return &cc, nil
}

func (r *clientRepo) All() ([]client.Client, error) {
	qt := r.quote(clientTableName)
	q := fmt.Sprintf("SELECT * FROM %s", qt)
	objs, err := r.executor(nil).Select(&clientModel{}, q)
	if err != nil {
		return nil, err
	}

	cs := make([]client.Client, len(objs))
	for i, obj := range objs {
		m, ok := obj.(*clientModel)
		if !ok {
			return nil, errors.New("unable to cast client identity to clientModel")
		}

		ci, err := m.Client()
		if err != nil {
			return nil, err
		}
		cs[i] = *ci
	}
	return cs, nil
}
