package plugin

import (
	plug "plugin"

	"github.com/dexidp/dex/connector"
	"github.com/sirupsen/logrus"
)

// Config holds configuration for loading a plugin
//
// An example config:
//  id: sample
//  name: sample
//  type: plugin
//  config:
//	plugin: sample.so
//	config:
//	  name: sample
//	  options: limitless
type Config struct {
	Plugin string
	Config interface{}
}

type pluginConfig interface {
	Open(interface{}, string, logrus.FieldLogger) (connector.Connector, error)
}

// Open returns a connector loaded as a plugin
func (c *Config) Open(id string, logger logrus.FieldLogger) (connector.Connector, error) {
	plug, err := plug.Open(c.Plugin)
	if err != nil {
		return nil, err
	}
	symConnector, err := plug.Lookup("Open")
	if err != nil {
		return nil, err
	}
	var connector pluginConfig
	connector, ok := symConnector.(pluginConfig)
	if !ok {
		return nil, err
	}
	return connector.Open(c.Config, id, logger)
}
