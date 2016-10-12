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
)

type testCreateAccountTemplate struct {
	tpl Template

	data createAccountTemplateData
}

func (t *testCreateAccountTemplate) Execute(w io.Writer, data interface{}) error {
	dataMap, ok := data.(createAccountTemplateData)
	if !ok {
		return errors.New("could not cast to createAccountTemplateData")
	}
	t.data = dataMap
	return t.tpl.Execute(w, data)
}

func TestHandleCreateAccount(t *testing.T) {

	testIssuerAuth := testIssuerURL
	testIssuerAuth.Path = "/auth"
	testLoginURL := "http://server.example.com/auth?client_id=client.example.com&connector_id=local&redirect_uri=http%3A%2F%2Fclient.example.com%2Fcallback&response_type=code&scope=openid&state="

	str := func(s string) []string {
		return []string{s}
	}
	tests := []struct {
		// inputs
		query             url.Values
		connID            string
		userAlreadyExists bool

		// want
		wantStatus                    int
		wantFormValues                url.Values
		wantUserExists                bool
		wantCreateAccountTemplateData *createAccountTemplateData
	}{
		{
			// User comes in with a valid code, redirected from the connector,
			// and is shown the form.
			query: url.Values{
				"code": str("code-2"),
			},
			connID: "local",

			wantStatus: http.StatusOK,
			wantFormValues: url.Values{
				"code":        str("code-3"),
				"fname":       str(""),
				"lname":       str(""),
				"company":     str(""),
				"email":       str(""),
				"invite-code": str(""),
				"validate":    str("1"),
			},
		},
		{
			// User comes in with a valid code, having submitted the form, but
			// has no first name.
			query: url.Values{
				"code":             str("code-2"),
				"validate":         str("1"),
				"fname":            str(""),
				"lname":            str("last-name"),
				"company":          str("company"),
				"email":            str("test@example.com"),
				"password":         str("ValidPassword#1"),
				"confirm-password": str("ValidPassword#1"),
				"invite-code":      str("123"),
				"terms":            str("on"),
			},
			connID:     "local",
			wantStatus: http.StatusBadRequest,
			wantCreateAccountTemplateData: &createAccountTemplateData{
				FormErrors: []formError{ErrorInvalidFirstName},
				FirstName:  "",
				LastName:   "last-name",
				Company:    "company",
				Email:      "test@example.com",
				Code:       "code-3",
				InviteCode: "123",
				LoginURL:   testLoginURL,
			},
		},
		{
			// User comes in with a valid code, having submitted the form, but
			// has no last name.
			query: url.Values{
				"code":             str("code-2"),
				"validate":         str("1"),
				"fname":            str("first-name"),
				"lname":            str(""),
				"company":          str("company"),
				"email":            str("test@example.com"),
				"password":         str("ValidPassword#1"),
				"confirm-password": str("ValidPassword#1"),
				"invite-code":      str("123"),
				"terms":            str("on"),
			},
			connID:     "local",
			wantStatus: http.StatusBadRequest,
			wantCreateAccountTemplateData: &createAccountTemplateData{
				FormErrors: []formError{ErrorInvalidLastName},
				FirstName:  "first-name",
				LastName:   "",
				Company:    "company",
				Email:      "test@example.com",
				Code:       "code-3",
				InviteCode: "123",
				LoginURL:   testLoginURL,
			},
		},
		{
			// User comes in with a valid code, having submitted the form, but
			// has no company name.
			query: url.Values{
				"code":             str("code-2"),
				"validate":         str("1"),
				"fname":            str("first-name"),
				"lname":            str("last-name"),
				"company":          str(""),
				"email":            str("test@example.com"),
				"password":         str("ValidPassword#1"),
				"confirm-password": str("ValidPassword#1"),
				"invite-code":      str("123"),
				"terms":            str("on"),
			},
			connID:     "local",
			wantStatus: http.StatusBadRequest,
			wantCreateAccountTemplateData: &createAccountTemplateData{
				FormErrors: []formError{ErrorInvalidCompany},
				FirstName:  "first-name",
				LastName:   "last-name",
				Company:    "",
				Email:      "test@example.com",
				Code:       "code-3",
				InviteCode: "123",
				LoginURL:   testLoginURL,
			},
		},
		{
			// User comes in with a valid code, having submitted the form, but
			// has no company name.
			query: url.Values{
				"code":             str("code-2"),
				"validate":         str("1"),
				"fname":            str("first-name"),
				"lname":            str("last-name"),
				"company":          str("OrgName-1"),
				"email":            str("test@example.com"),
				"password":         str("ValidPassword#1"),
				"confirm-password": str("ValidPassword#1"),
				"invite-code":      str("123"),
				"terms":            str("on"),
			},
			connID:     "local",
			wantStatus: http.StatusOK,
			wantCreateAccountTemplateData: &createAccountTemplateData{
				FormErrors: []formError{ErrorDuplicateCompanyName},
				FirstName:  "first-name",
				LastName:   "last-name",
				Company:    "OrgName-1",
				Email:      "test@example.com",
				Code:       "code-3",
				InviteCode: "123",
				LoginURL:   testLoginURL,
			},
		},
		{
			// User comes in with a valid code, having submitted the form, but
			// the provided company name already exists.
			query: url.Values{
				"code":             str("code-2"),
				"validate":         str("1"),
				"fname":            str("first-name"),
				"lname":            str("last-name"),
				"company":          str("company"),
				"email":            str(""),
				"password":         str("ValidPassword#1"),
				"confirm-password": str("ValidPassword#1"),
				"invite-code":      str("123"),
				"terms":            str("on"),
			},
			connID:     "local",
			wantStatus: http.StatusBadRequest,
			wantCreateAccountTemplateData: &createAccountTemplateData{
				FormErrors: []formError{ErrorInvalidEmail},
				FirstName:  "first-name",
				LastName:   "last-name",
				Company:    "company",
				Email:      "",
				Code:       "code-3",
				InviteCode: "123",
				LoginURL:   testLoginURL,
			},
		},
		{
			// User comes in with a valid code, having submitted the form. A new
			// user is created.
			query: url.Values{
				"code":             str("code-2"),
				"validate":         str("1"),
				"fname":            str("first-name"),
				"lname":            str("last-name"),
				"company":          str("company"),
				"email":            str("test@example.com"),
				"password":         str("ValidPassword#1"),
				"confirm-password": str("ValidPassword#1"),
				"invite-code":      str("123"),
				"terms":            str("on"),
			},
			connID:         "local",
			wantStatus:     http.StatusOK,
			wantUserExists: true,
		},
		{
			// User comes in with spaces in their email, having submitted the
			// form. The email is trimmed and the user is created.
			query: url.Values{
				"code":             str("code-2"),
				"validate":         str("1"),
				"fname":            str("first-name"),
				"lname":            str("last-name"),
				"company":          str("company"),
				"email":            str("\t\ntest@example.com "),
				"password":         str("ValidPassword#1"),
				"confirm-password": str("ValidPassword#1"),
				"invite-code":      str("123"),
				"terms":            str("on"),
			},
			connID:         "local",
			wantStatus:     http.StatusOK,
			wantUserExists: true,
		},
		{
			// User comes in with an invalid email, having submitted the form.
			// The email is rejected and the user is not created.
			query: url.Values{
				"code":             str("code-2"),
				"validate":         str("1"),
				"fname":            str("first-name"),
				"lname":            str("last-name"),
				"company":          str("company"),
				"email":            str("aninvalidemail"),
				"password":         str("ValidPassword#1"),
				"confirm-password": str("ValidPassword#1"),
				"invite-code":      str("123"),
				"terms":            str("on"),
			},
			connID:     "local",
			wantStatus: http.StatusBadRequest,
			wantCreateAccountTemplateData: &createAccountTemplateData{
				FormErrors: []formError{ErrorInvalidEmail},
				FirstName:  "first-name",
				LastName:   "last-name",
				Company:    "company",
				Email:      "aninvalidemail",
				Code:       "code-3",
				InviteCode: "123",
				LoginURL:   testLoginURL,
			},
		},
		{
			// User comes in with an existing email, having submitted the form.
			// The email is rejected and the user is not created.
			query: url.Values{
				"code":             str("code-2"),
				"validate":         str("1"),
				"fname":            str("first-name"),
				"lname":            str("last-name"),
				"company":          str("company"),
				"email":            str(testUserEmail1),
				"password":         str("ValidPassword#1"),
				"confirm-password": str("ValidPassword#1"),
				"invite-code":      str("123"),
				"terms":            str("on"),
			},
			userAlreadyExists: true,
			connID:            "local",
			wantStatus:        http.StatusOK,
			wantUserExists:    true,
			wantCreateAccountTemplateData: &createAccountTemplateData{
				FormErrors: []formError{ErrorDuplicateEmail},
				FirstName:  "first-name",
				LastName:   "last-name",
				Company:    "company",
				Email:      testUserEmail1,
				Code:       "code-3",
				InviteCode: "123",
				LoginURL:   testLoginURL,
			},
		},
		{
			// User comes in with an existing email in upper case, having submitted the form.
			// The email is rejected and the user is not created.
			query: url.Values{
				"code":             str("code-2"),
				"validate":         str("1"),
				"fname":            str("first-name"),
				"lname":            str("last-name"),
				"company":          str("company"),
				"email":            str(strings.ToUpper(testUserEmail1)),
				"password":         str("ValidPassword#1"),
				"confirm-password": str("ValidPassword#1"),
				"invite-code":      str("123"),
				"terms":            str("on"),
			},
			userAlreadyExists: true,
			connID:            "local",
			wantStatus:        http.StatusOK,
			wantUserExists:    true,
			wantCreateAccountTemplateData: &createAccountTemplateData{
				FormErrors: []formError{ErrorDuplicateEmail},
				FirstName:  "first-name",
				LastName:   "last-name",
				Company:    "company",
				Email:      strings.ToUpper(testUserEmail1),
				Code:       "code-3",
				InviteCode: "123",
				LoginURL:   testLoginURL,
			},
		},
		{
			// User comes in with a valid code, having submitted the form, but
			// there's no password or confirm-password.
			query: url.Values{
				"code":        str("code-2"),
				"validate":    str("1"),
				"fname":       str("first-name"),
				"lname":       str("last-name"),
				"company":     str("company"),
				"email":       str("test@example.com"),
				"invite-code": str("123"),
				"terms":       str("on"),
			},
			connID:     "local",
			wantStatus: http.StatusBadRequest,
			wantCreateAccountTemplateData: &createAccountTemplateData{
				FormErrors: []formError{ErrorInvalidPassword, ErrorNoConfirmPassword},
				FirstName:  "first-name",
				LastName:   "last-name",
				Company:    "company",
				Email:      "test@example.com",
				Code:       "code-3",
				InviteCode: "123",
				LoginURL:   testLoginURL,
			},
		},
		{
			// User comes in with a valid code, having submitted the form, but
			// the password does not match the requirements.
			query: url.Values{
				"code":             str("code-2"),
				"validate":         str("1"),
				"fname":            str("first-name"),
				"lname":            str("last-name"),
				"company":          str("company"),
				"email":            str("test@example.com"),
				"password":         str("invalidpassword"),
				"confirm-password": str("invalidpassword"),
				"invite-code":      str("123"),
				"terms":            str("on"),
			},
			connID:     "local",
			wantStatus: http.StatusOK,
			wantCreateAccountTemplateData: &createAccountTemplateData{
				FormErrors: []formError{ErrorInvalidPassword},
				FirstName:  "first-name",
				LastName:   "last-name",
				Company:    "company",
				Email:      "test@example.com",
				Code:       "code-3",
				InviteCode: "123",
				LoginURL:   testLoginURL,
			},
		},
		{
			// User comes in with a valid code, having submitted the form, but
			// the two passwords does not match.
			query: url.Values{
				"code":             str("code-2"),
				"validate":         str("1"),
				"fname":            str("first-name"),
				"lname":            str("last-name"),
				"company":          str("company"),
				"email":            str("test@example.com"),
				"password":         str("ValidPassword#1"),
				"confirm-password": str("ValidPassword#2"),
				"invite-code":      str("123"),
				"terms":            str("on"),
			},
			connID:     "local",
			wantStatus: http.StatusBadRequest,
			wantCreateAccountTemplateData: &createAccountTemplateData{
				FormErrors: []formError{ErrorPasswordNotMatch},
				FirstName:  "first-name",
				LastName:   "last-name",
				Company:    "company",
				Email:      "test@example.com",
				Code:       "code-3",
				InviteCode: "123",
				LoginURL:   testLoginURL,
			},
		},
		{
			// User comes in with a valid code, having submitted the form, but
			// has no invite code.
			query: url.Values{
				"code":             str("code-2"),
				"validate":         str("1"),
				"fname":            str("first-name"),
				"lname":            str("last-name"),
				"company":          str("company"),
				"email":            str("test@example.com"),
				"password":         str("ValidPassword#1"),
				"confirm-password": str("ValidPassword#1"),
				"invite-code":      str(""),
				"terms":            str("on"),
			},
			connID:     "local",
			wantStatus: http.StatusBadRequest,
			wantCreateAccountTemplateData: &createAccountTemplateData{
				FormErrors: []formError{ErrorInvalidInviteCode},
				FirstName:  "first-name",
				LastName:   "last-name",
				Company:    "company",
				Email:      "test@example.com",
				Code:       "code-3",
				InviteCode: "",
				LoginURL:   testLoginURL,
			},
		},
		{
			// User comes in with a valid code, having submitted the form, but
			// has not accepted the terms of use.
			query: url.Values{
				"code":             str("code-2"),
				"validate":         str("1"),
				"fname":            str("first-name"),
				"lname":            str("last-name"),
				"company":          str("company"),
				"email":            str("test@example.com"),
				"password":         str("ValidPassword#1"),
				"confirm-password": str("ValidPassword#1"),
				"invite-code":      str("123"),
				"terms":            str("off"),
			},
			connID:     "local",
			wantStatus: http.StatusBadRequest,
			wantCreateAccountTemplateData: &createAccountTemplateData{
				FormErrors: []formError{ErrorTermsNotAccepted},
				FirstName:  "first-name",
				LastName:   "last-name",
				Company:    "company",
				Email:      "test@example.com",
				Code:       "code-3",
				InviteCode: "123",
				LoginURL:   testLoginURL,
			},
		},
	}

	for i, tt := range tests {
		f, err := makeTestFixtures()
		if err != nil {
			t.Fatalf("case %d: could not make test fixtures: %v", i, err)
		}

		key, err := f.srv.NewSession(tt.connID, testClientID, "", f.redirectURL, "", true, []string{"openid"})
		t.Logf("case %d: key for NewSession: %v", i, key)

		tpl := &testCreateAccountTemplate{tpl: f.srv.CreateAccountTemplate}
		hdlr := handleCreateAccountFunc(f.srv, tpl)

		w := httptest.NewRecorder()
		u := "http://server.example.com"
		req, err := http.NewRequest("POST", u, strings.NewReader(tt.query.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		if err != nil {
			t.Errorf("case %d: unable to form HTTP request: %v", i, err)
		}

		hdlr.ServeHTTP(w, req)

		if tt.wantStatus != w.Code {
			t.Errorf("case %d: wantStatus=%v, got=%v", i, tt.wantStatus, w.Code)
		}

		if tt.userAlreadyExists {
			_, err = f.userRepo.GetByEmail(nil, testUserEmail1)
		} else {
			_, err = f.userRepo.GetByEmail(nil, "test@example.com")
		}

		if tt.wantUserExists {
			if err != nil {
				t.Errorf("case %d: user not created: %v", i, err)
			}
		} else if err != user.ErrorNotFound {
			t.Errorf("case %d: unexpected error looking up user: want=%v, got=%v ", i, user.ErrorNotFound, err)
		}

		if tt.wantFormValues != nil {
			values, err := html.FormValues("#createAccountForm", w.Body)
			if err != nil {
				t.Errorf("case %d: could not parse form: %v", i, err)
			}

			if diff := pretty.Compare(tt.wantFormValues, values); diff != "" {
				t.Errorf("case %d: Compare(want, got) = %v", i, diff)
			}
		}

		if tt.wantCreateAccountTemplateData != nil {
			if diff := pretty.Compare(*tt.wantCreateAccountTemplateData, tpl.data); diff != "" {
				t.Errorf("case %d: Compare(tt.wantCreateAccountTemplateData, tpl.data) = %v",
					i, diff)
			}
		}
	}
}
