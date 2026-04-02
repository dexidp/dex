package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/dexidp/dex/examples/example-app/session"
)

// handleLoginPage renders the login page with available scopes.
// When session-aware mode is enabled, it checks for an existing Dex session
// via prompt=none and displays the authenticated user if found.
func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	data := LoginPageData{
		ScopesSupported: s.scopesSupported,
		LogoURI:         dexLogoDataURI,
	}

	if s.sessionAware {
		authState := s.auth.Get()

		if authState.Claims != nil {
			data.User = authState.Claims
			data.LogoutURL = "/app-logout"
		} else if !authState.Checked {
			// First visit: redirect to Dex with prompt=none to check session.
			scopes := []string{"openid", "profile", "email"}

			var opts []oauth2.AuthCodeOption
			opts = append(opts, oauth2.SetAuthURLParam("prompt", "none"))
			if s.pkce {
				opts = append(opts, oauth2.SetAuthURLParam("code_challenge", s.codeChallenge))
				opts = append(opts, oauth2.SetAuthURLParam("code_challenge_method", "S256"))
			}

			authCodeURL := s.oauth2Config(scopes).AuthCodeURL(silentAuthState, opts...)
			http.Redirect(w, r, authCodeURL, http.StatusFound)
			return
		} else {
			data.NotLoggedIn = true
		}
	}

	s.renderer.RenderLoginPage(w, data)
}

// handleLogin initiates the Authorization Code Flow by redirecting to the IdP.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	scopes := buildScopes(r.Form["extra_scopes"], r.Form["cross_client"])
	connectorID := r.FormValue("connector_id")

	var authCodeOptions []oauth2.AuthCodeOption

	if s.pkce {
		authCodeOptions = append(authCodeOptions,
			oauth2.SetAuthURLParam("code_challenge", s.codeChallenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		)
	}

	// If provider doesn't support "offline_access" scope natively,
	// use "access_type=offline" parameter instead (e.g. Google).
	hasOfflineAccess := false
	for _, scope := range scopes {
		if scope == oidc.ScopeOfflineAccess {
			hasOfflineAccess = true
			break
		}
	}

	if hasOfflineAccess && !s.offlineAsScope {
		authCodeOptions = append(authCodeOptions, oauth2.AccessTypeOffline)
		filtered := make([]string, 0, len(scopes))
		for _, scope := range scopes {
			if scope != oidc.ScopeOfflineAccess {
				filtered = append(filtered, scope)
			}
		}
		scopes = filtered
	}

	authCodeURL := s.oauth2Config(scopes).AuthCodeURL(exampleAppState, authCodeOptions...)

	u, err := url.Parse(authCodeURL)
	if err != nil {
		http.Error(w, "Failed to parse auth URL", http.StatusInternalServerError)
		return
	}

	if connectorID != "" {
		query := u.Query()
		query.Set("connector_id", connectorID)
		u.RawQuery = query.Encode()
	}

	http.Redirect(w, r, u.String(), http.StatusSeeOther)
}

// handleAuthCallback handles the OAuth2 authorization redirect callback.
// It validates the state parameter and exchanges the authorization code for tokens.
// It also handles silent auth callbacks (prompt=none) for session detection.
func (s *Server) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")

	// Silent auth callback (prompt=none).
	if state == silentAuthState {
		ctx := oidc.ClientContext(r.Context(), s.client)
		claims, rawIDToken := s.exchangeSilentAuth(ctx, r, s.oauth2Config(nil))
		s.auth.Set(claims, rawIDToken)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Normal authorization code callback.
	if errMsg := r.FormValue("error"); errMsg != "" {
		http.Error(w, errMsg+": "+r.FormValue("error_description"), http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	if code == "" {
		http.Error(w, fmt.Sprintf("no code in request: %q", r.Form), http.StatusBadRequest)
		return
	}

	if state != exampleAppState {
		http.Error(w, fmt.Sprintf("expected state %q got %q", exampleAppState, state), http.StatusBadRequest)
		return
	}

	ctx := oidc.ClientContext(r.Context(), s.client)

	var exchangeOpts []oauth2.AuthCodeOption
	if s.pkce {
		exchangeOpts = append(exchangeOpts, oauth2.SetAuthURLParam("code_verifier", s.codeVerifier))
	}

	token, err := s.oauth2Config(nil).Exchange(ctx, code, exchangeOpts...)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get token: %v", err), http.StatusInternalServerError)
		return
	}

	s.renderTokenResult(w, r, token)
}

// exchangeSilentAuth attempts a token exchange for a silent auth callback.
// Returns the parsed claims and raw ID token on success, or (nil, "") on any failure.
func (s *Server) exchangeSilentAuth(ctx context.Context, r *http.Request, oauth2Config *oauth2.Config) (*session.UserClaims, string) {
	if r.FormValue("error") != "" {
		return nil, ""
	}

	code := r.FormValue("code")
	if code == "" {
		return nil, ""
	}

	var opts []oauth2.AuthCodeOption
	if s.pkce {
		opts = append(opts, oauth2.SetAuthURLParam("code_verifier", s.codeVerifier))
	}

	token, err := oauth2Config.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, ""
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		log.Printf("silent auth: no id_token in response")
		return nil, ""
	}

	idToken, err := s.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		return nil, ""
	}

	var claims session.UserClaims
	_ = idToken.Claims(&claims)
	return &claims, rawIDToken
}
