package grants

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
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
// A client with a delegation policy may also present an actor token; the
// issued token then carries the actor in its "act" claim (RFC 8693 §4.1),
// nested in front of any actor the subject token already names, and only the
// audiences and scopes the policy allows.
type tokenExchange struct {
	idJAG      *idJAG
	issuer     *tokens.Issuer
	storage    storage.Storage
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
	if (req.ActorToken == "") != (req.ActorTokenType == "") {
		return nil, invalidRequest("actor_token and actor_token_type must be sent together")
	}
	if g.idJAG != nil && req.RequestedTokenType == oauth2.TokenTypeIDJAG {
		return g.idJAG.Authorize(ctx, req, client, g)
	}

	switch requestedTokenType(req) {
	case oauth2.TokenTypeID, oauth2.TokenTypeAccess:
	default:
		return nil, &oauth2.Error{Type: oauth2.RequestNotSupported, Description: "Invalid requested_token_type.", Status: http.StatusBadRequest}
	}
	switch req.SubjectTokenType {
	case oauth2.TokenTypeID, oauth2.TokenTypeAccess: // ok, continue
	default:
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Invalid subject_token_type.", Status: http.StatusBadRequest}
	}
	if req.SubjectToken == "" {
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Missing subject_token", Status: http.StatusBadRequest}
	}
	if g.delegation.actorConnectors[req.ConnectorID] {
		return nil, invalidRequest("connector only verifies actor tokens")
	}

	// actor_token is optional (RFC 8693 §2.1): without one, any client
	// impersonates the subject.
	if req.ActorToken == "" {
		subject, err := g.verifySubject(ctx, req, client, conn, false)
		if err != nil {
			return nil, err
		}
		return g.impersonate(ctx, req, client, subject)
	}
	policy, delegating := g.delegation.byClient[client.ID]
	if !delegating || client.Public {
		return nil, invalidRequest("delegation is not enabled for this client")
	}
	if len(req.Resource) > 0 {
		return nil, invalidTarget("resource is not supported, use audience")
	}
	for _, aud := range req.Audience {
		if aud == client.ID {
			continue
		}
		if !slices.Contains(policy.AllowedAudiences, aud) {
			return nil, invalidTarget("audience not allowed for delegation")
		}
		trusted, err := tokens.CrossClientTrusted(ctx, g.storage, client.ID, aud)
		if err != nil {
			g.logger.ErrorContext(ctx, "error validating cross client trust", "client_id", client.ID, "peer_id", aud, "err", err)
			return nil, &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
		}
		if !trusted {
			return nil, invalidTarget("audience not allowed for delegation")
		}
	}
	for _, scope := range req.Scopes {
		if _, crossClient := tokens.ParseCrossClientScope(scope); crossClient {
			return nil, invalidScope("use the audience parameter for delegation")
		}
		if !standardScopes[scope] && !slices.Contains(policy.AllowedScopes, scope) {
			return nil, invalidScope("scope not allowed for delegation")
		}
	}

	subject, err := g.verifySubject(ctx, req, client, conn, true)
	if err != nil {
		return nil, err
	}
	return g.delegate(ctx, req, client, subject)
}

// subject is the verified subject token: the identity to issue for, the
// connector that becomes part of the sub, the browser session it is bound to,
// and the token's own claims when the verifier exposes them (nil for a
// connector without TokenClaimsConnector).
type subject struct {
	identity    connector.Identity
	connectorID string
	sessionID   string
	claims      map[string]any
}

// verifySubject verifies the subject token: with Dex's own keys when no
// connector_id was sent, otherwise through that connector. A token that does
// not verify is refused with invalid_request on every path (RFC 8693 §2.2.2).
func (g *tokenExchange) verifySubject(ctx context.Context, req *Request, client storage.Client, conn connectors.Connector, delegating bool) (*subject, error) {
	if req.ConnectorID == "" {
		return g.selfSubject(ctx, req, client)
	}
	return g.connectorSubject(ctx, req, client, conn, delegating)
}

