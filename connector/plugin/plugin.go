// Package plugin adds plugin authentication connectors.
package plugin

import (
	"errors"
	"log/slog"
	"os/exec"

	"github.com/dexidp/dex/connector"
	"github.com/hashicorp/go-plugin"
)

var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "DEX_PLUGIN",
	MagicCookieValue: "identityplugins",
}

var pluginMap = map[string]plugin.Plugin{
	"password": &PasswordConnectorPlugin{},
	"callback": &CallbackConnectorPlugin{},
}

type ConfigurableConnector interface {
	Config(config map[string]any) error
}

// Config holds configuration options for external plugins.
type Config struct {
	// Path to the binary of the plugin
	Path string `json:"path"`
	// Can be a 'password': PasswordConnector or 'callback': CallbackConnector
	Type string `json:"type"`
	// Config is not be validated by the main process,
	// but passed through as an arbitrary JSON object
	Config map[string]any `json:"config"`
	Client *plugin.Client
}

func (c *Config) Open(id string, logger *slog.Logger) (connector.Connector, error) {
	// Check if the requested connector type is registered in our plugin map.
	// This determines which plugin interface the dispensed plugin should implement.
	_, ok := pluginMap[c.Type]
	if !ok {
		// Log a detailed error if the connector type is explicitly not supported.
		logger.Error("unsupported plugin type requested", "requested_type", c.Type, "supported_types", pluginMap)
		return nil, errors.New("unsupported connector type: " + c.Type)
	}

	// Create a new plugin client to interact with the external process.
	c.Client = plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: handshakeConfig,      // Configuration for the handshake to establish communication.
		Plugins:         pluginMap,            // Map of plugin names to their implementations.
		Cmd:             exec.Command(c.Path), // The command to execute the plugin binary.
	})
	// It's crucial to kill the plugin process when we are done using it to avoid resource leaks.
	// defer client.Kill() TODO: This would kill on return, but we need it to live till after the return
	// because we don't have context in this part yet, we add the client to the struct and add a shutdown
	// method to the server.

	// Get the RPC client from the plugin process. This client is used to make calls to the plugin.
	rpcClient, err := c.Client.Client()
	if err != nil {
		logger.Error("failed to get RPC client from plugin", "error", err)
		return nil, err
	}

	// Dispense the plugin by its type. This requests the plugin process to return an instance
	// of the specified plugin interface.
	raw, err := rpcClient.Dispense(c.Type)
	if err != nil {
		logger.Error("failed to dispense plugin", "type", c.Type, "error", err)
		return nil, err
	}

	// Check if the dispensed plugin implements the ConfigurableConnector interface
	// and call its Config method if it does.
	if conn, ok := raw.(ConfigurableConnector); ok {
		err = conn.Config(c.Config)
		if err != nil {
			logger.Error("failed to configure plugin", "type", c.Type, "error", err)
			return nil, err
		}
	} else {
		logger.Warn("dispensed plugin does not implement config option", "raw_type", raw)
	}

	// Type assert the dispensed plugin to the expected connector interface based on its type.
	// This ensures that the plugin implements the necessary methods.
	switch c.Type {
	case "password":
		// For type "password", we expect the plugin to implement connector.PasswordConnector.
		conn, ok := raw.(connector.PasswordConnector)
		if !ok {
			logger.Error("dispensed plugin does not implement expected interface", "expected_interface", "connector.PasswordConnector", "raw_type", raw)
			return nil, errors.New("dispensed plugin is not a PasswordConnector")
		}
		return conn, nil
	case "callback":
		// For type "password", we expect the plugin to implement connector.PasswordConnector.
		conn, ok := raw.(connector.CallbackConnector)
		if !ok {
			logger.Error("dispensed plugin does not implement expected interface", "expected_interface", "connector.CallbackConnector", "raw_type", raw)
			return nil, errors.New("dispensed plugin is not a CallbackConnector")
		}
		return conn, nil
	default:
		// If the dispensed plugin type does not correspond to a known connector interface.
		logger.Error("unhandled dispensed plugin type", "type", c.Type)
		return nil, errors.New("unhandled dispensed plugin type: " + c.Type)
	}
}
