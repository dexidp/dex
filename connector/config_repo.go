package connector

import (
	"encoding/json"
	"io"
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
