package authflow

// finalize.go implements the post-authentication step shared by every login
// mechanism: it persists the identity onto the AuthRequest, records the offline
// session and the user identity, then returns the finalized request.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// finalizeLogin associates the user's identity with the current AuthRequest, then returns
// the approval page's path.
func (h *Handler) finalizeLogin(ctx context.Context, identity connector.Identity, authReq storage.AuthRequest, conn connector.Connector) (storage.AuthRequest, error) {
	// Refuse to complete login for a locked account. BlockedUntil lives on the
	// persisted UserIdentity, which only exists when the sessions feature is on;
	// a first-time login (no stored identity yet) cannot be blocked.
	if h.Sessions.Enabled() {
		storedIdentity, err := h.Storage.GetUserIdentity(ctx, identity.UserID, authReq.ConnectorID)
		switch {
		case err == nil:
			if !storedIdentity.BlockedUntil.IsZero() && h.Now().Before(storedIdentity.BlockedUntil) {
				h.Logger.WarnContext(ctx, "login rejected for locked account",
					"connector_id", authReq.ConnectorID, "user_id", identity.UserID, "blocked_until", storedIdentity.BlockedUntil)
				return storage.AuthRequest{}, fmt.Errorf("account is locked until %s", storedIdentity.BlockedUntil.Format(time.RFC3339))
			}
		case !errors.Is(err, storage.ErrNotFound):
			return storage.AuthRequest{}, fmt.Errorf("failed to look up user identity: %w", err)
		}
	}

	claims := storage.Claims{
		UserID:            identity.UserID,
		Username:          identity.Username,
		PreferredUsername: identity.PreferredUsername,
		Email:             identity.Email,
		EmailVerified:     identity.EmailVerified,
		Groups:            identity.Groups,
	}

	updater := func(a storage.AuthRequest) (storage.AuthRequest, error) {
		a.LoggedIn = true
		a.Claims = claims
		a.ConnectorData = identity.ConnectorData
		a.AuthTime = h.Now()
		return a, nil
	}
	if err := h.Storage.UpdateAuthRequest(ctx, authReq.ID, updater); err != nil {
		return storage.AuthRequest{}, fmt.Errorf("failed to update auth request: %v", err)
	}
	// Keep the in-memory copy in sync with what was persisted so later reads
	// (the next-step decision below) see the identity we just stored.
	authReq, _ = updater(authReq)

	email := claims.Email
	if !claims.EmailVerified {
		email += " (unverified)"
	}

	h.Logger.InfoContext(ctx, "login successful",
		"connector_id", authReq.ConnectorID, "user_id", claims.UserID,
		"username", claims.Username, "preferred_username", claims.PreferredUsername,
		"email", email, "groups", claims.Groups)

	offlineAccessRequested := false
	for _, scope := range authReq.Scopes {
		if scope == tokens.ScopeOfflineAccess {
			offlineAccessRequested = true
			break
		}
	}
	_, canRefresh := conn.(connector.RefreshConnector)

	if offlineAccessRequested && canRefresh {
		// Try to retrieve an existing OfflineSession object for the corresponding user.
		session, err := h.Storage.GetOfflineSessions(ctx, identity.UserID, authReq.ConnectorID)
		switch {
		case err != nil && err == storage.ErrNotFound:
			offlineSessions := storage.OfflineSessions{
				UserID:        identity.UserID,
				ConnID:        authReq.ConnectorID,
				Refresh:       make(map[string]*storage.RefreshTokenRef),
				ConnectorData: identity.ConnectorData,
			}

			// Create a new OfflineSession object for the user and add a reference object for
			// the newly received refreshtoken.
			if err := h.Storage.CreateOfflineSessions(ctx, offlineSessions); err != nil {
				h.Logger.ErrorContext(ctx, "failed to create offline session", "err", err)
				return storage.AuthRequest{}, err
			}
		case err == nil:
			// Update existing OfflineSession obj with new RefreshTokenRef.
			if err := h.Storage.UpdateOfflineSessions(ctx, session.UserID, session.ConnID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
				if len(identity.ConnectorData) > 0 {
					old.ConnectorData = identity.ConnectorData
				}
				return old, nil
			}); err != nil {
				h.Logger.ErrorContext(ctx, "failed to update offline session", "err", err)
				return storage.AuthRequest{}, err
			}
		default:
			h.Logger.ErrorContext(ctx, "failed to get offline session", "err", err)
			return storage.AuthRequest{}, err
		}
	}

	// Create or update UserIdentity to persist user claims across sessions.
	if h.Sessions.Enabled() {
		now := h.Now()

		_, err := h.Storage.GetUserIdentity(ctx, identity.UserID, authReq.ConnectorID)
		switch {
		case err != nil && errors.Is(err, storage.ErrNotFound):
			ui := storage.UserIdentity{
				UserID:      identity.UserID,
				ConnectorID: authReq.ConnectorID,
				Claims:      claims,
				Consents:    make(map[string][]string),
				CreatedAt:   now,
				LastLogin:   now,
			}
			if err := h.Storage.CreateUserIdentity(ctx, ui); err != nil {
				h.Logger.ErrorContext(ctx, "failed to create user identity", "err", err)
				return storage.AuthRequest{}, err
			}
		case err == nil:
			if err := h.Storage.UpdateUserIdentity(ctx, identity.UserID, authReq.ConnectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
				old.Claims = claims
				old.LastLogin = now
				return old, nil
			}); err != nil {
				h.Logger.ErrorContext(ctx, "failed to update user identity", "err", err)
				return storage.AuthRequest{}, err
			}
		default:
			h.Logger.ErrorContext(ctx, "failed to get user identity", "err", err)
			return storage.AuthRequest{}, err
		}
	}

	// The identity is persisted; return the finalized request so the caller can
	// create the session and advance the flow.
	return h.Storage.GetAuthRequest(ctx, authReq.ID)
}
