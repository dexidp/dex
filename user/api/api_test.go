package api

import (
	"encoding/base64"
	"net/url"
	"sort"
	"testing"
	"time"

	"github.com/coreos/go-oidc/oidc"
	"github.com/jonboulle/clockwork"
	"github.com/kylelemons/godebug/pretty"

	"github.com/coreos/dex/client"
	clientmanager "github.com/coreos/dex/client/manager"
	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/db"
	schema "github.com/coreos/dex/schema/workerschema"
	"github.com/coreos/dex/user"
	"github.com/coreos/dex/user/manager"
)

type testEmailer struct {
	cantEmail       bool
	lastEmail       string
	lastClientID    string
	lastRedirectURL url.URL
	lastWasInvite   bool
}

// SendResetPasswordEmail returns resetPasswordURL when it can't email, mimicking the behavior of the real UserEmailer.
func (t *testEmailer) SendResetPasswordEmail(email string, redirectURL url.URL, clientID string) (*url.URL, error) {
	return t.sendEmail(email, redirectURL, clientID, false)
}

func (t *testEmailer) SendInviteEmail(email string, redirectURL url.URL, clientID string) (*url.URL, error) {
	return t.sendEmail(email, redirectURL, clientID, true)
}

func (t *testEmailer) sendEmail(email string, redirectURL url.URL, clientID string, invite bool) (*url.URL, error) {
	t.lastEmail = email
	t.lastRedirectURL = redirectURL
	t.lastClientID = clientID
	t.lastWasInvite = invite

	var retURL *url.URL
	if t.cantEmail {
		retURL = &resetPasswordURL
	}
	return retURL, nil
}

var (
	clock            = clockwork.NewFakeClock()
	goodClientID     = "client.example.com"
	nonAdminClientID = "user.example.com"

	goodCreds = Creds{
		User: user.User{
			ID:    "ID-1",
			Admin: true,
		},
		ClientIDs: []string{goodClientID},
	}

	clientCreds = Creds{
		User: user.User{
			ID:    "",
			Admin: false,
		},
		ClientIDs: []string{goodClientID},
	}

	badCreds = Creds{
		User: user.User{
			ID: "ID-2",
		},
	}

	credsWithMultipleAudiences = Creds{
		User: user.User{
			ID:    "ID-1",
			Admin: true,
		},
		ClientIDs: []string{nonAdminClientID, goodClientID},
	}

	disabledCreds = Creds{
		User: user.User{
			ID:       "ID-1",
			Admin:    true,
			Disabled: true,
		},
		ClientIDs: []string{goodClientID},
	}

	resetPasswordURL = url.URL{
		Host: "dex.example.com",
		Path: "resetPassword",
	}

	validRedirURL = url.URL{
		Scheme: "http",
		Host:   goodClientID,
		Path:   "/callback",
	}
	validRedirURL2 = url.URL{
		Scheme: "http",
		Host:   nonAdminClientID,
		Path:   "/callback",
	}
)

