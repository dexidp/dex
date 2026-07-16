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
// was already authorized at /auth, so it is resolved inside Authorize rather than
// through the endpoint's connector step.
func (g *authorizationCode) ConnectorID(req *Request) string {
	return ""
}

// handle an access token request https://tools.ietf.org/html/rfc6749#section-4.1.3
func (g *authorizationCode) Authorize(ctx context.Context, req *Request, client storage.Client, conn connectors.Connector) (*Result, error) {
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

	// RFC 7636 (PKCE)
	codeChallengeFromStorage := authCode.PKCE.CodeChallenge
	switch {
	case req.CodeVerifier != "" && codeChallengeFromStorage != "":
		calculatedCodeChallenge, err := oauth2.CalculateCodeChallenge(req.CodeVerifier, authCode.PKCE.CodeChallengeMethod)
		if err != nil {
			g.logger.ErrorContext(ctx, "failed to calculate code challenge", "err", err)
			return nil, &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
		}
		if codeChallengeFromStorage != calculatedCodeChallenge {
			return nil, &oauth2.Error{Type: oauth2.InvalidGrant, Description: "Invalid code_verifier.", Status: http.StatusBadRequest}
		}
	case req.CodeVerifier != "":
		// Received no code_challenge on /auth, but a code_verifier on /token.
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "No PKCE flow started. Cannot check code_verifier.", Status: http.StatusBadRequest}
	case codeChallengeFromStorage != "":
		// Received PKCE request on /auth, but no code_verifier on /token.
		return nil, &oauth2.Error{Type: oauth2.InvalidGrant, Description: "Expecting parameter code_verifier in PKCE flow.", Status: http.StatusBadRequest}
	}

	if authCode.RedirectURI != req.RedirectURI {
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "redirect_uri did not match URI from initial request.", Status: http.StatusBadRequest}
	}

	return ExchangeAuthCode(ctx, g.storage, g.connectors, g.logger, authCode, client)
}

// ExchangeAuthCode consumes a validated authorization code and returns what to
// issue: it deletes the code and decides whether a refresh token is warranted.
// The standard mint turns the Result into tokens, binding the code into c_hash.
// It is shared by the authorization_code grant and the device flow, which both
// redeem an auth code for tokens.
func ExchangeAuthCode(ctx context.Context, s storage.Storage, conns *connectors.Cache, logger *slog.Logger, authCode storage.AuthCode, client storage.Client) (*Result, error) {
	auth := tokens.Authorization{
		Client:        client,
		Claims:        authCode.Claims,
		Scopes:        authCode.Scopes,
		ConnectorID:   authCode.ConnectorID,
		Nonce:         authCode.Nonce,
		AuthTime:      authCode.AuthTime,
		ConnectorData: authCode.ConnectorData,
	}

	// Consume the code before issuing so it cannot be replayed.
	if err := s.DeleteAuthCode(ctx, authCode.ID); err != nil {
		logger.ErrorContext(ctx, "failed to delete auth code", "err", err)
		return nil, &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
	}

	// A refresh token is only issued when the connector supports it, the grant
	// type is allowed and offline_access was requested (RFC 6749 §1.5).
	conn, err := conns.Get(ctx, authCode.ConnectorID)
	if err != nil {
		logger.ErrorContext(ctx, "connector not found", "connector_id", authCode.ConnectorID, "err", err)
		return nil, &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
	}

	return &Result{
		Authorization: auth,
		IssueRefresh:  shouldIssueRefreshToken(conn, authCode.Scopes),
		Code:          authCode.ID,
	}, nil
}
