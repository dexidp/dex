package server

// grant_clientcredentials.go implements the client_credentials grant (dispatch in token.go).

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

func (s *Server) handleClientCredentialsGrant(w http.ResponseWriter, r *http.Request, client storage.Client) {
	ctx := r.Context()

	// client_credentials requires a confidential client.
	if client.Public {
		s.tokenErrHelper(w, oauth2.UnauthorizedClient, "Public clients cannot use client_credentials grant.", http.StatusBadRequest)
		return
	}

	// Parse scopes from request.
	if err := r.ParseForm(); err != nil {
		s.tokenErrHelper(w, oauth2.InvalidRequest, "Couldn't parse data", http.StatusBadRequest)
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
			s.tokenErrHelper(w, oauth2.InvalidScope, "client_credentials grant does not support offline_access scope.", http.StatusBadRequest)
			return
		case tokens.ScopeFederatedID:
			s.tokenErrHelper(w, oauth2.InvalidScope, "client_credentials grant does not support federated:id scope.", http.StatusBadRequest)
			return
		default:
			peerID, ok := tokens.ParseCrossClientScope(scope)
			if !ok {
				unrecognized = append(unrecognized, scope)
				continue
			}

			isTrusted, err := s.validateCrossClientTrust(ctx, client.ID, peerID)
			if err != nil {
				s.logger.ErrorContext(ctx, "error validating cross client trust", "client_id", client.ID, "peer_id", peerID, "err", err)
				s.tokenErrHelper(w, oauth2.InvalidClient, "Error validating cross client trust.", http.StatusBadRequest)
				return
			}
			if !isTrusted {
				invalidScopes = append(invalidScopes, scope)
			}
		}
	}
	if len(unrecognized) > 0 {
		s.tokenErrHelper(w, oauth2.InvalidScope, fmt.Sprintf("Unrecognized scope(s) %q", unrecognized), http.StatusBadRequest)
		return
	}
	if len(invalidScopes) > 0 {
		s.tokenErrHelper(w, oauth2.InvalidScope, fmt.Sprintf("Client can't request scope(s) %q", invalidScopes), http.StatusBadRequest)
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

	// Empty connector ID is unique for cluster credentials grant
	// Creating connectors with an empty ID with the config and API is prohibited
	connID := ""

	auth := tokens.Authorization{
		Client:      client,
		Claims:      claims,
		Scopes:      scopes,
		ConnectorID: connID,
		Nonce:       nonce,
	}

	accessToken, expiry, err := s.issuer.SignAccessToken(ctx, auth)
	if err != nil {
		s.logger.ErrorContext(ctx, "client_credentials grant failed to create new access token", "err", err)
		s.tokenErrHelper(w, oauth2.ServerError, "", http.StatusInternalServerError)
		return
	}

	var idToken string
	if hasOpenIDScope {
		idToken, expiry, err = s.issuer.SignIDToken(ctx, auth, accessToken, "")
		if err != nil {
			s.logger.ErrorContext(ctx, "client_credentials grant failed to create new ID token", "err", err)
			s.tokenErrHelper(w, oauth2.ServerError, "", http.StatusInternalServerError)
			return
		}
	}

	if err := writeTokenResponse(w, tokens.TokenSet{AccessToken: accessToken, IDToken: idToken, Expiry: expiry}, s.now()); err != nil {
		s.logger.ErrorContext(ctx, "failed to write token response", "err", err)
		s.tokenErrHelper(w, oauth2.ServerError, "", http.StatusInternalServerError)
		return
	}
}
