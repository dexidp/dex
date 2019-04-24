package redis

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/storage"
	"github.com/go-redis/redis"
)

func encode(i interface{}) (string, error) {
	b, err := json.Marshal(i)
	return string(b), err
}

func decode(i string, out interface{}) error {
	return json.Unmarshal([]byte(i), out)
}

// Config contains options to create a Redis client
type Config redis.Options

// Open creates a new storage implementation backed by Redis
func (c *Config) Open(logger log.Logger) (storage.Storage, error) {
	db := redis.NewClient((*redis.Options)(c))
	err := db.Ping().Err()
	if err != nil {
		return nil, err
	}
	return (*conn)(db), nil
}

type conn redis.Client

// AuthRequest

const keyAuthRequestPrefix = "dex:auth:request:"

func (c *conn) keyAuthRequest(id string) string {
	return keyAuthRequestPrefix + id
}

func (c *conn) getAuthRequest(key string) (storage.AuthRequest, error) {
	var a storage.AuthRequest
	val, err := c.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			return a, storage.ErrNotFound
		}
		return a, err
	}
	err = decode(val, &a)
	return a, err
}

func (c *conn) listAuthRequests() ([]storage.AuthRequest, error) {
	keys, err := c.Keys(c.keyAuthRequest("*")).Result()
	if err != nil {
		return nil, err
	}
	requests := make([]storage.AuthRequest, len(keys))
	for i, k := range keys {
		requests[i], err = c.getAuthRequest(k)
		if err != nil {
			return nil, err
		}
	}
	return requests, nil
}

func (c *conn) setAuthRequest(a storage.AuthRequest) error {
	key := c.keyAuthRequest(a.ID)
	val, err := encode(a)
	if err != nil {
		return err
	}
	return c.Set(key, val, 0).Err()
}

// AuthCode

const keyAuthCodePrefix = "dex:auth:code:"

func (c *conn) keyAuthCode(id string) string {
	return keyAuthCodePrefix + id
}

func (c *conn) getAuthCode(key string) (storage.AuthCode, error) {
	var a storage.AuthCode
	val, err := c.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			return a, storage.ErrNotFound
		}
		return a, err
	}
	err = decode(val, &a)
	return a, err
}

func (c *conn) listAuthCodes() ([]storage.AuthCode, error) {
	keys, err := c.Keys(c.keyAuthCode("*")).Result()
	if err != nil {
		return nil, err
	}
	codes := make([]storage.AuthCode, len(keys))
	for i, k := range keys {
		codes[i], err = c.getAuthCode(k)
		if err != nil {
			return nil, err
		}
	}
	return codes, nil
}

func (c *conn) setAuthCode(a storage.AuthCode) error {
	key := c.keyAuthCode(a.ID)
	val, err := encode(a)
	if err != nil {
		return err
	}
	return c.Set(key, val, 0).Err()
}

// Client

const keyClientPrefix = "dex:client:"

func (c *conn) keyClient(id string) string {
	return keyClientPrefix + id
}

func (c *conn) getClient(key string) (storage.Client, error) {
	var cli storage.Client
	val, err := c.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			return cli, storage.ErrNotFound
		}
		return cli, err
	}
	err = decode(val, &cli)
	return cli, err
}

func (c *conn) listClients() ([]storage.Client, error) {
	keys, err := c.Keys(c.keyClient("*")).Result()
	if err != nil {
		return nil, err
	}
	clients := make([]storage.Client, len(keys))
	for i, k := range keys {
		clients[i], err = c.getClient(k)
		if err != nil {
			return nil, err
		}
	}
	return clients, nil
}

func (c *conn) setClient(cli storage.Client) error {
	key := c.keyClient(cli.ID)
	val, err := encode(cli)
	if err != nil {
		return err
	}
	return c.Set(key, val, 0).Err()
}

// Keys

const keyKeys = "dex:keys"

func (c *conn) getKeys(key string) (storage.Keys, error) {
	var k storage.Keys
	val, err := c.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			return k, storage.ErrNotFound
		}
		return k, err
	}
	err = decode(val, &k)
	return k, err
}

