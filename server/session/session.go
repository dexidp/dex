package session

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/reqctx"
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
	Storage   storage.Storage
	Config    *Config
	Now       func() time.Time
	Logger    *slog.Logger
	IssuerURL oauth2.IssuerURL
}

// Enabled reports whether sessions are configured.
func (m *Manager) Enabled() bool { return m.Config != nil }

// RememberMeDefault reports the configured default for the remember-me choice,
// or nil when sessions are disabled. The password form renders it as the
// checkbox's initial state; callers that need a plain bool (connector callbacks,
// which render no checkbox) treat nil as false.
func (m *Manager) RememberMeDefault() *bool {
	if m.Config == nil {
		return nil
	}
	v := m.Config.RememberMeCheckedByDefault
	return &v
}

// remoteIP returns the real IP from context (set by the real-IP resolver) or,
// failing that, the host part of r.RemoteAddr. Either way the result carries no
// port; only an address that parses as neither is returned verbatim.
func remoteIP(r *http.Request) string {
	if ip, ok := r.Context().Value(reqctx.RequestKeyRemoteIP).(string); ok && ip != "" {
		return ip
	}
	// RemoteAddr carries the ephemeral source port, which is noise in an audit
	// record: it identifies the connection, not the client. The resolver behind
	// the context value already strips it, so strip it here too.
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func (m *Manager) cookiePath() string {
	if m.IssuerURL.Path == "" {
		return "/"
	}
	return m.IssuerURL.Path
}

func (m *Manager) SetCookie(w http.ResponseWriter, sessionID, secret string, rememberMe bool) {
	cookie := &http.Cookie{
		Name:     m.Config.CookieName,
		Value:    internal.SessionCookieValue(sessionID, secret, m.Config.CookieEncryptionKey),
		Path:     m.cookiePath(),
		HttpOnly: true,
		Secure:   m.IssuerURL.Scheme == "https",
		SameSite: http.SameSiteLaxMode,
	}
	if rememberMe {
		cookie.MaxAge = int(m.Config.AbsoluteLifetime.Seconds())
	}
	http.SetCookie(w, cookie)
}

func (m *Manager) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.Config.CookieName,
		Value:    "",
		Path:     m.cookiePath(),
		HttpOnly: true,
		Secure:   m.IssuerURL.Scheme == "https",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// expiryReason names why a session is over, or "" while it is still live.
func expiryReason(session storage.AuthSession, now time.Time) string {
	if !session.AbsoluteExpiry.IsZero() && now.After(session.AbsoluteExpiry) {
		return "absolute lifetime"
	}
	if !session.IdleExpiry.IsZero() && now.After(session.IdleExpiry) {
		return "idle timeout"
	}
	return ""
}

// Alive reports whether the session a token names is still there. Tokens carry the
// session's ID as their "sid" claim, so this is a direct read — nothing has to be
// resolved from the user's identity and compared.
//
// Expiry counts as gone. Garbage collection runs on its own schedule, so between a
// session ending and its row disappearing there is a window in which reporting it
// alive would keep the tokens of an ended session working.
func (m *Manager) Alive(ctx context.Context, sessionID string) bool {
	if m == nil || m.Config == nil || sessionID == "" {
		return false
	}

	session, err := m.Storage.GetAuthSession(ctx, sessionID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			m.Logger.ErrorContext(ctx, "failed to get auth session", "err", err)
		}
		return false
	}

	return expiryReason(session, m.Now()) == ""
}

