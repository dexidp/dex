package api

import (
	"errors"

	"golang.org/x/net/context"

	"github.com/coreos/poke/api/apipb"
	"github.com/coreos/poke/storage"
)

// NewServer returns a gRPC server for talking to a storage.
func NewServer(s storage.Storage) apipb.StorageServer {
	return &server{s}
}

type server struct {
	storage storage.Storage
}

func fromPBClient(client *apipb.Client) storage.Client {
	return storage.Client{
		ID:           client.Id,
		Secret:       client.Secret,
		RedirectURIs: client.RedirectUris,
		TrustedPeers: client.TrustedPeers,
		Public:       client.Public,
		Name:         client.Name,
		LogoURL:      client.LogoUrl,
	}
}

func toPBClient(client storage.Client) *apipb.Client {
	return &apipb.Client{
		Id:           client.ID,
		Secret:       client.Secret,
		RedirectUris: client.RedirectURIs,
		TrustedPeers: client.TrustedPeers,
		Public:       client.Public,
		Name:         client.Name,
		LogoUrl:      client.LogoURL,
	}
}

func (s *server) CreateClient(ctx context.Context, req *apipb.CreateClientReq) (*apipb.CreateClientResp, error) {
	// TODO(ericchiang): Create a more centralized strategy for creating client IDs
	// and secrets which are restricted based on the storage.
	client := fromPBClient(req.Client)
	if client.ID == "" {
		client.ID = storage.NewNonce()
	}
	if client.Secret == "" {
		client.Secret = storage.NewNonce() + storage.NewNonce()
	}

	if err := s.storage.CreateClient(client); err != nil {
		return nil, err
	}
	return &apipb.CreateClientResp{Client: toPBClient(client)}, nil
}

func (s *server) UpdateClient(ctx context.Context, req *apipb.UpdateClientReq) (*apipb.UpdateClientResp, error) {
	switch {
	case req.Id == "":
		return nil, errors.New("no ID supplied")
	case req.MakePublic && req.MakePrivate:
		return nil, errors.New("cannot both make public and private")
	case req.MakePublic && len(req.RedirectUris) != 0:
		return nil, errors.New("redirect uris supplied for a public client")
	}

	var client *storage.Client
	updater := func(old storage.Client) (storage.Client, error) {
		if req.MakePublic {
			old.Public = true
		}
		if req.MakePrivate {
			old.Public = false
		}
		if req.Secret != "" {
			old.Secret = req.Secret
		}
		if req.Name != "" {
			old.Name = req.Name
		}
		if req.LogoUrl != "" {
			old.LogoURL = req.LogoUrl
		}
		if len(req.RedirectUris) != 0 {
			if old.Public {
				return old, errors.New("public clients cannot have redirect URIs")
			}
			old.RedirectURIs = req.RedirectUris
		}
		client = &old
		return old, nil
	}

	if err := s.storage.UpdateClient(req.Id, updater); err != nil {
		return nil, err
	}
	return &apipb.UpdateClientResp{Client: toPBClient(*client)}, nil
}

func (s *server) DeleteClient(ctx context.Context, req *apipb.DeleteClientReq) (*apipb.DeleteClientReq, error) {
	if req.Id == "" {
		return nil, errors.New("no client ID supplied")
	}
	if err := s.storage.DeleteClient(req.Id); err != nil {
		return nil, err
	}
	return &apipb.DeleteClientReq{}, nil
}

func (s *server) ListClients(ctx context.Context, req *apipb.ListClientsReq) (*apipb.ListClientsResp, error) {
	clients, err := s.storage.ListClients()
	if err != nil {
		return nil, err
	}
	resp := make([]*apipb.Client, len(clients))
	for i, client := range clients {
		resp[i] = toPBClient(client)
	}
	return &apipb.ListClientsResp{Clients: resp}, nil
}

func (s *server) GetClient(ctx context.Context, req *apipb.GetClientReq) (*apipb.GetClientResp, error) {
	if req.Id == "" {
		return nil, errors.New("no client ID supplied")
	}
	client, err := s.storage.GetClient(req.Id)
	if err != nil {
		return nil, err
	}
	return &apipb.GetClientResp{Client: toPBClient(client)}, nil
}
