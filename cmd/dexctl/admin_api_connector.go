package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"time"

	"fmt"
	"github.com/coreos/dex/client"
	"github.com/coreos/dex/schema/adminschema"
	"github.com/coreos/go-oidc/oidc"
)

type AdminAPIConnector struct {
	client  *http.Client
	baseURL string
	apiKey  string
}

func newAdminAPIConnector(adminApiURL string, adminAPIKey string, caFile string) (*AdminAPIConnector, error) {
	adminAPIConnector := &AdminAPIConnector{baseURL: adminApiURL}
	var transport *http.Transport
	if len(caFile) != 0 {
		caCert, err := ioutil.ReadFile(caFile)
		if err != nil {
			return adminAPIConnector, err
		}
		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM(caCert)
		if !ok {
			return adminAPIConnector, fmt.Errorf("Error in adding certs")
		}

		tlsConfig := &tls.Config{
			RootCAs: caCertPool,
		}
		transport = &http.Transport{TLSClientConfig: tlsConfig}
	} else {
		transport = http.DefaultTransport.(*http.Transport)
	}
	adminAPIConnector.client = &http.Client{
		Transport: newAdminAPITransport(adminAPIKey, transport),
		Timeout:   10 * time.Second,
	}
	return adminAPIConnector, nil
}

func (d *AdminAPIConnector) NewClient(meta oidc.ClientMetadata) (*oidc.ClientCredentials, error) {
	if err := meta.Valid(); err != nil {
		return nil, err
	}
	cli := client.Client{
		Metadata: meta,
	}
	credential := &oidc.ClientCredentials{}
	service, err := adminschema.NewWithBasePath(d.client, d.baseURL)
	if err != nil {
		return credential, err
	}
	urls := make([]string, len(cli.Metadata.RedirectURIs))
	for i, url := range cli.Metadata.RedirectURIs {
		urls[i] = url.String()
	}
	c := &adminschema.Client{RedirectURIs: urls}
	createClientRequest := &adminschema.ClientCreateRequest{Client: c}
	response, err := service.Client.Create(createClientRequest).Do()
	if err != nil {
		return credential, err
	}
	credential.ID = response.Client.Id
	credential.Secret = response.Client.Secret
	return credential, nil
}

func (d *AdminAPIConnector) ConnectorConfigs() ([]interface{}, error) {
	var connectors []interface{}
	service, err := adminschema.NewWithBasePath(d.client, d.baseURL)
	if err != nil {
		return connectors, err
	}
	response, err := service.Connectors.Get().Do()
	if err != nil {
		return connectors, err
	}
	connectors = response.Connectors
	return connectors, nil
}

func (d *AdminAPIConnector) SetConnectorConfigs(cfgs []interface{}) error {
	fmt.Println("Connectors ", cfgs)
	service, err := adminschema.NewWithBasePath(d.client, d.baseURL)
	if err != nil {
		return err
	}
	connectorAddRequest := &adminschema.ConnectorsSetRequest{Connectors: cfgs}
	if err := service.Connectors.Set(connectorAddRequest).Do(); err != nil {
		return err
	}
	return nil
}

type adminAPITransport struct {
	secret string
	rt     http.RoundTripper
}

func newAdminAPITransport(apiSecret string, roundTripper http.RoundTripper) adminAPITransport {
	adminTransport := adminAPITransport{secret: apiSecret, rt: roundTripper}
	return adminTransport
}
func (a adminAPITransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if len(r.Header.Get("Authorization")) != 0 {
		return a.rt.RoundTrip(r)
	}
	// It is not using a bearer token syntax, just setting the header directly.
	r.Header.Set("Authorization", a.secret)
	return a.rt.RoundTrip(r)
}
