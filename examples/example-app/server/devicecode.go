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

	s.sessions.StartDevice(s.session(w, r), &session.Device{
		DeviceCode:      deviceResp.DeviceCode,
		UserCode:        deviceResp.UserCode,
		VerificationURI: deviceResp.VerificationURI,
		PollInterval:    pollInterval,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
}

// handleDeviceStatus renders the device flow pending page with verification URL and user code.
func (s *Server) handleDeviceStatus(w http.ResponseWriter, r *http.Request) {
	device := s.session(w, r).Device
	if device == nil {
		http.Error(w, "No device flow in progress", http.StatusBadRequest)
		return
	}

	s.renderer.RenderDevicePage(w, DevicePageData{
		AdminEnabled:    s.admin != nil,
		DeviceCode:      device.DeviceCode,
		UserCode:        device.UserCode,
		VerificationURI: device.VerificationURI,
		PollInterval:    device.PollInterval,
		LogoURI:         dexLogoDataURI,
	})
}

// handleDevicePoll polls the token endpoint on behalf of the device.
func (s *Server) handleDevicePoll(w http.ResponseWriter, r *http.Request) {
	sess := s.session(w, r)
	device := sess.Device
	if device == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGone)
		json.NewEncoder(w).Encode(map[string]any{
			"error":             "no_device_flow",
			"error_description": "This browser has no device flow in progress",
		})
		return
	}

	// If we already have a token, return success.
	if device.Token != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "complete"})
		return
	}

	// Poll the token endpoint.
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("device_code", device.DeviceCode)
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

		s.sessions.SetDeviceToken(sess, token)

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

// handleDeviceComplete displays the token obtained via the Device Code Flow and
// signs this browser in with it, the same as any other completed flow.
func (s *Server) handleDeviceComplete(w http.ResponseWriter, r *http.Request) {
	sess := s.session(w, r)
	if sess.Device == nil || sess.Device.Token == nil {
		http.Error(w, "No token available", http.StatusBadRequest)
		return
	}

	token := sess.Device.Token
	rawIDToken, _ := token.Extra("id_token").(string)
	if claims, raw, err := s.claimsFromToken(r, token); err == nil {
		s.sessions.SignIn(sess, claims, token, raw)
		rawIDToken = raw
	}

	s.sessions.RememberTokens(sess, "Device code", token, rawIDToken)
	http.Redirect(w, r, "/tokens", http.StatusSeeOther)
}
