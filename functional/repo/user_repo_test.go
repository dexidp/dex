package repo

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/kylelemons/godebug/pretty"

	"github.com/coreos/dex/db"
	"github.com/coreos/dex/user"
)

var (
	testUsers = []user.UserWithRemoteIdentities{
		{
			User: user.User{
				ID:        "ID-1",
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
		{
			User: user.User{
				ID:        "ID-2",
				Email:     "Email-2@example.com",
				CreatedAt: time.Now(),
				Disabled:  true,
			},
			RemoteIdentities: []user.RemoteIdentity{
				{
					ConnectorID: "IDPC-2",
					ID:          "RID-2",
				},
			},
		},
	}
)

func newUserRepo(t *testing.T, users []user.UserWithRemoteIdentities) user.UserRepo {
	if users == nil {
		users = []user.UserWithRemoteIdentities{}
	}
	var dbMap *gorp.DbMap
	if os.Getenv("DEX_TEST_DSN") == "" {
		dbMap = db.NewMemDB()
	} else {
		dbMap = connect(t)
	}
	repo, err := db.NewUserRepoFromUsers(dbMap, users)
	if err != nil {
		t.Fatalf("Unable to add users: %v", err)
	}
	return repo
}

func TestNewUser(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	tests := []struct {
		user user.User
		err  error
	}{
		{
			user: user.User{
				ID:        "ID-bob",
				Email:     "bob@example.com",
				CreatedAt: now,
			},
			err: nil,
		},
		{
			user: user.User{
				ID:        "ID-admin",
				Email:     "admin@example.com",
				Admin:     true,
				CreatedAt: now,
			},
			err: nil,
		},
		{
			user: user.User{
				ID:            "ID-verified",
				Email:         "verified@example.com",
				EmailVerified: true,
				CreatedAt:     now,
			},
			err: nil,
		},
		{
			user: user.User{
				ID:          "ID-same",
				Email:       "Email-1@example.com",
				DisplayName: "Oops Same Email",
				CreatedAt:   now,
			},
			err: user.ErrorDuplicateEmail,
		},
		{
			user: user.User{
				Email:       "AnotherEmail@example.com",
				DisplayName: "Can't set your own ID!",
				CreatedAt:   now,
			},
			err: user.ErrorInvalidID,
		},
		{
			user: user.User{
				ID:          "ID-noemail",
				DisplayName: "No Email",
				CreatedAt:   now,
			},
			err: user.ErrorInvalidEmail,
		},
	}

	for i, tt := range tests {
		repo := newUserRepo(t, testUsers)
		err := repo.Create(nil, tt.user)
		if tt.err != nil {
			if err != tt.err {
				t.Errorf("case %d: want=%v, got=%v", i, tt.err, err)
			}
		} else {
			if err != nil {
				t.Errorf("case %d: want nil err, got %v", i, err)
			}

			gotUser, err := repo.Get(nil, tt.user.ID)
			if err != nil {
				t.Errorf("case %d: want nil err, got %v", i, err)
			}

			if diff := pretty.Compare(tt.user, gotUser); diff != "" {
				t.Errorf("case %d: Compare(want, got) = %v", i,
					diff)
			}
		}
	}
}

func TestUpdateUser(t *testing.T) {
	tests := []struct {
		user user.User
		err  error
	}{
		{
			// Update the email.
			user: user.User{
				ID:    "ID-1",
				Email: "Email-1.1@example.com",
			},
			err: nil,
		},
		{
			// No-op.
			user: user.User{
				ID:    "ID-1",
				Email: "Email-1@example.com",
			},
			err: nil,
		},
		{
			// No email.
			user: user.User{
				ID:    "ID-1",
				Email: "",
			},
			err: user.ErrorInvalidEmail,
		},
		{
			// Try Update on non-existent user.
			user: user.User{
				ID:    "NonExistent",
				Email: "GoodEmail@email.com",
			},
			err: user.ErrorNotFound,
		},
		{
			// Try update to someone else's email.
			user: user.User{
				ID:    "ID-2",
				Email: "Email-1@example.com",
			},
			err: user.ErrorDuplicateEmail,
		},
	}

	for i, tt := range tests {
		repo := newUserRepo(t, testUsers)
		err := repo.Update(nil, tt.user)
		if tt.err != nil {
			if err != tt.err {
				t.Errorf("case %d: want=%q, got=%q", i, tt.err, err)
			}
		} else {
			if err != nil {
				t.Errorf("case %d: want nil err, got %q", i, err)
			}

			gotUser, err := repo.Get(nil, tt.user.ID)
			if err != nil {
				t.Errorf("case %d: want nil err, got %q", i, err)
			}

			if diff := pretty.Compare(tt.user, gotUser); diff != "" {
				t.Errorf("case %d: Compare(want, got) = %v", i,
					diff)
			}
		}
	}
}

func TestDisableUser(t *testing.T) {
	tests := []struct {
		id      string
		disable bool
		err     error
	}{
		{
			id: "ID-1",
		},
		{
			id:      "ID-1",
			disable: true,
		},
		{
			id: "ID-2",
		},
		{
			id:      "ID-2",
			disable: true,
		},
		{
			id:  "NO SUCH ID",
			err: user.ErrorNotFound,
		},
		{
			id:      "NO SUCH ID",
			err:     user.ErrorNotFound,
			disable: true,
		},
		{
			id:  "",
			err: user.ErrorInvalidID,
		},
	}

	for i, tt := range tests {
		repo := newUserRepo(t, testUsers)
		err := repo.Disable(nil, tt.id, tt.disable)
		switch {
		case err != tt.err:
			t.Errorf("case %d: want=%q, got=%q", i, tt.err, err)
		case tt.err == nil:
			gotUser, err := repo.Get(nil, tt.id)
			if err != nil {
				t.Fatalf("case %d: want nil err, got %q", i, err)
			}

			if gotUser.Disabled != tt.disable {
				t.Errorf("case %d: disabled status want=%v got=%v",
					i, tt.disable, gotUser.Disabled)
			}
		}
	}
}

func TestAttachRemoteIdentity(t *testing.T) {
	tests := []struct {
		id  string
		rid user.RemoteIdentity
		err error
	}{
		{
			id: "ID-1",
			rid: user.RemoteIdentity{
				ConnectorID: "IDPC-1",
				ID:          "RID-1.1",
			},
		},
		{
			id: "ID-1",
			rid: user.RemoteIdentity{
				ConnectorID: "IDPC-2",
				ID:          "RID-2",
			},
			err: user.ErrorDuplicateRemoteIdentity,
		},
		{
			id: "NoSuchUser",
			rid: user.RemoteIdentity{
				ConnectorID: "IDPC-3",
				ID:          "RID-3",
			},
			err: user.ErrorNotFound,
		},
	}

	for i, tt := range tests {
		repo := newUserRepo(t, testUsers)
		err := repo.AddRemoteIdentity(nil, tt.id, tt.rid)
		if tt.err != nil {
			if err != tt.err {
				t.Errorf("case %d: want=%q, got=%q", i, tt.err, err)
			}
		} else {
			if err != nil {
				t.Errorf("case %d: want nil err, got %q", i, err)
			}

			gotUser, err := repo.GetByRemoteIdentity(nil, tt.rid)
			if err != nil {
				t.Errorf("case %d: want nil err, got %q", i, err)
			}

			wantUser, err := repo.Get(nil, tt.id)
			if err != nil {
				t.Errorf("case %d: want nil err, got %q", i, err)
			}

			gotRIDs, err := repo.GetRemoteIdentities(nil, tt.id)
			if err != nil {
				t.Errorf("case %d: want nil err, got %q", i, err)
			}

			if findRemoteIdentity(gotRIDs, tt.rid) == -1 {
				t.Errorf("case %d: user.RemoteIdentity not found", i)
			}

			if !reflect.DeepEqual(wantUser, gotUser) {
				t.Errorf("case %d: want=%#v, got=%#v", i,
					wantUser, gotUser)
			}
		}
	}
}

func TestRemoveRemoteIdentity(t *testing.T) {
	tests := []struct {
		id  string
		rid user.RemoteIdentity
		err error
	}{
		{
			id: "ID-1",
			rid: user.RemoteIdentity{
				ConnectorID: "IDPC-1",
				ID:          "RID-1",
			},
		},
		{
			id: "ID-1",
			rid: user.RemoteIdentity{
				ConnectorID: "IDPC-2",
				ID:          "RID-2",
			},
			err: user.ErrorNotFound,
		},
		{
			id: "NoSuchUser",
			rid: user.RemoteIdentity{
				ConnectorID: "IDPC-3",
				ID:          "RID-3",
			},
			err: user.ErrorNotFound,
		},
	}

	for i, tt := range tests {
		repo := newUserRepo(t, testUsers)
		err := repo.RemoveRemoteIdentity(nil, tt.id, tt.rid)
		if tt.err != nil {
			if err != tt.err {
				t.Errorf("case %d: want=%q, got=%q", i, tt.err, err)
			}
		} else {
			if err != nil {
				t.Errorf("case %d: want nil err, got %q", i, err)
			}

			gotUser, err := repo.GetByRemoteIdentity(nil, tt.rid)
			if err == nil {
				if gotUser.ID == tt.id {
					t.Errorf("case %d: user found.", i)

				}
			} else if err != user.ErrorNotFound {
				t.Errorf("case %d: want %q err, got %q err", i, user.ErrorNotFound, err)
			}

			gotRIDs, err := repo.GetRemoteIdentities(nil, tt.id)
			if err != nil {
				t.Errorf("case %d: want nil err, got %q", i, err)
			}

			if findRemoteIdentity(gotRIDs, tt.rid) != -1 {
				t.Errorf("case %d: user.RemoteIdentity found", i)
			}

		}
	}
}

func findRemoteIdentity(rids []user.RemoteIdentity, rid user.RemoteIdentity) int {
	for i, curRID := range rids {
		if curRID == rid {
			return i
		}
	}
	return -1
}

func TestGetByEmail(t *testing.T) {
	tests := []struct {
		email   string
		wantErr error
	}{
		{
			email:   "Email-1@example.com",
			wantErr: nil,
		},
		{
			email:   "NoSuchEmail@example.com",
			wantErr: user.ErrorNotFound,
		},
	}

	for i, tt := range tests {
		repo := newUserRepo(t, testUsers)
		gotUser, gotErr := repo.GetByEmail(nil, tt.email)
		if tt.wantErr != nil {
			if tt.wantErr != gotErr {
				t.Errorf("case %d: wantErr=%q, gotErr=%q", i, tt.wantErr, gotErr)
			}
			continue
		}

		if gotErr != nil {
			t.Errorf("case %d: want nil err:% q", i, gotErr)
		}

		if tt.email != gotUser.Email {
			t.Errorf("case %d: want=%q, got=%q", i, tt.email, gotUser.Email)
		}
	}
}

func TestGetAdminCount(t *testing.T) {
	tests := []struct {
		addUsers []user.User
		want     int
	}{
		{
			addUsers: []user.User{
				user.User{
					ID:    "ID-admin",
					Email: "Admin@example.com",
					Admin: true,
				},
			},
			want: 1,
		},
		{
			want: 0,
		},
		{
			addUsers: []user.User{
				user.User{
					ID:    "ID-admin",
					Email: "NotAdmin@example.com",
				},
			},
			want: 0,
		},
		{
			addUsers: []user.User{
				user.User{
					ID:    "ID-admin",
					Email: "Admin@example.com",
					Admin: true,
				},
				user.User{
					ID:    "ID-admin2",
					Email: "AnotherAdmin@example.com",
					Admin: true,
				},
			},
			want: 2,
		},
	}

	for i, tt := range tests {
		repo := newUserRepo(t, testUsers)
		for _, addUser := range tt.addUsers {
			err := repo.Create(nil, addUser)
			if err != nil {
				t.Fatalf("case %d: couldn't add user: %q", i, err)
			}
		}

		got, err := repo.GetAdminCount(nil)
		if err != nil {
			t.Errorf("case %d: couldn't get admin count: %q", i, err)
			continue
		}

		if tt.want != got {
			t.Errorf("case %d: want=%d, got=%d", i, tt.want, got)
		}
	}
}

func TestList(t *testing.T) {
	repoUsers := []user.UserWithRemoteIdentities{}
	for i := 0; i < 10; i++ {
		repoUsers = append(repoUsers, user.UserWithRemoteIdentities{
			User: user.User{
				ID:    fmt.Sprintf("%d", i),
				Email: fmt.Sprintf("%d@example.com", i),
			},
		})

	}
	tests := []struct {
		filter      user.UserFilter
		maxResults  int
		expectedIDs [][]string
	}{
		{
			maxResults:  5,
			expectedIDs: [][]string{{"0", "1", "2", "3", "4"}, {"5", "6", "7", "8", "9"}},
		},
		{
			maxResults:  3,
			expectedIDs: [][]string{{"0", "1", "2"}, {"3", "4", "5"}, {"6", "7", "8"}, {"9"}},
		},
		{
			maxResults:  9,
			expectedIDs: [][]string{{"0", "1", "2", "3", "4", "5", "6", "7", "8"}, {"9"}},
		},
		{
			maxResults:  10,
			expectedIDs: [][]string{{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}},
		},
	}

	for i, tt := range tests {
		repo := newUserRepo(t, repoUsers)
		var tok string
		gotIDs := [][]string{}
		done := false
		for !done {
			var users []user.User
			var err error
			users, tok, err = repo.List(nil, tt.filter, tt.maxResults, tok)
			if err != nil {
				t.Errorf("case %d: unexpected err: %v", i, err)
				done = true
				continue
			}
			ids := []string{}
			for _, user := range users {
				ids = append(ids, user.ID)
			}
			gotIDs = append(gotIDs, ids)
			if tok == "" {
				done = true
			}
		}
		if diff := pretty.Compare(tt.expectedIDs, gotIDs); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i,
				diff)
		}
	}
}

func TestListErrorNotFound(t *testing.T) {
	repo := newUserRepo(t, nil)
	_, _, err := repo.List(nil, user.UserFilter{}, 10, "")
	if err != user.ErrorNotFound {
		t.Errorf("want=%q, got=%q", user.ErrorNotFound, err)
	}
}
