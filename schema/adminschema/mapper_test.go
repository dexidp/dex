package adminschema

import (
	"net/url"
	"testing"

	"github.com/coreos/go-oidc/oidc"
	"github.com/kylelemons/godebug/pretty"

	"github.com/coreos/dex/client"
)

func TestMapSchemaClientToClient(t *testing.T) {
	tests := []struct {
		sc      Client
		want    client.Client
		wantErr bool
	}{
		{
			sc: Client{
				Id:     "123",
				Secret: "sec_123",
				RedirectURIs: []string{
					"https://client.example.com",
					"https://client2.example.com",
				},
				ClientName: "Bill",
				LogoURI:    "https://logo.example.com",
				ClientURI:  "https://clientURI.example.com",
			},
			want: client.Client{
				Credentials: oidc.ClientCredentials{
					ID:     "123",
					Secret: "sec_123",
				},
				Metadata: oidc.ClientMetadata{
					RedirectURIs: []url.URL{
						*mustParseURL(t, "https://client.example.com"),
						*mustParseURL(t, "https://client2.example.com"),
					},
					ClientName: "Bill",
					LogoURI:    mustParseURL(t, "https://logo.example.com"),
					ClientURI:  mustParseURL(t, "https://clientURI.example.com"),
				},
			},
		}, {
			sc: Client{
				Id:     "123",
				Secret: "sec_123",
				RedirectURIs: []string{
					"",
				},
			},
			wantErr: true,
		},
	}

	for i, tt := range tests {
		got, err := MapSchemaClientToClient(tt.sc)
		if tt.wantErr {
			if err == nil {
				t.Errorf("case %d: want non-nil error", i)
				t.Logf(pretty.Sprint(got))
			}
			continue
		}
		if err != nil {
			t.Errorf("case %d: unexpected error mapping: %v", i, err)
		}

		if diff := pretty.Compare(tt.want, got); diff != "" {
			t.Errorf("case %d: Compare(want, got): %v", i, diff)
		}

	}
}

func TestMapClientToClientSchema(t *testing.T) {
	tests := []struct {
		c    client.Client
		want Client
	}{
		{
			want: Client{
				Id:     "123",
				Secret: "sec_123",
				RedirectURIs: []string{
					"https://client.example.com",
					"https://client2.example.com",
				},
				ClientName: "Bill",
				LogoURI:    "https://logo.example.com",
				ClientURI:  "https://clientURI.example.com",
			},
			c: client.Client{
				Credentials: oidc.ClientCredentials{
					ID:     "123",
					Secret: "sec_123",
				},
				Metadata: oidc.ClientMetadata{
					RedirectURIs: []url.URL{
						*mustParseURL(t, "https://client.example.com"),
						*mustParseURL(t, "https://client2.example.com"),
					},
					ClientName: "Bill",
					LogoURI:    mustParseURL(t, "https://logo.example.com"),
					ClientURI:  mustParseURL(t, "https://clientURI.example.com"),
				},
			},
		},
	}

	for i, tt := range tests {
		got := MapClientToSchemaClient(tt.c)

		if diff := pretty.Compare(tt.want, got); diff != "" {
			t.Errorf("case %d: Compare(want, got): %v", i, diff)
		}

	}
}

func mustParseURL(t *testing.T, s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		t.Fatalf("Cannot parse %v as url: %v", s, err)
	}
	return u
}
