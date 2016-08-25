package main

import (
	"fmt"

	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/connector/github"
	"github.com/coreos/dex/connector/ldap"
	"github.com/coreos/dex/connector/mock"
	"github.com/coreos/dex/connector/oidc"
	"github.com/coreos/dex/storage"
	"github.com/coreos/dex/storage/kubernetes"
	"github.com/coreos/dex/storage/memory"
)

// Config is the config format for the main application.
type Config struct {
	Issuer     string      `yaml:"issuer"`
	Storage    Storage     `yaml:"storage"`
	Connectors []Connector `yaml:"connectors"`
	Web        Web         `yaml:"web"`
	OAuth2     OAuth2      `yaml:"oauth2"`

	StaticClients []storage.Client `yaml:"staticClients"`
}

// OAuth2 describes enabled OAuth2 extensions.
type OAuth2 struct {
	ResponseTypes []string `yaml:"responseTypes"`
}

// Web is the config format for the HTTP server.
type Web struct {
	HTTP    string `yaml:"http"`
	HTTPS   string `yaml:"https"`
	TLSCert string `yaml:"tlsCert"`
	TLSKey  string `yaml:"tlsKey"`
}

// Storage holds app's storage configuration.
type Storage struct {
	Type   string        `yaml:"type"`
	Config StorageConfig `yaml:"config"`
}

// UnmarshalYAML allows Storage to unmarshal its config field dynamically
// depending on the type of storage.
func (s *Storage) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var storageMeta struct {
		Type string `yaml:"type"`
	}
	if err := unmarshal(&storageMeta); err != nil {
		return err
	}
	s.Type = storageMeta.Type
	// TODO(ericchiang): replace this with a registration process.
	var err error
	switch storageMeta.Type {
	case "kubernetes":
		var config struct {
			Config kubernetes.Config `yaml:"config"`
		}
		err = unmarshal(&config)
		s.Config = &config.Config
	case "memory":
		var config struct {
			Config memory.Config `yaml:"config"`
		}
		err = unmarshal(&config)
		s.Config = &config.Config
	default:
		return fmt.Errorf("unknown storage type %q", storageMeta.Type)
	}
	return err
}

// StorageConfig is a configuration that can create a storage.
type StorageConfig interface {
	Open() (storage.Storage, error)
}

// Connector is a magical type that can unmarshal YAML dynamically. The
// Type field determines the connector type, which is then customized for Config.
type Connector struct {
	Type string `yaml:"type"`
	Name string `yaml:"name"`
	ID   string `yaml:"id"`

	Config ConnectorConfig `yaml:"config"`
}

// ConnectorConfig is a configuration that can open a connector.
type ConnectorConfig interface {
	Open() (connector.Connector, error)
}

// UnmarshalYAML allows Connector to unmarshal its config field dynamically
// depending on the type of connector.
func (c *Connector) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var connectorMetadata struct {
		Type string `yaml:"type"`
		Name string `yaml:"name"`
		ID   string `yaml:"id"`
	}
	if err := unmarshal(&connectorMetadata); err != nil {
		return err
	}
	c.Type = connectorMetadata.Type
	c.Name = connectorMetadata.Name
	c.ID = connectorMetadata.ID

	var err error
	switch c.Type {
	case "mockCallback":
		var config struct {
			Config mock.CallbackConfig `yaml:"config"`
		}
		err = unmarshal(&config)
		c.Config = &config.Config
	case "mockPassword":
		var config struct {
			Config mock.PasswordConfig `yaml:"config"`
		}
		err = unmarshal(&config)
		c.Config = &config.Config
	case "ldap":
		var config struct {
			Config ldap.Config `yaml:"config"`
		}
		err = unmarshal(&config)
		c.Config = &config.Config
	case "github":
		var config struct {
			Config github.Config `yaml:"config"`
		}
		err = unmarshal(&config)
		c.Config = &config.Config
	case "oidc":
		var config struct {
			Config oidc.Config `yaml:"config"`
		}
		err = unmarshal(&config)
		c.Config = &config.Config
	default:
		return fmt.Errorf("unknown connector type %q", c.Type)
	}
	return err
}
