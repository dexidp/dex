package server

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// Token type URNs from RFC 8693.
const (
	tokenTypeAccess = "urn:ietf:params:oauth:token-type:access_token"
	tokenTypeID     = "urn:ietf:params:oauth:token-type:id_token"
)

// handleRefreshGrant redeems a refresh token for a new token set. The token
// comes from the form rather than the session so that a refresh token from
// anywhere — another flow, a copy-paste — can be tried here.
func (s *Server) handleRefreshGrant(w http.ResponseWriter, r *http.Request) {
	refresh := r.FormValue("refresh_token")
	if refresh == "" {
		http.Error(w, "refresh_token is required", http.StatusBadRequest)
		return
	}

	ctx := oidc.ClientContext(r.Context(), s.client)

	// An expiry in the past is what tells the token source to redeem it now.
	stale := &oauth2.Token{RefreshToken: refresh, Expiry: time.Now().Add(-time.Hour)}
	token, err := s.oauth2Config(nil).TokenSource(ctx, stale).Token()
	if err != nil {
		http.Error(w, fmt.Sprintf("refresh failed: %v", err), http.StatusBadRequest)
		return
	}

	// A refresh renews this browser's sign-in as well, when it produced one.
	sess := s.session(w, r)
	if claims, rawIDToken, err := s.claimsFromToken(r, token); err == nil {
		s.sessions.SignIn(sess, claims, token, rawIDToken)
	}

	s.renderToken(w, r, "Refresh token", token)
}

// handleClientCredentialsGrant gets a token for the application itself. There
// is no user in this flow, so there is no ID token either — what comes back
// says only which client it belongs to.
func (s *Server) handleClientCredentialsGrant(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	cfg := clientcredentials.Config{
		ClientID:     s.clientID,
		ClientSecret: s.clientSecret,
		TokenURL:     s.provider.Endpoint().TokenURL,
		Scopes:       splitScopes(r.FormValue("scopes")),
	}

	ctx := oidc.ClientContext(r.Context(), s.client)
	token, err := cfg.Token(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("client credentials grant failed: %v", err), http.StatusBadRequest)
		return
	}

	s.renderToken(w, r, "Client credentials", token)
}

// handlePasswordGrant exchanges a username and password for tokens.
//
// The flow is deprecated — it puts the user's password through the application
// rather than keeping it at the provider, which is the thing OAuth exists to
// avoid — and dex only allows it for clients and connectors configured for it.
// It is here because dex still implements it and an example app is where you
// find out whether your configuration works.
func (s *Server) handlePasswordGrant(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	username, password := r.FormValue("username"), r.FormValue("password")
	if username == "" || password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	ctx := oidc.ClientContext(r.Context(), s.client)
	cfg := s.oauth2Config(splitScopes(r.FormValue("scopes")))
	token, err := cfg.PasswordCredentialsToken(ctx, username, password)
	if err != nil {
		http.Error(w, fmt.Sprintf("password grant failed: %v", err), http.StatusBadRequest)
		return
	}

	s.renderToken(w, r, "Password", token)
}

// handleTokenExchangeGrant trades a token from a connector for one issued by
// dex (RFC 8693).
//
// The response carries a single token and an issued_token_type rather than the
// usual set, so the app reads it directly instead of through the oauth2
// library. dex also requires connector_id, which is an extension to the RFC:
// the connector is what verifies the subject token.
func (s *Server) handleTokenExchangeGrant(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	subjectToken := r.FormValue("subject_token")
	connectorID := r.FormValue("connector_id")
	if subjectToken == "" || connectorID == "" {
		http.Error(w, "subject_token and connector_id are required", http.StatusBadRequest)
		return
	}

	subjectTokenType := r.FormValue("subject_token_type")
	if subjectTokenType == "" {
		subjectTokenType = tokenTypeAccess
	}

	form := url.Values{
		"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token":      {subjectToken},
		"subject_token_type": {subjectTokenType},
		"connector_id":       {connectorID},
	}
	if v := r.FormValue("requested_token_type"); v != "" {
		form.Set("requested_token_type", v)
	}
	if scopes := splitScopes(r.FormValue("scopes")); len(scopes) > 0 {
		form.Set("scope", strings.Join(scopes, " "))
	}

	body, err := s.postToTokenEndpoint(r, form)
	if err != nil {
		http.Error(w, fmt.Sprintf("token exchange failed: %v", err), http.StatusBadRequest)
		return
	}

	s.renderRawToken(w, r, "Token exchange", body)
}

// postToTokenEndpoint posts a form to the token endpoint with client
// authentication and returns the response body.
func (s *Server) postToTokenEndpoint(r *http.Request, form url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, s.provider.Endpoint().TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(url.QueryEscape(s.clientID), url.QueryEscape(s.clientSecret))

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// splitScopes parses a space or comma separated scope list.
func splitScopes(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ' ' || r == ',' || r == '\n' || r == '\t'
	})
	if len(fields) == 0 {
		return nil
	}
	return fields
}

// buildScopes assembles the scope list for an authorization request.
func buildScopes(extraScopes, crossClients []string) []string {
	scopes := []string{oidc.ScopeOpenID}
	seen := map[string]bool{oidc.ScopeOpenID: true}

	for _, scope := range extraScopes {
		if scope != "" && !seen[scope] {
			scopes = append(scopes, scope)
			seen[scope] = true
		}
	}
	for _, client := range crossClients {
		if client != "" {
			scopes = append(scopes, "audience:server:client_id:"+client)
		}
	}
	return scopes
}
