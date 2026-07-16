package grants

import (
	"context"
	"log/slog"
	"net/http"
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
type tokenExchange struct {
	issuer *tokens.Issuer
	logger *slog.Logger
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

	teConn, ok := conn.Connector.(connector.TokenIdentityConnector)
	if !ok {
		g.logger.ErrorContext(ctx, "connector doesn't implement token exchange", "connector_id", req.ConnectorID)
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Requested connector does not exist.", Status: http.StatusBadRequest}
	}
	identity, err := teConn.TokenIdentity(ctx, req.SubjectTokenType, req.SubjectToken)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to verify subject token", "err", err)
		return nil, &oauth2.Error{Type: oauth2.AccessDenied, Status: http.StatusUnauthorized}
	}

	email := identity.Email
	if !identity.EmailVerified {
		email += " (unverified)"
	}
	reqType := requestedTokenType(req)
	g.logger.InfoContext(ctx, "token exchange successful",
		"connector_id", req.ConnectorID, "client_id", client.ID,
		"user_id", identity.UserID,
		"username", identity.Username, "preferred_username", identity.PreferredUsername,
		"email", email, "groups", identity.Groups,
		"subject_token_type", req.SubjectTokenType, "requested_token_type", reqType)

	auth := tokens.Authorization{
		Client:      client,
		Claims:      tokens.ClaimsFromIdentity(identity),
		Scopes:      req.Scopes,
		ConnectorID: req.ConnectorID,
	}

	// RFC 8693 returns a single requested token plus issued_token_type, not the
	// standard access+id+refresh set, so it signs from the issuer primitives.
	var (
		token  string
		expiry time.Time
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

// requestedTokenType is the requested_token_type param, defaulting to an access
// token (RFC 8693 §2.1).
func requestedTokenType(req *Request) string {
	if req.RequestedTokenType != "" {
		return req.RequestedTokenType
	}
	return oauth2.TokenTypeAccess
}
