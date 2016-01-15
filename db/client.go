package db

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"

	"github.com/coreos/go-oidc/oidc"
	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
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

	bmeta, err := json.Marshal(newClientMetadataJSON(meta))
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

func newClientMetadataJSON(cm *oidc.ClientMetadata) *clientMetadataJSON {
	cmj := clientMetadataJSON{
		RedirectURLs: make([]string, len(cm.RedirectURLs)),
	}

	for i, u := range cm.RedirectURLs {
		cmj.RedirectURLs[i] = (&u).String()
	}

	return &cmj
}

type clientMetadataJSON struct {
	RedirectURLs []string `json:"redirectURLs"`
}

func (cmj clientMetadataJSON) ClientMetadata() (*oidc.ClientMetadata, error) {
	cm := oidc.ClientMetadata{
		RedirectURLs: make([]url.URL, len(cmj.RedirectURLs)),
	}

	for i, us := range cmj.RedirectURLs {
		up, err := url.Parse(us)
		if err != nil {
			return nil, err
		}
		cm.RedirectURLs[i] = *up
	}

	return &cm, nil
}

func (m *clientIdentityModel) ClientIdentity() (*oidc.ClientIdentity, error) {
	ci := oidc.ClientIdentity{
		Credentials: oidc.ClientCredentials{
			ID:     m.ID,
			Secret: string(m.Secret),
		},
	}

	var cmj clientMetadataJSON
	err := json.Unmarshal([]byte(m.Metadata), &cmj)
	if err != nil {
		return nil, err
	}

	cm, err := cmj.ClientMetadata()
	if err != nil {
		return nil, err
	}

	ci.Metadata = *cm
	return &ci, nil
}

func NewClientIdentityRepo(dbm *gorp.DbMap) client.ClientIdentityRepo {
	return &clientIdentityRepo{dbMap: dbm}
}

func NewClientIdentityRepoFromClients(dbm *gorp.DbMap, clients []oidc.ClientIdentity) (client.ClientIdentityRepo, error) {
	repo := NewClientIdentityRepo(dbm).(*clientIdentityRepo)
	for _, c := range clients {
		dec, err := base64.URLEncoding.DecodeString(c.Credentials.Secret)
		if err != nil {
			return nil, err
		}

		cm, err := newClientIdentityModel(c.Credentials.ID, dec, &c.Metadata)
		if err != nil {
			return nil, err
		}
		err = repo.dbMap.Insert(cm)
		if err != nil {
			return nil, err
		}
	}
	return repo, nil
}

type clientIdentityRepo struct {
	dbMap *gorp.DbMap
}

func (r *clientIdentityRepo) Metadata(clientID string) (*oidc.ClientMetadata, error) {
	m, err := r.dbMap.Get(clientIdentityModel{}, clientID)
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
	m, err := r.dbMap.Get(clientIdentityModel{}, clientID)
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
	tx, err := r.dbMap.Begin()
	if err != nil {
		return err
	}

	m, err := r.dbMap.Get(clientIdentityModel{}, clientID)
	if m == nil || err != nil {
		rollback(tx)
		return err
	}

	cim, ok := m.(*clientIdentityModel)
	if !ok {
		rollback(tx)
		log.Errorf("expected clientIdentityModel but found %v", reflect.TypeOf(m))
		return errors.New("unrecognized model")
	}

	cim.DexAdmin = isAdmin
	_, err = r.dbMap.Update(cim)
	if err != nil {
		rollback(tx)
		return err
	}

	err = tx.Commit()
	if err != nil {
		rollback(tx)
		return err
	}

	return nil
}

func (r *clientIdentityRepo) Authenticate(creds oidc.ClientCredentials) (bool, error) {
	m, err := r.dbMap.Get(clientIdentityModel{}, creds.ID)
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

func (r *clientIdentityRepo) New(id string, meta oidc.ClientMetadata) (*oidc.ClientCredentials, error) {
	secret, err := pcrypto.RandBytes(maxSecretLength)
	if err != nil {
		return nil, err
	}

	cim, err := newClientIdentityModel(id, secret, &meta)
	if err != nil {
		return nil, err
	}

	if err := r.dbMap.Insert(cim); err != nil {
		if perr, ok := err.(*pq.Error); ok && perr.Code == pgErrorCodeUniqueViolation {
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
	qt := pq.QuoteIdentifier(clientIdentityTableName)
	q := fmt.Sprintf("SELECT * FROM %s", qt)
	objs, err := r.dbMap.Select(&clientIdentityModel{}, q)
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
