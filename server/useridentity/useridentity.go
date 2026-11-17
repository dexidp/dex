// Package useridentity owns the per-user account record every login path
// shares: each login must enforce the account lock and keep the stored
// identity's claims and last-login time current, or the paths drift apart.
package useridentity

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// LockedError reports a login refused because the stored identity's
// BlockedUntil is still in the future.
type LockedError struct {
	Until time.Time
}

func (e *LockedError) Error() string {
	return fmt.Sprintf("account is locked until %s", e.Until.Format(time.RFC3339))
}

// RecordLogin refuses to proceed for a locked account, then creates the
// user's identity row on first login or refreshes its claims and last-login
// time on later ones.
//
// Call it only when the sessions feature is enabled: BlockedUntil and the
// UserIdentity row itself do not exist without it, and a first-time login
// (no stored identity yet) cannot be blocked.
func RecordLogin(ctx context.Context, s storage.Storage, logger *slog.Logger, now func() time.Time, identity connector.Identity, connectorID string) error {
	stored, err := s.GetUserIdentity(ctx, identity.UserID, connectorID)
	switch {
	case errors.Is(err, storage.ErrNotFound):
		t := now()
		ui := storage.UserIdentity{
			UserID:      identity.UserID,
			ConnectorID: connectorID,
			Claims:      tokens.ClaimsFromIdentity(identity),
			Consents:    make(map[string][]string),
			CreatedAt:   t,
			LastLogin:   t,
		}
		if err := s.CreateUserIdentity(ctx, ui); err != nil {
			logger.ErrorContext(ctx, "failed to create user identity", "err", err)
			return err
		}
		return nil
	case err != nil:
		return fmt.Errorf("failed to look up user identity: %w", err)
	}

	if !stored.BlockedUntil.IsZero() && now().Before(stored.BlockedUntil) {
		logger.WarnContext(ctx, "login rejected for locked account",
			"connector_id", connectorID, "user_id", identity.UserID, "blocked_until", stored.BlockedUntil)
		return &LockedError{Until: stored.BlockedUntil}
	}

	if err := s.UpdateUserIdentity(ctx, identity.UserID, connectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
		old.Claims = tokens.ClaimsFromIdentity(identity)
		old.LastLogin = now()
		return old, nil
	}); err != nil {
		logger.ErrorContext(ctx, "failed to update user identity", "err", err)
		return err
	}
	return nil
}
