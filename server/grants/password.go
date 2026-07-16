package grants

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// password serves the Resource Owner Password Credentials grant: the client
// exchanges a username and password for tokens via a password-capable connector.
type password struct {
	issuer      *tokens.Issuer
	logger      *slog.Logger
	connectorID string
}

func (g *password) GrantType() string {
	return oauth2.GrantTypePassword
}

func (g *password) RequiresClientAuth() bool {
	return true
}

var passwordScopePolicy = ScopePolicy{
	Standard: map[string]bool{
		tokens.ScopeOpenID:        true,
		tokens.ScopeOfflineAccess: true,
		tokens.ScopeEmail:         true,
		tokens.ScopeProfile:       true,
		tokens.ScopeGroups:        true,
		tokens.ScopeFederatedID:   true,
	},
	RequireOpenID: true,
	ErrorType:     oauth2.InvalidRequest,
}

func (g *password) ScopePolicy() ScopePolicy {
	return passwordScopePolicy
}

// ConnectorID is the connector the password grant is configured to use.
func (g *password) ConnectorID(ctx context.Context, req *Request, client storage.Client) (string, *oauth2.Error) {
	return g.connectorID, nil
}

func (g *password) Authorize(ctx context.Context, req *Request, client storage.Client, conn connectors.Connector) (Responder, error) {
	passwordConnector, ok := conn.Connector.(connector.PasswordConnector)
	if !ok {
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Requested password connector does not correct type.", Status: http.StatusBadRequest}
	}

	identity, ok, err := passwordConnector.Login(ctx, tokens.ParseScopes(req.Scopes), req.Username, req.Password)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to login user", "err", err)
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Could not login user", Status: http.StatusBadRequest}
	}
	if !ok {
		return nil, &oauth2.Error{Type: oauth2.AccessDenied, Description: "Invalid username or password", Status: http.StatusUnauthorized}
	}

	auth := tokens.Authorization{
		Client:        client,
		Claims:        tokens.ClaimsFromIdentity(identity),
		Scopes:        req.Scopes,
		ConnectorID:   g.connectorID,
		Nonce:         req.Nonce,
		ConnectorData: identity.ConnectorData,
	}
	return issue(ctx, g.logger, g.issuer, auth, "", shouldIssueRefreshToken(conn, req.Scopes))
}
