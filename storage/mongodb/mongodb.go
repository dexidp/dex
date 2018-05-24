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
	keyName       = "openid-connect-keys"
	defaultDBName = "dex"
)

// Setting for storage openid-connect-keys.
type Setting struct {
	ID   string `json:"id" bson:"id"`
	Item []byte `json:"item" bson:"item"`
}

// Config options for connecting to MongoDB.
// type: mongodb
// config:
//   endpoint: mongodb://user:pass@localhost:27017/testdex
type Config struct {
	Endpoint string `json:"endpoint" yaml:"endpoint"`
}

// Open creates a new storage implementation backed by MongoDB
func (c *Config) Open(logger logrus.FieldLogger) (storage.Storage, error) {
	ms, err := newMgoStorage(logger, c.Endpoint)
	if err != nil {
		return nil, err
	}

	ms.logger.Info("open a mongodb storage.")

	return ms, nil
}

type mgoStorage struct {
	mu     sync.Mutex
	logger logrus.FieldLogger
	db     *mgo.Database
}

// newMgoStorage generate a mongodb storage.
func newMgoStorage(logger logrus.FieldLogger, url string) (*mgoStorage, error) {
	ss, err := mgo.Dial(url)
	if err != nil {
		logger.Errorf("dial mongodb %s failed, %s", url, err)
		return nil, err
	}

	db := ss.DB(getDbNameFromURL(url))
	err = setupIndex(db)
	if err != nil {
		return nil, err
	}

	return &mgoStorage{
		logger: logger,
		db:     db,
	}, nil
}

func (m *mgoStorage) Close() error {
	m.db.Session.Close()
	return nil
}

func (m *mgoStorage) CreateAuthRequest(a storage.AuthRequest) error {
	return m.insertResource(ColAuthRequest, a)
}

func (m *mgoStorage) CreateClient(c storage.Client) error {
	return m.insertResource(ColClient, c)
}

func (m *mgoStorage) CreateAuthCode(c storage.AuthCode) error {
	return m.insertResource(ColAuthCode, c)
}

func (m *mgoStorage) CreateRefresh(r storage.RefreshToken) error {
	return m.insertResource(ColRefreshToken, r)
}

func (m *mgoStorage) CreatePassword(p storage.Password) error {
	m.logger.Infof("create password token %s", p.Email)
	return m.insertResource(ColPassword, p)
}

func (m *mgoStorage) CreateOfflineSessions(s storage.OfflineSessions) error {
	return m.insertResource(ColOfflineSessions, s)
}

func (m *mgoStorage) CreateConnector(c storage.Connector) error {
	return m.insertResource(ColConnector, c)
}

func (m *mgoStorage) GetAuthRequest(id string) (storage.AuthRequest, error) {
	var ret storage.AuthRequest
	err := m.getResource(ColAuthRequest, bson.M{"id": id}, &ret)
	if err != nil {
		return ret, err
	}

	// Meaningless code. just for pass the test.
	var nilArr []string
	if len(ret.Claims.Groups) == 0 {
		ret.Claims.Groups = nilArr
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

	ret.CreatedAt = ret.CreatedAt.UTC()
	ret.LastUsed = ret.LastUsed.UTC()

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
	for _, v := range ret.Refresh {
		v.CreatedAt = v.CreatedAt.UTC()
		v.LastUsed = v.LastUsed.UTC()
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
	return m.removeResource(ColAuthRequest, bson.M{"id": id})
}

func (m *mgoStorage) DeleteAuthCode(code string) error {
	return m.removeResource(ColAuthCode, bson.M{"id": code})
}

func (m *mgoStorage) DeleteClient(id string) error {
	return m.removeResource(ColClient, bson.M{"id": id})
}

func (m *mgoStorage) DeleteRefresh(id string) error {
	return m.removeResource(ColRefreshToken, bson.M{"id": id})
}

func (m *mgoStorage) DeletePassword(email string) error {
	m.logger.Infof("delete password token %s", email)
	return m.removeResource(ColPassword, bson.M{"email": email})
}

func (m *mgoStorage) DeleteOfflineSessions(userID string, connID string) error {
	return m.removeResource(ColOfflineSessions, bson.M{"userid": userID, "connid": connID})
}

func (m *mgoStorage) DeleteConnector(id string) error {
	return m.removeResource(ColConnector, bson.M{"id": id})
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

	return m.db.C(ColClient).Update(bson.M{"id": id}, updated)
}

func (m *mgoStorage) UpdateKeys(updater func(old storage.Keys) (storage.Keys, error)) error {
	current, err := m.GetKeys()
	if err != nil {
		if err != storage.ErrNotFound {
			return err
		}
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
	return err
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
	return m.db.C(ColAuthRequest).Update(bson.M{"id": id}, updated)
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
	return m.db.C(ColRefreshToken).Update(bson.M{"id": id}, updated)
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
	return m.db.C(ColPassword).Update(bson.M{"email": email}, updated)
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
	return m.db.C(ColOfflineSessions).Update(bson.M{"userid": userID, "connid": connID}, updated)
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
	return m.db.C(ColConnector).Update(bson.M{"id": id}, updated)
}

// GC authRequest & authCode
func (m *mgoStorage) GarbageCollect(now time.Time) (ret storage.GCResult, err error) {
	query := bson.M{"expiry": bson.M{"$lt": now.UTC()}}
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

func (m *mgoStorage) insertResource(col string, v interface{}) error {
	err := m.db.C(col).Insert(v)
	if err != nil {
		if mgo.IsDup(err) {
			return storage.ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (m *mgoStorage) updateResource(col string, query, updated interface{}) error {
	err := m.db.C(col).Update(query, updated)
	if err != nil {
		if err == mgo.ErrNotFound {
			return storage.ErrNotFound
		}
		return err
	}
	return nil
}

func (m *mgoStorage) removeResource(col string, query interface{}) error {
	err := m.db.C(col).Remove(query)
	if err != nil {
		if err == mgo.ErrNotFound {
			return storage.ErrNotFound
		}
		return err
	}
	return nil
}
