package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/kubernetes/k8sapi"
)

const (
	kindAuthCode        = "AuthCode"
	kindAuthRequest     = "AuthRequest"
	kindClient          = "OAuth2Client"
	kindRefreshToken    = "RefreshToken"
	kindKeys            = "SigningKey"
	kindPassword        = "Password"
	kindOfflineSessions = "OfflineSessions"
	kindConnector       = "Connector"
	kindDeviceRequest   = "DeviceRequest"
	kindDeviceToken     = "DeviceToken"
)

const (
	resourceAuthCode        = "authcodes"
	resourceAuthRequest     = "authrequests"
	resourceClient          = "oauth2clients"
	resourceRefreshToken    = "refreshtokens"
	resourceKeys            = "signingkeies" // Kubernetes attempts to pluralize.
	resourcePassword        = "passwords"
	resourceOfflineSessions = "offlinesessionses" // Again attempts to pluralize.
	resourceConnector       = "connectors"
	resourceDeviceRequest   = "devicerequests"
	resourceDeviceToken     = "devicetokens"
)

var _ storage.Storage = (*client)(nil)

const (
	gcResultLimit = 500
)

// Config values for the Kubernetes storage type.
type Config struct {
	InCluster      bool   `json:"inCluster"`
	KubeConfigFile string `json:"kubeConfigFile"`
}

// Open returns a storage using Kubernetes third party resource.
func (c *Config) Open(logger *slog.Logger) (storage.Storage, error) {
	cli, err := c.open(logger, false)
	if err != nil {
		return nil, err
	}
	return cli, nil
}

