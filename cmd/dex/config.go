package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/dexidp/dex/pkg/featureflags"
	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/server"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/ent"
	"github.com/dexidp/dex/storage/etcd"
	"github.com/dexidp/dex/storage/kubernetes"
	"github.com/dexidp/dex/storage/memory"
	"github.com/dexidp/dex/storage/sql"
)

// Config is the config format for the main application.
type Config struct {
	Issuer    string    `json:"issuer"`
	Storage   Storage   `json:"storage"`
	Web       Web       `json:"web"`
	Telemetry Telemetry `json:"telemetry"`
	OAuth2    OAuth2    `json:"oauth2"`
	GRPC      GRPC      `json:"grpc"`
	Expiry    Expiry    `json:"expiry"`
	Logger    Logger    `json:"logger"`

	Frontend server.WebConfig `json:"frontend"`

	// StaticConnectors are user defined connectors specified in the ConfigMap
	// Write operations, like updating a connector, will fail.
	StaticConnectors []Connector `json:"connectors"`

	// StaticClients cause the server to use this list of clients rather than
	// querying the storage. Write operations, like creating a client, will fail.
	StaticClients []storage.Client `json:"staticClients"`

	// If enabled, the server will maintain a list of passwords which can be used
	// to identify a user.
	EnablePasswordDB bool `json:"enablePasswordDB"`

	// StaticPasswords cause the server use this list of passwords rather than
	// querying the storage. Cannot be specified without enabling a passwords
	// database.
	StaticPasswords []password `json:"staticPasswords"`
}

// Validate the configuration
func (c Config) Validate() error {
	// Fast checks. Perform these first for a more responsive CLI.
	checks := []struct {
		bad    bool
		errMsg string
	}{
		{c.Issuer == "", "no issuer specified in config file"},
		{!c.EnablePasswordDB && len(c.StaticPasswords) != 0, "cannot specify static passwords without enabling password db"},
		{c.Storage.Config == nil, "no storage supplied in config file"},
		{c.Web.HTTP == "" && c.Web.HTTPS == "", "must supply a HTTP/HTTPS  address to listen on"},
		{c.Web.HTTPS != "" && c.Web.TLSCert == "", "no cert specified for HTTPS"},
		{c.Web.HTTPS != "" && c.Web.TLSKey == "", "no private key specified for HTTPS"},
		{c.Web.TLSMinVersion != "" && c.Web.TLSMinVersion != "1.2" && c.Web.TLSMinVersion != "1.3", "supported TLS versions are: 1.2, 1.3"},
		{c.Web.TLSMaxVersion != "" && c.Web.TLSMaxVersion != "1.2" && c.Web.TLSMaxVersion != "1.3", "supported TLS versions are: 1.2, 1.3"},
		{c.Web.TLSMaxVersion != "" && c.Web.TLSMinVersion != "" && c.Web.TLSMinVersion > c.Web.TLSMaxVersion, "TLSMinVersion greater than TLSMaxVersion"},
		{c.GRPC.TLSCert != "" && c.GRPC.Addr == "", "no address specified for gRPC"},
		{c.GRPC.TLSKey != "" && c.GRPC.Addr == "", "no address specified for gRPC"},
		{(c.GRPC.TLSCert == "") != (c.GRPC.TLSKey == ""), "must specific both a gRPC TLS cert and key"},
		{c.GRPC.TLSCert == "" && c.GRPC.TLSClientCA != "", "cannot specify gRPC TLS client CA without a gRPC TLS cert"},
		{c.GRPC.TLSMinVersion != "" && c.GRPC.TLSMinVersion != "1.2" && c.GRPC.TLSMinVersion != "1.3", "supported TLS versions are: 1.2, 1.3"},
		{c.GRPC.TLSMaxVersion != "" && c.GRPC.TLSMaxVersion != "1.2" && c.GRPC.TLSMaxVersion != "1.3", "supported TLS versions are: 1.2, 1.3"},
		{c.GRPC.TLSMaxVersion != "" && c.GRPC.TLSMinVersion != "" && c.GRPC.TLSMinVersion > c.GRPC.TLSMaxVersion, "TLSMinVersion greater than TLSMaxVersion"},
	}

	var checkErrors []string

	for _, check := range checks {
		if check.bad {
			checkErrors = append(checkErrors, check.errMsg)
		}
	}
	if len(checkErrors) != 0 {
		return fmt.Errorf("invalid Config:\n\t-\t%s", strings.Join(checkErrors, "\n\t-\t"))
	}
	return nil
}

type password storage.Password

