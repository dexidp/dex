package integration

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/coreos/go-oidc/oidc"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/api/googleapi"

	"github.com/coreos/dex/admin"
	"github.com/coreos/dex/client"
	"github.com/coreos/dex/client/manager"
	"github.com/coreos/dex/db"
	"github.com/coreos/dex/schema/adminschema"
	"github.com/coreos/dex/server"
	"github.com/coreos/dex/user"
)

const (
	adminAPITestSecret = "admin_secret"
)

type adminAPITestFixtures struct {
	ur       user.UserRepo
	pwr      user.PasswordInfoRepo
	cr       client.ClientRepo
	adAPI    *admin.AdminAPI
	adSrv    *server.AdminServer
	hSrv     *httptest.Server
	hc       *http.Client
	adClient *adminschema.Service
}

func (t *adminAPITestFixtures) close() {
	t.hSrv.Close()
}

var (
	adminUsers = []user.UserWithRemoteIdentities{
		{
			User: user.User{
				ID:    "ID-1",
				Email: "Email-1@example.com",
			},
		},
		{
			User: user.User{
				ID:    "ID-2",
				Email: "Email-2@example.com",
			},
		},
		{
			User: user.User{
				ID:    "ID-3",
				Email: "Email-3@example.com",
			},
		},
	}

	adminPasswords = []user.PasswordInfo{
		{
			UserID:   "ID-1",
			Password: []byte("hi."),
		},
	}
)

type adminAPITransport struct {
	secret string
}

func (a *adminAPITransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Authorization", a.secret)
	return http.DefaultTransport.RoundTrip(r)
}

func makeAdminAPITestFixtures() *adminAPITestFixtures {
	f := &adminAPITestFixtures{}

	dbMap, ur, pwr, um := makeUserObjects(adminUsers, adminPasswords)

	var cliCount int
	secGen := func() ([]byte, error) {
		id := []byte(fmt.Sprintf("client_%v", cliCount))
		cliCount++
		return id, nil
	}
	cr := db.NewClientRepo(dbMap)
	clientIDGenerator := func(hostport string) (string, error) {
		return fmt.Sprintf("client_%v", hostport), nil
	}
	cm := manager.NewClientManager(cr, db.TransactionFactory(dbMap), manager.ManagerOptions{SecretGenerator: secGen, ClientIDGenerator: clientIDGenerator})
	ccr := db.NewConnectorConfigRepo(dbMap)

	f.cr = cr
	f.ur = ur
	f.pwr = pwr
	f.adAPI = admin.NewAdminAPI(ur, pwr, cr, ccr, um, cm, "local")
	f.adSrv = server.NewAdminServer(f.adAPI, nil, adminAPITestSecret)
	f.hSrv = httptest.NewServer(f.adSrv.HTTPHandler())
	f.hc = &http.Client{
		Transport: &adminAPITransport{
			secret: adminAPITestSecret,
		},
	}
	f.adClient, _ = adminschema.NewWithBasePath(f.hc, f.hSrv.URL)

	return f
}

func TestGetAdmin(t *testing.T) {

	tests := []struct {
		id      string
		errCode int
	}{
		{
			id:      "ID-1",
			errCode: -1,
		},
		{
			id:      "ID-2",
			errCode: http.StatusNotFound,
		},
	}

	for i, tt := range tests {
		func() {
			f := makeAdminAPITestFixtures()
			defer f.close()
			admn, err := f.adClient.Admin.Get(tt.id).Do()
			if tt.errCode != -1 {
				if err == nil {
					t.Errorf("case %d: err was nil", i)
					return
				}
				gErr, ok := err.(*googleapi.Error)
				if !ok {
					t.Errorf("case %d: not a googleapi Error: %q", i, err)
					return
				}

				if gErr.Code != tt.errCode {
					t.Errorf("case %d: want=%d, got=%d", i, tt.errCode, gErr.Code)
					return
				}
			} else {
				if err != nil {
					t.Errorf("case %d: err != nil: %q", i, err)
				}
				if admn == nil {
					t.Errorf("case %d: admn was nil", i)
				}

				if admn.Id != "ID-1" {
					t.Errorf("case %d: want=%q, got=%q", i, tt.id, admn.Id)
				}
			}
		}()
	}
}

