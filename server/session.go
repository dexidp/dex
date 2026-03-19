package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/dexidp/dex/storage"
)

// rememberMeDefault returns a pointer to the default remember-me value if sessions are enabled, nil otherwise.
func (s *Server) rememberMeDefault() *bool {
	if s.sessionConfig == nil {
		return nil
	}
	v := s.sessionConfig.RememberMeCheckedByDefault
	return &v
}

// remoteIP returns the real IP from context (set by parseRealIP middleware) or falls back to r.RemoteAddr.
func remoteIP(r *http.Request) string {
	if ip, ok := r.Context().Value(RequestKeyRemoteIP).(string); ok && ip != "" {
		return ip
	}
	return r.RemoteAddr
}

// sessionCookieValue encodes session identity into a cookie value.
// Format: base64url(userID) + "." + base64url(connectorID) + "." + nonce
// TODO(nabokihms): consider cookie encoding
func sessionCookieValue(userID, connectorID, nonce string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(userID)) +
		"." + base64.RawURLEncoding.EncodeToString([]byte(connectorID)) +
		"." + nonce
}

// parseSessionCookie decodes a session cookie value into its components.
func parseSessionCookie(value string) (userID, connectorID, nonce string, err error) {
	parts := strings.SplitN(value, ".", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid session cookie format")
	}

	userIDBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", "", fmt.Errorf("decode userID: %w", err)
	}

	connectorIDBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", "", fmt.Errorf("decode connectorID: %w", err)
	}

	return string(userIDBytes), string(connectorIDBytes), parts[2], nil
}

func (s *Server) sessionCookiePath() string {
	if s.issuerURL.Path == "" {
		return "/"
	}
	return s.issuerURL.Path
}

func (s *Server) setSessionCookie(w http.ResponseWriter, userID, connectorID, nonce string, rememberMe bool) {
	cookie := &http.Cookie{
		Name:     s.sessionConfig.CookieName,
		Value:    sessionCookieValue(userID, connectorID, nonce),
		Path:     s.sessionCookiePath(),
		HttpOnly: true,
		Secure:   s.issuerURL.Scheme == "https",
		SameSite: http.SameSiteLaxMode,
	}
	if rememberMe {
		cookie.MaxAge = int(s.sessionConfig.AbsoluteLifetime.Seconds())
	}
	http.SetCookie(w, cookie)
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.sessionConfig.CookieName,
		Value:    "",
		Path:     s.sessionCookiePath(),
		HttpOnly: true,
		Secure:   s.issuerURL.Scheme == "https",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// getValidAuthSession returns a valid, non-expired session or nil.
// It parses the session cookie to extract (userID, connectorID, nonce),
// looks up the session by composite key, and verifies the nonce.
// Invalid or expired session cookies are cleared automatically.
func (s *Server) getValidAuthSession(ctx context.Context, w http.ResponseWriter, r *http.Request, authReq *storage.AuthRequest) *storage.AuthSession {
	if s.sessionConfig == nil {
		return nil
	}

	cookie, err := r.Cookie(s.sessionConfig.CookieName)
	if err != nil || cookie.Value == "" {
		return nil
	}

	userID, connectorID, nonce, err := parseSessionCookie(cookie.Value)
	if err != nil {
		s.logger.DebugContext(ctx, "invalid session cookie format", "err", err)
		s.clearSessionCookie(w)
		return nil
	}

	session, err := s.storage.GetAuthSession(ctx, userID, connectorID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			s.logger.ErrorContext(ctx, "failed to get auth session", "err", err)
		}
		s.clearSessionCookie(w)
		return nil
	}

	// Verify nonce to prevent cookie forgery.
	if session.Nonce != nonce {
		s.logger.DebugContext(ctx, "auth session nonce mismatch")
		s.clearSessionCookie(w)
		return nil
	}

	now := s.now()

	// Check absolute lifetime.
	if now.After(session.CreatedAt.Add(s.sessionConfig.AbsoluteLifetime)) {
		s.logger.InfoContext(ctx, "auth session expired (absolute lifetime)",
			"user_id", session.UserID, "connector_id", session.ConnectorID)
		if err := s.storage.DeleteAuthSession(ctx, session.UserID, session.ConnectorID); err != nil {
			s.logger.DebugContext(ctx, "failed to delete expired auth session", "err", err)
		}
		s.clearSessionCookie(w)
		return nil
	}

	// Check idle timeout.
	if now.After(session.LastActivity.Add(s.sessionConfig.ValidIfNotUsedFor)) {
		s.logger.InfoContext(ctx, "auth session expired (idle timeout)",
			"user_id", session.UserID, "connector_id", session.ConnectorID)
		if err := s.storage.DeleteAuthSession(ctx, session.UserID, session.ConnectorID); err != nil {
			s.logger.DebugContext(ctx, "failed to delete expired auth session", "err", err)
		}
		s.clearSessionCookie(w)
		return nil
	}

	// Only reuse sessions from the same connector.
	if session.ConnectorID != authReq.ConnectorID {
		return nil
	}

	return &session
}

