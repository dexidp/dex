package kubernetes

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gtank/cryptopasta"
	"golang.org/x/net/http2"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/kubernetes/k8sapi"
)

// Kubernetes names must match the regexp '[a-z0-9]([-a-z0-9]*[a-z0-9])?'.
var encoding = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567")

type client struct {
	client    *http.Client
	baseURL   string
	namespace string
	logger    log.Logger

	// Hash function to map IDs (which could span a large range) to Kubernetes names.
	// While this is not currently upgradable, it could be in the future.
	//
	// The default hash is a non-cryptographic hash, because cryptographic hashes
	// always produce sums too long to fit into a Kubernetes name. Because of this,
	// gets, updates, and deletes are _always_ checked for collisions.
	hash func() hash.Hash

	// API version of the oidc resources. For example "oidc.coreos.com". This is
	// currently not configurable, but could be in the future.
	apiVersion string

	// This is called once the client's Close method is called to signal goroutines,
	// such as the one creating third party resources, to stop.
	cancel context.CancelFunc
}

func newClient(cluster k8sapi.Cluster, user k8sapi.AuthInfo, namespace string, logger log.Logger) (*client, error) {
	tlsConfig := cryptopasta.DefaultTLSConfig()
	data := func(b string, file string) ([]byte, error) {
		if b != "" {
			return base64.StdEncoding.DecodeString(b)
		}
		if file == "" {
			return nil, nil
		}
		return ioutil.ReadFile(file)
	}

	if caData, err := data(cluster.CertificateAuthorityData, cluster.CertificateAuthority); err != nil {
		return nil, err
	} else if caData != nil {
		tlsConfig.RootCAs = x509.NewCertPool()
		if !tlsConfig.RootCAs.AppendCertsFromPEM(caData) {
			return nil, fmt.Errorf("no certificate data found: %v", err)
		}
	}

	clientCert, err := data(user.ClientCertificateData, user.ClientCertificate)
	if err != nil {
		return nil, err
	}
	clientKey, err := data(user.ClientKeyData, user.ClientKey)
	if err != nil {
		return nil, err
	}
	if clientCert != nil && clientKey != nil {
		cert, err := tls.X509KeyPair(clientCert, clientKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load client cert: %v", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	var t http.RoundTripper
	httpTransport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSClientConfig:       tlsConfig,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// Since we set a custom TLS client config we have to explicitly
	// enable HTTP/2.
	//
	// https://github.com/golang/go/blob/go1.7.4/src/net/http/transport.go#L200-L206
	if err := http2.ConfigureTransport(httpTransport); err != nil {
		return nil, err
	}
	t = httpTransport

	if user.Token != "" {
		t = transport{
			updateReq: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+user.Token)
			},
			base: t,
		}
	}

	if user.Username != "" && user.Password != "" {
		t = transport{
			updateReq: func(r *http.Request) {
				r.SetBasicAuth(user.Username, user.Password)
			},
			base: t,
		}
	}

	logger.Infof("kubernetes client apiVersion = %s", apiFullPath)
	return &client{
		client: &http.Client{
			Transport: t,
			Timeout:   15 * time.Second,
		},
		baseURL:    cluster.Server,
		hash:       func() hash.Hash { return fnv.New64() },
		namespace:  namespace,
		apiVersion: apiFullPath,
		logger:     logger,
	}, nil
}

// idToName maps an arbitrary ID, such as an email or client ID to a Kubernetes object name.
func (cli *client) idToName(s string) string {
	return idToName(s, cli.hash)
}

func idToName(s string, h func() hash.Hash) string {
	return strings.TrimRight(encoding.EncodeToString(h().Sum([]byte(s))), "=")
}

func (cli *client) getClient(id string) (Client, error) {
	var c Client
	name := cli.idToName(id)
	if err := cli.get(resourceClient, name, &c); err != nil {
		return Client{}, err
	}
	if c.ID != id {
		return Client{}, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("get client: ID %q mapped to client with ID %q", id, c.ID)}
	}
	return c, nil
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
		return Password{}, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("get email: email %q mapped to password with email %q", email, p.Email)}
	}
	return p, nil
}

func (cli *client) getRefreshToken(id string) (r RefreshToken, err error) {
	err = cli.get(resourceRefreshToken, id, &r)
	return
}

