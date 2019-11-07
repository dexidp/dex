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
	clientKey            = "dex-client"
	authCodeKey          = "dex-authcode"
	refreshTokenKey      = "dex-refreshtoken"
	authRequestKey       = "dex-authreq"
	passwordKey          = "dex-password"
	offlineSessionKey    = "dex-offlinesession"
	connectorKey         = "dex-connector"
	keysName             = "dex-openid-connect-keys"
	disableConsistencyRP = false
)

// conn is the main database connection.
type conn struct {
	db     *gocb.Bucket
	logger log.Logger
}

func (c *conn) Close() error {
	return c.db.Close()
}

// KeyDocument structure for defining the couchbase id document name
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

func (c *conn) InsertDocument(keyDocument string, document interface{}, ttl uint32) error {
	_, err := c.db.Insert(keyDocument, document, ttl)
	if err != nil {
		return err
	}
	if ttl > 0 {
		c.db.Touch(keyDocument, 0, ttl)
	}
	return nil
}

func (c *conn) UpsertDocument(keyDocument string, document interface{}, ttl uint32) error {
	_, err := c.db.Upsert(keyDocument, document, ttl)
	if err != nil {
		return err
	}
	if ttl > 0 {
		c.db.Touch(keyDocument, 0, ttl)
	}
	return nil
}

func (c *conn) ListIdsDocumentsByType(dexType string) ([]string, error) {
	query := fmt.Sprintf("SELECT meta().`id` FROM %s "+
		"WHERE meta().`id` LIKE '%s%s'", BucketName, dexType, "%")

	// create N1qlQuery using RequestPlus consistency option
	CbN1qlQuery := gocb.NewN1qlQuery(query).Consistency(gocb.RequestPlus)

	if disableConsistencyRP {
		CbN1qlQuery = gocb.NewN1qlQuery(query)
	}

	rows, err := c.db.ExecuteN1qlQuery(CbN1qlQuery, nil)
	if err != nil {
		return nil, err
	}
	var listIds []string

	var row KeyDocument
	for rows.Next(&row) {
		if row.Key != "" {
			listIds = append(listIds, row.Key)
		}
	}
	if err = rows.Close(); err != nil {
		c.logger.Infof("Couldn't get all the rows: %v\n", err)
	}
	return listIds, nil
}

func (c *conn) CreateClient(cli storage.Client) error {
	keyCb := keyID(clientKey, cli.ID)
	err := c.InsertDocument(keyCb, cli, 0)

	if err != nil {
		if alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert Client: %v", err)
	}

	return nil
}

func (c *conn) UpdateClient(id string, updater func(old storage.Client) (storage.Client, error)) error {
	keyCb := keyID(clientKey, id)
	var cliCurrent storage.Client
	err := c.getDocument(keyCb, &cliCurrent)
	if err != nil {
		return err
	}
	nc, err := updater(cliCurrent)
	if err != nil {
		return err
	}

	err = c.UpsertDocument(keyCb, nc, 0)
	if err != nil {
		return fmt.Errorf("update client: %v", err)
	}
	return nil

}

func (c *conn) GetClient(id string) (cli storage.Client, err error) {
	keyCb := keyID(clientKey, id)
	err = c.getDocument(keyCb, &cli)
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
	keyCb := keyID(authRequestKey, a.ID)
	var authCb AuthRequest = fromStorageAuthRequest(a)
	err := c.InsertDocument(keyCb, authCb, ttl)
	if err != nil {
		if alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert auth request: %v", err)
	}
	return nil
}

func (c *conn) UpdateAuthRequest(id string, updater func(a storage.AuthRequest) (storage.AuthRequest, error)) error {
	keyCb := keyID(authRequestKey, id)
	var authCurrent AuthRequest
	err := c.getDocument(keyCb, &authCurrent)
	ar := toStorageAuthRequest(authCurrent)
	ttl := uint32(int(time.Until(ar.Expiry).Seconds()))

	if err != nil {
		return err
	}
	newAuth, err := updater(ar)
	if err != nil {
		return err
	}
	err = c.UpsertDocument(keyCb, fromStorageAuthRequest(newAuth), ttl)
	if err != nil {
		return fmt.Errorf("update auth request: %v", err)
	}

	return nil
}

func (c *conn) GetAuthRequest(id string) (a storage.AuthRequest, err error) {
	keyCb := keyID(authRequestKey, id)
	var req AuthRequest
	err = c.getDocument(keyCb, &req)
	if err == nil {
		a = toStorageAuthRequest(req)
	}
	return a, err
}

