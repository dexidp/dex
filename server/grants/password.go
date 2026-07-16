package grants

import (
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
	logger      *slog.Logger
	connectorID string
}

func (g *password) GrantType() string {
	return oauth2.GrantTypePassword
}

func (g *password) RequiresClientAuth() bool {
	return true
}

func (g *password) Scopes() ScopePolicy {
	return ScopePolicy{
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
}

// ConnectorID is the connector the password grant is configured to use.
func (g *password) ConnectorID(r *http.Request) string {
	return g.connectorID
}

func (g *password) Authorize(r *http.Request, client storage.Client, scopes []string, conn connectors.Connector) (*Result, error) {
	ctx := r.Context()

	passwordConnector, ok := conn.Connector.(connector.PasswordConnector)
	if !ok {
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Requested password connector does not correct type.", Status: http.StatusBadRequest}
	}

	username := r.PostFormValue("username")
	pw := r.PostFormValue("password")
	identity, ok, err := passwordConnector.Login(ctx, tokens.ParseScopes(scopes), username, pw)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to login user", "err", err)
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Could not login user", Status: http.StatusBadRequest}
	}
	if !ok {
		return nil, &oauth2.Error{Type: oauth2.AccessDenied, Description: "Invalid username or password", Status: http.StatusUnauthorized}
	}

	return &Result{
		Authorization: tokens.Authorization{
			Client: client,
			Claims: storage.Claims{
				UserID:            identity.UserID,
				Username:          identity.Username,
				PreferredUsername: identity.PreferredUsername,
				Email:             identity.Email,
				EmailVerified:     identity.EmailVerified,
				Groups:            identity.Groups,
			},
			Scopes:        scopes,
			ConnectorID:   g.connectorID,
			Nonce:         r.PostFormValue("nonce"),
			ConnectorData: identity.ConnectorData,
		},
		IssueRefresh: shouldIssueRefreshToken(conn, scopes),
	}, nil
}
