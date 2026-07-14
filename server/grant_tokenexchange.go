package server

// grant_tokenexchange.go implements the token-exchange grant, RFC 8693 (dispatch in token.go).

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/storage"
)

func (s *Server) handleTokenExchange(w http.ResponseWriter, r *http.Request, client storage.Client) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		s.logger.ErrorContext(r.Context(), "could not parse request body", "err", err)
		s.tokenErrHelper(w, errInvalidRequest, "", http.StatusBadRequest)
		return
	}
	q := r.Form

	scopes := strings.Fields(q.Get("scope"))            // OPTIONAL, map to issued token scope
	requestedTokenType := q.Get("requested_token_type") // OPTIONAL, default to access token
	if requestedTokenType == "" {
		requestedTokenType = tokenTypeAccess
	}
	subjectToken := q.Get("subject_token")          // REQUIRED
	subjectTokenType := q.Get("subject_token_type") // REQUIRED
	connID := q.Get("connector_id")                 // REQUIRED, not in RFC

	switch subjectTokenType {
	case tokenTypeID, tokenTypeAccess: // ok, continue
	default:
		s.tokenErrHelper(w, errRequestNotSupported, "Invalid subject_token_type.", http.StatusBadRequest)
		return
	}

	if subjectToken == "" {
		s.tokenErrHelper(w, errInvalidRequest, "Missing subject_token", http.StatusBadRequest)
		return
	}

	if !s.checkConnectorAllowed(w, r, client, connID) {
		return
	}
	conn, err := s.getConnector(ctx, connID)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to get connector", "err", err)
		s.tokenErrHelper(w, errInvalidRequest, "Requested connector does not exist.", http.StatusBadRequest)
		return
	}
	if !GrantTypeAllowed(conn.GrantTypes, grantTypeTokenExchange) {
		s.logger.ErrorContext(r.Context(), "connector does not allow token exchange", "connector_id", connID)
		s.tokenErrHelper(w, errInvalidRequest, "Requested connector does not support token exchange.", http.StatusBadRequest)
		return
	}
	teConn, ok := conn.Connector.(connector.TokenIdentityConnector)
	if !ok {
		s.logger.ErrorContext(r.Context(), "connector doesn't implement token exchange", "connector_id", connID)
		s.tokenErrHelper(w, errInvalidRequest, "Requested connector does not exist.", http.StatusBadRequest)
		return
	}
	identity, err := teConn.TokenIdentity(ctx, subjectTokenType, subjectToken)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to verify subject token", "err", err)
		s.tokenErrHelper(w, errAccessDenied, "", http.StatusUnauthorized)
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
	resp := accessTokenResponse{
		IssuedTokenType: requestedTokenType,
		TokenType:       "bearer",
	}
	var expiry time.Time
	switch requestedTokenType {
	case tokenTypeID:
		resp.AccessToken, expiry, err = s.newIDToken(r.Context(), client.ID, claims, scopes, "", "", "", connID, time.Time{})
	case tokenTypeAccess:
		resp.AccessToken, expiry, err = s.newAccessToken(r.Context(), client.ID, claims, scopes, "", connID, time.Time{})
	default:
		s.tokenErrHelper(w, errRequestNotSupported, "Invalid requested_token_type.", http.StatusBadRequest)
		return
	}
	if err != nil {
		s.logger.ErrorContext(r.Context(), "token exchange failed to create new token", "requested_token_type", requestedTokenType, "err", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}
	resp.ExpiresIn = int(time.Until(expiry).Seconds())

	// Token response must include cache headers https://tools.ietf.org/html/rfc6749#section-5.1
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
