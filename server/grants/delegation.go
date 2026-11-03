package grants

import (
	"fmt"
	"slices"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/internal"
)

// DelegationPolicy enables RFC 8693 delegation on the token-exchange grant for
// one client: the client must present an actor token, verified by one of
// ActorConnectors, and the issued token carries the actor in its "act" claim.
type DelegationPolicy struct {
	ClientID        string
	ActorConnectors []string
	// Audiences are the cross-client audiences a delegated token may carry.
	// Consumers outside this list may not read the act claim, so a delegated
	// token never reaches them.
	Audiences []string
}

// delegationPolicies indexes the configured policies by client and collects the
// set of actor connectors, which no client may use as a subject connector.
type delegationPolicies struct {
	byClient        map[string]DelegationPolicy
	actorConnectors map[string]bool
}

func newDelegationPolicies(policies []DelegationPolicy) delegationPolicies {
	p := delegationPolicies{
		byClient:        make(map[string]DelegationPolicy, len(policies)),
		actorConnectors: map[string]bool{},
	}
	for _, policy := range policies {
		p.byClient[policy.ClientID] = policy
		for _, id := range policy.ActorConnectors {
			p.actorConnectors[id] = true
		}
	}
	return p
}

// claimString returns a string claim, or "" when absent or not a string.
func claimString(claims map[string]any, name string) string {
	s, _ := claims[name].(string)
	return s
}

// claimTime returns a NumericDate claim (RFC 7519 §2) as a time, and false when
// absent or malformed.
func claimTime(claims map[string]any, name string) (time.Time, bool) {
	switch v := claims[name].(type) {
	case float64:
		return time.Unix(int64(v), 0), true
	case int64:
		return time.Unix(v, 0), true
	}
	return time.Time{}, false
}

// audienceContains reports whether an aud claim, a string or an array of
// strings (RFC 7519 §4.1.3), names aud.
func audienceContains(claim any, aud string) bool {
	switch v := claim.(type) {
	case string:
		return v == aud
	case []any:
		return slices.Contains(v, any(aud))
	case []string:
		return slices.Contains(v, aud)
	}
	return false
}

// mayActAllows applies RFC 8693 §4.4: the actor's sub must equal may_act's sub,
// and its iss must equal may_act's iss when may_act states one. A may_act that
// is not an object authorizes nobody.
func mayActAllows(mayAct any, actor map[string]any) bool {
	allowed, ok := mayAct.(map[string]any)
	if !ok {
		return false
	}
	if claimString(allowed, "sub") == "" || claimString(allowed, "sub") != claimString(actor, "sub") {
		return false
	}
	if iss := claimString(allowed, "iss"); iss != "" && iss != claimString(actor, "iss") {
		return false
	}
	return true
}

// identityFromClaims rebuilds the identity a Dex-issued token was minted from:
// the user ID and connector ID encoded in its sub, and the profile claims it
// carries. Issuing from this identity reproduces the token's sub.
func identityFromClaims(claims map[string]any) (connector.Identity, string, error) {
	var subject internal.IDTokenSubject
	if err := internal.Unmarshal(claimString(claims, "sub"), &subject); err != nil {
		return connector.Identity{}, "", fmt.Errorf("decode subject: %w", err)
	}
	identity := connector.Identity{
		UserID:            subject.UserId,
		Username:          claimString(claims, "name"),
		PreferredUsername: claimString(claims, "preferred_username"),
		Email:             claimString(claims, "email"),
	}
	identity.EmailVerified, _ = claims["email_verified"].(bool)
	if groups, ok := claims["groups"].([]any); ok {
		for _, g := range groups {
			if s, ok := g.(string); ok {
				identity.Groups = append(identity.Groups, s)
			}
		}
	}
	return identity, subject.ConnId, nil
}
