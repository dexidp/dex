package main

import (
	"context"
	"errors"
	"os"

	"github.com/dexidp/dex/connector"
	xplugin "github.com/dexidp/dex/connector/plugin"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

// Here is a real implementation of Greeter
type ExamplePasswordConnector struct {
	logger   hclog.Logger
	username string
	password string
}

func (g *ExamplePasswordConnector) Prompt() string {
	return "username"
}

func (g *ExamplePasswordConnector) Login(ctx context.Context, s connector.Scopes, username, password string) (identity connector.Identity, validPassword bool, err error) {
	g.logger.Debug("message from ExamplePasswordConnector.Login")
	// This is a placeholder for actual login logic.
	if username == g.username && password == g.password {
		return connector.Identity{UserID: g.username, Username: g.username}, true, nil
	}

	return connector.Identity{}, false, nil
}

func (g *ExamplePasswordConnector) Config(config map[string]any) error {
	username, ok := config["username"].(string)
	if !ok || username == "" {
		return errors.New("invalid or missing 'username' in config")
	}
	g.username = username

	password, ok := config["password"].(string)
	if !ok || password == "" {
		return errors.New("invalid or missing 'password' in config")
	}
	g.password = password

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

	connector := &ExamplePasswordConnector{
		logger: logger,
	}

	pluginMap := map[string]plugin.Plugin{
		"password": &xplugin.PasswordConnectorPlugin{Impl: connector},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}
