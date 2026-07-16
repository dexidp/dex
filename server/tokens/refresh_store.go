package tokens

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

// RefreshStore creates, rotates, and revokes refresh tokens, keeping the
// offline-session references in sync. It is the single owner of refresh-token
// persistence.
type RefreshStore struct {
	storage storage.Storage
	now     func() time.Time
	logger  *slog.Logger
}

// NewRefreshStore returns a store backed by the given storage.
func NewRefreshStore(storage storage.Storage, now func() time.Time, logger *slog.Logger) *RefreshStore {
	return &RefreshStore{storage: storage, now: now, logger: logger}
}

// Create issues a new refresh token for the authorization and records it in the
// user's offline session. On any offline-session failure the refresh token is
// rolled back so no orphaned token is left behind.
func (rt *RefreshStore) Create(ctx context.Context, auth Authorization) (string, error) {
	refresh := storage.RefreshToken{
		ID:            storage.NewID(),
		Token:         storage.NewID(),
		ClientID:      auth.Client.ID,
		ConnectorID:   auth.ConnectorID,
		Scopes:        auth.Scopes,
		Claims:        auth.Claims,
		Nonce:         auth.Nonce,
		ConnectorData: auth.ConnectorData,
		CreatedAt:     rt.now(),
		LastUsed:      rt.now(),
	}

	rawToken, err := internal.Marshal(&internal.RefreshToken{RefreshId: refresh.ID, Token: refresh.Token})
	if err != nil {
		return "", fmt.Errorf("marshal refresh token: %w", err)
	}

	if err := rt.storage.CreateRefresh(ctx, refresh); err != nil {
		return "", fmt.Errorf("create refresh token: %w", err)
	}

	// Roll back the just-created refresh token if wiring it into the offline
	// session fails, so we never leave an orphaned token.
	rollback := func() {
		if err := rt.storage.DeleteRefresh(ctx, refresh.ID); err != nil && err != storage.ErrNotFound {
			rt.logger.ErrorContext(ctx, "failed to roll back refresh token", "err", err)
		}
	}

	tokenRef := storage.RefreshTokenRef{
		ID:        refresh.ID,
		ClientID:  refresh.ClientID,
		CreatedAt: refresh.CreatedAt,
		LastUsed:  refresh.LastUsed,
	}

	session, err := rt.storage.GetOfflineSessions(ctx, refresh.Claims.UserID, refresh.ConnectorID)
	switch {
	case err != nil && err != storage.ErrNotFound:
		rollback()
		return "", fmt.Errorf("get offline session: %w", err)
	case err != nil: // ErrNotFound: create a fresh offline session.
		offlineSessions := storage.OfflineSessions{
			UserID:        refresh.Claims.UserID,
			ConnID:        refresh.ConnectorID,
			Refresh:       map[string]*storage.RefreshTokenRef{tokenRef.ClientID: &tokenRef},
			ConnectorData: auth.ConnectorData,
		}
		if err := rt.storage.CreateOfflineSessions(ctx, offlineSessions); err != nil {
			rollback()
			return "", fmt.Errorf("create offline session: %w", err)
		}
	default:
		// Replace any existing refresh token for this client.
		if oldRef, ok := session.Refresh[tokenRef.ClientID]; ok {
			if err := rt.storage.DeleteRefresh(ctx, oldRef.ID); err != nil && err != storage.ErrNotFound {
				rollback()
				return "", fmt.Errorf("delete previous refresh token: %w", err)
			}
		}
		if err := rt.storage.UpdateOfflineSessions(ctx, session.UserID, session.ConnID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
			old.Refresh[tokenRef.ClientID] = &tokenRef
			if len(auth.ConnectorData) > 0 {
				old.ConnectorData = auth.ConnectorData
			}
			return old, nil
		}); err != nil {
			rollback()
			return "", fmt.Errorf("update offline session: %w", err)
		}
	}

	return rawToken, nil
}

// IdentityFromClaims projects stored claims into a connector identity. The
// refresh flow intentionally leaves UserID untouched by later updates.
func IdentityFromClaims(claims storage.Claims) connector.Identity {
	return connector.Identity{
		UserID:            claims.UserID,
		Username:          claims.Username,
		PreferredUsername: claims.PreferredUsername,
		Email:             claims.Email,
		EmailVerified:     claims.EmailVerified,
		Groups:            claims.Groups,
	}
}

// ClaimsFromIdentity maps a freshly authenticated connector identity to ID token
// claims. It is the inverse of IdentityFromClaims and the single place grants
// turn an identity into claims.
func ClaimsFromIdentity(identity connector.Identity) storage.Claims {
	return storage.Claims{
		UserID:            identity.UserID,
		Username:          identity.Username,
		PreferredUsername: identity.PreferredUsername,
		Email:             identity.Email,
		EmailVerified:     identity.EmailVerified,
		Groups:            identity.Groups,
	}
}

