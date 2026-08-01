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

	expiry := connectorExpiryFromProto(req.Connector.Expiry)
	if err := d.expiry.Validate(expiry); err != nil {
		return nil, fmt.Errorf("invalid expiry: %v", err)
	}

	c := storage.Connector{
		ID:              req.Connector.Id,
		Name:            req.Connector.Name,
		Type:            req.Connector.Type,
		ResourceVersion: "1",
		Config:          req.Connector.Config,
		GrantTypes:      req.Connector.GrantTypes,
		Expiry:          expiry,
	}
	if err := d.s.CreateConnector(ctx, c); err != nil {
		if err == storage.ErrAlreadyExists {
			return &api.CreateConnectorResp{AlreadyExists: true}, nil
		}
		d.logger.Error("api: failed to create connector", "err", err)
		return nil, fmt.Errorf("create connector: %v", err)
	}

	// Validated above, so Upsert cannot fail here.
	_ = d.expiry.Upsert(req.Connector.Id, expiry)

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
		req.NewGrantTypes != nil ||
		req.NewExpiry != nil
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

	// An absent req.NewExpiry leaves the override alone; one set without a
	// value clears it (nil newExpiry), one set with a value replaces it.
	var newExpiry *storage.ConnectorExpiry
	if req.NewExpiry != nil {
		newExpiry = connectorExpiryFromProto(req.NewExpiry.Value)
		if err := d.expiry.Validate(newExpiry); err != nil {
			return nil, fmt.Errorf("invalid expiry: %v", err)
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

		if req.NewExpiry != nil {
			old.Expiry = newExpiry
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

	if req.NewExpiry != nil {
		// Validated above, so Upsert cannot fail here.
		_ = d.expiry.Upsert(req.Id, newExpiry)
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

	// Drop any live override so a connector re-created with the same id
	// starts fresh. Upsert(_, nil) cannot error.
	_ = d.expiry.Upsert(req.Id, nil)

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
		c.Expiry = connectorExpiryToProto(connector.Expiry)
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

func connectorExpiryToProto(e *storage.ConnectorExpiry) *api.ConnectorExpiry {
	if e == nil {
		return nil
	}
	p := &api.ConnectorExpiry{IdTokens: e.IDTokens}
	if e.RefreshTokens != nil {
		p.RefreshTokens = &api.ConnectorRefreshExpiry{
			DisableRotation:   e.RefreshTokens.DisableRotation,
			ReuseInterval:     e.RefreshTokens.ReuseInterval,
			AbsoluteLifetime:  e.RefreshTokens.AbsoluteLifetime,
			ValidIfNotUsedFor: e.RefreshTokens.ValidIfNotUsedFor,
		}
	}
	return p
}

func connectorExpiryFromProto(p *api.ConnectorExpiry) *storage.ConnectorExpiry {
	if p == nil {
		return nil
	}
	e := &storage.ConnectorExpiry{IDTokens: p.IdTokens}
	if p.RefreshTokens != nil {
		e.RefreshTokens = &storage.ConnectorRefreshExpiry{
			DisableRotation:   p.RefreshTokens.DisableRotation,
			ReuseInterval:     p.RefreshTokens.ReuseInterval,
			AbsoluteLifetime:  p.RefreshTokens.AbsoluteLifetime,
			ValidIfNotUsedFor: p.RefreshTokens.ValidIfNotUsedFor,
		}
	}
	return e
}
