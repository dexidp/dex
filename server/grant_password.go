package server

// grant_password.go implements the password grant (dispatch in token.go).

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

func (s *Server) handlePasswordGrant(w http.ResponseWriter, r *http.Request, client storage.Client) {
	ctx := r.Context()
	// Parse the fields
	if err := r.ParseForm(); err != nil {
		s.tokenErrHelper(w, oauth2.InvalidRequest, "Couldn't parse data", http.StatusBadRequest)
		return
	}
	q := r.Form

	nonce := q.Get("nonce")
	// Some clients, like the old go-oidc, provide extra whitespace. Tolerate this.
	scopes := strings.Fields(q.Get("scope"))

	// Parse the scopes if they are passed
	var (
		unrecognized  []string
		invalidScopes []string
	)
	hasOpenIDScope := false
	for _, scope := range scopes {
		switch scope {
		case tokens.ScopeOpenID:
			hasOpenIDScope = true
		case tokens.ScopeOfflineAccess, tokens.ScopeEmail, tokens.ScopeProfile, tokens.ScopeGroups, tokens.ScopeFederatedID:
		default:
			peerID, ok := tokens.ParseCrossClientScope(scope)
			if !ok {
				unrecognized = append(unrecognized, scope)
				continue
			}

			isTrusted, err := s.validateCrossClientTrust(ctx, client.ID, peerID)
			if err != nil {
				s.logger.ErrorContext(r.Context(), "error validating cross client trust", "client_id", client.ID, "peer_id", peerID, "err", err)
				s.tokenErrHelper(w, oauth2.InvalidClient, "Error validating cross client trust.", http.StatusBadRequest)
				return
			}
			if !isTrusted {
				invalidScopes = append(invalidScopes, scope)
			}
		}
	}
	if !hasOpenIDScope {
		s.tokenErrHelper(w, oauth2.InvalidRequest, `Missing required scope(s) ["openid"].`, http.StatusBadRequest)
		return
	}
	if len(unrecognized) > 0 {
		s.tokenErrHelper(w, oauth2.InvalidRequest, fmt.Sprintf("Unrecognized scope(s) %q", unrecognized), http.StatusBadRequest)
		return
	}
	if len(invalidScopes) > 0 {
		s.tokenErrHelper(w, oauth2.InvalidRequest, fmt.Sprintf("Client can't request scope(s) %q", invalidScopes), http.StatusBadRequest)
		return
	}

	// Which connector
	connID := s.passwordConnector
	if !s.checkConnectorAllowed(w, r, client, connID) {
		return
	}
	conn, err := s.connectors.Get(ctx, connID)
	if err != nil {
		s.tokenErrHelper(w, oauth2.InvalidRequest, "Requested connector does not exist.", http.StatusBadRequest)
		return
	}
	if !GrantTypeAllowed(conn.GrantTypes, oauth2.GrantTypePassword) {
		s.logger.ErrorContext(r.Context(), "connector does not allow password grant", "connector_id", connID)
		s.tokenErrHelper(w, oauth2.InvalidRequest, "Requested connector does not support password grant.", http.StatusBadRequest)
		return
	}

	passwordConnector, ok := conn.Connector.(connector.PasswordConnector)
	if !ok {
		s.tokenErrHelper(w, oauth2.InvalidRequest, "Requested password connector does not correct type.", http.StatusBadRequest)
		return
	}

	// Login
	username := q.Get("username")
	password := q.Get("password")
	identity, ok, err := passwordConnector.Login(ctx, parseScopes(scopes), username, password)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to login user", "err", err)
		s.tokenErrHelper(w, oauth2.InvalidRequest, "Could not login user", http.StatusBadRequest)
		return
	}
	if !ok {
		s.tokenErrHelper(w, oauth2.AccessDenied, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Build the claims to send the id token
	claims := storage.Claims{
		UserID:            identity.UserID,
		Username:          identity.Username,
		PreferredUsername: identity.PreferredUsername,
		Email:             identity.Email,
		EmailVerified:     identity.EmailVerified,
		Groups:            identity.Groups,
	}

	// A refresh token is issued only when the connector supports it, the grant type
	// is allowed, and offline_access was requested (RFC 6749 §1.5, never mandatory).
	wantRefresh := false
	if _, ok := conn.Connector.(connector.RefreshConnector); ok && GrantTypeAllowed(conn.GrantTypes, oauth2.GrantTypeRefreshToken) {
		wantRefresh = slices.Contains(scopes, tokens.ScopeOfflineAccess)
	}

	ts, err := s.issuer.Issue(ctx, tokens.Authorization{
		Client:        client,
		Claims:        claims,
		Scopes:        scopes,
		ConnectorID:   connID,
		Nonce:         nonce,
		ConnectorData: identity.ConnectorData,
	}, wantRefresh)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "password grant failed to issue tokens", "err", err)
		s.tokenErrHelper(w, oauth2.ServerError, "", http.StatusInternalServerError)
		return
	}

	if err := writeTokenResponse(w, ts, s.now()); err != nil {
		s.logger.ErrorContext(r.Context(), "failed to write token response", "err", err)
		s.tokenErrHelper(w, oauth2.ServerError, "", http.StatusInternalServerError)
		return
	}
}