// createOrUpdateAuthSession creates a new session or updates an existing one
// after a successful login, and sets the session cookie.
// rememberMe controls whether the cookie is persistent (survives browser close).
func (s *Server) createOrUpdateAuthSession(ctx context.Context, r *http.Request, w http.ResponseWriter, authReq storage.AuthRequest, rememberMe bool) error {
	if s.sessionConfig == nil {
		return nil
	}

	now := s.now()
	userID := authReq.Claims.UserID
	connectorID := authReq.ConnectorID

	clientState := &storage.ClientAuthState{
		Active:       true,
		ExpiresAt:    now.Add(s.sessionConfig.AbsoluteLifetime),
		LastActivity: now,
	}

	// Try to reuse existing session for this (userID, connectorID).
	session, err := s.storage.GetAuthSession(ctx, userID, connectorID)
	if err == nil {
		// Session exists, update it.
		s.logger.DebugContext(ctx, "updating existing auth session",
			"user_id", userID, "connector_id", connectorID, "client_id", authReq.ClientID)

		if err := s.storage.UpdateAuthSession(ctx, userID, connectorID, func(old storage.AuthSession) (storage.AuthSession, error) {
			old.LastActivity = now
			if old.ClientStates == nil {
				old.ClientStates = make(map[string]*storage.ClientAuthState)
			}
			old.ClientStates[authReq.ClientID] = clientState
			return old, nil
		}); err != nil {
			return fmt.Errorf("update auth session: %w", err)
		}

		s.setSessionCookie(w, userID, connectorID, session.Nonce, rememberMe)
		return nil
	}

	// Unexpected error, exit the method.
	if !errors.Is(err, storage.ErrNotFound) {
		return fmt.Errorf("get auth session: %w", err)
	}

	nonce := storage.NewID()
	newSession := storage.AuthSession{
		UserID:      userID,
		ConnectorID: connectorID,
		Nonce:       nonce,
		ClientStates: map[string]*storage.ClientAuthState{
			authReq.ClientID: clientState,
		},
		CreatedAt:    now,
		LastActivity: now,
		IPAddress:    remoteIP(r),
		UserAgent:    r.UserAgent(),
	}

	if err := s.storage.CreateAuthSession(ctx, newSession); err != nil {
		return fmt.Errorf("create auth session: %w", err)
	}

	s.logger.DebugContext(ctx, "created new auth session",
		"user_id", userID, "connector_id", connectorID, "client_id", authReq.ClientID)
	s.setSessionCookie(w, userID, connectorID, nonce, rememberMe)
	return nil
}

