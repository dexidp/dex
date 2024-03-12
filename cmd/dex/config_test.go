package main

import (
	"os"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/kylelemons/godebug/pretty"

	"github.com/dexidp/dex/connector/mock"
	"github.com/dexidp/dex/connector/oidc"
	"github.com/dexidp/dex/server"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/sql"
)

var _ = yaml.YAMLToJSON

func TestValidConfiguration(t *testing.T) {
	configuration := Config{
		Issuer: "http://127.0.0.1:5556/dex",
		Storage: Storage{
			Type: "sqlite3",
			Config: &sql.SQLite3{
				File: "examples/dex.db",
			},
		},
		Web: Web{
			HTTP: "127.0.0.1:5556",
		},
		StaticConnectors: []Connector{
			{
				Type:   "mockCallback",
				ID:     "mock",
				Name:   "Example",
				Config: &mock.CallbackConfig{},
			},
		},
	}
	if err := configuration.Validate(); err != nil {
		t.Fatalf("this configuration should have been valid: %v", err)
	}
}

func TestInvalidConfiguration(t *testing.T) {
	configuration := Config{}
	err := configuration.Validate()
	if err == nil {
		t.Fatal("this configuration should be invalid")
	}
	got := err.Error()
	wanted := `invalid Config:
	-	no issuer specified in config file
	-	no storage supplied in config file
	-	must supply a HTTP/HTTPS  address to listen on`
	if got != wanted {
		t.Fatalf("Expected error message to be %q, got %q", wanted, got)
	}
}

func TestUnmarshalConfig(t *testing.T) {
	rawConfig := []byte(`
issuer: http://127.0.0.1:5556/dex
storage:
  type: postgres
  config:
    host: 10.0.0.1
    port: 65432
    maxOpenConns: 5
    maxIdleConns: 3
    connMaxLifetime: 30
    connectionTimeout: 3
web:
  https: 127.0.0.1:5556
  tlsMinVersion: 1.3
  tlsMaxVersion: 1.2
  headers:
    Strict-Transport-Security: "max-age=31536000; includeSubDomains"

frontend:
  dir: ./web
  extra:
    foo: bar

staticClients:
- id: example-app
  redirectURIs:
  - 'http://127.0.0.1:5555/callback'
  name: 'Example App'
  secret: ZXhhbXBsZS1hcHAtc2VjcmV0

oauth2:
  alwaysShowLoginScreen: true
  grantTypes:
  - refresh_token
  - "urn:ietf:params:oauth:grant-type:token-exchange"

connectors:
- type: mockCallback
  id: mock
  name: Example
- type: oidc
  id: google
  name: Google
  config:
    issuer: https://accounts.google.com
    clientID: foo
    clientSecret: bar
    redirectURI: http://127.0.0.1:5556/dex/callback/google

enablePasswordDB: true
staticPasswords:
- email: "admin@example.com"
  # bcrypt hash of the string "password"
  hash: "$2a$10$33EMT0cVYVlPy6WAMCLsceLYjWhuHpbz5yuZxu/GAFj03J9Lytjuy"
  username: "admin"
  userID: "08a8684b-db88-4b73-90a9-3cd1661f5466"
- email: "foo@example.com"
  # base64'd value of the same bcrypt hash above. We want to be able to parse both of these
  hash: "JDJhJDEwJDMzRU1UMGNWWVZsUHk2V0FNQ0xzY2VMWWpXaHVIcGJ6NXl1Wnh1L0dBRmowM0o5THl0anV5"
  username: "foo"
  userID: "41331323-6f44-45e6-b3b9-2c4b60c02be5"

expiry:
  signingKeys: "7h"
  idTokens: "25h"
  authRequests: "25h"
  deviceRequests: "10m"

logger:
  level: "debug"
  format: "json"
`)

	want := Config{
		Issuer: "http://127.0.0.1:5556/dex",
		Storage: Storage{
			Type: "postgres",
			Config: &sql.Postgres{
				NetworkDB: sql.NetworkDB{
					Host:              "10.0.0.1",
					Port:              65432,
					MaxOpenConns:      5,
					MaxIdleConns:      3,
					ConnMaxLifetime:   30,
					ConnectionTimeout: 3,
				},
			},
		},
		Web: Web{
			HTTPS:         "127.0.0.1:5556",
			TLSMinVersion: "1.3",
			TLSMaxVersion: "1.2",
			Headers: Headers{
				StrictTransportSecurity: "max-age=31536000; includeSubDomains",
			},
		},
		Frontend: server.WebConfig{
			Dir: "./web",
			Extra: map[string]string{
				"foo": "bar",
			},
		},
		StaticClients: []storage.Client{
			{
				ID:     "example-app",
				Secret: "ZXhhbXBsZS1hcHAtc2VjcmV0",
				Name:   "Example App",
				RedirectURIs: []string{
					"http://127.0.0.1:5555/callback",
				},
			},
		},
		OAuth2: OAuth2{
			AlwaysShowLoginScreen: true,
			GrantTypes: []string{
				"refresh_token",
				"urn:ietf:params:oauth:grant-type:token-exchange",
			},
		},
		StaticConnectors: []Connector{
			{
				Type:   "mockCallback",
				ID:     "mock",
				Name:   "Example",
				Config: &mock.CallbackConfig{},
			},
			{
				Type: "oidc",
				ID:   "google",
				Name: "Google",
				Config: &oidc.Config{
					Issuer:       "https://accounts.google.com",
					ClientID:     "foo",
					ClientSecret: "bar",
					RedirectURI:  "http://127.0.0.1:5556/dex/callback/google",
				},
			},
		},
		EnablePasswordDB: true,
		StaticPasswords: []password{
			{
				Email:    "admin@example.com",
				Hash:     []byte("$2a$10$33EMT0cVYVlPy6WAMCLsceLYjWhuHpbz5yuZxu/GAFj03J9Lytjuy"),
				Username: "admin",
				UserID:   "08a8684b-db88-4b73-90a9-3cd1661f5466",
			},
			{
				Email:    "foo@example.com",
				Hash:     []byte("$2a$10$33EMT0cVYVlPy6WAMCLsceLYjWhuHpbz5yuZxu/GAFj03J9Lytjuy"),
				Username: "foo",
				UserID:   "41331323-6f44-45e6-b3b9-2c4b60c02be5",
			},
		},
		Expiry: Expiry{
			SigningKeys:    "7h",
			IDTokens:       "25h",
			AuthRequests:   "25h",
			DeviceRequests: "10m",
		},
		Logger: Logger{
			Level:  "debug",
			Format: "json",
		},
	}

	var c Config
	if err := yaml.Unmarshal(rawConfig, &c); err != nil {
		t.Fatalf("failed to decode config: %v", err)
	}
	if diff := pretty.Compare(c, want); diff != "" {
		t.Errorf("got!=want: %s", diff)
	}
}

