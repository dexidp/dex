package server

import "fmt"

// TokenExchangePolicy defines access control rules for Token Exchange ID-JAG requests.
// It specifies which clients are permitted to obtain ID-JAGs for which audience URLs
// and with which scopes.
type TokenExchangePolicy struct {
	// ClientID is the client this policy applies to.
	// Use "*" to define a default policy for all clients.
	ClientID string `json:"clientID"`

	// AllowedAudiences is the list of Resource AS issuer URLs this client may request
	// ID-JAGs for. An empty list means no audiences are permitted.
	AllowedAudiences []string `json:"allowedAudiences"`

	// AllowedScopes restricts the scopes that may be requested in the ID-JAG.
	// If empty, all scopes are permitted for allowed audiences.
	AllowedScopes []string `json:"allowedScopes"`
}

// evaluateIDJAGPolicy checks whether a Token Exchange request for an ID-JAG is
// permitted by the configured policies.
//
// When no policies are configured, all requests are allowed (backward compatible).
// When policies are configured, the request is denied unless a matching policy
// explicitly allows the client/audience/scope combination.
func evaluateIDJAGPolicy(policies []TokenExchangePolicy, clientID, audience string, scopes []string) error {
	if len(policies) == 0 {
		// No policies configured: allow all (backward compatible).
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
