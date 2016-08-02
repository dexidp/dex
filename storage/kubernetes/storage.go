package kubernetes

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"golang.org/x/net/context"

	"github.com/coreos/poke/storage"
	"github.com/coreos/poke/storage/kubernetes/k8sapi"
)

const (
	kindAuthCode     = "AuthCode"
	kindAuthRequest  = "AuthRequest"
	kindClient       = "OAuth2Client"
	kindRefreshToken = "RefreshToken"
	kindKeys         = "SigningKey"
)

const (
	resourceAuthCode     = "authcodes"
	resourceAuthRequest  = "authrequests"
	resourceClient       = "oauth2clients"
	resourceRefreshToken = "refreshtokens"
	resourceKeys         = "signingkeies" // Kubernetes attempts to pluralize.
)

// Config values for the Kubernetes storage type.
type Config struct {
	InCluster      bool   `yaml:"inCluster"`
	KubeConfigPath string `yaml:"kubeConfigPath"`
	GCFrequency    int64  `yaml:"gcFrequency"` // seconds
}

// Open returns a storage using Kubernetes third party resource.
func (c *Config) Open() (storage.Storage, error) {
	cli, err := c.open()
	if err != nil {
		return nil, err
	}

	// start up garbage collection
	gcFrequency := c.GCFrequency
	if gcFrequency == 0 {
		gcFrequency = 600
	}
	ctx, cancel := context.WithCancel(context.Background())
	cli.cancel = cancel
	go cli.gc(ctx, time.Duration(gcFrequency)*time.Second)
	return cli, nil
}

// open returns a client with no garbage collection.
func (c *Config) open() (*client, error) {
	if c.InCluster && (c.KubeConfigPath != "") {
		return nil, errors.New("cannot specify both 'inCluster' and 'kubeConfigPath'")
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
		kubeConfigPath := c.KubeConfigPath
		if kubeConfigPath == "" {
			kubeConfigPath = os.Getenv("KUBECONFIG")
		}
		if kubeConfigPath == "" {
			p, err := homedir.Dir()
			if err != nil {
				return nil, fmt.Errorf("finding homedir: %v", err)
			}
			kubeConfigPath = filepath.Join(p, ".kube", "config")
		}
		cluster, user, namespace, err = loadKubeConfig(kubeConfigPath)
	}
	if err != nil {
		return nil, err
	}

	return newClient(cluster, user, namespace)
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

func (cli *client) CreateRefresh(r storage.Refresh) error {
	refresh := Refresh{
		TypeMeta: k8sapi.TypeMeta{
			Kind:       kindRefreshToken,
			APIVersion: cli.apiVersionForResource(resourceRefreshToken),
		},
		ObjectMeta: k8sapi.ObjectMeta{
			Name:      r.RefreshToken,
			Namespace: cli.namespace,
		},
		ClientID:    r.ClientID,
		ConnectorID: r.ConnectorID,
		Scopes:      r.Scopes,
		Nonce:       r.Nonce,
		Identity:    fromStorageIdentity(r.Identity),
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

func (cli *client) GetKeys() (storage.Keys, error) {
	var keys Keys
	if err := cli.get(resourceKeys, keysName, &keys); err != nil {
		return storage.Keys{}, err
	}
	return toStorageKeys(keys), nil
}

func (cli *client) GetRefresh(id string) (storage.Refresh, error) {
	var r Refresh
	if err := cli.get(resourceRefreshToken, id, &r); err != nil {
		return storage.Refresh{}, err
	}
	return storage.Refresh{
		RefreshToken: r.ObjectMeta.Name,
		ClientID:     r.ClientID,
		ConnectorID:  r.ConnectorID,
		Scopes:       r.Scopes,
		Nonce:        r.Nonce,
		Identity:     toStorageIdentity(r.Identity),
	}, nil
}

func (cli *client) ListClients() ([]storage.Client, error) {
	return nil, errors.New("not implemented")
}

func (cli *client) ListRefreshTokens() ([]storage.Refresh, error) {
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