func (c *conn) setKeys(k storage.Keys) error {
	val, err := encode(k)
	if err != nil {
		return err
	}
	return c.Set(keyKeys, val, 0).Err()
}

// Refresh

const keyRefreshPrefix = "dex:refresh:"

func (c *conn) keyRefresh(id string) string {
	return keyRefreshPrefix + id
}

func (c *conn) getRefresh(key string) (storage.RefreshToken, error) {
	var r storage.RefreshToken
	val, err := c.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			return r, storage.ErrNotFound
		}
		return r, err
	}
	err = decode(val, &r)
	return r, err
}

func (c *conn) listRefresh() ([]storage.RefreshToken, error) {
	keys, err := c.Keys(c.keyRefresh("*")).Result()
	if err != nil {
		return nil, err
	}
	refreshs := make([]storage.RefreshToken, len(keys))
	for i, k := range keys {
		refreshs[i], err = c.getRefresh(k)
		if err != nil {
			return nil, err
		}
	}
	return refreshs, nil
}

func (c *conn) setRefresh(r storage.RefreshToken) error {
	key := c.keyRefresh(r.ID)
	val, err := encode(r)
	if err != nil {
		return err
	}
	return c.Set(key, val, 0).Err()
}

// Password

const keyPasswordPrefix = "dex:password:"

func (c *conn) keyPassword(email string) string {
	return keyPasswordPrefix + strings.ToLower(email)
}

func (c *conn) getPassword(key string) (storage.Password, error) {
	var p storage.Password
	val, err := c.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			return p, storage.ErrNotFound
		}
		return p, err
	}
	err = decode(val, &p)
	return p, err
}

func (c *conn) listPasswords() ([]storage.Password, error) {
	keys, err := c.Keys(c.keyPassword("*")).Result()
	if err != nil {
		return nil, err
	}
	passwords := make([]storage.Password, len(keys))
	for i, k := range keys {
		passwords[i], err = c.getPassword(k)
		if err != nil {
			return nil, err
		}
	}
	return passwords, nil
}

func (c *conn) setPassword(p storage.Password) error {
	key := c.keyPassword(p.Email)
	val, err := encode(p)
	if err != nil {
		return err
	}
	return c.Set(key, val, 0).Err()
}

// Offline sessions

const keyOfflineSessionsPrefix = "dex:offline:sessions:"

func (c *conn) keyOfflineSessions(userID, connID string) string {
	return keyOfflineSessionsPrefix + userID + ":" + connID
}

func (c *conn) getOfflineSessions(key string) (storage.OfflineSessions, error) {
	var os storage.OfflineSessions
	val, err := c.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			return os, storage.ErrNotFound
		}
		return os, err
	}
	err = decode(val, &os)
	return os, err
}

func (c *conn) setOfflineSessions(os storage.OfflineSessions) error {
	key := c.keyOfflineSessions(os.UserID, os.ConnID)
	val, err := encode(os)
	if err != nil {
		return err
	}
	return c.Set(key, val, 0).Err()
}

// Connector

const keyConnectorPrefix = "dex:connector:"

func (c *conn) keyConnector(id string) string {
	return keyConnectorPrefix + id
}

func (c *conn) getConnector(key string) (storage.Connector, error) {
	var co storage.Connector
	val, err := c.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			return co, storage.ErrNotFound
		}
		return co, err
	}
	err = decode(val, &co)
	return co, err
}

func (c *conn) listConnectors() ([]storage.Connector, error) {
	keys, err := c.Keys(c.keyConnector("*")).Result()
	if err != nil {
		return nil, err
	}
	connectors := make([]storage.Connector, len(keys))
	for i, k := range keys {
		connectors[i], err = c.getConnector(k)
		if err != nil {
			return nil, err
		}
	}
	return connectors, nil
}

func (c *conn) setConnector(co storage.Connector) error {
	key := c.keyConnector(co.ID)
	val, err := encode(co)
	if err != nil {
		return err
	}
	return c.Set(key, val, 0).Err()
}

