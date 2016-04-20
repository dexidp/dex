package workerschema

import (
	"errors"
	"net/url"

	"github.com/coreos/dex/client"
	"github.com/coreos/go-oidc/oidc"
)

func MapSchemaClientToClient(sc Client) (client.Client, error) {
	ci := client.Client{
		Credentials: oidc.ClientCredentials{
			ID: sc.Id,
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

		ci.Metadata.RedirectURIs[i] = *u
	}

	return ci, nil
}

func MapClientToSchemaClient(c client.Client) Client {
	cl := Client{
		Id:           c.Credentials.ID,
		RedirectURIs: make([]string, len(c.Metadata.RedirectURIs)),
	}
	for i, u := range c.Metadata.RedirectURIs {
		cl.RedirectURIs[i] = u.String()
	}
	return cl
}

func MapClientToSchemaClientWithSecret(c client.Client) ClientWithSecret {
	cl := ClientWithSecret{
		Id:           c.Credentials.ID,
		Secret:       c.Credentials.Secret,
		RedirectURIs: make([]string, len(c.Metadata.RedirectURIs)),
	}
	for i, u := range c.Metadata.RedirectURIs {
		cl.RedirectURIs[i] = u.String()
	}
	return cl
}
