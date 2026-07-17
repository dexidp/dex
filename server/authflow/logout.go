package authflow

import (
	"context"
	"crypto/subtle"
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
// GET/POST /logout?id_token_hint=...&post_logout_redirect_uri=...&state=...
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
func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		h.RenderError(r, w, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}

	idTokenHint := r.FormValue("id_token_hint")
	postLogoutRedirectURI := r.FormValue("post_logout_redirect_uri")
	state := r.FormValue("state")

	// When no id_token_hint is provided and this is a GET request,
	// show a confirmation page instead of logging out immediately.
	// This follows the OIDC spec recommendation and prevents CSRF via
	// cross-site image/link requests (e.g. <img src="/logout">).
	if idTokenHint == "" && r.Method == http.MethodGet {
		if err := h.templates.Logout(r, w, "", false, true); err != nil {
			h.logger.ErrorContext(ctx, "server template error", "err", err)
		}
		return
	}

	var userID, connectorID, clientID string

	if idTokenHint != "" {
		idToken, err := h.req.ValidateIDTokenHint(ctx, idTokenHint)
		if err != nil {
			h.logger.ErrorContext(ctx, "logout: invalid id_token_hint", "err", err)
			h.RenderError(r, w, http.StatusBadRequest, "Invalid id_token_hint.")
			return
		}

		sub := new(internal.IDTokenSubject)
		if err := internal.Unmarshal(idToken.Subject, sub); err != nil {
			h.logger.ErrorContext(ctx, "logout: failed to unmarshal subject", "err", err)
			h.RenderError(r, w, http.StatusBadRequest, "Invalid id_token_hint subject.")
			return
		}

		userID = sub.UserId
		connectorID = sub.ConnId

		h.logger.DebugContext(ctx, "logout: parsed id_token_hint",
			"user_id", userID, "connector_id", connectorID)

		// When cross-client (trusted peers) scopes are used, the token may have
		// multiple audiences. In that case the requesting client is in the "azp"
		// claim, not necessarily Audience[0]. Use the same logic as token introspection.
		var claims struct {
			AuthorizingParty string `json:"azp"`
		}
		if err := idToken.Claims(&claims); err != nil {
			h.logger.ErrorContext(ctx, "logout: failed to decode id_token_hint claims", "err", err)
			h.RenderError(r, w, http.StatusBadRequest, "Invalid id_token_hint.")
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
		if uid, cid, nonce, ok := h.sessions.ParseCookie(r); ok {
			// Verify the session exists and nonce matches before trusting the cookie.
			if session, err := h.storage.GetAuthSession(ctx, uid, cid); err == nil && subtle.ConstantTimeCompare([]byte(session.Nonce), []byte(nonce)) == 1 {
				userID = uid
				connectorID = cid
				h.logger.DebugContext(ctx, "logout: identified user from session cookie",
					"user_id", userID, "connector_id", connectorID)
			}
		}
	}

	// Validate post_logout_redirect_uri against registered client URIs.
	if postLogoutRedirectURI != "" {
		if clientID == "" {
			h.RenderError(r, w, http.StatusBadRequest, "post_logout_redirect_uri requires id_token_hint.")
			return
		}

		client, err := h.storage.GetClient(ctx, clientID)
		if err != nil {
			h.logger.ErrorContext(ctx, "logout: failed to get client", "client_id", clientID, "err", err)
			h.RenderError(r, w, http.StatusBadRequest, "Invalid client.")
			return
		}

		if !slices.Contains(client.PostLogoutRedirectURIs, postLogoutRedirectURI) {
			h.logger.WarnContext(ctx, "logout: unregistered post_logout_redirect_uri",
				"uri", postLogoutRedirectURI, "client_id", clientID)
			h.RenderError(r, w, http.StatusBadRequest, "Unregistered post_logout_redirect_uri.")
			return
		}
	}

	// Revoke refresh tokens (does not touch the auth session or user identity).
	if userID != "" && connectorID != "" {
		h.issuer.Refresh.Revoke(ctx, userID, connectorID)
	}

	// Try upstream logout. This requires a live auth session to store the HMAC key
	// and logout parameters. If the session doesn't exist (expired, no cookie, etc.)
	// upstream logout is skipped — RP-Initiated Logout treats upstream SLO as best-effort.
	if redirectURL, ok := h.tryUpstreamLogout(ctx, userID, connectorID, postLogoutRedirectURI, state, clientID); ok {
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	// No upstream logout — delete session now, clear cookie, show page.
	h.logger.DebugContext(ctx, "logout: completing",
		"user_id", userID, "connector_id", connectorID, "client_id", clientID)
	loggedOut := h.deleteAuthSession(ctx, userID, connectorID)
	h.sessions.ClearCookie(w)
	h.finishLogout(w, r, postLogoutRedirectURI, state, loggedOut)
}

// handleLogoutCallback receives the redirect back from the upstream provider
// after it has completed its logout.
//
// Identity is resolved from the session cookie (HttpOnly, Secure, SameSite=Lax).
// The session must still exist in storage with a non-nil LogoutState (set before
// the upstream redirect). After validation, the session is deleted.
func (h *Handler) handleLogoutCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Resolve identity from the session cookie.
	userID, connectorID, nonce, ok := h.sessions.ParseCookie(r)
	if !ok {
		h.RenderError(r, w, http.StatusBadRequest, "Invalid session cookie.")
		return
	}

	// Load the session and verify nonce.
	session, err := h.storage.GetAuthSession(ctx, userID, connectorID)
	if err != nil {
		h.logger.ErrorContext(ctx, "logout callback: session not found", "err", err)
		h.RenderError(r, w, http.StatusBadRequest, "Session not found.")
		return
	}

	if subtle.ConstantTimeCompare([]byte(session.Nonce), []byte(nonce)) != 1 {
		h.RenderError(r, w, http.StatusBadRequest, "Invalid session.")
		return
	}

	if session.LogoutState == nil {
		h.RenderError(r, w, http.StatusBadRequest, "No logout in progress.")
		return
	}

	ls := session.LogoutState

	// Let the connector validate the upstream logout response if it supports it.
	if ls.ConnectorID != "" {
		conn, err := h.connectors.Get(ctx, ls.ConnectorID)
		if err == nil {
			if logoutConn, ok := conn.Connector.(connector.LogoutCallbackConnector); ok {
				if err := logoutConn.HandleLogoutCallback(ctx, r); err != nil {
					h.logger.ErrorContext(ctx, "logout: upstream logout response validation failed",
						"connector_id", ls.ConnectorID, "err", err)
				}
			}
		}
	}

	// Session kept alive until now — delete it and clear the cookie.
	h.deleteAuthSession(ctx, userID, connectorID)
	h.sessions.ClearCookie(w)
	h.finishLogout(w, r, ls.PostLogoutRedirectURI, ls.State, true)
}

// finishLogout renders the logout page with a "Back to Application" link.
// loggedOut indicates whether an active session was actually terminated.
func (h *Handler) finishLogout(w http.ResponseWriter, r *http.Request, postLogoutRedirectURI, state string, loggedOut bool) {
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

	if err := h.templates.Logout(r, w, backURL, loggedOut, false); err != nil {
		h.logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

// tryUpstreamLogout attempts to redirect to the upstream provider's logout endpoint.
// It stores LogoutState in the auth session before redirecting so the callback can
// read it back. Returns the redirect URL and true on success, or ("", false) if
// upstream logout is not possible (no session, connector doesn't support it, etc.).
func (h *Handler) tryUpstreamLogout(ctx context.Context, userID, connectorID, postLogoutRedirectURI, state, clientID string) (string, bool) {
	if connectorID == "" {
		return "", false
	}

	conn, err := h.connectors.Get(ctx, connectorID)
	if err != nil {
		return "", false
	}

	logoutConn, ok := conn.Connector.(connector.LogoutCallbackConnector)
	if !ok {
		return "", false
	}

	// Check that the session exists — we need it to store logout state.
	_, err = h.storage.GetAuthSession(ctx, userID, connectorID)
	if err != nil {
		h.logger.DebugContext(ctx, "logout: no auth session for upstream logout, skipping",
			"user_id", userID, "connector_id", connectorID)
		return "", false
	}

	// Store logout parameters in the session.
	if err := h.storage.UpdateAuthSession(ctx, userID, connectorID, func(old storage.AuthSession) (storage.AuthSession, error) {
		old.LogoutState = &storage.LogoutState{
			PostLogoutRedirectURI: postLogoutRedirectURI,
			State:                 state,
			ClientID:              clientID,
			ConnectorID:           connectorID,
		}
		return old, nil
	}); err != nil {
		h.logger.ErrorContext(ctx, "logout: failed to save logout state", "err", err)
		return "", false
	}

	callbackURI := h.AbsURL("/logout/callback")
	upstreamURL, err := logoutConn.LogoutURL(ctx, callbackURI)
	if err != nil {
		h.logger.ErrorContext(ctx, "logout: upstream connector error", "err", err)
		return "", false
	}
	if upstreamURL == "" {
		return "", false
	}

	u, err := url.Parse(upstreamURL)
	if err != nil {
		h.logger.ErrorContext(ctx, "logout: failed to parse upstream URL", "err", err)
		return "", false
	}

	return u.String(), true
}

// deleteAuthSession deletes the session and returns true if it existed.
func (h *Handler) deleteAuthSession(ctx context.Context, userID, connectorID string) bool {
	if userID == "" || connectorID == "" {
		return false
	}
	if err := h.storage.DeleteAuthSession(ctx, userID, connectorID); err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			h.logger.ErrorContext(ctx, "logout: failed to delete auth session", "err", err)
		}
		return false
	}
	h.logger.InfoContext(ctx, "logout successful", "user_id", userID, "connector_id", connectorID)
	return true
}
