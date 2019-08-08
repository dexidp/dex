package kubernetes

import (
	"fmt"
	"time"

	"github.com/dexidp/dex/storage"
)

// Close Storage Interface method

func (cli *client) Close() error {
	if cli.cancel != nil {
		cli.cancel()
	}
	return nil
}

// Create methods of Storage Interface

func (cli *client) CreateAuthRequest(a storage.AuthRequest) error {
	return cli.post(resourceAuthRequest, cli.fromStorageAuthRequest(a))
}

func (cli *client) CreateClient(c storage.Client) error {
	return cli.post(resourceClient, cli.fromStorageClient(c))
}

func (cli *client) CreateAuthCode(c storage.AuthCode) error {
	return cli.post(resourceAuthCode, cli.fromStorageAuthCode(c))
}

func (cli *client) CreateRefresh(r storage.RefreshToken) error {
	return cli.post(resourceRefreshToken, cli.fromStorageRefreshToken(r))
}

func (cli *client) CreatePassword(p storage.Password) error {
	return cli.post(resourcePassword, cli.fromStoragePassword(p))
}

func (cli *client) CreateOfflineSessions(o storage.OfflineSessions) error {
	return cli.post(resourceOfflineSessions, cli.fromStorageOfflineSessions(o))
}

func (cli *client) CreateConnector(c storage.Connector) error {
	return cli.post(resourceConnector, cli.fromStorageConnector(c))
}

// Get methods of storage Interface

func (cli *client) GetAuthRequest(id string) (storage.AuthRequest, error) {
	var req AuthRequest
	if err := cli.get(resourceAuthRequest, id, &req); err != nil {
		return storage.AuthRequest{}, err
	}
	return toStorageAuthRequest(req), nil
}

func (cli *client) GetAuthCode(id string) (storage.AuthCode, error) {
	var code AuthCode
	if err := cli.get(resourceAuthCode, id, &code); err != nil {
		return storage.AuthCode{}, err
	}
	return toStorageAuthCode(code), nil
}

func (cli *client) GetClient(id string) (storage.Client, error) {
	c, err := cli.getClient(id)
	if err != nil {
		return storage.Client{}, err
	}
	return toStorageClient(c), nil
}

func (cli *client) GetKeys() (storage.Keys, error) {
	var keys Keys
	if err := cli.get(resourceKeys, keysName, &keys); err != nil {
		return storage.Keys{}, err
	}
	return toStorageKeys(keys), nil
}

func (cli *client) GetRefresh(id string) (storage.RefreshToken, error) {
	r, err := cli.getRefreshToken(id)
	if err != nil {
		return storage.RefreshToken{}, err
	}
	return toStorageRefreshToken(r), nil
}

func (cli *client) GetPassword(email string) (storage.Password, error) {
	p, err := cli.getPassword(email)
	if err != nil {
		return storage.Password{}, err
	}
	return toStoragePassword(p), nil
}

func (cli *client) GetOfflineSessions(userID string, connID string) (storage.OfflineSessions, error) {
	o, err := cli.getOfflineSessions(userID, connID)
	if err != nil {
		return storage.OfflineSessions{}, err
	}
	return toStorageOfflineSessions(o), nil
}

func (cli *client) GetConnector(id string) (storage.Connector, error) {
	var c Connector
	if err := cli.get(resourceConnector, id, &c); err != nil {
		return storage.Connector{}, err
	}
	return toStorageConnector(c), nil
}

// List Methods of Storage Interface

func (cli *client) ListClients() ([]storage.Client, error) {
	return nil, storage.Error{Code: storage.ErrNotImplemented}
}

func (cli *client) ListRefreshTokens() ([]storage.RefreshToken, error) {
	return nil, storage.Error{Code: storage.ErrNotImplemented}
}

func (cli *client) ListPasswords() (passwords []storage.Password, err error) {
	var passwordList PasswordList
	if err = cli.list(resourcePassword, &passwordList); err != nil {
		return passwords, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("failed to list passwords: %s", err.Error())}
	}

	for _, password := range passwordList.Passwords {
		p := storage.Password{
			Email:    password.Email,
			Hash:     password.Hash,
			Username: password.Username,
			UserID:   password.UserID,
		}
		passwords = append(passwords, p)
	}

	return
}

func (cli *client) ListConnectors() (connectors []storage.Connector, err error) {
	var connectorList ConnectorList
	if err = cli.list(resourceConnector, &connectorList); err != nil {
		return connectors, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("failed to list connectors: %s", err.Error())}
	}

	connectors = make([]storage.Connector, len(connectorList.Connectors))
	for i, connector := range connectorList.Connectors {
		connectors[i] = toStorageConnector(connector)
	}

	return
}

// Delete methods of Storage Interface

func (cli *client) DeleteAuthRequest(id string) error {
	return cli.delete(resourceAuthRequest, id)
}

func (cli *client) DeleteAuthCode(code string) error {
	return cli.delete(resourceAuthCode, code)
}

func (cli *client) DeleteClient(id string) error {
	// Check for hash collision.
	c, err := cli.getClient(id)
	if err != nil {
		return err
	}
	return cli.delete(resourceClient, c.ObjectMeta.Name)
}

func (cli *client) DeleteRefresh(id string) error {
	return cli.delete(resourceRefreshToken, id)
}

func (cli *client) DeletePassword(email string) error {
	// Check for hash collision.
	p, err := cli.getPassword(email)
	if err != nil {
		return err
	}
	return cli.delete(resourcePassword, p.ObjectMeta.Name)
}