func makeTestFixtures(clientCredsFlag bool) (*UsersAPI, *testEmailer) {
	dbMap := db.NewMemDB()
	ur := func() user.UserRepo {
		repo, err := db.NewUserRepoFromUsers(dbMap, []user.UserWithRemoteIdentities{
			{
				User: user.User{
					ID:             "ID-1",
					Email:          "id1@example.com",
					OrganizationID: "OrgID-1",
					Admin:          true,
					CreatedAt:      clock.Now(),
				},
			}, {
				User: user.User{
					ID:             "ID-2",
					Email:          "id2@example.com",
					EmailVerified:  true,
					OrganizationID: "OrgID-2",
					CreatedAt:      clock.Now(),
				},
			}, {
				User: user.User{
					ID:             "ID-3",
					Email:          "id3@example.com",
					OrganizationID: "OrgID-3",
					CreatedAt:      clock.Now(),
				},
			}, {
				User: user.User{
					ID:        "ID-4",
					Email:     "id4@example.com",
					CreatedAt: clock.Now(),
					Disabled:  true,
				},
			},
		})
		if err != nil {
			panic("Failed to create user repo: " + err.Error())
		}
		return repo
	}()

	pwr := func() user.PasswordInfoRepo {
		repo, err := db.NewPasswordInfoRepoFromPasswordInfos(dbMap, []user.PasswordInfo{
			{
				UserID:   "ID-1",
				Password: []byte("password-1"),
			},
			{
				UserID:   "ID-2",
				Password: []byte("password-2"),
			},
		})
		if err != nil {
			panic("Failed to create user repo: " + err.Error())
		}
		return repo
	}()

	orgr := func() user.OrganizationRepo {
		repo, err := db.NewOrganizationRepoFromOrganizations(dbMap, []user.Organization{
			{
				OrganizationID: "OrgID-1",
				Name:           "OrgName-1",
				OwnerID:        "ID-1",
			},
			{
				OrganizationID: "OrgID-2",
				Name:           "OrgName-2",
				OwnerID:        "ID-2",
			},
			{
				OrganizationID: "OrgID-3",
				Name:           "OrgName-3",
				OwnerID:        "ID-3",
			},
		})
		if err != nil {
			panic("Failed to create organization repo: " + err.Error())
		}
		return repo
	}()

	ccr := func() connector.ConnectorConfigRepo {
		repo := db.NewConnectorConfigRepo(dbMap)
		c := []connector.ConnectorConfig{
			&connector.LocalConnectorConfig{ID: "local"},
		}
		if err := repo.Set(c); err != nil {
			panic(err)
		}
		return repo
	}()

	mgr := manager.NewUserManager(ur, pwr, orgr, ccr, db.TransactionFactory(dbMap), manager.ManagerOptions{})
	mgr.Clock = clock
	ci := client.Client{
		Credentials: oidc.ClientCredentials{
			ID:     goodClientID,
			Secret: base64.URLEncoding.EncodeToString([]byte("secret")),
		},
		Metadata: oidc.ClientMetadata{
			RedirectURIs: []url.URL{
				validRedirURL,
			},
		},
	}
	ci2 := client.Client{
		Credentials: oidc.ClientCredentials{
			ID:     nonAdminClientID,
			Secret: base64.URLEncoding.EncodeToString([]byte("anothersecret")),
		},
		Metadata: oidc.ClientMetadata{
			RedirectURIs: []url.URL{
				validRedirURL2,
			},
		},
	}

	clientIDGenerator := func(hostport string) (string, error) {
		return hostport, nil
	}
	secGen := func() ([]byte, error) {
		return []byte("secret"), nil
	}
	clientRepo, err := db.NewClientRepoFromClients(dbMap, []client.LoadableClient{{Client: ci}, {Client: ci2}})
	if err != nil {
		panic("Failed to create client manager: " + err.Error())
	}
	clientManager := clientmanager.NewClientManager(clientRepo, db.TransactionFactory(dbMap), clientmanager.ManagerOptions{ClientIDGenerator: clientIDGenerator, SecretGenerator: secGen})

	// Used in TestRevokeRefreshToken test.
	refreshTokens := []struct {
		clientID string
		userID   string
	}{
		{goodClientID, "ID-1"},
		{goodClientID, "ID-2"},
	}
	refreshRepo := db.NewRefreshTokenRepo(dbMap)
	for _, token := range refreshTokens {
		if _, err := refreshRepo.Create(token.userID, token.clientID, "local", []string{"openid"}); err != nil {
			panic("Failed to create refresh token: " + err.Error())
		}
	}

	emailer := &testEmailer{}
	api := NewUsersAPI(mgr, clientManager, refreshRepo, emailer, "local", clientCredsFlag)
	return api, emailer

}

