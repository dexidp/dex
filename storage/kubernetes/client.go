package kubernetes

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/gtank/cryptopasta"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/kubernetes/k8sapi"
)

type client struct {
	client    *http.Client
	baseURL   string
	namespace string
	logger    logrus.FieldLogger

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

// idToName maps an arbitrary ID, such as an email or client ID to a Kubernetes object name.
func (c *client) idToName(s string) string {
	return idToName(s, c.hash)
}

// offlineTokenName maps two arbitrary IDs, to a single Kubernetes object name.
// This is used when more than one field is used to uniquely identify the object.
func (c *client) offlineTokenName(userID string, connID string) string {
	return offlineTokenName(userID, connID, c.hash)
}

// Kubernetes names must match the regexp '[a-z0-9]([-a-z0-9]*[a-z0-9])?'.
var encoding = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567")

func idToName(s string, h func() hash.Hash) string {
	return strings.TrimRight(encoding.EncodeToString(h().Sum([]byte(s))), "=")
}

func offlineTokenName(userID string, connID string, h func() hash.Hash) string {
	hash := h()
	hash.Write([]byte(userID))
	hash.Write([]byte(connID))
	return strings.TrimRight(encoding.EncodeToString(hash.Sum(nil)), "=")
}

func (c *client) urlFor(apiVersion, namespace, resource, name string) string {
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
	if strings.HasSuffix(c.baseURL, "/") {
		return c.baseURL + p
	}
	return c.baseURL + "/" + p
}

// Define an error interface so we can get at the underlying status code if it's
// absolutely necessary. For instance when we need to see if an error indicates
// a resource already exists.
type httpError interface {
	StatusCode() int
}

var _ httpError = (*httpErr)(nil)

type httpErr struct {
	method string
	url    string
	status int
	body   []byte
}

func (e *httpErr) StatusCode() int {
	return e.status
}

func (e *httpErr) Error() string {
	return fmt.Sprintf("%s %s %s: response from server \"%s\"", e.method, e.url, http.StatusText(e.status), bytes.TrimSpace(e.body))
}

func checkHTTPErr(r *http.Response, validStatusCodes ...int) error {
	for _, status := range validStatusCodes {
		if r.StatusCode == status {
			return nil
		}
	}

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 2<<15)) // 64 KiB
	if err != nil {
		return fmt.Errorf("read response body: %v", err)
	}

	// Check this case after we read the body so the connection can be reused.
	if r.StatusCode == http.StatusNotFound {
		return storage.ErrNotFound
	}
	if r.Request.Method == http.MethodPost && r.StatusCode == http.StatusConflict {
		return storage.ErrAlreadyExists
	}

	var url, method string
	if r.Request != nil {
		method = r.Request.Method
		url = r.Request.URL.String()
	}
	return &httpErr{method, url, r.StatusCode, body}
}

// Close the response body. The initial request is drained so the connection can
// be reused.
func closeResp(r *http.Response) {
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()
}

func (c *client) get(resource, name string, v interface{}) error {
	return c.getResource(c.apiVersion, c.namespace, resource, name, v)
}

func (c *client) getResource(apiVersion, namespace, resource, name string, v interface{}) error {
	url := c.urlFor(apiVersion, namespace, resource, name)
	resp, err := c.client.Get(url)
	if err != nil {
		return err
	}
	defer closeResp(resp)
	if err := checkHTTPErr(resp, http.StatusOK); err != nil {
		return err
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func (c *client) list(resource string, v interface{}) error {
	return c.get(resource, "", v)
}

func (c *client) post(resource string, v interface{}) error {
	return c.postResource(c.apiVersion, c.namespace, resource, v)
}

func (c *client) postResource(apiVersion, namespace, resource string, v interface{}) error {
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal object: %v", err)
	}

	url := c.urlFor(apiVersion, namespace, resource, "")
	resp, err := c.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer closeResp(resp)
	return checkHTTPErr(resp, http.StatusCreated)
}

func (c *client) delete(resource, name string) error {
	url := c.urlFor(c.apiVersion, c.namespace, resource, name)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("create delete request: %v", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("delete request: %v", err)
	}
	defer closeResp(resp)
	return checkHTTPErr(resp, http.StatusOK)
}

func (c *client) deleteAll(resource string) error {
	var list struct {
		k8sapi.TypeMeta `json:",inline"`
		k8sapi.ListMeta `json:"metadata,omitempty"`
		Items           []struct {
			k8sapi.TypeMeta   `json:",inline"`
			k8sapi.ObjectMeta `json:"metadata,omitempty"`
		} `json:"items"`
	}
	if err := c.list(resource, &list); err != nil {
		return err
	}
	for _, item := range list.Items {
		if err := c.delete(resource, item.Name); err != nil {
			return err
		}
	}
	return nil
}

func (c *client) put(resource, name string, v interface{}) error {
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal object: %v", err)
	}

	url := c.urlFor(c.apiVersion, c.namespace, resource, name)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create patch request: %v", err)
	}

	req.Header.Set("Content-Length", strconv.Itoa(len(body)))

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("patch request: %v", err)
	}
	defer closeResp(resp)

	return checkHTTPErr(resp, http.StatusOK)
}

