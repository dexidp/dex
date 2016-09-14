package main

import (
	"net/http"
	"time"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/schema/adminschema"
	"github.com/coreos/go-oidc/oidc"
)

type AdminAPIDriver struct {
	client  *http.Client
	baseURL string
	apiKey  string
}

func newAdminAPIDriver(adminApiURL string, adminAPIKey string) (driver, error) {
	apiClient := &http.Client{
		Transport: adminAPITransport{secret: adminAPIKey},
		Timeout:   10 * time.Second,
	}
	adminAPIDriver := &AdminAPIDriver{
		baseURL: adminApiURL,
		client:  apiClient,
	}
	return adminAPIDriver, nil
}

func (d *AdminAPIDriver) NewClient(meta oidc.ClientMetadata) (*oidc.ClientCredentials, error) {
	if err := meta.Valid(); err != nil {
		return nil, err
	}
	cli := client.Client{
		Metadata: meta,
	}
	credential := &oidc.ClientCredentials{}
	service, err := adminschema.NewWithBasePath(d.client, d.baseURL)
	if err != nil {
		return credential, nil
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

func (d *AdminAPIDriver) ConnectorConfigs() ([]interface{}, error) {
	var connectors []interface{}
	service, err := adminschema.NewWithBasePath(d.client, d.baseURL)
	if err != nil {
		return connectors, nil
	}
	response, err := service.Connectors.Get().Do()
	if err != nil {
		return connectors, err
	}
	connectors = response.Connectors
	return connectors, nil
}

func (d *AdminAPIDriver) SetConnectorConfigs(cfgs []interface{}) error {
	service, err := adminschema.NewWithBasePath(d.client, d.baseURL)
	if err != nil {
		return nil
	}
	connectorAddRequest := &adminschema.ConnectorsSetRequest{Connectors: cfgs}
	if err := service.Connectors.Set(connectorAddRequest).Do(); err != nil {
		return err
	}
	return nil
}

type adminAPITransport struct {
	secret string
}

func (a adminAPITransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Authorization", a.secret)
	return http.DefaultTransport.RoundTrip(r)
}
