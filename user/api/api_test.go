package api

import (
	"net/url"
	"testing"
	"time"

	"github.com/coreos/go-oidc/oidc"
	"github.com/jonboulle/clockwork"
	"github.com/kylelemons/godebug/pretty"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/repo"
	schema "github.com/coreos/dex/schema/workerschema"
	"github.com/coreos/dex/user"
)

type testEmailer struct {
	cantEmail       bool
	lastEmail       string
	lastClientID    string
	lastRedirectURL url.URL
}

// SendResetPasswordEmail returns resetPasswordURL when it can't email, mimicking the behavior of the real UserEmailer.
func (t *testEmailer) SendResetPasswordEmail(email string, redirectURL url.URL, clientID string) (*url.URL, error) {
	t.lastEmail = email
	t.lastRedirectURL = redirectURL
	t.lastClientID = clientID

	var retURL *url.URL
	if t.cantEmail {
		retURL = &resetPasswordURL
	}
	return retURL, nil
}

var (
	clock = clockwork.NewFakeClock()

	goodCreds = Creds{
		User: user.User{
			ID:    "ID-1",
			Admin: true,
		},
		ClientID: "XXX",
	}

	badCreds = Creds{
		User: user.User{
			ID: "ID-2",
		},
	}

	resetPasswordURL = url.URL{
		Host: "dex.example.com",
		Path: "resetPassword",
	}

	validRedirURL = url.URL{
		Scheme: "http",
		Host:   "client.example.com",
		Path:   "/callback",
	}
)

func makeTestFixtures() (*UsersAPI, *testEmailer) {
	ur := user.NewUserRepoFromUsers([]user.UserWithRemoteIdentities{
		{
			User: user.User{
				ID:        "ID-1",
				Email:     "id1@example.com",
				Admin:     true,
				CreatedAt: clock.Now(),
			},
		}, {
			User: user.User{
				ID:        "ID-2",
				Email:     "id2@example.com",
				CreatedAt: clock.Now(),
			},
		}, {
			User: user.User{
				ID:        "ID-3",
				Email:     "id3@example.com",
				CreatedAt: clock.Now(),
			},
		},
	})
	pwr := user.NewPasswordInfoRepoFromPasswordInfos([]user.PasswordInfo{
		{
			UserID:   "ID-1",
			Password: []byte("password-1"),
		},
		{
			UserID:   "ID-2",
			Password: []byte("password-2"),
		},
	})
	mgr := user.NewManager(ur, pwr, repo.InMemTransactionFactory, user.ManagerOptions{})
	mgr.Clock = clock
	ci := oidc.ClientIdentity{
		Credentials: oidc.ClientCredentials{
			ID:     "XXX",
			Secret: "secrete",
		},
		Metadata: oidc.ClientMetadata{
			RedirectURLs: []url.URL{
				validRedirURL,
			},
		},
	}
	cir := client.NewClientIdentityRepo([]oidc.ClientIdentity{ci})

	emailer := &testEmailer{}
	api := NewUsersAPI(mgr, cir, emailer, "local")
	return api, emailer

}

