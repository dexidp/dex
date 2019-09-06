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
	err := c.db.Close()
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) GarbageCollect(now time.Time) (result storage.GCResult, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	authRequests, err := c.listAuthRequests(ctx)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return result, err
		}
		return result, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
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
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := delErr.(storage.Error); ok {
			return result, delErr
		}
		return result, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: delErr.Error()}
	}

	authCodes, err := c.listAuthCodes(ctx)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return result, err
		}
		return result, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
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
	if delErr != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := delErr.(storage.Error); ok {
			return result, delErr
		}
		return result, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: delErr.Error()}
	}
	return result, nil
}

func (c *conn) CreateAuthRequest(a storage.AuthRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	if err := c.txnCreate(ctx, keyID(authRequestPrefix, a.ID), fromStorageAuthRequest(a)); err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) GetAuthRequest(id string) (a storage.AuthRequest, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	var req AuthRequest
	if err = c.getKey(ctx, keyID(authRequestPrefix, id), &req); err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return
		}
		err = storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
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
				return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
			}
		}
		updated, err := updater(toStorageAuthRequest(current))
		if err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return nil, err
			}
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		output, err := json.Marshal(fromStorageAuthRequest(updated))
		if err != nil {
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		return output, nil
	})
}

func (c *conn) DeleteAuthRequest(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err := c.deleteKey(ctx, keyID(authRequestPrefix, id))
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) CreateAuthCode(a storage.AuthCode) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err := c.txnCreate(ctx, keyID(authCodePrefix, a.ID), fromStorageAuthCode(a))
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) GetAuthCode(id string) (a storage.AuthCode, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err = c.getKey(ctx, keyID(authCodePrefix, id), &a)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return a, err
		}
		return a, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return a, nil
}

func (c *conn) DeleteAuthCode(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err := c.deleteKey(ctx, keyID(authCodePrefix, id))
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) CreateRefresh(r storage.RefreshToken) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err := c.txnCreate(ctx, keyID(refreshTokenPrefix, r.ID), fromStorageRefreshToken(r))
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) GetRefresh(id string) (r storage.RefreshToken, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	var token RefreshToken
	if err = c.getKey(ctx, keyID(refreshTokenPrefix, id), &token); err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return r, err
		}
		return r, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
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
				return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
			}
		}
		updated, err := updater(toStorageRefreshToken(current))
		if err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return nil, err
			}
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		output, err := json.Marshal(fromStorageRefreshToken(updated))
		if err != nil {
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		return output, nil
	})
}

func (c *conn) DeleteRefresh(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err := c.deleteKey(ctx, keyID(refreshTokenPrefix, id))
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) ListRefreshTokens() (tokens []storage.RefreshToken, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	res, err := c.db.Get(ctx, refreshTokenPrefix, clientv3.WithPrefix())
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return tokens, err
		}
		return tokens, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	for _, v := range res.Kvs {
		var token RefreshToken
		if err = json.Unmarshal(v.Value, &token); err != nil {
			return tokens, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		tokens = append(tokens, toStorageRefreshToken(token))
	}
	return tokens, nil
}

func (c *conn) CreateClient(cli storage.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err := c.txnCreate(ctx, keyID(clientPrefix, cli.ID), cli)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) GetClient(id string) (cli storage.Client, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err = c.getKey(ctx, keyID(clientPrefix, id), &cli)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return cli, err
		}
		return cli, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return cli, err
}