// selfSubject verifies a subject token Dex issued itself, with its own keys and
// the session check of the client it was issued to. The identity is rebuilt
// from the token so the issued sub is the same; the token must have been
// issued to the requesting client, and the connector encoded in its sub must
// be one the requesting client may use for token exchange.
func (g *tokenExchange) selfSubject(ctx context.Context, req *Request, client storage.Client) (*subject, error) {
	verified, err := g.issuer.VerifyToken(ctx, req.SubjectToken)
	switch {
	case errors.Is(err, tokens.ErrSessionEnded):
		return nil, invalidRequest("subject token session has ended")
	case errors.Is(err, tokens.ErrInvalidToken):
		g.logger.InfoContext(ctx, "failed to verify subject token", "err", err)
		return nil, invalidRequest("invalid subject_token")
	case err != nil:
		g.logger.ErrorContext(ctx, "failed to verify subject token", "err", err)
		return nil, &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
	}
	claims := verified.Claims
	if !audienceContains(claims["aud"], client.ID) {
		return nil, invalidRequest("subject token audience does not include this client")
	}
	identity, connectorID, err := identityFromClaims(claims)
	if err != nil {
		g.logger.InfoContext(ctx, "failed to decode subject token", "err", err)
		return nil, invalidRequest("invalid subject_token")
	}
	if _, oerr := g.resolveConnector(ctx, connectorID, client, oauth2.GrantTypeTokenExchange); oerr != nil {
		return nil, invalidRequest(oerr.Description)
	}
	if g.delegation.actorConnectors[connectorID] {
		return nil, invalidRequest("connector only verifies actor tokens")
	}
	if verified.SessionID != "" && verified.Client.RefreshBoundToSession() && !client.RefreshBoundToSession() {
		return nil, invalidRequest("subject token is bound to a session this client does not end with")
	}
	return &subject{identity: identity, connectorID: connectorID, sessionID: verified.SessionID, claims: claims}, nil
}

// connectorSubject verifies a subject token through its connector. Delegation
// requires the token's claims. Impersonation reads them when the connector can
// return them for an id_token, only to refuse a delegated subject and to
// enforce may_act, so it otherwise issues as it did before.
func (g *tokenExchange) connectorSubject(ctx context.Context, req *Request, client storage.Client, conn connectors.Connector, delegating bool) (*subject, error) {
	teConn, ok := conn.Connector.(connector.TokenIdentityConnector)
	if !ok {
		g.logger.ErrorContext(ctx, "connector doesn't implement token exchange", "connector_id", req.ConnectorID)
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Requested connector does not exist.", Status: http.StatusBadRequest}
	}
	claimsConn, hasClaims := conn.Connector.(connector.TokenClaimsConnector)
	if delegating {
		if !hasClaims {
			g.logger.ErrorContext(ctx, "connector cannot return verified claims", "connector_id", req.ConnectorID)
			return nil, invalidRequest("subject connector does not support delegation")
		}
		if req.SubjectTokenType != oauth2.TokenTypeID {
			return nil, invalidRequest("token type not supported for delegation")
		}
	}
	identity, err := teConn.TokenIdentity(ctx, req.SubjectTokenType, req.SubjectToken)
	var unavailableErr *connector.UpstreamUnavailableError
	if errors.As(err, &unavailableErr) {
		g.logger.ErrorContext(ctx, "failed to verify subject token: upstream unavailable", "connector_id", req.ConnectorID, "err", err)
		return nil, &oauth2.Error{Type: oauth2.ServerError, Description: "The connector's upstream identity provider is unavailable.", Status: http.StatusServiceUnavailable, RetryAfter: 30 * time.Second}
	}
	if err != nil {
		g.logger.InfoContext(ctx, "failed to verify subject token", "connector_id", req.ConnectorID, "err", err)
		return nil, invalidRequest("invalid subject_token")
	}
	s := &subject{identity: identity, connectorID: req.ConnectorID}
	if !hasClaims || req.SubjectTokenType != oauth2.TokenTypeID {
		return s, nil
	}
	claims, err := claimsConn.TokenClaims(ctx, req.SubjectTokenType, req.SubjectToken)
	switch {
	case err != nil && delegating:
		g.logger.InfoContext(ctx, "failed to verify subject token", "connector_id", req.ConnectorID, "err", err)
		return nil, invalidRequest("invalid subject_token")
	case err != nil:
		// The token verified through TokenIdentity; without its claims
		// impersonation proceeds as for a connector that returns none.
		g.logger.DebugContext(ctx, "subject token claims unavailable", "connector_id", req.ConnectorID, "err", err)
	case delegating:
		if !audienceContains(claims["aud"], client.ID) {
			return nil, invalidRequest("subject token audience does not include this client")
		}
		s.claims = claims
	default:
		s.claims = map[string]any{}
		for _, name := range []string{"act", "may_act"} {
			if v, ok := claims[name]; ok {
				s.claims[name] = v
			}
		}
	}
	return s, nil
}

