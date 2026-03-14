package server

import "fmt"

// TokenExchangePolicy defines per-client access control for ID-JAG token exchange.
type TokenExchangePolicy struct {
	// ClientID is the client this policy applies to. Use "*" for a default policy.
	ClientID         string   `json:"clientID"`
	AllowedAudiences []string `json:"allowedAudiences"`
	AllowedScopes    []string `json:"allowedScopes"`
}

// evaluateIDJAGPolicy checks whether the client is permitted to obtain an ID-JAG
// for the given audience and scopes. No policies configured means allow all.
func evaluateIDJAGPolicy(policies []TokenExchangePolicy, clientID, audience string, scopes []string) error {
	if len(policies) == 0 {
		return nil
	}

	// Find the most-specific policy for this client: exact match first, then wildcard.
	var matched *TokenExchangePolicy
	for i := range policies {
		p := &policies[i]
		if p.ClientID == clientID {
			matched = p
			break
		}
		if p.ClientID == "*" && matched == nil {
			matched = p
		}
	}

	if matched == nil {
		return fmt.Errorf("no policy found for client %q: access_denied", clientID)
	}

	// Check audience.
	if !audienceAllowed(matched.AllowedAudiences, audience) {
		return fmt.Errorf("audience %q is not allowed for client %q: access_denied", audience, clientID)
	}

	// Check scopes (only if policy restricts them).
	if len(matched.AllowedScopes) > 0 {
		for _, scope := range scopes {
			if !scopeAllowed(matched.AllowedScopes, scope) {
				return fmt.Errorf("scope %q is not allowed for client %q: access_denied", scope, clientID)
			}
		}
	}

	return nil
}

func audienceAllowed(allowedAudiences []string, audience string) bool {
	for _, a := range allowedAudiences {
		if a == audience {
			return true
		}
	}
	return false
}

func scopeAllowed(allowedScopes []string, scope string) bool {
	for _, s := range allowedScopes {
		if s == scope {
			return true
		}
	}
	return false
}