func (c *conn) CreateAuthCode(a storage.AuthCode) error {
	var ttl uint32 = uint32(int(time.Until(a.Expiry).Seconds()))
	keyCb := keyID(authCodeKey, a.ID)
	var authCb AuthCode = fromStorageAuthCode(a)
	err := c.InsertDocument(keyCb, authCb, ttl)
	if err != nil {
		if alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert auth code: %v", err)
	}
	return nil
}

func (c *conn) GetAuthCode(id string) (a storage.AuthCode, err error) {
	keyCb := keyID(authCodeKey, id)
	var authCode AuthCode
	err = c.getDocument(keyCb, &authCode)
	if err == nil {
		a = toStorageAuthCode(authCode)
	}
	return a, err
}

func (c *conn) CreateRefresh(r storage.RefreshToken) error {
	var refreshTokenCb RefreshToken = fromStorageRefreshToken(r)
	err := c.InsertDocument(keyID(refreshTokenKey, r.ID), refreshTokenCb, 0)
	if err != nil {
		if alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert refresh token: %v", err)
	}
	return nil
}

func (c *conn) UpdateRefreshToken(id string, updater func(old storage.RefreshToken) (storage.RefreshToken, error)) error {
	keyCb := keyID(refreshTokenKey, id)
	refCurrent, err := c.GetRefresh(id)
	if err != nil {
		return err
	}
	newToken, err := updater(refCurrent)
	if err != nil {
		return err
	}
	err = c.UpsertDocument(keyCb, fromStorageRefreshToken(newToken), 0)
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
	keyCb := keyEmail(passwordKey, p.Email)
	err := c.InsertDocument(keyCb, p, 0)
	if err != nil {
		if alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert password: %v", err)
	}
	return nil
}

func (c *conn) UpdatePassword(email string, updater func(p storage.Password) (storage.Password, error)) error {
	keyCb := keyEmail(passwordKey, email)
	var refCurrent storage.Password
	err := c.getDocument(keyCb, &refCurrent)
	if err != nil {
		return err
	}
	newPass, err := updater(refCurrent)
	if err != nil {
		return err
	}
	err = c.UpsertDocument(keyCb, newPass, 0)
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
	keyCb := keySession(offlineSessionKey, s.UserID, s.ConnID)
	var sessionCb OfflineSessions = fromStorageOfflineSessions(s)
	err := c.InsertDocument(keyCb, sessionCb, 0)

	if err != nil {
		if alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert offline sesssion: %v", err)
	}
	return nil
}

func (c *conn) UpdateOfflineSessions(userID string, connID string, updater func(s storage.OfflineSessions) (storage.OfflineSessions, error)) error {
	keyCb := keySession(offlineSessionKey, userID, connID)

	var refCurrent OfflineSessions
	err := c.getDocument(keyCb, &refCurrent)
	if err != nil {
		return err
	}
	newSession, err := updater(toStorageOfflineSessions(refCurrent))
	if err != nil {
		return err
	}
	err = c.UpsertDocument(keyCb, fromStorageOfflineSessions(newSession), 0)
	if err != nil {
		return fmt.Errorf("update password: %v", err)
	}
	return nil
}

func (c *conn) GetOfflineSessions(userID string, connID string) (s storage.OfflineSessions, err error) {
	keyCb := keySession(offlineSessionKey, userID, connID)
	var os OfflineSessions
	err = c.getDocument(keyCb, &os)
	if err == nil {
		s = toStorageOfflineSessions(os)
	}
	return s, err
}

func (c *conn) CreateConnector(connector storage.Connector) error {
	keyCb := keyID(connectorKey, connector.ID)
	err := c.InsertDocument(keyCb, connector, 0)
	if err != nil {
		if alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert connector: %v", err)
	}
	return nil
}

func (c *conn) UpdateConnector(id string, updater func(s storage.Connector) (storage.Connector, error)) error {
	keyCb := keyID(connectorKey, id)
	var refCurrent storage.Connector
	err := c.getDocument(keyCb, &refCurrent)
	if err != nil {
		return err
	}
	newConnector, err := updater(refCurrent)
	if err != nil {
		return err
	}
	err = c.UpsertDocument(keyCb, newConnector, 0)
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
	var keyCb Keys
	err = c.getDocument(keysName, &keyCb)
	if err == nil {
		keys = toStorageKeys(keyCb)
	}
	return keys, err
}

func (c *conn) UpdateKeys(updater func(old storage.Keys) (storage.Keys, error)) error {
	keyCb := keysName
	var current Keys
	err := c.getDocument(keyCb, &current)
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
	err = c.UpsertDocument(keyCb, fromStorageKeys(nc), 0)
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