func (c *conn) UpdateClient(id string, updater func(old storage.Client) (storage.Client, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnUpdate(ctx, keyID(clientPrefix, id), func(currentValue []byte) ([]byte, error) {
		var current storage.Client
		if len(currentValue) > 0 {
			if err := json.Unmarshal(currentValue, &current); err != nil {
				return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
			}
		}
		updated, err := updater(current)
		if err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return nil, err
			}
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		output, err := json.Marshal(updated)
		if err != nil {
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		return output, nil
	})
}

func (c *conn) DeleteClient(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err := c.deleteKey(ctx, keyID(clientPrefix, id))
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) ListClients() (clients []storage.Client, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	res, err := c.db.Get(ctx, clientPrefix, clientv3.WithPrefix())
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return clients, err
		}
		return clients, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	for _, v := range res.Kvs {
		var cli storage.Client
		if err = json.Unmarshal(v.Value, &cli); err != nil {
			return clients, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		clients = append(clients, cli)
	}
	return clients, nil
}

func (c *conn) CreatePassword(p storage.Password) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err := c.txnCreate(ctx, passwordPrefix+strings.ToLower(p.Email), p)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) GetPassword(email string) (p storage.Password, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err = c.getKey(ctx, keyEmail(passwordPrefix, email), &p)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return p, err
		}
		return p, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return p, nil
}

func (c *conn) UpdatePassword(email string, updater func(p storage.Password) (storage.Password, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnUpdate(ctx, keyEmail(passwordPrefix, email), func(currentValue []byte) ([]byte, error) {
		var current storage.Password
		if len(currentValue) > 0 {
			if err := json.Unmarshal(currentValue, &current); err != nil {
				return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
			}
		}
		updated, err := updater(current)
		if err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return nil, err
			}
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		output, err := json.Marshal(updated)
		if err != nil {
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		return output, nil
	})
}

func (c *conn) DeletePassword(email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err := c.deleteKey(ctx, keyEmail(passwordPrefix, email))
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) ListPasswords() (passwords []storage.Password, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	res, err := c.db.Get(ctx, passwordPrefix, clientv3.WithPrefix())
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return passwords, err
		}
		return passwords, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	for _, v := range res.Kvs {
		var p storage.Password
		if err = json.Unmarshal(v.Value, &p); err != nil {
			return passwords, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		passwords = append(passwords, p)
	}
	return passwords, nil
}

func (c *conn) CreateOfflineSessions(s storage.OfflineSessions) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	err := c.txnCreate(ctx, keySession(offlineSessionPrefix, s.UserID, s.ConnID), fromStorageOfflineSessions(s))
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) UpdateOfflineSessions(userID string, connID string, updater func(s storage.OfflineSessions) (storage.OfflineSessions, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnUpdate(ctx, keySession(offlineSessionPrefix, userID, connID), func(currentValue []byte) ([]byte, error) {
		var current OfflineSessions
		if len(currentValue) > 0 {
			if err := json.Unmarshal(currentValue, &current); err != nil {
				return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
			}
		}
		updated, err := updater(toStorageOfflineSessions(current))
		if err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return nil, err
			}
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		output, err := json.Marshal(fromStorageOfflineSessions(updated))
		if err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return nil, err
			}
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		return output, nil
	})
}

func (c *conn) GetOfflineSessions(userID string, connID string) (s storage.OfflineSessions, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	var os OfflineSessions
	if err = c.getKey(ctx, keySession(offlineSessionPrefix, userID, connID), &os); err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return
		}
		return s, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return toStorageOfflineSessions(os), err
}

func (c *conn) DeleteOfflineSessions(userID string, connID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	if err := c.deleteKey(ctx, keySession(offlineSessionPrefix, userID, connID)); err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) CreateConnector(connector storage.Connector) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	if err := c.txnCreate(ctx, keyID(connectorPrefix, connector.ID), connector); err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) GetConnector(id string) (conn storage.Connector, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	if err = c.getKey(ctx, keyID(connectorPrefix, id), &conn); err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return
		}
		return conn, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return conn, nil
}

