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
// issued token then carries the actor in its "act" claim (RFC 8693 §4.1) and
// only the audiences the policy allows.
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

// ConnectorID reads the connector_id parameter (an RFC 8693 extension). Empty
// means the subject token was issued by Dex itself and is verified with its
// own keys.
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

	policy, delegating := g.delegation.byClient[client.ID]
	delegating = delegating && !client.Public
	if !delegating && req.ActorToken != "" {
		return nil, invalidRequest("delegation is not enabled for this client")
	}

	var (
		subject *subject
		err     error
	)
	if req.ConnectorID == "" {
		subject, err = g.selfSubject(ctx, req, client)
	} else {
		subject, err = g.connectorSubject(ctx, req, client, conn, delegating)
	}
	if err != nil {
		return nil, err
	}
	if !delegating {
		return g.impersonate(ctx, req, client, subject)
	}
	return g.delegate(ctx, req, client, subject, policy)
}

// subject is the verified subject token: the identity to issue for, the
// connector that becomes part of the sub, and the token's own claims when the
// verifier exposes them (nil for a connector without TokenClaimsConnector).
type subject struct {
	identity    connector.Identity
	connectorID string
	claims      map[string]any
}

// selfSubject verifies a subject token Dex issued itself, with its own keys.
// The identity is rebuilt from the token so the issued sub is the same, and
// the token must have been issued to the requesting client.
func (g *tokenExchange) selfSubject(ctx context.Context, req *Request, client storage.Client) (*subject, error) {
	claims, err := g.issuer.VerifyToken(ctx, req.SubjectToken)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to verify subject token", "err", err)
		return nil, &oauth2.Error{Type: oauth2.AccessDenied, Status: http.StatusUnauthorized}
	}
	if !audienceContains(claims["aud"], client.ID) {
		return nil, invalidRequest("subject token audience does not include this client")
	}
	identity, connectorID, err := identityFromClaims(claims)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to decode subject token", "err", err)
		return nil, &oauth2.Error{Type: oauth2.AccessDenied, Status: http.StatusUnauthorized}
	}
	return &subject{identity: identity, connectorID: connectorID, claims: claims}, nil
}

// connectorSubject verifies a subject token through its connector. The
// token's claims are fetched only for delegation, which needs them.
func (g *tokenExchange) connectorSubject(ctx context.Context, req *Request, client storage.Client, conn connectors.Connector, withClaims bool) (*subject, error) {
	teConn, ok := conn.Connector.(connector.TokenIdentityConnector)
	if !ok {
		g.logger.ErrorContext(ctx, "connector doesn't implement token exchange", "connector_id", req.ConnectorID)
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Requested connector does not exist.", Status: http.StatusBadRequest}
	}
	var claimsConn connector.TokenClaimsConnector
	if withClaims {
		if claimsConn, ok = conn.Connector.(connector.TokenClaimsConnector); !ok {
			g.logger.ErrorContext(ctx, "connector cannot return verified claims", "connector_id", req.ConnectorID)
			return nil, invalidRequest("subject connector does not support delegation")
		}
		if req.SubjectTokenType != oauth2.TokenTypeID {
			return nil, invalidRequest("token type not supported for delegation")
		}
	}
	identity, err := teConn.TokenIdentity(ctx, req.SubjectTokenType, req.SubjectToken)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to verify subject token", "err", err)
		return nil, &oauth2.Error{Type: oauth2.AccessDenied, Status: http.StatusUnauthorized}
	}
	s := &subject{identity: identity, connectorID: req.ConnectorID}
	if claimsConn != nil {
		if s.claims, err = claimsConn.TokenClaims(ctx, req.SubjectTokenType, req.SubjectToken); err != nil {
			g.logger.ErrorContext(ctx, "failed to verify subject token", "err", err)
			return nil, &oauth2.Error{Type: oauth2.AccessDenied, Status: http.StatusUnauthorized}
		}
		if !audienceContains(s.claims["aud"], client.ID) {
			return nil, invalidRequest("subject token audience does not include this client")
		}
	}
	return s, nil
}