// trySessionLogin checks if the user has a valid session for the same connector.
// If so, it finalizes login from the stored identity and returns a redirect URL.
// Returns ("", false) if session-based login is not possible.
func (s *Server) trySessionLogin(ctx context.Context, r *http.Request, w http.ResponseWriter, authReq *storage.AuthRequest) (string, bool) {
	session := s.getValidAuthSession(ctx, w, r, authReq)
	if session == nil {
		return "", false
	}

	clientState, ok := session.ClientStates[authReq.ClientID]
	if !ok || !clientState.Active {
		return "", false
	}

	now := s.now()
	if now.After(clientState.ExpiresAt) {
		return "", false
	}

	// Load identity from storage.
	ui, err := s.storage.GetUserIdentity(ctx, session.UserID, session.ConnectorID)
	if err != nil {
		s.logger.ErrorContext(ctx, "session: failed to get user identity", "err", err)
		return "", false
	}

	// Check max_age: if the user's last authentication is too old, force re-auth.
	if authReq.MaxAge >= 0 {
		if now.Sub(ui.LastLogin) > time.Duration(authReq.MaxAge)*time.Second {
			return "", false
		}
	}

	claims := storage.Claims{
		UserID:            ui.Claims.UserID,
		Username:          ui.Claims.Username,
		PreferredUsername: ui.Claims.PreferredUsername,
		Email:             ui.Claims.Email,
		EmailVerified:     ui.Claims.EmailVerified,
		Groups:            ui.Claims.Groups,
	}

	// Update AuthRequest with stored identity and auth_time from last login.
	if err := s.storage.UpdateAuthRequest(ctx, authReq.ID, func(a storage.AuthRequest) (storage.AuthRequest, error) {
		a.LoggedIn = true
		a.Claims = claims
		a.ConnectorID = session.ConnectorID
		a.AuthTime = ui.LastLogin
		return a, nil
	}); err != nil {
		s.logger.ErrorContext(ctx, "session: failed to update auth request", "err", err)
		return "", false
	}

	s.logger.DebugContext(ctx, "session: re-authenticated from session",
		"user_id", session.UserID, "connector_id", session.ConnectorID)

	// Update session activity.
	_ = s.storage.UpdateAuthSession(ctx, session.UserID, session.ConnectorID, func(old storage.AuthSession) (storage.AuthSession, error) {
		old.LastActivity = now
		if cs, ok := old.ClientStates[authReq.ClientID]; ok {
			cs.LastActivity = now
		}
		return old, nil
	})

	// Build HMAC for approval URL.
	h := hmac.New(sha256.New, authReq.HMACKey)
	h.Write([]byte(authReq.ID))
	mac := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	// Skip approval if globally configured or user already consented to the requested scopes.
	if !authReq.ForceApprovalPrompt && (s.skipApproval || scopesCoveredByConsent(ui.Consents[authReq.ClientID], authReq.Scopes)) {
		// Re-read to get the updated AuthRequest (LoggedIn, Claims, ConnectorID set above).
		updated, err := s.storage.GetAuthRequest(ctx, authReq.ID)
		if err != nil {
			s.logger.ErrorContext(ctx, "session: failed to get auth request", "err", err)
			return "", false
		}
		s.sendCodeResponse(w, r, updated)
		return "", true
	}

	returnURL := path.Join(s.issuerURL.Path, "/approval") + "?req=" + authReq.ID + "&hmac=" + mac
	return returnURL, true
}

// updateSessionTokenIssuedAt updates the session's LastTokenIssuedAt for the given client.
func (s *Server) updateSessionTokenIssuedAt(r *http.Request, clientID string) {
	if s.sessionConfig == nil {
		return
	}

	cookie, err := r.Cookie(s.sessionConfig.CookieName)
	if err != nil || cookie.Value == "" {
		return
	}

	userID, connectorID, _, err := parseSessionCookie(cookie.Value)
	if err != nil {
		return
	}

	now := s.now()
	_ = s.storage.UpdateAuthSession(r.Context(), userID, connectorID, func(old storage.AuthSession) (storage.AuthSession, error) {
		old.LastActivity = now
		if cs, ok := old.ClientStates[clientID]; ok {
			cs.LastTokenIssuedAt = now
			cs.LastActivity = now
		}
		return old, nil
	})
}
