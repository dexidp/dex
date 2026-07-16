package server

// Connector authorization for the browser-facing /auth endpoint. The token-grant
// invariant (client allows the connector, connector allows the grant type) now
// lives entirely in grants.Endpoint, which enforces it for every token grant.
// Browser/auth-code paths enforce the same AllowedConnectors policy here, with
// their own HTML/redirect error surface (login.go, authorize.go).

import (
	"github.com/dexidp/dex/storage"
)

// filterConnectors filters the list of connectors by the allowed connector IDs.
// If allowedConnectors is empty, all connectors are returned (no filtering).
func filterConnectors(connectors []storage.Connector, allowedConnectors []string) []storage.Connector {
	if len(allowedConnectors) == 0 {
		return connectors
	}

	allowed := make(map[string]bool, len(allowedConnectors))
	for _, id := range allowedConnectors {
		allowed[id] = true
	}

	filtered := make([]storage.Connector, 0, len(connectors))
	for _, c := range connectors {
		if allowed[c.ID] {
			filtered = append(filtered, c)
		}
	}
	return filtered
}