// open returns a kubernetes client, initializing the third party resources used
// by dex.
//
// waitForResources controls if errors creating the resources cause this method to return
// immediately (used during testing), or if the client will asynchronously retry.
func (c *Config) open(logger *slog.Logger, waitForResources bool) (*client, error) {
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

	cli, err := newClient(cluster, user, namespace, logger, c.InCluster)
	if err != nil {
		return nil, fmt.Errorf("create client: %v", err)
	}

	if err = cli.detectKubernetesVersion(); err != nil {
		return nil, fmt.Errorf("cannot get kubernetes version: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	logger.Info("creating custom Kubernetes resources")
	if !cli.registerCustomResources() {
		if waitForResources {
			cancel()
			return nil, fmt.Errorf("failed creating custom resources")
		}

		// Try to synchronously create the custom resources once. This doesn't mean
		// they'll immediately be available, but ensures that the client will actually try
		// once.
		go func() {
			for {
				if cli.registerCustomResources() {
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

	if waitForResources {
		if err := cli.waitForCRDs(ctx); err != nil {
			cancel()
			return nil, err
		}
	}

	// If the client is closed, stop trying to create resources.
	cli.cancel = cancel
	return cli, nil
}

// registerCustomResources attempts to create the custom resources dex
// requires or identifies that they're already enabled. This function creates
// custom resource definitions(CRDs)
// It logs all errors, returning true if the resources were created successfully.
//
// Creating a custom resource does not mean that they'll be immediately available.
func (cli *client) registerCustomResources() (ok bool) {
	ok = true

	definitions := customResourceDefinitions(cli.crdAPIVersion)
	length := len(definitions)

	for i := 0; i < length; i++ {
		var err error
		var resourceName string

		r := definitions[i]
		var i interface{}
		cli.logger.Info("checking if custom resource has already been created...", "object", r.ObjectMeta.Name)
		if err := cli.list(r.Spec.Names.Plural, &i); err == nil {
			cli.logger.Info("the custom resource already available, skipping create", "object", r.ObjectMeta.Name)
			continue
		} else {
			cli.logger.Info("failed to list custom resource, attempting to create", "object", r.ObjectMeta.Name, "err", err)
		}

		err = cli.postResource(cli.crdAPIVersion, "", "customresourcedefinitions", r)
		resourceName = r.ObjectMeta.Name

		if err != nil {
			switch err {
			case storage.ErrAlreadyExists:
				cli.logger.Info("custom resource already created", "object", resourceName)
			case storage.ErrNotFound:
				cli.logger.Error("custom resources not found, please enable the respective API group")
				ok = false
			default:
				cli.logger.Error("creating custom resource", "object", resourceName, "err", err)
				ok = false
			}
			continue
		}
		cli.logger.Error("create custom resource", "object", resourceName)
	}
	return ok
}

// waitForCRDs waits for all CRDs to be in a ready state, and is used
// by the tests to synchronize before running conformance.
func (cli *client) waitForCRDs(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	for _, crd := range customResourceDefinitions(cli.crdAPIVersion) {
		for {
			err := cli.isCRDReady(crd.Name)
			if err == nil {
				break
			}

			cli.logger.ErrorContext(ctx, "checking CRD", "err", err)

			select {
			case <-ctx.Done():
				return errors.New("timed out waiting for CRDs to be available")
			case <-time.After(time.Millisecond * 100):
			}
		}
	}
	return nil
}

// isCRDReady determines if a CRD is ready by inspecting its conditions.
func (cli *client) isCRDReady(name string) error {
	var r k8sapi.CustomResourceDefinition
	err := cli.getResource(cli.crdAPIVersion, "", "customresourcedefinitions", name, &r)
	if err != nil {
		return fmt.Errorf("get crd %s: %v", name, err)
	}

	conds := make(map[string]string) // For debugging, keep the conditions around.
	for _, c := range r.Status.Conditions {
		if c.Type == k8sapi.Established && c.Status == k8sapi.ConditionTrue {
			return nil
		}
		conds[string(c.Type)] = string(c.Status)
	}
	return fmt.Errorf("crd %s not ready %#v", name, conds)
}

func (cli *client) Close() error {
	if cli.cancel != nil {
		cli.cancel()
	}
	return nil
}

func (cli *client) CreateAuthRequest(ctx context.Context, a storage.AuthRequest) error {
	return cli.post(resourceAuthRequest, cli.fromStorageAuthRequest(a))
}

func (cli *client) CreateClient(ctx context.Context, c storage.Client) error {
	return cli.post(resourceClient, cli.fromStorageClient(c))
}

func (cli *client) CreateAuthCode(ctx context.Context, c storage.AuthCode) error {
	return cli.post(resourceAuthCode, cli.fromStorageAuthCode(c))
}

func (cli *client) CreatePassword(ctx context.Context, p storage.Password) error {
	return cli.post(resourcePassword, cli.fromStoragePassword(p))
}

func (cli *client) CreateRefresh(ctx context.Context, r storage.RefreshToken) error {
	return cli.post(resourceRefreshToken, cli.fromStorageRefreshToken(r))
}

func (cli *client) CreateOfflineSessions(ctx context.Context, o storage.OfflineSessions) error {
	return cli.post(resourceOfflineSessions, cli.fromStorageOfflineSessions(o))
}

func (cli *client) CreateConnector(ctx context.Context, c storage.Connector) error {
	return cli.post(resourceConnector, cli.fromStorageConnector(c))
}

func (cli *client) GetAuthRequest(ctx context.Context, id string) (storage.AuthRequest, error) {
	var req AuthRequest
	if err := cli.get(resourceAuthRequest, id, &req); err != nil {
		return storage.AuthRequest{}, err
	}
	return toStorageAuthRequest(req), nil
}

func (cli *client) GetAuthCode(ctx context.Context, id string) (storage.AuthCode, error) {
	var code AuthCode
	if err := cli.get(resourceAuthCode, id, &code); err != nil {
		return storage.AuthCode{}, err
	}
	return toStorageAuthCode(code), nil
}

func (cli *client) GetClient(ctx context.Context, id string) (storage.Client, error) {
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

func (cli *client) GetPassword(ctx context.Context, email string) (storage.Password, error) {
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

func (cli *client) GetKeys(ctx context.Context) (storage.Keys, error) {
	var keys Keys
	if err := cli.get(resourceKeys, keysName, &keys); err != nil {
		return storage.Keys{}, err
	}
	return toStorageKeys(keys), nil
}

func (cli *client) GetRefresh(ctx context.Context, id string) (storage.RefreshToken, error) {
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

func (cli *client) GetOfflineSessions(ctx context.Context, userID string, connID string) (storage.OfflineSessions, error) {
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

func (cli *client) GetConnector(ctx context.Context, id string) (storage.Connector, error) {
	var c Connector
	if err := cli.get(resourceConnector, id, &c); err != nil {
		return storage.Connector{}, err
	}
	return toStorageConnector(c), nil
}

func (cli *client) ListClients(ctx context.Context) ([]storage.Client, error) {
	return nil, errors.New("not implemented")
}

func (cli *client) ListRefreshTokens(ctx context.Context) ([]storage.RefreshToken, error) {
	return nil, errors.New("not implemented")
}

func (cli *client) ListPasswords(ctx context.Context) (passwords []storage.Password, err error) {
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

func (cli *client) ListConnectors(ctx context.Context) (connectors []storage.Connector, err error) {
	var connectorList ConnectorList
	if err = cli.list(resourceConnector, &connectorList); err != nil {
		return connectors, fmt.Errorf("failed to list connectors: %v", err)
	}

	connectors = make([]storage.Connector, len(connectorList.Connectors))
	for i, connector := range connectorList.Connectors {
		connectors[i] = toStorageConnector(connector)
	}

	return
}

func (cli *client) DeleteAuthRequest(ctx context.Context, id string) error {
	return cli.delete(resourceAuthRequest, id)
}

func (cli *client) DeleteAuthCode(ctx context.Context, code string) error {
	return cli.delete(resourceAuthCode, code)
}

func (cli *client) DeleteClient(ctx context.Context, id string) error {
	// Check for hash collision.
	c, err := cli.getClient(id)
	if err != nil {
		return err
	}
	return cli.delete(resourceClient, c.ObjectMeta.Name)
}

func (cli *client) DeleteRefresh(ctx context.Context, id string) error {
	return cli.delete(resourceRefreshToken, id)
}

func (cli *client) DeletePassword(ctx context.Context, email string) error {
	// Check for hash collision.
	p, err := cli.getPassword(email)
	if err != nil {
		return err
	}
	return cli.delete(resourcePassword, p.ObjectMeta.Name)
}

func (cli *client) DeleteOfflineSessions(ctx context.Context, userID string, connID string) error {
	// Check for hash collision.
	o, err := cli.getOfflineSessions(userID, connID)
	if err != nil {
		return err
	}
	return cli.delete(resourceOfflineSessions, o.ObjectMeta.Name)
}

func (cli *client) DeleteConnector(ctx context.Context, id string) error {
	return cli.delete(resourceConnector, id)
}

func (cli *client) UpdateRefreshToken(ctx context.Context, id string, updater func(old storage.RefreshToken) (storage.RefreshToken, error)) error {
	lock := newRefreshTokenLock(cli)

	if err := lock.Lock(id); err != nil {
		return err
	}
	defer lock.Unlock(id)

	return retryOnConflict(ctx, func() error {
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
	})
}

func (cli *client) UpdateClient(ctx context.Context, id string, updater func(old storage.Client) (storage.Client, error)) error {
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

func (cli *client) UpdatePassword(ctx context.Context, email string, updater func(old storage.Password) (storage.Password, error)) error {
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

func (cli *client) UpdateOfflineSessions(ctx context.Context, userID string, connID string, updater func(old storage.OfflineSessions) (storage.OfflineSessions, error)) error {
	return retryOnConflict(ctx, func() error {
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
	})
}

func (cli *client) UpdateKeys(ctx context.Context, updater func(old storage.Keys) (storage.Keys, error)) error {
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
		err = cli.post(resourceKeys, newKeys)
		if err != nil && errors.Is(err, storage.ErrAlreadyExists) {
			// We need to tolerate conflicts here in case of HA mode.
			cli.logger.Debug("Keys creation failed. It is possible that keys have already been created by another dex instance.", "err", err)
			return errors.New("keys already created by another server instance")
		}

		return err
	}

	newKeys.ObjectMeta = keys.ObjectMeta

	err = cli.put(resourceKeys, keysName, newKeys)
	if isKubernetesAPIConflictError(err) {
		// We need to tolerate conflicts here in case of HA mode.
		// Dex instances run keys rotation at the same time because they use SigningKey.nextRotation CR field as a trigger.
		cli.logger.Debug("Keys rotation failed. It is possible that keys have already been rotated by another dex instance.", "err", err)
		return errors.New("keys already rotated by another server instance")
	}

	return err
}

func (cli *client) UpdateAuthRequest(ctx context.Context, id string, updater func(a storage.AuthRequest) (storage.AuthRequest, error)) error {
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

func (cli *client) UpdateConnector(ctx context.Context, id string, updater func(a storage.Connector) (storage.Connector, error)) error {
	return retryOnConflict(ctx, func() error {
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
	})
}

func (cli *client) GarbageCollect(ctx context.Context, now time.Time) (result storage.GCResult, err error) {
	var authRequests AuthRequestList
	if err := cli.listN(resourceAuthRequest, &authRequests, gcResultLimit); err != nil {
		return result, fmt.Errorf("failed to list auth requests: %v", err)
	}

	var delErr error
	for _, authRequest := range authRequests.AuthRequests {
		if now.After(authRequest.Expiry) {
			if err := cli.delete(resourceAuthRequest, authRequest.ObjectMeta.Name); err != nil {
				cli.logger.Error("failed to delete auth request", "err", err)
				delErr = fmt.Errorf("failed to delete auth request: %v", err)
			}
			result.AuthRequests++
		}
	}
	if delErr != nil {
		return result, delErr
	}

	var authCodes AuthCodeList
	if err := cli.listN(resourceAuthCode, &authCodes, gcResultLimit); err != nil {
		return result, fmt.Errorf("failed to list auth codes: %v", err)
	}

	for _, authCode := range authCodes.AuthCodes {
		if now.After(authCode.Expiry) {
			if err := cli.delete(resourceAuthCode, authCode.ObjectMeta.Name); err != nil {
				cli.logger.Error("failed to delete auth code", "err", err)
				delErr = fmt.Errorf("failed to delete auth code: %v", err)
			}
			result.AuthCodes++
		}
	}

	var deviceRequests DeviceRequestList
	if err := cli.listN(resourceDeviceRequest, &deviceRequests, gcResultLimit); err != nil {
		return result, fmt.Errorf("failed to list device requests: %v", err)
	}

	for _, deviceRequest := range deviceRequests.DeviceRequests {
		if now.After(deviceRequest.Expiry) {
			if err := cli.delete(resourceDeviceRequest, deviceRequest.ObjectMeta.Name); err != nil {
				cli.logger.Error("failed to delete device request", "err", err)
				delErr = fmt.Errorf("failed to delete device request: %v", err)
			}
			result.DeviceRequests++
		}
	}

	var deviceTokens DeviceTokenList
	if err := cli.listN(resourceDeviceToken, &deviceTokens, gcResultLimit); err != nil {
		return result, fmt.Errorf("failed to list device tokens: %v", err)
	}

	for _, deviceToken := range deviceTokens.DeviceTokens {
		if now.After(deviceToken.Expiry) {
			if err := cli.delete(resourceDeviceToken, deviceToken.ObjectMeta.Name); err != nil {
				cli.logger.Error("failed to delete device token", "err", err)
				delErr = fmt.Errorf("failed to delete device token: %v", err)
			}
			result.DeviceTokens++
		}
	}

	if delErr != nil {
		return result, delErr
	}
	return result, delErr
}

func (cli *client) CreateDeviceRequest(ctx context.Context, d storage.DeviceRequest) error {
	return cli.post(resourceDeviceRequest, cli.fromStorageDeviceRequest(d))
}

func (cli *client) GetDeviceRequest(ctx context.Context, userCode string) (storage.DeviceRequest, error) {
	var req DeviceRequest
	if err := cli.get(resourceDeviceRequest, strings.ToLower(userCode), &req); err != nil {
		return storage.DeviceRequest{}, err
	}
	return toStorageDeviceRequest(req), nil
}

func (cli *client) CreateDeviceToken(ctx context.Context, t storage.DeviceToken) error {
	return cli.post(resourceDeviceToken, cli.fromStorageDeviceToken(t))
}

func (cli *client) GetDeviceToken(ctx context.Context, deviceCode string) (storage.DeviceToken, error) {
	var token DeviceToken
	if err := cli.get(resourceDeviceToken, deviceCode, &token); err != nil {
		return storage.DeviceToken{}, err
	}
	return toStorageDeviceToken(token), nil
}

func (cli *client) getDeviceToken(deviceCode string) (t DeviceToken, err error) {
	err = cli.get(resourceDeviceToken, deviceCode, &t)
	return
}

func (cli *client) UpdateDeviceToken(ctx context.Context, deviceCode string, updater func(old storage.DeviceToken) (storage.DeviceToken, error)) error {
	return retryOnConflict(ctx, func() error {
		r, err := cli.getDeviceToken(deviceCode)
		if err != nil {
			return err
		}
		updated, err := updater(toStorageDeviceToken(r))
		if err != nil {
			return err
		}
		updated.DeviceCode = deviceCode

		newToken := cli.fromStorageDeviceToken(updated)
		newToken.ObjectMeta = r.ObjectMeta
		return cli.put(resourceDeviceToken, r.ObjectMeta.Name, newToken)
	})
}

func isKubernetesAPIConflictError(err error) bool {
	if httpErr, ok := err.(httpError); ok {
		if httpErr.StatusCode() == http.StatusConflict {
			return true
		}
	}
	return false
}

func retryOnConflict(ctx context.Context, action func() error) error {
	policy := []int{10, 20, 100, 300, 600}

	attempts := 0
	getNextStep := func() time.Duration {
		step := policy[attempts]
		return time.Duration(step*5+rand.Intn(step)) * time.Microsecond
	}

	if err := action(); err == nil || !isKubernetesAPIConflictError(err) {
		return err
	}

	for {
		select {
		case <-time.After(getNextStep()):
			err := action()
			if err == nil || !isKubernetesAPIConflictError(err) {
				return err
			}

			attempts++
			if attempts >= 4 {
				return fmt.Errorf("maximum timeout reached while retrying a conflicted request: %w", err)
			}
		case <-ctx.Done():
			return errors.New("canceled")
		}
	}
}
