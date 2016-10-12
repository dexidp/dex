package server

import (
	"net/http"
	"strings"

	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/session"
	sessionmanager "github.com/coreos/dex/session/manager"
	"github.com/coreos/dex/user"
	usermanager "github.com/coreos/dex/user/manager"
	"github.com/coreos/go-oidc/oidc"
)

const (
	maxNameLength  = 50
	maxEmailLength = 254
)

var (
	ErrorInvalidFirstName = formError{
		Field: "fname",
		Error: "Please enter a valid first name ",
	}
	ErrorInvalidLastName = formError{
		Field: "lname",
		Error: "Please enter a valid last name",
	}
	ErrorInvalidCompany = formError{
		Field: "company",
		Error: "Please enter a valid company name",
	}
	ErrorDuplicateCompanyName = formError{
		Field: "company",
		Error: "The company name is already in use; please choose another.",
	}
	ErrorInvalidEmail = formError{
		Field: "email",
		Error: "Please enter a valid email",
	}
	ErrorDuplicateEmail = formError{
		Field: "email",
		Error: "The email is already in use; please choose another.",
	}
	ErrorInvalidPassword = formError{
		Field: "password",
		Error: "Please enter a valid password",
	}
	ErrorNoConfirmPassword = formError{
		Field: "confirm-password",
		Error: "Required",
	}
	ErrorPasswordNotMatch = formError{
		Field: "password-match",
		Error: "The passwords you entered are not matched. Please enter again.",
	}
	ErrorInvalidInviteCode = formError{
		Field: "invite-code",
		Error: "Required",
	}
	ErrorTermsNotAccepted = formError{
		Field: "terms",
		Error: "Please accept the Terms of Service below in order to create an account",
	}
)

type createAccountTemplateData struct {
	Error      bool
	FormErrors []formError
	Message    string
	FirstName  string
	LastName   string
	Company    string
	Email      string
	Code       string
	InviteCode string
	LoginURL   string
}

func (d createAccountTemplateData) FieldError(fieldName string) *formError {
	for _, e := range d.FormErrors {
		if e.Field == fieldName {
			return &e
		}
	}
	return nil
}

type emailConfirmationData struct {
	FirstName   string
	Email       string
	ClientID    string
	RedirectURL string
	LoginURL    string
	Resend      string
}

func handleCreateAccountFunc(s *Server, tpl Template) http.HandlerFunc {

	errPage := func(w http.ResponseWriter, msg string, code string, status int) {
		data := createAccountTemplateData{
			Error:   true,
			Message: msg,
			Code:    code,
		}
		execTemplateWithStatus(w, tpl, data, status)
	}

	internalError := func(w http.ResponseWriter, err error) {
		log.Errorf("Internal Error during registration: %v", err)
		errPage(w, "There was a problem processing your request.", "", http.StatusInternalServerError)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			internalError(w, err)
			return
		}

		// verify the user has a valid code.
		key := r.Form.Get("code")
		sessionID, err := s.SessionManager.ExchangeKey(key)
		if err != nil {
			errPage(w, "Please authenticate before registering.", "", http.StatusUnauthorized)
			return
		}

		// create a new code for them to use next time they hit the server.
		code, err := s.SessionManager.NewSessionKey(sessionID)
		if err != nil {
			internalError(w, err)
			return
		}
		ses, err := s.SessionManager.Get(sessionID)
		if err != nil || ses == nil {
			return
		}

		loginURL := newLoginURLFromSession(s.IssuerURL, ses, false, []string{}, "")
		validate := r.Form.Get("validate") == "1"
		formErrors := []formError{}

		firstName := strings.TrimSpace(r.Form.Get("fname"))
		lastName := strings.TrimSpace(r.Form.Get("lname"))
		company := strings.TrimSpace(r.Form.Get("company"))
		email := strings.TrimSpace(r.Form.Get("email"))
		password := r.Form.Get("password")
		confirmPassword := r.Form.Get("confirm-password")
		inviteCode := r.Form.Get("invite-code")
		terms := r.Form.Get("terms")

		if validate {
			if firstName == "" || len(firstName) > maxNameLength {
				formErrors = append(formErrors, ErrorInvalidFirstName)
			}
			if lastName == "" || len(lastName) > maxNameLength {
				formErrors = append(formErrors, ErrorInvalidLastName)
			}
			if company == "" || len(company) > maxNameLength {
				formErrors = append(formErrors, ErrorInvalidCompany)
			}
			if email == "" || len(email) > maxEmailLength || !user.ValidEmail(email) {
				formErrors = append(formErrors, ErrorInvalidEmail)
			}
			if password == "" {
				formErrors = append(formErrors, ErrorInvalidPassword)
			}
			if confirmPassword == "" {
				formErrors = append(formErrors, ErrorNoConfirmPassword)
			}
			if password != "" && confirmPassword != "" && password != confirmPassword {
				formErrors = append(formErrors, ErrorPasswordNotMatch)
			}
			if inviteCode == "" {
				formErrors = append(formErrors, ErrorInvalidInviteCode)
			}
			if terms != "on" {
				formErrors = append(formErrors, ErrorTermsNotAccepted)
			}
		}

		data := createAccountTemplateData{
			Code:       code,
			FirstName:  firstName,
			LastName:   lastName,
			Company:    company,
			Email:      email,
			InviteCode: inviteCode,
			LoginURL:   loginURL.String(),
		}

		if len(formErrors) > 0 || !validate {
			data.FormErrors = formErrors
			if !validate {
				execTemplate(w, tpl, data)
			} else {
				execTemplateWithStatus(w, tpl, data, http.StatusBadRequest)
			}
			return
		}

		var userID string
		userID, err = createOrgAndOwner(
			s.UserManager,
			s.SessionManager,
			ses,
			firstName,
			lastName,
			company,
			email,
			password)

		if err != nil {
			formErrors := errToFormErrors(err)
			if len(formErrors) > 0 {
				data.FormErrors = formErrors
				execTemplate(w, tpl, data)
				return
			}

			internalError(w, err)
			return
		}

		ses, err = s.SessionManager.AttachUser(sessionID, userID)
		if err != nil {
			internalError(w, err)
			return
		}

		usr, err := s.UserRepo.Get(nil, userID)
		if err != nil {
			internalError(w, err)
			return
		}

		_, err = s.UserEmailer.SendEmailVerification(usr.ID, ses.ClientID, ses.RedirectURL)
		if err != nil {
			log.Errorf("Error sending email verification: %v", err)
		}

		execTemplate(w, s.EmailConfirmationSentTemplate, emailConfirmationData{
			FirstName:   usr.FirstName,
			Email:       usr.Email,
			ClientID:    ses.ClientID,
			RedirectURL: ses.RedirectURL.String(),
			LoginURL:    loginURL.String(),
		})
		return
	}
}

func createOrgAndOwner(
	userManager *usermanager.UserManager,
	sessionManager *sessionmanager.SessionManager,
	ses *session.Session,
	firstName, lastName, company, email, password string) (string, error) {
	userID, err := userManager.RegisterUserAndOrganization(firstName, lastName, company, email, password, ses.ConnectorID)
	if err != nil {
		return "", err
	}

	ses, err = sessionManager.AttachRemoteIdentity(ses.ID, oidc.Identity{
		ID: userID,
	})
	if err != nil {
		return "", err
	}
	return userID, nil
}
