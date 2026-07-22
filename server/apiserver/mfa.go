package apiserver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/dexidp/dex/api/v2"
	"github.com/dexidp/dex/pkg/featureflags"
	"github.com/dexidp/dex/storage"
)

// errIdentityUnchanged signals that an UpdateUserIdentity callback found nothing
// to change, so the mutation (and its resource-version bump) is skipped.
var errIdentityUnchanged = errors.New("identity unchanged")

func storageMFADevicesToAPI(secrets map[string]*storage.MFASecret, credentials map[string][]storage.WebAuthnCredential) []*api.MFADeviceInfo {
	// Collect all authenticator IDs from both maps.
	authIDs := make(map[string]struct{})
	for id := range secrets {
		authIDs[id] = struct{}{}
	}
	for id := range credentials {
		authIDs[id] = struct{}{}
	}

	devices := make([]*api.MFADeviceInfo, 0, len(authIDs))
	for authID := range authIDs {
		device := &api.MFADeviceInfo{
			AuthenticatorId: authID,
		}

		if secret, ok := secrets[authID]; ok {
			device.MfaSecret = &api.MFASecret{
				AuthenticatorId: secret.AuthenticatorID,
				Type:            secret.Type,
				Confirmed:       secret.Confirmed,
				CreatedAt:       unixOrZero(secret.CreatedAt),
			}
		}

		if creds, ok := credentials[authID]; ok {
			apiCreds := make([]*api.WebAuthnCredential, 0, len(creds))
			for _, c := range creds {
				apiCreds = append(apiCreds, &api.WebAuthnCredential{
					CredentialId:    c.CredentialID,
					AttestationType: c.AttestationType,
					Aaguid:          c.AAGUID,
					SignCount:       c.SignCount,
					CloneWarning:    c.CloneWarning,
					Transport:       c.Transport,
					BackupEligible:  c.BackupEligible,
					BackupState:     c.BackupState,
					DisplayName:     c.DisplayName,
					CreatedAt:       unixOrZero(c.CreatedAt),
				})
			}
			device.WebauthnCredentials = apiCreds
		}

		devices = append(devices, device)
	}
	return devices
}

func (d dexAPI) ResetMFA(ctx context.Context, req *api.ResetMFAReq) (*api.ResetMFAResp, error) {
	if !featureflags.APISessionsIdentitiesCRUD.Enabled() {
		return nil, fmt.Errorf("%s feature flag is not enabled", featureflags.APISessionsIdentitiesCRUD.Name)
	}

	if req.UserId == "" {
		return nil, errors.New("no user_id supplied")
	}
	if req.ConnectorId == "" {
		return nil, errors.New("no connector_id supplied")
	}

	if err := d.s.UpdateUserIdentity(ctx, req.UserId, req.ConnectorId, func(old storage.UserIdentity) (storage.UserIdentity, error) {
		old.MFASecrets = nil
		old.WebAuthnCredentials = nil
		return old, nil
	}); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return &api.ResetMFAResp{NotFound: true}, nil
		}
		d.logger.Error("api: failed to reset MFA", "err", err)
		return nil, fmt.Errorf("reset MFA: %v", err)
	}

	d.logger.Info("api: reset MFA", "user_id", req.UserId, "connector_id", req.ConnectorId)
	return &api.ResetMFAResp{}, nil
}

func (d dexAPI) ListMFADevices(ctx context.Context, req *api.ListMFADevicesReq) (*api.ListMFADevicesResp, error) {
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
		d.logger.Error("api: failed to get user identity for MFA devices", "err", err)
		return nil, fmt.Errorf("list MFA devices: %v", err)
	}

	return &api.ListMFADevicesResp{
		Devices: storageMFADevicesToAPI(identity.MFASecrets, identity.WebAuthnCredentials),
	}, nil
}

func (d dexAPI) DeleteWebAuthnCredential(ctx context.Context, req *api.DeleteWebAuthnCredentialReq) (*api.DeleteWebAuthnCredentialResp, error) {
	if !featureflags.APISessionsIdentitiesCRUD.Enabled() {
		return nil, fmt.Errorf("%s feature flag is not enabled", featureflags.APISessionsIdentitiesCRUD.Name)
	}

	if req.UserId == "" {
		return nil, errors.New("no user_id supplied")
	}
	if req.ConnectorId == "" {
		return nil, errors.New("no connector_id supplied")
	}
	if len(req.CredentialId) == 0 {
		return nil, errors.New("no credential_id supplied")
	}

	if err := d.s.UpdateUserIdentity(ctx, req.UserId, req.ConnectorId, func(old storage.UserIdentity) (storage.UserIdentity, error) {
		for authID, creds := range old.WebAuthnCredentials {
			for i, cred := range creds {
				if bytes.Equal(cred.CredentialID, req.CredentialId) {
					old.WebAuthnCredentials[authID] = slices.Delete(creds, i, i+1)
					if len(old.WebAuthnCredentials[authID]) == 0 {
						delete(old.WebAuthnCredentials, authID)
					}
					return old, nil
				}
			}
		}
		return old, errIdentityUnchanged
	}); err != nil {
		if errors.Is(err, errIdentityUnchanged) || errors.Is(err, storage.ErrNotFound) {
			return &api.DeleteWebAuthnCredentialResp{NotFound: true}, nil
		}
		d.logger.Error("api: failed to delete WebAuthn credential", "err", err)
		return nil, fmt.Errorf("delete WebAuthn credential: %v", err)
	}

	d.logger.Info("api: deleted WebAuthn credential", "user_id", req.UserId, "connector_id", req.ConnectorId)
	return &api.DeleteWebAuthnCredentialResp{}, nil
}

func (d dexAPI) DeleteMFASecret(ctx context.Context, req *api.DeleteMFASecretReq) (*api.DeleteMFASecretResp, error) {
	if !featureflags.APISessionsIdentitiesCRUD.Enabled() {
		return nil, fmt.Errorf("%s feature flag is not enabled", featureflags.APISessionsIdentitiesCRUD.Name)
	}

	if req.UserId == "" {
		return nil, errors.New("no user_id supplied")
	}
	if req.ConnectorId == "" {
		return nil, errors.New("no connector_id supplied")
	}
	if req.AuthenticatorId == "" {
		return nil, errors.New("no authenticator_id supplied")
	}

	if err := d.s.UpdateUserIdentity(ctx, req.UserId, req.ConnectorId, func(old storage.UserIdentity) (storage.UserIdentity, error) {
		if _, ok := old.MFASecrets[req.AuthenticatorId]; !ok {
			return old, errIdentityUnchanged
		}
		delete(old.MFASecrets, req.AuthenticatorId)
		// Also remove associated WebAuthn credentials for the same authenticator.
		delete(old.WebAuthnCredentials, req.AuthenticatorId)
		return old, nil
	}); err != nil {
		if errors.Is(err, errIdentityUnchanged) || errors.Is(err, storage.ErrNotFound) {
			return &api.DeleteMFASecretResp{NotFound: true}, nil
		}
		d.logger.Error("api: failed to delete MFA secret", "err", err)
		return nil, fmt.Errorf("delete MFA secret: %v", err)
	}

	d.logger.Info("api: deleted MFA secret", "user_id", req.UserId, "connector_id", req.ConnectorId)
	return &api.DeleteMFASecretResp{}, nil
}
