package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	redisv8 "github.com/go-redis/redis/v8"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/storage"
)

const (
	clientPrefix         = "client/"
	authCodePrefix       = "auth_code/"
	refreshTokenPrefix   = "refresh_token/"
	authRequestPrefix    = "auth_req/"
	passwordPrefix       = "password/"
	offlineSessionPrefix = "offline_session/"
	connectorPrefix      = "connector/"
	keysName             = "openid-connect-keys"
	deviceRequestPrefix  = "device_req/"
	deviceTokenPrefix    = "device_token/"

	defaultStorageTimeout = 5 * time.Second
)

type gcEntity struct {
	Expiry time.Time
}

type client struct {
	db     redisv8.UniversalClient
	logger log.Logger
}

func (c *client) Close() error {
	return c.db.Close()
}

func (c *client) GarbageCollect(now time.Time) (result storage.GCResult, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()

	keyPrefixs := []string{
		authRequestPrefix,
		authCodePrefix,
		deviceRequestPrefix,
		deviceTokenPrefix,
	}
	var gcCounts [4]int64
	for i := 0; i < len(keyPrefixs); i++ {
		count, err := c.garbageCollect(ctx, now, keyPrefixs[i])
		if err != nil {
			return result, err
		}
		gcCounts[i] = count
	}

	result.AuthRequests = gcCounts[0]
	result.AuthCodes = gcCounts[1]
	result.DeviceRequests = gcCounts[2]
	result.DeviceTokens = gcCounts[3]
	return result, nil
}

func (c *client) garbageCollect(ctx context.Context, now time.Time, prefix string) (count int64, err error) {
	keys, err := c.db.Keys(ctx, keyPrefix(prefix)).Result()
	if err != nil {
		return 0, err
	}
	if len(keys) == 0 {
		return 0, nil
	}
	vals, err := c.db.MGet(ctx, keys...).Result()
	if err != nil {
		return 0, err
	}

	var gcEntity gcEntity
	for i := 0; i < len(vals); i++ {
		if vals[i] == nil {
			continue
		}
		v, ok := vals[i].(string)
		if !ok {
			return 0, fmt.Errorf("%v %v failed to string error", keys[i], vals[i])
		}
		if err = json.Unmarshal([]byte(v), &gcEntity); err != nil {
			return 0, err
		}
		if now.After(gcEntity.Expiry) {
			if err = c.deleteKey(ctx, keys[i]); err != nil {
				return 0, err
			}
			count++
		}
	}
	return count, nil
}

func (c *client) CreateAuthRequest(a storage.AuthRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.createKey(ctx, keyID(authRequestPrefix, a.ID), a)
}

func (c *client) GetAuthRequest(id string) (a storage.AuthRequest, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	if err = c.getKey(ctx, keyID(authRequestPrefix, id), &a); err != nil {
		return
	}
	return a, nil
}

func (c *client) UpdateAuthRequest(id string, updater func(a storage.AuthRequest) (storage.AuthRequest, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	var val storage.AuthRequest
	return c.updateKey(ctx, keyID(authRequestPrefix, id), &val, func(old interface{}) (interface{}, error) {
		newVal, err := updater(*old.(*storage.AuthRequest))
		return newVal, err
	})
}

func (c *client) DeleteAuthRequest(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.deleteKey(ctx, keyID(authRequestPrefix, id))
}

func (c *client) CreateAuthCode(a storage.AuthCode) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.createKey(ctx, keyID(authCodePrefix, a.ID), a)
}

func (c *client) GetAuthCode(id string) (a storage.AuthCode, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err = c.getKey(ctx, keyID(authCodePrefix, id), &a)
	return a, err
}

func (c *client) DeleteAuthCode(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.deleteKey(ctx, keyID(authCodePrefix, id))
}

func (c *client) CreateRefresh(r storage.RefreshToken) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.createKey(ctx, keyID(refreshTokenPrefix, r.ID), r)
}

func (c *client) GetRefresh(id string) (r storage.RefreshToken, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	if err = c.getKey(ctx, keyID(refreshTokenPrefix, id), &r); err != nil {
		return
	}
	return r, nil
}

