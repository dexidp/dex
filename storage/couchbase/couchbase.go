package couchbase

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/couchbase/gocb.v1"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/storage"
)

const (
	clientKey         = "dex-client"
	authCodeKey       = "dex-authcode"
	refreshTokenKey   = "dex-refreshtoken"
	authRequestKey    = "dex-authreq"
	passwordKey       = "dex-password"
	offlineSessionKey = "dex-offlinesession"
	connectorKey      = "dex-connector"
	keysName          = "dex-openid-connect-keys"
)

// conn is the main database connection.
type conn struct {
	db     *gocb.Bucket
	logger log.Logger
}

func (c *conn) Close() error {
	return c.db.Close()
}

type KeyDocument struct {
	Key string `json:"id"`
}

func keyID(prefix string, id string) string { return prefix + "-" + id }

func keyEmail(prefix, email string) string { return prefix + "-" + strings.ToLower(email) }

func keySession(prefix, userID, connID string) string {
	return prefix + "-" + strings.ToLower(userID+"-"+connID)
}

func alreadyExistsCheck(err error) bool {
	if strings.Contains(err.Error(), "key already exists") {
		return true
	}
	return false
}

func (c *conn) getDocument(key string, value interface{}) error {
	_, err := c.db.Get(key, &value)
	if err != nil {
		if strings.Contains(err.Error(), "key not found") {
			return storage.ErrNotFound
		}
		return err
	}
	return nil
}

func (c *conn) InsertDocument(key_document string, document interface{}, ttl uint32) error {
	_, err := c.db.Insert(key_document, document, ttl)
	if err != nil {
		return err
	}
	if ttl > 0 {
		c.db.Touch(key_document, 0, ttl)
	}
	return nil
}

func (c *conn) UpsertDocument(key_document string, document interface{}, ttl uint32) error {
	_, err := c.db.Upsert(key_document, document, ttl)
	if err != nil {
		return err
	}
	if ttl > 0 {
		c.db.Touch(key_document, 0, ttl)
	}
	return nil
}

func (c *conn) ListIdsDocumentsByType(dex_type string) ([]string, error) {
	query := fmt.Sprintf("SELECT meta().`id` FROM %s "+
		"WHERE meta().`id` LIKE 'dex-%s%s'", BucketName, dex_type, "%")
	myQuery := gocb.NewN1qlQuery(query)
	rows, err := c.db.ExecuteN1qlQuery(myQuery, nil)
	if err != nil {
		return nil, err
	}
	var list_ids []string

	var row KeyDocument
	for rows.Next(&row) {
		if row.Key != "" {
			list_ids = append(list_ids, row.Key)
		}
	}
	if err = rows.Close(); err != nil {
		c.logger.Infof("Couldn't get all the rows: %v\n", err)
	}
	return list_ids, nil
}

func (c *conn) CreateClient(cli storage.Client) error {
	key_cb := keyID(clientKey, cli.ID)
	err := c.InsertDocument(key_cb, cli, 0)

	if err != nil {
		if alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert Client: %v", err)
	}

	return nil
}

func (c *conn) UpdateClient(id string, updater func(old storage.Client) (storage.Client, error)) error {
	key_cb := keyID(clientKey, id)
	var clicurrent storage.Client
	err := c.getDocument(key_cb, &clicurrent)
	if err != nil {
		return err
	}
	nc, err := updater(clicurrent)
	if err != nil {
		return err
	}

	err = c.UpsertDocument(key_cb, nc, 0)
	if err != nil {
		return fmt.Errorf("update client: %v", err)
	}
	return nil

}

func (c *conn) GetClient(id string) (cli storage.Client, err error) {
	key_cb := keyID(clientKey, id)
	err = c.getDocument(key_cb, &cli)
	return cli, err
}

func (c *conn) ListClients() ([]storage.Client, error) {
	rows, err := c.ListIdsDocumentsByType(clientKey)

	if err != nil {
		return nil, err
	}
	var clients []storage.Client

	var cli storage.Client
	for _, v := range rows {
		if v != "" {
			err = c.getDocument(v, &cli)
			if err == nil {
				clients = append(clients, cli)
			}
		}
	}
	return clients, nil
}

func (c *conn) CreateAuthRequest(a storage.AuthRequest) error {
	var ttl uint32 = uint32(int(time.Until(a.Expiry).Seconds()))
	key_cb := keyID(authRequestKey, a.ID)
	var auth_cb AuthRequest = fromStorageAuthRequest(a)
	err := c.InsertDocument(key_cb, auth_cb, ttl)
	if err != nil {
		if alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert auth request: %v", err)
	}
	return nil
}

