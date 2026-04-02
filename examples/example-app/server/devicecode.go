package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"

	"github.com/dexidp/dex/examples/example-app/session"
)

// handleDeviceStart initiates the Device Code Flow by requesting a device code from the IdP.
func (s *Server) handleDeviceStart(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		Scopes       []string `json:"scopes"`
		CrossClients []string `json:"cross_clients"`
		ConnectorID  string   `json:"connector_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse request body: %v", err), http.StatusBadRequest)
		return
	}

	scopes := buildScopes(reqBody.Scopes, reqBody.CrossClients)

	data := url.Values{}
	data.Set("client_id", s.clientID)
	data.Set("client_secret", s.clientSecret)
	data.Set("scope", strings.Join(scopes, " "))
	if reqBody.ConnectorID != "" {
		data.Set("connector_id", reqBody.ConnectorID)
	}

	resp, err := s.client.PostForm(s.deviceAuthURL, data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to request device code: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body := new(bytes.Buffer)
		body.ReadFrom(resp.Body)
		http.Error(w, fmt.Sprintf("Device code request failed: %s", body.String()), resp.StatusCode)
		return
	}

	var deviceResp struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&deviceResp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode device response: %v", err), http.StatusInternalServerError)
		return
	}

	pollInterval := deviceResp.Interval
	if pollInterval == 0 {
		pollInterval = 5
	}

	sessionID := s.devices.Save(session.DeviceState{
		DeviceCode:      deviceResp.DeviceCode,
		UserCode:        deviceResp.UserCode,
		VerificationURI: deviceResp.VerificationURI,
		PollInterval:    pollInterval,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":     "ok",
		"session_id": sessionID,
	})
}

// handleDeviceStatus renders the device flow pending page with verification URL and user code.
func (s *Server) handleDeviceStatus(w http.ResponseWriter, r *http.Request) {
	// JS redirects here without session_id, so always show the latest session.
	sessionID, state, ok := s.devices.GetLatest()
	if !ok {
		http.Error(w, "No device flow in progress", http.StatusBadRequest)
		return
	}

	s.renderer.RenderDevicePage(w, DevicePageData{
		SessionID:       sessionID,
		DeviceCode:      state.DeviceCode,
		UserCode:        state.UserCode,
		VerificationURI: state.VerificationURI,
		PollInterval:    state.PollInterval,
		LogoURI:         dexLogoDataURI,
	})
}

// handleDevicePoll polls the token endpoint on behalf of the device.
func (s *Server) handleDevicePoll(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceCode string `json:"device_code"`
		SessionID  string `json:"session_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	state, ok := s.devices.Get(req.SessionID)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGone)
		json.NewEncoder(w).Encode(map[string]any{
			"error":             "session_expired",
			"error_description": "This device flow session has been superseded by a new one",
		})
		return
	}

	if req.DeviceCode != state.DeviceCode {
		http.Error(w, "Invalid device code", http.StatusBadRequest)
		return
	}

	// If we already have a token, return success.
	if state.Token != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "complete"})
		return
	}

	// Poll the token endpoint.
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("device_code", req.DeviceCode)
	data.Set("client_id", s.clientID)
	data.Set("client_secret", s.clientSecret)

	tokenResp, err := s.client.PostForm(s.provider.Endpoint().TokenURL, data)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "pending"})
		return
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode == http.StatusOK {
		var tokenData struct {
			AccessToken  string `json:"access_token"`
			TokenType    string `json:"token_type"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
			IDToken      string `json:"id_token"`
		}

		if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
			http.Error(w, "Failed to decode token", http.StatusInternalServerError)
			return
		}

		token := (&oauth2.Token{
			AccessToken:  tokenData.AccessToken,
			TokenType:    tokenData.TokenType,
			RefreshToken: tokenData.RefreshToken,
		}).WithExtra(map[string]any{
			"id_token": tokenData.IDToken,
		})

		s.devices.SetToken(req.SessionID, token)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "complete"})
		return
	}

	// Check for OAuth2 error response.
	var errorResp struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	if err := json.NewDecoder(tokenResp.Body).Decode(&errorResp); err == nil {
		if errorResp.Error == "authorization_pending" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "pending"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(tokenResp.StatusCode)
		json.NewEncoder(w).Encode(map[string]any{
			"error":             errorResp.Error,
			"error_description": errorResp.ErrorDescription,
		})
		return
	}

	// Unknown response — treat as pending.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "pending"})
}

// handleDeviceComplete displays the token obtained via the Device Code Flow.
func (s *Server) handleDeviceComplete(w http.ResponseWriter, r *http.Request) {
	// JS redirects here without session_id, so always show the latest session.
	_, state, ok := s.devices.GetLatest()
	if !ok || state.Token == nil {
		http.Error(w, "No token available", http.StatusBadRequest)
		return
	}

	s.renderTokenResult(w, r, state.Token)
}
