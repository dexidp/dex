package kubernetes

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"golang.org/x/net/context"

	"github.com/coreos/dex/storage"
	"github.com/coreos/dex/storage/kubernetes/k8sapi"
)

const (
	kindAuthCode     = "AuthCode"
	kindAuthRequest  = "AuthRequest"
	kindClient       = "OAuth2Client"
	kindRefreshToken = "RefreshToken"
	kindKeys         = "SigningKey"
	kindPassword     = "Password"
)

const (
	resourceAuthCode     = "authcodes"
	resourceAuthRequest  = "authrequests"
	resourceClient       = "oauth2clients"
	resourceRefreshToken = "refreshtokens"
	resourceKeys         = "signingkeies" // Kubernetes attempts to pluralize.
	resourcePassword     = "passwords"
)

// Config values for the Kubernetes storage type.
type Config struct {
	InCluster      bool   `yaml:"inCluster"`
	KubeConfigFile string `yaml:"kubeConfigFile"`
}

// Open returns a storage using Kubernetes third party resource.
func (c *Config) Open() (storage.Storage, error) {
	cli, err := c.open()
	if err != nil {
		return nil, err
	}
	return cli, nil
}

// open returns a client with no garbage collection.
func (c *Config) open() (*client, error) {
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

	cli, err := newClient(cluster, user, namespace)
	if err != nil {
		return nil, fmt.Errorf("create client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Try to synchronously create the third party resources once. This doesn't mean
	// they'll immediately be available, but ensures that the client will actually try
	// once.
	if err := cli.createThirdPartyResources(); err != nil {
		log.Printf("failed creating third party resources: %v", err)
		go func() {
			for {
				if err := cli.createThirdPartyResources(); err != nil {
					log.Printf("failed creating third party resources: %v", err)
				} else {
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
// requires or identifies that they're already enabled.
//
// Creating a third party resource does not mean that they'll be immediately available.
//
// TODO(ericchiang): Provide an option to wait for the third party resources
// to actually be available.
func (cli *client) createThirdPartyResources() error {
	for _, r := range thirdPartyResources {
		err := cli.postResource("extensions/v1beta1", "", "thirdpartyresources", r)
		if err != nil {
			if e, ok := err.(httpError); ok {
				if e.StatusCode() == http.StatusConflict {
					log.Printf("third party resource already created %q", r.ObjectMeta.Name)
					continue
				}
			}
			return err
		}
		log.Printf("create third party resource %q", r.ObjectMeta.Name)
	}
	return nil
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
	refresh := RefreshToken{
		TypeMeta: k8sapi.TypeMeta{
			Kind:       kindRefreshToken,
			APIVersion: cli.apiVersion,
		},
		ObjectMeta: k8sapi.ObjectMeta{
			Name:      r.RefreshToken,
			Namespace: cli.namespace,
		},
		ClientID:    r.ClientID,
		ConnectorID: r.ConnectorID,
		Scopes:      r.Scopes,
		Nonce:       r.Nonce,
		Claims:      fromStorageClaims(r.Claims),
	}
	return cli.post(resourceRefreshToken, refresh)
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
	var c Client
	if err := cli.get(resourceClient, id, &c); err != nil {
		return storage.Client{}, err
	}
	return toStorageClient(c), nil
}

func (cli *client) GetPassword(email string) (storage.Password, error) {
	var p Password
	if err := cli.get(resourcePassword, emailToID(email), &p); err != nil {
		return storage.Password{}, err
	}
	return toStoragePassword(p), nil
}

func (cli *client) GetKeys() (storage.Keys, error) {
	var keys Keys
	if err := cli.get(resourceKeys, keysName, &keys); err != nil {
		return storage.Keys{}, err
	}
	return toStorageKeys(keys), nil
}

func (cli *client) GetRefresh(id string) (storage.RefreshToken, error) {
	var r RefreshToken
	if err := cli.get(resourceRefreshToken, id, &r); err != nil {
		return storage.RefreshToken{}, err
	}
	return storage.RefreshToken{
		RefreshToken: r.ObjectMeta.Name,
		ClientID:     r.ClientID,
		ConnectorID:  r.ConnectorID,
		Scopes:       r.Scopes,
		Nonce:        r.Nonce,
		Claims:       toStorageClaims(r.Claims),
	}, nil
}

func (cli *client) ListClients() ([]storage.Client, error) {
	return nil, errors.New("not implemented")
}

func (cli *client) ListRefreshTokens() ([]storage.RefreshToken, error) {
	return nil, errors.New("not implemented")
}

func (cli *client) DeleteAuthRequest(id string) error {
	return cli.delete(resourceAuthRequest, id)
}

func (cli *client) DeleteAuthCode(code string) error {
	return cli.delete(resourceAuthCode, code)
}

func (cli *client) DeleteClient(id string) error {
	return cli.delete(resourceClient, id)
}

func (cli *client) DeleteRefresh(id string) error {
	return cli.delete(resourceRefreshToken, id)
}

func (cli *client) DeletePassword(email string) error {
	return cli.delete(resourcePassword, emailToID(email))
}

func (cli *client) UpdateClient(id string, updater func(old storage.Client) (storage.Client, error)) error {
	var c Client
	if err := cli.get(resourceClient, id, &c); err != nil {
		return err
	}
	updated, err := updater(toStorageClient(c))
	if err != nil {
		return err
	}

	newClient := cli.fromStorageClient(updated)
	newClient.ObjectMeta = c.ObjectMeta
	return cli.put(resourceClient, id, newClient)
}

func (cli *client) UpdatePassword(email string, updater func(old storage.Password) (storage.Password, error)) error {
	id := emailToID(email)
	var p Password
	if err := cli.get(resourcePassword, id, &p); err != nil {
		return err
	}

	updated, err := updater(toStoragePassword(p))
	if err != nil {
		return err
	}

	newPassword := cli.fromStoragePassword(updated)
	newPassword.ObjectMeta = p.ObjectMeta
	return cli.put(resourcePassword, id, newPassword)
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
				log.Printf("failed to delete auth request: %v", err)
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
				log.Printf("failed to delete auth code %v", err)
				delErr = fmt.Errorf("failed to delete auth code: %v", err)
			}
			result.AuthCodes++
		}
	}
	return result, delErr
}
