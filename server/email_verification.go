package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oidc"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/pkg/log"
	sessionmanager "github.com/coreos/dex/session/manager"
	"github.com/coreos/dex/user"
	useremail "github.com/coreos/dex/user/email"
	"github.com/coreos/dex/user/manager"
)

// handleVerifyEmailResendFunc will resend an email-verification email given a valid JWT for the user and a redirect URL.
// This handler is meant to be wrapped in clientTokenMiddleware, so a valid
// bearer token for the client is expected to be present.
// The user's JWT should be in the "token" parameter and the redirect URL should
// be in the "redirect_uri" param.
func handleVerifyEmailResendFunc(
	issuerURL url.URL,
	srvKeysFunc func() ([]key.PublicKey, error),
	emailer *useremail.UserEmailer,
	userRepo user.UserRepo,
	clientRepo client.ClientRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var params struct {
			Token       string `json:"token"`
			RedirectURI string `json:"redirectURI"`
		}
		err := decoder.Decode(&params)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, newAPIError(errorInvalidRequest,
				"unable to parse body as JSON"))
			return
		}

		token := params.Token
		if token == "" {
			writeAPIError(w, http.StatusBadRequest,
				newAPIError(errorInvalidRequest, "missing valid JWT"))
			return
		}

		clientID, err := getClientIDFromAuthorizedRequest(r)
		if err != nil {
			log.Errorf("Failed to extract clientID: %v", err)
			writeAPIError(w, http.StatusUnauthorized,
				newAPIError(errorInvalidRequest, "cilent could not be extracted from bearer token."))
			return
		}

		cm, err := clientRepo.Metadata(clientID)
		if err == client.ErrorNotFound {
			log.Errorf("No such client: %v", err)
			writeAPIError(w, http.StatusBadRequest,
				newAPIError(errorInvalidRequest, "invalid client_id"))
			return

		}
		if err != nil {
			log.Errorf("Error getting ClientMetadata: %v", err)
			writeAPIError(w, http.StatusInternalServerError,
				newAPIError(errorServerError, "could not send email at this time"))
			return
		}

		noop := func() error { return nil }
		keysFunc := func() []key.PublicKey {
			keys, err := srvKeysFunc()
			if err != nil {
				log.Errorf("Error getting keys: %v", err)
			}
			return keys
		}

		jwt, err := jose.ParseJWT(token)
		if err != nil {
			log.Errorf("Failed to Parse JWT: %v", err)
			writeAPIError(w, http.StatusBadRequest,
				newAPIError(errorInvalidRequest, "token could not be parsed"))
			return
		}

		verifier := oidc.NewJWTVerifier(issuerURL.String(), clientID, noop, keysFunc)
		if err := verifier.Verify(jwt); err != nil {
			log.Errorf("Failed to Verify JWT: %v", err)
			writeAPIError(w, http.StatusUnauthorized,
				newAPIError(errorAccessDenied, "invalid token could not be verified"))
			return
		}

		claims, err := jwt.Claims()
		if err != nil {
			log.Errorf("Failed to extract claims from JWT: %v", err)
			writeAPIError(w, http.StatusBadRequest,
				newAPIError(errorInvalidRequest, "invalid token could not be parsed"))
			return
		}

		sub, ok, err := claims.StringClaim("sub")
		if err != nil || !ok || sub == "" {
			log.Errorf("Failed to extract sub claim from JWT: err:%q ok:%v", err, ok)
			writeAPIError(w, http.StatusBadRequest,
				newAPIError(errorInvalidRequest, "could not extract sub claim from token"))
			return
		}

		usr, err := userRepo.Get(nil, sub)
		if err != nil {
			if err == user.ErrorNotFound {
				log.Errorf("Failed to find user specified by token: %v", err)
				writeAPIError(w, http.StatusBadRequest,
					newAPIError(errorInvalidRequest, "could not find user"))
				return
			}
			log.Errorf("Failed to fetch user: %v", err)
			writeAPIError(w, http.StatusInternalServerError,
				newAPIError(errorServerError, "could not send email at this time"))
			return
		}

		if usr.EmailVerified {
			log.Errorf("User's email already verified")
			writeAPIError(w, http.StatusBadRequest,
				newAPIError(errorInvalidRequest, "email already verified"))
			return
		}

		aud, _, _ := claims.StringClaim("aud")
		if aud != clientID {
			log.Errorf("aud of token and sub of bearer token must match: %v", err)
			writeAPIError(w, http.StatusForbidden,
				newAPIError(errorAccessDenied, "JWT is from another client."))
			return
		}

		nonce, _, _ := claims.StringClaim(user.ClaimTokenNonce)
		if err != nil {
			log.Errorf("Failed to extract nonce from token: %v", err)
			writeAPIError(w, http.StatusBadRequest,
				newAPIError(errorInvalidRequest, "invalid token could not be parsed"))
			return
		}

		scopes, _, _ := claims.StringsClaim(user.ClaimTokenScopes)
		if err != nil {
			log.Errorf("Failed to extract scopes from token: %v", err)
			writeAPIError(w, http.StatusBadRequest,
				newAPIError(errorInvalidRequest, "invalid token could not be parsed"))
			return
		}

		state, _, _ := claims.StringClaim(user.ClaimTokenState)
		if err != nil {
			log.Errorf("Failed to extract state from token: %v", err)
			writeAPIError(w, http.StatusBadRequest,
				newAPIError(errorInvalidRequest, "invalid token could not be parsed"))
			return
		}

		redirectURLStr := params.RedirectURI
		if redirectURLStr == "" {
			log.Errorf("No redirect URL: %v", err)
			writeAPIError(w, http.StatusBadRequest,
				newAPIError(errorInvalidRequest, "must provide a redirect_uri"))
			return
		}

		redirectURL, err := url.Parse(redirectURLStr)
		if err != nil {
			log.Errorf("Unparsable URL: %v", err)
			writeAPIError(w, http.StatusBadRequest,
				newAPIError(errorInvalidRequest, "invalid redirect_uri"))
			return
		}

		*redirectURL, err = client.ValidRedirectURL(redirectURL, cm.RedirectURIs)
		if err != nil {
			switch err {
			case (client.ErrorInvalidRedirectURL):
				log.Errorf("Request provided unregistered redirect URL: %s", redirectURLStr)
				writeAPIError(w, http.StatusBadRequest,
					newAPIError(errorInvalidRequest, "invalid redirect_uri"))
				return
			case (client.ErrorNoValidRedirectURLs):
				log.Errorf("There are no registered URLs for the requested client: %s", redirectURL)
				writeAPIError(w, http.StatusBadRequest,
					newAPIError(errorInvalidRequest, "invalid redirect_uri"))
				return
			}
		}

		_, err = emailer.SendEmailVerification(usr.ID, clientID, *redirectURL, state, nonce, scopes)
		if err != nil {
			log.Errorf("Failed to send email verification email: %v", err)
			writeAPIError(w, http.StatusInternalServerError,
				newAPIError(errorServerError, "could not send email at this time"))
			return
		}
		writeResponseWithBody(w, http.StatusOK, struct{}{})
	}
}

