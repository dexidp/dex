package server

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/dexidp/dex/examples/example-app/session"
)

// handleIndex renders the app's own view of who is signed in.
//
// Before rendering it makes sure that view is still true. An access token that
// has expired is refreshed; if the refresh fails the sign-in is over. And every
// so often the app asks the provider directly, with prompt=none, whether the
// session there still exists — otherwise signing out of dex in another tab
// would leave this app showing the user indefinitely.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	sess := s.session(w, r)

	if sess.SignedIn() {
		s.refreshIfExpired(r, sess)
	}

	if s.shouldCheckProvider(sess) {
		s.startAuth(w, r, sess, nil, "", true)
		return
	}

	data := IndexPageData{
		ScopesSupported: s.scopesSupported,
		LogoURI:         dexLogoDataURI,
		AdminEnabled:    s.admin != nil,
		PKCE:            s.pkce,
		SessionCheck:    s.sessionCheckInterval,

		DeviceSupported:            s.deviceAuthURL != "" && s.supportsGrant(grantDeviceCode),
		RefreshSupported:           s.supportsGrant(grantRefreshToken),
		ClientCredentialsSupported: s.supportsGrant(grantClientCredentials),
		PasswordSupported:          s.supportsGrant(grantPassword),
		TokenExchangeSupported:     s.supportsGrant(grantTokenExchange),
	}
	if sess.SignedIn() {
		data.User = sess.Claims
		data.Token = tokenSummary(sess.Token, sess.IDToken)
	}

	s.renderer.RenderIndexPage(w, data)
}

// shouldCheckProvider decides whether to spend a redirect confirming the
// session with the provider: for a signed-in browser once the app's answer has
// gone stale, and for a browser that has not asked at all, which is how the app
// picks up a session someone already has with dex.
func (s *Server) shouldCheckProvider(sess *session.Session) bool {
	if s.sessionCheckInterval <= 0 {
		return false
	}
	if sess.LastProviderCheck.IsZero() {
		return true
	}
	// A signed-in browser is confirmed on every load. The interval only holds
	// back the check for a browser that is not signed in, where it is looking
	// for a session rather than confirming one: a session dex has just been
	// told to end should not survive until a timer says to ask.
	if sess.SignedIn() {
		return true
	}
	return time.Since(sess.LastProviderCheck) > s.sessionCheckInterval
}

// refreshIfExpired renews an expired access token, and ends the app's session
// if the provider will not renew it — a refresh token dex has revoked is as
// good an answer as a failed session check.
func (s *Server) refreshIfExpired(r *http.Request, sess *session.Session) {
	if sess.Token == nil || sess.Token.Valid() {
		return
	}
	if sess.Token.RefreshToken == "" {
		s.sessions.SignOut(sess)
		return
	}

	ctx := oidc.ClientContext(r.Context(), s.client)
	token, err := s.oauth2Config(nil).TokenSource(ctx, sess.Token).Token()
	if err != nil {
		log.Printf("refreshing expired token: %v", err)
		s.sessions.SignOut(sess)
		return
	}

	claims, rawIDToken, err := s.claimsFromToken(r, token)
	if err != nil {
		log.Printf("verifying refreshed token: %v", err)
		s.sessions.SignOut(sess)
		return
	}
	s.sessions.SignIn(sess, claims, token, rawIDToken)
}

// handleLogin starts the authorization code flow.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	sess := s.session(w, r)
	scopes := buildScopes(r.Form["extra_scopes"], r.Form["cross_client"])
	s.startAuth(w, r, sess, scopes, r.FormValue("connector_id"), false)
}

// startAuth sends the browser to the provider. Each call gets its own state,
// nonce and PKCE verifier, kept on this browser's session: they are what ties
// the callback back to the request that started it, and reusing them across
// requests would leave nothing to check.
func (s *Server) startAuth(w http.ResponseWriter, r *http.Request, sess *session.Session, scopes []string, connectorID string, silent bool) {
	if silent {
		// A silent check needs only enough scope to identify the user.
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}

	pending := s.sessions.StartAuth(sess, silent)

	opts := []oauth2.AuthCodeOption{oidc.Nonce(pending.Nonce)}
	if s.pkce {
		opts = append(opts,
			oauth2.SetAuthURLParam("code_challenge", oauth2.S256ChallengeFromVerifier(pending.CodeVerifier)),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		)
	}
	if silent {
		opts = append(opts, oauth2.SetAuthURLParam("prompt", "none"))
	}
	if connectorID != "" {
		opts = append(opts, oauth2.SetAuthURLParam("connector_id", connectorID))
	}

	// Providers that do not take offline_access as a scope want a parameter.
	if !s.offlineAsScope && containsScope(scopes, oidc.ScopeOfflineAccess) {
		opts = append(opts, oauth2.AccessTypeOffline)
		scopes = withoutScope(scopes, oidc.ScopeOfflineAccess)
	}

	http.Redirect(w, r, s.oauth2Config(scopes).AuthCodeURL(pending.State, opts...), http.StatusSeeOther)
}

