package server

import (
	"bytes"
	"log/slog"
	"net"
	"slices"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/dexidp/dex/api/v2"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

// apiClient is a test gRPC client. When constructed, it runs a server in
// the background to exercise the serialization and network configuration
// instead of just this package's server implementation.
type apiClient struct {
	// Embedded gRPC client to talk to the server.
	api.DexClient
	// Close releases resources associated with this client, including shutting
	// down the background server.
	Close func()
}

func newLogger(t *testing.T) *slog.Logger {
	return slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// newAPI constructs a gRCP client connected to a backing server.
func newAPI(t *testing.T, s storage.Storage, logger *slog.Logger) *apiClient {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	serv := grpc.NewServer()
	api.RegisterDexServer(serv, NewAPI(s, logger, "test", nil))
	go serv.Serve(l)

	// NewClient will retry automatically if the serv.Serve() goroutine
	// hasn't started yet.
	conn, err := grpc.NewClient(l.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}

	return &apiClient{
		DexClient: api.NewDexClient(conn),
		Close: func() {
			conn.Close()
			serv.Stop()
			l.Close()
		},
	}
}

// Attempts to create, update and delete a test Password
func TestPassword(t *testing.T) {
	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	email := "test@example.com"
	p := api.Password{
		Email: email,
		// bcrypt hash of the value "test1" with cost 10
		Hash:     []byte("$2a$10$XVMN/Fid.Ks4CXgzo8fpR.iU1khOMsP5g9xQeXuBm1wXjRX8pjUtO"),
		Username: "test",
		UserId:   "test123",
	}

	createReq := api.CreatePasswordReq{
		Password: &p,
	}

	if resp, err := client.CreatePassword(ctx, &createReq); err != nil || resp.AlreadyExists {
		if resp.AlreadyExists {
			t.Fatalf("Unable to create password since %s already exists", createReq.Password.Email)
		}
		t.Fatalf("Unable to create password: %v", err)
	}

	// Attempt to create a password that already exists.
	if resp, _ := client.CreatePassword(ctx, &createReq); !resp.AlreadyExists {
		t.Fatalf("Created password %s twice", createReq.Password.Email)
	}

	// Attempt to verify valid password and email
	goodVerifyReq := &api.VerifyPasswordReq{
		Email:    email,
		Password: "test1",
	}
	goodVerifyResp, err := client.VerifyPassword(ctx, goodVerifyReq)
	if err != nil {
		t.Fatalf("Unable to run verify password we expected to be valid for correct email: %v", err)
	}
	if !goodVerifyResp.Verified {
		t.Fatalf("verify password failed for password expected to be valid for correct email. expected %t, found %t", true, goodVerifyResp.Verified)
	}
	if goodVerifyResp.NotFound {
		t.Fatalf("verify password failed to return not found response. expected %t, found %t", false, goodVerifyResp.NotFound)
	}

	// Check not found response for valid password with wrong email
	badEmailVerifyReq := &api.VerifyPasswordReq{
		Email:    "somewrongaddress@email.com",
		Password: "test1",
	}
	badEmailVerifyResp, err := client.VerifyPassword(ctx, badEmailVerifyReq)
	if err != nil {
		t.Fatalf("Unable to run verify password for incorrect email: %v", err)
	}
	if badEmailVerifyResp.Verified {
		t.Fatalf("verify password passed for password expected to be not found. expected %t, found %t", false, badEmailVerifyResp.Verified)
	}
	if !badEmailVerifyResp.NotFound {
		t.Fatalf("expected not found response for verify password with bad email. expected %t, found %t", true, badEmailVerifyResp.NotFound)
	}

	// Check that wrong password fails
	badPassVerifyReq := &api.VerifyPasswordReq{
		Email:    email,
		Password: "wrong_password",
	}
	badPassVerifyResp, err := client.VerifyPassword(ctx, badPassVerifyReq)
	if err != nil {
		t.Fatalf("Unable to run verify password for password we expected to be invalid: %v", err)
	}
	if badPassVerifyResp.Verified {
		t.Fatalf("verify password passed for password we expected to fail. expected %t, found %t", false, badPassVerifyResp.Verified)
	}
	if badPassVerifyResp.NotFound {
		t.Fatalf("did not expect expected not found response for verify password with bad email. expected %t, found %t", false, badPassVerifyResp.NotFound)
	}

	updateReq := api.UpdatePasswordReq{
		Email:       email,
		NewUsername: "test1",
	}

	if _, err := client.UpdatePassword(ctx, &updateReq); err != nil {
		t.Fatalf("Unable to update password: %v", err)
	}

	pass, err := s.GetPassword(ctx, updateReq.Email)
	if err != nil {
		t.Fatalf("Unable to retrieve password: %v", err)
	}

	if pass.Username != updateReq.NewUsername {
		t.Fatalf("UpdatePassword failed. Expected username %s retrieved %s", updateReq.NewUsername, pass.Username)
	}

	deleteReq := api.DeletePasswordReq{
		Email: "test@example.com",
	}

	if _, err := client.DeletePassword(ctx, &deleteReq); err != nil {
		t.Fatalf("Unable to delete password: %v", err)
	}
}

// Ensures checkCost returns expected values
func TestCheckCost(t *testing.T) {
	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	tests := []struct {
		name      string
		inputHash []byte

		wantErr bool
	}{
		{
			name: "valid cost",
			// bcrypt hash of the value "test1" with cost 12 (default)
			inputHash: []byte("$2a$12$M2Ot95Qty1MuQdubh1acWOiYadJDzeVg3ve4n5b.dgcgPdjCseKx2"),
		},
		{
			name:      "invalid hash",
			inputHash: []byte(""),
			wantErr:   true,
		},
		{
			name: "cost below default",
			// bcrypt hash of the value "test1" with cost 4
			inputHash: []byte("$2a$04$8bSTbuVCLpKzaqB3BmgI7edDigG5tIQKkjYUu/mEO9gQgIkw9m7eG"),
			wantErr:   true,
		},
		{
			name: "cost above recommendation",
			// bcrypt hash of the value "test1" with cost 17
			inputHash: []byte("$2a$17$tWuZkTxtSmRyWZAGWVHQE.7npdl.TgP8adjzLJD.SyjpFznKBftPe"),
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		if err := checkCost(tc.inputHash); err != nil {
			if !tc.wantErr {
				t.Errorf("%s: %s", tc.name, err)
			}
			continue
		}

		if tc.wantErr {
			t.Errorf("%s: expected err", tc.name)
			continue
		}
	}
}

// Attempts to list and revoke an existing refresh token.
func TestRefreshToken(t *testing.T) {
	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	// Creating a storage with an existing refresh token and offline session for the user.
	id := storage.NewID()
	r := storage.RefreshToken{
		ID:          id,
		Token:       "bar",
		Nonce:       "foo",
		ClientID:    "client_id",
		ConnectorID: "client_secret",
		Scopes:      []string{"openid", "email", "profile"},
		CreatedAt:   time.Now().UTC().Round(time.Millisecond),
		LastUsed:    time.Now().UTC().Round(time.Millisecond),
		Claims: storage.Claims{
			UserID:        "1",
			Username:      "jane",
			Email:         "jane.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
		ConnectorData: []byte(`{"some":"data"}`),
	}

	if err := s.CreateRefresh(ctx, r); err != nil {
		t.Fatalf("create refresh token: %v", err)
	}

	tokenRef := storage.RefreshTokenRef{
		ID:        r.ID,
		ClientID:  r.ClientID,
		CreatedAt: r.CreatedAt,
		LastUsed:  r.LastUsed,
	}

	session := storage.OfflineSessions{
		UserID:  r.Claims.UserID,
		ConnID:  r.ConnectorID,
		Refresh: make(map[string]*storage.RefreshTokenRef),
	}
	session.Refresh[tokenRef.ClientID] = &tokenRef

	if err := s.CreateOfflineSessions(ctx, session); err != nil {
		t.Fatalf("create offline session: %v", err)
	}

	subjectString, err := internal.Marshal(&internal.IDTokenSubject{
		UserId: r.Claims.UserID,
		ConnId: r.ConnectorID,
	})
	if err != nil {
		t.Errorf("failed to marshal offline session ID: %v", err)
	}

	// Testing the api.
	listReq := api.ListRefreshReq{
		UserId: subjectString,
	}

	listResp, err := client.ListRefresh(ctx, &listReq)
	if err != nil {
		t.Fatalf("Unable to list refresh tokens for user: %v", err)
	}

	for _, tok := range listResp.RefreshTokens {
		if tok.CreatedAt != r.CreatedAt.Unix() {
			t.Errorf("Expected CreatedAt timestamp %v, got %v", r.CreatedAt.Unix(), tok.CreatedAt)
		}

		if tok.LastUsed != r.LastUsed.Unix() {
			t.Errorf("Expected LastUsed timestamp %v, got %v", r.LastUsed.Unix(), tok.LastUsed)
		}
	}

	revokeReq := api.RevokeRefreshReq{
		UserId:   subjectString,
		ClientId: r.ClientID,
	}

	resp, err := client.RevokeRefresh(ctx, &revokeReq)
	if err != nil {
		t.Fatalf("Unable to revoke refresh tokens for user: %v", err)
	}
	if resp.NotFound {
		t.Errorf("refresh token session wasn't found")
	}

	// Try to delete again.
	//
	// See https://github.com/dexidp/dex/issues/1055
	resp, err = client.RevokeRefresh(ctx, &revokeReq)
	if err != nil {
		t.Fatalf("Unable to revoke refresh tokens for user: %v", err)
	}
	if !resp.NotFound {
		t.Errorf("refresh token session was found")
	}

	if resp, _ := client.ListRefresh(ctx, &listReq); len(resp.RefreshTokens) != 0 {
		t.Fatalf("Refresh token returned in spite of revoking it.")
	}
}

func TestUpdateClient(t *testing.T) {
	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	createClient := func(t *testing.T, clientId string) {
		resp, err := client.CreateClient(ctx, &api.CreateClientReq{
			Client: &api.Client{
				Id:           clientId,
				Secret:       "",
				RedirectUris: []string{},
				TrustedPeers: nil,
				Public:       true,
				Name:         "",
				LogoUrl:      "",
			},
		})
		if err != nil {
			t.Fatalf("unable to create the client: %v", err)
		}

		if resp == nil {
			t.Fatalf("create client returned no response")
		}
		if resp.AlreadyExists {
			t.Error("existing client was found")
		}

		if resp.Client == nil {
			t.Fatalf("no client created")
		}
	}

	deleteClient := func(t *testing.T, clientId string) {
		resp, err := client.DeleteClient(ctx, &api.DeleteClientReq{
			Id: clientId,
		})
		if err != nil {
			t.Fatalf("unable to delete the client: %v", err)
		}
		if resp == nil {
			t.Fatalf("delete client delete client returned no response")
		}
	}

	tests := map[string]struct {
		setup   func(t *testing.T, clientId string)
		cleanup func(t *testing.T, clientId string)
		req     *api.UpdateClientReq
		wantErr bool
		want    *api.UpdateClientResp
	}{
		"update client": {
			setup:   createClient,
			cleanup: deleteClient,
			req: &api.UpdateClientReq{
				Id:           "test",
				RedirectUris: []string{"https://redirect"},
				TrustedPeers: []string{"test"},
				Name:         "test",
				LogoUrl:      "https://logout",
			},
			wantErr: false,
			want: &api.UpdateClientResp{
				NotFound: false,
			},
		},
		"update client without ID": {
			setup:   createClient,
			cleanup: deleteClient,
			req: &api.UpdateClientReq{
				Id:           "",
				RedirectUris: nil,
				TrustedPeers: nil,
				Name:         "test",
				LogoUrl:      "test",
			},
			wantErr: true,
			want: &api.UpdateClientResp{
				NotFound: false,
			},
		},
		"update client which not exists ": {
			req: &api.UpdateClientReq{
				Id:           "test",
				RedirectUris: nil,
				TrustedPeers: nil,
				Name:         "test",
				LogoUrl:      "test",
			},
			wantErr: true,
			want: &api.UpdateClientResp{
				NotFound: false,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t, tc.req.Id)
			}
			resp, err := client.UpdateClient(ctx, tc.req)
			if err != nil && !tc.wantErr {
				t.Fatalf("failed to update the client: %v", err)
			}

			if !tc.wantErr {
				if resp == nil {
					t.Fatalf("update client response not found")
				}

				if tc.want.NotFound != resp.NotFound {
					t.Errorf("expected in response NotFound: %t", tc.want.NotFound)
				}

				client, err := s.GetClient(ctx, tc.req.Id)
				if err != nil {
					t.Errorf("no client found in the storage: %v", err)
				}

				if tc.req.Id != client.ID {
					t.Errorf("expected stored client with ID: %s, found %s", tc.req.Id, client.ID)
				}
				if tc.req.Name != client.Name {
					t.Errorf("expected stored client with Name: %s, found %s", tc.req.Name, client.Name)
				}
				if tc.req.LogoUrl != client.LogoURL {
					t.Errorf("expected stored client with LogoURL: %s, found %s", tc.req.LogoUrl, client.LogoURL)
				}
				for _, redirectURI := range tc.req.RedirectUris {
					found := slices.Contains(client.RedirectURIs, redirectURI)
					if !found {
						t.Errorf("expected redirect URI: %s", redirectURI)
					}
				}
				for _, peer := range tc.req.TrustedPeers {
					found := slices.Contains(client.TrustedPeers, peer)
					if !found {
						t.Errorf("expected trusted peer: %s", peer)
					}
				}
			}

			if tc.cleanup != nil {
				tc.cleanup(t, tc.req.Id)
			}
		})
	}
}

func TestCreateConnector(t *testing.T) {
	t.Setenv("DEX_API_CONNECTORS_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	connectorID := "connector123"
	connectorName := "TestConnector"
	connectorType := "TestType"
	connectorConfig := []byte(`{"key": "value"}`)

	createReq := api.CreateConnectorReq{
		Connector: &api.Connector{
			Id:     connectorID,
			Name:   connectorName,
			Type:   connectorType,
			Config: connectorConfig,
		},
	}

	// Test valid connector creation
	if resp, err := client.CreateConnector(ctx, &createReq); err != nil || resp.AlreadyExists {
		if err != nil {
			t.Fatalf("Unable to create connector: %v", err)
		} else if resp.AlreadyExists {
			t.Fatalf("Unable to create connector since %s already exists", connectorID)
		}
		t.Fatalf("Unable to create connector: %v", err)
	}

	// Test creating the same connector again (expecting failure)
	if resp, _ := client.CreateConnector(ctx, &createReq); !resp.AlreadyExists {
		t.Fatalf("Created connector %s twice", connectorID)
	}

	createReq.Connector.Config = []byte("invalid_json")

	// Test invalid JSON config
	if _, err := client.CreateConnector(ctx, &createReq); err == nil {
		t.Fatal("Expected an error for invalid JSON config, but none occurred")
	} else if !strings.Contains(err.Error(), "invalid config supplied") {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestUpdateConnector(t *testing.T) {
	t.Setenv("DEX_API_CONNECTORS_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	connectorID := "connector123"
	newConnectorName := "UpdatedConnector"
	newConnectorType := "UpdatedType"
	newConnectorConfig := []byte(`{"updated_key": "updated_value"}`)

	// Create a connector for testing
	createReq := api.CreateConnectorReq{
		Connector: &api.Connector{
			Id:     connectorID,
			Name:   "TestConnector",
			Type:   "TestType",
			Config: []byte(`{"key": "value"}`),
		},
	}
	client.CreateConnector(ctx, &createReq)

	updateReq := api.UpdateConnectorReq{
		Id:        connectorID,
		NewName:   newConnectorName,
		NewType:   newConnectorType,
		NewConfig: newConnectorConfig,
	}

	// Test valid connector update
	if _, err := client.UpdateConnector(ctx, &updateReq); err != nil {
		t.Fatalf("Unable to update connector: %v", err)
	}

	resp, err := client.ListConnectors(ctx, &api.ListConnectorReq{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	for _, connector := range resp.Connectors {
		if connector.Id == connectorID {
			if connector.Name != newConnectorName {
				t.Fatal("connector name should have been updated")
			}
			if string(connector.Config) != string(newConnectorConfig) {
				t.Fatal("connector config should have been updated")
			}
			if connector.Type != newConnectorType {
				t.Fatal("connector type should have been updated")
			}
		}
	}

	updateReq.NewConfig = []byte("invalid_json")

	// Test invalid JSON config in update request
	if _, err := client.UpdateConnector(ctx, &updateReq); err == nil {
		t.Fatal("Expected an error for invalid JSON config in update, but none occurred")
	} else if !strings.Contains(err.Error(), "invalid config supplied") {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestUpdateConnectorGrantTypes(t *testing.T) {
	t.Setenv("DEX_API_CONNECTORS_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	connectorID := "connector-gt"

	// Create a connector without grant types
	createReq := api.CreateConnectorReq{
		Connector: &api.Connector{
			Id:     connectorID,
			Name:   "TestConnector",
			Type:   "TestType",
			Config: []byte(`{"key": "value"}`),
		},
	}
	_, err := client.CreateConnector(ctx, &createReq)
	if err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	// Set grant types
	_, err = client.UpdateConnector(ctx, &api.UpdateConnectorReq{
		Id:            connectorID,
		NewGrantTypes: &api.GrantTypes{GrantTypes: []string{"authorization_code", "refresh_token"}},
	})
	if err != nil {
		t.Fatalf("failed to update connector grant types: %v", err)
	}

	resp, err := client.ListConnectors(ctx, &api.ListConnectorReq{})
	if err != nil {
		t.Fatalf("failed to list connectors: %v", err)
	}
	for _, c := range resp.Connectors {
		if c.Id == connectorID {
			if !slices.Equal(c.GrantTypes, []string{"authorization_code", "refresh_token"}) {
				t.Fatalf("expected grant types [authorization_code refresh_token], got %v", c.GrantTypes)
			}
		}
	}

	// Clear grant types by passing empty GrantTypes message
	_, err = client.UpdateConnector(ctx, &api.UpdateConnectorReq{
		Id:            connectorID,
		NewGrantTypes: &api.GrantTypes{},
	})
	if err != nil {
		t.Fatalf("failed to clear connector grant types: %v", err)
	}

	resp, err = client.ListConnectors(ctx, &api.ListConnectorReq{})
	if err != nil {
		t.Fatalf("failed to list connectors: %v", err)
	}
	for _, c := range resp.Connectors {
		if c.Id == connectorID {
			if len(c.GrantTypes) != 0 {
				t.Fatalf("expected empty grant types after clear, got %v", c.GrantTypes)
			}
		}
	}

	// Reject invalid grant type on update
	_, err = client.UpdateConnector(ctx, &api.UpdateConnectorReq{
		Id:            connectorID,
		NewGrantTypes: &api.GrantTypes{GrantTypes: []string{"bogus"}},
	})
	if err == nil {
		t.Fatal("expected error for invalid grant type, got nil")
	}
	if !strings.Contains(err.Error(), `unknown grant type "bogus"`) {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reject invalid grant type on create
	_, err = client.CreateConnector(ctx, &api.CreateConnectorReq{
		Connector: &api.Connector{
			Id:         "bad-gt",
			Name:       "Bad",
			Type:       "TestType",
			Config:     []byte(`{}`),
			GrantTypes: []string{"invalid_type"},
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid grant type on create, got nil")
	}
	if !strings.Contains(err.Error(), `unknown grant type "invalid_type"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteConnector(t *testing.T) {
	t.Setenv("DEX_API_CONNECTORS_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	connectorID := "connector123"

	// Create a connector for testing
	createReq := api.CreateConnectorReq{
		Connector: &api.Connector{
			Id:     connectorID,
			Name:   "TestConnector",
			Type:   "TestType",
			Config: []byte(`{"key": "value"}`),
		},
	}
	client.CreateConnector(ctx, &createReq)

	deleteReq := api.DeleteConnectorReq{
		Id: connectorID,
	}

	// Test valid connector deletion
	if _, err := client.DeleteConnector(ctx, &deleteReq); err != nil {
		t.Fatalf("Unable to delete connector: %v", err)
	}

	// Test non existent connector deletion
	resp, err := client.DeleteConnector(ctx, &deleteReq)
	if err != nil {
		t.Fatalf("Unable to delete connector: %v", err)
	}

	if !resp.NotFound {
		t.Fatal("Should return not found")
	}
}

func TestListConnectors(t *testing.T) {
	t.Setenv("DEX_API_CONNECTORS_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	// Create connectors for testing
	createReq1 := api.CreateConnectorReq{
		Connector: &api.Connector{
			Id:     "connector1",
			Name:   "Connector1",
			Type:   "Type1",
			Config: []byte(`{"key": "value1"}`),
		},
	}
	client.CreateConnector(ctx, &createReq1)

	createReq2 := api.CreateConnectorReq{
		Connector: &api.Connector{
			Id:     "connector2",
			Name:   "Connector2",
			Type:   "Type2",
			Config: []byte(`{"key": "value2"}`),
		},
	}
	client.CreateConnector(ctx, &createReq2)

	listReq := api.ListConnectorReq{}

	// Test listing connectors
	if resp, err := client.ListConnectors(ctx, &listReq); err != nil {
		t.Fatalf("Unable to list connectors: %v", err)
	} else if len(resp.Connectors) != 2 { // Check the number of connectors in the response
		t.Fatalf("Expected 2 connectors, found %d", len(resp.Connectors))
	}
}

func TestMissingConnectorsCRUDFeatureFlag(t *testing.T) {
	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	// Create connectors for testing
	createReq1 := api.CreateConnectorReq{
		Connector: &api.Connector{
			Id:     "connector1",
			Name:   "Connector1",
			Type:   "Type1",
			Config: []byte(`{"key": "value1"}`),
		},
	}
	client.CreateConnector(ctx, &createReq1)

	createReq2 := api.CreateConnectorReq{
		Connector: &api.Connector{
			Id:     "connector2",
			Name:   "Connector2",
			Type:   "Type2",
			Config: []byte(`{"key": "value2"}`),
		},
	}
	client.CreateConnector(ctx, &createReq2)

	listReq := api.ListConnectorReq{}

	if _, err := client.ListConnectors(ctx, &listReq); err == nil {
		t.Fatal("ListConnectors should have returned an error")
	}
}

func TestListClients(t *testing.T) {
	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	// List Clients
	listResp, err := client.ListClients(ctx, &api.ListClientReq{})
	if err != nil {
		t.Fatalf("Unable to list clients: %v", err)
	}
	if len(listResp.Clients) != 0 {
		t.Fatalf("Expected 0 clients, got %d", len(listResp.Clients))
	}

	client1 := &api.Client{
		Id:           "client1",
		Secret:       "secret1",
		RedirectUris: []string{"http://localhost:8080/callback"},
		TrustedPeers: []string{"peer1"},
		Public:       false,
		Name:         "Test Client 1",
		LogoUrl:      "http://example.com/logo1.png",
	}

	client2 := &api.Client{
		Id:           "client2",
		Secret:       "secret2",
		RedirectUris: []string{"http://localhost:8081/callback"},
		TrustedPeers: []string{"peer2"},
		Public:       true,
		Name:         "Test Client 2",
		LogoUrl:      "http://example.com/logo2.png",
	}

	_, err = client.CreateClient(ctx, &api.CreateClientReq{Client: client1})
	if err != nil {
		t.Fatalf("Unable to create client1: %v", err)
	}

	_, err = client.CreateClient(ctx, &api.CreateClientReq{Client: client2})
	if err != nil {
		t.Fatalf("Unable to create client2: %v", err)
	}

	listResp, err = client.ListClients(ctx, &api.ListClientReq{})
	if err != nil {
		t.Fatalf("Unable to list clients: %v", err)
	}

	if len(listResp.Clients) != 2 {
		t.Fatalf("Expected 2 clients, got %d", len(listResp.Clients))
	}

	clientMap := make(map[string]*api.ClientInfo)
	for _, c := range listResp.Clients {
		clientMap[c.Id] = c
	}

	if c1, exists := clientMap["client1"]; !exists {
		t.Fatal("client1 not found in list")
	} else {
		if c1.Name != "Test Client 1" {
			t.Errorf("Expected client1 name 'Test Client 1', got '%s'", c1.Name)
		}
		if len(c1.RedirectUris) != 1 || c1.RedirectUris[0] != "http://localhost:8080/callback" {
			t.Errorf("Expected client1 redirect URIs ['http://localhost:8080/callback'], got %v", c1.RedirectUris)
		}
		if c1.Public != false {
			t.Errorf("Expected client1 public false, got %v", c1.Public)
		}
		if c1.LogoUrl != "http://example.com/logo1.png" {
			t.Errorf("Expected client1 logo URL 'http://example.com/logo1.png', got '%s'", c1.LogoUrl)
		}
	}

	if c2, exists := clientMap["client2"]; !exists {
		t.Fatal("client2 not found in list")
	} else {
		if c2.Name != "Test Client 2" {
			t.Errorf("Expected client2 name 'Test Client 2', got '%s'", c2.Name)
		}
		if len(c2.RedirectUris) != 1 || c2.RedirectUris[0] != "http://localhost:8081/callback" {
			t.Errorf("Expected client2 redirect URIs ['http://localhost:8081/callback'], got %v", c2.RedirectUris)
		}
		if c2.Public != true {
			t.Errorf("Expected client2 public true, got %v", c2.Public)
		}
		if c2.LogoUrl != "http://example.com/logo2.png" {
			t.Errorf("Expected client2 logo URL 'http://example.com/logo2.png', got '%s'", c2.LogoUrl)
		}
	}
}

func TestGetAuthSession(t *testing.T) {
	t.Setenv("DEX_API_SESSIONS_IDENTITIES_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	now := time.Now().UTC().Round(time.Second)
	session := storage.AuthSession{
		UserID:      "user1",
		ConnectorID: "conn1",
		Nonce:       "nonce123",
		ClientStates: map[string]*storage.ClientAuthState{
			"client-a": {
				Active:            true,
				ExpiresAt:         now.Add(24 * time.Hour),
				LastActivity:      now,
				LastTokenIssuedAt: now,
			},
		},
		CreatedAt:      now,
		LastActivity:   now,
		IPAddress:      "10.0.0.1",
		UserAgent:      "TestAgent/1.0",
		AbsoluteExpiry: now.Add(24 * time.Hour),
		IdleExpiry:     now.Add(1 * time.Hour),
	}

	if err := s.CreateAuthSession(ctx, session); err != nil {
		t.Fatalf("create auth session: %v", err)
	}

	resp, err := client.GetAuthSession(ctx, &api.GetAuthSessionReq{
		UserId:      "user1",
		ConnectorId: "conn1",
	})
	if err != nil {
		t.Fatalf("get auth session: %v", err)
	}

	if resp.Session.UserId != "user1" {
		t.Errorf("expected user_id 'user1', got '%s'", resp.Session.UserId)
	}
	if resp.Session.IpAddress != "10.0.0.1" {
		t.Errorf("expected ip_address '10.0.0.1', got '%s'", resp.Session.IpAddress)
	}
	if len(resp.Session.ClientStates) != 1 {
		t.Fatalf("expected 1 client state, got %d", len(resp.Session.ClientStates))
	}
	cs := resp.Session.ClientStates[0]
	if cs.ClientId != "client-a" {
		t.Errorf("expected client_id 'client-a', got '%s'", cs.ClientId)
	}
	if !cs.Active {
		t.Error("expected client state to be active")
	}

	// Not found case.
	_, err = client.GetAuthSession(ctx, &api.GetAuthSessionReq{
		UserId:      "nonexistent",
		ConnectorId: "conn1",
	})
	if err == nil {
		t.Fatal("expected error for non-existent session")
	}
}

func TestListAuthSessions(t *testing.T) {
	t.Setenv("DEX_API_SESSIONS_IDENTITIES_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	now := time.Now().UTC().Round(time.Second)
	for _, sess := range []storage.AuthSession{
		{UserID: "user1", ConnectorID: "conn1", Nonce: "n1", ClientStates: map[string]*storage.ClientAuthState{}, CreatedAt: now, LastActivity: now, AbsoluteExpiry: now.Add(time.Hour), IdleExpiry: now.Add(time.Hour)},
		{UserID: "user1", ConnectorID: "conn2", Nonce: "n2", ClientStates: map[string]*storage.ClientAuthState{}, CreatedAt: now, LastActivity: now, AbsoluteExpiry: now.Add(time.Hour), IdleExpiry: now.Add(time.Hour)},
		{UserID: "user2", ConnectorID: "conn1", Nonce: "n3", ClientStates: map[string]*storage.ClientAuthState{}, CreatedAt: now, LastActivity: now, AbsoluteExpiry: now.Add(time.Hour), IdleExpiry: now.Add(time.Hour)},
	} {
		if err := s.CreateAuthSession(ctx, sess); err != nil {
			t.Fatalf("create auth session: %v", err)
		}
	}

	// List all.
	resp, err := client.ListAuthSessions(ctx, &api.ListAuthSessionsReq{})
	if err != nil {
		t.Fatalf("list auth sessions: %v", err)
	}
	if len(resp.Sessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(resp.Sessions))
	}

	// Filter by user_id.
	resp, err = client.ListAuthSessions(ctx, &api.ListAuthSessionsReq{UserId: "user1"})
	if err != nil {
		t.Fatalf("list auth sessions with filter: %v", err)
	}
	if len(resp.Sessions) != 2 {
		t.Fatalf("expected 2 sessions for user1, got %d", len(resp.Sessions))
	}
}

func TestDeleteAuthSession(t *testing.T) {
	t.Setenv("DEX_API_SESSIONS_IDENTITIES_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	now := time.Now().UTC().Round(time.Second)

	// Create session.
	session := storage.AuthSession{
		UserID: "user1", ConnectorID: "conn1", Nonce: "n1",
		ClientStates:   map[string]*storage.ClientAuthState{},
		CreatedAt:      now,
		LastActivity:   now,
		AbsoluteExpiry: now.Add(time.Hour),
		IdleExpiry:     now.Add(time.Hour),
	}
	if err := s.CreateAuthSession(ctx, session); err != nil {
		t.Fatalf("create auth session: %v", err)
	}

	// Create refresh token + offline session to verify cascading revocation.
	refreshID := storage.NewID()
	if err := s.CreateRefresh(ctx, storage.RefreshToken{
		ID: refreshID, Token: "tok", Nonce: "n", ClientID: "client1", ConnectorID: "conn1",
		Scopes: []string{"openid"}, CreatedAt: now, LastUsed: now,
		Claims: storage.Claims{UserID: "user1", Username: "test", Email: "test@test.com"},
	}); err != nil {
		t.Fatalf("create refresh: %v", err)
	}
	if err := s.CreateOfflineSessions(ctx, storage.OfflineSessions{
		UserID: "user1", ConnID: "conn1",
		Refresh: map[string]*storage.RefreshTokenRef{
			"client1": {ID: refreshID, ClientID: "client1", CreatedAt: now, LastUsed: now},
		},
	}); err != nil {
		t.Fatalf("create offline sessions: %v", err)
	}

	// Delete session.
	resp, err := client.DeleteAuthSession(ctx, &api.DeleteAuthSessionReq{
		UserId: "user1", ConnectorId: "conn1",
	})
	if err != nil {
		t.Fatalf("delete auth session: %v", err)
	}
	if resp.NotFound {
		t.Error("expected session to be found")
	}

	// Verify session is gone.
	_, err = s.GetAuthSession(ctx, "user1", "conn1")
	if err == nil {
		t.Error("expected auth session to be deleted")
	}

	// Verify refresh token was revoked.
	_, err = s.GetRefresh(ctx, refreshID)
	if err == nil {
		t.Error("expected refresh token to be revoked")
	}

	// Not found case.
	resp, err = client.DeleteAuthSession(ctx, &api.DeleteAuthSessionReq{
		UserId: "user1", ConnectorId: "conn1",
	})
	if err != nil {
		t.Fatalf("delete auth session: %v", err)
	}
	if !resp.NotFound {
		t.Error("expected not_found for already deleted session")
	}
}

func TestTerminateSessionsByConnector(t *testing.T) {
	t.Setenv("DEX_API_SESSIONS_IDENTITIES_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	now := time.Now().UTC().Round(time.Second)
	for _, sess := range []storage.AuthSession{
		{UserID: "user1", ConnectorID: "target-conn", Nonce: "n1", ClientStates: map[string]*storage.ClientAuthState{}, CreatedAt: now, LastActivity: now, AbsoluteExpiry: now.Add(time.Hour), IdleExpiry: now.Add(time.Hour)},
		{UserID: "user2", ConnectorID: "target-conn", Nonce: "n2", ClientStates: map[string]*storage.ClientAuthState{}, CreatedAt: now, LastActivity: now, AbsoluteExpiry: now.Add(time.Hour), IdleExpiry: now.Add(time.Hour)},
		{UserID: "user3", ConnectorID: "other-conn", Nonce: "n3", ClientStates: map[string]*storage.ClientAuthState{}, CreatedAt: now, LastActivity: now, AbsoluteExpiry: now.Add(time.Hour), IdleExpiry: now.Add(time.Hour)},
	} {
		if err := s.CreateAuthSession(ctx, sess); err != nil {
			t.Fatalf("create auth session: %v", err)
		}
	}

	resp, err := client.TerminateSessionsByConnector(ctx, &api.TerminateSessionsByConnectorReq{
		ConnectorId: "target-conn",
	})
	if err != nil {
		t.Fatalf("terminate sessions by connector: %v", err)
	}
	if resp.SessionsTerminated != 2 {
		t.Errorf("expected 2 terminated, got %d", resp.SessionsTerminated)
	}

	// Verify remaining session is untouched.
	remaining, err := s.ListAuthSessions(ctx)
	if err != nil {
		t.Fatalf("list auth sessions: %v", err)
	}
	if len(remaining) != 1 || remaining[0].ConnectorID != "other-conn" {
		t.Errorf("expected only other-conn session to remain, got %v", remaining)
	}
}

func TestTerminateSessionsByUser(t *testing.T) {
	t.Setenv("DEX_API_SESSIONS_IDENTITIES_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	now := time.Now().UTC().Round(time.Second)
	for _, sess := range []storage.AuthSession{
		{UserID: "target-user", ConnectorID: "conn1", Nonce: "n1", ClientStates: map[string]*storage.ClientAuthState{}, CreatedAt: now, LastActivity: now, AbsoluteExpiry: now.Add(time.Hour), IdleExpiry: now.Add(time.Hour)},
		{UserID: "target-user", ConnectorID: "conn2", Nonce: "n2", ClientStates: map[string]*storage.ClientAuthState{}, CreatedAt: now, LastActivity: now, AbsoluteExpiry: now.Add(time.Hour), IdleExpiry: now.Add(time.Hour)},
		{UserID: "other-user", ConnectorID: "conn1", Nonce: "n3", ClientStates: map[string]*storage.ClientAuthState{}, CreatedAt: now, LastActivity: now, AbsoluteExpiry: now.Add(time.Hour), IdleExpiry: now.Add(time.Hour)},
	} {
		if err := s.CreateAuthSession(ctx, sess); err != nil {
			t.Fatalf("create auth session: %v", err)
		}
	}

	resp, err := client.TerminateSessionsByUser(ctx, &api.TerminateSessionsByUserReq{
		UserId: "target-user",
	})
	if err != nil {
		t.Fatalf("terminate sessions by user: %v", err)
	}
	if resp.SessionsTerminated != 2 {
		t.Errorf("expected 2 terminated, got %d", resp.SessionsTerminated)
	}

	remaining, err := s.ListAuthSessions(ctx)
	if err != nil {
		t.Fatalf("list auth sessions: %v", err)
	}
	if len(remaining) != 1 || remaining[0].UserID != "other-user" {
		t.Errorf("expected only other-user session to remain")
	}
}

func TestGetUserIdentity(t *testing.T) {
	t.Setenv("DEX_API_SESSIONS_IDENTITIES_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	now := time.Now().UTC().Round(time.Second)
	identity := storage.UserIdentity{
		UserID:      "user1",
		ConnectorID: "conn1",
		Claims: storage.Claims{
			UserID:        "user1",
			Username:      "testuser",
			Email:         "test@example.com",
			EmailVerified: true,
			Groups:        []string{"admins"},
		},
		Consents: map[string][]string{
			"client-a": {"openid", "email"},
		},
		MFASecrets: map[string]*storage.MFASecret{
			"totp-1": {AuthenticatorID: "totp-1", Type: "TOTP", Secret: "secret123", Confirmed: true, CreatedAt: now},
		},
		WebAuthnCredentials: map[string][]storage.WebAuthnCredential{
			"webauthn-1": {{CredentialID: []byte("cred1"), AttestationType: "none", DisplayName: "YubiKey", CreatedAt: now}},
		},
		CreatedAt: now,
		LastLogin: now,
	}

	if err := s.CreateUserIdentity(ctx, identity); err != nil {
		t.Fatalf("create user identity: %v", err)
	}

	resp, err := client.GetUserIdentity(ctx, &api.GetUserIdentityReq{
		UserId: "user1", ConnectorId: "conn1",
	})
	if err != nil {
		t.Fatalf("get user identity: %v", err)
	}

	if resp.Identity.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", resp.Identity.Email)
	}
	if !resp.Identity.EmailVerified {
		t.Error("expected email_verified true")
	}
	if resp.Identity.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", resp.Identity.Username)
	}
	if len(resp.Identity.Consents) != 1 {
		t.Fatalf("expected 1 consent entry, got %d", len(resp.Identity.Consents))
	}
	if len(resp.Identity.MfaDevices) == 0 {
		t.Fatal("expected MFA devices")
	}
}

func TestListUserIdentities(t *testing.T) {
	t.Setenv("DEX_API_SESSIONS_IDENTITIES_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	now := time.Now().UTC().Round(time.Second)
	for _, id := range []storage.UserIdentity{
		{UserID: "user1", ConnectorID: "conn1", Claims: storage.Claims{Email: "a@test.com"}, CreatedAt: now, LastLogin: now},
		{UserID: "user2", ConnectorID: "conn1", Claims: storage.Claims{Email: "b@test.com"}, CreatedAt: now, LastLogin: now},
	} {
		if err := s.CreateUserIdentity(ctx, id); err != nil {
			t.Fatalf("create user identity: %v", err)
		}
	}

	resp, err := client.ListUserIdentities(ctx, &api.ListUserIdentitiesReq{})
	if err != nil {
		t.Fatalf("list user identities: %v", err)
	}
	if len(resp.Identities) != 2 {
		t.Fatalf("expected 2 identities, got %d", len(resp.Identities))
	}
}

func TestDeleteUserIdentity(t *testing.T) {
	t.Setenv("DEX_API_SESSIONS_IDENTITIES_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	now := time.Now().UTC().Round(time.Second)

	// Create identity + session + offline sessions + refresh token.
	if err := s.CreateUserIdentity(ctx, storage.UserIdentity{
		UserID: "user1", ConnectorID: "conn1",
		Claims: storage.Claims{Email: "test@test.com"}, CreatedAt: now, LastLogin: now,
	}); err != nil {
		t.Fatalf("create user identity: %v", err)
	}
	if err := s.CreateAuthSession(ctx, storage.AuthSession{
		UserID: "user1", ConnectorID: "conn1", Nonce: "n",
		ClientStates: map[string]*storage.ClientAuthState{}, CreatedAt: now, LastActivity: now,
		AbsoluteExpiry: now.Add(time.Hour), IdleExpiry: now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("create auth session: %v", err)
	}
	refreshID := storage.NewID()
	if err := s.CreateRefresh(ctx, storage.RefreshToken{
		ID: refreshID, Token: "tok", Nonce: "n", ClientID: "c1", ConnectorID: "conn1",
		Scopes: []string{"openid"}, CreatedAt: now, LastUsed: now,
		Claims: storage.Claims{UserID: "user1", Email: "test@test.com"},
	}); err != nil {
		t.Fatalf("create refresh: %v", err)
	}
	if err := s.CreateOfflineSessions(ctx, storage.OfflineSessions{
		UserID: "user1", ConnID: "conn1",
		Refresh: map[string]*storage.RefreshTokenRef{
			"c1": {ID: refreshID, ClientID: "c1", CreatedAt: now, LastUsed: now},
		},
	}); err != nil {
		t.Fatalf("create offline sessions: %v", err)
	}
	// Password record linked by the identity's email — must be purged too (GDPR).
	if err := s.CreatePassword(ctx, storage.Password{
		Email: "test@test.com", Hash: []byte("$2y$10$XXXXXXXXXXXXXXXXXXXXXX"), Username: "test", UserID: "user1",
	}); err != nil {
		t.Fatalf("create password: %v", err)
	}

	// Delete identity (cascading).
	resp, err := client.DeleteUserIdentity(ctx, &api.DeleteUserIdentityReq{
		UserId: "user1", ConnectorId: "conn1",
	})
	if err != nil {
		t.Fatalf("delete user identity: %v", err)
	}
	if resp.NotFound {
		t.Error("expected identity to be found")
	}

	// Verify everything is deleted.
	if _, err := s.GetUserIdentity(ctx, "user1", "conn1"); err == nil {
		t.Error("expected user identity to be deleted")
	}
	if _, err := s.GetAuthSession(ctx, "user1", "conn1"); err == nil {
		t.Error("expected auth session to be deleted")
	}
	if _, err := s.GetRefresh(ctx, refreshID); err == nil {
		t.Error("expected refresh token to be deleted")
	}
	if _, err := s.GetOfflineSessions(ctx, "user1", "conn1"); err == nil {
		t.Error("expected offline sessions to be deleted")
	}
	if _, err := s.GetPassword(ctx, "test@test.com"); err == nil {
		t.Error("expected password record to be deleted")
	}

	// Not found case.
	resp, err = client.DeleteUserIdentity(ctx, &api.DeleteUserIdentityReq{
		UserId: "user1", ConnectorId: "conn1",
	})
	if err != nil {
		t.Fatalf("delete user identity: %v", err)
	}
	if !resp.NotFound {
		t.Error("expected not_found for already deleted identity")
	}
}

func TestResetMFA(t *testing.T) {
	t.Setenv("DEX_API_SESSIONS_IDENTITIES_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	now := time.Now().UTC().Round(time.Second)
	if err := s.CreateUserIdentity(ctx, storage.UserIdentity{
		UserID: "user1", ConnectorID: "conn1",
		Claims: storage.Claims{Email: "test@test.com"}, CreatedAt: now, LastLogin: now,
		MFASecrets: map[string]*storage.MFASecret{
			"totp-1": {AuthenticatorID: "totp-1", Type: "TOTP", Secret: "s", Confirmed: true, CreatedAt: now},
		},
		WebAuthnCredentials: map[string][]storage.WebAuthnCredential{
			"webauthn-1": {{CredentialID: []byte("c1"), CreatedAt: now}},
		},
	}); err != nil {
		t.Fatalf("create user identity: %v", err)
	}

	resp, err := client.ResetMFA(ctx, &api.ResetMFAReq{
		UserId: "user1", ConnectorId: "conn1",
	})
	if err != nil {
		t.Fatalf("reset MFA: %v", err)
	}
	if resp.NotFound {
		t.Error("expected identity to be found")
	}

	// Verify MFA data is cleared.
	identity, err := s.GetUserIdentity(ctx, "user1", "conn1")
	if err != nil {
		t.Fatalf("get user identity: %v", err)
	}
	if len(identity.MFASecrets) != 0 {
		t.Errorf("expected MFASecrets to be cleared, got %d", len(identity.MFASecrets))
	}
	if len(identity.WebAuthnCredentials) != 0 {
		t.Errorf("expected WebAuthnCredentials to be cleared, got %d", len(identity.WebAuthnCredentials))
	}
	// Verify other fields are preserved.
	if identity.Claims.Email != "test@test.com" {
		t.Errorf("expected email to be preserved, got '%s'", identity.Claims.Email)
	}
}

func TestListMFADevices(t *testing.T) {
	t.Setenv("DEX_API_SESSIONS_IDENTITIES_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	now := time.Now().UTC().Round(time.Second)
	if err := s.CreateUserIdentity(ctx, storage.UserIdentity{
		UserID: "user1", ConnectorID: "conn1",
		Claims: storage.Claims{Email: "test@test.com"}, CreatedAt: now, LastLogin: now,
		MFASecrets: map[string]*storage.MFASecret{
			"totp-1": {AuthenticatorID: "totp-1", Type: "TOTP", Secret: "secret123", Confirmed: true, CreatedAt: now},
		},
		WebAuthnCredentials: map[string][]storage.WebAuthnCredential{
			"webauthn-1": {
				{CredentialID: []byte("cred1"), PublicKey: []byte("pk1"), DisplayName: "Key1", CreatedAt: now},
				{CredentialID: []byte("cred2"), PublicKey: []byte("pk2"), DisplayName: "Key2", CreatedAt: now},
			},
		},
	}); err != nil {
		t.Fatalf("create user identity: %v", err)
	}

	resp, err := client.ListMFADevices(ctx, &api.ListMFADevicesReq{
		UserId: "user1", ConnectorId: "conn1",
	})
	if err != nil {
		t.Fatalf("list MFA devices: %v", err)
	}
	if len(resp.Devices) != 2 {
		t.Fatalf("expected 2 device groups, got %d", len(resp.Devices))
	}

	// Find the TOTP device and verify secret is not exposed.
	for _, device := range resp.Devices {
		if device.AuthenticatorId == "totp-1" {
			if device.MfaSecret == nil {
				t.Fatal("expected MFA secret for totp-1")
			}
			if device.MfaSecret.Type != "TOTP" {
				t.Errorf("expected type TOTP, got %s", device.MfaSecret.Type)
			}
		}
		if device.AuthenticatorId == "webauthn-1" {
			if len(device.WebauthnCredentials) != 2 {
				t.Errorf("expected 2 webauthn credentials, got %d", len(device.WebauthnCredentials))
			}
		}
	}
}

func TestDeleteWebAuthnCredential(t *testing.T) {
	t.Setenv("DEX_API_SESSIONS_IDENTITIES_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	now := time.Now().UTC().Round(time.Second)
	if err := s.CreateUserIdentity(ctx, storage.UserIdentity{
		UserID: "user1", ConnectorID: "conn1",
		Claims: storage.Claims{Email: "test@test.com"}, CreatedAt: now, LastLogin: now,
		WebAuthnCredentials: map[string][]storage.WebAuthnCredential{
			"auth-1": {
				{CredentialID: []byte("cred-to-delete"), DisplayName: "Key1", CreatedAt: now},
				{CredentialID: []byte("cred-to-keep"), DisplayName: "Key2", CreatedAt: now},
			},
		},
	}); err != nil {
		t.Fatalf("create user identity: %v", err)
	}

	// Delete one credential.
	resp, err := client.DeleteWebAuthnCredential(ctx, &api.DeleteWebAuthnCredentialReq{
		UserId: "user1", ConnectorId: "conn1", CredentialId: []byte("cred-to-delete"),
	})
	if err != nil {
		t.Fatalf("delete webauthn credential: %v", err)
	}
	if resp.NotFound {
		t.Error("expected credential to be found")
	}

	// Verify only one credential remains.
	identity, err := s.GetUserIdentity(ctx, "user1", "conn1")
	if err != nil {
		t.Fatalf("get user identity: %v", err)
	}
	creds := identity.WebAuthnCredentials["auth-1"]
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential remaining, got %d", len(creds))
	}
	if !bytes.Equal(creds[0].CredentialID, []byte("cred-to-keep")) {
		t.Error("wrong credential was deleted")
	}

	// Not found case.
	resp, err = client.DeleteWebAuthnCredential(ctx, &api.DeleteWebAuthnCredentialReq{
		UserId: "user1", ConnectorId: "conn1", CredentialId: []byte("nonexistent"),
	})
	if err != nil {
		t.Fatalf("delete webauthn credential: %v", err)
	}
	if !resp.NotFound {
		t.Error("expected not_found for nonexistent credential")
	}
}

func TestDeleteMFASecret(t *testing.T) {
	t.Setenv("DEX_API_SESSIONS_IDENTITIES_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	now := time.Now().UTC().Round(time.Second)
	if err := s.CreateUserIdentity(ctx, storage.UserIdentity{
		UserID: "user1", ConnectorID: "conn1",
		Claims: storage.Claims{Email: "test@test.com"}, CreatedAt: now, LastLogin: now,
		MFASecrets: map[string]*storage.MFASecret{
			"totp-1": {AuthenticatorID: "totp-1", Type: "TOTP", Secret: "s", Confirmed: true, CreatedAt: now},
			"totp-2": {AuthenticatorID: "totp-2", Type: "TOTP", Secret: "s2", Confirmed: true, CreatedAt: now},
		},
		WebAuthnCredentials: map[string][]storage.WebAuthnCredential{
			"totp-1": {{CredentialID: []byte("c1"), CreatedAt: now}},
		},
	}); err != nil {
		t.Fatalf("create user identity: %v", err)
	}

	// Delete totp-1 (should also remove associated webauthn credentials).
	resp, err := client.DeleteMFASecret(ctx, &api.DeleteMFASecretReq{
		UserId: "user1", ConnectorId: "conn1", AuthenticatorId: "totp-1",
	})
	if err != nil {
		t.Fatalf("delete MFA secret: %v", err)
	}
	if resp.NotFound {
		t.Error("expected authenticator to be found")
	}

	identity, err := s.GetUserIdentity(ctx, "user1", "conn1")
	if err != nil {
		t.Fatalf("get user identity: %v", err)
	}
	if _, ok := identity.MFASecrets["totp-1"]; ok {
		t.Error("expected totp-1 to be deleted")
	}
	if _, ok := identity.MFASecrets["totp-2"]; !ok {
		t.Error("expected totp-2 to remain")
	}
	if _, ok := identity.WebAuthnCredentials["totp-1"]; ok {
		t.Error("expected webauthn credentials for totp-1 to be deleted")
	}

	// Not found case.
	resp, err = client.DeleteMFASecret(ctx, &api.DeleteMFASecretReq{
		UserId: "user1", ConnectorId: "conn1", AuthenticatorId: "nonexistent",
	})
	if err != nil {
		t.Fatalf("delete MFA secret: %v", err)
	}
	if !resp.NotFound {
		t.Error("expected not_found for nonexistent authenticator")
	}
}

func TestRevokeConsent(t *testing.T) {
	t.Setenv("DEX_API_SESSIONS_IDENTITIES_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	now := time.Now().UTC().Round(time.Second)
	if err := s.CreateUserIdentity(ctx, storage.UserIdentity{
		UserID: "user1", ConnectorID: "conn1",
		Claims: storage.Claims{Email: "test@test.com"}, CreatedAt: now, LastLogin: now,
		Consents: map[string][]string{
			"client-a": {"openid", "email"},
			"client-b": {"openid"},
		},
	}); err != nil {
		t.Fatalf("create user identity: %v", err)
	}

	// Revoke consent for client-a.
	resp, err := client.RevokeConsent(ctx, &api.RevokeConsentReq{
		UserId: "user1", ConnectorId: "conn1", ClientId: "client-a",
	})
	if err != nil {
		t.Fatalf("revoke consent: %v", err)
	}
	if resp.NotFound {
		t.Error("expected consent to be found")
	}

	// Verify only client-b consent remains.
	identity, err := s.GetUserIdentity(ctx, "user1", "conn1")
	if err != nil {
		t.Fatalf("get user identity: %v", err)
	}
	if _, ok := identity.Consents["client-a"]; ok {
		t.Error("expected client-a consent to be revoked")
	}
	if _, ok := identity.Consents["client-b"]; !ok {
		t.Error("expected client-b consent to remain")
	}

	// Not found case.
	resp, err = client.RevokeConsent(ctx, &api.RevokeConsentReq{
		UserId: "user1", ConnectorId: "conn1", ClientId: "nonexistent",
	})
	if err != nil {
		t.Fatalf("revoke consent: %v", err)
	}
	if !resp.NotFound {
		t.Error("expected not_found for nonexistent consent")
	}
}

func TestMissingSessionsIdentitiesCRUDFeatureFlag(t *testing.T) {
	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	if _, err := client.GetAuthSession(ctx, &api.GetAuthSessionReq{UserId: "u", ConnectorId: "c"}); err == nil {
		t.Error("GetAuthSession should fail without feature flag")
	}
	if _, err := client.ListAuthSessions(ctx, &api.ListAuthSessionsReq{}); err == nil {
		t.Error("ListAuthSessions should fail without feature flag")
	}
	if _, err := client.DeleteAuthSession(ctx, &api.DeleteAuthSessionReq{UserId: "u", ConnectorId: "c"}); err == nil {
		t.Error("DeleteAuthSession should fail without feature flag")
	}
	if _, err := client.TerminateSessionsByConnector(ctx, &api.TerminateSessionsByConnectorReq{ConnectorId: "c"}); err == nil {
		t.Error("TerminateSessionsByConnector should fail without feature flag")
	}
	if _, err := client.TerminateSessionsByUser(ctx, &api.TerminateSessionsByUserReq{UserId: "u"}); err == nil {
		t.Error("TerminateSessionsByUser should fail without feature flag")
	}
	if _, err := client.GetUserIdentity(ctx, &api.GetUserIdentityReq{UserId: "u", ConnectorId: "c"}); err == nil {
		t.Error("GetUserIdentity should fail without feature flag")
	}
	if _, err := client.ListUserIdentities(ctx, &api.ListUserIdentitiesReq{}); err == nil {
		t.Error("ListUserIdentities should fail without feature flag")
	}
	if _, err := client.DeleteUserIdentity(ctx, &api.DeleteUserIdentityReq{UserId: "u", ConnectorId: "c"}); err == nil {
		t.Error("DeleteUserIdentity should fail without feature flag")
	}
	if _, err := client.ResetMFA(ctx, &api.ResetMFAReq{UserId: "u", ConnectorId: "c"}); err == nil {
		t.Error("ResetMFA should fail without feature flag")
	}
	if _, err := client.ListMFADevices(ctx, &api.ListMFADevicesReq{UserId: "u", ConnectorId: "c"}); err == nil {
		t.Error("ListMFADevices should fail without feature flag")
	}
	if _, err := client.DeleteWebAuthnCredential(ctx, &api.DeleteWebAuthnCredentialReq{UserId: "u", ConnectorId: "c", CredentialId: []byte("cred")}); err == nil {
		t.Error("DeleteWebAuthnCredential should fail without feature flag")
	}
	if _, err := client.DeleteMFASecret(ctx, &api.DeleteMFASecretReq{UserId: "u", ConnectorId: "c", AuthenticatorId: "a"}); err == nil {
		t.Error("DeleteMFASecret should fail without feature flag")
	}
	if _, err := client.RevokeConsent(ctx, &api.RevokeConsentReq{UserId: "u", ConnectorId: "c", ClientId: "cl"}); err == nil {
		t.Error("RevokeConsent should fail without feature flag")
	}
}

// TestSessionsIdentitiesZeroTimeConversion verifies that unset time.Time fields
// serialize to 0 rather than the misleading year-1 epoch (-62135596800) that a
// naive t.Unix() produces.
func TestSessionsIdentitiesZeroTimeConversion(t *testing.T) {
	t.Setenv("DEX_API_SESSIONS_IDENTITIES_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	now := time.Now().UTC().Round(time.Second)

	// Client authenticated but no token issued yet: LastTokenIssuedAt is zero.
	if err := s.CreateAuthSession(ctx, storage.AuthSession{
		UserID: "user1", ConnectorID: "conn1", Nonce: "n",
		ClientStates: map[string]*storage.ClientAuthState{
			"client-a": {Active: true, ExpiresAt: now.Add(time.Hour), LastActivity: now},
		},
		CreatedAt: now, LastActivity: now, AbsoluteExpiry: now.Add(time.Hour), IdleExpiry: now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("create auth session: %v", err)
	}

	sessResp, err := client.GetAuthSession(ctx, &api.GetAuthSessionReq{UserId: "user1", ConnectorId: "conn1"})
	if err != nil {
		t.Fatalf("get auth session: %v", err)
	}
	if len(sessResp.Session.ClientStates) != 1 {
		t.Fatalf("expected 1 client state, got %d", len(sessResp.Session.ClientStates))
	}
	if got := sessResp.Session.ClientStates[0].LastTokenIssuedAt; got != 0 {
		t.Errorf("expected last_token_issued_at 0 for unset time, got %d", got)
	}

	// Identity that has never logged in and is not blocked: LastLogin and
	// BlockedUntil are zero.
	if err := s.CreateUserIdentity(ctx, storage.UserIdentity{
		UserID: "user1", ConnectorID: "conn1",
		Claims: storage.Claims{Email: "test@test.com"}, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create user identity: %v", err)
	}

	idResp, err := client.GetUserIdentity(ctx, &api.GetUserIdentityReq{UserId: "user1", ConnectorId: "conn1"})
	if err != nil {
		t.Fatalf("get user identity: %v", err)
	}
	if got := idResp.Identity.LastLogin; got != 0 {
		t.Errorf("expected last_login 0 for unset time, got %d", got)
	}
	if got := idResp.Identity.BlockedUntil; got != 0 {
		t.Errorf("expected blocked_until 0 for unset time, got %d", got)
	}
}

// TestSessionsIdentitiesValidation verifies that handlers reject requests
// missing required fields.
func TestSessionsIdentitiesValidation(t *testing.T) {
	t.Setenv("DEX_API_SESSIONS_IDENTITIES_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	client := newAPI(t, s, logger)
	defer client.Close()

	ctx := t.Context()

	if _, err := client.GetAuthSession(ctx, &api.GetAuthSessionReq{ConnectorId: "c"}); err == nil {
		t.Error("GetAuthSession should reject empty user_id")
	}
	if _, err := client.GetAuthSession(ctx, &api.GetAuthSessionReq{UserId: "u"}); err == nil {
		t.Error("GetAuthSession should reject empty connector_id")
	}
	if _, err := client.TerminateSessionsByConnector(ctx, &api.TerminateSessionsByConnectorReq{}); err == nil {
		t.Error("TerminateSessionsByConnector should reject empty connector_id")
	}
	if _, err := client.TerminateSessionsByUser(ctx, &api.TerminateSessionsByUserReq{}); err == nil {
		t.Error("TerminateSessionsByUser should reject empty user_id")
	}
	if _, err := client.DeleteWebAuthnCredential(ctx, &api.DeleteWebAuthnCredentialReq{UserId: "u", ConnectorId: "c"}); err == nil {
		t.Error("DeleteWebAuthnCredential should reject empty credential_id")
	}
	if _, err := client.DeleteMFASecret(ctx, &api.DeleteMFASecretReq{UserId: "u", ConnectorId: "c"}); err == nil {
		t.Error("DeleteMFASecret should reject empty authenticator_id")
	}
	if _, err := client.RevokeConsent(ctx, &api.RevokeConsentReq{UserId: "u", ConnectorId: "c"}); err == nil {
		t.Error("RevokeConsent should reject empty client_id")
	}
}
