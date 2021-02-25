package server

import (
	"encoding/json"
	"errors"
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
			s.logger.Errorf("Server template error: %v", err)
			s.renderError(r, w, http.StatusNotFound, "Page not found")
		}
	default:
		s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
	}
}

func (s *Server) handleDeviceCode(w http.ResponseWriter, r *http.Request) {
	pollIntervalSeconds := 5

	switch r.Method {
	case http.MethodPost:
		err := r.ParseForm()
		if err != nil {
			s.logger.Errorf("Could not parse Device Request body: %v", err)
			s.tokenErrHelper(w, errInvalidRequest, "", http.StatusNotFound)
			return
		}

		// Get the client id and scopes from the post
		clientID := r.Form.Get("client_id")
		clientSecret := r.Form.Get("client_secret")
		scopes := strings.Fields(r.Form.Get("scope"))

		s.logger.Infof("Received device request for client %v with scopes %v", clientID, scopes)

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

		if err := s.storage.CreateDeviceRequest(deviceReq); err != nil {
			s.logger.Errorf("Failed to store device request; %v", err)
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
		}

		if err := s.storage.CreateDeviceToken(deviceToken); err != nil {
			s.logger.Errorf("Failed to store device token %v", err)
			s.tokenErrHelper(w, errInvalidRequest, "", http.StatusInternalServerError)
			return
		}

		u, err := url.Parse(s.issuerURL.String())
		if err != nil {
			s.logger.Errorf("Could not parse issuer URL %v", err)
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

		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "   ")
		enc.Encode(code)

	default:
		s.renderError(r, w, http.StatusBadRequest, "Invalid device code request type")
		s.tokenErrHelper(w, errInvalidRequest, "", http.StatusBadRequest)
	}
}

func (s *Server) handleDeviceTokenDeprecated(w http.ResponseWriter, r *http.Request) {
	s.logger.Warn(`The deprecated "/device/token" endpoint was called. It will be removed, use "/token" instead.`)

	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodPost:
		err := r.ParseForm()
		if err != nil {
			s.logger.Warnf("Could not parse Device Token Request body: %v", err)
			s.tokenErrHelper(w, errInvalidRequest, "", http.StatusBadRequest)
			return
		}

		grantType := r.PostFormValue("grant_type")
		if grantType != grantTypeDeviceCode {
			s.tokenErrHelper(w, errInvalidGrant, "", http.StatusBadRequest)
			return
		}

		s.handleDeviceToken(w, r)
	default:
		s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
	}
}

func (s *Server) handleDeviceToken(w http.ResponseWriter, r *http.Request) {
	deviceCode := r.Form.Get("device_code")
	if deviceCode == "" {
		s.tokenErrHelper(w, errInvalidRequest, "No device code received", http.StatusBadRequest)
		return
	}

	now := s.now()

	// Grab the device token, check validity
	deviceToken, err := s.storage.GetDeviceToken(deviceCode)
	if err != nil {
		if err != storage.ErrNotFound {
			s.logger.Errorf("failed to get device code: %v", err)
		}
		s.tokenErrHelper(w, errInvalidRequest, "Invalid Device code.", http.StatusBadRequest)
		return
	} else if now.After(deviceToken.Expiry) {
		s.tokenErrHelper(w, deviceTokenExpired, "", http.StatusBadRequest)
		return
	}

	// Rate Limiting check
	slowDown := false
	pollInterval := deviceToken.PollIntervalSeconds
	minRequestTime := deviceToken.LastRequestTime.Add(time.Second * time.Duration(pollInterval))
	if now.Before(minRequestTime) {
		slowDown = true
		// Continually increase the poll interval until the user waits the proper time
		pollInterval += 5
	} else {
		pollInterval = 5
	}

	switch deviceToken.Status {
	case deviceTokenPending:
		updater := func(old storage.DeviceToken) (storage.DeviceToken, error) {
			old.PollIntervalSeconds = pollInterval
			old.LastRequestTime = now
			return old, nil
		}
		// Update device token last request time in storage
		if err := s.storage.UpdateDeviceToken(deviceCode, updater); err != nil {
			s.logger.Errorf("failed to update device token: %v", err)
			s.renderError(r, w, http.StatusInternalServerError, "")
			return
		}
		if slowDown {
			s.tokenErrHelper(w, deviceTokenSlowDown, "", http.StatusBadRequest)
		} else {
			s.tokenErrHelper(w, deviceTokenPending, "", http.StatusUnauthorized)
		}
	case deviceTokenComplete:
		w.Write([]byte(deviceToken.Token))
	}
}

