package server

// device_authorize.go implements the device-authorization request side of the device
// flow: the /device user-code entry page, the /device/code authorization request, and
// user-code verification. The browser callback is in device_callback.go and the
// device_code grant in grant_devicecode.go.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/dexidp/dex/storage"
)

type deviceCodeResponse struct {
	// The unique device code for device authentication
	DeviceCode string `json:"device_code"`
	// The code the user will exchange via a browser and log in
	UserCode string `json:"user_code"`
	// The url to verify the user code.
	VerificationURI string `json:"verification_uri"`
	// The verification uri with the user code appended for pre-filling form
	VerificationURIComplete string `json:"verification_uri_complete"`
	// The lifetime of the device code
	ExpireTime int `json:"expires_in"`
	// How often the device is allowed to poll to verify that the user login occurred
	PollInterval int `json:"interval"`
}

func (s *Server) getDeviceVerificationURI() string {
	return path.Join(s.issuerURL.Path, "/device/auth/verify_code")
}

func (s *Server) handleDeviceExchange(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Grab the parameter(s) from the query.
		// If "user_code" is set, pre-populate the user code text field.
		// If "invalid" is set, set the invalidAttempt boolean, which will display a message to the user that they
		// attempted to redeem an invalid or expired user code.
		userCode := r.URL.Query().Get("user_code")
		invalidAttempt, err := strconv.ParseBool(r.URL.Query().Get("invalid"))
		if err != nil {
			invalidAttempt = false
		}
		if err := s.templates.device(r, w, s.getDeviceVerificationURI(), userCode, invalidAttempt); err != nil {
			s.logger.ErrorContext(r.Context(), "server template error", "err", err)
			s.renderError(r, w, http.StatusNotFound, "Page not found")
		}
	default:
		s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
	}
}

func (s *Server) handleDeviceCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pollIntervalSeconds := 5

	switch r.Method {
	case http.MethodPost:
		err := r.ParseForm()
		if err != nil {
			s.logger.ErrorContext(r.Context(), "could not parse Device Request body", "err", err)
			s.tokenErrHelper(w, errInvalidRequest, "", http.StatusNotFound)
			return
		}

		// Get the client id and scopes from the post
		clientID := r.Form.Get("client_id")
		clientSecret := r.Form.Get("client_secret")
		scopes := strings.Fields(r.Form.Get("scope"))
		codeChallenge := r.Form.Get("code_challenge")
		codeChallengeMethod := r.Form.Get("code_challenge_method")

		if codeChallengeMethod == "" {
			codeChallengeMethod = codeChallengeMethodPlain
		}
		if codeChallengeMethod != codeChallengeMethodS256 && codeChallengeMethod != codeChallengeMethodPlain {
			description := fmt.Sprintf("Unsupported PKCE challenge method (%q).", codeChallengeMethod)
			s.tokenErrHelper(w, errInvalidRequest, description, http.StatusBadRequest)
			return
		}

		if len(scopes) == 0 {
			// per RFC8628 section 3.1, https://datatracker.ietf.org/doc/html/rfc8628#section-3.1
			// scope is optional but dex requires that it is always at least 'openid' so default it
			scopes = []string{"openid"}
		}

		s.logger.InfoContext(r.Context(), "received device request", "client_id", clientID, "scoped", scopes)

		// Make device code
		deviceCode := storage.NewDeviceCode()

		// make user code
		userCode := storage.NewUserCode()

		// Generate the expire time
		expireTime := time.Now().Add(s.deviceRequestsValidFor)

		// Store the Device Request
		deviceReq := storage.DeviceRequest{
			UserCode:     userCode,
			DeviceCode:   deviceCode,
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes:       scopes,
			Expiry:       expireTime,
		}

		if err := s.storage.CreateDeviceRequest(ctx, deviceReq); err != nil {
			s.logger.ErrorContext(r.Context(), "failed to store device request", "err", err)
			s.tokenErrHelper(w, errInvalidRequest, "", http.StatusInternalServerError)
			return
		}

		// Store the device token
		deviceToken := storage.DeviceToken{
			DeviceCode:          deviceCode,
			Status:              deviceTokenPending,
			Expiry:              expireTime,
			LastRequestTime:     s.now(),
			PollIntervalSeconds: 0,
			PKCE: storage.PKCE{
				CodeChallenge:       codeChallenge,
				CodeChallengeMethod: codeChallengeMethod,
			},
		}

		if err := s.storage.CreateDeviceToken(ctx, deviceToken); err != nil {
			s.logger.ErrorContext(r.Context(), "failed to store device token", "err", err)
			s.tokenErrHelper(w, errInvalidRequest, "", http.StatusInternalServerError)
			return
		}

		u, err := url.Parse(s.issuerURL.String())
		if err != nil {
			s.logger.ErrorContext(r.Context(), "could not parse issuer URL", "err", err)
			s.tokenErrHelper(w, errInvalidRequest, "", http.StatusInternalServerError)
			return
		}
		u.Path = path.Join(u.Path, "device")
		vURI := u.String()

		q := u.Query()
		q.Set("user_code", userCode)
		u.RawQuery = q.Encode()
		vURIComplete := u.String()

		code := deviceCodeResponse{
			DeviceCode:              deviceCode,
			UserCode:                userCode,
			VerificationURI:         vURI,
			VerificationURIComplete: vURIComplete,
			ExpireTime:              int(s.deviceRequestsValidFor.Seconds()),
			PollInterval:            pollIntervalSeconds,
		}

		// Device Authorization Response can contain cache control header according to
		// https://tools.ietf.org/html/rfc8628#section-3.2
		w.Header().Set("Cache-Control", "no-store")

		// Response type should be application/json according to
		// https://datatracker.ietf.org/doc/html/rfc6749#section-5.1
		w.Header().Set("Content-Type", "application/json")

		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "   ")
		enc.Encode(code)

	default:
		s.renderError(r, w, http.StatusBadRequest, "Invalid device code request type")
		s.tokenErrHelper(w, errInvalidRequest, "", http.StatusBadRequest)
	}
}

