package authflow

import (
	"context"
	"net/http"
	"time"

	"github.com/dexidp/dex/storage"
)

func (h *Handler) trySessionLogin(ctx context.Context, r *http.Request, w http.ResponseWriter, authReq *storage.AuthRequest) bool {
	session := h.Sessions.ValidAuthSession(ctx, w, r, authReq)
	return h.trySessionLoginWithSession(ctx, r, w, authReq, session)
}

// trySessionLoginWithSession completes the login from an existing session: a
// direct session for the client, or, failing that, an SSO session shared by
// another client. SSO sharing is unidirectional — a source sharing with a target
// does not mean the target shares back. Returns false when no session applies.
func (h *Handler) trySessionLoginWithSession(ctx context.Context, r *http.Request, w http.ResponseWriter, authReq *storage.AuthRequest, session *storage.AuthSession) bool {
	if session == nil {
		return false
	}

	now := h.Now()

	clientState, ok := session.ClientStates[authReq.ClientID]
	fallbackToSSO := !ok || !clientState.Active || now.After(clientState.ExpiresAt)

	if fallbackToSSO {
		// No direct session for this client — try SSO from a sharing client.
		sourceState := h.Sessions.FindSSO(ctx, session, authReq.ClientID)
		if sourceState == nil {
			return false
		}

		// Cap the derived state expiry at min(configured lifetime, source state expiry).
		expiresAt := h.Sessions.AbsoluteExpiry(now)
		if sourceState.ExpiresAt.Before(expiresAt) {
			expiresAt = sourceState.ExpiresAt
		}

		// Create a new client state for the target client via SSO.
		if err := h.Storage.UpdateAuthSession(ctx, session.UserID, session.ConnectorID, func(old storage.AuthSession) (storage.AuthSession, error) {
			if old.ClientStates == nil {
				old.ClientStates = make(map[string]*storage.ClientAuthState)
			}
			old.ClientStates[authReq.ClientID] = &storage.ClientAuthState{
				Active:       true,
				ExpiresAt:    expiresAt,
				LastActivity: now,
				ViaSSO:       true,
			}
			old.LastActivity = now
			old.IdleExpiry = h.Sessions.IdleExpiry(now)
			return old, nil
		}); err != nil {
			h.Logger.ErrorContext(ctx, "session: failed to create SSO client state", "err", err)
			return false
		}

		h.Logger.DebugContext(ctx, "session: SSO login from sharing client",
			"user_id", session.UserID, "connector_id", session.ConnectorID, "client_id", authReq.ClientID)
	}

	// Load identity from storage (same path for direct and SSO login).
	ui, err := h.Storage.GetUserIdentity(ctx, session.UserID, session.ConnectorID)
	if err != nil {
		h.Logger.ErrorContext(ctx, "session: failed to get user identity", "err", err)
		return false
	}

	// Check max_age: if the user's last authentication is too old, force re-auth.
	if authReq.MaxAge >= 0 {
		if now.Sub(ui.LastLogin) > time.Duration(authReq.MaxAge)*time.Second {
			return false
		}
	}

	if !fallbackToSSO {
		h.Logger.DebugContext(ctx, "session: re-authenticated from session",
			"user_id", session.UserID, "connector_id", session.ConnectorID)
	}

	return h.finishSessionLogin(ctx, r, w, authReq, session, &ui, now)
}

// finishSessionLogin completes a session-based login (direct or SSO) by updating the auth request
// with the user's identity, refreshing session activity, and returning the appropriate redirect URL.
func (h *Handler) finishSessionLogin(ctx context.Context, r *http.Request, w http.ResponseWriter, authReq *storage.AuthRequest, session *storage.AuthSession, ui *storage.UserIdentity, now time.Time) bool {
	claims := storage.Claims{
		UserID:            ui.Claims.UserID,
		Username:          ui.Claims.Username,
		PreferredUsername: ui.Claims.PreferredUsername,
		Email:             ui.Claims.Email,
		EmailVerified:     ui.Claims.EmailVerified,
		Groups:            ui.Claims.Groups,
	}

	// Update AuthRequest with stored identity and auth_time from last login.
	if err := h.Storage.UpdateAuthRequest(ctx, authReq.ID, func(a storage.AuthRequest) (storage.AuthRequest, error) {
		a.LoggedIn = true
		a.Claims = claims
		a.ConnectorID = session.ConnectorID
		a.AuthTime = ui.LastLogin
		return a, nil
	}); err != nil {
		h.Logger.ErrorContext(ctx, "session: failed to update auth request", "err", err)
		return false
	}

	// Update session activity.
	_ = h.Storage.UpdateAuthSession(ctx, session.UserID, session.ConnectorID, func(old storage.AuthSession) (storage.AuthSession, error) {
		old.LastActivity = now
		old.IdleExpiry = h.Sessions.IdleExpiry(now)
		if cs, ok := old.ClientStates[authReq.ClientID]; ok {
			cs.LastActivity = now
		}
		return old, nil
	})

	// Re-read to get the updated AuthRequest (LoggedIn, Claims, ConnectorID set above),
	// then let the shared decision pick the next step.
	updated, err := h.Storage.GetAuthRequest(ctx, authReq.ID)
	if err != nil {
		h.Logger.ErrorContext(ctx, "session: failed to get auth request", "err", err)
		return false
	}
	http.Redirect(w, r, h.buildContinueURL(updated), http.StatusSeeOther)
	return true
}

// updateSessionTokenIssuedAt updates the session's LastTokenIssuedAt for the given client.
