package server

// grant_password.go implements the password grant (dispatch in token.go).

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

func (s *Server) handlePasswordGrant(w http.ResponseWriter, r *http.Request, client storage.Client) {
	ctx := r.Context()
	// Parse the fields
	if err := r.ParseForm(); err != nil {
		s.tokenErrHelper(w, errInvalidRequest, "Couldn't parse data", http.StatusBadRequest)
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
		case scopeOpenID:
			hasOpenIDScope = true
		case scopeOfflineAccess, scopeEmail, scopeProfile, scopeGroups, scopeFederatedID:
		default:
			peerID, ok := parseCrossClientScope(scope)
			if !ok {
				unrecognized = append(unrecognized, scope)
				continue
			}

			isTrusted, err := s.validateCrossClientTrust(ctx, client.ID, peerID)
			if err != nil {
				s.logger.ErrorContext(r.Context(), "error validating cross client trust", "client_id", client.ID, "peer_id", peerID, "err", err)
				s.tokenErrHelper(w, errInvalidClient, "Error validating cross client trust.", http.StatusBadRequest)
				return
			}
			if !isTrusted {
				invalidScopes = append(invalidScopes, scope)
			}
		}
	}
	if !hasOpenIDScope {
		s.tokenErrHelper(w, errInvalidRequest, `Missing required scope(s) ["openid"].`, http.StatusBadRequest)
		return
	}
	if len(unrecognized) > 0 {
		s.tokenErrHelper(w, errInvalidRequest, fmt.Sprintf("Unrecognized scope(s) %q", unrecognized), http.StatusBadRequest)
		return
	}
	if len(invalidScopes) > 0 {
		s.tokenErrHelper(w, errInvalidRequest, fmt.Sprintf("Client can't request scope(s) %q", invalidScopes), http.StatusBadRequest)
		return
	}

	// Which connector
	connID := s.passwordConnector
	if !s.checkConnectorAllowed(w, r, client, connID) {
		return
	}
	conn, err := s.getConnector(ctx, connID)
	if err != nil {
		s.tokenErrHelper(w, errInvalidRequest, "Requested connector does not exist.", http.StatusBadRequest)
		return
	}
	if !GrantTypeAllowed(conn.GrantTypes, grantTypePassword) {
		s.logger.ErrorContext(r.Context(), "connector does not allow password grant", "connector_id", connID)
		s.tokenErrHelper(w, errInvalidRequest, "Requested connector does not support password grant.", http.StatusBadRequest)
		return
	}

	passwordConnector, ok := conn.Connector.(connector.PasswordConnector)
	if !ok {
		s.tokenErrHelper(w, errInvalidRequest, "Requested password connector does not correct type.", http.StatusBadRequest)
		return
	}

	// Login
	username := q.Get("username")
	password := q.Get("password")
	identity, ok, err := passwordConnector.Login(ctx, parseScopes(scopes), username, password)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to login user", "err", err)
		s.tokenErrHelper(w, errInvalidRequest, "Could not login user", http.StatusBadRequest)
		return
	}
	if !ok {
		s.tokenErrHelper(w, errAccessDenied, "Invalid username or password", http.StatusUnauthorized)
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

	accessToken, _, err := s.newAccessToken(ctx, client.ID, claims, scopes, nonce, connID, time.Time{})
	if err != nil {
		s.logger.ErrorContext(r.Context(), "password grant failed to create new access token", "err", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	idToken, expiry, err := s.newIDToken(ctx, client.ID, claims, scopes, nonce, accessToken, "", connID, time.Time{})
	if err != nil {
		s.logger.ErrorContext(r.Context(), "password grant failed to create new ID token", "err", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	reqRefresh := func() bool {
		// Same logic as in exchangeAuthCode: silently omit refresh token
		// when the connector doesn't support it or grantTypes forbids it.
		// See RFC 6749 §1.5 — refresh tokens are never mandatory.
		// https://datatracker.ietf.org/doc/html/rfc6749#section-1.5
		if _, ok := conn.Connector.(connector.RefreshConnector); !ok {
			return false
		}

		if !GrantTypeAllowed(conn.GrantTypes, grantTypeRefreshToken) {
			return false
		}

		for _, scope := range scopes {
			if scope == scopeOfflineAccess {
				return true
			}
		}
		return false
	}()
	var refreshToken string
	if reqRefresh {
		refresh := storage.RefreshToken{
			ID:          storage.NewID(),
			Token:       storage.NewID(),
			ClientID:    client.ID,
			ConnectorID: connID,
			Scopes:      scopes,
			Claims:      claims,
			Nonce:       nonce,
			// ConnectorData: authCode.ConnectorData,
			CreatedAt: s.now(),
			LastUsed:  s.now(),
		}
		token := &internal.RefreshToken{
			RefreshId: refresh.ID,
			Token:     refresh.Token,
		}
		if refreshToken, err = internal.Marshal(token); err != nil {
			s.logger.ErrorContext(r.Context(), "failed to marshal refresh token", "err", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
			return
		}

		if err := s.storage.CreateRefresh(ctx, refresh); err != nil {
			s.logger.ErrorContext(r.Context(), "failed to create refresh token", "err", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
			return
		}

		// deleteToken determines if we need to delete the newly created refresh token
		// due to a failure in updating/creating the OfflineSession object for the
		// corresponding user.
		var deleteToken bool
		defer func() {
			if deleteToken {
				// Delete newly created refresh token from storage.
				if err := s.storage.DeleteRefresh(ctx, refresh.ID); err != nil {
					s.logger.ErrorContext(r.Context(), "failed to delete refresh token", "err", err)
					s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
					return
				}
			}
		}()

		tokenRef := storage.RefreshTokenRef{
			ID:        refresh.ID,
			ClientID:  refresh.ClientID,
			CreatedAt: refresh.CreatedAt,
			LastUsed:  refresh.LastUsed,
		}

		// Try to retrieve an existing OfflineSession object for the corresponding user.
		if session, err := s.storage.GetOfflineSessions(ctx, refresh.Claims.UserID, refresh.ConnectorID); err != nil {
			if err != storage.ErrNotFound {
				s.logger.ErrorContext(r.Context(), "failed to get offline session", "err", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return
			}
			offlineSessions := storage.OfflineSessions{
				UserID:        refresh.Claims.UserID,
				ConnID:        refresh.ConnectorID,
				Refresh:       make(map[string]*storage.RefreshTokenRef),
				ConnectorData: identity.ConnectorData,
			}
			offlineSessions.Refresh[tokenRef.ClientID] = &tokenRef

			// Create a new OfflineSession object for the user and add a reference object for
			// the newly received refreshtoken.
			if err := s.storage.CreateOfflineSessions(ctx, offlineSessions); err != nil {
				s.logger.ErrorContext(r.Context(), "failed to create offline session", "err", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return
			}
		} else {
			if oldTokenRef, ok := session.Refresh[tokenRef.ClientID]; ok {
				// Delete old refresh token from storage.
				if err := s.storage.DeleteRefresh(ctx, oldTokenRef.ID); err != nil {
					if err == storage.ErrNotFound {
						s.logger.Warn("database inconsistent, refresh token missing", "token_id", oldTokenRef.ID)
					} else {
						s.logger.ErrorContext(r.Context(), "failed to delete refresh token", "err", err)
						s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
						deleteToken = true
						return
					}
				}
			}

			// Update existing OfflineSession obj with new RefreshTokenRef.
			if err := s.storage.UpdateOfflineSessions(ctx, session.UserID, session.ConnID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
				old.Refresh[tokenRef.ClientID] = &tokenRef
				old.ConnectorData = identity.ConnectorData
				return old, nil
			}); err != nil {
				s.logger.ErrorContext(r.Context(), "failed to update offline session", "err", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return
			}
		}
	}

	resp := s.toAccessTokenResponse(idToken, accessToken, refreshToken, expiry)
	s.writeAccessToken(w, resp)
}
