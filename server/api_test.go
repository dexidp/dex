package server

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/coreos/dex/api"
	"github.com/coreos/dex/server/internal"
	"github.com/coreos/dex/storage"
	"github.com/coreos/dex/storage/memory"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// apiClient is a test gRPC client. When constructed, it runs a server in
// the background to exercise the serialization and network configuration
// instead of just this package's server implementation.
type apiClient struct {
	// Embedded gRPC client to talk to the server.
	api.DexClient
	// Close releases resources associated with this client, includuing shutting
	// down the background server.
	Close func()
}

// newAPI constructs a gRCP client connected to a backing server.
func newAPI(s storage.Storage, logger logrus.FieldLogger, t *testing.T) *apiClient {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	serv := grpc.NewServer()
	api.RegisterDexServer(serv, NewAPI(s, logger))
	go serv.Serve(l)

	// Dial will retry automatically if the serv.Serve() goroutine
	// hasn't started yet.
	conn, err := grpc.Dial(l.Addr().String(), grpc.WithInsecure())
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
	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: &logrus.TextFormatter{DisableColors: true},
		Level:     logrus.DebugLevel,
	}

	s := memory.New(logger)
	client := newAPI(s, logger, t)
	defer client.Close()

	ctx := context.Background()
	p := api.Password{
		Email: "test@example.com",
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

	updateReq := api.UpdatePasswordReq{
		Email:       "test@example.com",
		NewUsername: "test1",
	}

	if _, err := client.UpdatePassword(ctx, &updateReq); err != nil {
		t.Fatalf("Unable to update password: %v", err)
	}

	pass, err := s.GetPassword(updateReq.Email)
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
	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: &logrus.TextFormatter{DisableColors: true},
		Level:     logrus.DebugLevel,
	}

	s := memory.New(logger)
	client := newAPI(s, logger, t)
	defer client.Close()

	tests := []struct {
		name         string
		inputHash    []byte
		expectedCost int
		wantErr      bool
	}{
		{
			name: "valid cost",
			// bcrypt hash of the value "test1" with cost 12
			inputHash:    []byte("$2a$12$M2Ot95Qty1MuQdubh1acWOiYadJDzeVg3ve4n5b.dgcgPdjCseKx2"),
			expectedCost: recCost,
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
			// bcrypt hash of the value "test1" with cost 20
			inputHash:    []byte("$2a$20$yODn5quqK9MZdePqYLs6Y.Jr4cOO1P0aXsKz0eTa2rxOmu8e7ETpi"),
			expectedCost: 20,
		},
	}

	for _, tc := range tests {
		cost, err := checkCost(tc.inputHash)
		if err != nil {
			if !tc.wantErr {
				t.Errorf("%s: %s", tc.name, err)
			}
			continue
		}

		if tc.wantErr {
			t.Errorf("%s: expected err", tc.name)
			continue
		}

		if cost != tc.expectedCost {
			t.Errorf("%s: exepcted cost = %d but got cost = %d", tc.name, tc.expectedCost, cost)
		}
	}
}

// Attempts to list and revoke an exisiting refresh token.
func TestRefreshToken(t *testing.T) {
	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: &logrus.TextFormatter{DisableColors: true},
		Level:     logrus.DebugLevel,
	}

	s := memory.New(logger)
	client := newAPI(s, logger, t)
	defer client.Close()

	ctx := context.Background()

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

	if err := s.CreateRefresh(r); err != nil {
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

	if err := s.CreateOfflineSessions(session); err != nil {
		t.Fatalf("create offline session: %v", err)
	}

	subjectString, err := internal.Marshal(&internal.IDTokenSubject{
		UserId: r.Claims.UserID,
		ConnId: r.ConnectorID,
	})
	if err != nil {
		t.Errorf("failed to marshal offline session ID: %v", err)
	}

	//Testing the api.
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
	if err != nil || resp.NotFound {
		t.Fatalf("Unable to revoke refresh tokens for user: %v", err)
	}

	if resp, _ := client.ListRefresh(ctx, &listReq); len(resp.RefreshTokens) != 0 {
		t.Fatalf("Refresh token returned inspite of revoking it.")
	}
}

// Attempts to create, update and delete a test Connector
func TestConnector(t *testing.T) {
	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: &logrus.TextFormatter{DisableColors: true},
		Level:     logrus.DebugLevel,
	}

	s := memory.New(logger)
	serv := NewAPI(s, logger)

	ctx := context.Background()
	id := storage.NewID()
	c := api.Connector{
		Id:     id,
		Type:   "oidc",
		Name:   "Test",
		Config: []byte(`{"issuer": "https://accounts.google.com"}`),
	}

	createReq := api.CreateConnectorReq{
		Connector: &c,
	}

	if resp, err := serv.CreateConnector(ctx, &createReq); err != nil || resp.AlreadyExists {
		if resp.AlreadyExists {
			t.Fatalf("Unable to create connector since %s already exists", createReq.Connector.Id)
		}
		t.Fatalf("Unable to create connector: %v", err)
	}

	// Attempt to create a connector that already exists.
	if resp, _ := serv.CreateConnector(ctx, &createReq); !resp.AlreadyExists {
		t.Fatalf("Created connector %s twice", createReq.Connector.Id)
	}

	updateReq := api.UpdateConnectorReq{
		Id:   id,
		Name: "Test1",
	}

	if _, err := serv.UpdateConnector(ctx, &updateReq); err != nil {
		t.Fatalf("Unable to update connector: %v", err)
	}

	conn, err := s.GetConnector(id)
	if err != nil {
		t.Fatalf("Unable to retrieve connector: %v", err)
	}

	if conn.Name != updateReq.Name {
		t.Fatalf("UpdateConnector failed. Expected name %s retrieved %s", updateReq.Name, conn.Name)
	}

	deleteReq := api.DeleteConnectorReq{
		Id: id,
	}

	if _, err := serv.DeleteConnector(ctx, &deleteReq); err != nil {
		t.Fatalf("Unable to delete connector: %v", err)
	}
}
