package server

import (
	"errors"
	"log"

	"golang.org/x/net/context"

	"github.com/coreos/dex/api"
	"github.com/coreos/dex/storage"
)

// NewAPI returns a server which implements the gRPC API interface.
func NewAPI(s storage.Storage) api.DexServer {
	return dexAPI{s: s}
}

type dexAPI struct {
	s storage.Storage
}

func (d dexAPI) CreateClient(ctx context.Context, req *api.CreateClientReq) (*api.CreateClientResp, error) {
	if req.Client == nil {
		return nil, errors.New("no client supplied")
	}

	if req.Client.Id == "" {
		req.Client.Id = storage.NewID()
	}
	if req.Client.Secret == "" {
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
	if err := d.s.CreateClient(c); err != nil {
		log.Printf("api: failed to create client: %v", err)
		// TODO(ericchiang): Surface "already exists" errors.
		return nil, err
	}

	return &api.CreateClientResp{
		Client: req.Client,
	}, nil
}

func (d dexAPI) DeleteClient(ctx context.Context, req *api.DeleteClientReq) (*api.DeleteClientResp, error) {
	err := d.s.DeleteClient(req.Id)
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.DeleteClientResp{NotFound: true}, nil
		}
		log.Printf("api: failed to delete client: %v", err)
		return nil, err
	}
	return &api.DeleteClientResp{}, nil
}
