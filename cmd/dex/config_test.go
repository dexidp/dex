package main

import (
	"testing"

	"github.com/ghodss/yaml"
	"github.com/kylelemons/godebug/pretty"

	"github.com/dexidp/dex/connector/mock"
	"github.com/dexidp/dex/connector/oidc"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/sql"
)

var _ = yaml.YAMLToJSON

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
  http: 127.0.0.1:5556
staticClients:
- id: example-app
  redirectURIs:
  - 'http://127.0.0.1:5555/callback'
  name: 'Example App'
  secret: ZXhhbXBsZS1hcHAtc2VjcmV0

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

logger:
  level: "debug"
  format: "json"
`)

	want := Config{
		Issuer: "http://127.0.0.1:5556/dex",
		Storage: Storage{
			Type: "postgres",
			Config: &sql.Postgres{
				Host:              "10.0.0.1",
				Port:              65432,
				MaxOpenConns:      5,
				MaxIdleConns:      3,
				ConnMaxLifetime:   30,
				ConnectionTimeout: 3,
			},
		},
		Web: Web{
			HTTP: "127.0.0.1:5556",
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
