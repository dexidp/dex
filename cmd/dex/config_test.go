package main

import (
	"testing"

	"github.com/coreos/dex/storage"
	"github.com/kylelemons/godebug/pretty"

	yaml "gopkg.in/yaml.v2"
)

func TestUnmarshalClients(t *testing.T) {
	data := `staticClients:
- id: example-app
  redirectURIs:
  - 'http://127.0.0.1:5555/callback'
  name: 'Example App'
  secret: ZXhhbXBsZS1hcHAtc2VjcmV0
`
	var c Config
	if err := yaml.Unmarshal([]byte(data), &c); err != nil {
		t.Fatal(err)
	}

	wantClients := []storage.Client{
		{
			ID:     "example-app",
			Name:   "Example App",
			Secret: "ZXhhbXBsZS1hcHAtc2VjcmV0",
			RedirectURIs: []string{
				"http://127.0.0.1:5555/callback",
			},
		},
	}

	if diff := pretty.Compare(wantClients, c.StaticClients); diff != "" {
		t.Errorf("did not get expected clients: %s", diff)
	}
}