func TestGetUser(t *testing.T) {
	tests := []struct {
		creds   Creds
		id      string
		wantErr error
	}{
		{
			creds: goodCreds,
			id:    "ID-1",
		},
		{
			creds:   badCreds,
			id:      "ID-1",
			wantErr: ErrorUnauthorized,
		},
		{
			creds:   goodCreds,
			id:      "NO_ID",
			wantErr: ErrorResourceNotFound,
		},
	}

	for i, tt := range tests {
		api, _ := makeTestFixtures()
		usr, err := api.GetUser(tt.creds, tt.id)
		if tt.wantErr != nil {
			if err != tt.wantErr {
				t.Errorf("case %d: want=%q, got=%q", i, tt.wantErr, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("case %d: want nil err, got: %q ", i, err)
		}

		if usr.Id != tt.id {
			t.Errorf("case %d: want=%v, got=%v ", i, tt.id, usr.Id)
		}
	}
}

func TestListUsers(t *testing.T) {
	tests := []struct {
		creds      Creds
		filter     user.UserFilter
		maxResults int
		pages      int
		wantErr    error
		wantIDs    [][]string
	}{
		{
			creds:      goodCreds,
			pages:      3,
			maxResults: 1,
			wantIDs:    [][]string{{"ID-1"}, {"ID-2"}, {"ID-3"}},
		},
		{
			creds:      goodCreds,
			pages:      1,
			maxResults: 3,
			wantIDs:    [][]string{{"ID-1", "ID-2", "ID-3"}},
		},
		{
			creds:      badCreds,
			pages:      3,
			maxResults: 1,
			wantErr:    ErrorUnauthorized,
		},
		{
			creds:      goodCreds,
			pages:      3,
			maxResults: 10000,
			wantErr:    ErrorMaxResultsTooHigh,
		},
	}

	for i, tt := range tests {
		api, _ := makeTestFixtures()

		gotIDs := [][]string{}
		var next string
		var err error
		var users []*schema.User
		for x := 0; x < tt.pages; x++ {
			users, next, err = api.ListUsers(tt.creds, tt.maxResults, next)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("case %d: want=%q, got=%q", i, tt.wantErr, err)
				}
				goto NextTest
			}

			var ids []string
			for _, usr := range users {
				ids = append(ids, usr.Id)
			}
			gotIDs = append(gotIDs, ids)
		}

		if diff := pretty.Compare(tt.wantIDs, gotIDs); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i,
				diff)
		}
	NextTest:
	}
}

func TestCreateUser(t *testing.T) {
	tests := []struct {
		creds     Creds
		usr       schema.User
		redirURL  url.URL
		cantEmail bool

		wantResponse schema.UserCreateResponse
		wantErr      error
	}{
		{
			creds: goodCreds,
			usr: schema.User{
				Email:         "newuser@example.com",
				DisplayName:   "New User",
				EmailVerified: true,
				Admin:         false,
			},
			redirURL: validRedirURL,

			wantResponse: schema.UserCreateResponse{
				EmailSent: true,
				User: &schema.User{
					Email:         "newuser@example.com",
					DisplayName:   "New User",
					EmailVerified: true,
					Admin:         false,
					CreatedAt:     clock.Now().Format(time.RFC3339),
				},
			},
		},
		{
			creds: goodCreds,
			usr: schema.User{
				Email:         "newuser@example.com",
				DisplayName:   "New User",
				EmailVerified: true,
				Admin:         false,
			},
			redirURL:  validRedirURL,
			cantEmail: true,

			wantResponse: schema.UserCreateResponse{
				User: &schema.User{
					Email:         "newuser@example.com",
					DisplayName:   "New User",
					EmailVerified: true,
					Admin:         false,
					CreatedAt:     clock.Now().Format(time.RFC3339),
				},
				ResetPasswordLink: resetPasswordURL.String(),
			},
		},
		{
			creds: goodCreds,
			usr: schema.User{
				Email:         "newuser@example.com",
				DisplayName:   "New User",
				EmailVerified: true,
				Admin:         false,
			},
			redirURL: url.URL{Host: "scammers.com"},

			wantErr: ErrorInvalidRedirectURL,
		},
		{
			creds: badCreds,
			usr: schema.User{
				Email:         "newuser@example.com",
				DisplayName:   "New User",
				EmailVerified: true,
				Admin:         false,
			},
			redirURL: validRedirURL,

			wantErr: ErrorUnauthorized,
		},
	}

	for i, tt := range tests {
		api, emailer := makeTestFixtures()
		emailer.cantEmail = tt.cantEmail

		response, err := api.CreateUser(tt.creds, tt.usr, tt.redirURL)
		if tt.wantErr != nil {
			if err != tt.wantErr {
				t.Errorf("case %d: want=%q, got=%q", i, tt.wantErr, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("case %d: want nil err, got: %q ", i, err)
		}

		newID := response.User.Id
		if newID == "" {
			t.Errorf("case %d: expected non-empty newID", i)
		}

		tt.wantResponse.User.Id = newID
		if diff := pretty.Compare(tt.wantResponse, response); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i,
				diff)
		}

		wantEmalier := testEmailer{
			cantEmail:       tt.cantEmail,
			lastEmail:       tt.usr.Email,
			lastClientID:    tt.creds.ClientID,
			lastRedirectURL: tt.redirURL,
		}
		if diff := pretty.Compare(wantEmalier, emailer); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i,
				diff)
		}
	}
}
