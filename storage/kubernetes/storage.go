package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/dex/storage"
	"github.com/coreos/dex/storage/kubernetes/k8sapi"
)

const (
	kindAuthCode        = "AuthCode"
	kindAuthRequest     = "AuthRequest"
	kindClient          = "OAuth2Client"
	kindRefreshToken    = "RefreshToken"
	kindKeys            = "SigningKey"
	kindPassword        = "Password"
	kindOfflineSessions = "OfflineSessions"
)

const (
	resourceAuthCode        = "authcodes"
	resourceAuthRequest     = "authrequests"
	resourceClient          = "oauth2clients"
	resourceRefreshToken    = "refreshtokens"
	resourceKeys            = "signingkeies" // Kubernetes attempts to pluralize.
	resourcePassword        = "passwords"
	resourceOfflineSessions = "offlinesessionses" // Again attempts to pluralize.
)

// Config values for the Kubernetes storage type.
type Config struct {
	InCluster      bool   `json:"inCluster"`
	KubeConfigFile string `json:"kubeConfigFile"`
}

// Open returns a storage using Kubernetes third party resource.
func (c *Config) Open(logger logrus.FieldLogger) (storage.Storage, error) {
	cli, err := c.open(logger, false)
	if err != nil {
		return nil, err
	}
	return cli, nil
}

