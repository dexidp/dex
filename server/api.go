package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/dexidp/dex/api/v2"
	"github.com/dexidp/dex/pkg/featureflags"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

// apiVersion increases every time a new call is added to the API. Clients should use this info
// to determine if the server supports specific features.
const apiVersion = 4

const (
	// recCost is the recommended bcrypt cost, which balances hash strength and
	// efficiency.
	recCost = 12

	// upBoundCost is a sane upper bound on bcrypt cost determined by benchmarking:
	// high enough to ensure secure encryption, low enough to not put unnecessary
	// load on a dex server.
	upBoundCost = 16
)

// NewAPI returns a server which implements the gRPC API interface.
func NewAPI(s storage.Storage, logger *slog.Logger, version string, server *Server) api.DexServer {
	return dexAPI{
		s:       s,
		logger:  logger.With("component", "api"),
		version: version,
		server:  server,
	}
}

type dexAPI struct {
	api.UnimplementedDexServer

	s       storage.Storage
	logger  *slog.Logger
	version string
	server  *Server
}

func (d dexAPI) GetClient(ctx context.Context, req *api.GetClientReq) (*api.GetClientResp, error) {
	c, err := d.s.GetClient(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &api.GetClientResp{
		Client: &api.Client{
			Id:                c.ID,
			Name:              c.Name,
			Secret:            c.Secret,
			RedirectUris:      c.RedirectURIs,
			TrustedPeers:      c.TrustedPeers,
			Public:            c.Public,
			LogoUrl:           c.LogoURL,
			AllowedConnectors: c.AllowedConnectors,
			SsoSharedWith:     c.SSOSharedWith,
		},
	}, nil
}

func (d dexAPI) CreateClient(ctx context.Context, req *api.CreateClientReq) (*api.CreateClientResp, error) {
	if req.Client == nil {
		return nil, errors.New("no client supplied")
	}

	if req.Client.Id == "" {
		req.Client.Id = storage.NewID()
	}
	if req.Client.Secret == "" && !req.Client.Public {
		req.Client.Secret = storage.NewID() + storage.NewID()
	}

	c := storage.Client{
		ID:                req.Client.Id,
		Secret:            req.Client.Secret,
		RedirectURIs:      req.Client.RedirectUris,
		TrustedPeers:      req.Client.TrustedPeers,
		Public:            req.Client.Public,
		Name:              req.Client.Name,
		LogoURL:           req.Client.LogoUrl,
		AllowedConnectors: req.Client.AllowedConnectors,
		SSOSharedWith:     req.Client.SsoSharedWith,
	}
	if err := d.s.CreateClient(ctx, c); err != nil {
		if err == storage.ErrAlreadyExists {
			return &api.CreateClientResp{AlreadyExists: true}, nil
		}
		d.logger.Error("failed to create client", "err", err)
		return nil, fmt.Errorf("create client: %v", err)
	}

	return &api.CreateClientResp{
		Client: req.Client,
	}, nil
}

func (d dexAPI) UpdateClient(ctx context.Context, req *api.UpdateClientReq) (*api.UpdateClientResp, error) {
	if req.Id == "" {
		return nil, errors.New("update client: no client ID supplied")
	}

	err := d.s.UpdateClient(ctx, req.Id, func(old storage.Client) (storage.Client, error) {
		if req.RedirectUris != nil {
			old.RedirectURIs = req.RedirectUris
		}
		if req.TrustedPeers != nil {
			old.TrustedPeers = req.TrustedPeers
		}
		if req.Name != "" {
			old.Name = req.Name
		}
		if req.LogoUrl != "" {
			old.LogoURL = req.LogoUrl
		}
		if req.AllowedConnectors != nil {
			old.AllowedConnectors = req.AllowedConnectors
		}
		if req.SsoSharedWith != nil {
			old.SSOSharedWith = req.SsoSharedWith
		}
		return old, nil
	})
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.UpdateClientResp{NotFound: true}, nil
		}
		d.logger.Error("failed to update the client", "err", err)
		return nil, fmt.Errorf("update client: %v", err)
	}
	return &api.UpdateClientResp{}, nil
}

func (d dexAPI) DeleteClient(ctx context.Context, req *api.DeleteClientReq) (*api.DeleteClientResp, error) {
	err := d.s.DeleteClient(ctx, req.Id)
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.DeleteClientResp{NotFound: true}, nil
		}
		d.logger.Error("failed to delete client", "err", err)
		return nil, fmt.Errorf("delete client: %v", err)
	}
	return &api.DeleteClientResp{}, nil
}

