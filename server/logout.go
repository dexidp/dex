package server

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

// handleLogout implements OIDC RP-Initiated Logout (https://openid.net/specs/openid-connect-rpinitiated-1_0.html).
//
// GET /logout?id_token_hint=...&post_logout_redirect_uri=...&state=...
//
// Flow:
//  1. Validate id_token_hint (signature + issuer; expiry skipped per spec)
//  2. Extract user identity (subject) and client (audience/azp) from the token
//  3. Validate post_logout_redirect_uri against the client's registered URIs
//  4. Revoke refresh tokens for the user/connector pair
//  5. If the auth session exists and upstream connector implements LogoutCallbackConnector:
//     a. Store LogoutState + HMAC key in the session (not deleted yet)
//     b. Redirect to upstream logout with signed state
//     c. On callback: verify HMAC, read LogoutState from session, delete session, render page
//  6. If no session or no upstream logout support: delete session, clear cookie, render page
//
// Upstream redirect requires a live AuthSession because the session stores the
// HMAC key and logout parameters. Without a session (e.g. already expired, or
// id_token_hint without a cookie) upstream logout is skipped — this is acceptable
// because RP-Initiated Logout treats upstream SLO as best-effort.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		s.renderError(r, w, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}

	idTokenHint := r.FormValue("id_token_hint")
	postLogoutRedirectURI := r.FormValue("post_logout_redirect_uri")
	state := r.FormValue("state")

	var userID, connectorID, clientID string

	if idTokenHint != "" {
		idToken, err := s.validateIDTokenHint(ctx, idTokenHint)
		if err != nil {
			s.logger.ErrorContext(ctx, "logout: invalid id_token_hint", "err", err)
			s.renderError(r, w, http.StatusBadRequest, "Invalid id_token_hint.")
			return
		}

		sub := new(internal.IDTokenSubject)
		if err := internal.Unmarshal(idToken.Subject, sub); err != nil {
			s.logger.ErrorContext(ctx, "logout: failed to unmarshal subject", "err", err)
			s.renderError(r, w, http.StatusBadRequest, "Invalid id_token_hint subject.")
			return
		}

		userID = sub.UserId
		connectorID = sub.ConnId

		s.logger.DebugContext(ctx, "logout: parsed id_token_hint",
			"user_id", userID, "connector_id", connectorID)

		// When cross-client (trusted peers) scopes are used, the token may have
		// multiple audiences. In that case the requesting client is in the "azp"
		// claim, not necessarily Audience[0]. Use the same logic as token introspection.
		var claims struct {
			AuthorizingParty string `json:"azp"`
		}
		if err := idToken.Claims(&claims); err != nil {
			s.logger.ErrorContext(ctx, "logout: failed to decode id_token_hint claims", "err", err)
			s.renderError(r, w, http.StatusBadRequest, "Invalid id_token_hint.")
			return
		}

		switch len(idToken.Audience) {
		case 0:
			// No audience — cannot determine client.
		case 1:
			clientID = idToken.Audience[0]
		default:
			clientID = claims.AuthorizingParty
		}
	}

	// If no id_token_hint, try to identify the user from the session cookie.
	// This allows logout without a hint when the user has an active session.
	if userID == "" && connectorID == "" {
		if cookie, err := r.Cookie(s.sessionConfig.CookieName); err == nil && cookie.Value != "" {
			if uid, cid, nonce, err := parseSessionCookie(cookie.Value); err == nil {
				// Verify the session exists and nonce matches before trusting the cookie.
				if session, err := s.storage.GetAuthSession(ctx, uid, cid); err == nil && session.Nonce == nonce {
					userID = uid
					connectorID = cid
					s.logger.DebugContext(ctx, "logout: identified user from session cookie",
						"user_id", userID, "connector_id", connectorID)
				}
			}
		}
	}

	// Validate post_logout_redirect_uri against registered client URIs.
	if postLogoutRedirectURI != "" {
		if clientID == "" {
			s.renderError(r, w, http.StatusBadRequest, "post_logout_redirect_uri requires id_token_hint.")
			return
		}

		client, err := s.storage.GetClient(ctx, clientID)
		if err != nil {
			s.logger.ErrorContext(ctx, "logout: failed to get client", "client_id", clientID, "err", err)
			s.renderError(r, w, http.StatusBadRequest, "Invalid client.")
			return
		}

		if !slices.Contains(client.PostLogoutRedirectURIs, postLogoutRedirectURI) {
			s.logger.WarnContext(ctx, "logout: unregistered post_logout_redirect_uri",
				"uri", postLogoutRedirectURI, "client_id", clientID)
			s.renderError(r, w, http.StatusBadRequest, "Unregistered post_logout_redirect_uri.")
			return
		}
	}

	// Revoke refresh tokens (does not touch the auth session or user identity).
	var connectorData []byte
	if userID != "" && connectorID != "" {
		connectorData = s.revokeRefreshTokens(ctx, userID, connectorID)
	}

	// Try upstream logout. This requires a live auth session to store the HMAC key
	// and logout parameters. If the session doesn't exist (expired, no cookie, etc.)
	// upstream logout is skipped — RP-Initiated Logout treats upstream SLO as best-effort.
	if redirectURL, ok := s.tryUpstreamLogout(ctx, userID, connectorID, connectorData, postLogoutRedirectURI, state, clientID); ok {
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	// No upstream logout — delete session now, clear cookie, show page.
	s.logger.DebugContext(ctx, "logout: completing",
		"user_id", userID, "connector_id", connectorID, "client_id", clientID)
	loggedOut := s.deleteAuthSession(ctx, userID, connectorID)
	s.clearSessionCookie(w)
	s.finishLogout(w, r, postLogoutRedirectURI, state, loggedOut)
}

