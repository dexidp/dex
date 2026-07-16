package apiserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/dexidp/dex/api/v2"
	"github.com/dexidp/dex/pkg/featureflags"
	"github.com/dexidp/dex/storage"
)

func storageUserIdentityToAPI(u storage.UserIdentity) *api.UserIdentity {
	consents := make([]*api.ConsentEntry, 0, len(u.Consents))
	for clientID, scopes := range u.Consents {
		consents = append(consents, &api.ConsentEntry{
			ClientId: clientID,
			Scopes:   scopes,
		})
	}

	identity := &api.UserIdentity{
		UserId:        u.UserID,
		ConnectorId:   u.ConnectorID,
		Email:         u.Claims.Email,
		EmailVerified: u.Claims.EmailVerified,
		Username:      u.Claims.Username,
		Groups:        u.Claims.Groups,
		Consents:      consents,
		MfaDevices:    storageMFADevicesToAPI(u.MFASecrets, u.WebAuthnCredentials),
		CreatedAt:     unixOrZero(u.CreatedAt),
		LastLogin:     unixOrZero(u.LastLogin),
		BlockedUntil:  unixOrZero(u.BlockedUntil),
	}

	return identity
}

func (d dexAPI) GetUserIdentity(ctx context.Context, req *api.GetUserIdentityReq) (*api.GetUserIdentityResp, error) {
	if !featureflags.APISessionsIdentitiesCRUD.Enabled() {
		return nil, fmt.Errorf("%s feature flag is not enabled", featureflags.APISessionsIdentitiesCRUD.Name)
	}

	if req.UserId == "" {
		return nil, errors.New("no user_id supplied")
	}
	if req.ConnectorId == "" {
		return nil, errors.New("no connector_id supplied")
	}

	identity, err := d.s.GetUserIdentity(ctx, req.UserId, req.ConnectorId)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, storage.ErrNotFound
		}
		d.logger.Error("api: failed to get user identity", "err", err)
		return nil, fmt.Errorf("get user identity: %v", err)
	}

	return &api.GetUserIdentityResp{
		Identity: storageUserIdentityToAPI(identity),
	}, nil
}

func (d dexAPI) ListUserIdentities(ctx context.Context, req *api.ListUserIdentitiesReq) (*api.ListUserIdentitiesResp, error) {
	if !featureflags.APISessionsIdentitiesCRUD.Enabled() {
		return nil, fmt.Errorf("%s feature flag is not enabled", featureflags.APISessionsIdentitiesCRUD.Name)
	}

	identityList, err := d.s.ListUserIdentities(ctx)
	if err != nil {
		d.logger.Error("api: failed to list user identities", "err", err)
		return nil, fmt.Errorf("list user identities: %v", err)
	}

	identities := make([]*api.UserIdentity, 0, len(identityList))
	for _, u := range identityList {
		identities = append(identities, storageUserIdentityToAPI(u))
	}

	return &api.ListUserIdentitiesResp{
		Identities: identities,
	}, nil
}

func (d dexAPI) DeleteUserIdentity(ctx context.Context, req *api.DeleteUserIdentityReq) (*api.DeleteUserIdentityResp, error) {
	if !featureflags.APISessionsIdentitiesCRUD.Enabled() {
		return nil, fmt.Errorf("%s feature flag is not enabled", featureflags.APISessionsIdentitiesCRUD.Name)
	}

	if req.UserId == "" {
		return nil, errors.New("no user_id supplied")
	}
	if req.ConnectorId == "" {
		return nil, errors.New("no connector_id supplied")
	}

	// Look up the identity first: report not-found cleanly without performing any
	// cascade, and capture the email needed to purge the linked password record.
	identity, err := d.s.GetUserIdentity(ctx, req.UserId, req.ConnectorId)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return &api.DeleteUserIdentityResp{NotFound: true}, nil
		}
		d.logger.Error("api: failed to get user identity during purge", "err", err)
		return nil, fmt.Errorf("delete user identity: %v", err)
	}

	// Cascade deletes. A real (non-not-found) failure aborts the purge and returns
	// an error so the caller is never told a GDPR purge succeeded while data was
	// left behind.

	// Cascade: delete auth session.
	if err := d.s.DeleteAuthSession(ctx, req.UserId, req.ConnectorId); err != nil && !errors.Is(err, storage.ErrNotFound) {
		d.logger.Error("api: failed to delete auth session during identity purge", "err", err)
		return nil, fmt.Errorf("purge auth session: %v", err)
	}

	// Cascade: revoke all refresh tokens (best-effort, consistent with logout flow).
	d.revokeUserRefreshTokens(ctx, req.UserId, req.ConnectorId)

	// Cascade: delete offline sessions.
	if err := d.s.DeleteOfflineSessions(ctx, req.UserId, req.ConnectorId); err != nil && !errors.Is(err, storage.ErrNotFound) {
		d.logger.Error("api: failed to delete offline sessions during identity purge", "err", err)
		return nil, fmt.Errorf("purge offline sessions: %v", err)
	}

	// Cascade: delete the password record (keyed by email, may not exist for
	// non-password connectors).
	if email := identity.Claims.Email; email != "" {
		if err := d.s.DeletePassword(ctx, email); err != nil && !errors.Is(err, storage.ErrNotFound) {
			d.logger.Error("api: failed to delete password during identity purge", "err", err)
			return nil, fmt.Errorf("purge password: %v", err)
		}
	}

	// Delete the user identity itself.
	if err := d.s.DeleteUserIdentity(ctx, req.UserId, req.ConnectorId); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return &api.DeleteUserIdentityResp{NotFound: true}, nil
		}
		d.logger.Error("api: failed to delete user identity", "err", err)
		return nil, fmt.Errorf("delete user identity: %v", err)
	}

	d.logger.Info("api: purged user identity", "user_id", req.UserId, "connector_id", req.ConnectorId)
	return &api.DeleteUserIdentityResp{}, nil
}
