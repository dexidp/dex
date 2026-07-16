package tokens

import (
	"strings"

	"github.com/dexidp/dex/connector"
)

// OIDC scope values. They all live here so scope handling stays in one place.
const (
	ScopeOpenID        = "openid"
	ScopeOfflineAccess = "offline_access" // Request a refresh token.
	ScopeGroups        = "groups"
	ScopeEmail         = "email"
	ScopeProfile       = "profile"
	ScopeFederatedID   = "federated:id"

	scopeCrossClientPrefix = "audience:server:client_id:"
)

// ParseCrossClientScope extracts the peer client ID from a cross-client audience
// scope, reporting whether the scope was one.
func ParseCrossClientScope(scope string) (string, bool) {
	if !strings.HasPrefix(scope, scopeCrossClientPrefix) {
		return "", false
	}
	return scope[len(scopeCrossClientPrefix):], true
}

// ParseScopes translates the requested OIDC scopes into the connector.Scopes a
// connector needs when authenticating or refreshing a user.
func ParseScopes(scopes []string) connector.Scopes {
	var s connector.Scopes
	for _, scope := range scopes {
		switch scope {
		case ScopeOfflineAccess:
			s.OfflineAccess = true
		case ScopeGroups:
			s.Groups = true
		}
	}
	return s
}