func (cli *client) DeleteOfflineSessions(userID string, connID string) error {
	// Check for hash collision.
	o, err := cli.getOfflineSessions(userID, connID)
	if err != nil {
		return err
	}
	return cli.delete(resourceOfflineSessions, o.ObjectMeta.Name)
}

func (cli *client) DeleteConnector(id string) error {
	return cli.delete(resourceConnector, id)
}

// Update methods of Storage Interface

func (cli *client) UpdateClient(id string, updater func(old storage.Client) (storage.Client, error)) error {
	c, err := cli.getClient(id)
	if err != nil {
		return err
	}

	updated, err := updater(toStorageClient(c))
	if err != nil {
		return err
	}
	updated.ID = c.ID

	newClient := cli.fromStorageClient(updated)
	newClient.ObjectMeta = c.ObjectMeta
	return cli.put(resourceClient, c.ObjectMeta.Name, newClient)
}

func (cli *client) UpdateKeys(updater func(old storage.Keys) (storage.Keys, error)) error {
	firstUpdate := false
	var keys Keys
	if err := cli.get(resourceKeys, keysName, &keys); err != nil {
		if !storage.IsErrorCode(err, storage.ErrNotFound) {
			return err
		}
		firstUpdate = true
	}
	var oldKeys storage.Keys
	if !firstUpdate {
		oldKeys = toStorageKeys(keys)
	}

	updated, err := updater(oldKeys)
	if err != nil {
		return err
	}
	newKeys := cli.fromStorageKeys(updated)
	if firstUpdate {
		return cli.post(resourceKeys, newKeys)
	}
	newKeys.ObjectMeta = keys.ObjectMeta
	return cli.put(resourceKeys, keysName, newKeys)
}

func (cli *client) UpdateAuthRequest(id string, updater func(a storage.AuthRequest) (storage.AuthRequest, error)) error {
	var req AuthRequest
	err := cli.get(resourceAuthRequest, id, &req)
	if err != nil {
		return err
	}

	updated, err := updater(toStorageAuthRequest(req))
	if err != nil {
		return err
	}

	newReq := cli.fromStorageAuthRequest(updated)
	newReq.ObjectMeta = req.ObjectMeta
	return cli.put(resourceAuthRequest, id, newReq)
}

func (cli *client) UpdateRefreshToken(id string, updater func(old storage.RefreshToken) (storage.RefreshToken, error)) error {
	r, err := cli.getRefreshToken(id)
	if err != nil {
		return err
	}
	updated, err := updater(toStorageRefreshToken(r))
	if err != nil {
		return err
	}
	updated.ID = id

	newToken := cli.fromStorageRefreshToken(updated)
	newToken.ObjectMeta = r.ObjectMeta
	return cli.put(resourceRefreshToken, r.ObjectMeta.Name, newToken)
}

func (cli *client) UpdatePassword(email string, updater func(old storage.Password) (storage.Password, error)) error {
	p, err := cli.getPassword(email)
	if err != nil {
		return err
	}

	updated, err := updater(toStoragePassword(p))
	if err != nil {
		return err
	}
	updated.Email = p.Email

	newPassword := cli.fromStoragePassword(updated)
	newPassword.ObjectMeta = p.ObjectMeta
	return cli.put(resourcePassword, p.ObjectMeta.Name, newPassword)
}

func (cli *client) UpdateOfflineSessions(userID string, connID string, updater func(old storage.OfflineSessions) (storage.OfflineSessions, error)) error {
	o, err := cli.getOfflineSessions(userID, connID)
	if err != nil {
		return err
	}

	updated, err := updater(toStorageOfflineSessions(o))
	if err != nil {
		return err
	}

	newOfflineSessions := cli.fromStorageOfflineSessions(updated)
	newOfflineSessions.ObjectMeta = o.ObjectMeta
	return cli.put(resourceOfflineSessions, o.ObjectMeta.Name, newOfflineSessions)
}

func (cli *client) UpdateConnector(id string, updater func(a storage.Connector) (storage.Connector, error)) error {
	var c Connector
	err := cli.get(resourceConnector, id, &c)
	if err != nil {
		return err
	}

	updated, err := updater(toStorageConnector(c))
	if err != nil {
		return err
	}

	newConn := cli.fromStorageConnector(updated)
	newConn.ObjectMeta = c.ObjectMeta
	return cli.put(resourceConnector, id, newConn)
}

// Garbage Collection Storage Interface methods

func (cli *client) GarbageCollect(now time.Time) (result storage.GCResult, err error) {
	var authRequests AuthRequestList
	if err := cli.list(resourceAuthRequest, &authRequests); err != nil {
		return result, fmt.Errorf("failed to list auth requests: %v", err)
	}

	var delErr error
	for _, authRequest := range authRequests.AuthRequests {
		if now.After(authRequest.Expiry) {
			if err := cli.delete(resourceAuthRequest, authRequest.ObjectMeta.Name); err != nil {
				cli.logger.Errorf("failed to delete auth request: %v", err)
				delErr = fmt.Errorf("failed to delete auth request: %v", err)
			}
			result.AuthRequests++
		}
	}
	if delErr != nil {
		return result, delErr
	}

	var authCodes AuthCodeList
	if err := cli.list(resourceAuthCode, &authCodes); err != nil {
		return result, fmt.Errorf("failed to list auth codes: %v", err)
	}

	for _, authCode := range authCodes.AuthCodes {
		if now.After(authCode.Expiry) {
			if err := cli.delete(resourceAuthCode, authCode.ObjectMeta.Name); err != nil {
				cli.logger.Errorf("failed to delete auth code %v", err)
				delErr = fmt.Errorf("failed to delete auth code: %v", err)
			}
			result.AuthCodes++
		}
	}
	return result, delErr
}
