package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/api/googleapi"

	"github.com/coreos/dex/admin"
	"github.com/coreos/dex/schema/adminschema"
	"github.com/coreos/dex/server"
	"github.com/coreos/dex/user"
	"github.com/coreos/go-oidc/oidc"
)

const (
	adminAPITestSecret = "admin_secret"
)

type adminAPITestFixtures struct {
	ur       user.UserRepo
	pwr      user.PasswordInfoRepo
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

	dbMap, ur, pwr, _ := makeUserObjects(adminUsers, adminPasswords)
	f.ur = ur
	f.pwr = pwr
	f.adAPI = admin.NewAdminAPI(dbMap, "local")
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

func TestCreateClient(t *testing.T) {
	tests := []struct {
		client    oidc.ClientMetadata
		wantError bool
	}{
		{
			client:    oidc.ClientMetadata{},
			wantError: true,
		},
		{
			client: oidc.ClientMetadata{
				RedirectURIs: []url.URL{
					{Scheme: "https", Host: "auth.example.com", Path: "/"},
				},
			},
		},
	}

	for i, tt := range tests {
		err := func() error {
			f := makeAdminAPITestFixtures()
			s := httptest.NewServer(f.adSrv.HTTPHandler())
			defer s.Close()

			req := struct {
				Client  oidc.ClientMetadata `json:"client"`
				IsAdmin bool                `json:"isAdmin"`
			}{tt.client, true}

			data, err := json.Marshal(&req)
			if err != nil {
				return err
			}
			resp, err := f.hc.Post(s.URL+"/api/v1/client", "application/json", bytes.NewReader(data))
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read body: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				if tt.wantError {
					return nil
				}
				return fmt.Errorf("expected status 200 got %d: %s", resp.StatusCode, body)
			}
			if tt.wantError {
				return errors.New("expected non nil error when creating client")
			}

			var clientResponse oidc.ClientRegistrationResponse
			if err := json.Unmarshal(body, &clientResponse); err != nil {
				return fmt.Errorf("failed to decode registration response: %v", err)
			}
			if diff := pretty.Compare(clientResponse.ClientMetadata, tt.client); diff != "" {
				return errors.New(diff)
			}
			return nil
		}()
		if err != nil {
			t.Errorf("case %d: %v", i, err)
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
