package repo

import (
	"encoding/base64"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/coreos/go-oidc/oidc"
	"github.com/go-gorp/gorp"
	"github.com/kylelemons/godebug/pretty"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/db"
	"github.com/coreos/dex/refresh"
	"github.com/coreos/dex/user"
)

func newRefreshRepo(t *testing.T, users []user.UserWithRemoteIdentities, clients []client.Client) refresh.RefreshTokenRepo {
	var dbMap *gorp.DbMap
	if dsn := os.Getenv("DEX_TEST_DSN"); dsn == "" {
		dbMap = db.NewMemDB()
	} else {
		dbMap = connect(t)
	}
	if _, err := db.NewUserRepoFromUsers(dbMap, users); err != nil {
		t.Fatalf("Unable to add users: %v", err)
	}
	if _, err := db.NewClientRepoFromClients(dbMap, clients); err != nil {
		t.Fatalf("Unable to add clients: %v", err)
	}
	return db.NewRefreshTokenRepo(dbMap)
}

func TestRefreshTokenRepo(t *testing.T) {
	clientID := "client1"
	userID := "user1"
	clients := []client.Client{
		{
			Credentials: oidc.ClientCredentials{
				ID:     clientID,
				Secret: base64.URLEncoding.EncodeToString([]byte("secret-2")),
			},
			Metadata: oidc.ClientMetadata{
				RedirectURIs: []url.URL{
					url.URL{Scheme: "https", Host: "client1.example.com", Path: "/callback"},
				},
			},
		},
	}
	users := []user.UserWithRemoteIdentities{
		{
			User: user.User{
				ID:        userID,
				Email:     "Email-1@example.com",
				CreatedAt: time.Now().Truncate(time.Second),
			},
			RemoteIdentities: []user.RemoteIdentity{
				{
					ConnectorID: "IDPC-1",
					ID:          "RID-1",
				},
			},
		},
	}

	repo := newRefreshRepo(t, users, clients)
	tok, err := repo.Create(userID, clientID)
	if err != nil {
		t.Fatalf("failed to create refresh token: %v", err)
	}
	if tokUserID, err := repo.Verify(clientID, tok); err != nil {
		t.Errorf("Could not verify token: %v", err)
	} else if tokUserID != userID {
		t.Errorf("Verified token returned wrong user id, want=%s, got=%s", userID, tokUserID)
	}

	if userClients, err := repo.ClientsWithRefreshTokens(userID); err != nil {
		t.Errorf("Failed to get the list of clients the user was logged into: %v", err)
	} else {
		if diff := pretty.Compare(userClients, clients); diff == "" {
			t.Errorf("Clients user logged into: want did not equal got %s", diff)
		}
	}

	if err := repo.RevokeTokensForClient(userID, clientID); err != nil {
		t.Errorf("Failed to revoke refresh token: %v", err)
	}

	if _, err := repo.Verify(clientID, tok); err == nil {
		t.Errorf("Token which should have been revoked was verified")
	}
}
