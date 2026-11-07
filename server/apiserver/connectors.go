package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/dexidp/dex/api/v2"
	"github.com/dexidp/dex/pkg/featureflags"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/storage"
)

func (d dexAPI) CreateConnector(ctx context.Context, req *api.CreateConnectorReq) (*api.CreateConnectorResp, error) {
	if !featureflags.APIConnectorsCRUD.Enabled() {
		return nil, fmt.Errorf("%s feature flag is not enabled", featureflags.APIConnectorsCRUD.Name)
	}

	if req.Connector == nil {
		return nil, errors.New("no connector supplied")
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
		if !connectors.ConnectorGrantTypes[gt] {
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
	if d.connectors != nil {
		d.connectors.Close(req.Connector.Id)
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
			if !connectors.ConnectorGrantTypes[gt] {
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
