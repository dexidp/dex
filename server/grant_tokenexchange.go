package server

// grant_tokenexchange.go implements the token-exchange grant, RFC 8693 (dispatch in token.go).

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

func (s *Server) handleTokenExchange(w http.ResponseWriter, r *http.Request, client storage.Client) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		s.logger.ErrorContext(r.Context(), "could not parse request body", "err", err)
		s.tokenErrHelper(w, oauth2.InvalidRequest, "", http.StatusBadRequest)
		return
	}
	q := r.Form

	scopes := strings.Fields(q.Get("scope"))            // OPTIONAL, map to issued token scope
	requestedTokenType := q.Get("requested_token_type") // OPTIONAL, default to access token
	if requestedTokenType == "" {
		requestedTokenType = oauth2.TokenTypeAccess
	}
	subjectToken := q.Get("subject_token")          // REQUIRED
	subjectTokenType := q.Get("subject_token_type") // REQUIRED
	connID := q.Get("connector_id")                 // REQUIRED, not in RFC

	switch subjectTokenType {
	case oauth2.TokenTypeID, oauth2.TokenTypeAccess: // ok, continue
	default:
		s.tokenErrHelper(w, oauth2.RequestNotSupported, "Invalid subject_token_type.", http.StatusBadRequest)
		return
	}

	if subjectToken == "" {
		s.tokenErrHelper(w, oauth2.InvalidRequest, "Missing subject_token", http.StatusBadRequest)
		return
	}

	if !s.checkConnectorAllowed(w, r, client, connID) {
		return
	}
	conn, err := s.connectors.Get(ctx, connID)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to get connector", "err", err)
		s.tokenErrHelper(w, oauth2.InvalidRequest, "Requested connector does not exist.", http.StatusBadRequest)
		return
	}
	if !connectors.GrantTypeAllowed(conn.GrantTypes, oauth2.GrantTypeTokenExchange) {
		s.logger.ErrorContext(r.Context(), "connector does not allow token exchange", "connector_id", connID)
		s.tokenErrHelper(w, oauth2.InvalidRequest, "Requested connector does not support token exchange.", http.StatusBadRequest)
		return
	}
	teConn, ok := conn.Connector.(connector.TokenIdentityConnector)
	if !ok {
		s.logger.ErrorContext(r.Context(), "connector doesn't implement token exchange", "connector_id", connID)
		s.tokenErrHelper(w, oauth2.InvalidRequest, "Requested connector does not exist.", http.StatusBadRequest)
		return
	}
	identity, err := teConn.TokenIdentity(ctx, subjectTokenType, subjectToken)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to verify subject token", "err", err)
		s.tokenErrHelper(w, oauth2.AccessDenied, "", http.StatusUnauthorized)
		return
	}

	email := identity.Email
	if !identity.EmailVerified {
		email += " (unverified)"
	}

	s.logger.InfoContext(ctx, "token exchange successful",
		"connector_id", connID, "client_id", client.ID,
		"user_id", identity.UserID,
		"username", identity.Username, "preferred_username", identity.PreferredUsername,
		"email", email, "groups", identity.Groups,
		"subject_token_type", subjectTokenType, "requested_token_type", requestedTokenType)

	claims := storage.Claims{
		UserID:            identity.UserID,
		Username:          identity.Username,
		PreferredUsername: identity.PreferredUsername,
		Email:             identity.Email,
		EmailVerified:     identity.EmailVerified,
		Groups:            identity.Groups,
	}
	resp := tokens.Response{
		IssuedTokenType: requestedTokenType,
		TokenType:       "bearer",
	}
	auth := tokens.Authorization{
		Client:      client,
		Claims:      claims,
		Scopes:      scopes,
		ConnectorID: connID,
	}
	var expiry time.Time
	switch requestedTokenType {
	case oauth2.TokenTypeID:
		resp.AccessToken, expiry, err = s.issuer.SignIDToken(r.Context(), auth, "", "")
	case oauth2.TokenTypeAccess:
		resp.AccessToken, expiry, err = s.issuer.SignAccessToken(r.Context(), auth)
	default:
		s.tokenErrHelper(w, oauth2.RequestNotSupported, "Invalid requested_token_type.", http.StatusBadRequest)
		return
	}
	if err != nil {
		s.logger.ErrorContext(r.Context(), "token exchange failed to create new token", "requested_token_type", requestedTokenType, "err", err)
		s.tokenErrHelper(w, oauth2.ServerError, "", http.StatusInternalServerError)
		return
	}
	resp.ExpiresIn = int(time.Until(expiry).Seconds())

	// Token response must include cache headers https://tools.ietf.org/html/rfc6749#section-5.1
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