func TestGetUser(t *testing.T) {
	tests := []struct {
		creds           Creds
		id              string
		wantErr         error
		clientCredsFlag bool
	}{
		{
			creds:           goodCreds,
			id:              "ID-1",
			clientCredsFlag: false,
		},
		{
			creds:           badCreds,
			id:              "ID-1",
			wantErr:         ErrorUnauthorized,
			clientCredsFlag: false,
		},
		{
			creds:           goodCreds,
			id:              "NO_ID",
			wantErr:         ErrorResourceNotFound,
			clientCredsFlag: false,
		},
		{
			creds:           credsWithMultipleAudiences,
			id:              "ID-1",
			clientCredsFlag: false,
		},
		{
			creds:           clientCreds,
			id:              "ID-1",
			clientCredsFlag: true,
		},
	}

	for i, tt := range tests {
		api, _ := makeTestFixtures(tt.clientCredsFlag)
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
		api, _ := makeTestFixtures(false)

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

		clientCredsFlag bool

		wantResponse schema.UserCreateResponse
		wantClientID string
		wantErr      error
	}{
		{
			creds: goodCreds,
			usr: schema.User{
				Email:         "newuser01@example.com",
				DisplayName:   "New User",
				EmailVerified: true,
				Admin:         false,
			},
			redirURL: validRedirURL,

			wantResponse: schema.UserCreateResponse{
				EmailSent: true,
				User: &schema.User{
					Email:         "newuser01@example.com",
					DisplayName:   "New User",
					EmailVerified: true,
					Admin:         false,
					CreatedAt:     clock.Now().Format(time.RFC3339),
				},
			},
			wantClientID:    goodClientID,
			clientCredsFlag: false,
		},
		{
			creds: clientCreds,
			usr: schema.User{
				Email:         "newuser01@example.com",
				DisplayName:   "New User",
				EmailVerified: true,
				Admin:         false,
			},
			redirURL: validRedirURL,

			wantResponse: schema.UserCreateResponse{
				EmailSent: true,
				User: &schema.User{
					Email:         "newuser01@example.com",
					DisplayName:   "New User",
					EmailVerified: true,
					Admin:         false,
					CreatedAt:     clock.Now().Format(time.RFC3339),
				},
			},
			wantClientID:    goodClientID,
			clientCredsFlag: true,
		},
		{
			creds: credsWithMultipleAudiences,
			usr: schema.User{
				Email:         "newuser01@example.com",
				DisplayName:   "New User",
				EmailVerified: true,
				Admin:         false,
			},
			redirURL: validRedirURL,

			wantResponse: schema.UserCreateResponse{
				EmailSent: true,
				User: &schema.User{
					Email:         "newuser01@example.com",
					DisplayName:   "New User",
					EmailVerified: true,
					Admin:         false,
					CreatedAt:     clock.Now().Format(time.RFC3339),
				},
			},
			wantClientID:    goodClientID,
			clientCredsFlag: false,
		},
		{
			creds: goodCreds,
			usr: schema.User{
				Email:         "newuser02@example.com",
				DisplayName:   "New User",
				EmailVerified: true,
				Admin:         false,
			},
			redirURL:  validRedirURL,
			cantEmail: true,

			wantResponse: schema.UserCreateResponse{
				User: &schema.User{
					Email:         "newuser02@example.com",
					DisplayName:   "New User",
					EmailVerified: true,
					Admin:         false,
					CreatedAt:     clock.Now().Format(time.RFC3339),
				},
				ResetPasswordLink: resetPasswordURL.String(),
			},
			wantClientID:    goodClientID,
			clientCredsFlag: false,
		},
		{
			creds: goodCreds,
			usr: schema.User{
				Email:         "newuser03@example.com",
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
				Email:         "newuser04@example.com",
				DisplayName:   "New User",
				EmailVerified: true,
				Admin:         false,
			},
			redirURL: validRedirURL,

			wantErr:         ErrorUnauthorized,
			clientCredsFlag: false,
		},
	}

	for i, tt := range tests {
		api, emailer := makeTestFixtures(tt.clientCredsFlag)
		emailer.cantEmail = tt.cantEmail

		response, err := api.CreateUser(tt.creds, tt.usr, tt.redirURL)
		if tt.wantErr != nil {
			if err != tt.wantErr {
				t.Errorf("case %d: want=%q, got=%q", i, tt.wantErr, err)
				continue
			}

			tok := ""
			for {
				list, tok, err := api.ListUsers(goodCreds, 100, tok)
				if err != nil {
					t.Fatalf("case %d: unexpected error: %v", i, err)
					break
				}
				for _, u := range list {
					if u.Email == tt.usr.Email {
						t.Errorf("case %d: got an error but user was still created", i)
					}
				}
				if tok == "" {
					break
				}
			}

			continue
		}
		if err != nil {
			t.Errorf("case %d: want nil err, got: %q ", i, err)
			continue
		}

		newID := response.User.Id
		if newID == "" {
			t.Errorf("case %d: expected non-empty newID", i)
			continue
		}

		tt.wantResponse.User.Id = newID
		if diff := pretty.Compare(tt.wantResponse, response); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i,
				diff)
		}

		wantEmalier := testEmailer{
			cantEmail:       tt.cantEmail,
			lastEmail:       tt.usr.Email,
			lastClientID:    tt.wantClientID,
			lastRedirectURL: tt.redirURL,
			lastWasInvite:   true,
		}
		if diff := pretty.Compare(wantEmalier, emailer); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i,
				diff)
		}
	}
}

