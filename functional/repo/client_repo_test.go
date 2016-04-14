package repo

import (
	"encoding/base64"
	"net/url"
	"os"
	"testing"

	"github.com/coreos/go-oidc/oidc"
	"github.com/go-gorp/gorp"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/db"
)

var (
	testClients = []client.Client{
		client.Client{
			Credentials: oidc.ClientCredentials{
				ID:     "client1",
				Secret: base64.URLEncoding.EncodeToString([]byte("secret-1")),
			},
			Metadata: oidc.ClientMetadata{
				RedirectURIs: []url.URL{
					url.URL{
						Scheme: "https",
						Host:   "client1.example.com",
						Path:   "/callback",
					},
				},
			},
		},
		client.Client{
			Credentials: oidc.ClientCredentials{
				ID:     "client2",
				Secret: base64.URLEncoding.EncodeToString([]byte("secret-2")),
			},
			Metadata: oidc.ClientMetadata{
				RedirectURIs: []url.URL{
					url.URL{
						Scheme: "https",
						Host:   "client2.example.com",
						Path:   "/callback",
					},
				},
			},
		},
	}
)

func newClientIdentityRepo(t *testing.T) client.ClientIdentityRepo {
	dsn := os.Getenv("DEX_TEST_DSN")
	var dbMap *gorp.DbMap
	if dsn == "" {
		dbMap = db.NewMemDB()
	} else {
		dbMap = connect(t)
	}
	repo, err := db.NewClientIdentityRepoFromClients(dbMap, testClients)
	if err != nil {
		t.Fatalf("failed to create client repo from clients: %v", err)
	}
	return repo
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

Tests:
	for i, tt := range tests {
		repo := newClientIdentityRepo(t)
		for _, cid := range startAdmins {
			err := repo.SetDexAdmin(cid, true)
			if err != nil {
				t.Errorf("case %d: failed to set dex admin: %v", i, err)
				continue Tests
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
			t.Errorf("case %d: unexpected error: %v", i, err)
		}
		if gotAdmin != tt.wantAdmin {
			t.Errorf("case %d: want=%v, got=%v", i, tt.wantAdmin, gotAdmin)
		}

		err = repo.SetDexAdmin(tt.cid, tt.setAdmin)
		if err != nil {
			t.Errorf("case %d: unexpected error: %v", i, err)
		}

		gotAdmin, err = repo.IsDexAdmin(tt.cid)
		if err != nil {
			t.Errorf("case %d: unexpected error: %v", i, err)
		}
		if gotAdmin != tt.setAdmin {
			t.Errorf("case %d: want=%v, got=%v", i, tt.setAdmin, gotAdmin)
		}

	}
}
