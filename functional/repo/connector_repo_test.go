package repo

import (
	"fmt"
	"github.com/coreos/dex/connector"
	"os"
	"testing"
)

type connectorConfigRepoFactory func(cfgs []connector.ConnectorConfig) connector.ConnectorConfigRepo

var makeTestConnectorConfigRepoFromConfigs connectorConfigRepoFactory

func init() {
	if dsn := os.Getenv("DEX_TEST_DSN"); dsn == "" {
		makeTestConnectorConfigRepoFromConfigs = connector.NewConnectorConfigRepoFromConfigs
	} else {
		makeTestConnectorConfigRepoFromConfigs = makeTestConnectorConfigRepoMem(dsn)
	}
}

func makeTestConnectorConfigRepoMem(dsn string) connectorConfigRepoFactory {
	return func(cfgs []connector.ConnectorConfig) connector.ConnectorConfigRepo {
		dbMap := initDB(dsn)

		repo := dbMap.NewConnectorConfigRepo()
		if err := repo.Set(cfgs); err != nil {
			panic(fmt.Sprintf("Unable to set connector configs: %v", err))
		}
		return repo
	}
}

func TestConnectorConfigRepoGetByID(t *testing.T) {
	tests := []struct {
		cfgs []connector.ConnectorConfig
		id   string
		err  error
	}{
		{
			cfgs: []connector.ConnectorConfig{
				&connector.LocalConnectorConfig{ID: "local"},
			},
			id: "local",
		},
		{
			cfgs: []connector.ConnectorConfig{
				&connector.LocalConnectorConfig{ID: "local1"},
				&connector.LocalConnectorConfig{ID: "local2"},
			},
			id: "local2",
		},
		{
			cfgs: []connector.ConnectorConfig{
				&connector.LocalConnectorConfig{ID: "local1"},
				&connector.LocalConnectorConfig{ID: "local2"},
			},
			id:  "foo",
			err: connector.ErrorNotFound,
		},
	}

	for i, tt := range tests {
		repo := makeTestConnectorConfigRepoFromConfigs(tt.cfgs)
		if _, err := repo.GetConnectorByID(nil, tt.id); err != tt.err {
			t.Errorf("case %d: want=%v, got=%v", i, tt.err, err)
		}
	}
}
