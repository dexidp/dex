package main

import (
	"encoding/json"
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
	Issuer     string      `json:"issuer"`
	Storage    Storage     `json:"storage"`
	Connectors []Connector `json:"connectors"`
	Web        Web         `json:"web"`
	OAuth2     OAuth2      `json:"oauth2"`
	GRPC       GRPC        `json:"grpc"`

	Templates server.TemplateConfig `json:"templates"`

	// StaticClients cause the server to use this list of clients rather than
	// querying the storage. Write operations, like creating a client, will fail.
	StaticClients []storage.Client `json:"staticClients"`

	// If enabled, the server will maintain a list of passwords which can be used
	// to identify a user.
	EnablePasswordDB bool `json:"enablePasswordDB"`

	// StaticPasswords cause the server use this list of passwords rather than
	// querying the storage. Cannot be specified without enabling a passwords
	// database.
	StaticPasswords []storage.Password `json:"staticPasswords"`
}

// OAuth2 describes enabled OAuth2 extensions.
type OAuth2 struct {
	ResponseTypes []string `json:"responseTypes"`
	// If specified, do not prompt the user to approve client authorization. The
	// act of logging in implies authorization.
	SkipApprovalScreen bool `json:"skipApprovalScreen"`
}

// Web is the config format for the HTTP server.
type Web struct {
	HTTP    string `json:"http"`
	HTTPS   string `json:"https"`
	TLSCert string `json:"tlsCert"`
	TLSKey  string `json:"tlsKey"`
}

// GRPC is the config for the gRPC API.
type GRPC struct {
	// The port to listen on.
	Addr        string `json:"addr"`
	TLSCert     string `json:"tlsCert"`
	TLSKey      string `json:"tlsKey"`
	TLSClientCA string `json:"tlsClientCA"`
}

// Storage holds app's storage configuration.
type Storage struct {
	Type   string        `json:"type"`
	Config StorageConfig `json:"config"`
}

// StorageConfig is a configuration that can create a storage.
type StorageConfig interface {
	Open() (storage.Storage, error)
}

var storages = map[string]func() StorageConfig{
	"kubernetes": func() StorageConfig { return new(kubernetes.Config) },
	"memory":     func() StorageConfig { return new(memory.Config) },
	"sqlite3":    func() StorageConfig { return new(sql.SQLite3) },
	"postgres":   func() StorageConfig { return new(sql.Postgres) },
}

// UnmarshalJSON allows Storage to implement the unmarshaler interface to
// dynamically determine the type of the storage config.
func (s *Storage) UnmarshalJSON(b []byte) error {
	var store struct {
		Type   string          `json:"type"`
		Config json.RawMessage `json:"config"`
	}
	if err := json.Unmarshal(b, &store); err != nil {
		return fmt.Errorf("parse storage: %v", err)
	}
	f, ok := storages[store.Type]
	if !ok {
		return fmt.Errorf("unknown storage type %q", store.Type)
	}

	storageConfig := f()
	if len(store.Config) != 0 {
		if err := json.Unmarshal([]byte(store.Config), storageConfig); err != nil {
			return fmt.Errorf("parse storace config: %v", err)
		}
	}
	*s = Storage{
		Type:   store.Type,
		Config: storageConfig,
	}
	return nil
}

// Connector is a magical type that can unmarshal YAML dynamically. The
// Type field determines the connector type, which is then customized for Config.
type Connector struct {
	Type string `json:"type"`
	Name string `json:"name"`
	ID   string `json:"id"`

	Config ConnectorConfig `json:"config"`
}

// ConnectorConfig is a configuration that can open a connector.
type ConnectorConfig interface {
	Open() (connector.Connector, error)
}

var connectors = map[string]func() ConnectorConfig{
	"mockCallback": func() ConnectorConfig { return new(mock.CallbackConfig) },
	"mockPassword": func() ConnectorConfig { return new(mock.PasswordConfig) },
	"ldap":         func() ConnectorConfig { return new(ldap.Config) },
	"github":       func() ConnectorConfig { return new(github.Config) },
	"oidc":         func() ConnectorConfig { return new(oidc.Config) },
}

// UnmarshalJSON allows Connector to implement the unmarshaler interface to
// dynamically determine the type of the connector config.
func (c *Connector) UnmarshalJSON(b []byte) error {
	var conn struct {
		Type string `json:"type"`
		Name string `json:"name"`
		ID   string `json:"id"`

		Config json.RawMessage `json:"config"`
	}
	if err := json.Unmarshal(b, &conn); err != nil {
		return fmt.Errorf("parse connector: %v", err)
	}
	f, ok := connectors[conn.Type]
	if !ok {
		return fmt.Errorf("unknown connector type %q", conn.Type)
	}

	connConfig := f()
	if len(conn.Config) != 0 {
		if err := json.Unmarshal([]byte(conn.Config), connConfig); err != nil {
			return fmt.Errorf("parse connector config: %v", err)
		}
	}
	*c = Connector{
		Type:   conn.Type,
		Name:   conn.Type,
		ID:     conn.ID,
		Config: connConfig,
	}
	return nil
}