// Create

func (c *conn) CreateAuthRequest(a storage.AuthRequest) error {
	if c.Exists(c.keyAuthRequest(a.ID)).Val() == 1 {
		return storage.ErrAlreadyExists
	}
	return c.setAuthRequest(a)
}

func (c *conn) CreateAuthCode(a storage.AuthCode) error {
	if c.Exists(c.keyAuthCode(a.ID)).Val() == 1 {
		return storage.ErrAlreadyExists
	}
	return c.setAuthCode(a)
}

func (c *conn) CreateClient(cli storage.Client) error {
	if c.Exists(c.keyClient(cli.ID)).Val() == 1 {
		return storage.ErrAlreadyExists
	}
	return c.setClient(cli)
}

func (c *conn) CreateRefresh(r storage.RefreshToken) error {
	if c.Exists(c.keyRefresh(r.ID)).Val() == 1 {
		return storage.ErrAlreadyExists
	}
	return c.setRefresh(r)
}

func (c *conn) CreatePassword(p storage.Password) error {
	if c.Exists(c.keyPassword(p.Email)).Val() == 1 {
		return storage.ErrAlreadyExists
	}
	return c.setPassword(p)
}

func (c *conn) CreateOfflineSessions(os storage.OfflineSessions) error {
	if c.Exists(c.keyOfflineSessions(os.UserID, os.ConnID)).Val() == 1 {
		return storage.ErrAlreadyExists
	}
	return c.setOfflineSessions(os)
}

func (c *conn) CreateConnector(co storage.Connector) error {
	if c.Exists(c.keyConnector(co.ID)).Val() == 1 {
		return storage.ErrAlreadyExists
	}
	return c.setConnector(co)
}

// Get

func (c *conn) GetAuthRequest(id string) (storage.AuthRequest, error) {
	return c.getAuthRequest(c.keyAuthRequest(id))
}

func (c *conn) GetAuthCode(id string) (storage.AuthCode, error) {
	return c.getAuthCode(c.keyAuthCode(id))
}

func (c *conn) GetClient(id string) (storage.Client, error) {
	return c.getClient(c.keyClient(id))
}

func (c *conn) GetKeys() (storage.Keys, error) {
	return c.getKeys(keyKeys)
}

func (c *conn) GetRefresh(id string) (storage.RefreshToken, error) {
	return c.getRefresh(c.keyRefresh(id))
}

func (c *conn) GetPassword(email string) (storage.Password, error) {
	return c.getPassword(c.keyPassword(email))
}

func (c *conn) GetOfflineSessions(userID, connID string) (storage.OfflineSessions, error) {
	return c.getOfflineSessions(c.keyOfflineSessions(userID, connID))
}

func (c *conn) GetConnector(id string) (storage.Connector, error) {
	return c.getConnector(c.keyConnector(id))
}

// List

func (c *conn) ListClients() ([]storage.Client, error)             { return c.listClients() }
func (c *conn) ListRefreshTokens() ([]storage.RefreshToken, error) { return c.listRefresh() }
func (c *conn) ListPasswords() ([]storage.Password, error)         { return c.listPasswords() }
func (c *conn) ListConnectors() ([]storage.Connector, error)       { return c.listConnectors() }

// Delete

func (c *conn) DeleteAuthRequest(id string) error {
	if c.Exists(c.keyAuthRequest(id)).Val() == 0 {
		return storage.ErrNotFound
	}
	return c.Del(c.keyAuthRequest(id)).Err()
}

func (c *conn) DeleteAuthCode(id string) error {
	if c.Exists(c.keyAuthCode(id)).Val() == 0 {
		return storage.ErrNotFound
	}
	return c.Del(c.keyAuthCode(id)).Err()
}

func (c *conn) DeleteClient(id string) error {
	if c.Exists(c.keyClient(id)).Val() == 0 {
		return storage.ErrNotFound
	}
	return c.Del(c.keyClient(id)).Err()
}

