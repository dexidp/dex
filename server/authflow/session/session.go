// Package session owns dex's browser session: the session cookie, SSO session
// lookup, and auth-session CRUD. The interactive auth flow delegates to a
// Manager; it never touches these internals directly.
package session

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/log"
	"github.com/dexidp/dex/storage"
)

// Config holds resolved session configuration.
type Config struct {
	CookieName                 string
	CookieEncryptionKey        []byte
	AbsoluteLifetime           time.Duration
	ValidIfNotUsedFor          time.Duration
	RememberMeCheckedByDefault bool
	// SSOSharedWithDefault is the default SSO sharing policy for clients without explicit SSOSharedWith.
	// "all" = share with all clients, "none" or "" = share with no one (default).
	SSOSharedWithDefault string
}

// Manager owns the session lifecycle. Construct it with New.
type Manager struct {
	storage   storage.Storage
	config    *Config
	now       func() time.Time
	logger    *slog.Logger
	issuerURL url.URL
}

// New builds a session Manager.
func New(storage storage.Storage, config *Config, now func() time.Time, logger *slog.Logger, issuerURL url.URL) *Manager {
	return &Manager{storage: storage, config: config, now: now, logger: logger, issuerURL: issuerURL}
}

// Enabled reports whether sessions are configured.
func (m *Manager) Enabled() bool { return m.config != nil }

func (m *Manager) RememberMeDefault() *bool {
	if m.config == nil {
		return nil
	}
	v := m.config.RememberMeCheckedByDefault
	return &v
}

// remoteIP returns the real IP from context (set by parseRealIP middleware) or falls back to r.RemoteAddr.
func remoteIP(r *http.Request) string {
	if ip, ok := r.Context().Value(log.RequestKeyRemoteIP).(string); ok && ip != "" {
		return ip
	}
	return r.RemoteAddr
}

func (m *Manager) cookiePath() string {
	if m.issuerURL.Path == "" {
		return "/"
	}
	return m.issuerURL.Path
}

func (m *Manager) SetCookie(w http.ResponseWriter, userID, connectorID, nonce string, rememberMe bool) {
	cookie := &http.Cookie{
		Name:     m.config.CookieName,
		Value:    internal.SessionCookieValue(userID, connectorID, nonce, m.config.CookieEncryptionKey),
		Path:     m.cookiePath(),
		HttpOnly: true,
		Secure:   m.issuerURL.Scheme == "https",
		SameSite: http.SameSiteLaxMode,
	}
	if rememberMe {
		cookie.MaxAge = int(m.config.AbsoluteLifetime.Seconds())
	}
	http.SetCookie(w, cookie)
}

func (m *Manager) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.config.CookieName,
		Value:    "",
		Path:     m.cookiePath(),
		HttpOnly: true,
		Secure:   m.issuerURL.Scheme == "https",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// getValidSession returns a valid, non-expired session or nil.
// It parses the session cookie to extract (userID, connectorID, nonce),
// looks up the session by composite key, and verifies the nonce.
// Invalid or expired session cookies are cleared automatically.
func (m *Manager) ValidSession(ctx context.Context, w http.ResponseWriter, r *http.Request) *storage.AuthSession {
	if m.config == nil {
		return nil
	}

	cookie, err := r.Cookie(m.config.CookieName)
	if err != nil || cookie.Value == "" {
		return nil
	}

	userID, connectorID, nonce, err := internal.ParseSessionCookie(cookie.Value, m.config.CookieEncryptionKey)
	if err != nil {
		m.logger.DebugContext(ctx, "invalid session cookie format", "err", err)
		m.ClearCookie(w)
		return nil
	}

	session, err := m.storage.GetAuthSession(ctx, userID, connectorID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			m.logger.ErrorContext(ctx, "failed to get auth session", "err", err)
		}
		m.ClearCookie(w)
		return nil
	}

	// Verify nonce to prevent cookie forgery.
	// Use constant-time comparison to prevent timing attacks that could
	// allow an attacker to recover the nonce byte-by-byte.
	if subtle.ConstantTimeCompare([]byte(session.Nonce), []byte(nonce)) != 1 {
		m.logger.DebugContext(ctx, "auth session nonce mismatch")
		m.ClearCookie(w)
		return nil
	}

	now := m.now()

	// Check absolute lifetime using the stored expiry (set once at creation).
	if !session.AbsoluteExpiry.IsZero() && now.After(session.AbsoluteExpiry) {
		m.logger.InfoContext(ctx, "auth session expired (absolute lifetime)",
			"user_id", session.UserID, "connector_id", session.ConnectorID)
		if err := m.storage.DeleteAuthSession(ctx, session.UserID, session.ConnectorID); err != nil {
			m.logger.DebugContext(ctx, "failed to delete expired auth session", "err", err)
		}
		m.ClearCookie(w)
		return nil
	}

	// Check idle timeout using the stored expiry (updated on every activity).
	if !session.IdleExpiry.IsZero() && now.After(session.IdleExpiry) {
		m.logger.InfoContext(ctx, "auth session expired (idle timeout)",
			"user_id", session.UserID, "connector_id", session.ConnectorID)
		if err := m.storage.DeleteAuthSession(ctx, session.UserID, session.ConnectorID); err != nil {
			m.logger.DebugContext(ctx, "failed to delete expired auth session", "err", err)
		}
		m.ClearCookie(w)
		return nil
	}

	return &session
}

