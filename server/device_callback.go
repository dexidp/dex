package server

// device_callback.go implements the /device/callback browser callback that completes
// the device flow: it exchanges the auth code and stores the token that the device_code
// grant (grant_devicecode.go) then serves.

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/storage"
)

func (s *Server) handleDeviceCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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
			// Log the error details but don't expose them to the user
			s.logger.ErrorContext(r.Context(), "OAuth2 authorization error",
				"error", errMsg,
				"error_description", r.FormValue("error_description"))
			s.renderError(r, w, http.StatusBadRequest, "Authorization failed. Please try again.")
			return
		}

		authCode, err := s.storage.GetAuthCode(ctx, code)
		if err != nil || s.now().After(authCode.Expiry) {
			errCode := http.StatusBadRequest
			if err != nil && err != storage.ErrNotFound {
				s.logger.ErrorContext(r.Context(), "failed to get auth code", "err", err)
				errCode = http.StatusInternalServerError
			}
			s.renderError(r, w, errCode, "Invalid or expired auth code.")
			return
		}

		// Grab the device request from storage
		deviceReq, err := s.storage.GetDeviceRequest(ctx, userCode)
		if err != nil || s.now().After(deviceReq.Expiry) {
			errCode := http.StatusBadRequest
			if err != nil && err != storage.ErrNotFound {
				s.logger.ErrorContext(r.Context(), "failed to get device code", "err", err)
				errCode = http.StatusInternalServerError
			}
			s.renderError(r, w, errCode, "Invalid or expired user code.")
			return
		}

		client, err := s.storage.GetClient(ctx, deviceReq.ClientID)
		if err != nil {
			if err != storage.ErrNotFound {
				s.logger.ErrorContext(r.Context(), "failed to get client", "err", err)
				s.tokenErrHelper(w, oauth2.ServerError, "", http.StatusInternalServerError)
			} else {
				s.tokenErrHelper(w, oauth2.InvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
			}
			return
		}
		if client.Secret != deviceReq.ClientSecret {
			s.tokenErrHelper(w, oauth2.InvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
			return
		}

		resp, err := s.exchangeAuthCode(ctx, w, authCode, client)
		if err != nil {
			s.logger.ErrorContext(r.Context(), "could not exchange auth code for clien", "client_id", deviceReq.ClientID, "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "Failed to exchange auth code.")
			return
		}

		// Grab the device token from storage
		old, err := s.storage.GetDeviceToken(ctx, deviceReq.DeviceCode)
		if err != nil || s.now().After(old.Expiry) {
			errCode := http.StatusBadRequest
			if err != nil && err != storage.ErrNotFound {
				s.logger.ErrorContext(r.Context(), "failed to get device token", "err", err)
				errCode = http.StatusInternalServerError
			}
			s.renderError(r, w, errCode, "Invalid or expired device code.")
			return
		}

		updater := func(old storage.DeviceToken) (storage.DeviceToken, error) {
			if old.Status == oauth2.DeviceTokenComplete {
				return old, errors.New("device token already complete")
			}
			respStr, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				s.logger.ErrorContext(r.Context(), "failed to marshal device token response", "err", err)
				s.renderError(r, w, http.StatusInternalServerError, "")
				return old, err
			}

			old.Token = string(respStr)
			old.Status = oauth2.DeviceTokenComplete
			return old, nil
		}

		// Update refresh token in the storage, store the token and mark as complete
		if err := s.storage.UpdateDeviceToken(ctx, deviceReq.DeviceCode, updater); err != nil {
			s.logger.ErrorContext(r.Context(), "failed to update device token", "err", err)
			s.renderError(r, w, http.StatusBadRequest, "")
			return
		}

		if err := s.templates.DeviceSuccess(r, w, client.Name); err != nil {
			s.logger.ErrorContext(r.Context(), "Server template error", "err", err)
			s.renderError(r, w, http.StatusNotFound, "Page not found")
		}

	default:
		s.logger.ErrorContext(r.Context(), "unsupported method in device callback", "method", r.Method)
		s.renderError(r, w, http.StatusBadRequest, ErrMsgMethodNotAllowed)
		return
	}
}
