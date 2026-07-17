package apiserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/dexidp/dex/api/v2"
	"github.com/dexidp/dex/pkg/featureflags"
	"github.com/dexidp/dex/storage"
)

func storageAuthSessionToAPI(s storage.AuthSession) *api.AuthSession {
	clientStates := make([]*api.ClientAuthState, 0, len(s.ClientStates))
	for clientID, state := range s.ClientStates {
		if state == nil {
			continue
		}
		clientStates = append(clientStates, &api.ClientAuthState{
			ClientId:          clientID,
			Active:            state.Active,
			ExpiresAt:         unixOrZero(state.ExpiresAt),
			LastActivity:      unixOrZero(state.LastActivity),
			LastTokenIssuedAt: unixOrZero(state.LastTokenIssuedAt),
		})
	}

	return &api.AuthSession{
		UserId:         s.UserID,
		ConnectorId:    s.ConnectorID,
		ClientStates:   clientStates,
		CreatedAt:      unixOrZero(s.CreatedAt),
		LastActivity:   unixOrZero(s.LastActivity),
		IpAddress:      s.IPAddress,
		UserAgent:      s.UserAgent,
		AbsoluteExpiry: unixOrZero(s.AbsoluteExpiry),
		IdleExpiry:     unixOrZero(s.IdleExpiry),
	}
}

func (d dexAPI) GetAuthSession(ctx context.Context, req *api.GetAuthSessionReq) (*api.GetAuthSessionResp, error) {
	if !featureflags.APISessionsIdentitiesCRUD.Enabled() {
		return nil, fmt.Errorf("%s feature flag is not enabled", featureflags.APISessionsIdentitiesCRUD.Name)
	}

	if req.UserId == "" {
		return nil, errors.New("no user_id supplied")
	}
	if req.ConnectorId == "" {
		return nil, errors.New("no connector_id supplied")
	}

	session, err := d.s.GetAuthSession(ctx, req.UserId, req.ConnectorId)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, storage.ErrNotFound
		}
		d.logger.Error("api: failed to get auth session", "err", err)
		return nil, fmt.Errorf("get auth session: %v", err)
	}

	return &api.GetAuthSessionResp{
		Session: storageAuthSessionToAPI(session),
	}, nil
}

func (d dexAPI) ListAuthSessions(ctx context.Context, req *api.ListAuthSessionsReq) (*api.ListAuthSessionsResp, error) {
	if !featureflags.APISessionsIdentitiesCRUD.Enabled() {
		return nil, fmt.Errorf("%s feature flag is not enabled", featureflags.APISessionsIdentitiesCRUD.Name)
	}

	sessionList, err := d.s.ListAuthSessions(ctx)
	if err != nil {
		d.logger.Error("api: failed to list auth sessions", "err", err)
		return nil, fmt.Errorf("list auth sessions: %v", err)
	}

	sessions := make([]*api.AuthSession, 0, len(sessionList))
	for _, s := range sessionList {
		if req.UserId != "" && s.UserID != req.UserId {
			continue
		}
		sessions = append(sessions, storageAuthSessionToAPI(s))
	}

	return &api.ListAuthSessionsResp{
		Sessions: sessions,
	}, nil
}

func (d dexAPI) DeleteAuthSession(ctx context.Context, req *api.DeleteAuthSessionReq) (*api.DeleteAuthSessionResp, error) {
	if !featureflags.APISessionsIdentitiesCRUD.Enabled() {
		return nil, fmt.Errorf("%s feature flag is not enabled", featureflags.APISessionsIdentitiesCRUD.Name)
	}

	if req.UserId == "" {
		return nil, errors.New("no user_id supplied")
	}
	if req.ConnectorId == "" {
		return nil, errors.New("no connector_id supplied")
	}

	// Revoke refresh tokens (best-effort, consistent with logout flow).
	d.revokeUserRefreshTokens(ctx, req.UserId, req.ConnectorId)

	if err := d.s.DeleteAuthSession(ctx, req.UserId, req.ConnectorId); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return &api.DeleteAuthSessionResp{NotFound: true}, nil
		}
		d.logger.Error("api: failed to delete auth session", "err", err)
		return nil, fmt.Errorf("delete auth session: %v", err)
	}

	d.logger.Info("api: deleted auth session", "user_id", req.UserId, "connector_id", req.ConnectorId)
	return &api.DeleteAuthSessionResp{}, nil
}

