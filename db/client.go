package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/coreos/go-oidc/oidc"
	"github.com/go-gorp/gorp"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/repo"
)

const (
	clientTableName      = "client_identity"
	trustedPeerTableName = "trusted_peers"

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

	register(table{
		name:    trustedPeerTableName,
		model:   trustedPeerModel{},
		autoinc: false,
		pkey:    []string{"client_id", "trusted_client_id"},
	})
}

func newClientModel(cli client.Client) (*clientModel, error) {
	hashed, err := client.HashSecret(cli.Credentials)
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

type trustedPeerModel struct {
	ClientID        string `db:"client_id"`
	TrustedClientID string `db:"trusted_client_id"`
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
	return &clientRepo{
		db: &db{dbm},
	}
}

type clientRepo struct {
	*db
}

func (r *clientRepo) Get(tx repo.Transaction, clientID string) (client.Client, error) {
	m, err := r.executor(tx).Get(clientModel{}, clientID)
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

func (r *clientRepo) GetSecret(tx repo.Transaction, clientID string) ([]byte, error) {
	m, err := r.getModel(tx, clientID)
	if err != nil || m == nil {
		return nil, err
	}
	return m.Secret, nil
}

func (r *clientRepo) Update(tx repo.Transaction, cli client.Client) error {
	if cli.Credentials.ID == "" {
		return client.ErrorNotFound
	}
	// make sure this client exists already
	_, err := r.get(tx, cli.Credentials.ID)
	if err != nil {
		return err
	}
	err = r.update(tx, cli)
	if err != nil {
		return err
	}
	return nil
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

func (r *clientRepo) New(tx repo.Transaction, cli client.Client) (*oidc.ClientCredentials, error) {
	cim, err := newClientModel(cli)

	if err != nil {
		return nil, err
	}

	if err := r.executor(tx).Insert(cim); err != nil {
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

func (r *clientRepo) All(tx repo.Transaction) ([]client.Client, error) {
	qt := r.quote(clientTableName)
	q := fmt.Sprintf("SELECT * FROM %s", qt)
	objs, err := r.executor(tx).Select(&clientModel{}, q)
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

func NewClientRepoFromClients(dbm *gorp.DbMap, cs []client.Client) (client.ClientRepo, error) {
	repo := NewClientRepo(dbm).(*clientRepo)
	for _, c := range cs {
		cm, err := newClientModel(c)
		if err != nil {
			return nil, err
		}
		err = repo.executor(nil).Insert(cm)
	}
	return repo, nil
}

func (r *clientRepo) get(tx repo.Transaction, clientID string) (client.Client, error) {
	cm, err := r.getModel(tx, clientID)
	if err != nil {
		return client.Client{}, err
	}

	cli, err := cm.Client()
	if err != nil {
		return client.Client{}, err
	}

	return *cli, nil
}

func (r *clientRepo) getModel(tx repo.Transaction, clientID string) (*clientModel, error) {
	ex := r.executor(tx)

	m, err := ex.Get(clientModel{}, clientID)
	if err != nil {
		return nil, err
	}

	if m == nil {
		return nil, client.ErrorNotFound
	}

	cm, ok := m.(*clientModel)
	if !ok {
		log.Errorf("expected clientModel but found %v", reflect.TypeOf(m))
		return nil, errors.New("unrecognized model")
	}
	return cm, nil
}

func (r *clientRepo) update(tx repo.Transaction, cli client.Client) error {
	ex := r.executor(tx)
	cm, err := newClientModel(cli)
	if err != nil {
		return err
	}
	_, err = ex.Update(cm)
	return err
}

func (r *clientRepo) GetTrustedPeers(tx repo.Transaction, clientID string) ([]string, error) {
	ex := r.executor(tx)
	if clientID == "" {
		return nil, client.ErrorInvalidClientID
	}

	qt := r.quote(trustedPeerTableName)
	var ids []string
	_, err := ex.Select(&ids, fmt.Sprintf("SELECT trusted_client_id from %s where client_id = $1", qt), clientID)

	if err != nil {
		if err != sql.ErrNoRows {
			return nil, err
		}
		return nil, nil
	}

	return ids, nil
}

func (r *clientRepo) SetTrustedPeers(tx repo.Transaction, clientID string, clientIDs []string) error {
	ex := r.executor(tx)
	qt := r.quote(trustedPeerTableName)

	// First delete all existing rows
	_, err := ex.Exec(fmt.Sprintf("DELETE from %s where client_id = $1", qt), clientID)
	if err != nil {
		return err
	}

	// Ensure that the client exists.
	_, err = r.get(tx, clientID)
	if err != nil {
		return err
	}

	// Verify that all the clients are valid
	for _, curID := range clientIDs {
		_, err := r.get(tx, curID)
		if err != nil {
			return err
		}
	}

	// Set the clients
	rows := []interface{}{}
	for _, curID := range clientIDs {
		rows = append(rows, &trustedPeerModel{
			ClientID:        clientID,
			TrustedClientID: curID,
		})
	}
	err = ex.Insert(rows...)
	if err != nil {
		return err
	}

	return nil
}
