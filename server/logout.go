package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

// logoutState holds validated logout request parameters passed between handlers.
type logoutState struct {
	PostLogoutRedirectURI string `json:"ru,omitempty"`
	State                 string `json:"st,omitempty"`
	ClientID              string `json:"ci,omitempty"`
	ConnectorID           string `json:"co,omitempty"`
}

// handleLogout implements OIDC RP-Initiated Logout (https://openid.net/specs/openid-connect-rpinitiated-1_0.html).
//
// GET /logout?id_token_hint=...&post_logout_redirect_uri=...&state=...
//
// Flow:
//  1. Validate id_token_hint (signature + issuer; expiry skipped per spec)
//  2. Extract user identity (subject) and client (audience/azp) from the token
//  3. Validate post_logout_redirect_uri against the client's registered URIs
//  4. Delete auth session and clear cookie (Dex state cleaned first)
//  5. Revoke refresh tokens for the user/connector pair
//  6. If upstream connector implements LogoutConnector, redirect to upstream logout
//  7. Otherwise, render the logged-out page
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

	// Clean up Dex state FIRST (before any upstream redirect).
	var connectorData []byte
	var loggedOut bool

	if userID != "" && connectorID != "" {
		// Delete auth session.
		if err := s.storage.DeleteAuthSession(ctx, userID, connectorID); err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				s.logger.ErrorContext(ctx, "logout: failed to delete auth session", "err", err)
			}
		} else {
			loggedOut = true
		}

		// Revoke all refresh tokens for this user/connector pair.
		connectorData = s.revokeRefreshTokens(ctx, userID, connectorID)
	}

	// Clear the session cookie.
	s.clearSessionCookie(w)

	// Try upstream logout if the connector supports it.
	if redirectURL, ok := s.upstreamLogoutURL(ctx, connectorID, connectorData, postLogoutRedirectURI, state, clientID); ok {
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	s.finishLogout(w, r, postLogoutRedirectURI, state, loggedOut)
}

// handleLogoutCallback receives the redirect back from the upstream provider
// after it has completed its logout. It validates the upstream response (if the
// connector implements LogoutCallbackConnector) and renders the logged-out page.
func (s *Server) handleLogoutCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stateParam := r.FormValue("state")
	if stateParam == "" {
		s.renderError(r, w, http.StatusBadRequest, "Missing state parameter.")
		return
	}

	ls, err := decodeLogoutState(s.logoutHMACKey, stateParam)
	if err != nil {
		s.logger.ErrorContext(ctx, "logout callback: invalid state", "err", err)
		s.renderError(r, w, http.StatusBadRequest, "Invalid state parameter.")
		return
	}

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

	s.finishLogout(w, r, ls.PostLogoutRedirectURI, ls.State, true)
}

// finishLogout renders the logout page with a "Back to Application" link.
// loggedOut indicates whether an active session was actually terminated.
func (s *Server) finishLogout(w http.ResponseWriter, r *http.Request, postLogoutRedirectURI, state string, loggedOut bool) {
	// Build the redirect URL with state parameter if provided.
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

// upstreamLogoutURL builds an upstream provider logout URL if the connector supports it.
// Returns the redirect URL and true, or ("", false) if upstream logout is not available.
func (s *Server) upstreamLogoutURL(ctx context.Context, connectorID string, connectorData []byte, postLogoutRedirectURI, state, clientID string) (string, bool) {
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

	ls := logoutState{
		PostLogoutRedirectURI: postLogoutRedirectURI,
		State:                 state,
		ClientID:              clientID,
		ConnectorID:           connectorID,
	}
	stateParam, err := encodeLogoutState(s.logoutHMACKey, ls)
	if err != nil {
		s.logger.ErrorContext(ctx, "logout: failed to encode state", "err", err)
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

	q := u.Query()
	q.Set("state", stateParam)
	u.RawQuery = q.Encode()
	return u.String(), true
}

// revokeRefreshTokens deletes all refresh tokens for the given user/connector
// and clears the references in the offline session (but keeps the session object).
// Returns the connector data from the offline session (for upstream logout).
func (s *Server) revokeRefreshTokens(ctx context.Context, userID, connectorID string) []byte {
	// Get offline sessions to find refresh token references.
	offlineSessions, err := s.storage.GetOfflineSessions(ctx, userID, connectorID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			s.logger.ErrorContext(ctx, "logout: failed to get offline sessions", "err", err)
		}
		return nil
	}

	// Delete each refresh token.
	for _, ref := range offlineSessions.Refresh {
		if err := s.storage.DeleteRefresh(ctx, ref.ID); err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				s.logger.ErrorContext(ctx, "logout: failed to delete refresh token", "token_id", ref.ID, "err", err)
			}
		}
	}

	// Clear refresh token references from the offline session, but keep the object.
	if err := s.storage.UpdateOfflineSessions(ctx, userID, connectorID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
		old.Refresh = make(map[string]*storage.RefreshTokenRef)
		return old, nil
	}); err != nil {
		s.logger.ErrorContext(ctx, "logout: failed to update offline sessions", "err", err)
	}

	return offlineSessions.ConnectorData
}

// encodeLogoutState encodes logout state as HMAC-signed, URL-safe string.
// Format: base64url(json) + "." + base64url(hmac_sha256(json))
func encodeLogoutState(key []byte, ls logoutState) (string, error) {
	payload, err := json.Marshal(ls)
	if err != nil {
		return "", fmt.Errorf("marshal logout state: %w", err)
	}

	payloadEncoded := base64.RawURLEncoding.EncodeToString(payload)

	mac := hmac.New(sha256.New, key)
	mac.Write(payload)
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return payloadEncoded + "." + sig, nil
}

// decodeLogoutState decodes and verifies the HMAC-signed state parameter.
func decodeLogoutState(key []byte, encoded string) (logoutState, error) {
	dotIdx := strings.LastIndex(encoded, ".")
	if dotIdx < 0 {
		return logoutState{}, fmt.Errorf("invalid logout state format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(encoded[:dotIdx])
	if err != nil {
		return logoutState{}, fmt.Errorf("decode logout state payload: %w", err)
	}

	sig, err := base64.RawURLEncoding.DecodeString(encoded[dotIdx+1:])
	if err != nil {
		return logoutState{}, fmt.Errorf("decode logout state signature: %w", err)
	}

	mac := hmac.New(sha256.New, key)
	mac.Write(payload)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return logoutState{}, fmt.Errorf("logout state HMAC verification failed")
	}

	var ls logoutState
	if err := json.Unmarshal(payload, &ls); err != nil {
		return logoutState{}, fmt.Errorf("unmarshal logout state: %w", err)
	}
	return ls, nil
}
