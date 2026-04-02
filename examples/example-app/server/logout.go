package server

import (
	"net/http"
	"net/url"
)

// handleAppLogout clears the local session and redirects to the provider's
// end_session_endpoint for RP-Initiated Logout (if available).
func (s *Server) handleAppLogout(w http.ResponseWriter, r *http.Request) {
	idToken := s.auth.Clear()

	if s.endSessionEndpoint == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	logoutURL, err := url.Parse(s.endSessionEndpoint)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	q := logoutURL.Query()
	if idToken != "" {
		q.Set("id_token_hint", idToken)
	}
	// Derive app base URL from redirect URI for post-logout redirect.
	if appURL, err := url.Parse(s.redirectURI); err == nil {
		appURL.Path = "/"
		appURL.RawQuery = ""
		q.Set("post_logout_redirect_uri", appURL.String())
	}
	logoutURL.RawQuery = q.Encode()
	http.Redirect(w, r, logoutURL.String(), http.StatusFound)
}