func (d dexAPI) ListClients(ctx context.Context, req *api.ListClientReq) (*api.ListClientResp, error) {
	clientList, err := d.s.ListClients(ctx)
	if err != nil {
		d.logger.Error("failed to list clients", "err", err)
		return nil, fmt.Errorf("list clients: %v", err)
	}

	clients := make([]*api.ClientInfo, 0, len(clientList))
	for _, client := range clientList {
		c := api.ClientInfo{
			Id:                client.ID,
			Name:              client.Name,
			RedirectUris:      client.RedirectURIs,
			TrustedPeers:      client.TrustedPeers,
			Public:            client.Public,
			LogoUrl:           client.LogoURL,
			AllowedConnectors: client.AllowedConnectors,
			SsoSharedWith:     client.SSOSharedWith,
		}
		clients = append(clients, &c)
	}

	return &api.ListClientResp{
		Clients: clients,
	}, nil
}

// checkCost returns an error if the hash provided does not meet lower or upper
// bound cost requirements.
func checkCost(hash []byte) error {
	actual, err := bcrypt.Cost(hash)
	if err != nil {
		return fmt.Errorf("parsing bcrypt hash: %v", err)
	}
	if actual < bcrypt.DefaultCost {
		return fmt.Errorf("given hash cost = %d does not meet minimum cost requirement = %d", actual, bcrypt.DefaultCost)
	}
	if actual > upBoundCost {
		return fmt.Errorf("given hash cost = %d is above upper bound cost = %d, recommended cost = %d", actual, upBoundCost, recCost)
	}
	return nil
}

func (d dexAPI) CreatePassword(ctx context.Context, req *api.CreatePasswordReq) (*api.CreatePasswordResp, error) {
	if req.Password == nil {
		return nil, errors.New("no password supplied")
	}
	if req.Password.UserId == "" {
		return nil, errors.New("no user ID supplied")
	}
	if req.Password.Hash != nil {
		if err := checkCost(req.Password.Hash); err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("no hash of password supplied")
	}

	p := storage.Password{
		Email:    req.Password.Email,
		Hash:     req.Password.Hash,
		Username: req.Password.Username,
		UserID:   req.Password.UserId,
	}
	if err := d.s.CreatePassword(ctx, p); err != nil {
		if err == storage.ErrAlreadyExists {
			return &api.CreatePasswordResp{AlreadyExists: true}, nil
		}
		d.logger.Error("failed to create password", "err", err)
		return nil, fmt.Errorf("create password: %v", err)
	}

	return &api.CreatePasswordResp{}, nil
}

func (d dexAPI) UpdatePassword(ctx context.Context, req *api.UpdatePasswordReq) (*api.UpdatePasswordResp, error) {
	if req.Email == "" {
		return nil, errors.New("no email supplied")
	}
	if req.NewHash == nil && req.NewUsername == "" {
		return nil, errors.New("nothing to update")
	}

	if req.NewHash != nil {
		if err := checkCost(req.NewHash); err != nil {
			return nil, err
		}
	}

	updater := func(old storage.Password) (storage.Password, error) {
		if req.NewHash != nil {
			old.Hash = req.NewHash
		}

		if req.NewUsername != "" {
			old.Username = req.NewUsername
		}

		return old, nil
	}

	if err := d.s.UpdatePassword(ctx, req.Email, updater); err != nil {
		if err == storage.ErrNotFound {
			return &api.UpdatePasswordResp{NotFound: true}, nil
		}
		d.logger.Error("failed to update password", "err", err)
		return nil, fmt.Errorf("update password: %v", err)
	}

	return &api.UpdatePasswordResp{}, nil
}

func (d dexAPI) DeletePassword(ctx context.Context, req *api.DeletePasswordReq) (*api.DeletePasswordResp, error) {
	if req.Email == "" {
		return nil, errors.New("no email supplied")
	}

	err := d.s.DeletePassword(ctx, req.Email)
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.DeletePasswordResp{NotFound: true}, nil
		}
		d.logger.Error("failed to delete password", "err", err)
		return nil, fmt.Errorf("delete password: %v", err)
	}
	return &api.DeletePasswordResp{}, nil
}