// impersonate is the exchange without an actor: the issued token says nothing
// about who presented the subject token. A subject that names an actor or
// restricts who may act for it is checked where its claims are available.
func (g *tokenExchange) impersonate(ctx context.Context, req *Request, client storage.Client, subject *subject) (Responder, error) {
	if _, _, err := g.authorizeActor(ctx, req, client, subject.claims); err != nil {
		return nil, err
	}
	reqType := requestedTokenType(req)
	g.logger.InfoContext(ctx, "token exchange successful", append(identityLogAttrs(subject, client),
		"subject_token_type", req.SubjectTokenType, "requested_token_type", reqType)...)

	return g.respond(ctx, tokens.Authorization{
		Client:      client,
		Claims:      tokens.ClaimsFromIdentity(subject.identity),
		Scopes:      req.Scopes,
		ConnectorID: subject.connectorID,
		SessionID:   subject.sessionID,
		ExpiresAt:   earliestExpiry(subject.claims),
	}, reqType, "")
}

// delegate is the exchange with an actor (RFC 8693 §1.1 delegation). The
// actor token must be a signed token whose claims its connector returns, so
// the grant can check its audience, the subject's may_act, and bound the
// issued token's lifetime by both tokens.
func (g *tokenExchange) delegate(ctx context.Context, req *Request, client storage.Client, subject *subject) (Responder, error) {
	actor, expiresAt, err := g.authorizeActor(ctx, req, client, subject.claims)
	if err != nil {
		return nil, err
	}
	scopes := delegatedScopes(req.Scopes, subject.claims["scope"], subject.claims["act"] != nil)
	if len(req.Scopes) > 0 && len(scopes) == 0 {
		return nil, invalidScope("subject token does not carry the requested scopes")
	}

	reqType := requestedTokenType(req)

	granted := strings.Join(scopes, " ")
	return g.respond(ctx, tokens.Authorization{
		Client:      client,
		Claims:      tokens.ClaimsFromIdentity(subject.identity),
		Scopes:      scopes,
		ConnectorID: subject.connectorID,
		SessionID:   subject.sessionID,
		Act:         actor,
		ScopeClaim:  granted,
		Audience:    req.Audience,
		ExpiresAt:   expiresAt,
	}, reqType, granted)
}

// verifyActor verifies the actor token with the client's actor connectors, in
// order, and returns the claims of the first that verifies it and that
// connector's ID. The request does not name the connector (RFC 8693 §2.1 has
// no such parameter). A connector the client may not use for token exchange,
// or that cannot return verified claims, is a configuration error: it is
// logged and skipped. It returns nil claims when no connector verifies.
func (g *tokenExchange) verifyActor(ctx context.Context, req *Request, client storage.Client, policy DelegationPolicy) (map[string]any, string) {
	for _, id := range policy.ActorConnectors {
		conn, oerr := g.resolveConnector(ctx, id, client, oauth2.GrantTypeTokenExchange)
		if oerr != nil {
			g.logger.ErrorContext(ctx, "actor connector unusable for this client", "client_id", client.ID, "actor_connector_id", id, "err", oerr.Description)
			continue
		}
		claimsConn, ok := conn.Connector.(connector.TokenClaimsConnector)
		if !ok {
			g.logger.ErrorContext(ctx, "actor connector cannot return verified claims", "actor_connector_id", id)
			continue
		}
		claims, err := claimsConn.TokenClaims(ctx, req.ActorTokenType, req.ActorToken)
		if err != nil {
			g.logger.DebugContext(ctx, "actor token not verified by connector", "actor_connector_id", id, "err", err)
			continue
		}
		return claims, id
	}
	g.logger.InfoContext(ctx, "failed to verify actor token", "client_id", client.ID)
	return nil, ""
}

