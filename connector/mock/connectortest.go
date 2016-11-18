// Package mock implements connectors which help test various server components.
package mock

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/net/context"

	"github.com/coreos/dex/connector"
)

// NewCallbackConnector returns a mock connector which requires no user interaction. It always returns
// the same (fake) identity.
func NewCallbackConnector() connector.Connector {
	return callbackConnector{}
}

var (
	_ connector.CallbackConnector = callbackConnector{}

	_ connector.PasswordConnector = passwordConnector{}
)

type callbackConnector struct{}

func (m callbackConnector) Close() error { return nil }

func (m callbackConnector) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
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

func (m callbackConnector) HandleCallback(s connector.Scopes, r *http.Request) (connector.Identity, error) {
	var groups []string
	if s.Groups {
		groups = []string{"authors"}
	}

	return connector.Identity{
		UserID:        "0-385-28089-0",
		Username:      "Kilgore Trout",
		Email:         "kilgore@kilgore.trout",
		EmailVerified: true,
		Groups:        groups,
		ConnectorData: connectorData,
	}, nil
}

// CallbackConfig holds the configuration parameters for a connector which requires no interaction.
type CallbackConfig struct{}

// Open returns an authentication strategy which requires no user interaction.
func (c *CallbackConfig) Open() (connector.Connector, error) {
	return NewCallbackConnector(), nil
}

// PasswordConfig holds the configuration for a mock connector which prompts for the supplied
// username and password.
type PasswordConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Open returns an authentication strategy which prompts for a predefined username and password.
func (c *PasswordConfig) Open() (connector.Connector, error) {
	if c.Username == "" {
		return nil, errors.New("no username supplied")
	}
	if c.Password == "" {
		return nil, errors.New("no password supplied")
	}
	return &passwordConnector{c.Username, c.Password}, nil
}

type passwordConnector struct {
	username string
	password string
}

func (p passwordConnector) Close() error { return nil }

func (p passwordConnector) Login(ctx context.Context, s connector.Scopes, username, password string) (identity connector.Identity, validPassword bool, err error) {
	if username == p.username && password == p.password {
		return connector.Identity{
			UserID:        "0-385-28089-0",
			Username:      "Kilgore Trout",
			Email:         "kilgore@kilgore.trout",
			EmailVerified: true,
		}, true, nil
	}
	return identity, false, nil
}
