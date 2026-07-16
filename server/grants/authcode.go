package grants

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// authorizationCode serves the authorization_code grant: the client redeems a
// code minted at the /auth endpoint for tokens.
type authorizationCode struct {
	issuer     *tokens.Issuer
	storage    storage.Storage
	connectors *connectors.Cache
	now        func() time.Time
	logger     *slog.Logger
}

func (g *authorizationCode) GrantType() string {
	return oauth2.GrantTypeAuthorizationCode
}

func (g *authorizationCode) RequiresClientAuth() bool {
	return true
}

// Scopes are passed through: they were validated at /auth and stored on the code.
func (g *authorizationCode) ScopePolicy() ScopePolicy {
	return ScopePolicy{}
}

// ConnectorID is empty: the connector is recorded on the stored auth code and
// was already authorized at /auth. The grant resolves it (without re-running the
// invariant) only to decide on a refresh token, inside ExchangeAuthCode.
func (g *authorizationCode) ConnectorID(ctx context.Context, req *Request, client storage.Client) (string, *oauth2.Error) {
	return "", nil
}

// handle an access token request https://tools.ietf.org/html/rfc6749#section-4.1.3
func (g *authorizationCode) Authorize(ctx context.Context, req *Request, client storage.Client, conn connectors.Connector) (Responder, error) {
	if req.Code == "" {
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Required param: code.", Status: http.StatusBadRequest}
	}

	authCode, err := g.storage.GetAuthCode(ctx, req.Code)
	if err != nil || g.now().After(authCode.Expiry) || authCode.ClientID != client.ID {
		if err != nil && err != storage.ErrNotFound {
			g.logger.ErrorContext(ctx, "failed to get auth code", "err", err)
			return nil, &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
		}
		return nil, &oauth2.Error{Type: oauth2.InvalidGrant, Description: "Invalid or expired code parameter.", Status: http.StatusBadRequest}
	}

	if oerr := verifyPKCE(req.CodeVerifier, authCode.PKCE); oerr != nil {
		return nil, oerr
	}

	if authCode.RedirectURI != req.RedirectURI {
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "redirect_uri did not match URI from initial request.", Status: http.StatusBadRequest}
	}

	auth, withRefresh, err := ExchangeAuthCode(ctx, g.connectors, g.logger, authCode, client)
	if err != nil {
		return nil, err
	}

	resp, err := issue(ctx, g.logger, g.issuer, auth, authCode.ID, withRefresh)
	if err != nil {
		return nil, err
	}

	// Consume the code only after the tokens are minted, so a signing failure
	// leaves it redeemable.
	if err := g.storage.DeleteAuthCode(ctx, authCode.ID); err != nil {
		g.logger.ErrorContext(ctx, "failed to delete auth code", "err", err)
		return nil, &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
	}
	return resp, nil
}

// ExchangeAuthCode turns a validated authorization code into the authorization to
// issue tokens for and whether a refresh token is warranted. The caller mints the
// tokens (binding authCode.ID into c_hash) and then consumes the code. It is
// shared by the authorization_code grant and the device flow, which both redeem
// an auth code for tokens.
func ExchangeAuthCode(ctx context.Context, conns *connectors.Cache, logger *slog.Logger, authCode storage.AuthCode, client storage.Client) (tokens.Authorization, bool, error) {
	auth := tokens.Authorization{
		Client:        client,
		Claims:        authCode.Claims,
		Scopes:        authCode.Scopes,
		ConnectorID:   authCode.ConnectorID,
		Nonce:         authCode.Nonce,
		AuthTime:      authCode.AuthTime,
		ConnectorData: authCode.ConnectorData,
	}

	// A refresh token is only issued when the connector supports it, the grant
	// type is allowed and offline_access was requested (RFC 6749 §1.5).
	conn, err := conns.Get(ctx, authCode.ConnectorID)
	if err != nil {
		logger.ErrorContext(ctx, "connector not found", "connector_id", authCode.ConnectorID, "err", err)
		return tokens.Authorization{}, false, &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
	}

	return auth, shouldIssueRefreshToken(conn, authCode.Scopes), nil
}