func TestCreateAdmin(t *testing.T) {
	tests := []struct {
		admn     *adminschema.Admin
		errCode  int
		secret   string
		noSecret bool
	}{
		{
			admn: &adminschema.Admin{
				Email:    "foo@example.com",
				Password: "foopass",
			},
			errCode: -1,
		},
		{
			admn: &adminschema.Admin{
				Email:    "foo@example.com",
				Password: "foopass",
			},
			errCode: http.StatusUnauthorized,
			secret:  "bad_secret",
		},
		{
			admn: &adminschema.Admin{
				Email:    "foo@example.com",
				Password: "foopass",
			},
			errCode:  http.StatusUnauthorized,
			noSecret: true,
		},
		{
			// duplicate Email
			admn: &adminschema.Admin{
				Email:    "Email-1@example.com",
				Password: "foopass",
			},
			errCode: http.StatusBadRequest,
		},
		{
			// missing Email
			admn: &adminschema.Admin{
				Password: "foopass",
			},
			errCode: http.StatusBadRequest,
		},
	}
	for i, tt := range tests {
		func() {
			f := makeAdminAPITestFixtures()
			if tt.secret != "" {
				f.hc.Transport = &adminAPITransport{
					secret: tt.secret,
				}
			}
			if tt.noSecret {
				f.hc.Transport = http.DefaultTransport
			}
			defer f.close()

			admn, err := f.adClient.Admin.Create(tt.admn).Do()
			if tt.errCode != -1 {
				if err == nil {
					t.Errorf("case %d: err was nil", i)
					return
				}
				gErr, ok := err.(*googleapi.Error)
				if !ok {
					t.Errorf("case %d: not a googleapi Error: %q", i, err)
					return
				}

				if gErr.Code != tt.errCode {
					t.Errorf("case %d: want=%d, got=%d", i, tt.errCode, gErr.Code)
					return
				}
			} else {
				if err != nil {
					t.Errorf("case %d: err != nil: %q", i, err)
				}

				tt.admn.Id = admn.Id
				if diff := pretty.Compare(tt.admn, admn); diff != "" {
					t.Errorf("case %d: Compare(want, got) = %v", i, diff)
				}

				gotAdmn, err := f.adClient.Admin.Get(admn.Id).Do()
				if err != nil {
					t.Errorf("case %d: err != nil: %q", i, err)
				}
				if diff := pretty.Compare(admn, gotAdmn); diff != "" {
					t.Errorf("case %d: Compare(want, got) = %v", i, diff)
				}

				usr, err := f.ur.GetByRemoteIdentity(nil, user.RemoteIdentity{
					ConnectorID: "local",
					ID:          tt.admn.Id,
				})
				if err != nil {
					t.Errorf("case %d: err != nil: %q", i, err)
				}

				if usr.ID != tt.admn.Id {
					t.Errorf("case %d: want=%q, got=%q", i, tt.admn.Id, usr.ID)
				}

			}
		}()
	}
}

func TestConnectors(t *testing.T) {
	tests := []struct {
		req     adminschema.ConnectorsSetRequest
		want    adminschema.ConnectorsGetResponse
		wantErr bool
	}{
		{
			req: adminschema.ConnectorsSetRequest{
				Connectors: []interface{}{
					map[string]string{
						"type": "local",
						"id":   "local",
					},
				},
			},
			want: adminschema.ConnectorsGetResponse{
				Connectors: []interface{}{
					map[string]string{
						"id": "local",
					},
				},
			},
			wantErr: false,
		},
		{
			req: adminschema.ConnectorsSetRequest{
				Connectors: []interface{}{
					map[string]string{
						"type":         "github",
						"id":           "github",
						"clientID":     "foo",
						"clientSecret": "bar",
					},
					map[string]interface{}{
						"type":                 "oidc",
						"id":                   "oidc",
						"issuerURL":            "https://auth.example.com",
						"clientID":             "foo",
						"clientSecret":         "bar",
						"trustedEmailProvider": true,
					},
				},
			},
			want: adminschema.ConnectorsGetResponse{
				Connectors: []interface{}{
					map[string]string{
						"id":           "github",
						"clientID":     "foo",
						"clientSecret": "bar",
					},
					map[string]interface{}{
						"id":                   "oidc",
						"issuerURL":            "https://auth.example.com",
						"clientID":             "foo",
						"clientSecret":         "bar",
						"trustedEmailProvider": true,
					},
				},
			},
			wantErr: false,
		},
		{
			// Missing "type" argument
			req: adminschema.ConnectorsSetRequest{
				Connectors: []interface{}{
					map[string]string{
						"id": "local",
					},
				},
			},
			wantErr: true,
		},
	}

	for i, tt := range tests {
		f := makeAdminAPITestFixtures()
		if err := f.adClient.Connectors.Set(&tt.req).Do(); err != nil {
			if !tt.wantErr {
				t.Errorf("case %d: failed to set connectors: %v", i, err)
			}
			continue
		}
		if tt.wantErr {
			t.Errorf("case %d: expected error setting connectors", i)
			continue
		}

		resp, err := f.adClient.Connectors.Get().Do()
		if err != nil {
			t.Errorf("case %d: failed toget connectors: %v", i, err)
			continue
		}
		if diff := pretty.Compare(tt.want, resp); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %s", i, diff)
		}
	}
}

