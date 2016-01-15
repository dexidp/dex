package db

import (
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/coreos/dex/db"
	pcrypto "github.com/coreos/dex/pkg/crypto"
	"github.com/coreos/go-oidc/key"
)

const (
	keyTableName = "key"
)

func init() {
	register(table{
		name:    keyTableName,
		model:   privateKeySetBlob{},
		autoinc: false,
	})
}

func newPrivateKeySetModel(pks *key.PrivateKeySet) (*privateKeySetModel, error) {
	pkeys := pks.Keys()
	keys := make([]privateKeyModel, len(pkeys))
	for i, pkey := range pkeys {
		keys[i] = privateKeyModel{
			ID:    pkey.ID(),
			PKCS1: x509.MarshalPKCS1PrivateKey(pkey.PrivateKey),
		}
	}

	m := privateKeySetModel{
		Keys:      keys,
		ExpiresAt: pks.ExpiresAt(),
	}

	return &m, nil
}

type privateKeyModel struct {
	ID    string `json:"id"`
	PKCS1 []byte `json:"pkcs1"`
}

func (m *privateKeyModel) PrivateKey() (*key.PrivateKey, error) {
	d, err := x509.ParsePKCS1PrivateKey(m.PKCS1)
	if err != nil {
		return nil, err
	}

	pk := key.PrivateKey{
		KeyID:      m.ID,
		PrivateKey: d,
	}

	return &pk, nil
}

type privateKeySetModel struct {
	Keys      []privateKeyModel `json:"keys"`
	ExpiresAt time.Time         `json:"expires_at"`
}

func (m *privateKeySetModel) PrivateKeySet() (*key.PrivateKeySet, error) {
	keys := make([]*key.PrivateKey, len(m.Keys))
	for i, pkm := range m.Keys {
		pk, err := pkm.PrivateKey()
		if err != nil {
			return nil, err
		}
		keys[i] = pk
	}
	return key.NewPrivateKeySet(keys, m.ExpiresAt), nil
}

type privateKeySetBlob struct {
	Value []byte `db:"value"`
}

func NewPrivateKeySetRepo(dbm *gorp.DbMap, useOldFormat bool, secrets ...[]byte) (*PrivateKeySetRepo, error) {
	if len(secrets) == 0 {
		return nil, errors.New("must provide at least one key secret")
	}
	for i, secret := range secrets {
		if len(secret) != 32 {
			return nil, fmt.Errorf("key secret %d: expected 32-byte secret", i)
		}
	}

	r := &PrivateKeySetRepo{
		dbMap:        dbm,
		useOldFormat: useOldFormat,
		secrets:      secrets,
	}

	return r, nil
}

type PrivateKeySetRepo struct {
	dbMap        *gorp.DbMap
	useOldFormat bool
	secrets      [][]byte
}

func (r *PrivateKeySetRepo) Set(ks key.KeySet) error {
	qt := pq.QuoteIdentifier(keyTableName)
	_, err := r.dbMap.Exec(fmt.Sprintf("DELETE FROM %s", qt))
	if err != nil {
		return err
	}

	pks, ok := ks.(*key.PrivateKeySet)
	if !ok {
		return errors.New("unable to cast to PrivateKeySet")
	}

	m, err := newPrivateKeySetModel(pks)
	if err != nil {
		return err
	}

	j, err := json.Marshal(m)
	if err != nil {
		return err
	}

	var v []byte

	if r.useOldFormat {
		v, err = pcrypto.AESEncrypt(j, r.active())
	} else {
		v, err = pcrypto.Encrypt(j, r.active())
	}

	if err != nil {
		return err
	}

	b := &privateKeySetBlob{Value: v}
	return r.dbMap.Insert(b)
}

func (r *PrivateKeySetRepo) Get() (key.KeySet, error) {
	qt := pq.QuoteIdentifier(keyTableName)
	objs, err := r.dbMap.Select(&privateKeySetBlob{}, fmt.Sprintf("SELECT * FROM %s", qt))
	if err != nil {
		return nil, err
	}

	if len(objs) == 0 {
		return nil, key.ErrorNoKeys
	}

	b, ok := objs[0].(*privateKeySetBlob)
	if !ok {
		return nil, errors.New("unable to cast to KeySet")
	}

	var pks *key.PrivateKeySet
	for _, secret := range r.secrets {
		var j []byte

		if r.useOldFormat {
			j, err = pcrypto.AESDecrypt(b.Value, secret)
		} else {
			j, err = pcrypto.Decrypt(b.Value, secret)
		}

		if err != nil {
			continue
		}

		var m privateKeySetModel
		if err = json.Unmarshal(j, &m); err != nil {
			continue
		}

		pks, err = m.PrivateKeySet()
		if err != nil {
			continue
		}
		break
	}

	if err != nil {
		return nil, db.ErrorCannotDecryptKeys
	}
	return key.KeySet(pks), nil
}

func (r *PrivateKeySetRepo) active() []byte {
	return r.secrets[0]
}