// ValidSession returns the live session the request's cookie names, or nil. The
// cookie is cleared when it names nothing, names something expired, or fails to
// prove itself.
func (m *Manager) ValidSession(ctx context.Context, w http.ResponseWriter, r *http.Request) *storage.AuthSession {
	if m.Config == nil {
		return nil
	}

	sessionID, secret, ok := m.ParseCookie(r)
	if !ok {
		// A cookie that cannot be decoded is still a cookie the browser will keep
		// sending, so drop it. Nothing to drop when there was none.
		if c, err := r.Cookie(m.Config.CookieName); err == nil && c.Value != "" {
			m.ClearCookie(w)
		}
		return nil
	}

	session, err := m.Storage.GetAuthSession(ctx, sessionID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			m.Logger.ErrorContext(ctx, "failed to get auth session", "err", err)
		}
		m.ClearCookie(w)
		return nil
	}

	// The ID is published in every token as "sid", so it proves nothing on its own.
	// Constant-time, as a byte-by-byte comparison would leak the secret by timing.
	if subtle.ConstantTimeCompare([]byte(session.Secret), []byte(secret)) != 1 {
		m.Logger.DebugContext(ctx, "auth session secret mismatch", "session_id", sessionID)
		m.ClearCookie(w)
		return nil
	}

	// A browser presenting an expired session is the one place worth spending a
	// write on: the cookie is in hand, so the row can go now rather than waiting
	// for the collector.
	if reason := expiryReason(session, m.Now()); reason != "" {
		m.Logger.InfoContext(ctx, "auth session expired",
			"session_id", session.ID, "user_id", session.UserID, "reason", reason)
		if err := m.Storage.DeleteAuthSession(ctx, session.ID); err != nil {
			m.Logger.DebugContext(ctx, "failed to delete expired auth session", "err", err)
		}
		m.ClearCookie(w)
		return nil
	}

	return &session
}

// ValidAuthSession returns a valid session matching the auth request's connector, or nil.
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

// CreateOrUpdateAuthSession records a successful login on this browser's session
// and sets the cookie. rememberMe makes that cookie persistent.
//
// The browser's own cookie decides whether there is a session to continue: a login
// from a second device joins nothing, it starts its own session, which is what makes
// signing out on one device leave the other alone. A cookie naming someone else's
// session — another user, or another connector — is about to be overwritten, so the
// session it named is deleted rather than left unreachable until it times out.
func (m *Manager) CreateOrUpdateAuthSession(ctx context.Context, r *http.Request, w http.ResponseWriter, authReq storage.AuthRequest, rememberMe bool) error {
	if m.Config == nil {
		return nil
	}

	now := m.Now()
	userID := authReq.Claims.UserID
	connectorID := authReq.ConnectorID

	clientState := &storage.ClientAuthState{
		AuthenticatedAt: now,
		LastActivity:    now,
	}

	if current := m.ValidSession(ctx, w, r); current != nil {
		if current.UserID == userID && current.ConnectorID == connectorID {
			m.Logger.DebugContext(ctx, "updating existing auth session",
				"session_id", current.ID, "user_id", userID, "client_id", authReq.ClientID)

			if err := m.Storage.UpdateAuthSession(ctx, current.ID, func(old storage.AuthSession) (storage.AuthSession, error) {
				old.LastActivity = now
				old.IdleExpiry = now.Add(m.Config.ValidIfNotUsedFor)
				old.IPAddress = remoteIP(r)
				old.UserAgent = r.UserAgent()
				if old.ClientStates == nil {
					old.ClientStates = make(map[string]*storage.ClientAuthState)
				}
				old.ClientStates[authReq.ClientID] = clientState
				return old, nil
			}); err != nil {
				return fmt.Errorf("update auth session: %w", err)
			}

			m.SetCookie(w, current.ID, current.Secret, rememberMe)
			return nil
		}

		if err := m.Storage.DeleteAuthSession(ctx, current.ID); err != nil && !errors.Is(err, storage.ErrNotFound) {
			m.Logger.ErrorContext(ctx, "failed to delete superseded auth session", "err", err)
		}
	}

	newSession := storage.AuthSession{
		ID:          storage.NewID(),
		Secret:      storage.NewID(),
		UserID:      userID,
		ConnectorID: connectorID,
		ClientStates: map[string]*storage.ClientAuthState{
			authReq.ClientID: clientState,
		},
		CreatedAt:      now,
		LastActivity:   now,
		IPAddress:      remoteIP(r),
		UserAgent:      r.UserAgent(),
		AbsoluteExpiry: now.Add(m.Config.AbsoluteLifetime),
		IdleExpiry:     now.Add(m.Config.ValidIfNotUsedFor),
	}

	if err := m.Storage.CreateAuthSession(ctx, newSession); err != nil {
		return fmt.Errorf("create auth session: %w", err)
	}

	m.Logger.DebugContext(ctx, "created new auth session",
		"session_id", newSession.ID, "user_id", userID, "client_id", authReq.ClientID)
	m.SetCookie(w, newSession.ID, newSession.Secret, rememberMe)
	return nil
}

