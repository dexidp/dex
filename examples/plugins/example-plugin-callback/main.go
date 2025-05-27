package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/dexidp/dex/connector"
	xplugin "github.com/dexidp/dex/connector/plugin"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

// Here is a real implementation of Greeter
type ExampleCallbackConnector struct {
	logger   hclog.Logger
	Identity connector.Identity
}

func (m *ExampleCallbackConnector) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
	u, err := url.Parse(callbackURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse callbackURL %q: %v", callbackURL, err)
	}
	v := u.Query()
	v.Set("state", state)
	u.RawQuery = v.Encode()
	return u.String(), nil
}

// HandleCallback parses the request and returns the user's identity
func (m *ExampleCallbackConnector) HandleCallback(s connector.Scopes, r *http.Request) (connector.Identity, error) {
	return m.Identity, nil
}

// Refresh updates the identity during a refresh token request.
func (m *ExampleCallbackConnector) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	return m.Identity, nil
}

func (m *ExampleCallbackConnector) TokenIdentity(ctx context.Context, subjectTokenType, subjectToken string) (connector.Identity, error) {
	return m.Identity, nil
}

var connectorData = []byte("foobar")

func (m *ExampleCallbackConnector) Config(config map[string]any) error {
	m.Identity = connector.Identity{
		UserID:        "0-385-28089-0",
		Username:      "Kilgore Trout",
		Email:         "kilgore@kilgore.trout",
		EmailVerified: true,
		Groups:        []string{"authors"},
		ConnectorData: connectorData,
	}

	// Override fields from config if present
	if username, ok := config["username"].(string); ok && username != "" {
		m.Identity.Username = username
	}

	if email, ok := config["email"].(string); ok && email != "" {
		m.Identity.Email = email
	}

	if emailVerified, ok := config["emailVerified"].(bool); ok {
		m.Identity.EmailVerified = emailVerified
	}

	if groups, ok := config["groups"].([]any); ok {
		stringGroups := make([]string, len(groups))
		for i, v := range groups {
			if s, ok := v.(string); ok {
				stringGroups[i] = s
			} else {
				m.logger.Warn("ignoring non-string value in groups config", "value", v)
			}
		}
		m.Identity.Groups = stringGroups
	}

	return nil
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	handshakeConfig := plugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "DEX_PLUGIN",
		MagicCookieValue: "identityplugins",
	}

	connector := &ExampleCallbackConnector{
		logger: logger,
	}

	pluginMap := map[string]plugin.Plugin{
		"callback": &xplugin.CallbackConnectorPlugin{Impl: connector},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}