func TestDisableUsers(t *testing.T) {
	tests := []struct {
		id      string
		disable bool
	}{
		{
			id:      "ID-1",
			disable: true,
		},
		{
			id:      "ID-1",
			disable: false,
		},
		{
			id:      "ID-4",
			disable: true,
		},
		{
			id:      "ID-4",
			disable: false,
		},
	}

	for i, tt := range tests {
		api, _ := makeTestFixtures(false)
		_, err := api.DisableUser(goodCreds, tt.id, tt.disable)
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}

		usr, err := api.GetUser(goodCreds, tt.id)
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}

		if usr.Disabled != tt.disable {
			t.Errorf("case %d: user disable state wrong. wanted: %v got: %v", i, tt.disable, usr.Disabled)
		}
	}
}
func TestResendEmailInvitation(t *testing.T) {
	tests := []struct {
		creds     Creds
		userID    string
		email     string
		redirURL  url.URL
		cantEmail bool

		clientCredsFlag bool

		wantResponse schema.ResendEmailInvitationResponse
		wantErr      error
		wantClientID string
	}{
		{
			creds:    goodCreds,
			userID:   "ID-1",
			email:    "id1@example.com",
			redirURL: validRedirURL,

			wantResponse: schema.ResendEmailInvitationResponse{
				EmailSent: true,
			},
			wantClientID:    goodClientID,
			clientCredsFlag: false,
		},
		{
			creds:    clientCreds,
			userID:   "ID-1",
			email:    "id1@example.com",
			redirURL: validRedirURL,

			wantResponse: schema.ResendEmailInvitationResponse{
				EmailSent: true,
			},
			wantClientID:    goodClientID,
			clientCredsFlag: true,
		},
		{
			creds:     goodCreds,
			userID:    "ID-1",
			email:     "id1@example.com",
			redirURL:  validRedirURL,
			cantEmail: true,

			wantResponse: schema.ResendEmailInvitationResponse{
				EmailSent:         false,
				ResetPasswordLink: resetPasswordURL.String(),
			},
			wantClientID:    goodClientID,
			clientCredsFlag: false,
		},
		{
			creds:     credsWithMultipleAudiences,
			userID:    "ID-1",
			email:     "id1@example.com",
			redirURL:  validRedirURL,
			cantEmail: true,

			wantResponse: schema.ResendEmailInvitationResponse{
				EmailSent:         false,
				ResetPasswordLink: resetPasswordURL.String(),
			},
			wantClientID:    goodClientID,
			clientCredsFlag: false,
		},
		{
			creds:    badCreds,
			userID:   "ID-1",
			email:    "id1@example.com",
			redirURL: validRedirURL,

			wantErr:         ErrorUnauthorized,
			clientCredsFlag: false,
		},
		{
			creds:    goodCreds,
			userID:   "ID-1",
			email:    "id1@example.com",
			redirURL: url.URL{Host: "scammers.com"},

			wantErr:         ErrorInvalidRedirectURL,
			clientCredsFlag: false,
		},
		{
			creds:    goodCreds,
			userID:   "ID-2",
			email:    "id2@example.com",
			redirURL: validRedirURL,

			wantErr:         ErrorVerifiedEmail,
			clientCredsFlag: false,
		},
		{
			creds:    goodCreds,
			userID:   "non-existent",
			email:    "non-existent@example.com",
			redirURL: validRedirURL,

			wantErr:         ErrorResourceNotFound,
			clientCredsFlag: false,
		},
	}

	for i, tt := range tests {
		api, emailer := makeTestFixtures(tt.clientCredsFlag)
		emailer.cantEmail = tt.cantEmail

		response, err := api.ResendEmailInvitation(tt.creds, tt.userID, tt.redirURL)
		if tt.wantErr != nil {
			if err != tt.wantErr {
				t.Errorf("case %d: want=%q, got=%q", i, tt.wantErr, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("case %d: want nil err, got: %q ", i, err)
		}

		if diff := pretty.Compare(tt.wantResponse, response); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i, diff)
		}

		wantEmailer := testEmailer{
			cantEmail:       tt.cantEmail,
			lastEmail:       tt.email,
			lastClientID:    tt.wantClientID,
			lastRedirectURL: tt.redirURL,
			lastWasInvite:   true,
		}
		if diff := pretty.Compare(wantEmailer, emailer); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i, diff)
		}
	}
}

