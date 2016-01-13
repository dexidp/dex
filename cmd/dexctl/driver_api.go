package main

import (
	"errors"
	"net/http"

	"github.com/coreos/dex/connector"
	schema "github.com/coreos/dex/schema/workerschema"
	"github.com/coreos/go-oidc/oidc"
)

func newAPIDriver(pcfg oidc.ProviderConfig, creds oidc.ClientCredentials) (driver, error) {
	ccfg := oidc.ClientConfig{
		ProviderConfig: pcfg,
		Credentials:    creds,
	}
	oc, err := oidc.NewClient(ccfg)
	if err != nil {
		return nil, err
	}

	trans := &oidc.AuthenticatedTransport{
		TokenRefresher: &oidc.ClientCredsTokenRefresher{
			Issuer:     pcfg.Issuer.String(),
			OIDCClient: oc,
		},
		RoundTripper: http.DefaultTransport,
	}
	hc := &http.Client{Transport: trans}
	svc, err := schema.NewWithBasePath(hc, pcfg.Issuer.String())
	if err != nil {
		return nil, err
	}

	return &apiDriver{svc: svc}, nil
}

type apiDriver struct {
	svc *schema.Service
}

func (d *apiDriver) NewClient(meta oidc.ClientMetadata) (*oidc.ClientCredentials, error) {
	sc := &schema.Client{
		RedirectURIs: make([]string, len(meta.RedirectURIs)),
	}

	for i, u := range meta.RedirectURIs {
		sc.RedirectURIs[i] = u.String()
	}

	call := d.svc.Clients.Create(sc)
	scs, err := call.Do()
	if err != nil {
		return nil, err
	}

	creds := &oidc.ClientCredentials{
		ID:     scs.Id,
		Secret: scs.Secret,
	}

	return creds, nil
}

func (d *apiDriver) ConnectorConfigs() ([]connector.ConnectorConfig, error) {
	return nil, errors.New("unable to get connector configs from HTTP API")
}

func (d *apiDriver) SetConnectorConfigs(cfgs []connector.ConnectorConfig) error {
	return errors.New("unable to set connector configs through HTTP API")
}
