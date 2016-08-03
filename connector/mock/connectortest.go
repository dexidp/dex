// Package mock implements a mock connector which requires no user interaction.
package mock

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/coreos/poke/connector"
)

// New returns a mock connector which requires no user interaction. It always returns
// the same (fake) identity.
func New() connector.Connector {
	return mockConnector{}
}

var (
	_ connector.CallbackConnector = mockConnector{}
	_ connector.GroupsConnector   = mockConnector{}
)

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

var connectorData = []byte("foobar")

func (m mockConnector) HandleCallback(r *http.Request) (connector.Identity, string, error) {
	return connector.Identity{
		UserID:        "0-385-28089-0",
		Username:      "Kilgore Trout",
		Email:         "kilgore@kilgore.trout",
		EmailVerified: true,
		ConnectorData: connectorData,
	}, r.URL.Query().Get("state"), nil
}

func (m mockConnector) Groups(identity connector.Identity) ([]string, error) {
	if !bytes.Equal(identity.ConnectorData, connectorData) {
		return nil, errors.New("connector data mismatch")
	}
	return []string{"authors"}, nil
}

// Config holds the configuration parameters for the mock connector.
type Config struct{}

// Open returns an authentication strategy which requires no user interaction.
func (c *Config) Open() (connector.Connector, error) {
	return New(), nil
}