func TestRevokeRefreshToken(t *testing.T) {
	tests := []struct {
		userID   string
		toRevoke string
		before   []string // clientIDs expected before the change.
		after    []string // clientIDs expected after the change.
	}{
		{"ID-1", goodClientID, []string{goodClientID}, []string{}},
		{"ID-2", goodClientID, []string{goodClientID}, []string{}},
	}

	api, _ := makeTestFixtures(false)

	listClientsWithRefreshTokens := func(creds Creds, userID string) ([]string, error) {
		clients, err := api.ListClientsWithRefreshTokens(creds, userID)
		if err != nil {
			return nil, err
		}
		clientIDs := make([]string, len(clients))
		for i, client := range clients {
			clientIDs[i] = client.ClientID
		}
		sort.Strings(clientIDs)
		return clientIDs, nil
	}

	for i, tt := range tests {
		creds := Creds{User: user.User{ID: tt.userID}}

		gotBefore, err := listClientsWithRefreshTokens(creds, tt.userID)
		if err != nil {
			t.Errorf("case %d: list clients failed: %v", i, err)
		} else {
			if diff := pretty.Compare(tt.before, gotBefore); diff != "" {
				t.Errorf("case %d: before exp!=got: %s", i, diff)
			}
		}

		if err := api.RevokeRefreshTokensForClient(creds, tt.userID, tt.toRevoke); err != nil {
			t.Errorf("case %d: failed to revoke client: %v", i, err)
			continue
		}

		gotAfter, err := listClientsWithRefreshTokens(creds, tt.userID)
		if err != nil {
			t.Errorf("case %d: list clients failed: %v", i, err)
		} else {
			if diff := pretty.Compare(tt.after, gotAfter); diff != "" {
				t.Errorf("case %d: after exp!=got: %s", i, diff)
			}
		}
	}
}