func TestUnmarshalConfigWithEnvNoExpand(t *testing.T) {
	// If the env variable DEX_EXPAND_ENV is set and has a "falsy" value, os.ExpandEnv is disabled.
	// ParseBool: "It accepts 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False."
	checkUnmarshalConfigWithEnv(t, "0", false)
	checkUnmarshalConfigWithEnv(t, "f", false)
	checkUnmarshalConfigWithEnv(t, "F", false)
	checkUnmarshalConfigWithEnv(t, "FALSE", false)
	checkUnmarshalConfigWithEnv(t, "false", false)
	checkUnmarshalConfigWithEnv(t, "False", false)
	os.Unsetenv("DEX_EXPAND_ENV")
}

func TestUnmarshalConfigWithEnvExpand(t *testing.T) {
	// If the env variable DEX_EXPAND_ENV is unset or has a "truthy" or unknown value, os.ExpandEnv is enabled.
	// ParseBool: "It accepts 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False."
	checkUnmarshalConfigWithEnv(t, "1", true)
	checkUnmarshalConfigWithEnv(t, "t", true)
	checkUnmarshalConfigWithEnv(t, "T", true)
	checkUnmarshalConfigWithEnv(t, "TRUE", true)
	checkUnmarshalConfigWithEnv(t, "true", true)
	checkUnmarshalConfigWithEnv(t, "True", true)
	// Values that can't be parsed as bool:
	checkUnmarshalConfigWithEnv(t, "UNSET", true)
	checkUnmarshalConfigWithEnv(t, "", true)
	checkUnmarshalConfigWithEnv(t, "whatever - true is default", true)
	os.Unsetenv("DEX_EXPAND_ENV")
}

