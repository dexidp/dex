package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/session"
	"github.com/coreos/dex/user"
	"github.com/coreos/dex/user/manager"
	"github.com/coreos/go-oidc/oidc"
)

type formError struct {
	Field string
	Error string
}

type remoteExistsData struct {
	Login string

	Register string
}

type registerTemplateData struct {
	Error        bool
	FormErrors   []formError
	Message      string
	Email        string
	Code         string
	Password     string
	Local        bool
	RemoteExists *remoteExistsData
}

var (
	errToFormErrorMap = map[error]formError{
		user.ErrorInvalidEmail: formError{
			Field: "email",
			Error: "Please enter a valid email",
		},
		user.ErrorInvalidPassword: formError{
			Field: "password",
			Error: "Please enter a valid password",
		},
		user.ErrorDuplicateEmail: formError{
			Field: "email",
			Error: "That email is already in use; please choose another.",
		},
	}
)

func handleRegisterFunc(s *Server, tpl Template) http.HandlerFunc {

	errPage := func(w http.ResponseWriter, msg string, code string, status int) {
		data := registerTemplateData{
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

	idx := makeConnectorMap(s.Connectors)

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

		var exists bool
		exists, err = remoteIdentityExists(s.UserRepo, ses.ConnectorID, ses.Identity.ID)
		if err != nil {
			internalError(w, err)
			return
		}

		if exists {
			// we have to create a new session to be able to run the server.Login function
			newSessionKey, err := s.NewSession(ses.ConnectorID, ses.ClientID,
				ses.ClientState, ses.RedirectURL, ses.Nonce, false, ses.Scope)
			if err != nil {
				internalError(w, err)
				return
			}
			// make sure to clean up the old session
			if err = s.KillSession(code); err != nil {
				internalError(w, err)
			}

			// finally, we can create a valid redirect URL for them.
			redirURL, err := s.Login(ses.Identity, newSessionKey)
			if err != nil {
				internalError(w, err)
				return
			}

			registerURL := newLoginURLFromSession(
				s.IssuerURL, ses, true, []string{}, "")

			execTemplate(w, tpl, registerTemplateData{
				RemoteExists: &remoteExistsData{
					Login:    redirURL,
					Register: registerURL.String(),
				},
			})

			return
		}

		// determine whether or not this is a local or remote ID that is going
		// to be registered.
		idpc, ok := idx[ses.ConnectorID]
		if !ok {
			internalError(w, fmt.Errorf("no such IDPC: %v", ses.ConnectorID))
			return
		}
		_, local := idpc.(*connector.LocalConnector)

		// Does the email comes from a trusted provider?
		trustedEmail := ses.Identity.Email != "" && idpc.TrustedEmailProvider()
		validate := r.Form.Get("validate") == "1"
		formErrors := []formError{}
		email := strings.TrimSpace(r.Form.Get("email"))

		// only auto-populate the first time the page is GETted, not on
		// subsequent POSTs
		if email == "" && r.Method == "GET" {
			email = ses.Identity.Email
		}

		password := r.Form.Get("password")
		if validate {
			if email == "" || !user.ValidEmail(email) {
				formErrors = append(formErrors, formError{"email", "Please supply a valid email"})
			}
			if local && password == "" {
				formErrors = append(formErrors, formError{"password", "Please supply a valid password"})
			}
		}

		data := registerTemplateData{
			Code:     code,
			Email:    email,
			Password: password,
			Local:    local,
		}

		// If there are form errors or this is the initial request
		// (i.e. validate==false), and we are not going to auto-submit a
		// trusted email, then show the form.
		if (len(formErrors) > 0 || !validate) && !trustedEmail {
			data.FormErrors = formErrors
			if !validate {
				execTemplate(w, tpl, data)
			} else {
				execTemplateWithStatus(w, tpl, data, http.StatusBadRequest)
			}
			return
		}

		var userID string
		if local {
			userID, err = registerFromLocalConnector(
				s.UserManager,
				s.SessionManager,
				ses,
				email, password)
		} else {
			if trustedEmail {
				// in the case of a trusted email provider, make sure we are
				// getting the email address from the session, not from the
				// query string, to prevent forgeries.
				email = ses.Identity.Email
			}
			userID, err = registerFromRemoteConnector(
				s.UserManager,
				ses,
				email,
				trustedEmail)
		}
		if err == user.ErrorDuplicateEmail {
			// In this case, the user probably just forgot that they registered.
			connID, err := getConnectorForUserByEmail(s.UserRepo, email)
			if err != nil {
				internalError(w, err)
			}
			loginURL := newLoginURLFromSession(
				s.IssuerURL, ses, false, []string{connID}, "login-maybe")
			if err = s.KillSession(code); err != nil {
				log.Errorf("Error killing session: %v", err)
			}
			http.Redirect(w, r, loginURL.String(), http.StatusSeeOther)
			return
		}

		if err != nil {
			formErrors := errToFormErrors(err)
			if len(formErrors) > 0 {
				data.FormErrors = formErrors
				execTemplate(w, tpl, data)
				return
			}
			if err == user.ErrorDuplicateRemoteIdentity {
				errPage(w, "You already registered an account with this identity", "", http.StatusConflict)
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

		if !trustedEmail {
			_, err = s.UserEmailer.SendEmailVerification(usr.ID, ses.ClientID, ses.RedirectURL)

			if err != nil {
				log.Errorf("Error sending email verification: %v", err)
			}
		}

		w.Header().Set("Location", makeClientRedirectURL(
			ses.RedirectURL, code, ses.ClientState).String())
		w.WriteHeader(http.StatusSeeOther)
		return
	}
}

func makeClientRedirectURL(baseRedirURL url.URL, code, clientState string) *url.URL {
	ru := baseRedirURL
	q := ru.Query()
	q.Set("code", code)
	q.Set("state", clientState)
	ru.RawQuery = q.Encode()
	return &ru
}

func registerFromLocalConnector(userManager *manager.UserManager, sessionManager *session.SessionManager, ses *session.Session, email, password string) (string, error) {
	userID, err := userManager.RegisterWithPassword(email, password, ses.ConnectorID)
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

func registerFromRemoteConnector(userManager *manager.UserManager, ses *session.Session, email string, emailVerified bool) (string, error) {
	if ses.Identity.ID == "" {
		return "", errors.New("No Identity found in session.")
	}
	rid := user.RemoteIdentity{
		ConnectorID: ses.ConnectorID,
		ID:          ses.Identity.ID,
	}
	userID, err := userManager.RegisterWithRemoteIdentity(email, emailVerified, rid)
	if err != nil {
		return "", err
	}

	return userID, nil
}

func errToFormErrors(err error) []formError {
	fes := []formError{}
	fe, ok := errToFormErrorMap[err]
	if ok {
		fes = append(fes, fe)
	}
	return fes
}

func getConnectorForUserByEmail(ur user.UserRepo, email string) (string, error) {
	usr, err := ur.GetByEmail(nil, email)
	if err != nil {
		return "", err
	}

	rids, err := ur.GetRemoteIdentities(nil, usr.ID)
	if err != nil {
		return "", err
	}

	if len(rids) == 0 {
		return "", fmt.Errorf("No remote Identities for user %v", usr.ID)
	}

	return rids[0].ConnectorID, nil
}

func newLoginURLFromSession(issuer url.URL, ses *session.Session, register bool, connectorFilter []string, msgCode string) *url.URL {
	loginURL := issuer
	v := loginURL.Query()
	loginURL.Path = httpPathAuth
	v.Set("redirect_uri", ses.RedirectURL.String())
	v.Set("state", ses.ClientState)
	v.Set("client_id", ses.ClientID)
	if register {
		v.Set("register", "1")
	}
	if len(connectorFilter) > 0 {
		v.Set("show_connectors", strings.Join(connectorFilter, ","))
	}
	if msgCode != "" {
		v.Set("msg_code", msgCode)
	}
	if len(ses.Scope) > 0 {
		v.Set("scope", strings.Join(ses.Scope, " "))
	}

	loginURL.RawQuery = v.Encode()
	return &loginURL
}

func remoteIdentityExists(ur user.UserRepo, connectorID, id string) (bool, error) {
	_, err := ur.GetByRemoteIdentity(nil, user.RemoteIdentity{
		ConnectorID: connectorID,
		ID:          id,
	})

	if err == user.ErrorNotFound {
		return false, nil
	}

	if err == nil {
		return true, nil
	}

	return false, err
}