// handleLogoutCallback receives the redirect back from the upstream provider
// after it has completed its logout.
//
// Identity is resolved from the session cookie (HttpOnly, Secure, SameSite=Lax).
// The session must still exist in storage with a non-nil LogoutState (set before
// the upstream redirect). After validation, the session is deleted.
func (s *Server) handleLogoutCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Resolve identity from the session cookie.
	cookie, err := r.Cookie(s.sessionConfig.CookieName)
	if err != nil || cookie.Value == "" {
		s.renderError(r, w, http.StatusBadRequest, "Missing session cookie.")
		return
	}

	userID, connectorID, nonce, err := parseSessionCookie(cookie.Value)
	if err != nil {
		s.renderError(r, w, http.StatusBadRequest, "Invalid session cookie.")
		return
	}

	// Load the session and verify nonce.
	session, err := s.storage.GetAuthSession(ctx, userID, connectorID)
	if err != nil {
		s.logger.ErrorContext(ctx, "logout callback: session not found", "err", err)
		s.renderError(r, w, http.StatusBadRequest, "Session not found.")
		return
	}

	if session.Nonce != nonce {
		s.renderError(r, w, http.StatusBadRequest, "Invalid session.")
		return
	}

	if session.LogoutState == nil {
		s.renderError(r, w, http.StatusBadRequest, "No logout in progress.")
		return
	}

	ls := session.LogoutState

	// Let the connector validate the upstream logout response if it supports it.
	if ls.ConnectorID != "" {
		conn, err := s.getConnector(ctx, ls.ConnectorID)
		if err == nil {
			if logoutConn, ok := conn.Connector.(connector.LogoutCallbackConnector); ok {
				if err := logoutConn.HandleLogoutCallback(ctx, r); err != nil {
					s.logger.ErrorContext(ctx, "logout: upstream logout response validation failed",
						"connector_id", ls.ConnectorID, "err", err)
				}
			}
		}
	}

	// Session kept alive until now — delete it and clear the cookie.
	s.deleteAuthSession(ctx, userID, connectorID)
	s.clearSessionCookie(w)
	s.finishLogout(w, r, ls.PostLogoutRedirectURI, ls.State, true)
}