func checkUnmarshalConfigWithEnv(t *testing.T, dexExpandEnv string, wantExpandEnv bool) {
	// For hashFromEnv:
	os.Setenv("DEX_FOO_USER_PASSWORD", "$2a$10$33EMT0cVYVlPy6WAMCLsceLYjWhuHpbz5yuZxu/GAFj03J9Lytjuy")
	// For os.ExpandEnv ($VAR -> value_of_VAR):
	os.Setenv("DEX_FOO_POSTGRES_HOST", "10.0.0.1")
	os.Setenv("DEX_FOO_OIDC_CLIENT_SECRET", "bar")
	if dexExpandEnv != "UNSET" {
		os.Setenv("DEX_EXPAND_ENV", dexExpandEnv)
	} else {
		os.Unsetenv("DEX_EXPAND_ENV")
	}

	rawConfig := []byte(`
issuer: http://127.0.0.1:5556/dex
storage:
  type: postgres
  config:
    # Env variables are expanded in raw YAML source.
    # Single quotes work fine, as long as the env variable doesn't contain any.
    host: '$DEX_FOO_POSTGRES_HOST'
    port: 65432
    maxOpenConns: 5
    maxIdleConns: 3
    connMaxLifetime: 30
    connectionTimeout: 3
web:
  http: 127.0.0.1:5556

frontend:
  dir: ./web
  extra:
    foo: bar

staticClients:
- id: example-app
  redirectURIs:
  - 'http://127.0.0.1:5555/callback'
  name: 'Example App'
  secret: ZXhhbXBsZS1hcHAtc2VjcmV0

oauth2:
  alwaysShowLoginScreen: true

connectors:
- type: mockCallback
  id: mock
  name: Example
- type: oidc
  id: google
  name: Google
  config:
    issuer: https://accounts.google.com
    clientID: foo
    # Env variables are expanded in raw YAML source.
    # Single quotes work fine, as long as the env variable doesn't contain any.
    clientSecret: '$DEX_FOO_OIDC_CLIENT_SECRET'
    redirectURI: http://127.0.0.1:5556/dex/callback/google

enablePasswordDB: true
staticPasswords:
- email: "admin@example.com"
  # bcrypt hash of the string "password"
  hash: "$2a$10$33EMT0cVYVlPy6WAMCLsceLYjWhuHpbz5yuZxu/GAFj03J9Lytjuy"
  username: "admin"
  userID: "08a8684b-db88-4b73-90a9-3cd1661f5466"
- email: "foo@example.com"
  hashFromEnv: "DEX_FOO_USER_PASSWORD"
  username: "foo"
  userID: "41331323-6f44-45e6-b3b9-2c4b60c02be5"

expiry:
  signingKeys: "7h"
  idTokens: "25h"
  authRequests: "25h"

logger:
  level: "debug"
  format: "json"
`)

	// This is not a valid hostname. It's only used to check whether os.ExpandEnv was applied or not.
	wantPostgresHost := "$DEX_FOO_POSTGRES_HOST"
	wantOidcClientSecret := "$DEX_FOO_OIDC_CLIENT_SECRET"
	if wantExpandEnv {
		wantPostgresHost = "10.0.0.1"
		wantOidcClientSecret = "bar"
	}

	want := Config{
		Issuer: "http://127.0.0.1:5556/dex",
		Storage: Storage{
			Type: "postgres",
			Config: &sql.Postgres{
				NetworkDB: sql.NetworkDB{
					Host:              wantPostgresHost,
					Port:              65432,
					MaxOpenConns:      5,
					MaxIdleConns:      3,
					ConnMaxLifetime:   30,
					ConnectionTimeout: 3,
				},
			},
		},
		Web: Web{
			HTTP: "127.0.0.1:5556",
		},
		Frontend: server.WebConfig{
			Dir: "./web",
			Extra: map[string]string{
				"foo": "bar",
			},
		},
		StaticClients: []storage.Client{
			{
				ID:     "example-app",
				Secret: "ZXhhbXBsZS1hcHAtc2VjcmV0",
				Name:   "Example App",
				RedirectURIs: []string{
					"http://127.0.0.1:5555/callback",
				},
			},
		},
		OAuth2: OAuth2{
			AlwaysShowLoginScreen: true,
		},
		StaticConnectors: []Connector{
			{
				Type:   "mockCallback",
				ID:     "mock",
				Name:   "Example",
				Config: &mock.CallbackConfig{},
			},
			{
				Type: "oidc",
				ID:   "google",
				Name: "Google",
				Config: &oidc.Config{
					Issuer:       "https://accounts.google.com",
					ClientID:     "foo",
					ClientSecret: wantOidcClientSecret,
					RedirectURI:  "http://127.0.0.1:5556/dex/callback/google",
				},
			},
		},
		EnablePasswordDB: true,
		StaticPasswords: []password{
			{
				Email:    "admin@example.com",
				Hash:     []byte("$2a$10$33EMT0cVYVlPy6WAMCLsceLYjWhuHpbz5yuZxu/GAFj03J9Lytjuy"),
				Username: "admin",
				UserID:   "08a8684b-db88-4b73-90a9-3cd1661f5466",
			},
			{
				Email:    "foo@example.com",
				Hash:     []byte("$2a$10$33EMT0cVYVlPy6WAMCLsceLYjWhuHpbz5yuZxu/GAFj03J9Lytjuy"),
				Username: "foo",
				UserID:   "41331323-6f44-45e6-b3b9-2c4b60c02be5",
			},
		},
		Expiry: Expiry{
			SigningKeys:  "7h",
			IDTokens:     "25h",
			AuthRequests: "25h",
		},
		Logger: Logger{
			Level:  "debug",
			Format: "json",
		},
	}

	var c Config
	if err := yaml.Unmarshal(rawConfig, &c); err != nil {
		t.Fatalf("failed to decode config: %v", err)
	}
	if diff := pretty.Compare(c, want); diff != "" {
		t.Errorf("got!=want: %s", diff)
	}
}
