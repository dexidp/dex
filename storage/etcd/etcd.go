package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"

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

	// defaultStorageTimeout will be applied to all storage's operations.
	defaultStorageTimeout = 5 * time.Second
)

type conn struct {
	db     *clientv3.Client
	logger log.Logger
}

func (c *conn) Close() error {
	return c.db.Close()
}

func (c *conn) GarbageCollect(now time.Time) (result storage.GCResult, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	authRequests, err := c.listAuthRequests(ctx)
	if err != nil {
		return result, err
	}

	var delErr error
	for _, authRequest := range authRequests {
		if now.After(authRequest.Expiry) {
			if err := c.deleteKey(ctx, keyID(authRequestPrefix, authRequest.ID)); err != nil {
				c.logger.Errorf("failed to delete auth request: %v", err)
				delErr = fmt.Errorf("failed to delete auth request: %v", err)
			}
			result.AuthRequests++
		}
	}
	if delErr != nil {
		return result, delErr
	}

	authCodes, err := c.listAuthCodes(ctx)
	if err != nil {
		return result, err
	}

	for _, authCode := range authCodes {
		if now.After(authCode.Expiry) {
			if err := c.deleteKey(ctx, keyID(authCodePrefix, authCode.ID)); err != nil {
				c.logger.Errorf("failed to delete auth code %v", err)
				delErr = fmt.Errorf("failed to delete auth code: %v", err)
			}
			result.AuthCodes++
		}
	}
	return result, delErr
}

func (c *conn) CreateAuthRequest(a storage.AuthRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnCreate(ctx, keyID(authRequestPrefix, a.ID), fromStorageAuthRequest(a))
}

func (c *conn) GetAuthRequest(id string) (a storage.AuthRequest, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	var req AuthRequest
	if err = c.getKey(ctx, keyID(authRequestPrefix, id), &req); err != nil {
		return
	}
	return toStorageAuthRequest(req), nil
}

func (c *conn) UpdateAuthRequest(id string, updater func(a storage.AuthRequest) (storage.AuthRequest, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnUpdate(ctx, keyID(authRequestPrefix, id), func(currentValue []byte) ([]byte, error) {
		var current AuthRequest
		if len(currentValue) > 0 {
			if err := json.Unmarshal(currentValue, &current); err != nil {
				return nil, err
			}
		}
		updated, err := updater(toStorageAuthRequest(current))
		if err != nil {
			return nil, err
		}
		return json.Marshal(fromStorageAuthRequest(updated))
	})
}

func (c *conn) DeleteAuthRequest(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.deleteKey(ctx, keyID(authRequestPrefix, id))
}

func (c *conn) CreateAuthCode(a storage.AuthCode) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnCreate(ctx, keyID(authCodePrefix, a.ID), fromStorageAuthCode(a))
}

func (c *conn) GetAuthCode(id string) (a storage.AuthCode, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err = c.getKey(ctx, keyID(authCodePrefix, id), &a)
	return a, err
}

func (c *conn) DeleteAuthCode(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.deleteKey(ctx, keyID(authCodePrefix, id))
}

func (c *conn) CreateRefresh(r storage.RefreshToken) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnCreate(ctx, keyID(refreshTokenPrefix, r.ID), fromStorageRefreshToken(r))
}

func (c *conn) GetRefresh(id string) (r storage.RefreshToken, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	var token RefreshToken
	if err = c.getKey(ctx, keyID(refreshTokenPrefix, id), &token); err != nil {
		return
	}
	return toStorageRefreshToken(token), nil
}

