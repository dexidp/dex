package main

import (
	"fmt"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/db"
	_ "github.com/coreos/dex/db/postgresql"
	"github.com/coreos/go-oidc/oidc"
)

func newDBDriver(storage, dsn string) (driver, error) {
	rd := db.GetDriver(storage)
	if rd == nil {
		return nil, fmt.Errorf("Storage driver not found")
	}
	dbc, err := rd.NewWithMap(map[string]interface{}{"url": dsn})
	if err != nil {
		return nil, err
	}

	drv := &dbDriver{
		ciRepo:  dbc.NewClientIdentityRepo(),
		cfgRepo: dbc.NewConnectorConfigRepo(),
	}

	return drv, nil
}

type dbDriver struct {
	ciRepo  client.ClientIdentityRepo
	cfgRepo connector.ConnectorConfigRepo
}

func (d *dbDriver) NewClient(meta oidc.ClientMetadata) (*oidc.ClientCredentials, error) {
	if err := meta.Valid(); err != nil {
		return nil, err
	}

	clientID, err := oidc.GenClientID(meta.RedirectURLs[0].Host)
	if err != nil {
		return nil, err
	}

	return d.ciRepo.New(clientID, meta)
}

func (d *dbDriver) ConnectorConfigs() ([]connector.ConnectorConfig, error) {
	return d.cfgRepo.All()
}

func (d *dbDriver) SetConnectorConfigs(cfgs []connector.ConnectorConfig) error {
	return d.cfgRepo.Set(cfgs)
}
