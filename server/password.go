package server

import (
	"html/template"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oidc"

	"github.com/coreos/dex/client"
	clientmanager "github.com/coreos/dex/client/manager"
	"github.com/coreos/dex/pkg/log"
	sessionmanager "github.com/coreos/dex/session/manager"
	"github.com/coreos/dex/user"
	useremail "github.com/coreos/dex/user/email"
	usermanager "github.com/coreos/dex/user/manager"
)

type sendResetPasswordEmailData struct {
	Error             bool
	Message           string
	EmailSent         bool
	Email             string
	ClientID          string
	RedirectURL       string
	RedirectURLParsed url.URL
	SessionKey        string
}

type SendResetPasswordEmailHandler struct {
	issuerURL url.URL
	tpl       *template.Template
	emailer   *useremail.UserEmailer
	sm        *sessionmanager.SessionManager
	cm        *clientmanager.ClientManager
}

func (h *SendResetPasswordEmailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.handleGET(w, r)
		return
	case "POST":
		h.handlePOST(w, r)
		return
	default:
		writeAPIError(w, http.StatusMethodNotAllowed, newAPIError(errorInvalidRequest,
			"method not allowed"))
		return
	}
}

func (h *SendResetPasswordEmailHandler) handleGET(w http.ResponseWriter, r *http.Request) {
	sessionKey := r.URL.Query().Get("session_key")
	clientID := r.URL.Query().Get("client_id")
	redirectURL := r.URL.Query().Get("redirect_uri")

	if sessionKey == "" {
		writeAPIError(w, http.StatusBadRequest, newAPIError(errorInvalidRequest,
			"missing required parameters"))
		return
	}

	if clientID == "" || redirectURL == "" {
		clientID, redirectURL, err := h.getClientAndRedirectByKey(sessionKey)
		if err == nil {
			handleURL := *r.URL
			q := r.URL.Query()
			q.Set("redirect_uri", redirectURL.String())
			q.Set("client_id", clientID)
			handleURL.RawQuery = q.Encode()
			http.Redirect(w, r, handleURL.String(), http.StatusSeeOther)
			return
		}
		// Even though we could not exchange the sessionKey to get a
		// redirect URL, we can still continue as if they didn't pass
		// one in, so we don't return here.
		log.Errorf("could not exchange sessionKey: %v", err)
	}

	data := sendResetPasswordEmailData{}
	if err := h.fillData(r, &data); err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
	}

	if data.ClientID == "" {
		writeAPIError(w, http.StatusBadRequest, newAPIError(errorInvalidRequest,
			"missing required parameters"))
		return
	}

	execTemplate(w, h.tpl, data)
}

func (h *SendResetPasswordEmailHandler) fillData(r *http.Request, data *sendResetPasswordEmailData) *apiError {
	data.Email = r.FormValue("email")
	data.SessionKey = r.FormValue("session_key")
	data.ClientID = r.FormValue("client_id")
	redirectURL := r.FormValue("redirect_uri")

	if redirectURL != "" && data.ClientID != "" {
		if parsed, ok := h.validateRedirectURL(data.ClientID, redirectURL); ok {
			data.RedirectURL = redirectURL
			data.RedirectURLParsed = parsed
		} else {
			return newAPIError(errorInvalidRequest, "invalid redirect url")
		}
	}

	return nil
}

func (h *SendResetPasswordEmailHandler) handlePOST(w http.ResponseWriter, r *http.Request) {
	data := sendResetPasswordEmailData{}
	if err := h.fillData(r, &data); err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
	}

	if data.ClientID == "" {
		writeAPIError(w, http.StatusBadRequest, newAPIError(errorInvalidRequest, "client id missing"))
		return
	}

	if !user.ValidEmail(data.Email) {
		h.errPage(w, "Valid email required", http.StatusBadRequest, &data)
		return
	}

	data.EmailSent = true
	execTemplate(w, h.tpl, data)

	// We spawn this in new goroutine because we don't want anyone using timing
	// attacks to guess if an email address exists or not.
	go h.emailer.SendResetPasswordEmail(data.Email, data.RedirectURLParsed, data.ClientID)
}

func (h *SendResetPasswordEmailHandler) validateRedirectURL(clientID string, redirectURL string) (url.URL, bool) {
	parsed, err := url.Parse(redirectURL)
	if err != nil {
		log.Errorf("Error parsing redirectURL: %v", err)
		return url.URL{}, false
	}

	cm, err := h.cm.Metadata(clientID)
	if err != nil || cm == nil {
		log.Errorf("Error getting ClientMetadata: %v", err)
		return url.URL{}, false
	}

	validURL, err := client.ValidRedirectURL(parsed, cm.RedirectURIs)
	if err != nil {
		log.Errorf("Invalid redirectURL for clientID: redirectURL:%q, clientID:%q", redirectURL, clientID)
		return url.URL{}, false
	}

	return validURL, true
}

func (h *SendResetPasswordEmailHandler) errPage(w http.ResponseWriter, msg string, status int, data *sendResetPasswordEmailData) {
	data.Error = true
	data.Message = msg
	execTemplateWithStatus(w, h.tpl, data, status)
}

