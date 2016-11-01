package server

import (
	"context"
	"testing"

	"github.com/coreos/dex/api"
	"github.com/coreos/dex/storage/memory"
)

// Attempts to create, update and delete a test Password
func TestPassword(t *testing.T) {
	s := memory.New()
	serv := NewAPI(s)

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

	if _, err := serv.CreatePassword(ctx, &createReq); err != nil {
		t.Fatalf("Unable to create password: %v", err)
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
