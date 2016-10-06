package main

import (
	"encoding/base64"
	"fmt"

	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/connector/github"
	"github.com/coreos/dex/connector/ldap"
	"github.com/coreos/dex/connector/mock"
	"github.com/coreos/dex/connector/oidc"
	"github.com/coreos/dex/server"
	"github.com/coreos/dex/storage"
	"github.com/coreos/dex/storage/kubernetes"
	"github.com/coreos/dex/storage/memory"
	"github.com/coreos/dex/storage/sql"
)

// Config is the config format for the main application.
type Config struct {
	Issuer     string      `yaml:"issuer"`
	Storage    Storage     `yaml:"storage"`
	Connectors []Connector `yaml:"connectors"`
	Web        Web         `yaml:"web"`
	OAuth2     OAuth2      `yaml:"oauth2"`
	GRPC       GRPC        `yaml:"grpc"`

	Templates server.TemplateConfig `yaml:"templates"`

	// StaticClients cause the server to use this list of clients rather than
	// querying the storage. Write operations, like creating a client, will fail.
	StaticClients []storage.Client `yaml:"staticClients"`

	// If enabled, the server will maintain a list of passwords which can be used
	// to identify a user.
	EnablePasswordDB bool `yaml:"enablePasswordDB"`

	// StaticPasswords cause the server use this list of passwords rather than
	// querying the storage. Cannot be specified without enabling a passwords
	// database.
	//
	// The "password" type is identical to the storage.Password type, but does
	// unmarshaling into []byte correctly.
	StaticPasswords []password `yaml:"staticPasswords"`
}

type password struct {
	Email    string `yaml:"email"`
	Username string `yaml:"username"`
	UserID   string `yaml:"userID"`

	// Because our YAML parser doesn't base64, we have to do it ourselves.
	//
	// TODO(ericchiang): switch to github.com/ghodss/yaml
	Hash string `yaml:"hash"`
}

// decode the hash appropriately and convert to the storage passwords.
func (p password) toPassword() (storage.Password, error) {
	hash, err := base64.StdEncoding.DecodeString(p.Hash)
	if err != nil {
		return storage.Password{}, fmt.Errorf("decoding hash: %v", err)
	}
	return storage.Password{
		Email:    p.Email,
		Username: p.Username,
		UserID:   p.UserID,
		Hash:     hash,
	}, nil
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

// GRPC is the config for the gRPC API.
type GRPC struct {
	// The port to listen on.
	Addr    string `yaml:"addr"`
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
	case "sqlite3":
		var config struct {
			Config sql.SQLite3 `yaml:"config"`
		}
		err = unmarshal(&config)
		s.Config = &config.Config
	case "postgres":
		var config struct {
			Config sql.Postgres `yaml:"config"`
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
