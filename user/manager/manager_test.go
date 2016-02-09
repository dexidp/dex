package manager

import (
	"net/url"
	"testing"
	"time"

	"github.com/coreos/go-oidc/jose"
	"github.com/jonboulle/clockwork"
	"github.com/kylelemons/godebug/pretty"

	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/db"
	"github.com/coreos/dex/user"
)

type testFixtures struct {
	ur    user.UserRepo
	pwr   user.PasswordInfoRepo
	ccr   connector.ConnectorConfigRepo
	mgr   *UserManager
	clock clockwork.Clock
}

func makeTestFixtures() *testFixtures {
	f := &testFixtures{}
	f.clock = clockwork.NewFakeClock()

	dbMap := db.NewMemDB()
	f.ur = func() user.UserRepo {
		repo, err := db.NewUserRepoFromUsers(dbMap, []user.UserWithRemoteIdentities{
			{
				User: user.User{
					ID:    "ID-1",
					Email: "Email-1@example.com",
				},
				RemoteIdentities: []user.RemoteIdentity{
					{
						ConnectorID: "local",
						ID:          "1",
					},
				},
			}, {
				User: user.User{
					ID:            "ID-2",
					Email:         "Email-2@example.com",
					EmailVerified: true,
				},
				RemoteIdentities: []user.RemoteIdentity{
					{
						ConnectorID: "local",
						ID:          "2",
					},
				},
			},
		})
		if err != nil {
			panic("Failed to create user repo: " + err.Error())
		}
		return repo
	}()

	f.pwr = user.NewPasswordInfoRepoFromPasswordInfos([]user.PasswordInfo{
		{
			UserID:   "ID-1",
			Password: []byte("password-1"),
		},
		{
			UserID:   "ID-2",
			Password: []byte("password-2"),
		},
	})
	f.ccr = connector.NewConnectorConfigRepoFromConfigs([]connector.ConnectorConfig{
		&connector.LocalConnectorConfig{ID: "local"},
	})
	f.mgr = NewUserManager(f.ur, f.pwr, f.ccr, db.TransactionFactory(dbMap), ManagerOptions{})
	f.mgr.Clock = f.clock
	return f
}

func TestRegisterWithRemoteIdentity(t *testing.T) {
	tests := []struct {
		email         string
		emailVerified bool
		rid           user.RemoteIdentity
		err           error
	}{
		{
			email:         "email@example.com",
			emailVerified: false,
			rid: user.RemoteIdentity{
				ConnectorID: "local",
				ID:          "1234",
			},
			err: nil,
		},
		{
			emailVerified: false,
			rid: user.RemoteIdentity{
				ConnectorID: "local",
				ID:          "1234",
			},
			err: user.ErrorInvalidEmail,
		},
		{
			email:         "email@example.com",
			emailVerified: false,
			rid: user.RemoteIdentity{
				ConnectorID: "local",
				ID:          "1",
			},
			err: user.ErrorDuplicateRemoteIdentity,
		},
		{
			email:         "anotheremail@example.com",
			emailVerified: false,
			rid: user.RemoteIdentity{
				ConnectorID: "idonotexist",
				ID:          "1",
			},
			err: connector.ErrorNotFound,
		},
	}

	for i, tt := range tests {
		f := makeTestFixtures()
		userID, err := f.mgr.RegisterWithRemoteIdentity(
			tt.email,
			tt.emailVerified,
			tt.rid)

		if tt.err != nil {
			if tt.err != err {
				t.Errorf("case %d: want=%q, got=%q", i, tt.err, err)
			}
			continue
		}

		usr, err := f.ur.Get(nil, userID)
		if err != nil {
			t.Errorf("case %d: err != nil: %q", i, err)
		}

		if usr.Email != tt.email {
			t.Errorf("case %d: user.Email: want=%q, got=%q", i, tt.email, usr.Email)
		}
		if usr.EmailVerified != tt.emailVerified {
			t.Errorf("case %d: user.EmailVerified: want=%v, got=%v", i, tt.emailVerified, usr.EmailVerified)
		}

		ridUSR, err := f.ur.GetByRemoteIdentity(nil, tt.rid)
		if err != nil {
			t.Errorf("case %d: err != nil: %q", i, err)
		}
		if diff := pretty.Compare(usr, ridUSR); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i, diff)
		}
	}
}