func (c *conn) UpdateConnector(id string, updater func(s storage.Connector) (storage.Connector, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnUpdate(ctx, keyID(connectorPrefix, id), func(currentValue []byte) ([]byte, error) {
		var current storage.Connector
		if len(currentValue) > 0 {
			if err := json.Unmarshal(currentValue, &current); err != nil {
				return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
			}
		}
		updated, err := updater(current)
		if err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return nil, err
			}
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		output, err := json.Marshal(updated)
		if err != nil {
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		return output, err
	})
}

func (c *conn) DeleteConnector(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	if err := c.deleteKey(ctx, keyID(connectorPrefix, id)); err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) ListConnectors() (connectors []storage.Connector, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	res, err := c.db.Get(ctx, connectorPrefix, clientv3.WithPrefix())
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return nil, err
		}
		return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	for _, v := range res.Kvs {
		var c storage.Connector
		if err = json.Unmarshal(v.Value, &c); err != nil {
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
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
		return keys, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	if res.Count > 0 && len(res.Kvs) > 0 {
		err = json.Unmarshal(res.Kvs[0].Value, &keys)
	}
	return keys, nil
}

func (c *conn) UpdateKeys(updater func(old storage.Keys) (storage.Keys, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultStorageTimeout)
	defer cancel()
	return c.txnUpdate(ctx, keysName, func(currentValue []byte) ([]byte, error) {
		var current storage.Keys
		if len(currentValue) > 0 {
			if err := json.Unmarshal(currentValue, &current); err != nil {
				return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
			}
		}
		updated, err := updater(current)
		if err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return nil, err
			}
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		output, err := json.Marshal(updated)
		if err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return nil, err
			}
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		return output, nil
	})
}

func (c *conn) deleteKey(ctx context.Context, key string) error {
	res, err := c.db.Delete(ctx, key)
	if err != nil {
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	if res.Deleted == 0 {
		return storage.Error{Code: storage.ErrNotFound}
	}
	return nil
}

func (c *conn) getKey(ctx context.Context, key string, value interface{}) error {
	r, err := c.db.Get(ctx, key)
	if err != nil {
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	if r.Count == 0 {
		return storage.Error{Code: storage.ErrNotFound}
	}
	err = json.Unmarshal(r.Kvs[0].Value, value)
	if err != nil {
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) listAuthRequests(ctx context.Context) (reqs []AuthRequest, err error) {
	res, err := c.db.Get(ctx, authRequestPrefix, clientv3.WithPrefix())
	if err != nil {
		return reqs, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	for _, v := range res.Kvs {
		var r AuthRequest
		if err = json.Unmarshal(v.Value, &r); err != nil {
			return reqs, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		reqs = append(reqs, r)
	}
	return reqs, nil
}

func (c *conn) listAuthCodes(ctx context.Context) (codes []AuthCode, err error) {
	res, err := c.db.Get(ctx, authCodePrefix, clientv3.WithPrefix())
	if err != nil {
		return codes, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	for _, v := range res.Kvs {
		var c AuthCode
		if err = json.Unmarshal(v.Value, &c); err != nil {
			return codes, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		codes = append(codes, c)
	}
	return codes, nil
}

func (c *conn) txnCreate(ctx context.Context, key string, value interface{}) error {
	b, err := json.Marshal(value)
	if err != nil {
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	txn := c.db.Txn(ctx)
	res, err := txn.
		If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
		Then(clientv3.OpPut(key, string(b))).
		Commit()
	if err != nil {
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	if !res.Succeeded {
		return storage.Error{Code: storage.ErrAlreadyExists}
	}
	return nil
}

func (c *conn) txnUpdate(ctx context.Context, key string, update func(current []byte) ([]byte, error)) error {
	getResp, err := c.db.Get(ctx, key)
	if err != nil {
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	var currentValue []byte
	var modRev int64
	if len(getResp.Kvs) > 0 {
		currentValue = getResp.Kvs[0].Value
		modRev = getResp.Kvs[0].ModRevision
	}

	updatedValue, err := update(currentValue)
	if err != nil {
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}

	txn := c.db.Txn(ctx)
	updateResp, err := txn.
		If(clientv3.Compare(clientv3.ModRevision(key), "=", modRev)).
		Then(clientv3.OpPut(key, string(updatedValue))).
		Commit()
	if err != nil {
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	if !updateResp.Succeeded {
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("failed to update key=%q: concurrent conflicting update happened", key)}
	}
	return nil
}

func keyID(prefix, id string) string       { return prefix + id }
func keyEmail(prefix, email string) string { return prefix + strings.ToLower(email) }
func keySession(prefix, userID, connID string) string {
	return prefix + strings.ToLower(userID+"|"+connID)
}
