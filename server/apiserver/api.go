package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/dexidp/dex/api/v2"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/discovery"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// apiVersion increases every time a new call is added to the API. Clients should use this info
// to determine if the server supports specific features.
const apiVersion = 4

// NewAPI returns a server which implements the gRPC API interface. It takes only
// the narrow dependencies it needs — the connector cache to invalidate on
// connector CRUD, and the discovery handler to serve the same document as HTTP —
// rather than the whole Server.
func NewAPI(s storage.Storage, logger *slog.Logger, version string, conns *connectors.Cache, disc *discovery.Handler) api.DexServer {
	apiLogger := logger.With("component", "api")
	return dexAPI{
		s:          s,
		logger:     apiLogger,
		version:    version,
		connectors: conns,
		discovery:  disc,
		refresh:    tokens.NewRefreshStore(s, time.Now, apiLogger),
	}
}

type dexAPI struct {
	api.UnimplementedDexServer

	s          storage.Storage
	logger     *slog.Logger
	version    string
	connectors *connectors.Cache
	discovery  *discovery.Handler
	refresh    *tokens.RefreshStore
}

func (d dexAPI) GetVersion(ctx context.Context, req *api.VersionReq) (*api.VersionResp, error) {
	return &api.VersionResp{
		Server: d.version,
		Api:    apiVersion,
	}, nil
}

func (d dexAPI) GetDiscovery(ctx context.Context, req *api.DiscoveryReq) (*api.DiscoveryResp, error) {
	if d.discovery == nil {
		return nil, fmt.Errorf("discovery is not configured")
	}
	discoveryDoc := d.discovery.Construct(ctx)
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

// unixOrZero returns the Unix timestamp for t, or 0 when t is the zero value.
// A naive t.Unix() on a zero time.Time yields -62135596800 (a year-1 epoch),
// which is a misleading value to expose through the API; callers want 0/unset.
func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}
