package grants

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// tokenExchange serves the RFC 8693 token-exchange grant: a subject token
// (ID or access token) verified by a connector is exchanged for a new token.
// Its response carries a single requested token plus issued_token_type, so it
// builds its own response from the issuer primitives instead of the standard
// Issue mint.
//
// A client with a delegation policy must also present an actor token; the
// issued token then carries the actor in its "act" claim (RFC 8693 §4.1).
type tokenExchange struct {
	issuer     *tokens.Issuer
	logger     *slog.Logger
	delegation delegationPolicies
	// resolveConnector is the Handler's connector resolution, reused for the
	// actor connector so the client's allowedConnectors and the connector's
	// grantTypes apply to it as they do to the subject connector.
	resolveConnector func(ctx context.Context, connID string, client storage.Client, grantType string) (connectors.Connector, *oauth2.Error)
}

func (g *tokenExchange) GrantType() string {
	return oauth2.GrantTypeTokenExchange
}

func (g *tokenExchange) RequiresClientAuth() bool {
	return true
}

// Scopes are passed through: for token exchange the requested scope maps to the
// issued token's scope and is not validated against a fixed set.
func (g *tokenExchange) ScopePolicy() ScopePolicy {
	return ScopePolicy{}
}

// ConnectorID reads the required connector_id parameter (an RFC 8693 extension).
func (g *tokenExchange) ConnectorID(ctx context.Context, req *Request, client storage.Client) (string, *oauth2.Error) {
	return req.ConnectorID, nil
}

func (g *tokenExchange) Authorize(ctx context.Context, req *Request, client storage.Client, conn connectors.Connector) (Responder, error) {
	switch req.SubjectTokenType {
	case oauth2.TokenTypeID, oauth2.TokenTypeAccess: // ok, continue
	default:
		return nil, &oauth2.Error{Type: oauth2.RequestNotSupported, Description: "Invalid subject_token_type.", Status: http.StatusBadRequest}
	}
	if req.SubjectToken == "" {
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Missing subject_token", Status: http.StatusBadRequest}
	}
	if (req.ActorToken == "") != (req.ActorTokenType == "") {
		return nil, invalidRequest("actor_token and actor_token_type must be sent together")
	}
	if g.delegation.actorConnectors[req.ConnectorID] {
		return nil, invalidRequest("connector only verifies actor tokens")
	}

	teConn, ok := conn.Connector.(connector.TokenIdentityConnector)
	if !ok {
		g.logger.ErrorContext(ctx, "connector doesn't implement token exchange", "connector_id", req.ConnectorID)
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Requested connector does not exist.", Status: http.StatusBadRequest}
	}

	policy, delegating := g.delegation.byClient[client.ID]
	if !delegating || client.Public {
		if req.ActorToken != "" {
			return nil, invalidRequest("delegation is not enabled for this client")
		}
		return g.impersonate(ctx, req, client, teConn)
	}
	return g.delegate(ctx, req, client, conn, policy)
}

// impersonate is the exchange without an actor: the issued token says nothing
// about who presented the subject token.
func (g *tokenExchange) impersonate(ctx context.Context, req *Request, client storage.Client, teConn connector.TokenIdentityConnector) (Responder, error) {
	identity, err := teConn.TokenIdentity(ctx, req.SubjectTokenType, req.SubjectToken)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to verify subject token", "err", err)
		return nil, &oauth2.Error{Type: oauth2.AccessDenied, Status: http.StatusUnauthorized}
	}

	reqType := requestedTokenType(req)
	g.logger.InfoContext(ctx, "token exchange successful", append(identityLogAttrs(req, client, identity),
		"subject_token_type", req.SubjectTokenType, "requested_token_type", reqType)...)

	return g.respond(ctx, tokens.Authorization{
		Client:      client,
		Claims:      tokens.ClaimsFromIdentity(identity),
		Scopes:      req.Scopes,
		ConnectorID: req.ConnectorID,
	}, reqType)
}