func (c *client) UpdateRefreshToken(id string, updater func(old storage.RefreshToken) (storage.RefreshToken, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()

	var val storage.RefreshToken
	return c.updateKey(ctx, keyID(refreshTokenPrefix, id), &val, func(old interface{}) (interface{}, error) {
		newVal, err := updater(*old.(*storage.RefreshToken))
		return newVal, err
	})
}

func (c *client) DeleteRefresh(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.deleteKey(ctx, keyID(refreshTokenPrefix, id))
}

func (c *client) ListRefreshTokens() (tokens []storage.RefreshToken, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()

	kvs, err := c.getKvs(ctx, refreshTokenPrefix)
	if err != nil {
		return nil, err
	}
	for _, val := range kvs {
		var token storage.RefreshToken
		err = json.Unmarshal([]byte(val), &token)
		if err != nil {
			return tokens, err
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func (c *client) CreateClient(cli storage.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.createKey(ctx, keyID(clientPrefix, cli.ID), cli)
}

func (c *client) GetClient(id string) (cli storage.Client, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err = c.getKey(ctx, keyID(clientPrefix, id), &cli)
	return cli, err
}

func (c *client) UpdateClient(id string, updater func(old storage.Client) (storage.Client, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()

	var val storage.Client
	return c.updateKey(ctx, keyID(clientPrefix, id), &val, func(old interface{}) (interface{}, error) {
		newVal, err := updater(*old.(*storage.Client))
		return newVal, err
	})
}

func (c *client) DeleteClient(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.deleteKey(ctx, keyID(clientPrefix, id))
}

func (c *client) ListClients() (clients []storage.Client, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()

	kvs, err := c.getKvs(ctx, clientPrefix)
	if err != nil {
		return nil, err
	}
	for _, val := range kvs {
		var client storage.Client
		err = json.Unmarshal([]byte(val), &client)
		if err != nil {
			return clients, err
		}
		clients = append(clients, client)
	}
	return clients, nil
}

func (c *client) CreatePassword(p storage.Password) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.createKey(ctx, keyEmail(p.Email), p)
}

func (c *client) GetPassword(email string) (p storage.Password, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err = c.getKey(ctx, keyEmail(email), &p)
	return p, err
}

func (c *client) UpdatePassword(email string, updater func(p storage.Password) (storage.Password, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	var val storage.Password
	return c.updateKey(ctx, keyEmail(email), &val, func(old interface{}) (interface{}, error) {
		newVal, err := updater(*old.(*storage.Password))
		return newVal, err
	})
}

func (c *client) DeletePassword(email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.deleteKey(ctx, keyEmail(email))
}

func (c *client) ListPasswords() (passwords []storage.Password, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()

	kvs, err := c.getKvs(ctx, passwordPrefix)
	if err != nil {
		return nil, err
	}
	for _, val := range kvs {
		var password storage.Password
		err = json.Unmarshal([]byte(val), &password)
		if err != nil {
			return passwords, err
		}
		passwords = append(passwords, password)
	}
	return passwords, nil
}

func (c *client) CreateOfflineSessions(s storage.OfflineSessions) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.createKey(ctx, keySession(s.UserID, s.ConnID), s)
}

func (c *client) UpdateOfflineSessions(userID string, connID string, updater func(s storage.OfflineSessions) (storage.OfflineSessions, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	var val storage.OfflineSessions
	return c.updateKey(ctx, keySession(userID, connID), &val, func(old interface{}) (interface{}, error) {
		newVal, err := updater(*old.(*storage.OfflineSessions))
		return newVal, err
	})
}

func (c *client) GetOfflineSessions(userID string, connID string) (s storage.OfflineSessions, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	if err = c.getKey(ctx, keySession(userID, connID), &s); err != nil {
		return
	}
	return s, nil
}

func (c *client) DeleteOfflineSessions(userID string, connID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.deleteKey(ctx, keySession(userID, connID))
}

func (c *client) CreateConnector(connector storage.Connector) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.createKey(ctx, keyID(connectorPrefix, connector.ID), connector)
}

func (c *client) GetConnector(id string) (client storage.Connector, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err = c.getKey(ctx, keyID(connectorPrefix, id), &client)
	return client, err
}

func (c *client) UpdateConnector(id string, updater func(s storage.Connector) (storage.Connector, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	var val storage.Connector
	return c.updateKey(ctx, keyID(connectorPrefix, id), &val, func(old interface{}) (interface{}, error) {
		newVal, err := updater(*old.(*storage.Connector))
		return newVal, err
	})
}

func (c *client) DeleteConnector(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.deleteKey(ctx, keyID(connectorPrefix, id))
}

func (c *client) ListConnectors() (connectors []storage.Connector, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()

	kvs, err := c.getKvs(ctx, connectorPrefix)
	if err != nil {
		return nil, err
	}
	for _, val := range kvs {
		var connector storage.Connector
		err = json.Unmarshal([]byte(val), &connector)
		if err != nil {
			return connectors, err
		}
		connectors = append(connectors, connector)
	}
	return connectors, nil
}

func (c *client) GetKeys() (keys storage.Keys, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()

	err = c.getKey(ctx, keysName, &keys)
	if err == storage.ErrNotFound {
		return keys, nil
	}
	return keys, err
}

func (c *client) UpdateKeys(updater func(old storage.Keys) (storage.Keys, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()

	var val storage.Keys
	err := c.getKey(ctx, keysName, &val)
	if err != nil && err != storage.ErrNotFound {
		return err
	}

	newVal, err := updater(val)
	if err != nil {
		return err
	}
	bytes, err := json.Marshal(newVal)
	if err != nil {
		return err
	}
	return c.db.Set(ctx, keysName, string(bytes), 0).Err()
}

func (c *client) CreateDeviceRequest(d storage.DeviceRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.createKey(ctx, keyID(deviceRequestPrefix, d.UserCode), d)
}

func (c *client) GetDeviceRequest(userCode string) (r storage.DeviceRequest, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err = c.getKey(ctx, keyID(deviceRequestPrefix, userCode), &r)
	return r, err
}

func (c *client) CreateDeviceToken(t storage.DeviceToken) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.createKey(ctx, keyID(deviceTokenPrefix, t.DeviceCode), t)
}

func (c *client) GetDeviceToken(deviceCode string) (t storage.DeviceToken, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err = c.getKey(ctx, keyID(deviceTokenPrefix, deviceCode), &t)
	return t, err
}

func (c *client) UpdateDeviceToken(deviceCode string, updater func(old storage.DeviceToken) (storage.DeviceToken, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()

	var val storage.DeviceToken
	return c.updateKey(ctx, keyID(deviceTokenPrefix, deviceCode), &val, func(old interface{}) (interface{}, error) {
		newVal, err := updater(*old.(*storage.DeviceToken))
		return newVal, err
	})
}

func (c *client) deleteKey(ctx context.Context, key string) error {
	val, err := c.db.Del(ctx, key).Result()
	if err != nil {
		return err
	}
	if val == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (c *client) getKey(ctx context.Context, key string, value interface{}) error {
	val, err := c.db.Get(ctx, key).Result()
	if err != nil {
		if err == redisv8.Nil {
			return storage.ErrNotFound
		}
		return err
	}
	return json.Unmarshal([]byte(val), value)
}

func (c *client) getKvs(ctx context.Context, prefix string) (map[string]string, error) {
	keys, err := c.db.Keys(ctx, keyPrefix(prefix)).Result()
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return map[string]string{}, nil
	}
	vals, err := c.db.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	kvs := make(map[string]string, len(vals))
	for i := 0; i < len(vals); i++ {
		if vals[i] == nil {
			continue
		}
		v, ok := vals[i].(string)
		if !ok {
			return nil, fmt.Errorf("%v %v failed to string error", keys[i], vals[i])
		}
		kvs[keys[i]] = v
	}
	return kvs, nil
}

func (c *client) createKey(ctx context.Context, key string, value interface{}) error {
	val, err := json.Marshal(value)
	if err != nil {
		return err
	}
	ok, err := c.db.SetNX(ctx, key, string(val), 0).Result()
	if err != nil {
		return err
	}
	if !ok {
		return storage.ErrAlreadyExists
	}
	return nil
}

func (c *client) updateKey(ctx context.Context, key string, val interface{}, updater func(interface{}) (interface{}, error)) error {
	err := c.getKey(ctx, key, val)
	if err != nil {
		return err
	}
	newVal, err := updater(val)
	if err != nil {
		return err
	}
	bytes, err := json.Marshal(newVal)
	if err != nil {
		return err
	}
	return c.db.Set(ctx, key, string(bytes), 0).Err()
}

func keyPrefix(prefix string) string {
	return prefix + "*"
}

func keyID(prefix, id string) string {
	return prefix + id
}

func keyEmail(email string) string {
	return passwordPrefix + strings.ToLower(email)
}

func keySession(userID, connID string) string {
	return offlineSessionPrefix + strings.ToLower(userID+"|"+connID)
}
