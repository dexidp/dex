package email

import (
	"fmt"
	htmltemplate "html/template"
	"net/url"
	"testing"
	"text/template"
	"time"

	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/key"
	"github.com/kylelemons/godebug/pretty"

	"github.com/coreos/dex/email"
	"github.com/coreos/dex/user"
)

var (
	validityWindow      = time.Hour * 1
	issuerURL           = url.URL{Host: "dex.example.com"}
	fromAddress         = "dex@example.com"
	passwordResetURL    = url.URL{Host: "dex.example.com", Path: "passwordReset"}
	verifyEmailURL      = url.URL{Host: "dex.example.com", Path: "verifyEmail"}
	acceptInvitationURL = url.URL{Host: "dex.example.com", Path: "acceptInvitation"}
	redirURL            = url.URL{Host: "client.example.com", Path: "/redirURL"}
	clientID            = "XXX"
)

type testEmailer struct {
	from, subject, text, html string
	to                        []string
	sent                      bool
}

func (t *testEmailer) SendMail(from, subject, text, html string, to ...string) error {
	t.from = from
	t.subject = subject
	t.text = text
	t.html = html
	t.to = to
	t.sent = true

	return nil
}

func makeTestFixtures() (*UserEmailer, *testEmailer, *key.PublicKey) {
	ur := user.NewUserRepoFromUsers([]user.UserWithRemoteIdentities{
		{
			User: user.User{
				ID:    "ID-1",
				Email: "id1@example.com",
				Admin: true,
			},
		}, {
			User: user.User{
				ID:    "ID-2",
				Email: "id2@example.com",
			},
		}, {
			User: user.User{
				ID:    "ID-3",
				Email: "id3@example.com",
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

	privKey, err := key.GeneratePrivateKey()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate private key, error=%v", err))
	}

	publicKey := key.NewPublicKey(privKey.JWK())
	signer := privKey.Signer()
	signerFn := func() (jose.Signer, error) {
		return signer, nil
	}

	textTemplateString := `{{define "password-reset.txt"}}{{.link}}{{end}}
{{define "verify-email.txt"}}{{.link}}{{end}}"`
	textTemplates := template.New("text")
	_, err = textTemplates.Parse(textTemplateString)
	if err != nil {
		panic(fmt.Sprintf("error parsing text templates: %v", err))
	}

	htmlTemplates := htmltemplate.New("html")

	emailer := &testEmailer{}
	tEmailer := email.NewTemplatizedEmailerFromTemplates(textTemplates, htmlTemplates, emailer)

	userEmailer := NewUserEmailer(ur, pwr, signerFn, validityWindow, issuerURL, tEmailer, fromAddress, passwordResetURL, verifyEmailURL, acceptInvitationURL)

	return userEmailer, emailer, publicKey
}

func TestSendResetPasswordEmail(t *testing.T) {
	tests := []struct {
		email      string
		hasEmailer bool

		wantUserID   string
		wantPassword string
		wantURL      bool
		wantEmail    bool
		wantErr      bool
	}{
		{
			// typical case with an emailer.
			email:      "id1@example.com",
			hasEmailer: true,

			wantURL:      false,
			wantUserID:   "ID-1",
			wantPassword: "password-1",
			wantEmail:    true,
		},
		{

			// typical case without an emailer.
			email:      "id1@example.com",
			hasEmailer: false,

			wantURL:      true,
			wantUserID:   "ID-1",
			wantPassword: "password-1",
			wantEmail:    false,
		},
		{
			// no such user.
			email:      "noone@example.com",
			hasEmailer: false,
			wantErr:    true,
		},
		{
			// user with no local password.
			email:      "id3@example.com",
			hasEmailer: false,
			wantErr:    true,
		},
	}

	for i, tt := range tests {
		ue, emailer, pubKey := makeTestFixtures()
		if !tt.hasEmailer {
			ue.SetEmailer(nil)
		}
		resetLink, err := ue.SendResetPasswordEmail(tt.email, redirURL, clientID)
		if tt.wantErr {
			if err == nil {
				t.Errorf("case %d: want non-nil err.", i)
			}
			continue
		}

		if tt.wantURL {
			if resetLink == nil {
				t.Errorf("case %d: want non-nil resetLink", i)
				continue
			}
		} else if resetLink != nil {
			t.Errorf("case %d: want resetLink==nil, got==%v", i, resetLink.String())
			continue
		}

		if tt.wantEmail {
			if !emailer.sent {
				t.Errorf("case %d: want emailer.sent", i)
				continue
			}

			// In this case the link is in the email.
			resetLink, err = url.Parse(emailer.text)
			if err != nil {
				t.Errorf("case %d: want non-nil err, got: %q", i, err)
			}
			if tt.email != emailer.to[0] {
				t.Errorf("case %d: want==%v, got==%v", i, tt.email, emailer.to[0])
			}

			if fromAddress != emailer.from {
				t.Errorf("case %d: want==%v, got==%v", i, fromAddress, emailer.from)
			}

		} else if emailer.sent {
			t.Errorf("case %d: want !emailer.sent", i)
		}

		token := resetLink.Query().Get("token")
		pr, err := user.ParseAndVerifyPasswordResetToken(token, issuerURL,
			[]key.PublicKey{*pubKey})

		if diff := pretty.Compare(redirURL, pr.Callback()); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i, diff)
		}

		if tt.wantUserID != pr.UserID() {
			t.Errorf("case %d: want==%v, got==%v", i, tt.wantUserID, pr.UserID())
		}
	}
}

func TestSendEmailVerificationEmail(t *testing.T) {
	tests := []struct {
		userID     string
		hasEmailer bool

		wantEmailAddress string
		wantURL          bool
		wantEmail        bool
		wantErr          bool
	}{
		{
			// typical case with an emailer.
			userID:     "ID-1",
			hasEmailer: true,

			wantURL:          false,
			wantEmailAddress: "id1@example.com",
			wantEmail:        true,
		},
		{

			// typical case without an emailer.
			userID:     "ID-1",
			hasEmailer: false,

			wantURL:          true,
			wantEmailAddress: "id1@example.com",
			wantEmail:        false,
		},
		{
			// no such user.
			userID:     "noone@example.com",
			hasEmailer: false,
			wantErr:    true,
		},
		{
			// user with no local password.
			userID:     "id3@example.com",
			hasEmailer: false,
			wantErr:    true,
		},
	}

	for i, tt := range tests {
		ue, emailer, pubKey := makeTestFixtures()
		if !tt.hasEmailer {
			ue.SetEmailer(nil)
		}
		verifyLink, err := ue.SendEmailVerification(tt.userID, clientID, redirURL)
		if tt.wantErr {
			if err == nil {
				t.Errorf("case %d: want non-nil err.", i)
			}
			continue
		}

		if tt.wantURL {
			if verifyLink == nil {
				t.Errorf("case %d: want non-nil verifyLink", i)
				continue
			}
		} else if verifyLink != nil {
			t.Errorf("case %d: want verifyLink==nil, got==%v", i, verifyLink.String())
			continue
		}

		if tt.wantEmail {
			if !emailer.sent {
				t.Errorf("case %d: want emailer.sent", i)
				continue
			}

			// In this case the link is in the email.
			verifyLink, err = url.Parse(emailer.text)
			if err != nil {
				t.Errorf("case %d: want non-nil err, got: %q", i, err)
			}
			if tt.wantEmailAddress != emailer.to[0] {
				t.Errorf("case %d: want==%v, got==%v", i, tt.wantEmailAddress, emailer.to[0])
			}

			if fromAddress != emailer.from {
				t.Errorf("case %d: want==%v, got==%v", i, fromAddress, emailer.from)
			}

		} else if emailer.sent {
			t.Errorf("case %d: want !emailer.sent", i)
		}

		token := verifyLink.Query().Get("token")
		ev, err := user.ParseAndVerifyEmailVerificationToken(token, issuerURL,
			[]key.PublicKey{*pubKey})

		if diff := pretty.Compare(redirURL, ev.Callback()); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i, diff)
		}

		if tt.wantEmailAddress != ev.Email() {
			t.Errorf("case %d: want==%v, got==%v", i, tt.wantEmailAddress, ev.UserID())
		}
	}
}
