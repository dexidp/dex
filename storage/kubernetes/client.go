package kubernetes

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gtank/cryptopasta"
	"golang.org/x/net/context"
	yaml "gopkg.in/yaml.v2"

	"github.com/coreos/dex/storage"
	"github.com/coreos/dex/storage/kubernetes/k8sapi"
)

type client struct {
	client     *http.Client
	baseURL    string
	namespace  string
	apiVersion string

	now func() time.Time

	// If not nil, the cancel function for stopping garbage colletion.
	cancel context.CancelFunc

	// BUG: currently each third party API group can only have one resource in it,
	// so for each resource this storage uses, it need a unique API group.
	//
	// Prepend the name of each resource to the API group for a predictable mapping.
	//
	// See: https://github.com/kubernetes/kubernetes/pull/28414
	prependResourceNameToAPIGroup bool
}

func (c *client) apiVersionForResource(resource string) string {
	if !c.prependResourceNameToAPIGroup {
		return c.apiVersion
	}
	return resource + "." + c.apiVersion
}

func (c *client) urlFor(apiVersion, namespace, resource, name string) string {
	basePath := "apis/"
	if apiVersion == "v1" {
		basePath = "api/"
	}

	if c.prependResourceNameToAPIGroup && apiVersion != "" && resource != "" {
		apiVersion = resource + "." + apiVersion
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

type httpErr struct {
	method string
	url    string
	status string
	body   []byte
}

func (e *httpErr) Error() string {
	return fmt.Sprintf("%s %s %s: response from server \"%s\"", e.method, e.url, e.status, bytes.TrimSpace(e.body))
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

	var url, method string
	if r.Request != nil {
		method = r.Request.Method
		url = r.Request.URL.String()
	}
	err = &httpErr{method, url, r.Status, body}
	log.Printf("%s", err)

	if r.StatusCode == http.StatusNotFound {
		return storage.ErrNotFound
	}
	return err
}

// Close the response body. The initial request is drained so the connection can
// be reused.
func closeResp(r *http.Response) {
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()
}

func (c *client) get(resource, name string, v interface{}) error {
	url := c.urlFor(c.apiVersion, c.namespace, resource, name)
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
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal object: %v", err)
	}

	url := c.urlFor(c.apiVersion, c.namespace, resource, "")
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

func newClient(cluster k8sapi.Cluster, user k8sapi.AuthInfo, namespace string) (*client, error) {
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

	var t http.RoundTripper = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSClientConfig:       tlsConfig,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

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

	// TODO(ericchiang): make API Group and version configurable.
	return &client{
		client:     &http.Client{Transport: t},
		baseURL:    cluster.Server,
		namespace:  namespace,
		apiVersion: "oidc.coreos.com/v1",
		now:        time.Now,
		prependResourceNameToAPIGroup: true,
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

	if namespace = os.Getenv("KUBERNETES_POD_NAMESPACE"); namespace == "" {
		err = fmt.Errorf("unable to load in-cluster configuration, KUBERNETES_POD_NAMESPACE must be defined")
		return
	}

	token, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return
	}
	user = k8sapi.AuthInfo{Token: string(token)}
	return
}

func currentContext(config *k8sapi.Config) (cluster k8sapi.Cluster, user k8sapi.AuthInfo, ns string, err error) {
	if config.CurrentContext == "" {
		return cluster, user, "", errors.New("kubeconfig has no current context")
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

func newInClusterClient() (*client, error) {
	return nil, nil
}
