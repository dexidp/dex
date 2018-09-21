package main

import (
	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/connector/mock"
	"github.com/sirupsen/logrus"
)

func main() {}

type plugin struct{}

// Open prints provided config and returns a mock CallbackConnector
func (plugin) Open(config interface{}, id string, logger logrus.FieldLogger) (connector.Connector, error) {
	logger.Println("loading connector with configuration ", config)
	return mock.NewCallbackConnector(logger), nil
}

// Open exports the plugin for loading
var Open plugin
