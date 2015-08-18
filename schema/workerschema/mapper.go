package workerschema

import (
	"errors"
	"net/url"

	"github.com/coreos/go-oidc/oidc"
)

func MapSchemaClientToClientIdentity(sc Client) (oidc.ClientIdentity, error) {
	ci := oidc.ClientIdentity{
		Credentials: oidc.ClientCredentials{
			ID: sc.Id,
		},
		Metadata: oidc.ClientMetadata{
			RedirectURLs: make([]url.URL, len(sc.RedirectURIs)),
		},
	}

	for i, ru := range sc.RedirectURIs {
		if ru == "" {
			return oidc.ClientIdentity{}, errors.New("redirect URL empty")
		}

		u, err := url.Parse(ru)
		if err != nil {
			return oidc.ClientIdentity{}, errors.New("redirect URL invalid")
		}

		ci.Metadata.RedirectURLs[i] = *u
	}

	return ci, nil
}

func MapClientIdentityToSchemaClient(c oidc.ClientIdentity) Client {
	cl := Client{
		Id:           c.Credentials.ID,
		RedirectURIs: make([]string, len(c.Metadata.RedirectURLs)),
	}
	for i, u := range c.Metadata.RedirectURLs {
		cl.RedirectURIs[i] = u.String()
	}
	return cl
}

func MapClientIdentityToSchemaClientWithSecret(c oidc.ClientIdentity) ClientWithSecret {
	cl := ClientWithSecret{
		Id:           c.Credentials.ID,
		Secret:       c.Credentials.Secret,
		RedirectURIs: make([]string, len(c.Metadata.RedirectURLs)),
	}
	for i, u := range c.Metadata.RedirectURLs {
		cl.RedirectURIs[i] = u.String()
	}
	return cl
}
