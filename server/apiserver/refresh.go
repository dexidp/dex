package apiserver

import (
	"context"

	"github.com/dexidp/dex/api/v2"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

func (d dexAPI) ListRefresh(ctx context.Context, req *api.ListRefreshReq) (*api.ListRefreshResp, error) {
	id := new(internal.IDTokenSubject)
	if err := internal.Unmarshal(req.UserId, id); err != nil {
		d.logger.Error("failed to unmarshal ID Token subject", "err", err)
		return nil, err
	}

	offlineSessions, err := d.s.GetOfflineSessions(ctx, id.UserId, id.ConnId)
	if err != nil {
		if err == storage.ErrNotFound {
			// This means that this user-client pair does not have a refresh token yet.
			// An empty list should be returned instead of an error.
			return &api.ListRefreshResp{}, nil
		}
		d.logger.Error("failed to list refresh tokens here", "err", err)
		return nil, err
	}

	refreshTokenRefs := make([]*api.RefreshTokenRef, 0, len(offlineSessions.Refresh))
	for _, session := range offlineSessions.Refresh {
		r := api.RefreshTokenRef{
			Id:        session.ID,
			ClientId:  session.ClientID,
			CreatedAt: session.CreatedAt.Unix(),
			LastUsed:  session.LastUsed.Unix(),
		}
		refreshTokenRefs = append(refreshTokenRefs, &r)
	}

	return &api.ListRefreshResp{
		RefreshTokens: refreshTokenRefs,
	}, nil
}

func (d dexAPI) RevokeRefresh(ctx context.Context, req *api.RevokeRefreshReq) (*api.RevokeRefreshResp, error) {
	id := new(internal.IDTokenSubject)
	if err := internal.Unmarshal(req.UserId, id); err != nil {
		d.logger.Error("failed to unmarshal ID Token subject", "err", err)
		return nil, err
	}

	var (
		refreshID string
		notFound  bool
	)
	updater := func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
		refreshRef := old.Refresh[req.ClientId]
		if refreshRef == nil || refreshRef.ID == "" {
			d.logger.Error("refresh token issued to client not found for deletion", "client_id", req.ClientId, "user_id", id.UserId)
			notFound = true
			return old, storage.ErrNotFound
		}

		refreshID = refreshRef.ID

		// Remove entry from Refresh list of the OfflineSession object.
		delete(old.Refresh, req.ClientId)

		return old, nil
	}

	if err := d.s.UpdateOfflineSessions(ctx, id.UserId, id.ConnId, updater); err != nil {
		if err == storage.ErrNotFound {
			return &api.RevokeRefreshResp{NotFound: true}, nil
		}
		d.logger.Error("failed to update offline session object", "err", err)
		return nil, err
	}

	if notFound {
		return &api.RevokeRefreshResp{NotFound: true}, nil
	}

	// Delete the refresh token from the storage
	//
	// TODO(ericchiang): we don't have any good recourse if this call fails.
	// Consider garbage collection of refresh tokens with no associated ref.
	if err := d.s.DeleteRefresh(ctx, refreshID); err != nil {
		d.logger.Error("failed to delete refresh token", "err", err)
		return nil, err
	}

	return &api.RevokeRefreshResp{}, nil
}

// revokeUserRefreshTokens revokes all refresh tokens for a user/connector pair
// and cleans up offline session references. Errors are logged but not returned
// (best-effort).
func (d dexAPI) revokeUserRefreshTokens(ctx context.Context, userID, connectorID string) {
	d.refresh.Revoke(ctx, userID, connectorID)
}