// getValidAuthSession returns a valid session matching the auth request's connector, or nil.
func (m *Manager) ValidAuthSession(ctx context.Context, w http.ResponseWriter, r *http.Request, authReq *storage.AuthRequest) *storage.AuthSession {
	session := m.ValidSession(ctx, w, r)
	if session == nil {
		return nil
	}

	// Only reuse sessions from the same connector.
	if session.ConnectorID != authReq.ConnectorID {
		return nil
	}

	return session
}

// createOrUpdateAuthSession creates a new session or updates an existing one
// after a successful login, and sets the session cookie.
// rememberMe controls whether the cookie is persistent (survives browser close).
func (m *Manager) CreateOrUpdateAuthSession(ctx context.Context, r *http.Request, w http.ResponseWriter, authReq storage.AuthRequest, rememberMe bool) error {
	if m.config == nil {
		return nil
	}

	now := m.now()
	userID := authReq.Claims.UserID
	connectorID := authReq.ConnectorID

	clientState := &storage.ClientAuthState{
		Active:       true,
		ExpiresAt:    now.Add(m.config.AbsoluteLifetime),
		LastActivity: now,
	}

	// Try to reuse existing session for this (userID, connectorID).
	session, err := m.storage.GetAuthSession(ctx, userID, connectorID)
	if err == nil {
		// Session exists, update it.
		m.logger.DebugContext(ctx, "updating existing auth session",
			"user_id", userID, "connector_id", connectorID, "client_id", authReq.ClientID)

		if err := m.storage.UpdateAuthSession(ctx, userID, connectorID, func(old storage.AuthSession) (storage.AuthSession, error) {
			old.LastActivity = now
			old.IdleExpiry = now.Add(m.config.ValidIfNotUsedFor)
			if old.ClientStates == nil {
				old.ClientStates = make(map[string]*storage.ClientAuthState)
			}
			old.ClientStates[authReq.ClientID] = clientState
			return old, nil
		}); err != nil {
			return fmt.Errorf("update auth session: %w", err)
		}

		m.SetCookie(w, userID, connectorID, session.Nonce, rememberMe)
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
		CreatedAt:      now,
		LastActivity:   now,
		IPAddress:      remoteIP(r),
		UserAgent:      r.UserAgent(),
		AbsoluteExpiry: now.Add(m.config.AbsoluteLifetime),
		IdleExpiry:     now.Add(m.config.ValidIfNotUsedFor),
	}

	if err := m.storage.CreateAuthSession(ctx, newSession); err != nil {
		return fmt.Errorf("create auth session: %w", err)
	}

	m.logger.DebugContext(ctx, "created new auth session",
		"user_id", userID, "connector_id", connectorID, "client_id", authReq.ClientID)
	m.SetCookie(w, userID, connectorID, nonce, rememberMe)
	return nil
}

