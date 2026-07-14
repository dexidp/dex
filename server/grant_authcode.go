package server

// grant_authcode.go implements the authorization_code grant (dispatch in token.go).

import (
	"context"
	"net/http"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

// handle an access token request https://tools.ietf.org/html/rfc6749#section-4.1.3
func (s *Server) handleAuthCode(w http.ResponseWriter, r *http.Request, client storage.Client) {
	ctx := r.Context()
	code := r.PostFormValue("code")
	redirectURI := r.PostFormValue("redirect_uri")

	if code == "" {
		s.tokenErrHelper(w, errInvalidRequest, `Required param: code.`, http.StatusBadRequest)
		return
	}

	authCode, err := s.storage.GetAuthCode(ctx, code)
	if err != nil || s.now().After(authCode.Expiry) || authCode.ClientID != client.ID {
		if err != storage.ErrNotFound {
			s.logger.ErrorContext(r.Context(), "failed to get auth code", "err", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		} else {
			s.tokenErrHelper(w, errInvalidGrant, "Invalid or expired code parameter.", http.StatusBadRequest)
		}
		return
	}

	// RFC 7636 (PKCE)
	codeChallengeFromStorage := authCode.PKCE.CodeChallenge
	providedCodeVerifier := r.PostFormValue("code_verifier")

	switch {
	case providedCodeVerifier != "" && codeChallengeFromStorage != "":
		calculatedCodeChallenge, err := s.calculateCodeChallenge(providedCodeVerifier, authCode.PKCE.CodeChallengeMethod)
		if err != nil {
			s.logger.ErrorContext(r.Context(), "failed to calculate code challenge", "err", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
			return
		}
		if codeChallengeFromStorage != calculatedCodeChallenge {
			s.tokenErrHelper(w, errInvalidGrant, "Invalid code_verifier.", http.StatusBadRequest)
			return
		}
	case providedCodeVerifier != "":
		// Received no code_challenge on /auth, but a code_verifier on /token
		s.tokenErrHelper(w, errInvalidRequest, "No PKCE flow started. Cannot check code_verifier.", http.StatusBadRequest)
		return
	case codeChallengeFromStorage != "":
		// Received PKCE request on /auth, but no code_verifier on /token
		s.tokenErrHelper(w, errInvalidGrant, "Expecting parameter code_verifier in PKCE flow.", http.StatusBadRequest)
		return
	}

	if authCode.RedirectURI != redirectURI {
		s.tokenErrHelper(w, errInvalidRequest, "redirect_uri did not match URI from initial request.", http.StatusBadRequest)
		return
	}

	tokenResponse, err := s.exchangeAuthCode(ctx, w, authCode, client)
	if err != nil {
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}
	s.writeAccessToken(w, tokenResponse)
}

func (s *Server) exchangeAuthCode(ctx context.Context, w http.ResponseWriter, authCode storage.AuthCode, client storage.Client) (*accessTokenResponse, error) {
	accessToken, _, err := s.newAccessToken(ctx, client.ID, authCode.Claims, authCode.Scopes, authCode.Nonce, authCode.ConnectorID, authCode.AuthTime)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create new access token", "err", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return nil, err
	}

	idToken, expiry, err := s.newIDToken(ctx, client.ID, authCode.Claims, authCode.Scopes, authCode.Nonce, accessToken, authCode.ID, authCode.ConnectorID, authCode.AuthTime)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create ID token", "err", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return nil, err
	}

	if err := s.storage.DeleteAuthCode(ctx, authCode.ID); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete auth code", "err", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return nil, err
	}

	reqRefresh := func() bool {
		// Determine whether to issue a refresh token. A refresh token is only
		// issued when all of the following are true:
		//   1. The connector implements RefreshConnector.
		//   2. The connector's grantTypes config allows refresh_token.
		//   3. The client requested the offline_access scope.
		//
		// When any condition is not met, the refresh token is silently omitted
		// rather than returning an error. This matches the OAuth2 spec: the
		// server is never required to issue a refresh token (RFC 6749 §1.5).
		// https://datatracker.ietf.org/doc/html/rfc6749#section-1.5
		conn, err := s.getConnector(ctx, authCode.ConnectorID)
		if err != nil {
			s.logger.ErrorContext(ctx, "connector not found", "connector_id", authCode.ConnectorID, "err", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
			return false
		}

		_, ok := conn.Connector.(connector.RefreshConnector)
		if !ok {
			return false
		}

		if !GrantTypeAllowed(conn.GrantTypes, grantTypeRefreshToken) {
			return false
		}

		for _, scope := range authCode.Scopes {
			if scope == scopeOfflineAccess {
				return true
			}
		}
		return false
	}()
	var refreshToken string
	if reqRefresh {
		refresh := storage.RefreshToken{
			ID:            storage.NewID(),
			Token:         storage.NewID(),
			ClientID:      authCode.ClientID,
			ConnectorID:   authCode.ConnectorID,
			Scopes:        authCode.Scopes,
			Claims:        authCode.Claims,
			Nonce:         authCode.Nonce,
			ConnectorData: authCode.ConnectorData,
			CreatedAt:     s.now(),
			LastUsed:      s.now(),
		}
		token := &internal.RefreshToken{
			RefreshId: refresh.ID,
			Token:     refresh.Token,
		}
		if refreshToken, err = internal.Marshal(token); err != nil {
			s.logger.ErrorContext(ctx, "failed to marshal refresh token", "err", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
			return nil, err
		}

		if err := s.storage.CreateRefresh(ctx, refresh); err != nil {
			s.logger.ErrorContext(ctx, "failed to create refresh token", "err", err)
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
			return nil, err
		}

		// deleteToken determines if we need to delete the newly created refresh token
		// due to a failure in updating/creating the OfflineSession object for the
		// corresponding user.
		var deleteToken bool
		defer func() {
			if deleteToken {
				// Delete newly created refresh token from storage.
				if err := s.storage.DeleteRefresh(ctx, refresh.ID); err != nil {
					s.logger.ErrorContext(ctx, "failed to delete refresh token", "err", err)
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
				s.logger.ErrorContext(ctx, "failed to get offline session", "err", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return nil, err
			}
			offlineSessions := storage.OfflineSessions{
				UserID:        refresh.Claims.UserID,
				ConnID:        refresh.ConnectorID,
				Refresh:       make(map[string]*storage.RefreshTokenRef),
				ConnectorData: refresh.ConnectorData,
			}
			offlineSessions.Refresh[tokenRef.ClientID] = &tokenRef

			// Create a new OfflineSession object for the user and add a reference object for
			// the newly received refreshtoken.
			if err := s.storage.CreateOfflineSessions(ctx, offlineSessions); err != nil {
				s.logger.ErrorContext(ctx, "failed to create offline session", "err", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return nil, err
			}
		} else {
			if oldTokenRef, ok := session.Refresh[tokenRef.ClientID]; ok {
				// Delete old refresh token from storage.
				if err := s.storage.DeleteRefresh(ctx, oldTokenRef.ID); err != nil && err != storage.ErrNotFound {
					s.logger.ErrorContext(ctx, "failed to delete refresh token", "err", err)
					s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
					deleteToken = true
					return nil, err
				}
			}

			// Update existing OfflineSession obj with new RefreshTokenRef.
			if err := s.storage.UpdateOfflineSessions(ctx, session.UserID, session.ConnID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
				old.Refresh[tokenRef.ClientID] = &tokenRef
				if len(refresh.ConnectorData) > 0 {
					old.ConnectorData = refresh.ConnectorData
				}
				return old, nil
			}); err != nil {
				s.logger.ErrorContext(ctx, "failed to update offline session", "err", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return nil, err
			}
		}
	}
	return s.toAccessTokenResponse(idToken, accessToken, refreshToken, expiry), nil
}
