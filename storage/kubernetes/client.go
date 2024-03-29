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
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/ghodss/yaml"
	"golang.org/x/net/http2"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/kubernetes/k8sapi"
)

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
	// API version of the custom resource definitions.
	// Different Kubernetes version requires to create CRD in certain API. It will be discovered automatically on
	// storage opening.
	crdAPIVersion string

	// This is called once the client's Close method is called to signal goroutines,
	// such as the one creating third party resources, to stop.
	cancel context.CancelFunc
}

// idToName maps an arbitrary ID, such as an email or client ID to a Kubernetes object name.
func (cli *client) idToName(s string) string {
	return idToName(s, cli.hash)
}

// offlineTokenName maps two arbitrary IDs, to a single Kubernetes object name.
// This is used when more than one field is used to uniquely identify the object.
func (cli *client) offlineTokenName(userID string, connID string) string {
	return offlineTokenName(userID, connID, cli.hash)
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

// https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-subdomain-names
const kubeResourceMaxLen = 253

var kubeResourceNameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)

func (cli *client) urlForWithParams(
	apiVersion, namespace, resource, name string, params url.Values,
) (string, error) {
	basePath := "apis/"
	if apiVersion == "v1" {
		basePath = "api/"
	}

	if name != "" && (len(name) > kubeResourceMaxLen || !kubeResourceNameRegex.MatchString(name)) {
		// The actual name can be found in auth request or auth code objects and equals to the state value
		return "", fmt.Errorf(
			"invalid kubernetes resource name: must match the pattern %s and be no longer than %d characters",
			kubeResourceNameRegex.String(),
			kubeResourceMaxLen)
	}

	var p string
	if namespace != "" {
		p = path.Join(basePath, apiVersion, "namespaces", namespace, resource, name)
	} else {
		p = path.Join(basePath, apiVersion, resource, name)
	}

	encodedParams := params.Encode()
	paramsSuffix := ""
	if len(encodedParams) > 0 {
		paramsSuffix = "?" + encodedParams
	}

	if strings.HasSuffix(cli.baseURL, "/") {
		return cli.baseURL + p + paramsSuffix, nil
	}

	return cli.baseURL + "/" + p + paramsSuffix, nil
}

func (cli *client) urlFor(apiVersion, namespace, resource, name string) (string, error) {
	return cli.urlForWithParams(apiVersion, namespace, resource, name, url.Values{})
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

	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<15)) // 64 KiB
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
	io.Copy(io.Discard, r.Body)
	r.Body.Close()
}

func (cli *client) get(resource, name string, v interface{}) error {
	return cli.getResource(cli.apiVersion, cli.namespace, resource, name, v)
}

func (cli *client) getURL(url string, v interface{}) error {
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

func (cli *client) getResource(apiVersion, namespace, resource, name string, v interface{}) error {
	u, err := cli.urlFor(apiVersion, namespace, resource, name)
	if err != nil {
		return err
	}
	return cli.getURL(u, v)
}

func (cli *client) listN(resource string, v interface{}, n int) error { //nolint:unparam // In practice, n is the gcResultLimit constant.
	params := url.Values{}
	params.Add("limit", fmt.Sprintf("%d", n))
	u, err := cli.urlForWithParams(cli.apiVersion, cli.namespace, resource, "", params)
	if err != nil {
		return err
	}
	return cli.getURL(u, v)
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

	url, err := cli.urlFor(apiVersion, namespace, resource, "")
	if err != nil {
		return err
	}
	resp, err := cli.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer closeResp(resp)
	return checkHTTPErr(resp, http.StatusCreated)
}

func (cli *client) detectKubernetesVersion() error {
	var version struct{ GitVersion string }

	url := cli.baseURL + "/version"
	resp, err := cli.client.Get(url)
	if err != nil {
		return err
	}

	defer closeResp(resp)
	if err := checkHTTPErr(resp, http.StatusOK); err != nil {
		return err
	}

	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		return err
	}

	clusterVersion, err := semver.NewVersion(version.GitVersion)
	if err != nil {
		cli.logger.Warnf("cannot detect Kubernetes version (%s): %v", clusterVersion, err)
		return nil
	}

	if clusterVersion.LessThan(semver.MustParse("v1.16.0")) {
		cli.crdAPIVersion = legacyCRDAPIVersion
	}

	return nil
}

func (cli *client) delete(resource, name string) error {
	url, err := cli.urlFor(cli.apiVersion, cli.namespace, resource, name)
	if err != nil {
		return err
	}
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

	url, err := cli.urlFor(cli.apiVersion, cli.namespace, resource, name)
	if err != nil {
		return err
	}

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

// Copied from https://github.com/gtank/cryptopasta
func defaultTLSConfig() *tls.Config {
	return &tls.Config{
		// Avoids most of the memorably-named TLS attacks
		MinVersion: tls.VersionTLS12,
		// Causes servers to use Go's default ciphersuite preferences,
		// which are tuned to avoid attacks. Does nothing on clients.
		PreferServerCipherSuites: true,
		// Only use curves which have constant-time implementations
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
		},
	}
}

