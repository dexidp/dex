package main

import (
	"github.com/coreos/dex/connector"
	"github.com/coreos/go-oidc/oidc"
)

type driver interface {
	NewClient(oidc.ClientMetadata) (*oidc.ClientCredentials, error)

	ConnectorConfigs() ([]connector.ConnectorConfig, error)
	SetConnectorConfigs([]connector.ConnectorConfig) error
}