// finishLogout renders the logout page with a "Back to Application" link.
// loggedOut indicates whether an active session was actually terminated.
func (s *Server) finishLogout(w http.ResponseWriter, r *http.Request, postLogoutRedirectURI, state string, loggedOut bool) {
	var backURL string
	if postLogoutRedirectURI != "" {
		u, err := url.Parse(postLogoutRedirectURI)
		if err == nil {
			if state != "" {
				q := u.Query()
				q.Set("state", state)
				u.RawQuery = q.Encode()
			}
			backURL = u.String()
		}
	}

	if err := s.templates.logout(r, w, backURL, loggedOut); err != nil {
		s.logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

// tryUpstreamLogout attempts to redirect to the upstream provider's logout endpoint.
// It stores LogoutState in the auth session before redirecting so the callback can
// read it back. Returns the redirect URL and true on success, or ("", false) if
// upstream logout is not possible (no session, connector doesn't support it, etc.).
func (s *Server) tryUpstreamLogout(ctx context.Context, userID, connectorID string, connectorData []byte, postLogoutRedirectURI, state, clientID string) (string, bool) {
	if connectorID == "" {
		return "", false
	}

	conn, err := s.getConnector(ctx, connectorID)
	if err != nil {
		return "", false
	}

	logoutConn, ok := conn.Connector.(connector.LogoutCallbackConnector)
	if !ok {
		return "", false
	}

	// Check that the session exists — we need it to store logout state.
	_, err = s.storage.GetAuthSession(ctx, userID, connectorID)
	if err != nil {
		s.logger.DebugContext(ctx, "logout: no auth session for upstream logout, skipping",
			"user_id", userID, "connector_id", connectorID)
		return "", false
	}

	// Store logout parameters in the session.
	if err := s.storage.UpdateAuthSession(ctx, userID, connectorID, func(old storage.AuthSession) (storage.AuthSession, error) {
		old.LogoutState = &storage.LogoutState{
			PostLogoutRedirectURI: postLogoutRedirectURI,
			State:                 state,
			ClientID:              clientID,
			ConnectorID:           connectorID,
		}
		return old, nil
	}); err != nil {
		s.logger.ErrorContext(ctx, "logout: failed to save logout state", "err", err)
		return "", false
	}

	callbackURI := s.absURL("/logout/callback")
	upstreamURL, err := logoutConn.LogoutURL(ctx, connectorData, callbackURI)
	if err != nil {
		s.logger.ErrorContext(ctx, "logout: upstream connector error", "err", err)
		return "", false
	}
	if upstreamURL == "" {
		return "", false
	}

	u, err := url.Parse(upstreamURL)
	if err != nil {
		s.logger.ErrorContext(ctx, "logout: failed to parse upstream URL", "err", err)
		return "", false
	}

	return u.String(), true
}

// deleteAuthSession deletes the session and returns true if it existed.
func (s *Server) deleteAuthSession(ctx context.Context, userID, connectorID string) bool {
	if userID == "" || connectorID == "" {
		return false
	}
	if err := s.storage.DeleteAuthSession(ctx, userID, connectorID); err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			s.logger.ErrorContext(ctx, "logout: failed to delete auth session", "err", err)
		}
		return false
	}
	s.logger.InfoContext(ctx, "logout successful", "user_id", userID, "connector_id", connectorID)
	return true
}

// revokeRefreshTokens deletes all refresh tokens for the given user/connector
// and clears the references in the offline session (but keeps the session object).
// Returns the connector data from the offline session (for upstream logout).
func (s *Server) revokeRefreshTokens(ctx context.Context, userID, connectorID string) []byte {
	offlineSessions, err := s.storage.GetOfflineSessions(ctx, userID, connectorID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			s.logger.ErrorContext(ctx, "logout: failed to get offline sessions", "err", err)
		}
		return nil
	}

	for _, ref := range offlineSessions.Refresh {
		if err := s.storage.DeleteRefresh(ctx, ref.ID); err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				s.logger.ErrorContext(ctx, "logout: failed to delete refresh token", "token_id", ref.ID, "err", err)
			}
		}
	}

	if err := s.storage.UpdateOfflineSessions(ctx, userID, connectorID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
		old.Refresh = make(map[string]*storage.RefreshTokenRef)
		return old, nil
	}); err != nil {
		s.logger.ErrorContext(ctx, "logout: failed to update offline sessions", "err", err)
	}

	return offlineSessions.ConnectorData
}