func (c *conn) DeleteRefresh(id string) error {
	if c.Exists(c.keyRefresh(id)).Val() == 0 {
		return storage.ErrNotFound
	}
	return c.Del(c.keyRefresh(id)).Err()
}

func (c *conn) DeletePassword(email string) error {
	if c.Exists(c.keyPassword(email)).Val() == 0 {
		return storage.ErrNotFound
	}
	return c.Del(c.keyPassword(email)).Err()
}

func (c *conn) DeleteOfflineSessions(userID, connID string) error {
	if c.Exists(c.keyOfflineSessions(userID, connID)).Val() == 0 {
		return storage.ErrNotFound
	}
	return c.Del(c.keyOfflineSessions(userID, connID)).Err()
}

func (c *conn) DeleteConnector(id string) error {
	if c.Exists(c.keyConnector(id)).Val() == 0 {
		return storage.ErrNotFound
	}
	return c.Del(c.keyConnector(id)).Err()
}

// Update

func (c *conn) UpdateAuthRequest(id string, updater func(storage.AuthRequest) (storage.AuthRequest, error)) error {
	a, err := c.getAuthRequest(c.keyAuthRequest(id))
	if err != nil {
		return err
	}
	a, err = updater(a)
	if err != nil {
		return err
	}
	return c.setAuthRequest(a)
}

func (c *conn) UpdateClient(id string, updater func(storage.Client) (storage.Client, error)) error {
	cli, err := c.getClient(c.keyClient(id))
	if err != nil {
		return err
	}
	cli, err = updater(cli)
	if err != nil {
		return err
	}
	return c.setClient(cli)
}

func (c *conn) UpdateKeys(updater func(storage.Keys) (storage.Keys, error)) error {
	k, err := c.getKeys(keyKeys)
	if err != nil {
		if err != storage.ErrNotFound {
			return err
		}
	}
	k, err = updater(k)
	if err != nil {
		return err
	}
	return c.setKeys(k)
}

func (c *conn) UpdateRefreshToken(id string, updater func(storage.RefreshToken) (storage.RefreshToken, error)) error {
	r, err := c.getRefresh(c.keyRefresh(id))
	if err != nil {
		return err
	}
	r, err = updater(r)
	if err != nil {
		return err
	}
	return c.setRefresh(r)
}

func (c *conn) UpdatePassword(email string, updater func(storage.Password) (storage.Password, error)) error {
	p, err := c.getPassword(c.keyPassword(email))
	if err != nil {
		return err
	}
	p, err = updater(p)
	if err != nil {
		return err
	}
	return c.setPassword(p)
}

func (c *conn) UpdateOfflineSessions(userID, connID string,
	updater func(storage.OfflineSessions) (storage.OfflineSessions, error)) error {
	os, err := c.getOfflineSessions(c.keyOfflineSessions(userID, connID))
	if err != nil {
		return err
	}
	os, err = updater(os)
	if err != nil {
		return err
	}
	return c.setOfflineSessions(os)
}

func (c *conn) UpdateConnector(id string, updater func(storage.Connector) (storage.Connector, error)) error {
	co, err := c.getConnector(c.keyConnector(id))
	if err != nil {
		return err
	}
	co, err = updater(co)
	if err != nil {
		return err
	}
	return c.setConnector(co)
}

// Garbage collection

func (c *conn) GarbageCollect(now time.Time) (result storage.GCResult, err error) {
	var keys []string

	requests, err := c.listAuthRequests()
	if err != nil {
		return result, err
	}
	for _, req := range requests {
		if req.Expiry.Sub(now) < 0 {
			result.AuthRequests++
			keys = append(keys, c.keyAuthRequest(req.ID))
		}
	}

	codes, err := c.listAuthCodes()
	for _, code := range codes {
		if code.Expiry.Sub(now) < 0 {
			result.AuthCodes++
			keys = append(keys, c.keyAuthCode(code.ID))
		}
	}

	if len(keys) > 0 {
		err = c.Del(keys...).Err()
	}
	return result, err
}
