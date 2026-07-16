package connectors

import (
	"slices"

	"github.com/dexidp/dex/server/oauth2"
)

// ConnectorGrantTypes is the set of grant types that can be restricted per connector.
var ConnectorGrantTypes = map[string]bool{
	oauth2.GrantTypeAuthorizationCode: true,
	oauth2.GrantTypeRefreshToken:      true,
	oauth2.GrantTypeImplicit:          true,
	oauth2.GrantTypePassword:          true,
	oauth2.GrantTypeDeviceCode:        true,
	oauth2.GrantTypeTokenExchange:     true,
}

// GrantTypeAllowed reports whether grantType is allowed for a connector with the
// given configured grant types. If none are configured, all are allowed.
func GrantTypeAllowed(configuredTypes []string, grantType string) bool {
	return len(configuredTypes) == 0 || slices.Contains(configuredTypes, grantType)
}

// ConnectorAllowed reports whether connectorID is in a client's allowed
// connectors list. If the list is empty, all connectors are allowed.
func ConnectorAllowed(allowedConnectors []string, connectorID string) bool {
	if len(allowedConnectors) == 0 {
		return true
	}
	return slices.Contains(allowedConnectors, connectorID)
}
