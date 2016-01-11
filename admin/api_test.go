// NOTE: These tests are begin updated so they compile (see #257). Until then ignore.
// +build ignore

package admin

import (
	"net/http"
	"testing"

	"github.com/coreos/dex/schema/adminschema"
	"github.com/coreos/dex/user"

	"github.com/kylelemons/godebug/pretty"
)

type testFixtures struct {
	ur    user.UserRepo
	pwr   user.PasswordInfoRepo
	adAPI *AdminAPI
}

func makeTestFixtures() *testFixtures {
	f := &testFixtures{}

	f.ur = user.NewUserRepoFromUsers([]user.UserWithRemoteIdentities{
		{
			User: user.User{
				ID:   "ID-1",
				Name: "Name-1",
			},
		},
	})
	f.pwr = user.NewPasswordInfoRepoFromPasswordInfos([]user.PasswordInfo{
		{
			UserID:   "ID-1",
			Password: []byte("hi."),
		},
	})

	f.adAPI = NewAdminAPI(f.ur, f.pwr)

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
		f := makeTestFixtures()

		admn, err := f.adAPI.GetAdmin(tt.id)
		if tt.errCode != -1 {
			if err == nil {
				t.Errorf("case %d: err was nil", i)
				continue
			}
			aErr, ok := err.(Error)
			if !ok {
				t.Errorf("case %d: not an admin.Error: %q", i, err)
				continue
			}

			if aErr.Code != tt.errCode {
				t.Errorf("case %d: want=%d, got=%d", i, tt.errCode, aErr.Code)
				continue
			}
		} else {
			if err != nil {
				t.Errorf("case %d: err != nil: %q", i, err)
			}
			continue

			if admn.Id != "ID-1" {
				t.Errorf("case %d: want=%q, got=%q", i, tt.id, admn.Id)
			}
		}

	}

}

func TestCreateAdmin(t *testing.T) {
	tests := []struct {
		admn    adminschema.Admin
		errCode int
	}{
		{
			admn: adminschema.Admin{
				Name:         "foo",
				PasswordHash: user.Password([]byte("foopass")).EncodeBase64(),
			},
			errCode: -1,
		},
		{
			// duplicate Name
			admn: adminschema.Admin{
				Name:         "Name-1",
				PasswordHash: user.Password([]byte("foopass")).EncodeBase64(),
			},
			errCode: http.StatusBadRequest,
		},
		{
			// missing Name
			admn: adminschema.Admin{
				PasswordHash: user.Password([]byte("foopass")).EncodeBase64(),
			},
			errCode: http.StatusBadRequest,
		},
	}
	for i, tt := range tests {
		f := makeTestFixtures()

		id, err := f.adAPI.CreateAdmin(tt.admn)
		if tt.errCode != -1 {
			if err == nil {
				t.Errorf("case %d: err was nil", i)
				continue
			}
			aErr, ok := err.(Error)
			if !ok {
				t.Errorf("case %d: not a admin.Error: %#v", i, err)
				continue
			}

			if aErr.Code != tt.errCode {
				t.Errorf("case %d: want=%d, got=%d", i, tt.errCode, aErr.Code)
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
					Name:  "Admin",
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
		f := makeTestFixtures()
		for _, usr := range tt.addUsers {
			_, err := f.ur.Create(usr)
			if err != nil {
				t.Fatalf("case %d: err != nil", i, err)
			}
		}

		got, err := f.adAPI.GetState()
		if err != nil {
			t.Errorf("case %d: err != nil", i, err)
		}

		if diff := pretty.Compare(tt.want, got); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i, diff)
		}
	}
}
