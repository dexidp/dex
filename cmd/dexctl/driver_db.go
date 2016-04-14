package main

import (
	"github.com/coreos/dex/client"
	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/db"
	"github.com/coreos/go-oidc/oidc"
)

func newDBDriver(dsn string) (driver, error) {
	dbc, err := db.NewConnection(db.Config{DSN: dsn})
	if err != nil {
		return nil, err
	}

	drv := &dbDriver{
		ciRepo:  db.NewClientRepo(dbc),
		cfgRepo: db.NewConnectorConfigRepo(dbc),
	}

	return drv, nil
}

type dbDriver struct {
	ciRepo  client.ClientRepo
	cfgRepo *db.ConnectorConfigRepo
}

func (d *dbDriver) NewClient(meta oidc.ClientMetadata) (*oidc.ClientCredentials, error) {
	if err := meta.Valid(); err != nil {
		return nil, err
	}

	clientID, err := oidc.GenClientID(meta.RedirectURIs[0].Host)
	if err != nil {
		return nil, err
	}

	return d.ciRepo.New(clientID, meta, false)
}

func (d *dbDriver) ConnectorConfigs() ([]connector.ConnectorConfig, error) {
	return d.cfgRepo.All()
}

func (d *dbDriver) SetConnectorConfigs(cfgs []connector.ConnectorConfig) error {
	return d.cfgRepo.Set(cfgs)
}
