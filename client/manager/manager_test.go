package manager

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"testing"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/db"
	"github.com/coreos/go-oidc/oidc"
)

type testFixtures struct {
	clientRepo client.ClientRepo
	mgr        *ClientManager
}

var (
	goodSecret = base64.URLEncoding.EncodeToString([]byte("secret"))
)

func makeTestFixtures() *testFixtures {
	f := &testFixtures{}

	dbMap := db.NewMemDB()
	clients := []client.LoadableClient{
		{
			Client: client.Client{
				Credentials: oidc.ClientCredentials{
					ID:     "client.example.com",
					Secret: goodSecret,
				},
				Metadata: oidc.ClientMetadata{
					RedirectURIs: []url.URL{
						{Scheme: "http", Host: "client.example.com", Path: "/"},
					},
				},
				Admin: true,
			},
		},
	}
	clientIDGenerator := func(hostport string) (string, error) {
		return hostport, nil
	}
	secGen := func() ([]byte, error) {
		return []byte("secret"), nil
	}

	var err error
	f.clientRepo, err = db.NewClientRepoFromClients(dbMap, clients)
	if err != nil {
		panic("Failed to create client manager: " + err.Error())
	}

	clientManager := NewClientManager(f.clientRepo, db.TransactionFactory(dbMap), ManagerOptions{ClientIDGenerator: clientIDGenerator, SecretGenerator: secGen})
	f.mgr = clientManager
	return f
}

func TestMetadata(t *testing.T) {
	tests := []struct {
		clientID string
		uri      string
		wantErr  bool
	}{
		{
			clientID: "client.example.com",
			uri:      "http://client.example.com/",
			wantErr:  false,
		},
	}

	for i, tt := range tests {
		f := makeTestFixtures()
		md, err := f.mgr.Metadata(tt.clientID)
		if err != nil {
			t.Errorf("case %d: unexpected err: %v", i, err)
			continue
		}
		if md.RedirectURIs[0].String() != tt.uri {
			t.Errorf("case %d: manager.Metadata.RedirectURIs: want=%q got=%q", i, tt.uri, md.RedirectURIs[0].String())
			continue
		}
	}
}

func TestIsDexAdmin(t *testing.T) {
	tests := []struct {
		clientID string
		isAdmin  bool
		wantErr  bool
	}{
		{
			clientID: "client.example.com",
			isAdmin:  true,
			wantErr:  false,
		},
	}

	for i, tt := range tests {
		f := makeTestFixtures()
		admin, err := f.mgr.IsDexAdmin(tt.clientID)
		if err != nil {
			t.Errorf("case %d: unexpected err: %v", i, err)
			continue
		}
		if admin != tt.isAdmin {
			t.Errorf("case %d: manager.Admin want=%t got=%t", i, tt.isAdmin, admin)
			continue
		}
	}
}

func TestSetDexAdmin(t *testing.T) {
	f := makeTestFixtures()
	err := f.mgr.SetDexAdmin("client.example.com", false)
	if err != nil {
		t.Errorf("unexpected err: %v", err)
	}
	admin, _ := f.mgr.IsDexAdmin("client.example.com")
	if admin {
		t.Errorf("expected admin to be false")
	}
}

func TestAuthenticate(t *testing.T) {
	f := makeTestFixtures()
	cm := oidc.ClientMetadata{
		RedirectURIs: []url.URL{
			url.URL{Scheme: "http", Host: "example.com", Path: "/cb"},
		},
	}
	cli := client.Client{
		Metadata: cm,
	}
	cc, err := f.mgr.New(cli, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}

	ok, err := f.mgr.Authenticate(*cc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	} else if !ok {
		t.Fatalf("Authentication failed for good creds")
	}

	creds := []oidc.ClientCredentials{
		//completely made up
		oidc.ClientCredentials{ID: "foo", Secret: "bar"},

		// good client ID, bad secret
		oidc.ClientCredentials{ID: cc.ID, Secret: "bar"},

		// bad client ID, good secret
		oidc.ClientCredentials{ID: "foo", Secret: cc.Secret},

		// good client ID, secret with some fluff on the end
		oidc.ClientCredentials{ID: cc.ID, Secret: fmt.Sprintf("%sfluff", cc.Secret)},
	}
	for i, c := range creds {
		ok, err := f.mgr.Authenticate(c)
		if err != nil {
			t.Errorf("case %d: unexpected error: %v", i, err)
		} else if ok {
			t.Errorf("case %d: authentication succeeded for bad creds", i)
		}
	}
}

func TestValidateClient(t *testing.T) {
	tests := []struct {
		cli     client.Client
		wantErr error
	}{
		{
			cli: client.Client{
				Metadata: oidc.ClientMetadata{
					RedirectURIs: []url.URL{mustParseURL("http://auth.google.com")},
				},
			},
		},
		{
			cli:     client.Client{},
			wantErr: client.ErrorMissingRedirectURI,
		},
		{
			cli: client.Client{
				Metadata: oidc.ClientMetadata{
					ClientName: "frank",
				},
				Public: true,
			},
		},
		{
			cli: client.Client{
				Metadata: oidc.ClientMetadata{
					RedirectURIs: []url.URL{mustParseURL("http://auth.google.com")},
					ClientName:   "frank",
				},
				Public: true,
			},
			wantErr: client.ErrorPublicClientRedirectURIs,
		},
		{
			cli: client.Client{
				Public: true,
			},
			wantErr: client.ErrorPublicClientMissingName,
		},
	}

	for i, tt := range tests {
		err := validateClient(tt.cli)
		if err != tt.wantErr {
			t.Errorf("case %d: want=%v, got=%v", i, tt.wantErr, err)
		}
	}
}
