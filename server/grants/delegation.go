package grants

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/tokens"
)

// DelegationPolicy enables RFC 8693 delegation on the token-exchange grant for
// one client: an actor token is verified by one of
// ActorConnectors, and the issued token carries the actor in its "act" claim.
type DelegationPolicy struct {
	ClientID        string
	ActorConnectors []string
	// AllowedAudiences are the client IDs a delegated token may name through
	// the audience parameter, besides the client's own ID. Consumers outside
	// this list may not read the act claim, so a delegated token never reaches
	// them.
	AllowedAudiences []string
	// AllowedScopes are the scope values a delegated token may carry beyond
	// Dex's standard scopes.
	AllowedScopes []string
}

// standardScopes are the scopes any delegating client may request; others
// must be listed in its AllowedScopes.
var standardScopes = map[string]bool{
	tokens.ScopeOpenID:      true,
	tokens.ScopeEmail:       true,
	tokens.ScopeProfile:     true,
	tokens.ScopeGroups:      true,
	tokens.ScopeFederatedID: true,
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

// mayActInformational are may_act members that describe the authorized party
// without identifying it (RFC 8693 §4.4 gives email as an example).
var mayActInformational = map[string]bool{"email": true, "email_verified": true, "name": true, "preferred_username": true}

// mayActAllows applies Dex's may_act rule (RFC 8693 §4.4 leaves matching to
// the authorization server). may_act must carry sub or client_id, and iss only
// together with sub; any member other than those and the informational ones
// refuses, since Dex cannot tell whether it identifies the actor.
//
// In delegation, sub must equal the actor token's sub, iss its iss; without
// iss the client must have a single actor connector, so a sub from another
// issuer cannot match. client_id must equal the requesting client's ID.
//
// In impersonation (actor nil) the client becomes the actor: may_act must
// carry client_id equal to the requesting client and no sub, which only an
// actor token could prove.
func mayActAllows(mayAct any, actor map[string]any, clientID string, singleActorIssuer bool) bool {
	allowed, ok := mayAct.(map[string]any)
	if !ok {
		return false
	}
	for name, value := range allowed {
		switch {
		case mayActInformational[name]:
		case name == "sub" || name == "iss" || name == "client_id":
			if _, ok := value.(string); !ok {
				return false
			}
		default:
			return false
		}
	}
	sub, iss, cid := claimString(allowed, "sub"), claimString(allowed, "iss"), claimString(allowed, "client_id")
	if sub == "" && cid == "" || iss != "" && sub == "" {
		return false
	}
	if cid != "" && cid != clientID {
		return false
	}
	if actor == nil {
		return sub == "" && cid == clientID
	}
	if sub != "" {
		if sub != claimString(actor, "sub") {
			return false
		}
		if iss == "" && !singleActorIssuer || iss != "" && iss != claimString(actor, "iss") {
			return false
		}
	}
	return true
}

// priorActor decodes a delegated subject token's act claim, with any nested
// chain, so a new actor can be placed in front of it (RFC 8693 §4.1). It
// returns nil when the claim is absent and an error when it is present but has
// no sub at some level.
func priorActor(claim any) (*tokens.ActorClaims, error) {
	if claim == nil {
		return nil, nil
	}
	m, ok := claim.(map[string]any)
	if !ok || claimString(m, "sub") == "" {
		return nil, errors.New("malformed act claim")
	}
	inner, err := priorActor(m["act"])
	if err != nil {
		return nil, err
	}
	return &tokens.ActorClaims{Issuer: claimString(m, "iss"), Subject: claimString(m, "sub"), Act: inner}, nil
}

// delegatedScopes keeps the requested scopes the subject token allows, so a
// chain can only narrow. A subject with a scope claim keeps the requested
// scopes it lists. A delegated subject (one carrying act) without a scope
// claim grants none, since an earlier hop may have narrowed it to nothing. A
// subject that is neither leaves the request unchanged.
func delegatedScopes(requested []string, subjectScope any, delegatedSubject bool) []string {
	bound, ok := subjectScope.(string)
	if !ok {
		if delegatedSubject {
			return nil
		}
		return requested
	}
	allowed := strings.Fields(bound)
	var kept []string
	for _, scope := range requested {
		if slices.Contains(allowed, scope) {
			kept = append(kept, scope)
		}
	}
	return kept
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