// impersonate is the exchange without an actor: the issued token says nothing
// about who presented the subject token.
func (g *tokenExchange) impersonate(ctx context.Context, req *Request, client storage.Client, subject *subject) (Responder, error) {
	reqType := requestedTokenType(req)
	g.logger.InfoContext(ctx, "token exchange successful", append(identityLogAttrs(subject, client),
		"subject_token_type", req.SubjectTokenType, "requested_token_type", reqType)...)

	return g.respond(ctx, tokens.Authorization{
		Client:      client,
		Claims:      tokens.ClaimsFromIdentity(subject.identity),
		Scopes:      req.Scopes,
		ConnectorID: subject.connectorID,
		ExpiresAt:   earliestExpiry(subject.claims),
	}, reqType)
}

// delegate is the exchange with an actor (RFC 8693 §1.1 delegation). The
// actor token must be a signed token whose claims its connector returns, so
// the grant can check its audience, the subject's may_act, and bound the
// issued token's lifetime by both tokens.
func (g *tokenExchange) delegate(ctx context.Context, req *Request, client storage.Client, subject *subject, policy DelegationPolicy) (Responder, error) {
	if req.ActorToken == "" {
		return nil, invalidRequest("this client must present an actor_token")
	}
	if !slices.Contains([]string{oauth2.TokenTypeID, oauth2.TokenTypeJWT}, req.ActorTokenType) {
		return nil, invalidRequest("token type not supported for delegation")
	}
	for _, scope := range req.Scopes {
		if peerID, ok := tokens.ParseCrossClientScope(scope); ok && peerID != client.ID && !slices.Contains(policy.Audiences, peerID) {
			return nil, &oauth2.Error{Type: oauth2.InvalidTarget, Description: "audience not allowed for delegation", Status: http.StatusBadRequest}
		}
	}
	if !slices.Contains(policy.ActorConnectors, req.ActorConnectorID) {
		return nil, invalidRequest("actor connector not allowed for this client")
	}
	actorConn, oerr := g.resolveConnector(ctx, req.ActorConnectorID, client, oauth2.GrantTypeTokenExchange)
	if oerr != nil {
		return nil, oerr
	}
	actorClaimsConn, ok := actorConn.Connector.(connector.TokenClaimsConnector)
	if !ok {
		g.logger.ErrorContext(ctx, "connector cannot return verified claims", "connector_id", req.ActorConnectorID)
		return nil, invalidRequest("actor connector does not support delegation")
	}
	if subject.claims["act"] != nil {
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
	if mayAct, ok := subject.claims["may_act"]; ok && !mayActAllows(mayAct, actorClaims) {
		return nil, invalidRequest("actor is not authorized by the subject token's may_act claim")
	}
	actor := &tokens.ActorClaims{Issuer: claimString(actorClaims, "iss"), Subject: claimString(actorClaims, "sub")}
	if actor.Subject == "" {
		return nil, invalidRequest("invalid actor_token")
	}

	reqType := requestedTokenType(req)
	g.logger.InfoContext(ctx, "token exchange successful", append(identityLogAttrs(subject, client),
		"subject_token_type", req.SubjectTokenType, "requested_token_type", reqType,
		"actor_connector_id", req.ActorConnectorID, "actor_iss", actor.Issuer, "actor_sub", actor.Subject)...)

	return g.respond(ctx, tokens.Authorization{
		Client:      client,
		Claims:      tokens.ClaimsFromIdentity(subject.identity),
		Scopes:      req.Scopes,
		ConnectorID: subject.connectorID,
		Act:         actor,
		ExpiresAt:   earliestExpiry(subject.claims, actorClaims),
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

func identityLogAttrs(subject *subject, client storage.Client) []any {
	identity := subject.identity
	email := identity.Email
	if !identity.EmailVerified {
		email += " (unverified)"
	}
	return []any{
		"connector_id", subject.connectorID, "client_id", client.ID,
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
