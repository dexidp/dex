package web3

import (
	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/log"
)

type Config struct{}

func (c *Config) Open(id string, logger log.Logger) (connector.Connector, error) {
	return &Web3Connector{}, nil
}

type Web3Connector struct {
}

func (c *Web3Connector) A() int {
	return 1
}
