package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (a *app) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form to get access token
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	accessToken := r.FormValue("access_token")
	if accessToken == "" {
		http.Error(w, "access_token is required", http.StatusBadRequest)
		return
	}

	// Get UserInfo endpoint from provider
	userInfoEndpoint := a.provider.Endpoint().AuthURL
	if len(userInfoEndpoint) > 5 {
		// Replace /auth with /userinfo
		userInfoEndpoint = userInfoEndpoint[:len(userInfoEndpoint)-5] + "/userinfo"
	}

	// Create request to UserInfo endpoint
	req, err := http.NewRequestWithContext(r.Context(), "GET", userInfoEndpoint, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create request: %v", err), http.StatusInternalServerError)
		return
	}

	// Add Authorization header with access token
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Make the request
	resp, err := a.client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch userinfo: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		http.Error(w, fmt.Sprintf("UserInfo request failed: %s", string(body)), resp.StatusCode)
		return
	}

	// Parse and return the userinfo
	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode userinfo: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userInfo)
}