// trySessionLogin checks if the user has a valid session for the same connector.
// If so, it finalizes login from the stored identity and returns a redirect URL.
// Returns ("", false) if session-based login is not possible.
func (m *Manager) ClientSharesWith(sourceClient storage.Client, targetClientID string) bool {
	ssoSharedWith := sourceClient.SSOSharedWith

	// If client has no explicit ssoSharedWith, use default from session config.
	if ssoSharedWith == nil {
		switch m.config.SSOSharedWithDefault {
		case "all":
			return true
		default: // "none" or ""
			return false
		}
	}

	// Explicit empty slice means share with no one.
	if len(ssoSharedWith) == 0 {
		return false
	}

	for _, peer := range ssoSharedWith {
		if peer == "*" || peer == targetClientID {
			return true
		}
	}
	return false
}

// findSSOSession checks whether any active client in the session shares its
// authentication with targetClientID via the ssoSharedWith policy.
//
// Note: the caller already has the target client loaded (for AllowedConnectors
// validation), but here we need the *source* client configs - those are the
// clients the user previously authenticated for, and their ssoSharedWith
// policies determine whether SSO is allowed. These are different clients,
// so the GetClient calls below are not redundant.
func (m *Manager) FindSSO(ctx context.Context, session *storage.AuthSession, targetClientID string) *storage.ClientAuthState {
	now := m.now()

	for sourceClientID, state := range session.ClientStates {
		if !state.Active || now.After(state.ExpiresAt) {
			continue
		}

		// Only directly-authenticated states may act as SSO sources. Skipping
		// SSO-derived states keeps sharing unidirectional and prevents transitive
		// A->B->C chains (a user authenticated only to A must not be SSO'd into C
		// just because B shares with C).
		if state.ViaSSO {
			continue
		}

		sourceClient, err := m.storage.GetClient(ctx, sourceClientID)
		if err != nil {
			m.logger.DebugContext(ctx, "session: SSO lookup failed to get source client",
				"source_client_id", sourceClientID, "err", err)
			continue
		}

		if m.ClientSharesWith(sourceClient, targetClientID) {
			return state
		}
	}

	return nil
}

// trySessionLoginWithSession is like trySessionLogin but accepts a pre-retrieved session.
// This allows callers to inspect the session (e.g., for id_token_hint comparison) before
// attempting session-based login.
func (m *Manager) UpdateTokenIssuedAt(r *http.Request, clientID string) {
	if m.config == nil {
		return
	}

	cookie, err := r.Cookie(m.config.CookieName)
	if err != nil || cookie.Value == "" {
		return
	}

	userID, connectorID, _, err := internal.ParseSessionCookie(cookie.Value, m.config.CookieEncryptionKey)
	if err != nil {
		return
	}

	now := m.now()
	_ = m.storage.UpdateAuthSession(r.Context(), userID, connectorID, func(old storage.AuthSession) (storage.AuthSession, error) {
		old.LastActivity = now
		old.IdleExpiry = now.Add(m.config.ValidIfNotUsedFor)
		if cs, ok := old.ClientStates[clientID]; ok {
			cs.LastTokenIssuedAt = now
			cs.LastActivity = now
		}
		return old, nil
	})
}

// --- Handler orchestration (uses sessionManager primitives) ---

// ParseCookie reads and decodes the session cookie from the request. ok is false
// when sessions are disabled, the cookie is absent, or it fails to decode.
func (m *Manager) ParseCookie(r *http.Request) (userID, connectorID, nonce string, ok bool) {
	if m.config == nil {
		return "", "", "", false
	}
	cookie, err := r.Cookie(m.config.CookieName)
	if err != nil {
		return "", "", "", false
	}
	uid, cid, n, err := internal.ParseSessionCookie(cookie.Value, m.config.CookieEncryptionKey)
	if err != nil {
		return "", "", "", false
	}
	return uid, cid, n, true
}

// AbsoluteExpiry returns the absolute session expiry measured from now.
func (m *Manager) AbsoluteExpiry(now time.Time) time.Time {
	return now.Add(m.config.AbsoluteLifetime)
}

// IdleExpiry returns the idle-timeout expiry measured from now.
func (m *Manager) IdleExpiry(now time.Time) time.Time {
	return now.Add(m.config.ValidIfNotUsedFor)
}

// DefaultRememberMe reports the configured default for the remember-me choice.
func (m *Manager) DefaultRememberMe() bool {
	return m.config != nil && m.config.RememberMeCheckedByDefault
}