func (cli *client) getOfflineSessions(userID string, connID string) (o OfflineSessions, err error) {
	name := cli.offlineTokenName(userID, connID)
	if err = cli.get(resourceOfflineSessions, name, &o); err != nil {
		return OfflineSessions{}, err
	}
	if userID != o.UserID || connID != o.ConnID {
		return OfflineSessions{}, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: errWrongSessionRetrieved}
	}
	return o, nil
}

// offlineTokenName maps two arbitrary IDs, to a single Kubernetes object name.
// This is used when more than one field is used to uniquely identify the object.
func (cli *client) offlineTokenName(userID string, connID string) string {
	return offlineTokenName(userID, connID, cli.hash)
}

// TODO(venezia) - Does this still need to be a separate function since its only called within a (client).offlineTokenName ?
func offlineTokenName(userID string, connID string, h func() hash.Hash) string {
	hash := h()
	hash.Write([]byte(userID))
	hash.Write([]byte(connID))
	return strings.TrimRight(encoding.EncodeToString(hash.Sum(nil)), "=")
}

// Kubernetes Resource CRUDers

func (cli *client) get(resource, name string, v interface{}) error {
	return cli.getResource(cli.apiVersion, cli.namespace, resource, name, v)
}

func (cli *client) getResource(apiVersion, namespace, resource, name string, v interface{}) error {
	url := cli.urlFor(apiVersion, namespace, resource, name)
	resp, err := cli.client.Get(url)
	if err != nil {
		return err
	}
	defer closeResp(resp)
	if err := checkHTTPErr(resp, http.StatusOK); err != nil {
		return err
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func (cli *client) list(resource string, v interface{}) error {
	return cli.get(resource, "", v)
}

func (cli *client) post(resource string, v interface{}) error {
	return cli.postResource(cli.apiVersion, cli.namespace, resource, v)
}

func (cli *client) postResource(apiVersion, namespace, resource string, v interface{}) error {
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal object: %v", err)
	}

	url := cli.urlFor(apiVersion, namespace, resource, "")
	resp, err := cli.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer closeResp(resp)
	return checkHTTPErr(resp, http.StatusCreated)
}

func (cli *client) delete(resource, name string) error {
	url := cli.urlFor(cli.apiVersion, cli.namespace, resource, name)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("create delete request: %v", err)
	}
	resp, err := cli.client.Do(req)
	if err != nil {
		return fmt.Errorf("delete request: %v", err)
	}
	defer closeResp(resp)
	return checkHTTPErr(resp, http.StatusOK)
}

func (cli *client) deleteAll(resource string) error {
	var list struct {
		k8sapi.TypeMeta `json:",inline"`
		k8sapi.ListMeta `json:"metadata,omitempty"`
		Items           []struct {
			k8sapi.TypeMeta   `json:",inline"`
			k8sapi.ObjectMeta `json:"metadata,omitempty"`
		} `json:"items"`
	}
	if err := cli.list(resource, &list); err != nil {
		return err
	}
	for _, item := range list.Items {
		if err := cli.delete(resource, item.Name); err != nil {
			return err
		}
	}
	return nil
}

func (cli *client) put(resource, name string, v interface{}) error {
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal object: %v", err)
	}

	url := cli.urlFor(cli.apiVersion, cli.namespace, resource, name)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create patch request: %v", err)
	}

	req.Header.Set("Content-Length", strconv.Itoa(len(body)))

	resp, err := cli.client.Do(req)
	if err != nil {
		return fmt.Errorf("patch request: %v", err)
	}
	defer closeResp(resp)

	return checkHTTPErr(resp, http.StatusOK)
}

// Close the response body. The initial request is drained so the connection can
// be reused.
func closeResp(r *http.Response) {
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()
}

func (cli *client) urlFor(apiVersion, namespace, resource, name string) string {
	basePath := "apis/"
	if apiVersion == "v1" {
		basePath = "api/"
	}

	var p string
	if namespace != "" {
		p = path.Join(basePath, apiVersion, "namespaces", namespace, resource, name)
	} else {
		p = path.Join(basePath, apiVersion, resource, name)
	}
	if strings.HasSuffix(cli.baseURL, "/") {
		return cli.baseURL + p
	}
	return cli.baseURL + "/" + p
}