func newClient(cluster k8sapi.Cluster, user k8sapi.AuthInfo, namespace string, logger log.Logger, inCluster bool) (*client, error) {
	tlsConfig := defaultTLSConfig()
	data := func(b string, file string) ([]byte, error) {
		if b != "" {
			return base64.StdEncoding.DecodeString(b)
		}
		if file == "" {
			return nil, nil
		}
		return os.ReadFile(file)
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
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
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
	t = wrapRoundTripper(httpTransport, user, inCluster)

	apiVersion := "dex.coreos.com/v1"

	logger.Infof("kubernetes client apiVersion = %s", apiVersion)
	return &client{
		client: &http.Client{
			Transport: t,
			Timeout:   15 * time.Second,
		},
		baseURL:       cluster.Server,
		hash:          func() hash.Hash { return fnv.New64() },
		namespace:     namespace,
		apiVersion:    apiVersion,
		crdAPIVersion: crdAPIVersion,
		logger:        logger,
	}, nil
}

func loadKubeConfig(kubeConfigPath string) (cluster k8sapi.Cluster, user k8sapi.AuthInfo, namespace string, err error) {
	data, err := os.ReadFile(kubeConfigPath)
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

func namespaceFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func getInClusterConfigNamespace(token, namespaceENV, namespacePath string) (string, error) {
	namespace := os.Getenv(namespaceENV)
	if namespace != "" {
		return namespace, nil
	}

	namespace, err := namespaceFromServiceAccountJWT(token)
	if err == nil {
		return namespace, nil
	}

	err = fmt.Errorf("inspect service account token: %v", err)
	namespace, fileErr := namespaceFromFile(namespacePath)
	if fileErr == nil {
		return namespace, nil
	}

	return "", fmt.Errorf("%v: trying to get namespace from file: %v", err, fileErr)
}

func inClusterConfig() (k8sapi.Cluster, k8sapi.AuthInfo, string, error) {
	const (
		serviceAccountPath          = "/var/run/secrets/kubernetes.io/serviceaccount/"
		serviceAccountTokenPath     = serviceAccountPath + "token"
		serviceAccountCAPath        = serviceAccountPath + "ca.crt"
		serviceAccountNamespacePath = serviceAccountPath + "namespace"

		kubernetesServiceHostENV  = "KUBERNETES_SERVICE_HOST"
		kubernetesServicePortENV  = "KUBERNETES_SERVICE_PORT"
		kubernetesPodNamespaceENV = "KUBERNETES_POD_NAMESPACE"
	)

	host, port := os.Getenv(kubernetesServiceHostENV), os.Getenv(kubernetesServicePortENV)
	if len(host) == 0 || len(port) == 0 {
		return k8sapi.Cluster{}, k8sapi.AuthInfo{}, "", fmt.Errorf(
			"unable to load in-cluster configuration, %s and %s must be defined",
			kubernetesServiceHostENV,
			kubernetesServicePortENV,
		)
	}
	// we need to wrap IPv6 addresses in square brackets
	// IPv4 also works with square brackets
	host = "[" + host + "]"
	cluster := k8sapi.Cluster{
		Server:               "https://" + host + ":" + port,
		CertificateAuthority: serviceAccountCAPath,
	}

	token, err := os.ReadFile(serviceAccountTokenPath)
	if err != nil {
		return cluster, k8sapi.AuthInfo{}, "", err
	}

	user := k8sapi.AuthInfo{Token: string(token)}

	namespace, err := getInClusterConfigNamespace(user.Token, kubernetesPodNamespaceENV, serviceAccountNamespacePath)
	if err != nil {
		return cluster, user, "", err
	}

	return cluster, user, namespace, nil
}

func currentContext(config *k8sapi.Config) (cluster k8sapi.Cluster, user k8sapi.AuthInfo, ns string, err error) {
	if config.CurrentContext == "" {
		if len(config.Contexts) == 1 {
			config.CurrentContext = config.Contexts[0].Name
		} else {
			return cluster, user, "", errors.New("kubeconfig has no current context")
		}
	}
	k8sContext, ok := func() (k8sapi.Context, bool) {
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
			if namedCluster.Name == k8sContext.Cluster {
				return namedCluster.Cluster, true
			}
		}
		return k8sapi.Cluster{}, false
	}()
	if !ok {
		return cluster, user, "", fmt.Errorf("no cluster named %q found", k8sContext.Cluster)
	}

	user, ok = func() (k8sapi.AuthInfo, bool) {
		for _, namedAuthInfo := range config.AuthInfos {
			if namedAuthInfo.Name == k8sContext.AuthInfo {
				return namedAuthInfo.AuthInfo, true
			}
		}
		return k8sapi.AuthInfo{}, false
	}()
	if !ok {
		return cluster, user, "", fmt.Errorf("no user named %q found", k8sContext.AuthInfo)
	}
	return cluster, user, k8sContext.Namespace, nil
}
