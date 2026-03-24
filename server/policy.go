package server

// PolicyDenialReason categorizes why an ID-JAG policy check failed.
type PolicyDenialReason string

const (
	PolicyDenialClientHasNoPolicy  PolicyDenialReason = "client_has_no_policy"
	PolicyDenialAudienceNotAllowed PolicyDenialReason = "audience_not_allowed"
)

// PolicyResult holds the outcome of an ID-JAG policy evaluation.
type PolicyResult struct {
	Denied       bool
	DenialReason PolicyDenialReason
	// GrantedScopes is the set of scopes that passed policy evaluation.
	// May be smaller than the requested scopes if policy restricts them.
	GrantedScopes []string
}

// TokenExchangePolicy defines per-client access control for ID-JAG token exchange.
type TokenExchangePolicy struct {
	// ClientID is the client this policy applies to. Use "*" for a default policy.
	ClientID         string   `json:"clientID"`
	AllowedAudiences []string `json:"allowedAudiences"`
	AllowedScopes    []string `json:"allowedScopes"`
}

// evaluateIDJAGPolicy checks whether the client is permitted to obtain an ID-JAG
// for the given audience and scopes. Clients without a matching policy are denied
// by default (default-deny).
func evaluateIDJAGPolicy(policies []TokenExchangePolicy, clientID, audience string, scopes []string) (PolicyResult, error) {
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
		return PolicyResult{
			Denied:       true,
			DenialReason: PolicyDenialClientHasNoPolicy,
		}, nil
	}

	// Check audience.
	if !audienceAllowed(matched.AllowedAudiences, audience) {
		return PolicyResult{
			Denied:       true,
			DenialReason: PolicyDenialAudienceNotAllowed,
		}, nil
	}

	// Filter scopes: if the policy restricts scopes, only grant those that are allowed.
	grantedScopes := scopes
	if len(matched.AllowedScopes) > 0 && len(scopes) > 0 {
		var filtered []string
		for _, scope := range scopes {
			if scopeAllowed(matched.AllowedScopes, scope) {
				filtered = append(filtered, scope)
			}
		}
		grantedScopes = filtered
	}

	return PolicyResult{
		Denied:        false,
		GrantedScopes: grantedScopes,
	}, nil
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
