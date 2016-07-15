package scope

import "strings"

const (
	// Scope prefix which indicates initiation of a cross-client authentication flow.
	// See https://developers.google.com/identity/protocols/CrossClientAuth
	ScopeGoogleCrossClient = "audience:server:client_id:"

	// ScopeGroups indicates that groups should be added to the ID Token.
	ScopeGroups = "groups"
)

type Scopes []string

func (s Scopes) OfflineAccess() bool {
	return s.HasScope("offline_access")
}

func (s Scopes) HasScope(scope string) bool {
	for _, curScope := range s {
		if curScope == scope {
			return true
		}
	}
	return false
}

func (s Scopes) CrossClientIDs() []string {
	clients := []string{}
	for _, scope := range s {
		if strings.HasPrefix(scope, ScopeGoogleCrossClient) {
			clients = append(clients, scope[len(ScopeGoogleCrossClient):])
		}
	}
	return clients
}

func (s Scopes) Contains(other Scopes) bool {
	rScopes := map[string]struct{}{}
	for _, scope := range s {
		rScopes[scope] = struct{}{}
	}

	for _, scope := range other {
		if _, ok := rScopes[scope]; !ok {
			if scope == "" {
				continue
			}
			return false
		}
	}
	return true
}