func (c *conn) UpdateAuthRequest(id string, updater func(a storage.AuthRequest) (storage.AuthRequest, error)) error {
	key_cb := keyID(authRequestKey, id)
	var authcurrent AuthRequest
	err := c.getDocument(key_cb, &authcurrent)
	ar := toStorageAuthRequest(authcurrent)
	ttl := uint32(int(time.Until(ar.Expiry).Seconds()))

	if err != nil {
		return err
	}
	nauth, err := updater(ar)
	if err != nil {
		return err
	}
	err = c.UpsertDocument(key_cb, fromStorageAuthRequest(nauth), ttl)
	if err != nil {
		return fmt.Errorf("update auth request: %v", err)
	}

	return nil
}

func (c *conn) GetAuthRequest(id string) (a storage.AuthRequest, err error) {
	key_cb := keyID(authRequestKey, id)
	var req AuthRequest
	err = c.getDocument(key_cb, &req)
	if err == nil {
		a = toStorageAuthRequest(req)
	}
	return a, err
}

func (c *conn) CreateAuthCode(a storage.AuthCode) error {
	var ttl uint32 = uint32(int(time.Until(a.Expiry).Seconds()))
	key_cb := keyID(authCodeKey, a.ID)
	var auth_cb AuthCode = fromStorageAuthCode(a)
	err := c.InsertDocument(key_cb, auth_cb, ttl)
	if err != nil {
		if alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert auth code: %v", err)
	}
	return nil
}

func (c *conn) GetAuthCode(id string) (a storage.AuthCode, err error) {
	key_cb := keyID(authCodeKey, id)
	var auth_code AuthCode
	err = c.getDocument(key_cb, &auth_code)
	if err == nil {
		a = toStorageAuthCode(auth_code)
	}
	return a, err
}

func (c *conn) CreateRefresh(r storage.RefreshToken) error {
	var refresh_token_cb RefreshToken = fromStorageRefreshToken(r)
	err := c.InsertDocument(keyID(refreshTokenKey, r.ID), refresh_token_cb, 0)
	if err != nil {
		if alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert refresh token: %v", err)
	}
	return nil
}

func (c *conn) UpdateRefreshToken(id string, updater func(old storage.RefreshToken) (storage.RefreshToken, error)) error {
	key_cb := keyID(refreshTokenKey, id)
	refcurrent, err := c.GetRefresh(id)
	if err != nil {
		return err
	}
	nauth, err := updater(refcurrent)
	if err != nil {
		return err
	}
	err = c.UpsertDocument(key_cb, fromStorageRefreshToken(nauth), 0)
	if err != nil {
		return fmt.Errorf("update refresh token: %v", err)
	}
	return nil
}

func (c *conn) GetRefresh(id string) (r storage.RefreshToken, err error) {
	var token RefreshToken
	err = c.getDocument(keyID(refreshTokenKey, id), &token)
	if err == nil {
		r = toStorageRefreshToken(token)
	}
	return r, err
}

func (c *conn) ListRefreshTokens() ([]storage.RefreshToken, error) {
	rows, err := c.ListIdsDocumentsByType(refreshTokenKey)

	if err != nil {
		return nil, err
	}

	var tokens []storage.RefreshToken

	var row RefreshToken
	for _, v := range rows {
		if v != "" {
			err = c.getDocument(v, &row)
			if err == nil {
				tokens = append(tokens, toStorageRefreshToken(row))
			}
		}
	}
	return tokens, nil
}

func (c *conn) CreatePassword(p storage.Password) error {
	p.Email = strings.ToLower(p.Email)
	key_cb := keyEmail(passwordKey, p.Email)
	err := c.InsertDocument(key_cb, p, 0)
	if err != nil {
		if alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert password: %v", err)
	}
	return nil
}

func (c *conn) UpdatePassword(email string, updater func(p storage.Password) (storage.Password, error)) error {
	key_cb := keyEmail(passwordKey, email)
	var refcurrent storage.Password
	err := c.getDocument(key_cb, &refcurrent)
	if err != nil {
		return err
	}
	nauth, err := updater(refcurrent)
	if err != nil {
		return err
	}
	err = c.UpsertDocument(key_cb, nauth, 0)
	if err != nil {
		return fmt.Errorf("update password: %v", err)
	}
	return nil
}

func (c *conn) GetPassword(email string) (p storage.Password, err error) {
	err = c.getDocument(keyEmail(passwordKey, email), &p)
	return p, err
}

func (c *conn) ListPasswords() ([]storage.Password, error) {
	rows, err := c.ListIdsDocumentsByType(passwordKey)

	if err != nil {
		return nil, err
	}
	var passwords []storage.Password

	var row storage.Password
	for _, v := range rows {
		if v != "" {
			err = c.getDocument(v, &row)
			if err == nil {
				passwords = append(passwords, row)
			}
		}
	}
	return passwords, nil
}