func newClient(cluster k8sapi.Cluster, user k8sapi.AuthInfo, namespace string, logger logrus.FieldLogger, useTPR bool) (*client, error) {
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

	// the API Group and version differ depending on if CRDs or TPRs are used.
	apiVersion := "dex.coreos.com/v1"
	if useTPR {
		apiVersion = "oidc.coreos.com/v1"
	}

	logger.Infof("kubernetes client apiVersion = %s", apiVersion)
	return &client{
		client: &http.Client{
			Transport: t,
			Timeout:   15 * time.Second,
		},
		baseURL:    cluster.Server,
		hash:       func() hash.Hash { return fnv.New64() },
		namespace:  namespace,
		apiVersion: apiVersion,
		logger:     logger,
	}, nil
}

type transport struct {
	updateReq func(r *http.Request)
	base      http.RoundTripper
}

func (t transport) RoundTrip(r *http.Request) (*http.Response, error) {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	t.updateReq(r2)
	return t.base.RoundTrip(r2)
}

func loadKubeConfig(kubeConfigPath string) (cluster k8sapi.Cluster, user k8sapi.AuthInfo, namespace string, err error) {
	data, err := ioutil.ReadFile(kubeConfigPath)
	if err != nil {
		err = fmt.Errorf("read %s: %v", kubeConfigPath, err)
		return
	}

	var c k8sapi.Config
	if err = yaml.Unmarshal(data, &c); err != nil {
		err = fmt.Errorf("unmarshal %s: %v", kubeConfigPath, err)
		return
	}

	cluster, user, namespace, err = currentContext(&c)
	if namespace == "" {
		namespace = "default"
	}
	return
}

func namespaceFromServiceAccountJWT(s string) (string, error) {
	// The service account token is just a JWT. Parse it as such.
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		// It's extremely important we don't log the actual service account token.
		return "", fmt.Errorf("malformed service account token: expected 3 parts got %d", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("malformed service account token: %v", err)
	}
	var data struct {
		// The claim Kubernetes uses to identify which namespace a service account belongs to.
		//
		// See: https://github.com/kubernetes/kubernetes/blob/v1.4.3/pkg/serviceaccount/jwt.go#L42
		Namespace string `json:"kubernetes.io/serviceaccount/namespace"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return "", fmt.Errorf("malformed service account token: %v", err)
	}
	if data.Namespace == "" {
		return "", errors.New(`jwt claim "kubernetes.io/serviceaccount/namespace" not found`)
	}
	return data.Namespace, nil
}

func inClusterConfig() (cluster k8sapi.Cluster, user k8sapi.AuthInfo, namespace string, err error) {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) == 0 || len(port) == 0 {
		err = fmt.Errorf("unable to load in-cluster configuration, KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined")
		return
	}
	cluster = k8sapi.Cluster{
		Server:               "https://" + host + ":" + port,
		CertificateAuthority: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
	}
	token, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return
	}
	user = k8sapi.AuthInfo{Token: string(token)}

	if namespace = os.Getenv("KUBERNETES_POD_NAMESPACE"); namespace == "" {
		namespace, err = namespaceFromServiceAccountJWT(user.Token)
		if err != nil {
			err = fmt.Errorf("failed to inspect service account token: %v", err)
			return
		}
	}

	return
}

func currentContext(config *k8sapi.Config) (cluster k8sapi.Cluster, user k8sapi.AuthInfo, ns string, err error) {
	if config.CurrentContext == "" {
		if len(config.Contexts) == 1 {
			config.CurrentContext = config.Contexts[0].Name
		} else {
			return cluster, user, "", errors.New("kubeconfig has no current context")
		}
	}
	context, ok := func() (k8sapi.Context, bool) {
		for _, namedContext := range config.Contexts {
			if namedContext.Name == config.CurrentContext {
				return namedContext.Context, true
			}
		}
		return k8sapi.Context{}, false
	}()
	if !ok {
		return cluster, user, "", fmt.Errorf("no context named %q found", config.CurrentContext)
	}

	cluster, ok = func() (k8sapi.Cluster, bool) {
		for _, namedCluster := range config.Clusters {
			if namedCluster.Name == context.Cluster {
				return namedCluster.Cluster, true
			}
		}
		return k8sapi.Cluster{}, false
	}()
	if !ok {
		return cluster, user, "", fmt.Errorf("no cluster named %q found", context.Cluster)
	}

	user, ok = func() (k8sapi.AuthInfo, bool) {
		for _, namedAuthInfo := range config.AuthInfos {
			if namedAuthInfo.Name == context.AuthInfo {
				return namedAuthInfo.AuthInfo, true
			}
		}
		return k8sapi.AuthInfo{}, false
	}()
	if !ok {
		return cluster, user, "", fmt.Errorf("no user named %q found", context.AuthInfo)
	}
	return cluster, user, context.Namespace, nil
}
