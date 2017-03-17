package server

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/dex/api"
	"github.com/coreos/dex/server/internal"
	"github.com/coreos/dex/storage"
	"github.com/coreos/dex/storage/memory"
)

// Attempts to create, update and delete a test Password
func TestPassword(t *testing.T) {
	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: &logrus.TextFormatter{DisableColors: true},
		Level:     logrus.DebugLevel,
	}

	s := memory.New(logger)
	serv := NewAPI(s, logger)

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

	if resp, err := serv.CreatePassword(ctx, &createReq); err != nil || resp.AlreadyExists {
		if resp.AlreadyExists {
			t.Fatalf("Unable to create password since %s already exists", createReq.Password.Email)
		}
		t.Fatalf("Unable to create password: %v", err)
	}

	// Attempt to create a password that already exists.
	if resp, _ := serv.CreatePassword(ctx, &createReq); !resp.AlreadyExists {
		t.Fatalf("Created password %s twice", createReq.Password.Email)
	}

	updateReq := api.UpdatePasswordReq{
		Email:       "test@example.com",
		NewUsername: "test1",
	}

	if _, err := serv.UpdatePassword(ctx, &updateReq); err != nil {
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

	if _, err := serv.DeletePassword(ctx, &deleteReq); err != nil {
		t.Fatalf("Unable to delete password: %v", err)
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
	serv := NewAPI(s, logger)

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

	listResp, err := serv.ListRefresh(ctx, &listReq)
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

	resp, err := serv.RevokeRefresh(ctx, &revokeReq)
	if err != nil || resp.NotFound {
		t.Fatalf("Unable to revoke refresh tokens for user: %v", err)
	}

	if resp, _ := serv.ListRefresh(ctx, &listReq); len(resp.RefreshTokens) != 0 {
		t.Fatalf("Refresh token returned inspite of revoking it.")
	}
}
