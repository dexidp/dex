package adminschema

import (
	"errors"
	"net/url"

	"github.com/coreos/dex/client"
	"github.com/coreos/go-oidc/oidc"
)

func MapSchemaClientToClient(sc Client) (client.Client, error) {
	c := client.Client{
		Credentials: oidc.ClientCredentials{
			ID:     sc.Id,
			Secret: sc.Secret,
		},
		Metadata: oidc.ClientMetadata{
			RedirectURIs: make([]url.URL, len(sc.RedirectURIs)),
		},
	}
	for i, ru := range sc.RedirectURIs {
		if ru == "" {
			return client.Client{}, errors.New("redirect URL empty")
		}

		u, err := url.Parse(ru)
		if err != nil {
			return client.Client{}, errors.New("redirect URL invalid")
		}

		c.Metadata.RedirectURIs[i] = *u
	}

	c.Metadata.ClientName = sc.ClientName

	if sc.LogoURI != "" {
		logoURI, err := url.Parse(sc.LogoURI)
		if err != nil {
			return client.Client{}, errors.New("logoURI invalid")
		}
		c.Metadata.LogoURI = logoURI
	}

	if sc.ClientURI != "" {
		clientURI, err := url.Parse(sc.ClientURI)
		if err != nil {
			return client.Client{}, errors.New("clientURI invalid")
		}
		c.Metadata.ClientURI = clientURI
	}
	return c, nil
}

func MapClientToSchemaClient(c client.Client) Client {
	cl := Client{
		Id:           c.Credentials.ID,
		Secret:       c.Credentials.Secret,
		RedirectURIs: make([]string, len(c.Metadata.RedirectURIs)),
	}
	for i, u := range c.Metadata.RedirectURIs {
		cl.RedirectURIs[i] = u.String()
	}

	cl.ClientName = c.Metadata.ClientName

	if c.Metadata.LogoURI != nil {
		cl.LogoURI = c.Metadata.LogoURI.String()
	}
	if c.Metadata.ClientURI != nil {
		cl.ClientURI = c.Metadata.ClientURI.String()
	}
	return cl
}