func (p *password) UnmarshalJSON(b []byte) error {
	var data struct {
		Email       string `json:"email"`
		Username    string `json:"username"`
		UserID      string `json:"userID"`
		Hash        string `json:"hash"`
		HashFromEnv string `json:"hashFromEnv"`
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	*p = password(storage.Password{
		Email:    data.Email,
		Username: data.Username,
		UserID:   data.UserID,
	})
	if len(data.Hash) == 0 && len(data.HashFromEnv) > 0 {
		data.Hash = os.Getenv(data.HashFromEnv)
	}
	if len(data.Hash) == 0 {
		return fmt.Errorf("no password hash provided")
	}

	// If this value is a valid bcrypt, use it.
	_, bcryptErr := bcrypt.Cost([]byte(data.Hash))
	if bcryptErr == nil {
		p.Hash = []byte(data.Hash)
		return nil
	}

	// For backwards compatibility try to base64 decode this value.
	hashBytes, err := base64.StdEncoding.DecodeString(data.Hash)
	if err != nil {
		return fmt.Errorf("malformed bcrypt hash: %v", bcryptErr)
	}
	if _, err := bcrypt.Cost(hashBytes); err != nil {
		return fmt.Errorf("malformed bcrypt hash: %v", err)
	}
	p.Hash = hashBytes
	return nil
}

// OAuth2 describes enabled OAuth2 extensions.
type OAuth2 struct {
	// list of allowed grant types,
	// defaults to all supported types
	GrantTypes []string `json:"grantTypes"`

	ResponseTypes []string `json:"responseTypes"`
	// If specified, do not prompt the user to approve client authorization. The
	// act of logging in implies authorization.
	SkipApprovalScreen bool `json:"skipApprovalScreen"`
	// If specified, show the connector selection screen even if there's only one
	AlwaysShowLoginScreen bool `json:"alwaysShowLoginScreen"`
	// This is the connector that can be used for password grant
	PasswordConnector string `json:"passwordConnector"`
}

// Web is the config format for the HTTP server.
type Web struct {
	HTTP           string   `json:"http"`
	HTTPS          string   `json:"https"`
	Headers        Headers  `json:"headers"`
	TLSCert        string   `json:"tlsCert"`
	TLSKey         string   `json:"tlsKey"`
	TLSMinVersion  string   `json:"tlsMinVersion"`
	TLSMaxVersion  string   `json:"tlsMaxVersion"`
	AllowedOrigins []string `json:"allowedOrigins"`
	AllowedHeaders []string `json:"allowedHeaders"`
}

type Headers struct {
	// Set the Content-Security-Policy header to HTTP responses.
	// Unset if blank.
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy
	ContentSecurityPolicy string `json:"Content-Security-Policy"`
	// Set the X-Frame-Options header to HTTP responses.
	// Unset if blank. Accepted values are deny and sameorigin.
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Frame-Options
	XFrameOptions string `json:"X-Frame-Options"`
	// Set the X-Content-Type-Options header to HTTP responses.
	// Unset if blank. Accepted value is nosniff.
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Content-Type-Options
	XContentTypeOptions string `json:"X-Content-Type-Options"`
	// Set the X-XSS-Protection header to all responses.
	// Unset if blank.
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-XSS-Protection
	XXSSProtection string `json:"X-XSS-Protection"`
	// Set the Strict-Transport-Security header to HTTP responses.
	// Unset if blank.
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Strict-Transport-Security
	StrictTransportSecurity string `json:"Strict-Transport-Security"`
}

func (h *Headers) ToHTTPHeader() http.Header {
	if h == nil {
		return make(map[string][]string)
	}
	header := make(map[string][]string)
	if h.ContentSecurityPolicy != "" {
		header["Content-Security-Policy"] = []string{h.ContentSecurityPolicy}
	}
	if h.XFrameOptions != "" {
		header["X-Frame-Options"] = []string{h.XFrameOptions}
	}
	if h.XContentTypeOptions != "" {
		header["X-Content-Type-Options"] = []string{h.XContentTypeOptions}
	}
	if h.XXSSProtection != "" {
		header["X-XSS-Protection"] = []string{h.XXSSProtection}
	}
	if h.StrictTransportSecurity != "" {
		header["Strict-Transport-Security"] = []string{h.StrictTransportSecurity}
	}
	return header
}

// Telemetry is the config format for telemetry including the HTTP server config.
type Telemetry struct {
	HTTP string `json:"http"`
	// EnableProfiling makes profiling endpoints available via web interface host:port/debug/pprof/
	EnableProfiling bool `json:"enableProfiling"`
}

// GRPC is the config for the gRPC API.
type GRPC struct {
	// The port to listen on.
	Addr          string `json:"addr"`
	TLSCert       string `json:"tlsCert"`
	TLSKey        string `json:"tlsKey"`
	TLSClientCA   string `json:"tlsClientCA"`
	TLSMinVersion string `json:"tlsMinVersion"`
	TLSMaxVersion string `json:"tlsMaxVersion"`
	Reflection    bool   `json:"reflection"`
}

// Storage holds app's storage configuration.
type Storage struct {
	Type   string        `json:"type"`
	Config StorageConfig `json:"config"`
}

// StorageConfig is a configuration that can create a storage.
type StorageConfig interface {
	Open(logger log.Logger) (storage.Storage, error)
}

var (
	_ StorageConfig = (*etcd.Etcd)(nil)
	_ StorageConfig = (*kubernetes.Config)(nil)
	_ StorageConfig = (*memory.Config)(nil)
	_ StorageConfig = (*sql.SQLite3)(nil)
	_ StorageConfig = (*sql.Postgres)(nil)
	_ StorageConfig = (*sql.MySQL)(nil)
	_ StorageConfig = (*ent.SQLite3)(nil)
	_ StorageConfig = (*ent.Postgres)(nil)
	_ StorageConfig = (*ent.MySQL)(nil)
)

func getORMBasedSQLStorage(normal, entBased StorageConfig) func() StorageConfig {
	return func() StorageConfig {
		if featureflags.EntEnabled.Enabled() {
			return entBased
		}
		return normal
	}
}

var storages = map[string]func() StorageConfig{
	"etcd":       func() StorageConfig { return new(etcd.Etcd) },
	"kubernetes": func() StorageConfig { return new(kubernetes.Config) },
	"memory":     func() StorageConfig { return new(memory.Config) },
	"sqlite3":    getORMBasedSQLStorage(&sql.SQLite3{}, &ent.SQLite3{}),
	"postgres":   getORMBasedSQLStorage(&sql.Postgres{}, &ent.Postgres{}),
	"mysql":      getORMBasedSQLStorage(&sql.MySQL{}, &ent.MySQL{}),
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
		data := []byte(store.Config)
		if featureflags.ExpandEnv.Enabled() {
			// Caution, we're expanding in the raw JSON/YAML source. This may not be what the admin expects.
			data = []byte(os.ExpandEnv(string(store.Config)))
		}
		if err := json.Unmarshal(data, storageConfig); err != nil {
			return fmt.Errorf("parse storage config: %v", err)
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

	Config server.ConnectorConfig `json:"config"`
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
	f, ok := server.ConnectorsConfig[conn.Type]
	if !ok {
		return fmt.Errorf("unknown connector type %q", conn.Type)
	}

	connConfig := f()
	if len(conn.Config) != 0 {
		data := []byte(conn.Config)
		if featureflags.ExpandEnv.Enabled() {
			// Caution, we're expanding in the raw JSON/YAML source. This may not be what the admin expects.
			data = []byte(os.ExpandEnv(string(conn.Config)))
		}
		if err := json.Unmarshal(data, connConfig); err != nil {
			return fmt.Errorf("parse connector config: %v", err)
		}
	}
	*c = Connector{
		Type:   conn.Type,
		Name:   conn.Name,
		ID:     conn.ID,
		Config: connConfig,
	}
	return nil
}

// ToStorageConnector converts an object to storage connector type.
func ToStorageConnector(c Connector) (storage.Connector, error) {
	data, err := json.Marshal(c.Config)
	if err != nil {
		return storage.Connector{}, fmt.Errorf("failed to marshal connector config: %v", err)
	}

	return storage.Connector{
		ID:     c.ID,
		Type:   c.Type,
		Name:   c.Name,
		Config: data,
	}, nil
}

// Expiry holds configuration for the validity period of components.
type Expiry struct {
	// SigningKeys defines the duration of time after which the SigningKeys will be rotated.
	SigningKeys string `json:"signingKeys"`

	// IdTokens defines the duration of time for which the IdTokens will be valid.
	IDTokens string `json:"idTokens"`

	// AuthRequests defines the duration of time for which the AuthRequests will be valid.
	AuthRequests string `json:"authRequests"`

	// DeviceRequests defines the duration of time for which the DeviceRequests will be valid.
	DeviceRequests string `json:"deviceRequests"`

	// RefreshTokens defines refresh tokens expiry policy
	RefreshTokens RefreshToken `json:"refreshTokens"`
}

// Logger holds configuration required to customize logging for dex.
type Logger struct {
	// Level sets logging level severity.
	Level string `json:"level"`

	// Format specifies the format to be used for logging.
	Format string `json:"format"`
}

type RefreshToken struct {
	DisableRotation   bool   `json:"disableRotation"`
	ReuseInterval     string `json:"reuseInterval"`
	AbsoluteLifetime  string `json:"absoluteLifetime"`
	ValidIfNotUsedFor string `json:"validIfNotUsedFor"`
}