func (d dexAPI) terminateSessions(ctx context.Context, match func(storage.AuthSession) bool) (int64, error) {
	sessionList, err := d.s.ListAuthSessions(ctx)
	if err != nil {
		d.logger.Error("api: failed to list auth sessions", "err", err)
		return 0, fmt.Errorf("list auth sessions: %v", err)
	}

	var terminated int64
	for _, s := range sessionList {
		if !match(s) {
			continue
		}

		d.revokeUserRefreshTokens(ctx, s.UserID, s.ConnectorID)

		if err := d.s.DeleteAuthSession(ctx, s.UserID, s.ConnectorID); err != nil {
			d.logger.Error("api: failed to delete auth session during batch terminate",
				"user_id", s.UserID, "connector_id", s.ConnectorID, "err", err)
			continue
		}
		terminated++
	}
	return terminated, nil
}

func (d dexAPI) TerminateSessionsByConnector(ctx context.Context, req *api.TerminateSessionsByConnectorReq) (*api.TerminateSessionsByConnectorResp, error) {
	if !featureflags.APISessionsIdentitiesCRUD.Enabled() {
		return nil, fmt.Errorf("%s feature flag is not enabled", featureflags.APISessionsIdentitiesCRUD.Name)
	}

	if req.ConnectorId == "" {
		return nil, errors.New("no connector_id supplied")
	}

	terminated, err := d.terminateSessions(ctx, func(s storage.AuthSession) bool {
		return s.ConnectorID == req.ConnectorId
	})
	if err != nil {
		return nil, err
	}

	d.logger.Info("api: terminated sessions by connector", "connector_id", req.ConnectorId, "count", terminated)
	return &api.TerminateSessionsByConnectorResp{SessionsTerminated: terminated}, nil
}

func (d dexAPI) TerminateSessionsByUser(ctx context.Context, req *api.TerminateSessionsByUserReq) (*api.TerminateSessionsByUserResp, error) {
	if !featureflags.APISessionsIdentitiesCRUD.Enabled() {
		return nil, fmt.Errorf("%s feature flag is not enabled", featureflags.APISessionsIdentitiesCRUD.Name)
	}

	if req.UserId == "" {
		return nil, errors.New("no user_id supplied")
	}

	terminated, err := d.terminateSessions(ctx, func(s storage.AuthSession) bool {
		return s.UserID == req.UserId
	})
	if err != nil {
		return nil, err
	}

	d.logger.Info("api: terminated sessions by user", "user_id", req.UserId, "count", terminated)
	return &api.TerminateSessionsByUserResp{SessionsTerminated: terminated}, nil
}

func (d dexAPI) RevokeConsent(ctx context.Context, req *api.RevokeConsentReq) (*api.RevokeConsentResp, error) {
	if !featureflags.APISessionsIdentitiesCRUD.Enabled() {
		return nil, fmt.Errorf("%s feature flag is not enabled", featureflags.APISessionsIdentitiesCRUD.Name)
	}

	if req.UserId == "" {
		return nil, errors.New("no user_id supplied")
	}
	if req.ConnectorId == "" {
		return nil, errors.New("no connector_id supplied")
	}
	if req.ClientId == "" {
		return nil, errors.New("no client_id supplied")
	}

	if err := d.s.UpdateUserIdentity(ctx, req.UserId, req.ConnectorId, func(old storage.UserIdentity) (storage.UserIdentity, error) {
		if _, ok := old.Consents[req.ClientId]; !ok {
			return old, errIdentityUnchanged
		}
		delete(old.Consents, req.ClientId)
		return old, nil
	}); err != nil {
		if errors.Is(err, errIdentityUnchanged) || errors.Is(err, storage.ErrNotFound) {
			return &api.RevokeConsentResp{NotFound: true}, nil
		}
		d.logger.Error("api: failed to revoke consent", "err", err)
		return nil, fmt.Errorf("revoke consent: %v", err)
	}

	d.logger.Info("api: revoked consent", "user_id", req.UserId, "connector_id", req.ConnectorId, "client_id", req.ClientId)
	return &api.RevokeConsentResp{}, nil
}
