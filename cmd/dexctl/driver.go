package main

import (
	"github.com/coreos/go-oidc/oidc"
)

type driver interface {
	NewClient(oidc.ClientMetadata) (*oidc.ClientCredentials, error)

	ConnectorConfigs() ([]interface{}, error)
	SetConnectorConfigs([]interface{}) error
}
