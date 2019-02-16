package server

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/dexidp/dex/api"
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
		t.Fatalf("Refresh token returned inspite of revoking it.")
	}
}

func TestUpdateClient(t *testing.T) {
	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: &logrus.TextFormatter{DisableColors: true},
		Level:     logrus.DebugLevel,
	}

	s := memory.New(logger)
	client := newAPI(s, logger, t)
	defer client.Close()
	ctx := context.Background()

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

				client, err := s.GetClient(tc.req.Id)
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
					found := find(redirectURI, client.RedirectURIs)
					if !found {
						t.Errorf("expected redirect URI: %s", redirectURI)
					}
				}
				for _, peer := range tc.req.TrustedPeers {
					found := find(peer, client.TrustedPeers)
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

func find(item string, items []string) bool {
	for _, i := range items {
		if item == i {
			return true
		}
	}
	return false
}