func (d dexAPI) GetVersion(ctx context.Context, req *api.VersionReq) (*api.VersionResp, error) {
	return &api.VersionResp{
		Server: d.version,
		Api:    apiVersion,
	}, nil
}

func (d dexAPI) GetDiscovery(ctx context.Context, req *api.DiscoveryReq) (*api.DiscoveryResp, error) {
	discoveryDoc := d.server.constructDiscovery(ctx)
	data, err := json.Marshal(discoveryDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal discovery data: %v", err)
	}
	resp := api.DiscoveryResp{}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal discovery data: %v", err)
	}
	return &resp, nil
}

func (d dexAPI) ListPasswords(ctx context.Context, req *api.ListPasswordReq) (*api.ListPasswordResp, error) {
	passwordList, err := d.s.ListPasswords(ctx)
	if err != nil {
		d.logger.Error("failed to list passwords", "err", err)
		return nil, fmt.Errorf("list passwords: %v", err)
	}

	passwords := make([]*api.Password, 0, len(passwordList))
	for _, password := range passwordList {
		p := api.Password{
			Email:    password.Email,
			Username: password.Username,
			UserId:   password.UserID,
		}
		passwords = append(passwords, &p)
	}

	return &api.ListPasswordResp{
		Passwords: passwords,
	}, nil
}

func (d dexAPI) VerifyPassword(ctx context.Context, req *api.VerifyPasswordReq) (*api.VerifyPasswordResp, error) {
	if req.Email == "" {
		return nil, errors.New("no email supplied")
	}

	if req.Password == "" {
		return nil, errors.New("no password to verify supplied")
	}

	password, err := d.s.GetPassword(ctx, req.Email)
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.VerifyPasswordResp{
				NotFound: true,
			}, nil
		}
		d.logger.Error("there was an error retrieving the password", "err", err)
		return nil, fmt.Errorf("verify password: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword(password.Hash, []byte(req.Password)); err != nil {
		d.logger.Info("password check failed", "err", err)
		return &api.VerifyPasswordResp{
			Verified: false,
		}, nil
	}
	return &api.VerifyPasswordResp{
		Verified: true,
	}, nil
}

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

func (d dexAPI) CreateConnector(ctx context.Context, req *api.CreateConnectorReq) (*api.CreateConnectorResp, error) {
	if !featureflags.APIConnectorsCRUD.Enabled() {
		return nil, fmt.Errorf("%s feature flag is not enabled", featureflags.APIConnectorsCRUD.Name)
	}

	if req.Connector.Id == "" {
		return nil, errors.New("no id supplied")
	}

	if req.Connector.Type == "" {
		return nil, errors.New("no type supplied")
	}

	if req.Connector.Name == "" {
		return nil, errors.New("no name supplied")
	}

	if len(req.Connector.Config) == 0 {
		return nil, errors.New("no config supplied")
	}

	if !json.Valid(req.Connector.Config) {
		return nil, errors.New("invalid config supplied")
	}

	for _, gt := range req.Connector.GrantTypes {
		if !ConnectorGrantTypes[gt] {
			return nil, fmt.Errorf("unknown grant type %q", gt)
		}
	}

	c := storage.Connector{
		ID:              req.Connector.Id,
		Name:            req.Connector.Name,
		Type:            req.Connector.Type,
		ResourceVersion: "1",
		Config:          req.Connector.Config,
		GrantTypes:      req.Connector.GrantTypes,
	}
	if err := d.s.CreateConnector(ctx, c); err != nil {
		if err == storage.ErrAlreadyExists {
			return &api.CreateConnectorResp{AlreadyExists: true}, nil
		}
		d.logger.Error("api: failed to create connector", "err", err)
		return nil, fmt.Errorf("create connector: %v", err)
	}

	// Make sure we don't reuse stale entries in the cache
	if d.server != nil {
		d.server.CloseConnector(req.Connector.Id)
	}

	return &api.CreateConnectorResp{}, nil
}

