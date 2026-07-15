package tokens

import "strings"

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