// handleAuthCallback finishes an authorization this browser started.
func (s *Server) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	sess := s.session(w, r)

	state := r.FormValue("state")
	pending, ok := s.sessions.TakeAuth(sess, state)
	if !ok {
		// A state is good for one callback, so coming back to this URL — the
		// browser's Back button, a reload, a bookmark — finds nothing pending.
		// For a browser that already holds what the callback produced that is
		// not an error, it is the page it is looking for.
		if sess.LastTokens != nil {
			http.Redirect(w, r, "/tokens", http.StatusSeeOther)
			return
		}
		http.Error(w, "no authorization in progress for this state — a callback is good once, so start the flow again", http.StatusBadRequest)
		return
	}

	// A silent check answers a question, so its failure is a result: the
	// provider has no session for this browser, and neither should the app.
	if pending.Silent {
		s.sessions.MarkChecked(sess)

		if r.FormValue("error") != "" {
			// The provider has no session for this browser. Whether it never had
			// one or has just been told to end it, the app has no business
			// showing a signed-in user either way.
			s.sessions.SignOut(sess)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		if err := s.completeAuth(w, r, sess, pending); err != nil {
			log.Printf("silent session check: %v", err)
			s.sessions.SignOut(sess)
		}
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if errMsg := r.FormValue("error"); errMsg != "" {
		http.Error(w, errMsg+": "+r.FormValue("error_description"), http.StatusBadRequest)
		return
	}

	if err := s.completeAuth(w, r, sess, pending); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// The result lives at its own address rather than at the callback: a page
	// rendered on a URL that carries a one-use code is a page that cannot be
	// reloaded or come back to.
	s.sessions.RememberTokens(sess, "Authorization code", sess.Token, sess.IDToken)
	http.Redirect(w, r, "/tokens", http.StatusSeeOther)
}

// completeAuth exchanges the code and records the sign-in.
func (s *Server) completeAuth(w http.ResponseWriter, r *http.Request, sess *session.Session, pending *session.PendingAuth) error {
	code := r.FormValue("code")
	if code == "" {
		return fmt.Errorf("no code in callback: %q", r.URL.RawQuery)
	}

	ctx := oidc.ClientContext(r.Context(), s.client)

	var opts []oauth2.AuthCodeOption
	if s.pkce {
		opts = append(opts, oauth2.VerifierOption(pending.CodeVerifier))
	}

	token, err := s.oauth2Config(nil).Exchange(ctx, code, opts...)
	if err != nil {
		return fmt.Errorf("failed to get token: %v", err)
	}

	claims, rawIDToken, err := s.claimsFromToken(r, token)
	if err != nil {
		return err
	}

	// The nonce ties the ID token to the authorization this browser started.
	idToken, err := s.verifier.Verify(r.Context(), rawIDToken)
	if err == nil && idToken.Nonce != pending.Nonce {
		return fmt.Errorf("id token nonce does not match the authorization request")
	}

	s.sessions.SignIn(sess, claims, token, rawIDToken)
	return nil
}

// claimsFromToken verifies the ID token in a token response and returns its
// claims. A response without one is not an error for every grant, so callers
// that can live without it check the returned raw token instead.
func (s *Server) claimsFromToken(r *http.Request, token *oauth2.Token) (*session.UserClaims, string, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, "", fmt.Errorf("no id_token in token response")
	}

	idToken, err := s.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to verify ID token: %v", err)
	}

	var claims session.UserClaims
	if err := idToken.Claims(&claims); err != nil {
		return nil, "", fmt.Errorf("failed to decode ID token claims: %v", err)
	}
	return &claims, rawIDToken, nil
}

// handleLogout ends the app's session and, when the provider supports it, the
// provider's session too.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	sess := s.session(w, r)
	idToken := s.sessions.SignOut(sess)

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
	if appURL, err := url.Parse(s.redirectURI); err == nil {
		appURL.Path = "/"
		appURL.RawQuery = ""
		q.Set("post_logout_redirect_uri", appURL.String())
	}
	logoutURL.RawQuery = q.Encode()

	http.Redirect(w, r, logoutURL.String(), http.StatusFound)
}

func containsScope(scopes []string, want string) bool {
	for _, s := range scopes {
		if s == want {
			return true
		}
	}
	return false
}

func withoutScope(scopes []string, drop string) []string {
	out := make([]string, 0, len(scopes))
	for _, s := range scopes {
		if s != drop {
			out = append(out, s)
		}
	}
	return out
}
