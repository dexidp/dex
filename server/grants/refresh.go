package grants

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// refresh serves the refresh_token grant: it validates and rotates a refresh
// token, re-reads the identity (from the session or the upstream connector) and
// issues a fresh token set. Its response reuses the rotated refresh token rather
// than minting a new one, so it mints its own instead of the standard Issue.
type refresh struct {
	storage         storage.Storage
	connectors      *connectors.Cache
	issuer          *tokens.Issuer
	policy          *tokens.RefreshStrategy
	sessionsEnabled bool
	now             func() time.Time
	logger          *slog.Logger
}

func (g *refresh) GrantType() string {
	return oauth2.GrantTypeRefreshToken
}

func (g *refresh) RequiresClientAuth() bool {
	return true
}

// Scopes are validated against the token's originally authorized scopes in
// Authorize, not against a fixed set, so the shared phase passes them through.
func (g *refresh) ScopePolicy() ScopePolicy {
	return ScopePolicy{}
}

// ConnectorID is empty: the connector is recorded on the stored refresh token, so
// it is resolved and re-checked inside Authorize on every refresh.
func (g *refresh) ConnectorID(req *Request) string {
	return ""
}

func (g *refresh) Authorize(ctx context.Context, req *Request, client storage.Client, conn connectors.Connector) (*Result, error) {
	token, oerr := parseRefreshToken(req.RefreshToken)
	if oerr != nil {
		return nil, oerr
	}

	refreshToken, err := tokens.LookupRefreshToken(ctx, g.storage, g.policy, g.logger, &client.ID, token)
	if err != nil {
		return nil, refreshLookupError(err)
	}

	// Resolve the connector and re-check authorization on every refresh: the
	// connector may have been removed from the client's allowed list, or had the
	// refresh grant revoked, after this token was issued.
	if !connectors.ConnectorAllowed(client.AllowedConnectors, refreshToken.ConnectorID) {
		g.logger.WarnContext(ctx, "connector not allowed for client", "client_id", client.ID, "connector_id", refreshToken.ConnectorID)
		return nil, &oauth2.Error{Type: oauth2.InvalidGrant, Description: "Connector not allowed for this client.", Status: http.StatusBadRequest}
	}
	upstream, err := g.connectors.Get(ctx, refreshToken.ConnectorID)
	if err != nil {
		g.logger.ErrorContext(ctx, "connector not found", "connector_id", refreshToken.ConnectorID, "err", err)
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Status: http.StatusInternalServerError}
	}
	if !connectors.GrantTypeAllowed(upstream.GrantTypes, oauth2.GrantTypeRefreshToken) {
		g.logger.ErrorContext(ctx, "connector does not allow refresh token grant", "connector_id", refreshToken.ConnectorID)
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Connector does not support refresh tokens.", Status: http.StatusBadRequest}
	}

	scopes, oerr := g.refreshScopes(req, refreshToken)
	if oerr != nil {
		return nil, oerr
	}

	var userIdent *storage.UserIdentity
	if g.sessionsEnabled {
		ui, err := g.storage.GetUserIdentity(ctx, refreshToken.Claims.UserID, refreshToken.ConnectorID)
		if err != nil {
			g.logger.ErrorContext(ctx, "failed to get user identity", "err", err)
			return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Status: http.StatusInternalServerError}
		}
		userIdent = &ui
	}

	authTime := time.Time{}
	if userIdent != nil {
		authTime = userIdent.LastLogin
	}

	// When sessions are enabled, downstream refresh is disconnected from the
	// upstream provider: use the claims cached in UserIdentity at the last login
	// instead of contacting the connector (which may fail if the upstream token
	// has expired). Otherwise re-read the identity from the connector.
	freshIdentity := func(ctx context.Context) (connector.Identity, error) {
		if userIdent != nil {
			return tokens.IdentityFromClaims(userIdent.Claims), nil
		}
		connectorData, err := g.refreshConnectorData(ctx, refreshToken)
		if err != nil {
			return connector.Identity{}, err
		}
		return g.refreshWithConnector(ctx, upstream, connectorData, scopes, tokens.IdentityFromClaims(refreshToken.Claims))
	}

	rawNewToken, ident, err := g.issuer.Refresh.Rotate(ctx, refreshToken, token, g.policy, freshIdentity)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to rotate refresh token", "err", err)
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Status: http.StatusInternalServerError}
	}

	return &Result{
		Authorization: tokens.Authorization{
			Client: client,
			Claims: storage.Claims{
				UserID:            ident.UserID,
				Username:          ident.Username,
				PreferredUsername: ident.PreferredUsername,
				Email:             ident.Email,
				EmailVerified:     ident.EmailVerified,
				Groups:            ident.Groups,
			},
			Scopes:      scopes,
			ConnectorID: refreshToken.ConnectorID,
			Nonce:       refreshToken.Nonce,
			AuthTime:    authTime,
		},
		RefreshToken: rawNewToken,
	}, nil
}