func (h *SendResetPasswordEmailHandler) getClientAndRedirectByKey(key string) (string, url.URL, error) {
	id, err := h.sm.GetSessionByKey(key)
	if err != nil {
		log.Errorf("error exchanging key: %v ", err)
		return "", url.URL{}, err
	}

	ses, err := h.sm.Get(id)
	if err != nil {
		log.Errorf("error getting session: %v", err)
		return "", url.URL{}, err
	}

	return ses.ClientID, ses.RedirectURL, nil
}

type resetPasswordTemplateData struct {
	Error        string
	Message      string
	Token        string
	DontShowForm bool
	Success      bool
	LoginURL     string
}

type ResetPasswordHandler struct {
	tpl       *template.Template
	emailer   *useremail.UserEmailer
	issuerURL url.URL
	um        *usermanager.UserManager
	keysFunc  func() ([]key.PublicKey, error)
}

type resetPasswordRequest struct {
	// A resetPasswordRequest starts with these objects.
	h    *ResetPasswordHandler
	r    *http.Request
	w    http.ResponseWriter
	data *resetPasswordTemplateData

	// These get filled in by sub-handlers.
	pwReset user.PasswordReset
}

func (h *ResetPasswordHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := &resetPasswordRequest{
		h:    h,
		r:    r,
		w:    w,
		data: &resetPasswordTemplateData{},
	}
	req.HandleRequest()
}

func (r *resetPasswordRequest) HandleRequest() {
	switch r.r.Method {
	case "GET":
		r.handleGET()
		return
	case "POST":
		r.handlePOST()
		return
	default:
		writeAPIError(r.w, http.StatusMethodNotAllowed, newAPIError(errorInvalidRequest,
			"method not allowed"))
		return
	}
}

func (r *resetPasswordRequest) handleGET() {
	if !r.parseAndVerifyToken() {
		return
	}
	execTemplate(r.w, r.h.tpl, r.data)
}

func (r *resetPasswordRequest) handlePOST() {
	if !r.parseAndVerifyToken() {
		return
	}

	issuerURL := r.h.issuerURL
	clientID := r.pwReset.ClientID()
	callback := r.pwReset.Callback()
	redirectURL := ""
	if callback != nil {
		redirectURL = callback.String()
	}

	loginURL := newLoginURLFromClientData(issuerURL, clientID, redirectURL)
	r.data.LoginURL = loginURL.String()

	plaintext := r.r.FormValue("password")
	_, err := r.h.um.ChangePassword(r.pwReset, plaintext)
	if err != nil {
		switch err {
		case usermanager.ErrorPasswordAlreadyChanged:
			r.data.Error = "Link Expired"
			r.data.Message = "The link in your email is no longer valid. If you need to change your password, generate a new email."
			r.data.DontShowForm = true
			execTemplateWithStatus(r.w, r.h.tpl, r.data, http.StatusBadRequest)
			return
		case user.ErrorInvalidPassword:
			r.data.Error = "Invalid Password"
			r.data.Message = "Password must be at least 6 characters in length with at least one uppercase letter and one symbol."
			execTemplateWithStatus(r.w, r.h.tpl, r.data, http.StatusBadRequest)
			return
		default:
			r.data.Error = "Error Processing Request"
			r.data.Message = "Please try again later."
			execTemplateWithStatus(r.w, r.h.tpl, r.data, http.StatusInternalServerError)
			return
		}
	}

	r.data.Success = true
	execTemplate(r.w, r.h.tpl, r.data)

	// Spawn this in new goroutine to send email in an async way
	go r.h.emailer.SendPasswordChangedEmail(r.pwReset.UserID())
}

func (r *resetPasswordRequest) parseAndVerifyToken() bool {
	keys, err := r.h.keysFunc()
	if err != nil {
		log.Errorf("problem getting keys: %v", err)
		r.data.Error = "There's been an error processing your request."
		r.data.Message = "Plesae try again later."
		execTemplateWithStatus(r.w, r.h.tpl, r.data, http.StatusInternalServerError)
		return false
	}

	token := r.r.FormValue("token")
	pwReset, err := user.ParseAndVerifyPasswordResetToken(token, r.h.issuerURL, keys)
	if err != nil {
		log.Errorf("Reset Password unverifiable token: %v", err)
		r.data.Error = "Bad Password Reset Token"
		r.data.Message = "That was not a verifiable token."
		r.data.DontShowForm = true
		execTemplateWithStatus(r.w, r.h.tpl, r.data, http.StatusBadRequest)
		return false
	}
	r.pwReset = pwReset
	r.data.Token = token
	return true
}

func newLoginURLFromClientData(issuer url.URL, clientID, redirectURL string) *url.URL {
	loginURL := issuer
	v := loginURL.Query()
	loginURL.Path = path.Join(loginURL.Path, httpPathAuth)
	v.Set("redirect_uri", redirectURL)
	v.Set("client_id", clientID)
	v.Set("response_type", "code")
	scope := append(oidc.DefaultScope, "offline_access")
	v.Set("scope", strings.Join(scope, " "))

	loginURL.RawQuery = v.Encode()
	return &loginURL
}