// respond signs the requested token type. RFC 8693 returns a single requested
// token plus issued_token_type, not the standard access+id+refresh set, so it
// signs from the issuer primitives.
// scope, when not empty, is reported as the response's scope parameter (RFC
// 8693 §2.2.1).
func (g *tokenExchange) respond(ctx context.Context, auth tokens.Authorization, reqType, scope string) (Responder, error) {
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
		ExpiresIn:       int(expiry.Sub(g.issuer.Now()).Seconds()),
		Scope:           scope,
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

func invalidTarget(description string) *oauth2.Error {
	return &oauth2.Error{Type: oauth2.InvalidTarget, Description: description, Status: http.StatusBadRequest}
}

func invalidScope(description string) *oauth2.Error {
	return &oauth2.Error{Type: oauth2.InvalidScope, Description: description, Status: http.StatusBadRequest}
}

// requestedTokenType is the requested_token_type param, defaulting to an access
// token (RFC 8693 §2.1).
func requestedTokenType(req *Request) string {
	if req.RequestedTokenType != "" {
		return req.RequestedTokenType
	}
	return oauth2.TokenTypeAccess
}

func (g *tokenExchange) authorizeActor(ctx context.Context, req *Request, client storage.Client, subjectClaims map[string]any) (*tokens.ActorClaims, time.Time, error) {
	if req.ActorToken == "" {
		if subjectClaims["act"] != nil {
			return nil, time.Time{}, invalidRequest("delegated subject tokens require an actor_token")
		}
		if mayAct, ok := subjectClaims["may_act"]; ok && !mayActAllows(mayAct, nil, client.ID, false) {
			return nil, time.Time{}, invalidRequest("actor is not authorized by the subject token's may_act claim")
		}
		return nil, time.Time{}, nil
	}
	policy, enabled := g.delegation.byClient[client.ID]
	if !enabled || client.Public {
		return nil, time.Time{}, invalidRequest("delegation is not enabled for this client")
	}
	if !slices.Contains([]string{oauth2.TokenTypeID, oauth2.TokenTypeJWT}, req.ActorTokenType) {
		return nil, time.Time{}, invalidRequest("token type not supported for delegation")
	}
	prior, err := priorActor(subjectClaims["act"])
	if err != nil {
		return nil, time.Time{}, invalidRequest("invalid subject_token")
	}

	actorClaims, actorConnID := g.verifyActor(ctx, req, client, policy)
	if actorClaims == nil {
		return nil, time.Time{}, invalidRequest("invalid actor_token")
	}
	if !audienceContains(actorClaims["aud"], client.ID) {
		return nil, time.Time{}, invalidRequest("actor token audience does not include this client")
	}
	if mayAct, ok := subjectClaims["may_act"]; ok && !mayActAllows(mayAct, actorClaims, client.ID, len(policy.ActorConnectors) == 1) {
		return nil, time.Time{}, invalidRequest("actor is not authorized by the subject token's may_act claim")
	}
	actor := &tokens.ActorClaims{Issuer: claimString(actorClaims, "iss"), Subject: claimString(actorClaims, "sub"), Act: prior}
	if actor.Subject == "" {
		return nil, time.Time{}, invalidRequest("invalid actor_token")
	}
	g.logger.InfoContext(ctx, "token exchange actor verified", "client_id", client.ID, "actor_connector_id", actorConnID, "actor_iss", actor.Issuer, "actor_sub", actor.Subject)
	return actor, earliestExpiry(subjectClaims, actorClaims), nil
}
