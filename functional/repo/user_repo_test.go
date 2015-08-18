package repo

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"

	"github.com/coreos/dex/db"
	"github.com/coreos/dex/user"
)

var makeTestUserRepoFromUsers func(users []user.UserWithRemoteIdentities) user.UserRepo

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

func init() {
	dsn := os.Getenv("DEX_TEST_DSN")
	if dsn == "" {
		makeTestUserRepoFromUsers = makeTestUserRepoMem
	} else {
		makeTestUserRepoFromUsers = makeTestUserRepoDB(dsn)
	}
}

func makeTestUserRepoMem(users []user.UserWithRemoteIdentities) user.UserRepo {
	return user.NewUserRepoFromUsers(users)
}

func makeTestUserRepoDB(dsn string) func([]user.UserWithRemoteIdentities) user.UserRepo {
	return func(users []user.UserWithRemoteIdentities) user.UserRepo {
		c := initDB(dsn)

		repo, err := db.NewUserRepoFromUsers(c, users)
		if err != nil {
			panic(fmt.Sprintf("Unable to add users: %v", err))
		}
		return repo
	}

}

func makeTestUserRepo() user.UserRepo {
	return makeTestUserRepoFromUsers(testUsers)
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
		repo := makeTestUserRepo()
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
		repo := makeTestUserRepo()
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
		repo := makeTestUserRepo()
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
		repo := makeTestUserRepo()
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

func TestNewUserRepoFromUsers(t *testing.T) {
	tests := []struct {
		users []user.UserWithRemoteIdentities
	}{
		{
			users: []user.UserWithRemoteIdentities{
				{
					User: user.User{
						ID:    "123",
						Email: "email123@example.com",
					},
					RemoteIdentities: []user.RemoteIdentity{},
				},
				{
					User: user.User{
						ID:    "456",
						Email: "email456@example.com",
					},
					RemoteIdentities: []user.RemoteIdentity{
						{
							ID:          "remoteID",
							ConnectorID: "connID",
						},
					},
				},
			},
		},
	}

	for i, tt := range tests {
		repo := user.NewUserRepoFromUsers(tt.users)
		for _, want := range tt.users {
			gotUser, err := repo.Get(nil, want.User.ID)
			if err != nil {
				t.Errorf("case %d: want nil err: %v", i, err)
			}

			gotRIDs, err := repo.GetRemoteIdentities(nil, want.User.ID)
			if err != nil {
				t.Errorf("case %d: want nil err: %v", i, err)
			}

			if !reflect.DeepEqual(want.User, gotUser) {
				t.Errorf("case %d: want=%#v got=%#v", i, want.User, gotUser)
			}

			if !reflect.DeepEqual(want.RemoteIdentities, gotRIDs) {
				t.Errorf("case %d: want=%#v got=%#v", i, want.RemoteIdentities, gotRIDs)
			}
		}
	}
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
		repo := makeTestUserRepo()
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
		repo := makeTestUserRepo()
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
		repo := makeTestUserRepoFromUsers(repoUsers)
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
	repo := makeTestUserRepoFromUsers(nil)
	_, _, err := repo.List(nil, user.UserFilter{}, 10, "")
	if err != user.ErrorNotFound {
		t.Errorf("want=%q, got=%q", user.ErrorNotFound, err)
	}
}
