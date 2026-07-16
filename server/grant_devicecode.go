package server

// grant_devicecode.go implements the device_code grant: polling for the token
// after the user has authorized the device (dispatched from token.go and the
// deprecated /device/token endpoint). The device authorization request and
// browser callback live in deviceflowhandlers.go.

import (
	"net/http"
	"time"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/storage"
)

func (s *Server) handleDeviceTokenDeprecated(w http.ResponseWriter, r *http.Request) {
	s.logger.Warn(`the /device/token endpoint was called. It will be removed, use /token instead.`, "deprecated", true)

	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodPost:
		err := r.ParseForm()
		if err != nil {
			s.logger.Warn("could not parse Device Token Request body", "err", err)
			s.tokenErrHelper(w, oauth2.InvalidRequest, "", http.StatusBadRequest)
			return
		}

		grantType := r.PostFormValue("grant_type")
		if grantType != oauth2.GrantTypeDeviceCode {
			s.tokenErrHelper(w, oauth2.InvalidGrant, "", http.StatusBadRequest)
			return
		}

		s.handleDeviceToken(w, r)
	default:
		s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
	}
}

func (s *Server) handleDeviceToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	deviceCode := r.Form.Get("device_code")
	if deviceCode == "" {
		s.tokenErrHelper(w, oauth2.InvalidRequest, "No device code received", http.StatusBadRequest)
		return
	}

	now := s.now()

	// Grab the device token, check validity
	deviceToken, err := s.storage.GetDeviceToken(ctx, deviceCode)
	if err != nil {
		if err != storage.ErrNotFound {
			s.logger.ErrorContext(r.Context(), "failed to get device code", "err", err)
		}
		s.tokenErrHelper(w, oauth2.InvalidRequest, "Invalid Device code.", http.StatusBadRequest)
		return
	} else if now.After(deviceToken.Expiry) {
		s.tokenErrHelper(w, oauth2.DeviceTokenExpired, "", http.StatusBadRequest)
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
	case oauth2.DeviceTokenPending:
		updater := func(old storage.DeviceToken) (storage.DeviceToken, error) {
			old.PollIntervalSeconds = pollInterval
			old.LastRequestTime = now
			return old, nil
		}
		// Update device token last request time in storage
		if err := s.storage.UpdateDeviceToken(ctx, deviceCode, updater); err != nil {
			s.logger.ErrorContext(r.Context(), "failed to update device token", "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "")
			return
		}
		if slowDown {
			s.tokenErrHelper(w, oauth2.DeviceTokenSlowDown, "", http.StatusBadRequest)
		} else {
			s.tokenErrHelper(w, oauth2.DeviceTokenPending, "", http.StatusBadRequest)
		}
	case oauth2.DeviceTokenComplete:
		codeChallengeFromStorage := deviceToken.PKCE.CodeChallenge
		providedCodeVerifier := r.Form.Get("code_verifier")

		switch {
		case providedCodeVerifier != "" && codeChallengeFromStorage != "":
			calculatedCodeChallenge, err := s.calculateCodeChallenge(providedCodeVerifier, deviceToken.PKCE.CodeChallengeMethod)
			if err != nil {
				s.logger.ErrorContext(r.Context(), "failed to calculate code challenge", "err", err)
				s.tokenErrHelper(w, oauth2.ServerError, "", http.StatusInternalServerError)
				return
			}
			if codeChallengeFromStorage != calculatedCodeChallenge {
				s.tokenErrHelper(w, oauth2.InvalidGrant, "Invalid code_verifier.", http.StatusBadRequest)
				return
			}
		case providedCodeVerifier != "":
			// Received no code_challenge on /auth, but a code_verifier on /token
			s.tokenErrHelper(w, oauth2.InvalidRequest, "No PKCE flow started. Cannot check code_verifier.", http.StatusBadRequest)
			return
		case codeChallengeFromStorage != "":
			// Received PKCE request on /auth, but no code_verifier on /token
			s.tokenErrHelper(w, oauth2.InvalidGrant, "Expecting parameter code_verifier in PKCE flow.", http.StatusBadRequest)
			return
		}
		w.Write([]byte(deviceToken.Token))
	}
}
