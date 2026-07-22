package apiserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/dexidp/dex/api/v2"
	"github.com/dexidp/dex/storage"
)

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