// open returns a kubernetes client, initializing the third party resources used
// by dex.
//
// errOnTPRs controls if errors creating the resources cause this method to return
// immediately (used during testing), or if the client will asynchronously retry.
func (c *Config) open(logger logrus.FieldLogger, errOnTPRs bool) (*client, error) {
	if c.InCluster && (c.KubeConfigFile != "") {
		return nil, errors.New("cannot specify both 'inCluster' and 'kubeConfigFile'")
	}
	if !c.InCluster && (c.KubeConfigFile == "") {
		return nil, errors.New("must specify either 'inCluster' or 'kubeConfigFile'")
	}

	var (
		cluster   k8sapi.Cluster
		user      k8sapi.AuthInfo
		namespace string
		err       error
	)
	if c.InCluster {
		cluster, user, namespace, err = inClusterConfig()
	} else {
		cluster, user, namespace, err = loadKubeConfig(c.KubeConfigFile)
	}
	if err != nil {
		return nil, err
	}

	cli, err := newClient(cluster, user, namespace, logger)
	if err != nil {
		return nil, fmt.Errorf("create client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	if !cli.createThirdPartyResources() {
		if errOnTPRs {
			cancel()
			return nil, fmt.Errorf("failed creating third party resources")
		}

		// Try to synchronously create the third party resources once. This doesn't mean
		// they'll immediately be available, but ensures that the client will actually try
		// once.
		logger.Errorf("failed creating third party resources: %v", err)
		go func() {
			for {
				if cli.createThirdPartyResources() {
					return
				}

				select {
				case <-ctx.Done():
					return
				case <-time.After(30 * time.Second):
				}
			}
		}()
	}

	// If the client is closed, stop trying to create third party resources.
	cli.cancel = cancel
	return cli, nil
}

// createThirdPartyResources attempts to create the third party resources dex
// requires or identifies that they're already enabled. It logs all errors,
// returning true if the third party resources were created successfully.
//
// Creating a third party resource does not mean that they'll be immediately available.
//
// TODO(ericchiang): Provide an option to wait for the third party resources
// to actually be available.
func (cli *client) createThirdPartyResources() (ok bool) {
	ok = true
	for _, r := range thirdPartyResources {
		err := cli.postResource("extensions/v1beta1", "", "thirdpartyresources", r)
		if err != nil {
			switch err {
			case storage.ErrAlreadyExists:
				cli.logger.Infof("third party resource already created %s", r.ObjectMeta.Name)
			case storage.ErrNotFound:
				cli.logger.Errorf("third party resources not found, please enable API group extensions/v1beta1")
				ok = false
			default:
				cli.logger.Errorf("creating third party resource %s: %v", r.ObjectMeta.Name, err)
				ok = false
			}
			continue
		}
		cli.logger.Errorf("create third party resource %s", r.ObjectMeta.Name)
	}
	return ok
}

func (cli *client) Close() error {
	if cli.cancel != nil {
		cli.cancel()
	}
	return nil
}

func (cli *client) CreateAuthRequest(a storage.AuthRequest) error {
	return cli.post(resourceAuthRequest, cli.fromStorageAuthRequest(a))
}

func (cli *client) CreateClient(c storage.Client) error {
	return cli.post(resourceClient, cli.fromStorageClient(c))
}

func (cli *client) CreateAuthCode(c storage.AuthCode) error {
	return cli.post(resourceAuthCode, cli.fromStorageAuthCode(c))
}

func (cli *client) CreatePassword(p storage.Password) error {
	return cli.post(resourcePassword, cli.fromStoragePassword(p))
}

func (cli *client) CreateRefresh(r storage.RefreshToken) error {
	return cli.post(resourceRefreshToken, cli.fromStorageRefreshToken(r))
}

func (cli *client) CreateOfflineSessions(o storage.OfflineSessions) error {
	return cli.post(resourceOfflineSessions, cli.fromStorageOfflineSessions(o))
}

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

func (cli *client) getClient(id string) (Client, error) {
	var c Client
	name := cli.idToName(id)
	if err := cli.get(resourceClient, name, &c); err != nil {
		return Client{}, err
	}
	if c.ID != id {
		return Client{}, fmt.Errorf("get client: ID %q mapped to client with ID %q", id, c.ID)
	}
	return c, nil
}

func (cli *client) GetPassword(email string) (storage.Password, error) {
	p, err := cli.getPassword(email)
	if err != nil {
		return storage.Password{}, err
	}
	return toStoragePassword(p), nil
}

func (cli *client) getPassword(email string) (Password, error) {
	// TODO(ericchiang): Figure out whose job it is to lowercase emails.
	email = strings.ToLower(email)
	var p Password
	name := cli.idToName(email)
	if err := cli.get(resourcePassword, name, &p); err != nil {
		return Password{}, err
	}
	if email != p.Email {
		return Password{}, fmt.Errorf("get email: email %q mapped to password with email %q", email, p.Email)
	}
	return p, nil
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

func (cli *client) getRefreshToken(id string) (r RefreshToken, err error) {
	err = cli.get(resourceRefreshToken, id, &r)
	return
}

func (cli *client) GetOfflineSessions(userID string, connID string) (storage.OfflineSessions, error) {
	o, err := cli.getOfflineSessions(userID, connID)
	if err != nil {
		return storage.OfflineSessions{}, err
	}
	return toStorageOfflineSessions(o), nil
}

func (cli *client) getOfflineSessions(userID string, connID string) (o OfflineSessions, err error) {
	name := cli.offlineTokenName(userID, connID)
	if err = cli.get(resourceOfflineSessions, name, &o); err != nil {
		return OfflineSessions{}, err
	}
	if userID != o.UserID || connID != o.ConnID {
		return OfflineSessions{}, fmt.Errorf("get offline session: wrong session retrieved")
	}
	return o, nil
}

func (cli *client) ListClients() ([]storage.Client, error) {
	return nil, errors.New("not implemented")
}

func (cli *client) ListRefreshTokens() ([]storage.RefreshToken, error) {
	return nil, errors.New("not implemented")
}

func (cli *client) ListPasswords() (passwords []storage.Password, err error) {
	var passwordList PasswordList
	if err = cli.list(resourcePassword, &passwordList); err != nil {
		return passwords, fmt.Errorf("failed to list passwords: %v", err)
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

func (cli *client) DeleteAuthRequest(id string) error {
	return cli.delete(resourceAuthRequest, id)
}

func (cli *client) DeleteAuthCode(code string) error {
	return cli.delete(resourceAuthCode, code)
}

func (cli *client) DeleteClient(id string) error {
	// Check for hash collition.
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
	// Check for hash collition.
	p, err := cli.getPassword(email)
	if err != nil {
		return err
	}
	return cli.delete(resourcePassword, p.ObjectMeta.Name)
}

func (cli *client) DeleteOfflineSessions(userID string, connID string) error {
	// Check for hash collition.
	o, err := cli.getOfflineSessions(userID, connID)
	if err != nil {
		return err
	}
	return cli.delete(resourceOfflineSessions, o.ObjectMeta.Name)
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

func (cli *client) UpdateKeys(updater func(old storage.Keys) (storage.Keys, error)) error {
	firstUpdate := false
	var keys Keys
	if err := cli.get(resourceKeys, keysName, &keys); err != nil {
		if err != storage.ErrNotFound {
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
