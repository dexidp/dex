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
	clientIdentityTableName = "client_identity"

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
		name:    clientIdentityTableName,
		model:   clientIdentityModel{},
		autoinc: false,
		pkey:    []string{"id"},
	})
}

func newClientIdentityModel(id string, secret []byte, meta *oidc.ClientMetadata) (*clientIdentityModel, error) {
	hashed, err := bcrypt.GenerateFromPassword(secret, bcryptHashCost)
	if err != nil {
		return nil, err
	}

	bmeta, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}

	cim := clientIdentityModel{
		ID:       id,
		Secret:   hashed,
		Metadata: string(bmeta),
	}

	return &cim, nil
}

type clientIdentityModel struct {
	ID       string `db:"id"`
	Secret   []byte `db:"secret"`
	Metadata string `db:"metadata"`
	DexAdmin bool   `db:"dex_admin"`
}

func (m *clientIdentityModel) ClientIdentity() (*oidc.ClientIdentity, error) {
	ci := oidc.ClientIdentity{
		Credentials: oidc.ClientCredentials{
			ID:     m.ID,
			Secret: string(m.Secret),
		},
	}

	if err := json.Unmarshal([]byte(m.Metadata), &ci.Metadata); err != nil {
		return nil, err
	}

	return &ci, nil
}

func NewClientIdentityRepo(dbm *gorp.DbMap) client.ClientIdentityRepo {
	return newClientIdentityRepo(dbm)
}

func newClientIdentityRepo(dbm *gorp.DbMap) *clientIdentityRepo {
	return &clientIdentityRepo{db: &db{dbm}}
}

func NewClientIdentityRepoFromClients(dbm *gorp.DbMap, clients []oidc.ClientIdentity) (client.ClientIdentityRepo, error) {
	repo := newClientIdentityRepo(dbm)
	tx, err := repo.begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	exec := repo.executor(tx)
	for _, c := range clients {
		dec, err := base64.URLEncoding.DecodeString(c.Credentials.Secret)
		if err != nil {
			return nil, err
		}
		cm, err := newClientIdentityModel(c.Credentials.ID, dec, &c.Metadata)
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

type clientIdentityRepo struct {
	*db
}

func (r *clientIdentityRepo) Metadata(clientID string) (*oidc.ClientMetadata, error) {
	m, err := r.executor(nil).Get(clientIdentityModel{}, clientID)
	if err == sql.ErrNoRows || m == nil {
		return nil, client.ErrorNotFound
	}
	if err != nil {
		return nil, err
	}

	cim, ok := m.(*clientIdentityModel)
	if !ok {
		log.Errorf("expected clientIdentityModel but found %v", reflect.TypeOf(m))
		return nil, errors.New("unrecognized model")
	}

	ci, err := cim.ClientIdentity()
	if err != nil {
		return nil, err
	}

	return &ci.Metadata, nil
}

func (r *clientIdentityRepo) IsDexAdmin(clientID string) (bool, error) {
	m, err := r.executor(nil).Get(clientIdentityModel{}, clientID)
	if m == nil || err != nil {
		return false, err
	}

	cim, ok := m.(*clientIdentityModel)
	if !ok {
		log.Errorf("expected clientIdentityModel but found %v", reflect.TypeOf(m))
		return false, errors.New("unrecognized model")
	}

	return cim.DexAdmin, nil
}

func (r *clientIdentityRepo) SetDexAdmin(clientID string, isAdmin bool) error {
	tx, err := r.begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	exec := r.executor(tx)

	m, err := exec.Get(clientIdentityModel{}, clientID)
	if m == nil || err != nil {
		return err
	}

	cim, ok := m.(*clientIdentityModel)
	if !ok {
		log.Errorf("expected clientIdentityModel but found %v", reflect.TypeOf(m))
		return errors.New("unrecognized model")
	}

	cim.DexAdmin = isAdmin
	_, err = exec.Update(cim)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *clientIdentityRepo) Authenticate(creds oidc.ClientCredentials) (bool, error) {
	m, err := r.executor(nil).Get(clientIdentityModel{}, creds.ID)
	if m == nil || err != nil {
		return false, err
	}

	cim, ok := m.(*clientIdentityModel)
	if !ok {
		log.Errorf("expected clientIdentityModel but found %v", reflect.TypeOf(m))
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

func (r *clientIdentityRepo) New(id string, meta oidc.ClientMetadata, admin bool) (*oidc.ClientCredentials, error) {
	secret, err := pcrypto.RandBytes(maxSecretLength)
	if err != nil {
		return nil, err
	}

	cim, err := newClientIdentityModel(id, secret, &meta)
	if err != nil {
		return nil, err
	}
	cim.DexAdmin = admin

	if err := r.executor(nil).Insert(cim); err != nil {
		if isAlreadyExistsErr(err) {
			err = errors.New("client ID already exists")
		}
		return nil, err
	}

	cc := oidc.ClientCredentials{
		ID:     id,
		Secret: base64.URLEncoding.EncodeToString(secret),
	}

	return &cc, nil
}

func (r *clientIdentityRepo) All() ([]oidc.ClientIdentity, error) {
	qt := r.quote(clientIdentityTableName)
	q := fmt.Sprintf("SELECT * FROM %s", qt)
	objs, err := r.executor(nil).Select(&clientIdentityModel{}, q)
	if err != nil {
		return nil, err
	}

	cs := make([]oidc.ClientIdentity, len(objs))
	for i, obj := range objs {
		m, ok := obj.(*clientIdentityModel)
		if !ok {
			return nil, errors.New("unable to cast client identity to clientIdentityModel")
		}

		ci, err := m.ClientIdentity()
		if err != nil {
			return nil, err
		}
		cs[i] = *ci
	}
	return cs, nil
}