// Rotate advances the stored refresh token according to the rotation strategy and
// syncs the offline session. It returns the marshaled token to hand back to the
// client and the identity its claims were refreshed to.
//
// freshIdentity supplies the up-to-date identity. It is invoked inside the
// storage transaction, and only when a token is actually minted, so the upstream
// connector is contacted at most once even under concurrent refreshes.
func (rt *RefreshStore) Rotate(ctx context.Context, storageToken *storage.RefreshToken, requestToken *internal.RefreshToken, strategy *RefreshStrategy, freshIdentity func(context.Context) (connector.Identity, error)) (string, connector.Identity, error) {
	var idErr error

	newToken := &internal.RefreshToken{
		Token:     requestToken.Token,
		RefreshId: requestToken.RefreshId,
	}

	lastUsed := rt.now()
	ident := IdentityFromClaims(storageToken.Claims)

	refreshTokenUpdater := func(old storage.RefreshToken) (storage.RefreshToken, error) {
		rotationEnabled := strategy.RotationEnabled()
		reusingAllowed := strategy.AllowedToReuse(old.LastUsed)

		switch {
		case !rotationEnabled && reusingAllowed:
			// Rotation disabled and the token was used recently: nothing to advance.
			old.ConnectorData = nil
			return old, nil

		case rotationEnabled && reusingAllowed:
			if old.Token != requestToken.Token && old.ObsoleteToken != requestToken.Token {
				return old, errors.New("refresh token claimed twice")
			}

			// Return the previously generated token for requests carrying an obsolete one.
			if old.ObsoleteToken == requestToken.Token {
				newToken.Token = old.Token
			}

			// Do not advance last-used while the token may still be reused.
			lastUsed = old.LastUsed
			old.ConnectorData = nil
			return old, nil

		case rotationEnabled && !reusingAllowed:
			if old.Token != requestToken.Token {
				return old, errors.New("refresh token claimed twice")
			}

			// Issue a new refresh token, keeping the previous one claimable within reuse.
			old.ObsoleteToken = old.Token
			newToken.Token = storage.NewID()
		}

		old.Token = newToken.Token
		old.LastUsed = lastUsed

		// ConnectorData has been moved to the offline session.
		old.ConnectorData = nil

		ident, idErr = freshIdentity(ctx)
		if idErr != nil {
			return old, idErr
		}

		// Refresh the stored claims. UserID intentionally left untouched.
		old.Claims.Username = ident.Username
		old.Claims.PreferredUsername = ident.PreferredUsername
		old.Claims.Email = ident.Email
		old.Claims.EmailVerified = ident.EmailVerified
		old.Claims.Groups = ident.Groups

		return old, nil
	}

	if err := rt.storage.UpdateRefreshToken(ctx, storageToken.ID, refreshTokenUpdater); err != nil {
		rt.logger.ErrorContext(ctx, "failed to update refresh token", "err", err)
		return "", ident, err
	}

	if err := rt.updateOfflineSession(ctx, storageToken, ident, lastUsed); err != nil {
		return "", ident, err
	}

	rawToken, err := internal.Marshal(newToken)
	if err != nil {
		return "", ident, fmt.Errorf("marshal refresh token: %w", err)
	}

	return rawToken, ident, nil
}

// updateOfflineSession records the refresh token's new last-used time, and any
// updated connector data, on the user's offline session.
func (rt *RefreshStore) updateOfflineSession(ctx context.Context, refresh *storage.RefreshToken, ident connector.Identity, lastUsed time.Time) error {
	offlineSessionUpdater := func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
		if old.Refresh[refresh.ClientID].ID != refresh.ID {
			return old, errors.New("refresh token invalid")
		}

		old.Refresh[refresh.ClientID].LastUsed = lastUsed
		if len(ident.ConnectorData) > 0 {
			old.ConnectorData = ident.ConnectorData
		}

		rt.logger.DebugContext(ctx, "saved connector data", "user_id", ident.UserID, "connector_data", ident.ConnectorData)
		return old, nil
	}

	if err := rt.storage.UpdateOfflineSessions(ctx, refresh.Claims.UserID, refresh.ConnectorID, offlineSessionUpdater); err != nil {
		rt.logger.ErrorContext(ctx, "failed to update offline session", "err", err)
		return err
	}

	return nil
}

// Revoke deletes every refresh token for the user/connector pair and clears the
// references in the offline session, keeping the session object. Errors are
// logged but not returned: revocation is best-effort.
//
// To avoid a race where a token issued between the snapshot and the offline
// session update would have its reference wiped, we snapshot the token IDs,
// remove only those references (the updater sees the latest state, so a
// concurrently added reference survives), then delete the tokens.
func (rt *RefreshStore) Revoke(ctx context.Context, userID, connectorID string) {
	offlineSessions, err := rt.storage.GetOfflineSessions(ctx, userID, connectorID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			rt.logger.ErrorContext(ctx, "failed to get offline sessions", "err", err)
		}
		return
	}

	tokenIDs := make(map[string]struct{}, len(offlineSessions.Refresh))
	for _, ref := range offlineSessions.Refresh {
		tokenIDs[ref.ID] = struct{}{}
	}

	if err := rt.storage.UpdateOfflineSessions(ctx, userID, connectorID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
		for clientID, ref := range old.Refresh {
			if _, ok := tokenIDs[ref.ID]; ok {
				delete(old.Refresh, clientID)
			}
		}
		return old, nil
	}); err != nil {
		rt.logger.ErrorContext(ctx, "failed to update offline sessions", "err", err)
	}

	for id := range tokenIDs {
		if err := rt.storage.DeleteRefresh(ctx, id); err != nil && !errors.Is(err, storage.ErrNotFound) {
			rt.logger.ErrorContext(ctx, "failed to delete refresh token", "token_id", id, "err", err)
		}
	}
}
