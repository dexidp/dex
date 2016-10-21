package server

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"

	"github.com/coreos/dex/pkg/html"
	"github.com/coreos/dex/user"
	"github.com/coreos/go-oidc/oidc"
)

type testTemplate struct {
	tpl Template

	data registerTemplateData
}

func (t *testTemplate) Execute(w io.Writer, data interface{}) error {
	dataMap, ok := data.(registerTemplateData)
	if !ok {
		return errors.New("could not cast to registerTemplateData")
	}
	t.data = dataMap
	return t.tpl.Execute(w, data)
}

func TestHandleRegister(t *testing.T) {

	testIssuerAuth := testIssuerURL
	testIssuerAuth.Path = "/auth"

	str := func(s string) []string {
		return []string{s}
	}
	tests := []struct {
		// inputs
		query               url.Values
		connID              string
		attachRemote        bool
		remoteIdentityEmail string
		remoteAlreadyExists bool

		// want
		wantStatus               int
		wantFormValues           url.Values
		wantUserExists           bool
		wantRedirectURL          url.URL
		wantRegisterTemplateData *registerTemplateData
	}{
		{
			// User comes in with a valid code, redirected from the connector,
			// and is shown the form.
			query: url.Values{
				"code": []string{"code-2"},
			},
			connID: "local",

			wantStatus: http.StatusOK,
			wantFormValues: url.Values{
				"code":     str("code-2"),
				"email":    str(""),
				"password": str(""),
				"validate": str("1"),
			},
		},
		{
			// User comes in with a valid code, redirected from the connector,
			// user is created with a verified email, because it's a trusted
			// email provider.
			query: url.Values{
				"code": []string{"code-3"},
			},
			connID:              "oidc-trusted",
			remoteIdentityEmail: "test@example.com",
			attachRemote:        true,

			wantStatus:     http.StatusSeeOther,
			wantUserExists: true,
		},
		{
			// User comes in with a valid code, redirected from the connector.
			// User is redirected to dex page with msg_code "login-maybe",
			// because the remote identity already exists.
			query: url.Values{
				"code": []string{"code-3"},
			},
			connID:              "oidc-trusted",
			remoteIdentityEmail: "test@example.com",
			attachRemote:        true,
			remoteAlreadyExists: true,

			wantStatus:     http.StatusOK,
			wantUserExists: true,
			wantRegisterTemplateData: &registerTemplateData{
				RemoteExists: &remoteExistsData{
					Login: newURLWithParams(testRedirectURL, url.Values{
						"code":  []string{"code-6"},
						"state": []string{""},
					}).String(),
					Register: newURLWithParams(testIssuerAuth, url.Values{
						"client_id":     []string{testClientID},
						"redirect_uri":  []string{testRedirectURL.String()},
						"register":      []string{"1"},
						"scope":         []string{"openid"},
						"state":         []string{""},
						"response_type": []string{"code"},
						"connector_id":  []string{"oidc-trusted"},
					}).String(),
				},
			},
		},
		{
			// User comes in with a valid code, redirected from the connector,
			// user is created with a verified email, because it's a trusted
			// email provider. In addition, the email provided on the URL is
			// ignored, and instead comes from the remote identity.
			query: url.Values{
				"code":  []string{"code-3"},
				"email": []string{"sneaky@example.com"},
			},
			connID:              "oidc-trusted",
			remoteIdentityEmail: "test@example.com",
			attachRemote:        true,

			wantStatus:     http.StatusSeeOther,
			wantUserExists: true,
		},
		{
			// User comes in with a valid code, redirected from the connector,
			// it's a trusted provider, but no email so no user created, and the
			// form comes back with the code.
			query: url.Values{
				"code": []string{"code-3"},
			},
			connID:              "oidc-trusted",
			remoteIdentityEmail: "",
			attachRemote:        true,

			wantStatus:     http.StatusOK,
			wantUserExists: false,
			wantFormValues: url.Values{
				"code":     str("code-3"),
				"email":    str(""),
				"validate": str("1"),
			},
		},
		{
			// User comes in with a valid code, redirected from the connector,
			// it's a trusted provider, but the email is invalid, so no user
			// created, and the form comes back with the code.
			query: url.Values{
				"code": []string{"code-3"},
			},
			connID:              "oidc-trusted",
			remoteIdentityEmail: "notanemail",
			attachRemote:        true,

			wantStatus:     http.StatusOK,
			wantUserExists: false,
			wantFormValues: url.Values{
				"code":     str("code-3"),
				"email":    str(""),
				"validate": str("1"),
			},
		},
		{
			// User comes in with a valid code, having submitted the form, but
			// has a invalid email.
			query: url.Values{
				"code":     []string{"code-2"},
				"validate": []string{"1"},
				"email":    str(""),
				"password": str("password"),
			},
			connID:     "local",
			wantStatus: http.StatusBadRequest,
			wantFormValues: url.Values{
				"code":     str("code-2"),
				"email":    str(""),
				"password": str("password"),
				"validate": str("1"),
			},
		},
		{
			// User comes in with a valid code, having submitted the form. A new
			// user is created.
			query: url.Values{
				"code":     []string{"code-2"},
				"validate": []string{"1"},
				"email":    str("test@example.com"),
				"password": str("Password#1"),
			},
			connID:         "local",
			wantStatus:     http.StatusSeeOther,
			wantUserExists: true,
		},
		{
			// User comes in with spaces in their email, having submitted the
			// form. The email is trimmed and the user is created.
			query: url.Values{
				"code":     []string{"code-2"},
				"validate": []string{"1"},
				"email":    str("\t\ntest@example.com "),
				"password": str("Password#1"),
			},
			connID:         "local",
			wantStatus:     http.StatusSeeOther,
			wantUserExists: true,
		},
		{
			// User comes in with an invalid email, having submitted the form.
			// The email is rejected and the user is not created.
			query: url.Values{
				"code":     []string{"code-2"},
				"validate": []string{"1"},
				"email":    str("aninvalidemail"),
				"password": str("password"),
			},
			connID:     "local",
			wantStatus: http.StatusBadRequest,
			wantFormValues: url.Values{
				"code":     str("code-2"),
				"email":    str("aninvalidemail"),
				"password": str("password"),
				"validate": str("1"),
			},
		},
		{
			// User comes in with a valid code, having submitted the form, but
			// there's no password.
			query: url.Values{
				"code":     []string{"code-2"},
				"validate": []string{"1"},
				"email":    str("test@example.com"),
			},
			connID:         "local",
			wantStatus:     http.StatusBadRequest,
			wantUserExists: false,
			wantFormValues: url.Values{
				"code":     str("code-2"),
				"email":    str("test@example.com"),
				"password": str(""),
				"validate": str("1"),
			},
		},
		{
			// User comes in with a valid code, having submitted the form, but
			// there's no password, but they don't need one because connector ID
			// is oidc.
			query: url.Values{
				"code":     []string{"code-3"},
				"validate": []string{"1"},
				"email":    str("test@example.com"),
			},
			connID:         "oidc",
			attachRemote:   true,
			wantStatus:     http.StatusSeeOther,
			wantUserExists: true,
		},
		{
			// Same as before, but missing a code.
			query: url.Values{
				"validate": []string{"1"},
				"email":    str("test@example.com"),
			},
			connID:         "oidc",
			attachRemote:   true,
			wantStatus:     http.StatusUnauthorized,
			wantUserExists: false,
			wantFormValues: url.Values{
				"code":     str(""),
				"email":    str(""),
				"validate": str("1"),
			},
		},
	}

	for i, tt := range tests {
		f, err := makeTestFixtures()
		if err != nil {
			t.Fatalf("case %d: could not make test fixtures: %v", i, err)
		}

		if tt.remoteAlreadyExists {
			f.userRepo.Create(nil, user.User{
				ID:            "register-test-new-user",
				Email:         tt.remoteIdentityEmail,
				EmailVerified: true,
			})

			f.userRepo.AddRemoteIdentity(nil, "register-test-new-user",
				user.RemoteIdentity{
					ID:          "remoteID",
					ConnectorID: tt.connID,
				})
		}

		key, err := f.srv.NewSession(tt.connID, testClientID, "", f.redirectURL, "", true, []string{"openid"})
		t.Logf("case %d: key for NewSession: %v", i, key)

		if tt.attachRemote {
			sesID, err := f.sessionManager.ExchangeKey(key)
			if err != nil {
				t.Fatalf("case %d: expected non-nil error: %v", i, err)
			}
			ses, err := f.sessionManager.Get(sesID)
			if err != nil {
				t.Fatalf("case %d: expected non-nil error: %v", i, err)
			}

			_, err = f.sessionManager.AttachRemoteIdentity(ses.ID, oidc.Identity{
				ID:    "remoteID",
				Email: tt.remoteIdentityEmail,
			})

			key, err := f.sessionManager.NewSessionKey(sesID)
			if err != nil {
				t.Fatalf("case %d: expected non-nil error: %v", i, err)
			}
			t.Logf("case %d: key for NewSession: %v", i, key)
		}

		tpl := &testTemplate{tpl: f.srv.RegisterTemplate}
		hdlr := handleRegisterFunc(f.srv, tpl)

		w := httptest.NewRecorder()
		u := "http://server.example.com"
		req, err := http.NewRequest("POST", u, strings.NewReader(tt.query.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		if err != nil {
			t.Errorf("case %d: unable to form HTTP request: %v", i, err)
		}

		hdlr.ServeHTTP(w, req)

		if tt.wantRedirectURL.String() != "" {
			locationHdr := w.HeaderMap.Get("Location")
			redirURL, err := url.Parse(locationHdr)
			if err != nil {
				t.Errorf("case %d: unexpected error parsing url %q: %q", i, locationHdr, err)
			} else {
				if diff := pretty.Compare(*redirURL, tt.wantRedirectURL); diff != "" {
					t.Errorf("case %d: Compare(redirURL, tt.wantRedirectURL) = %v", i, diff)
				}
			}
		}

		if tt.wantStatus != w.Code {
			t.Errorf("case %d: wantStatus=%v, got=%v", i, tt.wantStatus, w.Code)
		}

		_, err = f.userRepo.GetByEmail(nil, "test@example.com")
		if tt.wantUserExists {
			if err != nil {
				t.Errorf("case %d: user not created: %v", i, err)
			}
		} else if err != user.ErrorNotFound {
			t.Errorf("case %d: unexpected error looking up user: want=%v, got=%v ", i, user.ErrorNotFound, err)
		}

		values, err := html.FormValues("#registerForm", w.Body)
		if err != nil {
			t.Errorf("case %d: could not parse form: %v", i, err)
		}

		if diff := pretty.Compare(tt.wantFormValues, values); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i, diff)
		}

		if tt.wantRegisterTemplateData != nil {
			if diff := pretty.Compare(*tt.wantRegisterTemplateData, tpl.data); diff != "" {
				t.Errorf("case %d: Compare(tt.wantRegisterTemplateData, tpl.data) = %v",
					i, diff)
			}
		}
	}
}

func newURLWithParams(u url.URL, values url.Values) *url.URL {
	newU := u
	newU.RawQuery = values.Encode()
	return &newU
}