func (d dexAPI) UpdateConnector(ctx context.Context, req *api.UpdateConnectorReq) (*api.UpdateConnectorResp, error) {
	if !featureflags.APIConnectorsCRUD.Enabled() {
		return nil, fmt.Errorf("%s feature flag is not enabled", featureflags.APIConnectorsCRUD.Name)
	}

	if req.Id == "" {
		return nil, errors.New("no email supplied")
	}

	hasUpdate := len(req.NewConfig) != 0 ||
		req.NewName != "" ||
		req.NewType != "" ||
		req.NewGrantTypes != nil
	if !hasUpdate {
		return nil, errors.New("nothing to update")
	}

	if len(req.NewConfig) != 0 && !json.Valid(req.NewConfig) {
		return nil, errors.New("invalid config supplied")
	}

	if req.NewGrantTypes != nil {
		for _, gt := range req.NewGrantTypes.GrantTypes {
			if !ConnectorGrantTypes[gt] {
				return nil, fmt.Errorf("unknown grant type %q", gt)
			}
		}
	}

	updater := func(old storage.Connector) (storage.Connector, error) {
		if req.NewType != "" {
			old.Type = req.NewType
		}

		if req.NewName != "" {
			old.Name = req.NewName
		}

		if len(req.NewConfig) != 0 {
			old.Config = req.NewConfig
		}

		if req.NewGrantTypes != nil {
			old.GrantTypes = req.NewGrantTypes.GrantTypes
		}

		if rev, err := strconv.Atoi(defaultTo(old.ResourceVersion, "0")); err == nil {
			old.ResourceVersion = strconv.Itoa(rev + 1)
		}

		return old, nil
	}

	if err := d.s.UpdateConnector(ctx, req.Id, updater); err != nil {
		if err == storage.ErrNotFound {
			return &api.UpdateConnectorResp{NotFound: true}, nil
		}
		d.logger.Error("api: failed to update connector", "err", err)
		return nil, fmt.Errorf("update connector: %v", err)
	}

	return &api.UpdateConnectorResp{}, nil
}

func (d dexAPI) DeleteConnector(ctx context.Context, req *api.DeleteConnectorReq) (*api.DeleteConnectorResp, error) {
	if !featureflags.APIConnectorsCRUD.Enabled() {
		return nil, fmt.Errorf("%s feature flag is not enabled", featureflags.APIConnectorsCRUD.Name)
	}

	if req.Id == "" {
		return nil, errors.New("no id supplied")
	}

	err := d.s.DeleteConnector(ctx, req.Id)
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.DeleteConnectorResp{NotFound: true}, nil
		}
		d.logger.Error("api: failed to delete connector", "err", err)
		return nil, fmt.Errorf("delete connector: %v", err)
	}

	return &api.DeleteConnectorResp{}, nil
}

func (d dexAPI) ListConnectors(ctx context.Context, req *api.ListConnectorReq) (*api.ListConnectorResp, error) {
	if !featureflags.APIConnectorsCRUD.Enabled() {
		return nil, fmt.Errorf("%s feature flag is not enabled", featureflags.APIConnectorsCRUD.Name)
	}

	connectorList, err := d.s.ListConnectors(ctx)
	if err != nil {
		d.logger.Error("api: failed to list connectors", "err", err)
		return nil, fmt.Errorf("list connectors: %v", err)
	}

	connectors := make([]*api.Connector, 0, len(connectorList))
	for _, connector := range connectorList {
		c := api.Connector{
			Id:         connector.ID,
			Name:       connector.Name,
			Type:       connector.Type,
			Config:     connector.Config,
			GrantTypes: connector.GrantTypes,
		}
		connectors = append(connectors, &c)
	}

	return &api.ListConnectorResp{
		Connectors: connectors,
	}, nil
}

func defaultTo[T comparable](v, def T) T {
	var zeroT T
	if v == zeroT {
		return def
	}
	return v
}

// revokeUserRefreshTokens revokes all refresh tokens for a user/connector pair
// and cleans up offline session references. Errors are logged but not returned
// (best-effort). Uses the shared revokeRefreshTokensFromStorage helper.
func (d dexAPI) revokeUserRefreshTokens(ctx context.Context, userID, connectorID string) {
	revokeRefreshTokensFromStorage(ctx, d.s, d.logger, userID, connectorID)
}

// unixOrZero returns the Unix timestamp for t, or 0 when t is the zero value.
// A naive t.Unix() on a zero time.Time yields -62135596800 (a year-1 epoch),
// which is a misleading value to expose through the API; callers want 0/unset.
func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

// errIdentityUnchanged signals that an UpdateUserIdentity callback found nothing
// to change. Returning it aborts the storage write (avoiding a needless version
// bump and optimistic-concurrency conflict) and is treated as "not found" by the
// caller.
var errIdentityUnchanged = errors.New("identity unchanged")

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
