package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

func (a *app) handleDeviceLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body to get options
	var reqBody struct {
		Scopes       []string `json:"scopes"`
		CrossClients []string `json:"cross_clients"`
		ConnectorID  string   `json:"connector_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse request body: %v", err), http.StatusBadRequest)
		return
	}

	// Build complete scope list with audience scopes (same as handleLogin)
	scopes := buildScopes(reqBody.Scopes, reqBody.CrossClients)

	// Build scope string
	scopeStr := strings.Join(scopes, " ")

	// Get device authorization endpoint
	// Properly construct the device code endpoint URL
	authURL := a.provider.Endpoint().AuthURL
	deviceAuthURL := strings.TrimSuffix(authURL, "/auth") + "/device/code"

	// Request device code
	data := url.Values{}
	data.Set("client_id", a.clientID)
	data.Set("client_secret", a.clientSecret)
	data.Set("scope", scopeStr)

	// Add connector_id if specified
	if reqBody.ConnectorID != "" {
		data.Set("connector_id", reqBody.ConnectorID)
	}

	resp, err := a.client.PostForm(deviceAuthURL, data)
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
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresIn               int    `json:"expires_in"`
		Interval                int    `json:"interval"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&deviceResp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode device response: %v", err), http.StatusInternalServerError)
		return
	}

	// Store device flow data with new session
	sessionID := generateSessionID()

	a.deviceFlowMutex.Lock()
	a.deviceFlowData.sessionID = sessionID
	a.deviceFlowData.deviceCode = deviceResp.DeviceCode
	a.deviceFlowData.userCode = deviceResp.UserCode
	a.deviceFlowData.verificationURI = deviceResp.VerificationURI
	a.deviceFlowData.pollInterval = deviceResp.Interval
	if a.deviceFlowData.pollInterval == 0 {
		a.deviceFlowData.pollInterval = 5
	}
	a.deviceFlowData.token = nil
	a.deviceFlowMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"session_id": sessionID,
	})
}

func (a *app) handleDevicePage(w http.ResponseWriter, r *http.Request) {
	a.deviceFlowMutex.Lock()
	data := devicePageData{
		SessionID:       a.deviceFlowData.sessionID,
		DeviceCode:      a.deviceFlowData.deviceCode,
		UserCode:        a.deviceFlowData.userCode,
		VerificationURI: a.deviceFlowData.verificationURI,
		PollInterval:    a.deviceFlowData.pollInterval,
		LogoURI:         dexLogoDataURI,
	}
	a.deviceFlowMutex.Unlock()

	if data.DeviceCode == "" {
		http.Error(w, "No device flow in progress", http.StatusBadRequest)
		return
	}

	renderDevice(w, data)
}

func (a *app) handleDevicePoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DeviceCode string `json:"device_code"`
		SessionID  string `json:"session_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	a.deviceFlowMutex.Lock()
	storedSessionID := a.deviceFlowData.sessionID
	storedDeviceCode := a.deviceFlowData.deviceCode
	existingToken := a.deviceFlowData.token
	a.deviceFlowMutex.Unlock()

	// Check if this session has been superseded by a new one
	if req.SessionID != storedSessionID {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGone)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":             "session_expired",
			"error_description": "This device flow session has been superseded by a new one",
		})
		return
	}

	if req.DeviceCode != storedDeviceCode {
		http.Error(w, "Invalid device code", http.StatusBadRequest)
		return
	}

	// If we already have a token, return success
	if existingToken != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "complete",
		})
		return
	}

	// Poll the token endpoint
	tokenURL := a.provider.Endpoint().TokenURL

	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("device_code", req.DeviceCode)
	data.Set("client_id", a.clientID)
	data.Set("client_secret", a.clientSecret)

	tokenResp, err := a.client.PostForm(tokenURL, data)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "pending",
		})
		return
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode == http.StatusOK {
		// Success! We got the token
		// Parse the full response including id_token
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

		// Create oauth2.Token with all fields
		token := &oauth2.Token{
			AccessToken:  tokenData.AccessToken,
			TokenType:    tokenData.TokenType,
			RefreshToken: tokenData.RefreshToken,
		}

		// Add id_token to Extra
		token = token.WithExtra(map[string]interface{}{
			"id_token": tokenData.IDToken,
		})

		// Store the token
		a.deviceFlowMutex.Lock()
		a.deviceFlowData.token = token
		a.deviceFlowMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "complete",
		})
		return
	}

	// Check for errors
	var errorResp struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	if err := json.NewDecoder(tokenResp.Body).Decode(&errorResp); err == nil {
		if errorResp.Error == "authorization_pending" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status": "pending",
			})
			return
		}

		// Other errors
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(tokenResp.StatusCode)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":             errorResp.Error,
			"error_description": errorResp.ErrorDescription,
		})
		return
	}

	// Unknown response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "pending",
	})
}

func (a *app) handleDeviceResult(w http.ResponseWriter, r *http.Request) {
	a.deviceFlowMutex.Lock()
	token := a.deviceFlowData.token
	a.deviceFlowMutex.Unlock()

	if token == nil {
		http.Error(w, "No token available", http.StatusBadRequest)
		return
	}

	parseAndRenderToken(w, r, a, token)
}