func TestRegisterWithPassword(t *testing.T) {
	tests := []struct {
		email     string
		plaintext string
		err       error
	}{
		{
			email:     "email@example.com",
			plaintext: "secretpassword123",
			err:       nil,
		},
		{
			plaintext: "secretpassword123",
			err:       user.ErrorInvalidEmail,
		},
		{
			email: "email@example.com",
			err:   user.ErrorInvalidPassword,
		},
	}

	for i, tt := range tests {
		f := makeTestFixtures()
		connID := "local"
		userID, err := f.mgr.RegisterWithPassword(
			tt.email,
			tt.plaintext,
			connID)

		if tt.err != nil {
			if tt.err != err {
				t.Errorf("case %d: want=%q, got=%q", i, tt.err, err)
			}
			continue
		}

		usr, err := f.ur.Get(nil, userID)
		if err != nil {
			t.Errorf("case %d: err != nil: %q", i, err)
		}

		if usr.Email != tt.email {
			t.Errorf("case %d: user.Email: want=%q, got=%q", i, tt.email, usr.Email)
		}
		if usr.EmailVerified != false {
			t.Errorf("case %d: user.EmailVerified: want=%v, got=%v", i, false, usr.EmailVerified)
		}

		ridUSR, err := f.ur.GetByRemoteIdentity(nil, user.RemoteIdentity{
			ID:          userID,
			ConnectorID: connID,
		})
		if err != nil {
			t.Errorf("case %d: err != nil: %q", i, err)
		}
		if diff := pretty.Compare(usr, ridUSR); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i, diff)
		}

		pwi, err := f.pwr.Get(nil, userID)
		if err != nil {
			t.Errorf("case %d: err != nil: %q", i, err)
		}
		ident, err := pwi.Authenticate(tt.plaintext)
		if err != nil {
			t.Errorf("case %d: err != nil: %q", i, err)
		}
		if ident.ID != userID {
			t.Errorf("case %d: ident.ID: want=%q, got=%q", i, userID, ident.ID)
		}

		_, err = pwi.Authenticate(tt.plaintext + "WRONG")
		if err == nil {
			t.Errorf("case %d: want non-nil err", i)
		}
	}
}

