// Package mock implements a mock connector which requires no user interaction.
package mock

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/coreos/poke/connector"
	"github.com/coreos/poke/storage"
)

// New returns a mock connector which requires no user interaction. It always returns
// the same (fake) identity.
func New() connector.Connector {
	return mockConnector{}
}

type mockConnector struct{}

func (m mockConnector) Close() error { return nil }

func (m mockConnector) LoginURL(callbackURL, state string) (string, error) {
	u, err := url.Parse(callbackURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse callbackURL %q: %v", callbackURL, err)
	}
	v := u.Query()
	v.Set("state", state)
	u.RawQuery = v.Encode()
	return u.String(), nil
}

func (m mockConnector) HandleCallback(r *http.Request) (storage.Identity, string, error) {
	return storage.Identity{
		UserID:        "0-385-28089-0",
		Username:      "Kilgore Trout",
		Email:         "kilgore@kilgore.trout",
		EmailVerified: true,
	}, r.URL.Query().Get("state"), nil
}

func (m mockConnector) Groups(identity storage.Identity) ([]string, error) {
	return []string{"authors"}, nil
}

// Config holds the configuration parameters for the mock connector.
type Config struct{}

// Open returns an authentication strategy which requires no user interaction.
func (c *Config) Open() (connector.Connector, error) {
	return New(), nil
}
