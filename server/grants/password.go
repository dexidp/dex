package grants

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// password serves the Resource Owner Password Credentials grant: the client
// exchanges a username and password for tokens via a password-capable connector.
type password struct {
	issuer          *tokens.Issuer
	logger          *slog.Logger
	connectorID     string
	storage         storage.Storage
	sessionsEnabled bool
	now             func() time.Time
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

	// Persist the user identity the same way the browser flow's finalize step
	// does, so a refresh under sessions can resolve the cached claims instead
	// of failing on a missing identity. Without this row a password-grant
	// refresh token can never be used while sessions are enabled.
	if g.sessionsEnabled {
		if oerr := g.persistUserIdentity(ctx, identity); oerr != nil {
			return nil, oerr
		}
	}

	auth := tokens.Authorization{
		Client:        client,
		Claims:        tokens.ClaimsFromIdentity(identity),
		Scopes:        req.Scopes,
		ConnectorID:   g.connectorID,
		Nonce:         req.Nonce,
		ConnectorData: identity.ConnectorData,
	}
	return issueTokens(ctx, g.logger, g.issuer, auth, "", shouldIssueRefreshToken(conn, req.Scopes))
}

// persistUserIdentity creates the user's identity row on first login and
// refreshes its claims and last-login time on later ones — the same write the
// browser flow performs in server/authflow/finalize.go.
func (g *password) persistUserIdentity(ctx context.Context, identity connector.Identity) *oauth2.Error {
	now := g.now()

	_, err := g.storage.GetUserIdentity(ctx, identity.UserID, g.connectorID)
	switch {
	case errors.Is(err, storage.ErrNotFound):
		ui := storage.UserIdentity{
			UserID:      identity.UserID,
			ConnectorID: g.connectorID,
			Claims:      tokens.ClaimsFromIdentity(identity),
			Consents:    make(map[string][]string),
			CreatedAt:   now,
			LastLogin:   now,
		}
		if err := g.storage.CreateUserIdentity(ctx, ui); err != nil {
			g.logger.ErrorContext(ctx, "failed to create user identity", "err", err)
			return &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
		}
	case err == nil:
		if err := g.storage.UpdateUserIdentity(ctx, identity.UserID, g.connectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
			old.Claims = tokens.ClaimsFromIdentity(identity)
			old.LastLogin = now
			return old, nil
		}); err != nil {
			g.logger.ErrorContext(ctx, "failed to update user identity", "err", err)
			return &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
		}
	default:
		g.logger.ErrorContext(ctx, "failed to get user identity", "err", err)
		return &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
	}
	return nil
}