func TestVerifyEmail(t *testing.T) {
	now := time.Now()
	issuer, _ := url.Parse("http://example.com")
	clientID := "myclient"
	callback := "http://client.example.com/callback"
	expires := time.Hour * 3

	makeClaims := func(usr user.User) jose.Claims {
		return map[string]interface{}{
			"iss": issuer.String(),
			"aud": clientID,
			user.ClaimEmailVerificationCallback: callback,
			user.ClaimEmailVerificationEmail:    usr.Email,
			"exp": float64(now.Add(expires).Unix()),
			"sub": usr.ID,
			"iat": float64(now.Unix()),
		}
	}

	tests := []struct {
		evClaims jose.Claims
		wantErr  bool
	}{
		{
			// happy path
			evClaims: makeClaims(user.User{ID: "ID-1", Email: "Email-1@example.com"}),
		},
		{
			// non-matching email
			evClaims: makeClaims(user.User{ID: "ID-1", Email: "Email-2@example.com"}),
			wantErr:  true,
		},
		{
			// already verified email
			evClaims: makeClaims(user.User{ID: "ID-2", Email: "Email-2@example.com"}),
			wantErr:  true,
		},
		{
			// non-existent user.
			evClaims: makeClaims(user.User{ID: "ID-UNKNOWN", Email: "noone@example.com"}),
			wantErr:  true,
		},
	}

	for i, tt := range tests {
		f := makeTestFixtures()
		cb, err := f.mgr.VerifyEmail(user.EmailVerification{tt.evClaims})
		if tt.wantErr {
			if err == nil {
				t.Errorf("case %d: want non-nil err", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("case %d: want err=nil got=%q", i, err)
		}

		if cb.String() != tt.evClaims[user.ClaimEmailVerificationCallback] {
			t.Errorf("case %d: want=%q, got=%q", i, cb.String(),
				tt.evClaims[user.ClaimEmailVerificationCallback])
		}
	}
}

func TestChangePassword(t *testing.T) {
	now := time.Now()
	issuer, _ := url.Parse("http://example.com")
	clientID := "myclient"
	callback := "http://client.example.com/callback"
	expires := time.Hour * 3
	password := "password-1"

	makeClaims := func(usrID, callback string) jose.Claims {
		return map[string]interface{}{
			"iss": issuer.String(),
			"aud": clientID,
			user.ClaimPasswordResetCallback: callback,
			user.ClaimPasswordResetPassword: password,
			"exp": float64(now.Add(expires).Unix()),
			"sub": usrID,
			"iat": float64(now.Unix()),
		}
	}

	tests := []struct {
		pwrClaims   jose.Claims
		newPassword string
		wantErr     bool
	}{
		{
			// happy path
			pwrClaims:   makeClaims("ID-1", callback),
			newPassword: "password-1.1",
		},
		{
			// happy path with no callback
			pwrClaims:   makeClaims("ID-1", ""),
			newPassword: "password-1.1",
		},
		{
			// passwords don't match changed
			pwrClaims:   makeClaims("ID-2", callback),
			newPassword: "password-1.1",
			wantErr:     true,
		},
		{
			// user doesn't exist
			pwrClaims:   makeClaims("ID-123", callback),
			newPassword: "password-1.1",
			wantErr:     true,
		},
	}

	for i, tt := range tests {
		f := makeTestFixtures()
		cb, err := f.mgr.ChangePassword(user.PasswordReset{tt.pwrClaims}, tt.newPassword)
		if tt.wantErr {
			if err == nil {
				t.Errorf("case %d: want non-nil err", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("case %d: want err=nil got=%q", i, err)
			continue
		}

		var cbString string
		if cb != nil {
			cbString = cb.String()
		}
		if cbString != tt.pwrClaims[user.ClaimPasswordResetCallback] {
			t.Errorf("case %d: want=%q, got=%q", i, cb.String(),
				tt.pwrClaims[user.ClaimPasswordResetCallback])
		}
	}
}

func TestCreateUser(t *testing.T) {
	tests := []struct {
		usr      user.User
		hashedPW user.Password
		localID  string // defaults to "local"

		wantErr bool
	}{
		{
			usr: user.User{
				DisplayName: "Bob Exampleson",
				Email:       "bob@example.com",
			},
			hashedPW: user.Password("I am a hash"),
		},
		{
			usr: user.User{
				DisplayName: "Al Adminson",
				Email:       "al@example.com",
				Admin:       true,
			},
			hashedPW: user.Password("I am a hash"),
		},
		{
			usr: user.User{
				DisplayName: "Ed Emailless",
			},
			hashedPW: user.Password("I am a hash"),
			wantErr:  true,
		},
		{
			usr: user.User{
				DisplayName: "Eric Exampleson",
				Email:       "eric@example.com",
			},
			hashedPW: user.Password("I am a hash"),
			localID:  "abadlocalid",
			wantErr:  true,
		},
	}

	for i, tt := range tests {
		f := makeTestFixtures()
		localID := "local"
		if tt.localID != "" {
			localID = tt.localID
		}
		id, err := f.mgr.CreateUser(tt.usr, tt.hashedPW, localID)
		if tt.wantErr {
			if err == nil {
				t.Errorf("case %d: want non-nil err", i)
			}
			continue
		}
		if id == "" {
			t.Errorf("case %d: want non-empty id", i)
		}

		if err != nil {
			t.Errorf("case %d: unexpected err: %v", i, err)
			continue
		}

		gotUsr, err := f.ur.Get(nil, id)
		if err != nil {
			t.Errorf("case %d: unexpected err: %v", i, err)
		}

		tt.usr.ID = id
		tt.usr.CreatedAt = f.clock.Now()
		if diff := pretty.Compare(tt.usr, gotUsr); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i, diff)
		}

		pwi, err := f.pwr.Get(nil, id)
		if err != nil {
			t.Errorf("case %d: unexpected err: %v", i, err)
		}

		if string(pwi.Password) != string(tt.hashedPW) {
			t.Errorf("case %d: want=%q, got=%q", i, tt.hashedPW, pwi.Password)
		}

		ridUser, err := f.ur.GetByRemoteIdentity(nil, user.RemoteIdentity{
			ID:          id,
			ConnectorID: "local",
		})
		if err != nil {
			t.Errorf("case %d: err != nil: %q", i, err)
		}
		if diff := pretty.Compare(gotUsr, ridUser); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i, diff)
		}
	}
}