// delegate is the exchange with an actor (RFC 8693 §1.1 delegation). Both
// tokens must be signed tokens whose claims the connectors return, so the grant
// can check the actor's audience, the subject's may_act, and bound the issued
// token's lifetime by both.
func (g *tokenExchange) delegate(ctx context.Context, req *Request, client storage.Client, subjectConn connectors.Connector, policy DelegationPolicy) (Responder, error) {
	if req.ActorToken == "" {
		return nil, invalidRequest("this client must present an actor_token")
	}
	if req.SubjectTokenType != oauth2.TokenTypeID || !slices.Contains([]string{oauth2.TokenTypeID, oauth2.TokenTypeJWT}, req.ActorTokenType) {
		return nil, invalidRequest("token type not supported for delegation")
	}
	if !slices.Contains(policy.ActorConnectors, req.ActorConnectorID) {
		return nil, invalidRequest("actor connector not allowed for this client")
	}
	actorConn, oerr := g.resolveConnector(ctx, req.ActorConnectorID, client, oauth2.GrantTypeTokenExchange)
	if oerr != nil {
		return nil, oerr
	}
	subjectClaimsConn, ok := subjectConn.Connector.(connector.TokenClaimsConnector)
	if !ok {
		g.logger.ErrorContext(ctx, "connector cannot return verified claims", "connector_id", req.ConnectorID)
		return nil, invalidRequest("subject connector does not support delegation")
	}
	actorClaimsConn, ok := actorConn.Connector.(connector.TokenClaimsConnector)
	if !ok {
		g.logger.ErrorContext(ctx, "connector cannot return verified claims", "connector_id", req.ActorConnectorID)
		return nil, invalidRequest("actor connector does not support delegation")
	}

	identity, err := subjectConn.Connector.(connector.TokenIdentityConnector).TokenIdentity(ctx, req.SubjectTokenType, req.SubjectToken)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to verify subject token", "err", err)
		return nil, &oauth2.Error{Type: oauth2.AccessDenied, Status: http.StatusUnauthorized}
	}
	subjectClaims, err := subjectClaimsConn.TokenClaims(ctx, req.SubjectTokenType, req.SubjectToken)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to verify subject token", "err", err)
		return nil, &oauth2.Error{Type: oauth2.AccessDenied, Status: http.StatusUnauthorized}
	}
	if subjectClaims["act"] != nil {
		return nil, invalidRequest("delegated subject tokens are not supported")
	}

	actorClaims, err := actorClaimsConn.TokenClaims(ctx, req.ActorTokenType, req.ActorToken)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to verify actor token", "actor_connector_id", req.ActorConnectorID, "err", err)
		return nil, invalidRequest("invalid actor_token")
	}
	if !audienceContains(actorClaims["aud"], client.ID) {
		return nil, invalidRequest("actor token audience does not include this client")
	}
	if mayAct, ok := subjectClaims["may_act"]; ok && !mayActAllows(mayAct, actorClaims) {
		return nil, invalidRequest("actor is not authorized by the subject token's may_act claim")
	}
	actor := &tokens.ActorClaims{Issuer: claimString(actorClaims, "iss"), Subject: claimString(actorClaims, "sub")}
	if actor.Subject == "" {
		return nil, invalidRequest("invalid actor_token")
	}

	reqType := requestedTokenType(req)
	g.logger.InfoContext(ctx, "token exchange successful", append(identityLogAttrs(req, client, identity),
		"subject_token_type", req.SubjectTokenType, "requested_token_type", reqType,
		"actor_connector_id", req.ActorConnectorID, "actor_iss", actor.Issuer, "actor_sub", actor.Subject)...)

	return g.respond(ctx, tokens.Authorization{
		Client:      client,
		Claims:      tokens.ClaimsFromIdentity(identity),
		Scopes:      req.Scopes,
		ConnectorID: req.ConnectorID,
		Act:         actor,
		ExpiresAt:   earliestExpiry(subjectClaims, actorClaims),
	}, reqType)
}

// respond signs the requested token type. RFC 8693 returns a single requested
// token plus issued_token_type, not the standard access+id+refresh set, so it
// signs from the issuer primitives.
func (g *tokenExchange) respond(ctx context.Context, auth tokens.Authorization, reqType string) (Responder, error) {
	var (
		token  string
		expiry time.Time
		err    error
	)
	switch reqType {
	case oauth2.TokenTypeID:
		token, expiry, err = g.issuer.SignIDToken(ctx, auth, "", "")
	case oauth2.TokenTypeAccess:
		token, expiry, err = g.issuer.SignAccessToken(ctx, auth)
	default:
		return nil, &oauth2.Error{Type: oauth2.RequestNotSupported, Description: "Invalid requested_token_type.", Status: http.StatusBadRequest}
	}
	if err != nil {
		g.logger.ErrorContext(ctx, "token exchange failed to create new token", "requested_token_type", reqType, "err", err)
		return nil, &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
	}

	return tokens.Response{
		AccessToken:     token,
		IssuedTokenType: reqType,
		TokenType:       "bearer",
		ExpiresIn:       int(time.Until(expiry).Seconds()),
	}, nil
}

func identityLogAttrs(req *Request, client storage.Client, identity connector.Identity) []any {
	email := identity.Email
	if !identity.EmailVerified {
		email += " (unverified)"
	}
	return []any{
		"connector_id", req.ConnectorID, "client_id", client.ID,
		"user_id", identity.UserID,
		"username", identity.Username, "preferred_username", identity.PreferredUsername,
		"email", email, "groups", identity.Groups,
	}
}

// earliestExpiry is the earliest exp among the given claim sets; zero when none
// carries one.
func earliestExpiry(claimSets ...map[string]any) time.Time {
	var earliest time.Time
	for _, claims := range claimSets {
		if exp, ok := claimTime(claims, "exp"); ok && (earliest.IsZero() || exp.Before(earliest)) {
			earliest = exp
		}
	}
	return earliest
}

func invalidRequest(description string) *oauth2.Error {
	return &oauth2.Error{Type: oauth2.InvalidRequest, Description: description, Status: http.StatusBadRequest}
}

// requestedTokenType is the requested_token_type param, defaulting to an access
// token (RFC 8693 §2.1).
func requestedTokenType(req *Request) string {
	if req.RequestedTokenType != "" {
		return req.RequestedTokenType
	}
	return oauth2.TokenTypeAccess
}