// ClientSharesWith reports whether sourceClient's ssoSharedWith policy lets its
// authenticated state be reused for targetClientID. A nil policy falls back to
// the server-wide SSOSharedWithDefault; an explicit empty slice shares with no one.
func (m *Manager) ClientSharesWith(sourceClient storage.Client, targetClientID string) bool {
	ssoSharedWith := sourceClient.SSOSharedWith

	// If client has no explicit ssoSharedWith, use default from session config.
	if ssoSharedWith == nil {
		switch m.Config.SSOSharedWithDefault {
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

// FindSSO checks whether any active client in the session shares its
// authentication with targetClientID via the ssoSharedWith policy.
//
// Note: the caller already has the target client loaded (for AllowedConnectors
// validation), but here we need the *source* client configs - those are the
// clients the user previously authenticated for, and their ssoSharedWith
// policies determine whether SSO is allowed. These are different clients,
// so the GetClient calls below are not redundant.
func (m *Manager) FindSSO(ctx context.Context, session *storage.AuthSession, targetClientID string) *storage.ClientAuthState {
	for sourceClientID, state := range session.ClientStates {
		// Only directly-authenticated states may act as SSO sources. Skipping
		// SSO-derived states keeps sharing unidirectional and prevents transitive
		// A->B->C chains (a user authenticated only to A must not be SSO'd into C
		// just because B shares with C).
		if state.ViaSSO {
			continue
		}

		sourceClient, err := m.Storage.GetClient(ctx, sourceClientID)
		if err != nil {
			m.Logger.DebugContext(ctx, "session: SSO lookup failed to get source client",
				"source_client_id", sourceClientID, "err", err)
			continue
		}

		if m.ClientSharesWith(sourceClient, targetClientID) {
			return state
		}
	}

	return nil
}

// UpdateTokenIssuedAt records that a token was just issued to clientID: it refreshes
// the session's idle timeout and the client state's last-issued timestamp. A missing,
// undecodable or superseded cookie is a no-op.
func (m *Manager) UpdateTokenIssuedAt(r *http.Request, clientID string) {
	if m.Config == nil {
		return
	}

	sessionID, secret, ok := m.ParseCookie(r)
	if !ok {
		return
	}

	now := m.Now()
	_ = m.Storage.UpdateAuthSession(r.Context(), sessionID, func(old storage.AuthSession) (storage.AuthSession, error) {
		// Prove the cookie as every other session-trusting path does: a stale one
		// must not refresh a live session's idle timeout. Aborting leaves it untouched.
		if subtle.ConstantTimeCompare([]byte(old.Secret), []byte(secret)) != 1 {
			return old, errors.New("session secret mismatch")
		}
		old.LastActivity = now
		old.IdleExpiry = now.Add(m.Config.ValidIfNotUsedFor)
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
func (m *Manager) ParseCookie(r *http.Request) (sessionID, secret string, ok bool) {
	if m.Config == nil {
		return "", "", false
	}
	cookie, err := r.Cookie(m.Config.CookieName)
	if err != nil || cookie.Value == "" {
		return "", "", false
	}
	id, sec, err := internal.ParseSessionCookie(cookie.Value, m.Config.CookieEncryptionKey)
	if err != nil {
		m.Logger.DebugContext(r.Context(), "invalid session cookie format", "err", err)
		return "", "", false
	}
	return id, sec, true
}

// AbsoluteExpiry returns the absolute session expiry measured from now.
func (m *Manager) AbsoluteExpiry(now time.Time) time.Time {
	return now.Add(m.Config.AbsoluteLifetime)
}

// IdleExpiry returns the idle-timeout expiry measured from now.
func (m *Manager) IdleExpiry(now time.Time) time.Time {
	return now.Add(m.Config.ValidIfNotUsedFor)
}
