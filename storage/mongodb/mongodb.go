package mongodb

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/coreos/dex/storage"
)

const (
	keyName = "openid-connect-keys"
)

// Setting: for storage openid-connect-keys.
type Setting struct {
	ID   string `json:"id" bson:"id"`
	Item []byte `json:"item" bson:"item"`
}

// Config
// type: mongodb
// config:
//   endpoint: mongodb://user:pass@localhost:27017
//   db: testdex
type Config struct {
	Endpoint string `json:"endpoint" yaml:"endpoint"`
	DB       string `json:"db" yaml:"db"`
}

func (c *Config) Open(logger logrus.FieldLogger) (storage.Storage, error) {
	ss, err := mgo.Dial(c.Endpoint)
	if err != nil {
		logger.Errorf("dial mongodb %s failed, %s", c.Endpoint, err)
		return nil, err
	}

	ms := &mgoStorage{
		logger: logger,
		db:     ss.DB(c.DB),
	}

	setupIndex(ms.db)
	ms.logger.Info("open a mongodb storage.")

	return ms, nil
}

type mgoStorage struct {
	mu     sync.Mutex
	logger logrus.FieldLogger
	db     *mgo.Database
}

func (m *mgoStorage) Close() error {
	m.db.Session.Close()
	return nil
}

func (m *mgoStorage) CreateAuthRequest(a storage.AuthRequest) error {
	err := m.db.C(ColAuthRequest).Insert(a)
	if err != nil {
		m.logger.Errorf("insert auth-request failed, %s", err)
		return err
	}
	return nil
}

func (m *mgoStorage) CreateClient(c storage.Client) error {
	err := m.db.C(ColClient).Insert(c)
	if err != nil {
		m.logger.Errorf("insert client failed, %s", err)
		return err
	}
	return nil
}

func (m *mgoStorage) CreateAuthCode(c storage.AuthCode) error {
	err := m.db.C(ColAuthCode).Insert(c)
	if err != nil {
		m.logger.Errorf("insert auth-code failed, %s", err)
		return err
	}
	return nil
}

func (m *mgoStorage) CreateRefresh(r storage.RefreshToken) error {
	err := m.db.C(ColRefreshToken).Insert(r)
	if err != nil {
		m.logger.Errorf("insert refresh failed, %s", err)
		return err
	}
	return nil
}

func (m *mgoStorage) CreatePassword(p storage.Password) error {
	err := m.db.C(ColPassword).Insert(p)
	if err != nil {
		m.logger.Errorf("insert password failed, %s", err)
		return err
	}
	return nil
}

func (m *mgoStorage) CreateOfflineSessions(s storage.OfflineSessions) error {
	err := m.db.C(ColOfflineSessions).Insert(s)
	if err != nil {
		m.logger.Errorf("insert offline-sessions failed, %s", err)
		return err
	}
	return nil
}

func (m *mgoStorage) CreateConnector(c storage.Connector) error {
	err := m.db.C(ColConnector).Insert(c)
	if err != nil {
		m.logger.Errorf("insert connector failed, %s", err)
		return err
	}
	return nil
}