func TestCreateClient(t *testing.T) {
	mustParseURL := func(s string) *url.URL {
		u, err := url.Parse(s)
		if err != nil {
			t.Fatalf("couldn't parse URL: %v", err)
		}
		return u
	}

	addIDAndSecret := func(cli adminschema.Client) *adminschema.Client {
		if cli.Id == "" {
			if cli.Public {
				cli.Id = "client_" + cli.ClientName
			} else {
				cli.Id = "client_auth.example.com"
			}
		}

		if cli.Secret == "" {
			cli.Secret = base64.URLEncoding.EncodeToString([]byte("client_0"))
		}
		return &cli
	}

	adminClientGood := adminschema.Client{
		RedirectURIs: []string{"https://auth.example.com/"},
	}
	clientGood := client.Client{
		Credentials: oidc.ClientCredentials{
			ID: "client_auth.example.com",
		},
		Metadata: oidc.ClientMetadata{
			RedirectURIs: []url.URL{*mustParseURL("https://auth.example.com/")},
		},
	}

	clientPublicGood := clientGood
	clientPublicGood.Public = true
	clientPublicGood.Metadata.ClientName = "PublicName"
	clientPublicGood.Metadata.RedirectURIs = []url.URL{}
	clientPublicGood.Credentials.ID = "client_PublicName"

	adminPublicClientGood := adminClientGood
	adminPublicClientGood.Public = true
	adminPublicClientGood.ClientName = "PublicName"
	adminPublicClientGood.RedirectURIs = []string{}

	adminPublicClientMissingName := adminPublicClientGood
	adminPublicClientMissingName.ClientName = ""

	adminPublicClientHasARedirect := adminPublicClientGood
	adminPublicClientHasARedirect.RedirectURIs = []string{"https://auth.example.com/"}

	adminAdminClient := adminClientGood
	adminAdminClient.IsAdmin = true
	clientGoodAdmin := clientGood
	clientGoodAdmin.Admin = true

	adminMultiRedirect := adminClientGood
	adminMultiRedirect.RedirectURIs = []string{"https://auth.example.com/", "https://auth2.example.com/"}
	clientMultiRedirect := clientGood
	clientMultiRedirect.Metadata.RedirectURIs = append(
		clientMultiRedirect.Metadata.RedirectURIs,
		*mustParseURL("https://auth2.example.com/"))

	adminClientWithPeers := adminClientGood
	adminClientWithPeers.TrustedPeers = []string{"test_client_0"}

	adminClientOwnID := adminClientGood
	adminClientOwnID.Id = "my_own_id"

	clientGoodOwnID := clientGood
	clientGoodOwnID.Credentials.ID = "my_own_id"

	adminClientOwnSecret := adminClientGood
	adminClientOwnSecret.Secret = base64.URLEncoding.EncodeToString([]byte("my_own_secret"))
	clientGoodOwnSecret := clientGood

	adminClientOwnIDAndSecret := adminClientGood
	adminClientOwnIDAndSecret.Id = "my_own_id"
	adminClientOwnIDAndSecret.Secret = base64.URLEncoding.EncodeToString([]byte("my_own_secret"))
	clientGoodOwnIDAndSecret := clientGoodOwnID

	adminClientBadSecret := adminClientGood
	adminClientBadSecret.Secret = "not_base64_encoded"

	tests := []struct {
		req              adminschema.ClientCreateRequest
		want             adminschema.ClientCreateResponse
		wantClient       client.Client
		wantError        int
		wantTrustedPeers []string
	}{
		{
			req:       adminschema.ClientCreateRequest{},
			wantError: http.StatusBadRequest,
		}, {
			req: adminschema.ClientCreateRequest{
				Client: &adminschema.Client{
					IsAdmin: true,
				},
			},
			wantError: http.StatusBadRequest,
		}, {
			req: adminschema.ClientCreateRequest{
				Client: &adminschema.Client{
					RedirectURIs: []string{"909090"},
				},
			},
			wantError: http.StatusBadRequest,
		}, {
			req: adminschema.ClientCreateRequest{
				Client: &adminClientGood,
			},
			want: adminschema.ClientCreateResponse{
				Client: addIDAndSecret(adminClientGood),
			},
			wantClient: clientGood,
		}, {
			req: adminschema.ClientCreateRequest{
				Client: &adminAdminClient,
			},
			want: adminschema.ClientCreateResponse{
				Client: addIDAndSecret(adminAdminClient),
			},
			wantClient: clientGoodAdmin,
		}, {
			req: adminschema.ClientCreateRequest{
				Client: &adminMultiRedirect,
			},
			want: adminschema.ClientCreateResponse{
				Client: addIDAndSecret(adminMultiRedirect),
			},
			wantClient: clientMultiRedirect,
		}, {
			req: adminschema.ClientCreateRequest{
				Client: &adminClientWithPeers,
			},
			want: adminschema.ClientCreateResponse{
				Client: addIDAndSecret(adminClientWithPeers),
			},
			wantClient:       clientGood,
			wantTrustedPeers: []string{"test_client_0"},
		}, {
			req: adminschema.ClientCreateRequest{
				Client: &adminPublicClientGood,
			},
			want: adminschema.ClientCreateResponse{
				Client: addIDAndSecret(adminPublicClientGood),
			},
			wantClient: clientPublicGood,
		}, {
			req: adminschema.ClientCreateRequest{
				Client: &adminPublicClientMissingName,
			},
			wantError: http.StatusBadRequest,
		}, {
			req: adminschema.ClientCreateRequest{
				Client: &adminPublicClientHasARedirect,
			},
			wantError: http.StatusBadRequest,
		}, {
			req: adminschema.ClientCreateRequest{
				Client: &adminClientOwnID,
			},
			want: adminschema.ClientCreateResponse{
				Client: addIDAndSecret(adminClientOwnID),
			},
			wantClient: clientGoodOwnID,
		}, {
			req: adminschema.ClientCreateRequest{
				Client: &adminClientOwnSecret,
			},
			want: adminschema.ClientCreateResponse{
				Client: addIDAndSecret(adminClientOwnSecret),
			},
			wantClient: clientGoodOwnSecret,
		}, {
			req: adminschema.ClientCreateRequest{
				Client: &adminClientOwnIDAndSecret,
			},
			want: adminschema.ClientCreateResponse{
				Client: addIDAndSecret(adminClientOwnIDAndSecret),
			},
			wantClient: clientGoodOwnIDAndSecret,
		}, {
			req: adminschema.ClientCreateRequest{
				Client: &adminClientBadSecret,
			},
			wantError: http.StatusBadRequest,
		},
	}

	for i, tt := range tests {
		f := makeAdminAPITestFixtures()
		for j, r := range []string{"https://client0.example.com",
			"https://client1.example.com"} {
			_, err := f.cr.New(nil, client.Client{
				Credentials: oidc.ClientCredentials{
					ID: fmt.Sprintf("test_client_%d", j),
				},
				Metadata: oidc.ClientMetadata{
					RedirectURIs: []url.URL{*mustParseURL(r)},
				},
			})
			if err != nil {
				t.Errorf("case %d, client %d: unexpected error creating client: %v", i, j, err)
				continue
			}
		}

		resp, err := f.adClient.Client.Create(&tt.req).Do()
		if tt.wantError != 0 {
			if err == nil {
				t.Errorf("case %d: want non-nil error.", i)
				continue
			}

			aErr, ok := err.(*googleapi.Error)
			if !ok {
				t.Errorf("case %d: could not assert as adminSchema.Error: %v", i, err)
				continue
			}
			if aErr.Code != tt.wantError {
				t.Errorf("case %d: want aErr.Code=%v, got %v", i, tt.wantError, aErr.Code)
				continue
			}
			continue
		}

		if err != nil {
			t.Errorf("case %d: unexpected error creating client: %v", i, err)
			continue
		}

		if diff := pretty.Compare(tt.want, resp); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i, diff)
		}

		repoClient, err := f.cr.Get(nil, resp.Client.Id)
		if err != nil {
			t.Errorf("case %d: Unexpected error getting client: %v", i, err)
			continue
		}

		if diff := pretty.Compare(tt.wantClient, repoClient); diff != "" {
			t.Errorf("case %d: Compare(wantClient, repoClient) = %v", i, diff)
		}
	}
}

func TestGetState(t *testing.T) {
	tests := []struct {
		addUsers []user.User
		want     adminschema.State
	}{
		{
			addUsers: []user.User{
				user.User{
					ID:    "ID-admin",
					Email: "Admin@example.com",
					Admin: true,
				},
			},
			want: adminschema.State{
				AdminUserCreated: true,
			},
		},
		{
			want: adminschema.State{
				AdminUserCreated: false,
			},
		},
	}

	for i, tt := range tests {
		func() {
			f := makeAdminAPITestFixtures()
			defer f.close()

			for _, usr := range tt.addUsers {
				err := f.ur.Create(nil, usr)
				if err != nil {
					t.Fatalf("case %d: err != nil: %v", i, err)
				}
			}

			got, err := f.adClient.State.Get().Do()
			if err != nil {
				t.Errorf("case %d: err != nil: %q", i, err)
			}

			if diff := pretty.Compare(tt.want, got); diff != "" {
				t.Errorf("case %d: Compare(want, got) = %v", i, diff)
			}

		}()
	}

}