func (s *Server) verifyUserCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	switch r.Method {
	case http.MethodPost:
		err := r.ParseForm()
		if err != nil {
			s.logger.Warn("could not parse user code verification request body", "err", err)
			s.renderError(r, w, http.StatusBadRequest, "")
			return
		}

		userCode := r.Form.Get("user_code")
		if userCode == "" {
			s.renderError(r, w, http.StatusBadRequest, "No user code received")
			return
		}

		userCode = strings.ToUpper(userCode)

		// Find the user code in the available requests
		deviceRequest, err := s.storage.GetDeviceRequest(ctx, userCode)
		if err != nil || s.now().After(deviceRequest.Expiry) {
			if err != nil && err != storage.ErrNotFound {
				s.logger.ErrorContext(r.Context(), "failed to get device request", "err", err)
			}
			if err := s.templates.device(r, w, s.getDeviceVerificationURI(), userCode, true); err != nil {
				s.logger.ErrorContext(r.Context(), "Server template error", "err", err)
				s.renderError(r, w, http.StatusNotFound, "Page not found")
			}
			return
		}

		// Redirect to Dex Auth Endpoint
		authURL := s.absURL("/auth")
		u, err := url.Parse(authURL)
		if err != nil {
			s.renderError(r, w, http.StatusInternalServerError, "Invalid auth URI.")
			return
		}
		q := u.Query()
		q.Set("client_id", deviceRequest.ClientID)
		q.Set("client_secret", deviceRequest.ClientSecret)
		q.Set("state", deviceRequest.UserCode)
		q.Set("response_type", "code")
		q.Set("redirect_uri", s.absPath(deviceCallbackURI))
		q.Set("scope", strings.Join(deviceRequest.Scopes, " "))
		u.RawQuery = q.Encode()

		http.Redirect(w, r, u.String(), http.StatusFound)

	default:
		s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
	}
}