// Mint issues the token set with the already-rotated refresh token, so it does
// not use the standard mint (which would create a second refresh token).
func (g *refresh) Mint(ctx context.Context, req *Request, res *Result) (tokens.Response, error) {
	accessToken, _, err := g.issuer.SignAccessToken(ctx, res.Authorization)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to create new access token", "err", err)
		return tokens.Response{}, &oauth2.Error{Type: oauth2.InvalidRequest, Status: http.StatusInternalServerError}
	}

	idToken, expiry, err := g.issuer.SignIDToken(ctx, res.Authorization, accessToken, "")
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to create ID token", "err", err)
		return tokens.Response{}, &oauth2.Error{Type: oauth2.InvalidRequest, Status: http.StatusInternalServerError}
	}

	ts := tokens.TokenSet{AccessToken: accessToken, IDToken: idToken, RefreshToken: res.RefreshToken, Expiry: expiry}
	return ts.Response(g.now()), nil
}

// refreshScopes resolves the scopes for this refresh. Per RFC 6749 §6 the client
// may omit them (defaulting to the originally authorized scopes) but may not
// widen them.
func (g *refresh) refreshScopes(req *Request, refreshToken *storage.RefreshToken) ([]string, *oauth2.Error) {
	if len(req.Scopes) == 0 {
		return refreshToken.Scopes, nil
	}

	var unauthorized []string
	for _, scope := range req.Scopes {
		if !contains(refreshToken.Scopes, scope) {
			unauthorized = append(unauthorized, scope)
		}
	}
	if len(unauthorized) > 0 {
		return nil, oauth2.Errorf(oauth2.InvalidRequest, http.StatusBadRequest, "Requested scopes contain unauthorized scope(s): %q.", unauthorized)
	}
	return req.Scopes, nil
}

// refreshConnectorData returns the connector data for the upstream refresh: the
// token's own data for legacy tokens that still carry it, otherwise the value on
// the user's offline session.
func (g *refresh) refreshConnectorData(ctx context.Context, refreshToken *storage.RefreshToken) ([]byte, error) {
	if len(refreshToken.ConnectorData) > 0 {
		return refreshToken.ConnectorData, nil
	}

	session, err := g.storage.GetOfflineSessions(ctx, refreshToken.Claims.UserID, refreshToken.ConnectorID)
	if err != nil {
		if err != storage.ErrNotFound {
			g.logger.ErrorContext(ctx, "failed to get offline session", "err", err)
			return nil, err
		}
		return nil, nil
	}
	return session.ConnectorData, nil
}

// refreshWithConnector re-reads the identity from the upstream connector when it
// supports refreshing.
func (g *refresh) refreshWithConnector(ctx context.Context, conn connectors.Connector, connectorData []byte, scopes []string, ident connector.Identity) (connector.Identity, error) {
	refreshConn, ok := conn.Connector.(connector.RefreshConnector)
	if !ok {
		return ident, nil
	}

	ident.ConnectorData = connectorData
	g.logger.Debug("connector data before refresh", "connector_data", ident.ConnectorData)

	newIdent, err := refreshConn.Refresh(ctx, tokens.ParseScopes(scopes), ident)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to refresh identity", "err", err)
		return ident, err
	}
	return newIdent, nil
}

// parseRefreshToken decodes the refresh_token parameter, tolerating the legacy
// raw-ID form for backward compatibility.
func parseRefreshToken(code string) (*internal.RefreshToken, *oauth2.Error) {
	if code == "" {
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "No refresh token is found in request.", Status: http.StatusBadRequest}
	}

	token := new(internal.RefreshToken)
	if err := internal.Unmarshal(code, token); err != nil {
		// Assume a raw refresh token ID generated by an older server that has no
		// Token value. Reuse is still rejected because Token stays empty.
		token = &internal.RefreshToken{RefreshId: code, Token: ""}
	}
	return token, nil
}

// refreshLookupError maps a tokens.LookupRefreshToken sentinel to the grant's
// OAuth2 error response.
func refreshLookupError(err error) *oauth2.Error {
	const claimedDesc = "Refresh token is invalid or has already been claimed by another client."
	switch {
	case errors.Is(err, tokens.ErrRefreshTokenInvalid):
		return &oauth2.Error{Type: oauth2.InvalidRequest, Description: claimedDesc, Status: http.StatusBadRequest}
	case errors.Is(err, tokens.ErrRefreshTokenClaimedByOtherClient):
		return &oauth2.Error{Type: oauth2.InvalidGrant, Description: claimedDesc, Status: http.StatusBadRequest}
	case errors.Is(err, tokens.ErrRefreshTokenExpired):
		return &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Refresh token expired.", Status: http.StatusBadRequest}
	default:
		return &oauth2.Error{Type: oauth2.InvalidRequest, Status: http.StatusInternalServerError}
	}
}

func contains(arr []string, item string) bool {
	for _, v := range arr {
		if v == item {
			return true
		}
	}
	return false
}
