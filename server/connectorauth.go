package server

// This file handles connector authorization for the token grants that still live
// in the server package: deciding whether a given client may use a given
// connector, and whether a connector permits a given grant type.
//
// INVARIANT: every token grant that resolves a connector MUST enforce both
// policies before issuing tokens:
//   1. connectors.ConnectorAllowed(client.AllowedConnectors, connID) — the connector
//      is in the client's AllowedConnectors.
//   2. connectors.GrantTypeAllowed(conn.GrantTypes, grantType) — the connector permits
//      this grant type.
//
// Grants that have moved to the grants package enforce the invariant through
// grants.Endpoint.resolveConnector, which runs both checks in one call. The
// refresh grant, still here, resolves the connector and runs #2 inside
// getRefreshTokenFromStorage (shared with token introspection), then runs #1 via
// checkConnectorAllowed in handleRefreshToken — a different order, but both still
// gate token issuance.
//
// Browser/auth-code paths enforce the same policies with their own HTML/redirect
// error surface (login.go, authorize.go).

import (
	"net/http"

	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/oauth2"
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

// checkConnectorAllowed writes an OAuth2 token error and returns false if connID is
// not in the client's AllowedConnectors. Token grant handlers call it before
// resolving the connector, so a disallowed connector is rejected without being
// instantiated and no grant can silently forget the check. The connector's own
// grant-type policy (GrantTypes) is enforced separately, right after the connector
// is resolved, because it needs the resolved connector and a grant-specific message.
// Browser/auth-code paths enforce the same AllowedConnectors policy with their own
// HTML/redirect error surface.
func (s *Server) checkConnectorAllowed(w http.ResponseWriter, r *http.Request, client storage.Client, connID string) bool {
	if connectors.ConnectorAllowed(client.AllowedConnectors, connID) {
		return true
	}
	s.logger.WarnContext(r.Context(), "connector not allowed for client",
		"client_id", client.ID, "connector_id", connID)
	s.tokenErrHelper(w, oauth2.InvalidGrant, "Connector not allowed for this client.", http.StatusBadRequest)
	return false
}
