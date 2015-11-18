package server

import (
	"net/http"
	"net/url"
	"time"

	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/user"
	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/key"
)

type invitationTemplateData struct {
	Error, Message string
}

type InvitationHandler struct {
	issuerURL              url.URL
	passwordResetURL       url.URL
	um                     *user.Manager
	keysFunc               func() ([]key.PublicKey, error)
	signerFunc             func() (jose.Signer, error)
	redirectValidityWindow time.Duration
}

func (h *InvitationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.handleGET(w, r)
	default:
		writeAPIError(w, http.StatusMethodNotAllowed, newAPIError(errorInvalidRequest,
			"method not allowed"))
	}
}

func (h *InvitationHandler) handleGET(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	token := q.Get("token")

	keys, err := h.keysFunc()
	if err != nil {
		log.Errorf("internal error getting public keys: %v", err)
		writeAPIError(w, http.StatusInternalServerError, newAPIError(errorServerError,
			"There's been an error processing your request."))
		return
	}

	invite, err := user.ParseAndVerifyInvitationToken(token, h.issuerURL, keys)
	if err != nil {
		log.Debugf("invalid invitation token: %v (%v)", err, token)
		writeAPIError(w, http.StatusBadRequest, newAPIError(errorInvalidRequest,
			"Your invitation could not be verified"))
		return
	}

	_, err = h.um.VerifyEmail(invite)
	if err != nil && err != user.ErrorEmailAlreadyVerified {
		// Allow AlreadyVerified folks to pass through- otherwise
		// folks who encounter an error after passing this point will
		// never be able to set their passwords.
		log.Debugf("error attempting to verify email: %v", err)
		switch err {
		case user.ErrorEVEmailDoesntMatch:
			writeAPIError(w, http.StatusBadRequest, newAPIError(errorInvalidRequest,
				"Your email does not match the email address on file"))
			return
		default:
			log.Errorf("internal error verifying email: %v", err)
			writeAPIError(w, http.StatusInternalServerError, newAPIError(errorServerError,
				"There's been an error processing your request."))
			return
		}
	}

	passwordReset := invite.PasswordReset(h.issuerURL, h.redirectValidityWindow)
	signer, err := h.signerFunc()
	if err != nil || signer == nil {
		log.Errorf("error getting signer: %v (signer: %v)", err, signer)
		writeAPIError(w, http.StatusInternalServerError, newAPIError(errorServerError,
			"There's been an error processing your request."))
		return
	}

	jwt, err := jose.NewSignedJWT(passwordReset.Claims, signer)
	if err != nil {
		log.Errorf("error constructing or signing PasswordReset from Invitation JWT: %v", err)
		writeAPIError(w, http.StatusInternalServerError, newAPIError(errorServerError,
			"There's been an error processing your request."))
		return
	}
	passwordResetToken := jwt.Encode()

	passwordResetURL := h.passwordResetURL
	newQuery := passwordResetURL.Query()
	newQuery.Set("token", passwordResetToken)
	passwordResetURL.RawQuery = newQuery.Encode()
	http.Redirect(w, r, passwordResetURL.String(), http.StatusSeeOther)
}
