package server

import (
	"context"
	"errors"
	"log/slog"

	"github.com/dexidp/dex/storage"
)

// revokeRefreshTokensFromStorage deletes all refresh tokens for the given
// user/connector pair and clears the references in the offline session
// (but keeps the session object). Returns the connector data from the
// offline session (for upstream logout).
//
// Errors are logged but not returned (best-effort revocation).
//
// To avoid a race condition where a new token issued between deletion and
// the OfflineSessions update would have its reference wiped, we:
//  1. Snapshot the token IDs to revoke
//  2. Remove only those specific references from OfflineSessions (the updater
//     sees the latest state, so concurrently added refs are preserved)
//  3. Delete the actual refresh tokens
func revokeRefreshTokensFromStorage(ctx context.Context, s storage.Storage, logger *slog.Logger, userID, connectorID string) []byte {
	offlineSessions, err := s.GetOfflineSessions(ctx, userID, connectorID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			logger.ErrorContext(ctx, "failed to get offline sessions", "err", err)
		}
		return nil
	}

	tokenIDs := make(map[string]struct{}, len(offlineSessions.Refresh))
	for _, ref := range offlineSessions.Refresh {
		tokenIDs[ref.ID] = struct{}{}
	}

	if err := s.UpdateOfflineSessions(ctx, userID, connectorID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
		for clientID, ref := range old.Refresh {
			if _, ok := tokenIDs[ref.ID]; ok {
				delete(old.Refresh, clientID)
			}
		}
		return old, nil
	}); err != nil {
		logger.ErrorContext(ctx, "failed to update offline sessions", "err", err)
	}

	for id := range tokenIDs {
		if err := s.DeleteRefresh(ctx, id); err != nil && !errors.Is(err, storage.ErrNotFound) {
			logger.ErrorContext(ctx, "failed to delete refresh token", "token_id", id, "err", err)
		}
	}

	return offlineSessions.ConnectorData
}
