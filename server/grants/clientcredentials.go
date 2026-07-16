package grants

import (
	"context"
	"net/http"

	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// clientCredentials serves the client_credentials grant: a confidential client
// obtains tokens for itself, with no user involved.
type clientCredentials struct{}

func (g *clientCredentials) GrantType() string {
	return oauth2.GrantTypeClientCredentials
}

func (g *clientCredentials) RequiresClientAuth() bool {
	return true
}

var clientCredentialsScopePolicy = ScopePolicy{
	Standard: map[string]bool{
		tokens.ScopeOpenID:  true,
		tokens.ScopeEmail:   true,
		tokens.ScopeProfile: true,
		tokens.ScopeGroups:  true,
	},
	Rejected: map[string]string{
		tokens.ScopeOfflineAccess: "client_credentials grant does not support offline_access scope.",
		tokens.ScopeFederatedID:   "client_credentials grant does not support federated:id scope.",
	},
	ErrorType: oauth2.InvalidScope,
}

func (g *clientCredentials) ScopePolicy() ScopePolicy {
	return clientCredentialsScopePolicy
}

// ConnectorID is empty: client_credentials involves no connector.
func (g *clientCredentials) ConnectorID(ctx context.Context, req *Request, client storage.Client) (string, *oauth2.Error) {
	return "", nil
}

func (g *clientCredentials) Authorize(ctx context.Context, req *Request, client storage.Client, conn connectors.Connector) (*Result, error) {
	// client_credentials requires a confidential client.
	if client.Public {
		return nil, &oauth2.Error{Type: oauth2.UnauthorizedClient, Description: "Public clients cannot use client_credentials grant.", Status: http.StatusBadRequest}
	}

	// Build claims from the client itself — no user involved.
	claims := storage.Claims{UserID: client.ID}
	for _, scope := range req.Scopes {
		switch scope {
		case tokens.ScopeProfile:
			claims.Username = client.Name
			claims.PreferredUsername = client.Name
		case tokens.ScopeGroups:
			if client.ClientCredentialsClaims != nil {
				claims.Groups = client.ClientCredentialsClaims.Groups
			}
		}
	}

	return &Result{
		Authorization: tokens.Authorization{
			Client: client,
			Claims: claims,
			Scopes: req.Scopes,
			// Empty connector ID is unique for client credentials grant. Creating
			// connectors with an empty ID via the config and API is prohibited.
			ConnectorID: "",
			Nonce:       req.Nonce,
		},
	}, nil
}