func (c *conn) CreateOfflineSessions(s storage.OfflineSessions) error {
	key_cb := keySession(offlineSessionKey, s.UserID, s.ConnID)
	var session_cb OfflineSessions = fromStorageOfflineSessions(s)
	err := c.InsertDocument(key_cb, session_cb, 0)

	if err != nil {
		if alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert offline sesssion: %v", err)
	}
	return nil
}

func (c *conn) UpdateOfflineSessions(userID string, connID string, updater func(s storage.OfflineSessions) (storage.OfflineSessions, error)) error {
	key_cb := keySession(offlineSessionKey, userID, connID)

	var refcurrent OfflineSessions
	err := c.getDocument(key_cb, &refcurrent)
	if err != nil {
		return err
	}
	nauth, err := updater(toStorageOfflineSessions(refcurrent))
	if err != nil {
		return err
	}
	err = c.UpsertDocument(key_cb, fromStorageOfflineSessions(nauth), 0)
	if err != nil {
		return fmt.Errorf("update password: %v", err)
	}
	return nil
}

func (c *conn) GetOfflineSessions(userID string, connID string) (s storage.OfflineSessions, err error) {
	key_cb := keySession(offlineSessionKey, userID, connID)
	var os OfflineSessions
	err = c.getDocument(key_cb, &os)
	if err == nil {
		s = toStorageOfflineSessions(os)
	}
	return s, err
}

func (c *conn) CreateConnector(connector storage.Connector) error {
	key_cb := keyID(connectorKey, connector.ID)
	err := c.InsertDocument(key_cb, connector, 0)
	if err != nil {
		if alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert connector: %v", err)
	}
	return nil
}

func (c *conn) UpdateConnector(id string, updater func(s storage.Connector) (storage.Connector, error)) error {
	key_cb := keyID(connectorKey, id)
	var refcurrent storage.Connector
	err := c.getDocument(key_cb, &refcurrent)
	if err != nil {
		return err
	}
	nauth, err := updater(refcurrent)
	if err != nil {
		return err
	}
	err = c.UpsertDocument(key_cb, nauth, 0)
	if err != nil {
		return fmt.Errorf("update password: %v", err)
	}
	return nil
}

func (c *conn) GetConnector(id string) (connector storage.Connector, err error) {
	err = c.getDocument(keyID(connectorKey, id), &connector)
	return connector, err
}

func (c *conn) ListConnectors() ([]storage.Connector, error) {
	rows, err := c.ListIdsDocumentsByType(connectorKey)

	if err != nil {
		return nil, err
	}

	var connectors []storage.Connector

	var row storage.Connector
	for _, v := range rows {
		if v != "" {
			err = c.getDocument(v, &row)
			if err == nil {
				connectors = append(connectors, row)
			}
		}
	}
	return connectors, nil
}

func (c *conn) GetKeys() (keys storage.Keys, err error) {
	var key_cb Keys
	err = c.getDocument(keysName, &key_cb)
	if err == nil {
		keys = toStorageKeys(key_cb)
	}
	return keys, err
}

func (c *conn) UpdateKeys(updater func(old storage.Keys) (storage.Keys, error)) error {
	key_cb := keysName
	var current Keys
	err := c.getDocument(key_cb, &current)
	if err != nil {
		if err != storage.ErrNotFound {
			return fmt.Errorf("get keys from updatekeys: %v", err)
		}
		current = Keys{}
	}
	nc, err := updater(toStorageKeys(current))
	if err != nil {
		return err
	}
	err = c.UpsertDocument(key_cb, fromStorageKeys(nc), 0)
	if err != nil {
		return fmt.Errorf("update keys: %v", err)
	}

	return nil
}

func (c *conn) delete(id string) error {
	_, err := c.db.Remove(id, 0)
	if err != nil {
		return fmt.Errorf("delete %v", id)
	}
	return nil
}

func (c *conn) DeleteAuthRequest(id string) error { return c.delete(keyID(authRequestKey, id)) }
func (c *conn) DeleteAuthCode(id string) error    { return c.delete(keyID(authCodeKey, id)) }
func (c *conn) DeleteClient(id string) error      { return c.delete(keyID(clientKey, id)) }
func (c *conn) DeleteRefresh(id string) error     { return c.delete(keyID(refreshTokenKey, id)) }
func (c *conn) DeletePassword(email string) error {
	return c.delete(keyEmail(passwordKey, strings.ToLower(email)))
}
func (c *conn) DeleteConnector(id string) error { return c.delete(keyID(connectorKey, id)) }

func (c *conn) DeleteOfflineSessions(userID string, connID string) error {
	return c.delete(keySession(offlineSessionKey, userID, connID))
}

func (c *conn) GarbageCollect(now time.Time) (result storage.GCResult, err error) {
	// nothing here, becuase an expiry time is set for the authrequest and authcode documents using touch
	return
}
