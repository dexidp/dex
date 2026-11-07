package connectors

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/storage"
)

// LocalConnector is the local passwordDB connector: an internal connector,
// backed by the password store, that is not part of the injected config map.
const LocalConnector = "local"

// ConnectorConfig is a configuration that can open a connector.
type ConnectorConfig interface {
	Open(id string, logger *slog.Logger) (connector.Connector, error)
}

// Resolver returns a ResolveFunc that builds the underlying implementation for a
// stored connector: the built-in local password DB (backed by storage), or a
// connector from the given config map. The map is injected by the caller so this
// package need not import any connector implementation — a library consumer can
// pass its own set of connectors.
func Resolver(store storage.Storage, logger *slog.Logger, configs map[string]func() ConnectorConfig) ResolveFunc {
	return func(conn storage.Connector) (connector.Connector, error) {
		if conn.Type == LocalConnector {
			return NewPasswordDB(store), nil
		}
		return openConnector(logger, configs, conn)
	}
}

// openConnector parses the stored config and opens the connector named by its type.
func openConnector(logger *slog.Logger, configs map[string]func() ConnectorConfig, conn storage.Connector) (connector.Connector, error) {
	var c connector.Connector

	f, ok := configs[conn.Type]
	if !ok {
		return c, fmt.Errorf("unknown connector type %q", conn.Type)
	}

	connConfig := f()
	if len(conn.Config) != 0 {
		if err := json.Unmarshal(conn.Config, connConfig); err != nil {
			return c, fmt.Errorf("parse connector config: %v", err)
		}
	}

	c, err := connConfig.Open(conn.ID, logger)
	if err != nil {
		return c, fmt.Errorf("failed to create connector %s: %v", conn.ID, err)
	}

	return c, nil
}
