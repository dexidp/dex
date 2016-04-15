package client

import (
	"encoding/base64"
	"net/url"
	"strings"
	"testing"

	"github.com/coreos/go-oidc/oidc"
	"github.com/kylelemons/godebug/pretty"
)

var (
	goodSecret1 = base64.URLEncoding.EncodeToString([]byte("my_secret"))
	goodSecret2 = base64.URLEncoding.EncodeToString([]byte("my_other_secret"))

	goodClient1 = `{ 
  "id": "my_id",
  "secret": "` + goodSecret1 + `",
  "redirectURLs": ["https://client.example.com"]
}`

	goodClient2 = `{ 
  "id": "my_other_id",
  "secret": "` + goodSecret2 + `",
  "redirectURLs": ["https://client2.example.com","https://client2_a.example.com"]
}`

	badURLClient = `{ 
  "id": "my_id",
  "secret": "` + goodSecret1 + `",
  "redirectURLs": ["hdtp:/\(bad)(u)(r)(l)"]
}`

	badSecretClient = `{ 
  "id": "my_id",
  "secret": "` + "****" + `",
  "redirectURLs": ["https://client.example.com"]
}`

	noSecretClient = `{ 
  "id": "my_id",
  "redirectURLs": ["https://client.example.com"]
}`
	noIDClient = `{ 
  "secret": "` + goodSecret1 + `",
  "redirectURLs": ["https://client.example.com"]
}`
)

func TestClientsFromReader(t *testing.T) {
	tests := []struct {
		json    string
		want    []Client
		wantErr bool
	}{
		{
			json: "[]",
			want: []Client{},
		},
		{
			json: "[" + goodClient1 + "]",
			want: []Client{
				{
					Credentials: oidc.ClientCredentials{
						ID:     "my_id",
						Secret: "my_secret",
					},
					Metadata: oidc.ClientMetadata{
						RedirectURIs: []url.URL{
							mustParseURL(t, "https://client.example.com"),
						},
					},
				},
			},
		},
		{
			json: "[" + strings.Join([]string{goodClient1, goodClient2}, ",") + "]",
			want: []Client{
				{
					Credentials: oidc.ClientCredentials{
						ID:     "my_id",
						Secret: "my_secret",
					},
					Metadata: oidc.ClientMetadata{
						RedirectURIs: []url.URL{
							mustParseURL(t, "https://client.example.com"),
						},
					},
				},
				{
					Credentials: oidc.ClientCredentials{
						ID:     "my_other_id",
						Secret: "my_other_secret",
					},
					Metadata: oidc.ClientMetadata{
						RedirectURIs: []url.URL{
							mustParseURL(t, "https://client2.example.com"),
							mustParseURL(t, "https://client2_a.example.com"),
						},
					},
				},
			},
		}, {
			json:    "[" + badURLClient + "]",
			wantErr: true,
		},
		{
			json:    "[" + badSecretClient + "]",
			wantErr: true,
		},
		{
			json:    "[" + noSecretClient + "]",
			wantErr: true,
		},
		{
			json:    "[" + noIDClient + "]",
			wantErr: true,
		},
	}

	for i, tt := range tests {
		r := strings.NewReader(tt.json)
		cs, err := ClientsFromReader(r)
		if tt.wantErr {
			if err == nil {
				t.Errorf("case %d: want non-nil err", i)
				t.Logf(pretty.Sprint(cs))
			}
			continue
		}
		if err != nil {
			t.Errorf("case %d: got unexpected error parsing clients: %v", i, err)
			t.Logf(tt.json)
		}

		if diff := pretty.Compare(tt.want, cs); diff != "" {
			t.Errorf("case %d: Compare(want, got): %v", i, diff)
		}
	}
}

func mustParseURL(t *testing.T, s string) url.URL {
	u, err := url.Parse(s)
	if err != nil {
		t.Fatalf("Cannot parse %v as url: %v", s, err)
	}
	return *u
}
