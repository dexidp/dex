package grants

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
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
	issuer          *tokens.Issuer
	expiry          *tokens.Expiry
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

// ConnectorID validates the refresh token and reports the connector recorded on
// it, so the endpoint resolves and re-checks that connector on every refresh: a
// client's allowed connectors, or a connector's grant types, may have been
// tightened after the token was issued. The looked-up and decoded token is
// stashed on the request so Authorize reuses it without a second lookup or parse.
func (g *refresh) ConnectorID(ctx context.Context, req *Request, client storage.Client) (string, *oauth2.Error) {
	token, oerr := parseRefreshToken(req.RefreshToken)
	if oerr != nil {
		return "", oerr
	}

	refreshToken, err := tokens.LookupRefreshToken(ctx, g.storage, g.expiry, g.logger, &client.ID, token)
	if err != nil {
		return "", refreshLookupError(err)
	}

	req.refresh, req.refreshID = refreshToken, token
	return refreshToken.ConnectorID, nil
}

// Authorize rotates the refresh token, re-reads the identity against the resolved
// connector, and returns the token set — reusing the rotated refresh token, so it
// mints its own response rather than the standard set (which would mint a second
// refresh token).
func (g *refresh) Authorize(ctx context.Context, req *Request, client storage.Client, conn connectors.Connector) (Responder, error) {
	refreshToken := req.refresh

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
		return g.refreshWithConnector(ctx, conn, connectorData, scopes, tokens.IdentityFromClaims(refreshToken.Claims))
	}

	rawNewToken, ident, err := g.issuer.Refresh.Rotate(ctx, refreshToken, req.refreshID, g.expiry.RefreshStrategy(refreshToken.ConnectorID), freshIdentity)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to rotate refresh token", "err", err)
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Status: http.StatusInternalServerError}
	}

	auth := tokens.Authorization{
		Client:      client,
		Claims:      tokens.ClaimsFromIdentity(ident),
		Scopes:      scopes,
		ConnectorID: refreshToken.ConnectorID,
		Nonce:       refreshToken.Nonce,
		AuthTime:    authTime,
	}

	accessToken, _, err := g.issuer.SignAccessToken(ctx, auth)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to create new access token", "err", err)
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Status: http.StatusInternalServerError}
	}
	idToken, expiry, err := g.issuer.SignIDToken(ctx, auth, accessToken, "")
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to create ID token", "err", err)
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Status: http.StatusInternalServerError}
	}

	ts := tokens.TokenSet{AccessToken: accessToken, IDToken: idToken, RefreshToken: rawNewToken, Expiry: expiry}
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
		if !slices.Contains(refreshToken.Scopes, scope) {
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
