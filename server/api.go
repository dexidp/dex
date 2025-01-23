package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"golang.org/x/crypto/bcrypt"

	"github.com/dexidp/dex/api/v2"
	"github.com/dexidp/dex/pkg/featureflags"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

// apiVersion increases every time a new call is added to the API. Clients should use this info
// to determine if the server supports specific features.
const apiVersion = 2

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
			Id:           c.ID,
			Name:         c.Name,
			Secret:       c.Secret,
			RedirectUris: c.RedirectURIs,
			TrustedPeers: c.TrustedPeers,
			Public:       c.Public,
			LogoUrl:      c.LogoURL,
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
		ID:           req.Client.Id,
		Secret:       req.Client.Secret,
		RedirectURIs: req.Client.RedirectUris,
		TrustedPeers: req.Client.TrustedPeers,
		Public:       req.Client.Public,
		Name:         req.Client.Name,
		LogoURL:      req.Client.LogoUrl,
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
	discoveryDoc := d.server.constructDiscovery()
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

	c := storage.Connector{
		ID:              req.Connector.Id,
		Name:            req.Connector.Name,
		Type:            req.Connector.Type,
		ResourceVersion: "1",
		Config:          req.Connector.Config,
	}
	if err := d.s.CreateConnector(ctx, c); err != nil {
		if err == storage.ErrAlreadyExists {
			return &api.CreateConnectorResp{AlreadyExists: true}, nil
		}
		d.logger.Error("api: failed to create connector", "err", err)
		return nil, fmt.Errorf("create connector: %v", err)
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

	if len(req.NewConfig) == 0 && req.NewName == "" && req.NewType == "" {
		return nil, errors.New("nothing to update")
	}

	if !json.Valid(req.NewConfig) {
		return nil, errors.New("invalid config supplied")
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
			Id:     connector.ID,
			Name:   connector.Name,
			Type:   connector.Type,
			Config: connector.Config,
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