func (s *Server) handleDeviceCallback(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		userCode := r.FormValue("state")
		code := r.FormValue("code")

		if userCode == "" || code == "" {
			s.renderError(r, w, http.StatusBadRequest, "Request was missing parameters")
			return
		}

		// Authorization redirect callback from OAuth2 auth flow.
		if errMsg := r.FormValue("error"); errMsg != "" {
			http.Error(w, errMsg+": "+r.FormValue("error_description"), http.StatusBadRequest)
			return
		}

		authCode, err := s.storage.GetAuthCode(code)
		if err != nil || s.now().After(authCode.Expiry) {
			errCode := http.StatusBadRequest
			if err != nil && err != storage.ErrNotFound {
				s.logger.Errorf("failed to get auth code: %v", err)
				errCode = http.StatusInternalServerError
			}
			s.renderError(r, w, errCode, "Invalid or expired auth code.")
			return
		}

		// Grab the device request from storage
		deviceReq, err := s.storage.GetDeviceRequest(userCode)
		if err != nil || s.now().After(deviceReq.Expiry) {
			errCode := http.StatusBadRequest
			if err != nil && err != storage.ErrNotFound {
				s.logger.Errorf("failed to get device code: %v", err)
				errCode = http.StatusInternalServerError
			}
			s.renderError(r, w, errCode, "Invalid or expired user code.")
			return
		}

		client, err := s.storage.GetClient(deviceReq.ClientID)
		if err != nil {
			if err != storage.ErrNotFound {
				s.logger.Errorf("failed to get client: %v", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
			} else {
				s.tokenErrHelper(w, errInvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
			}
			return
		}
		if client.Secret != deviceReq.ClientSecret {
			s.tokenErrHelper(w, errInvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
			return
		}

		resp, err := s.exchangeAuthCode(w, authCode, client)
		if err != nil {
			s.logger.Errorf("Could not exchange auth code for client %q: %v", deviceReq.ClientID, err)
			s.renderError(r, w, http.StatusInternalServerError, "Failed to exchange auth code.")
			return
		}

		// Grab the device token from storage
		old, err := s.storage.GetDeviceToken(deviceReq.DeviceCode)
		if err != nil || s.now().After(old.Expiry) {
			errCode := http.StatusBadRequest
			if err != nil && err != storage.ErrNotFound {
				s.logger.Errorf("failed to get device token: %v", err)
				errCode = http.StatusInternalServerError
			}
			s.renderError(r, w, errCode, "Invalid or expired device code.")
			return
		}

		updater := func(old storage.DeviceToken) (storage.DeviceToken, error) {
			if old.Status == deviceTokenComplete {
				return old, errors.New("device token already complete")
			}
			respStr, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				s.logger.Errorf("failed to marshal device token response: %v", err)
				s.renderError(r, w, http.StatusInternalServerError, "")
				return old, err
			}

			old.Token = string(respStr)
			old.Status = deviceTokenComplete
			return old, nil
		}

		// Update refresh token in the storage, store the token and mark as complete
		if err := s.storage.UpdateDeviceToken(deviceReq.DeviceCode, updater); err != nil {
			s.logger.Errorf("failed to update device token: %v", err)
			s.renderError(r, w, http.StatusBadRequest, "")
			return
		}

		if err := s.templates.deviceSuccess(r, w, client.Name); err != nil {
			s.logger.Errorf("Server template error: %v", err)
			s.renderError(r, w, http.StatusNotFound, "Page not found")
		}

	default:
		http.Error(w, fmt.Sprintf("method not implemented: %s", r.Method), http.StatusBadRequest)
		return
	}
}

func (s *Server) verifyUserCode(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		err := r.ParseForm()
		if err != nil {
			s.logger.Warnf("Could not parse user code verification request body : %v", err)
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
		deviceRequest, err := s.storage.GetDeviceRequest(userCode)
		if err != nil || s.now().After(deviceRequest.Expiry) {
			if err != nil && err != storage.ErrNotFound {
				s.logger.Errorf("failed to get device request: %v", err)
			}
			if err := s.templates.device(r, w, s.getDeviceVerificationURI(), userCode, true); err != nil {
				s.logger.Errorf("Server template error: %v", err)
				s.renderError(r, w, http.StatusNotFound, "Page not found")
			}
			return
		}

		// Redirect to Dex Auth Endpoint
		authURL := path.Join(s.issuerURL.Path, "/auth")
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
		q.Set("redirect_uri", "/device/callback")
		q.Set("scope", strings.Join(deviceRequest.Scopes, " "))
		u.RawQuery = q.Encode()

		http.Redirect(w, r, u.String(), http.StatusFound)

	default:
		s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
	}
}
