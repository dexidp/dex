package main

import (
	"github.com/coreos/dex/client"
	"github.com/coreos/dex/client/manager"
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
		cfgRepo:   db.NewConnectorConfigRepo(dbc),
		ciManager: manager.NewClientManager(db.NewClientRepo(dbc), db.TransactionFactory(dbc), manager.ManagerOptions{}),
	}

	return drv, nil
}

type dbDriver struct {
	ciManager *manager.ClientManager
	cfgRepo   *db.ConnectorConfigRepo
}

func (d *dbDriver) NewClient(meta oidc.ClientMetadata) (*oidc.ClientCredentials, error) {
	if err := meta.Valid(); err != nil {
		return nil, err
	}
	cli := client.Client{
		Metadata: meta,
	}
	return d.ciManager.New(cli, nil)
}

func (d *dbDriver) ConnectorConfigs() ([]connector.ConnectorConfig, error) {
	return d.cfgRepo.All()
}

func (d *dbDriver) SetConnectorConfigs(cfgs []connector.ConnectorConfig) error {
	return d.cfgRepo.Set(cfgs)
}
