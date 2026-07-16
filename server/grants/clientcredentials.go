package grants

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// clientCredentials serves the client_credentials grant: a confidential client
// obtains tokens for itself, with no user involved.
type clientCredentials struct {
	issuer  *tokens.Issuer
	storage storage.Storage
	now     func() time.Time
	logger  *slog.Logger
}

// NewClientCredentials returns the client_credentials grant.
func NewClientCredentials(issuer *tokens.Issuer, s storage.Storage, now func() time.Time, logger *slog.Logger) Grant {
	return &clientCredentials{issuer: issuer, storage: s, now: now, logger: logger}
}

func (g *clientCredentials) GrantType() string        { return oauth2.GrantTypeClientCredentials }
func (g *clientCredentials) RequiresClientAuth() bool { return true }

func (g *clientCredentials) Handle(w http.ResponseWriter, r *http.Request, client storage.Client) {
	ctx := r.Context()

	// client_credentials requires a confidential client.
	if client.Public {
		g.writeError(w, oauth2.UnauthorizedClient, "Public clients cannot use client_credentials grant.", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		g.writeError(w, oauth2.InvalidRequest, "Couldn't parse data", http.StatusBadRequest)
		return
	}
	scopes := strings.Fields(r.Form.Get("scope"))

	// Validate scopes.
	var (
		unrecognized  []string
		invalidScopes []string
	)
	hasOpenIDScope := false
	for _, scope := range scopes {
		switch scope {
		case tokens.ScopeOpenID:
			hasOpenIDScope = true
		case tokens.ScopeEmail, tokens.ScopeProfile, tokens.ScopeGroups:
			// allowed
		case tokens.ScopeOfflineAccess:
			g.writeError(w, oauth2.InvalidScope, "client_credentials grant does not support offline_access scope.", http.StatusBadRequest)
			return
		case tokens.ScopeFederatedID:
			g.writeError(w, oauth2.InvalidScope, "client_credentials grant does not support federated:id scope.", http.StatusBadRequest)
			return
		default:
			peerID, ok := tokens.ParseCrossClientScope(scope)
			if !ok {
				unrecognized = append(unrecognized, scope)
				continue
			}

			isTrusted, err := tokens.CrossClientTrusted(ctx, g.storage, client.ID, peerID)
			if err != nil {
				g.logger.ErrorContext(ctx, "error validating cross client trust", "client_id", client.ID, "peer_id", peerID, "err", err)
				g.writeError(w, oauth2.InvalidClient, "Error validating cross client trust.", http.StatusBadRequest)
				return
			}
			if !isTrusted {
				invalidScopes = append(invalidScopes, scope)
			}
		}
	}
	if len(unrecognized) > 0 {
		g.writeError(w, oauth2.InvalidScope, fmt.Sprintf("Unrecognized scope(s) %q", unrecognized), http.StatusBadRequest)
		return
	}
	if len(invalidScopes) > 0 {
		g.writeError(w, oauth2.InvalidScope, fmt.Sprintf("Client can't request scope(s) %q", invalidScopes), http.StatusBadRequest)
		return
	}

	// Build claims from the client itself — no user involved.
	claims := storage.Claims{
		UserID: client.ID,
	}

	// Populate optional claims based on requested scopes.
	for _, scope := range scopes {
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

	nonce := r.Form.Get("nonce")

	// Empty connector ID is unique for client credentials grant.
	// Creating connectors with an empty ID with the config and API is prohibited.
	connID := ""

	auth := tokens.Authorization{
		Client:      client,
		Claims:      claims,
		Scopes:      scopes,
		ConnectorID: connID,
		Nonce:       nonce,
	}

	accessToken, expiry, err := g.issuer.SignAccessToken(ctx, auth)
	if err != nil {
		g.logger.ErrorContext(ctx, "client_credentials grant failed to create new access token", "err", err)
		g.writeError(w, oauth2.ServerError, "", http.StatusInternalServerError)
		return
	}

	var idToken string
	if hasOpenIDScope {
		idToken, expiry, err = g.issuer.SignIDToken(ctx, auth, accessToken, "")
		if err != nil {
			g.logger.ErrorContext(ctx, "client_credentials grant failed to create new ID token", "err", err)
			g.writeError(w, oauth2.ServerError, "", http.StatusInternalServerError)
			return
		}
	}

	if err := writeTokenResponse(w, tokens.TokenSet{AccessToken: accessToken, IDToken: idToken, Expiry: expiry}, g.now()); err != nil {
		g.logger.ErrorContext(ctx, "failed to write token response", "err", err)
		g.writeError(w, oauth2.ServerError, "", http.StatusInternalServerError)
		return
	}
}

func (g *clientCredentials) writeError(w http.ResponseWriter, typ, description string, statusCode int) {
	writeError(g.logger, w, typ, description, statusCode)
}
