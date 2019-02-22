// Package mock implements connectors which help test various server components.
package mock

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/log"
)

// NewCallbackConnector returns a mock connector which requires no user interaction. It always returns
// the same (fake) identity.
func NewCallbackConnector(logger log.Logger) connector.Connector {
	return &Callback{
		Identity: connector.Identity{
			UserID:        "0-385-28089-0",
			Username:      "Kilgore Trout",
			Email:         "kilgore@kilgore.trout",
			EmailVerified: true,
			Groups:        []string{"authors"},
			ConnectorData: connectorData,
		},
		Logger: logger,
	}
}

var (
	_ connector.CallbackConnector = &Callback{}

	_ connector.PasswordConnector = passwordConnector{}
	_ connector.RefreshConnector  = passwordConnector{}
)

// Callback is a connector that requires no user interaction and always returns the same identity.
type Callback struct {
	// The returned identity.
	Identity connector.Identity
	Logger   log.Logger
}

// LoginURL returns the URL to redirect the user to login with.
func (m *Callback) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
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

// HandleCallback parses the request and returns the user's identity
func (m *Callback) HandleCallback(s connector.Scopes, r *http.Request) (connector.Identity, error) {
	return m.Identity, nil
}

// Refresh updates the identity during a refresh token request.
func (m *Callback) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	return m.Identity, nil
}

// CallbackConfig holds the configuration parameters for a connector which requires no interaction.
type CallbackConfig struct{}

// Open returns an authentication strategy which requires no user interaction.
func (c *CallbackConfig) Open(id string, logger log.Logger) (connector.Connector, error) {
	return NewCallbackConnector(logger), nil
}

// PasswordConfig holds the configuration for a mock connector which prompts for the supplied
// username and password.
type PasswordConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Open returns an authentication strategy which prompts for a predefined username and password.
func (c *PasswordConfig) Open(id string, logger log.Logger) (connector.Connector, error) {
	if c.Username == "" {
		return nil, errors.New("no username supplied")
	}
	if c.Password == "" {
		return nil, errors.New("no password supplied")
	}
	return &passwordConnector{c.Username, c.Password, logger}, nil
}

type passwordConnector struct {
	username string
	password string
	logger   log.Logger
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

func (p passwordConnector) Prompt() string { return "" }

func (p passwordConnector) Refresh(_ context.Context, _ connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	return identity, nil
}
