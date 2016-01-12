package repo

import (
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/coreos/go-oidc/oidc"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/db"
)

var makeTestClientIdentityRepoFromClients func(clients []oidc.ClientIdentity) client.ClientIdentityRepo

var (
	testClients = []oidc.ClientIdentity{
		oidc.ClientIdentity{
			Credentials: oidc.ClientCredentials{
				ID:     "client1",
				Secret: "secret-1",
			},
			Metadata: oidc.ClientMetadata{
				RedirectURIs: []*url.URL{
					&url.URL{
						Scheme: "https",
						Host:   "client1.example.com/callback",
					},
				},
			},
		},
		oidc.ClientIdentity{
			Credentials: oidc.ClientCredentials{
				ID:     "client2",
				Secret: "secret-2",
			},
			Metadata: oidc.ClientMetadata{
				RedirectURIs: []*url.URL{
					&url.URL{
						Scheme: "https",
						Host:   "client2.example.com/callback",
					},
				},
			},
		},
	}
)

func init() {
	dsn := os.Getenv("DEX_TEST_DSN")
	if dsn == "" {
		makeTestClientIdentityRepoFromClients = makeTestClientIdentityRepoMem
	} else {
		makeTestClientIdentityRepoFromClients = makeTestClientIdentityRepoDB(dsn)
	}
}

func makeTestClientIdentityRepoMem(clients []oidc.ClientIdentity) client.ClientIdentityRepo {
	return client.NewClientIdentityRepo(clients)
}

func makeTestClientIdentityRepoDB(dsn string) func([]oidc.ClientIdentity) client.ClientIdentityRepo {
	return func(clients []oidc.ClientIdentity) client.ClientIdentityRepo {
		c := initDB(dsn)

		repo, err := db.NewClientIdentityRepoFromClients(c, clients)
		if err != nil {
			panic(fmt.Sprintf("Unable to add clients: %v", err))
		}
		return repo
	}

}

func makeTestClientIdentityRepo() client.ClientIdentityRepo {
	return makeTestClientIdentityRepoFromClients(testClients)
}

func TestGetSetAdminClient(t *testing.T) {
	startAdmins := []string{"client2"}
	tests := []struct {
		// client ID
		cid string

		// initial state of client
		wantAdmin bool

		// final state of client
		setAdmin bool

		wantErr bool
	}{
		{
			cid:       "client1",
			wantAdmin: false,
			setAdmin:  true,
		},
		{
			cid:       "client1",
			wantAdmin: false,
			setAdmin:  false,
		},
		{
			cid:       "client2",
			wantAdmin: true,
			setAdmin:  true,
		},
		{
			cid:       "client2",
			wantAdmin: true,
			setAdmin:  false,
		},
	}

	for i, tt := range tests {
		repo := makeTestClientIdentityRepo()
		for _, cid := range startAdmins {
			err := repo.SetDexAdmin(cid, true)
			if err != nil {
				t.Fatalf("case %d: unexpected error: %v", i, err)
			}
		}

		gotAdmin, err := repo.IsDexAdmin(tt.cid)
		if tt.wantErr {
			if err == nil {
				t.Errorf("case %d: want non-nil err", i)
			}
			continue
		}
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}
		if gotAdmin != tt.wantAdmin {
			t.Errorf("case %d: want=%v, got=%v", i, tt.wantAdmin, gotAdmin)
		}

		err = repo.SetDexAdmin(tt.cid, tt.setAdmin)
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}

		gotAdmin, err = repo.IsDexAdmin(tt.cid)
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}
		if gotAdmin != tt.setAdmin {
			t.Errorf("case %d: want=%v, got=%v", i, tt.setAdmin, gotAdmin)
		}

	}
}
