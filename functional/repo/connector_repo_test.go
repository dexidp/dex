package repo

import (
	"os"
	"testing"

	"github.com/go-gorp/gorp"

	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/db"
)

func newConnectorConfigRepo(t *testing.T, configs []connector.ConnectorConfig) connector.ConnectorConfigRepo {
	var dbMap *gorp.DbMap
	if os.Getenv("DEX_TEST_DSN") == "" {
		dbMap = db.NewMemDB()
	} else {
		dbMap = connect(t)
	}
	repo := db.NewConnectorConfigRepo(dbMap)
	if err := repo.Set(configs); err != nil {
		t.Fatalf("Unable to set connector configs: %v", err)
	}
	return repo
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
		repo := newConnectorConfigRepo(t, tt.cfgs)
		if _, err := repo.GetConnectorByID(nil, tt.id); err != tt.err {
			t.Errorf("case %d: want=%v, got=%v", i, tt.err, err)
		}
	}
}