func (m *mgoStorage) GetAuthRequest(id string) (storage.AuthRequest, error) {
	var ret storage.AuthRequest
	err := m.getResource(ColAuthRequest, bson.M{"id": id}, &ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func (m *mgoStorage) GetAuthCode(id string) (storage.AuthCode, error) {
	var ret storage.AuthCode
	err := m.getResource(ColAuthCode, bson.M{"id": id}, &ret)
	if err != nil {
		return ret, err
	}
	return ret, nil

}

func (m *mgoStorage) GetClient(id string) (storage.Client, error) {
	var ret storage.Client
	err := m.getResource(ColClient, bson.M{"id": id}, &ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func (m *mgoStorage) GetKeys() (storage.Keys, error) {
	var (
		ret  Setting
		keys storage.Keys
	)
	err := m.getResource(ColSetting, bson.M{"id": keyName}, &ret)
	if err != nil {
		if err == mgo.ErrNotFound {
			return storage.Keys{}, nil
		}
		return storage.Keys{}, err
	}

	err = json.Unmarshal(ret.Item, &keys)
	if err != nil {
		return storage.Keys{}, err
	}

	return keys, nil
}

func (m *mgoStorage) GetRefresh(id string) (storage.RefreshToken, error) {
	var ret storage.RefreshToken
	err := m.getResource(ColRefreshToken, bson.M{"id": id}, &ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func (m *mgoStorage) GetPassword(email string) (storage.Password, error) {
	var ret storage.Password
	err := m.getResource(ColPassword, bson.M{"email": email}, &ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func (m *mgoStorage) GetOfflineSessions(userID string, connID string) (storage.OfflineSessions, error) {
	var ret storage.OfflineSessions
	err := m.getResource(ColOfflineSessions, bson.M{"userid": userID, "connid": connID}, &ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func (m *mgoStorage) GetConnector(id string) (storage.Connector, error) {
	var ret storage.Connector
	err := m.getResource(ColConnector, bson.M{"id": id}, &ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func (m *mgoStorage) ListClients() ([]storage.Client, error) {
	ret := []storage.Client{}
	err := m.listResources(ColClient, nil, &ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func (m *mgoStorage) ListRefreshTokens() ([]storage.RefreshToken, error) {
	ret := []storage.RefreshToken{}
	err := m.listResources(ColRefreshToken, nil, &ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func (m *mgoStorage) ListPasswords() ([]storage.Password, error) {
	ret := []storage.Password{}
	err := m.listResources(ColPassword, nil, &ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func (m *mgoStorage) ListConnectors() ([]storage.Connector, error) {
	ret := []storage.Connector{}
	err := m.listResources(ColConnector, nil, &ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func (m *mgoStorage) DeleteAuthRequest(id string) error {
	_, err := m.db.C(ColAuthRequest).RemoveAll(bson.M{"id": id})
	if err != nil {
		m.logger.Errorf("delete %s failed, %s", ColAuthRequest, err)
		return err
	}
	return nil
}

func (m *mgoStorage) DeleteAuthCode(code string) error {
	_, err := m.db.C(ColAuthCode).RemoveAll(bson.M{"id": code})
	if err != nil {
		m.logger.Errorf("delete %s failed, %s", ColAuthCode, err)
		return err
	}
	return nil
}

func (m *mgoStorage) DeleteClient(id string) error {
	_, err := m.db.C(ColClient).RemoveAll(bson.M{"id": id})
	if err != nil {
		m.logger.Errorf("delete %s failed, %s", ColClient, err)
		return err
	}
	return nil
}

func (m *mgoStorage) DeleteRefresh(id string) error {
	_, err := m.db.C(ColRefreshToken).RemoveAll(bson.M{"id": id})
	if err != nil {
		m.logger.Errorf("delete %s failed, %s", ColRefreshToken, err)
		return err
	}
	return nil
}

func (m *mgoStorage) DeletePassword(email string) error {
	_, err := m.db.C(ColPassword).RemoveAll(bson.M{"email": email})
	if err != nil {
		m.logger.Errorf("delete %s failed, %s", ColPassword, err)
		return err
	}
	return nil
}

func (m *mgoStorage) DeleteOfflineSessions(userID string, connID string) error {
	_, err := m.db.C(ColOfflineSessions).RemoveAll(bson.M{"userid": userID, "connid": connID})
	if err != nil {
		m.logger.Errorf("delete %s failed, %s", ColOfflineSessions, err)
		return err
	}
	return nil
}

func (m *mgoStorage) DeleteConnector(id string) error {
	_, err := m.db.C(ColConnector).RemoveAll(bson.M{"id": id})
	if err != nil {
		m.logger.Errorf("delete %s failed, %s", ColConnector, err)
		return err
	}
	return nil
}

func (m *mgoStorage) UpdateClient(id string, updater func(old storage.Client) (storage.Client, error)) error {
	current, err := m.GetClient(id)
	if err != nil {
		return err
	}
	updated, err := updater(current)
	if err != nil {
		return err
	}

	_, err = m.db.C(ColClient).Upsert(bson.M{"id": id}, updated)
	return err
}

func (m *mgoStorage) UpdateKeys(updater func(old storage.Keys) (storage.Keys, error)) error {
	current, err := m.GetKeys()
	if err != nil {
		return err
	}

	updated, err := updater(current)
	if err != nil {
		return err
	}

	bs, err := json.Marshal(updated)
	if err != nil {
		return err
	}

	_, err = m.db.C(ColSetting).Upsert(bson.M{"id": keyName}, Setting{keyName, bs})
	if err != nil {
		m.logger.Errorf("update keys failed, %s", err)
		return err
	}
	return nil
}

func (m *mgoStorage) UpdateAuthRequest(id string, updater func(a storage.AuthRequest) (storage.AuthRequest, error)) error {
	current, err := m.GetAuthRequest(id)
	if err != nil {
		return err
	}

	updated, err := updater(current)
	if err != nil {
		return err
	}
	_, err = m.db.C(ColAuthRequest).Upsert(bson.M{"id": id}, updated)
	return err
}

func (m *mgoStorage) UpdateRefreshToken(id string, updater func(r storage.RefreshToken) (storage.RefreshToken, error)) error {
	current, err := m.GetRefresh(id)
	if err != nil {
		return err
	}

	updated, err := updater(current)
	if err != nil {
		return err
	}
	_, err = m.db.C(ColRefreshToken).Upsert(bson.M{"id": id}, updated)
	return err
}

func (m *mgoStorage) UpdatePassword(email string, updater func(p storage.Password) (storage.Password, error)) error {
	current, err := m.GetPassword(email)
	if err != nil {
		return err
	}

	updated, err := updater(current)
	if err != nil {
		return err
	}
	_, err = m.db.C(ColPassword).Upsert(bson.M{"email": email}, updated)
	return err
}

func (m *mgoStorage) UpdateOfflineSessions(userID string, connID string, updater func(s storage.OfflineSessions) (storage.OfflineSessions, error)) error {
	current, err := m.GetOfflineSessions(userID, connID)
	if err != nil {
		return err
	}

	updated, err := updater(current)
	if err != nil {
		return err
	}
	_, err = m.db.C(ColOfflineSessions).Upsert(bson.M{"userid": userID, "connid": connID}, updated)
	return err
}

func (m *mgoStorage) UpdateConnector(id string, updater func(c storage.Connector) (storage.Connector, error)) error {
	current, err := m.GetConnector(id)
	if err != nil {
		return err
	}

	updated, err := updater(current)
	if err != nil {
		return err
	}
	_, err = m.db.C(ColConnector).Upsert(bson.M{"id": id}, updated)
	return err
}

// GC authRequest & authCode
func (m *mgoStorage) GarbageCollect(now time.Time) (ret storage.GCResult, err error) {
	query := bson.M{"expiry": bson.M{"$lt": now}}
	info, err := m.db.C(ColAuthCode).RemoveAll(query)
	if err != nil {
		return ret, err
	}
	ret.AuthCodes = int64(info.Removed)

	info, err = m.db.C(ColAuthRequest).RemoveAll(query)
	if err != nil {
		return ret, err
	}
	ret.AuthRequests = int64(info.Removed)
	return ret, nil
}

func (m *mgoStorage) getResource(col string, query interface{}, ret interface{}) error {
	err := m.db.C(col).Find(query).One(ret)
	if err != nil {
		if err == mgo.ErrNotFound {
			return storage.ErrNotFound
		}
		m.logger.Errorf("get %s by query(%+v) failed, %s", col, query, err)
		return err
	}
	return nil
}

func (m *mgoStorage) listResources(c string, query interface{}, ret interface{}) error {
	err := m.db.C(c).Find(query).All(ret)
	if err != nil {
		m.logger.Errorf("list %s by query(%+v) failed, %s", c, query, err)
		return err
	}
	return nil
}