func (c *conn) UpdateRefreshToken(id string, updater func(old storage.RefreshToken) (storage.RefreshToken, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnUpdate(ctx, keyID(refreshTokenPrefix, id), func(currentValue []byte) ([]byte, error) {
		var current RefreshToken
		if len(currentValue) > 0 {
			if err := json.Unmarshal([]byte(currentValue), &current); err != nil {
				return nil, err
			}
		}
		updated, err := updater(toStorageRefreshToken(current))
		if err != nil {
			return nil, err
		}
		return json.Marshal(fromStorageRefreshToken(updated))
	})
}

func (c *conn) DeleteRefresh(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.deleteKey(ctx, keyID(refreshTokenPrefix, id))
}

func (c *conn) ListRefreshTokens() (tokens []storage.RefreshToken, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	res, err := c.db.Get(ctx, refreshTokenPrefix, clientv3.WithPrefix())
	if err != nil {
		return tokens, err
	}
	for _, v := range res.Kvs {
		var token RefreshToken
		if err = json.Unmarshal(v.Value, &token); err != nil {
			return tokens, err
		}
		tokens = append(tokens, toStorageRefreshToken(token))
	}
	return tokens, nil
}

func (c *conn) CreateClient(cli storage.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnCreate(ctx, keyID(clientPrefix, cli.ID), cli)
}

func (c *conn) GetClient(id string) (cli storage.Client, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err = c.getKey(ctx, keyID(clientPrefix, id), &cli)
	return cli, err
}

func (c *conn) UpdateClient(id string, updater func(old storage.Client) (storage.Client, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnUpdate(ctx, keyID(clientPrefix, id), func(currentValue []byte) ([]byte, error) {
		var current storage.Client
		if len(currentValue) > 0 {
			if err := json.Unmarshal(currentValue, &current); err != nil {
				return nil, err
			}
		}
		updated, err := updater(current)
		if err != nil {
			return nil, err
		}
		return json.Marshal(updated)
	})
}

func (c *conn) DeleteClient(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.deleteKey(ctx, keyID(clientPrefix, id))
}

func (c *conn) ListClients() (clients []storage.Client, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	res, err := c.db.Get(ctx, clientPrefix, clientv3.WithPrefix())
	if err != nil {
		return clients, err
	}
	for _, v := range res.Kvs {
		var cli storage.Client
		if err = json.Unmarshal(v.Value, &cli); err != nil {
			return clients, err
		}
		clients = append(clients, cli)
	}
	return clients, nil
}

func (c *conn) CreatePassword(p storage.Password) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnCreate(ctx, passwordPrefix+strings.ToLower(p.Email), p)
}

func (c *conn) GetPassword(email string) (p storage.Password, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err = c.getKey(ctx, keyEmail(passwordPrefix, email), &p)
	return p, err
}

func (c *conn) UpdatePassword(email string, updater func(p storage.Password) (storage.Password, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnUpdate(ctx, keyEmail(passwordPrefix, email), func(currentValue []byte) ([]byte, error) {
		var current storage.Password
		if len(currentValue) > 0 {
			if err := json.Unmarshal(currentValue, &current); err != nil {
				return nil, err
			}
		}
		updated, err := updater(current)
		if err != nil {
			return nil, err
		}
		return json.Marshal(updated)
	})
}

func (c *conn) DeletePassword(email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.deleteKey(ctx, keyEmail(passwordPrefix, email))
}

func (c *conn) ListPasswords() (passwords []storage.Password, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	res, err := c.db.Get(ctx, passwordPrefix, clientv3.WithPrefix())
	if err != nil {
		return passwords, err
	}
	for _, v := range res.Kvs {
		var p storage.Password
		if err = json.Unmarshal(v.Value, &p); err != nil {
			return passwords, err
		}
		passwords = append(passwords, p)
	}
	return passwords, nil
}

func (c *conn) CreateOfflineSessions(s storage.OfflineSessions) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnCreate(ctx, keySession(offlineSessionPrefix, s.UserID, s.ConnID), fromStorageOfflineSessions(s))
}

func (c *conn) UpdateOfflineSessions(userID string, connID string, updater func(s storage.OfflineSessions) (storage.OfflineSessions, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnUpdate(ctx, keySession(offlineSessionPrefix, userID, connID), func(currentValue []byte) ([]byte, error) {
		var current OfflineSessions
		if len(currentValue) > 0 {
			if err := json.Unmarshal(currentValue, &current); err != nil {
				return nil, err
			}
		}
		updated, err := updater(toStorageOfflineSessions(current))
		if err != nil {
			return nil, err
		}
		return json.Marshal(fromStorageOfflineSessions(updated))
	})
}

func (c *conn) GetOfflineSessions(userID string, connID string) (s storage.OfflineSessions, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	var os OfflineSessions
	if err = c.getKey(ctx, keySession(offlineSessionPrefix, userID, connID), &os); err != nil {
		return
	}
	return toStorageOfflineSessions(os), nil
}

func (c *conn) DeleteOfflineSessions(userID string, connID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.deleteKey(ctx, keySession(offlineSessionPrefix, userID, connID))
}

func (c *conn) CreateConnector(connector storage.Connector) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnCreate(ctx, keyID(connectorPrefix, connector.ID), connector)
}

func (c *conn) GetConnector(id string) (conn storage.Connector, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err = c.getKey(ctx, keyID(connectorPrefix, id), &conn)
	return conn, err
}

func (c *conn) UpdateConnector(id string, updater func(s storage.Connector) (storage.Connector, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnUpdate(ctx, keyID(connectorPrefix, id), func(currentValue []byte) ([]byte, error) {
		var current storage.Connector
		if len(currentValue) > 0 {
			if err := json.Unmarshal(currentValue, &current); err != nil {
				return nil, err
			}
		}
		updated, err := updater(current)
		if err != nil {
			return nil, err
		}
		return json.Marshal(updated)
	})
}

func (c *conn) DeleteConnector(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.deleteKey(ctx, keyID(connectorPrefix, id))
}

func (c *conn) ListConnectors() (connectors []storage.Connector, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	res, err := c.db.Get(ctx, connectorPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	for _, v := range res.Kvs {
		var c storage.Connector
		if err = json.Unmarshal(v.Value, &c); err != nil {
			return nil, err
		}
		connectors = append(connectors, c)
	}
	return connectors, nil
}

func (c *conn) GetKeys() (keys storage.Keys, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	res, err := c.db.Get(ctx, keysName)
	if err != nil {
		return keys, err
	}
	if res.Count > 0 && len(res.Kvs) > 0 {
		err = json.Unmarshal(res.Kvs[0].Value, &keys)
	}
	return keys, err
}

func (c *conn) UpdateKeys(updater func(old storage.Keys) (storage.Keys, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnUpdate(ctx, keysName, func(currentValue []byte) ([]byte, error) {
		var current storage.Keys
		if len(currentValue) > 0 {
			if err := json.Unmarshal(currentValue, &current); err != nil {
				return nil, err
			}
		}
		updated, err := updater(current)
		if err != nil {
			return nil, err
		}
		return json.Marshal(updated)
	})
}

func (c *conn) deleteKey(ctx context.Context, key string) error {
	res, err := c.db.Delete(ctx, key)
	if err != nil {
		return err
	}
	if res.Deleted == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (c *conn) getKey(ctx context.Context, key string, value interface{}) error {
	r, err := c.db.Get(ctx, key)
	if err != nil {
		return err
	}
	if r.Count == 0 {
		return storage.ErrNotFound
	}
	return json.Unmarshal(r.Kvs[0].Value, value)
}

func (c *conn) listAuthRequests(ctx context.Context) (reqs []AuthRequest, err error) {
	res, err := c.db.Get(ctx, authRequestPrefix, clientv3.WithPrefix())
	if err != nil {
		return reqs, err
	}
	for _, v := range res.Kvs {
		var r AuthRequest
		if err = json.Unmarshal(v.Value, &r); err != nil {
			return reqs, err
		}
		reqs = append(reqs, r)
	}
	return reqs, nil
}

func (c *conn) listAuthCodes(ctx context.Context) (codes []AuthCode, err error) {
	res, err := c.db.Get(ctx, authCodePrefix, clientv3.WithPrefix())
	if err != nil {
		return codes, err
	}
	for _, v := range res.Kvs {
		var c AuthCode
		if err = json.Unmarshal(v.Value, &c); err != nil {
			return codes, err
		}
		codes = append(codes, c)
	}
	return codes, nil
}

func (c *conn) txnCreate(ctx context.Context, key string, value interface{}) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	txn := c.db.Txn(ctx)
	res, err := txn.
		If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
		Then(clientv3.OpPut(key, string(b))).
		Commit()
	if err != nil {
		return err
	}
	if !res.Succeeded {
		return storage.ErrAlreadyExists
	}
	return nil
}

func (c *conn) txnUpdate(ctx context.Context, key string, update func(current []byte) ([]byte, error)) error {
	getResp, err := c.db.Get(ctx, key)
	if err != nil {
		return err
	}
	var currentValue []byte
	var modRev int64
	if len(getResp.Kvs) > 0 {
		currentValue = getResp.Kvs[0].Value
		modRev = getResp.Kvs[0].ModRevision
	}

	updatedValue, err := update(currentValue)
	if err != nil {
		return err
	}

	txn := c.db.Txn(ctx)
	updateResp, err := txn.
		If(clientv3.Compare(clientv3.ModRevision(key), "=", modRev)).
		Then(clientv3.OpPut(key, string(updatedValue))).
		Commit()
	if err != nil {
		return err
	}
	if !updateResp.Succeeded {
		return fmt.Errorf("failed to update key=%q: concurrent conflicting update happened", key)
	}
	return nil
}

func keyID(prefix, id string) string       { return prefix + id }
func keyEmail(prefix, email string) string { return prefix + strings.ToLower(email) }
func keySession(prefix, userID, connID string) string {
	return prefix + strings.ToLower(userID+"|"+connID)
}
