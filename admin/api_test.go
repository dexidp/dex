package admin

import (
	"testing"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/db"
	"github.com/coreos/dex/schema/adminschema"
	"github.com/coreos/dex/user"
	"github.com/coreos/dex/user/manager"

	"github.com/kylelemons/godebug/pretty"
)

type testFixtures struct {
	ur    user.UserRepo
	pwr   user.PasswordInfoRepo
	cr    client.ClientRepo
	mgr   *manager.UserManager
	adAPI *AdminAPI
}

func makeTestFixtures() *testFixtures {
	f := &testFixtures{}

	dbMap := db.NewMemDB()
	f.ur = func() user.UserRepo {
		repo, err := db.NewUserRepoFromUsers(dbMap, []user.UserWithRemoteIdentities{
			{
				User: user.User{
					ID:          "ID-1",
					Email:       "email-1@example.com",
					DisplayName: "Name-1",
				},
			},
			{
				User: user.User{
					ID:          "ID-2",
					Email:       "email-2@example.com",
					DisplayName: "Name-2",
				},
			},
		})
		if err != nil {
			panic("Failed to create user repo: " + err.Error())
		}
		return repo
	}()

	f.pwr = func() user.PasswordInfoRepo {
		repo, err := db.NewPasswordInfoRepoFromPasswordInfos(dbMap, []user.PasswordInfo{
			{
				UserID:   "ID-1",
				Password: []byte("hi."),
			},
		})
		if err != nil {
			panic("Failed to create user repo: " + err.Error())
		}
		return repo
	}()

	ccr := func() connector.ConnectorConfigRepo {
		c := []connector.ConnectorConfig{&connector.LocalConnectorConfig{ID: "local"}}
		repo := db.NewConnectorConfigRepo(dbMap)
		if err := repo.Set(c); err != nil {
			panic(err)
		}
		return repo
	}()

	f.mgr = manager.NewUserManager(f.ur, f.pwr, ccr, db.TransactionFactory(dbMap), manager.ManagerOptions{})
	f.adAPI = NewAdminAPI(f.ur, f.pwr, f.cr, f.mgr, "local")

	return f
}

func TestGetAdmin(t *testing.T) {
	tests := []struct {
		id      string
		wantErr error
	}{
		{
			id: "ID-1",
		},
		{
			// Not found
			id:      "ID-3",
			wantErr: user.ErrorNotFound,
		},
	}

	for i, tt := range tests {
		f := makeTestFixtures()

		admn, err := f.adAPI.GetAdmin(tt.id)
		if tt.wantErr != nil {
			if err == nil {
				t.Errorf("case %d: err was nil", i)
				continue
			}
			aErr, ok := err.(Error)
			if !ok {
				t.Errorf("case %d: not an admin.Error: %q", i, err)
				continue
			}

			if aErr.Internal != tt.wantErr {
				t.Errorf("case %d: want=%q, got=%q", i, tt.wantErr, aErr.Internal)
				continue
			}
		} else {
			if err != nil {
				t.Errorf("case %d: err != nil: %q", i, err)
				continue
			}

			if admn.Id != "ID-1" {
				t.Errorf("case %d: want=%q, got=%q", i, tt.id, admn.Id)
			}
		}

	}

}

func TestCreateAdmin(t *testing.T) {
	hashedPassword, _ := user.NewPasswordFromPlaintext("foopass")
	tests := []struct {
		admn    adminschema.Admin
		wantErr error
	}{
		{
			//hashed password
			admn: adminschema.Admin{
				Email:    "goodemail@example.com",
				Password: string(hashedPassword),
			},
		},
		{
			//plaintext password
			admn: adminschema.Admin{
				Email:    "goodemail@example.com",
				Password: "foopass",
			},
		},
		{
			// duplicate Email
			admn: adminschema.Admin{
				Email:    "email-2@example.com",
				Password: "foopass",
			},
			wantErr: user.ErrorDuplicateEmail,
		},
		{
			// bad email
			admn: adminschema.Admin{
				Email:    "badEmailexample",
				Password: "foopass",
			},
			wantErr: user.ErrorInvalidEmail,
		},
		{
			// missing Email
			admn: adminschema.Admin{
				Password: "foopass",
			},
			wantErr: user.ErrorInvalidEmail,
		},
	}
	for i, tt := range tests {
		f := makeTestFixtures()

		id, err := f.adAPI.CreateAdmin(tt.admn)
		if tt.wantErr != nil {
			if err == nil {
				t.Errorf("case %d: err was nil", i)
				continue
			}
			aErr, ok := err.(Error)
			if !ok {
				t.Errorf("case %d: not a admin.Error: %#v", i, err)
				continue
			}

			if aErr.Internal != tt.wantErr {
				t.Errorf("case %d: want=%q, got=%q", i, tt.wantErr, aErr.Internal)
				continue
			}
		} else {
			if err != nil {
				t.Errorf("case %d: err != nil: %q", i, err)
			}

			gotAdmn, err := f.adAPI.GetAdmin(id)
			if err != nil {
				t.Errorf("case %d: err != nil: %q", i, err)
			}

			tt.admn.Id = id
			if diff := pretty.Compare(tt.admn, gotAdmn); diff != "" {
				t.Errorf("case %d: Compare(want, got) = %v", i, diff)
			}
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
					ID:          "ID-3",
					Email:       "email-3@example.com",
					DisplayName: "Admin",
					Admin:       true,
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
		f := makeTestFixtures()
		for _, usr := range tt.addUsers {
			_, err := f.mgr.CreateUser(usr, user.Password("foopass"), f.adAPI.localConnectorID)
			if err != nil {
				t.Fatalf("case %d: err != nil: %q", i, err)
			}
		}

		got, err := f.adAPI.GetState()
		if err != nil {
			t.Errorf("case %d: err != nil: %q", i, err)
		}

		if diff := pretty.Compare(tt.want, got); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i, diff)
		}
	}
}