type emailVerifiedTemplateData struct {
	Error   string
	Message string
}

func handleEmailVerifyFunc(verifiedTpl *template.Template, s *sessionmanager.SessionManager, issuer url.URL, keysFunc func() ([]key.PublicKey,
	error), userManager *manager.UserManager) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		token := q.Get("token")

		runtpl := func(error string, message string, statusCode int) {
			execTemplateWithStatus(w, verifiedTpl, emailVerifiedTemplateData{
				Error:   error,
				Message: message,
			}, statusCode)
		}

		keys, err := keysFunc()
		if err != nil {
			runtpl("There's been an error processing your request.", "Please try again later.", http.StatusInternalServerError)
			return
		}

		ev, err := user.ParseAndVerifyEmailVerificationToken(token, issuer, keys)
		if err != nil {
			runtpl("Bad Email Verification Token", "That was not a verifiable token.", http.StatusBadRequest)
			return
		}

		aud, ok, err := ev.Claims.StringClaim("aud")
		if !ok {
			runtpl("Unable to get 'aud' from token", "That was not a verifiable token.", http.StatusBadRequest)
			return
		}

		sub, ok, err := ev.Claims.StringClaim("sub")
		if !ok {
			runtpl("Unable to get 'sub' from token", "That was not a verifiable token.", http.StatusBadRequest)
			return
		}

		state, ok, err := ev.Claims.StringClaim(user.ClaimTokenState)
		if !ok {
			runtpl("Unable to get 'sub' from token", "That was not a verifiable token.", http.StatusBadRequest)
			return
		}

		nonce, ok, err := ev.Claims.StringClaim(user.ClaimTokenNonce)
		if !ok {
			runtpl("Unable to get 'sub' from token", "That was not a verifiable token.", http.StatusBadRequest)
			return
		}

		scopes, ok, err := ev.Claims.StringsClaim(user.ClaimTokenScopes)
		if !ok {
			runtpl("Unable to get 'sub' from token", "That was not a verifiable token.", http.StatusBadRequest)
			return
		}

		callback, ok, err := ev.Claims.StringClaim(user.ClaimEmailVerificationCallback)
		if !ok {
			runtpl("Unable to get callback URL from token", "That was not a verifiable token.", http.StatusBadRequest)
			return
		}

		redirectURL, err := url.Parse(callback)
		if err != nil {
			runtpl("Unable to parse callback URL", "The callback URL inside the token couldn't be parsed.", http.StatusBadRequest)
			return
		}

		sessionID, err := s.NewSession("local", aud, state, *redirectURL, nonce, false, scopes)
		if err != nil {
			runtpl("Unable to create session", "Unable to create session.", http.StatusUnauthorized)
			return
		}

		ident, err := oidc.IdentityFromClaims(ev.Claims)
		if err != nil {
			runtpl("Unable to retrieve identity", "Could not extract identity from claims.", http.StatusBadRequest)
			return
		}

		_, err = s.AttachRemoteIdentity(sessionID, *ident)
		if err != nil {
			runtpl("Unable to attach identity to session", "Could not attach identity to session.", http.StatusInternalServerError)
			return
		}

		usr, err := userManager.Get(sub)
		if err != nil {
			runtpl("Unable to retrieve user", "Unable to retrieve user.", http.StatusUnauthorized)
			return
		}

		_, err = s.AttachUser(sessionID, usr.ID)
		if err != nil {
			runtpl("Unable to attach user to session", "Unable to attach user to session.", http.StatusInternalServerError)
			return
		}

		code, err := s.NewSessionKey(sessionID)
		if err != nil {
			runtpl("Unable to generate session key", "Unable to generate session key.", http.StatusInternalServerError)
			return
		}

		cbURL, err := userManager.VerifyEmail(ev)
		if err != nil {
			switch err {
			case manager.ErrorEmailAlreadyVerified:
				runtpl("Invalid Verification Link", "Your email link has expired or has already been verified.", http.StatusBadRequest)
			case user.ErrorNotFound:
				runtpl("Invalid Verification Link", "Your email link does not match the email address on file. Perhaps you have a more recent verification link?", http.StatusUnauthorized)
			default:
				runtpl("Error Processing Request", "Please try again later.", http.StatusInternalServerError)
			}
			return
		}

		qr := cbURL.Query()
		qr.Set("code", code)
		qr.Set("state", state)
		cbURL.RawQuery = qr.Encode()

		http.SetCookie(w, &http.Cookie{
			HttpOnly: true,
			Name:     "ShowEmailVerifiedMessage",
			MaxAge:   int(60 * 5),
			Expires:  time.Now().Add(time.Minute * 5),
		})
		http.Redirect(w, r, cbURL.String(), http.StatusSeeOther)
	}
}
