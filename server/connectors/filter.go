package connectors

import "github.com/dexidp/dex/storage"

// Filter returns the connectors allowed for a client. When allowedConnectors is
// empty the list is returned unfiltered. It is the browser auth flow's counterpart
// to ConnectorAllowed (which checks a single id).
func Filter(conns []storage.Connector, allowedConnectors []string) []storage.Connector {
	if len(allowedConnectors) == 0 {
		return conns
	}

	allowed := make(map[string]bool, len(allowedConnectors))
	for _, id := range allowedConnectors {
		allowed[id] = true
	}

	filtered := make([]storage.Connector, 0, len(conns))
	for _, c := range conns {
		if allowed[c.ID] {
			filtered = append(filtered, c)
		}
	}
	return filtered
}
