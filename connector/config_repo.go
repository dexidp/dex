package connector

import (
	"encoding/json"
	"io"

	"github.com/coreos/dex/repo"
)

func ReadConfigs(r io.Reader) ([]ConnectorConfig, error) {
	var ms []map[string]interface{}
	if err := json.NewDecoder(r).Decode(&ms); err != nil {
		return nil, err
	}
	cfgs := make([]ConnectorConfig, len(ms))
	for i, m := range ms {
		cfg, err := newConnectorConfigFromMap(m)
		if err != nil {
			return nil, err
		}
		cfgs[i] = cfg
	}
	return cfgs, nil
}

type memConnectorConfigRepo struct {
	configs []ConnectorConfig
}

func NewConnectorConfigRepoFromConfigs(cfgs []ConnectorConfig) ConnectorConfigRepo {
	return &memConnectorConfigRepo{configs: cfgs}
}

func (r *memConnectorConfigRepo) All() ([]ConnectorConfig, error) {
	return r.configs, nil
}

func (r *memConnectorConfigRepo) GetConnectorByID(_ repo.Transaction, id string) (ConnectorConfig, error) {
	for _, cfg := range r.configs {
		if cfg.ConnectorID() == id {
			return cfg, nil
		}
	}
	return nil, ErrorNotFound
}
