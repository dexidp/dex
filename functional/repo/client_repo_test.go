package repo

import (
	"encoding/base64"
	"net/url"

	"github.com/coreos/go-oidc/oidc"

	"github.com/coreos/dex/client"
)

var (
	testClients = []client.Client{
		client.Client{
			Credentials: oidc.ClientCredentials{
				ID:     "client1",
				Secret: base64.URLEncoding.EncodeToString([]byte("secret-1")),
			},
			Metadata: oidc.ClientMetadata{
				RedirectURIs: []url.URL{
					url.URL{
						Scheme: "https",
						Host:   "client1.example.com",
						Path:   "/callback",
					},
				},
			},
		},
		client.Client{
			Credentials: oidc.ClientCredentials{
				ID:     "client2",
				Secret: base64.URLEncoding.EncodeToString([]byte("secret-2")),
			},
			Metadata: oidc.ClientMetadata{
				RedirectURIs: []url.URL{
					url.URL{
						Scheme: "https",
						Host:   "client2.example.com",
						Path:   "/callback",
					},
				},
			},
		},
	}
)
