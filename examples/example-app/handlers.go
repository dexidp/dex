package main

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func (a *app) handleIndex(w http.ResponseWriter, r *http.Request) {
	renderIndex(w, indexPageData{
		ScopesSupported: a.scopesSupported,
		LogoURI:         dexLogoDataURI,
	})
}

func (a *app) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	// Only use scopes that are checked in the form
	scopes := r.Form["extra_scopes"]
	crossClients := r.Form["cross_client"]

	// Build complete scope list with audience scopes
	scopes = buildScopes(scopes, crossClients)

	connectorID := ""
	if id := r.FormValue("connector_id"); id != "" {
		connectorID = id
	}

	authCodeURL := ""

	var authCodeOptions []oauth2.AuthCodeOption

	if a.pkce {
		authCodeOptions = append(authCodeOptions, oauth2.SetAuthURLParam("code_challenge", codeChallenge))
		authCodeOptions = append(authCodeOptions, oauth2.SetAuthURLParam("code_challenge_method", "S256"))
	}

	// Check if offline_access scope is present to determine offline access mode
	hasOfflineAccess := false
	for _, scope := range scopes {
		if scope == "offline_access" {
			hasOfflineAccess = true
			break
		}
	}

	if hasOfflineAccess && !a.offlineAsScope {
		// Provider uses access_type=offline instead of offline_access scope
		authCodeOptions = append(authCodeOptions, oauth2.AccessTypeOffline)
		// Remove offline_access from scopes as it's not supported
		filteredScopes := make([]string, 0, len(scopes))
		for _, scope := range scopes {
			if scope != "offline_access" {
				filteredScopes = append(filteredScopes, scope)
			}
		}
		scopes = filteredScopes
	}

	authCodeURL = a.oauth2Config(scopes).AuthCodeURL(exampleAppState, authCodeOptions...)

	// Parse the auth code URL and safely add connector_id parameter if provided
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

func (a *app) handleCallback(w http.ResponseWriter, r *http.Request) {
	var (
		err   error
		token *oauth2.Token
	)

	ctx := oidc.ClientContext(r.Context(), a.client)
	oauth2Config := a.oauth2Config(nil)
	switch r.Method {
	case http.MethodGet:
		// Authorization redirect callback from OAuth2 auth flow.
		if errMsg := r.FormValue("error"); errMsg != "" {
			http.Error(w, errMsg+": "+r.FormValue("error_description"), http.StatusBadRequest)
			return
		}
		code := r.FormValue("code")
		if code == "" {
			http.Error(w, fmt.Sprintf("no code in request: %q", r.Form), http.StatusBadRequest)
			return
		}
		if state := r.FormValue("state"); state != exampleAppState {
			http.Error(w, fmt.Sprintf("expected state %q got %q", exampleAppState, state), http.StatusBadRequest)
			return
		}

		var authCodeOptions []oauth2.AuthCodeOption
		if a.pkce {
			authCodeOptions = append(authCodeOptions, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
		}

		token, err = oauth2Config.Exchange(ctx, code, authCodeOptions...)
	case http.MethodPost:
		// Form request from frontend to refresh a token.
		refresh := r.FormValue("refresh_token")
		if refresh == "" {
			http.Error(w, fmt.Sprintf("no refresh_token in request: %q", r.Form), http.StatusBadRequest)
			return
		}
		t := &oauth2.Token{
			RefreshToken: refresh,
			Expiry:       time.Now().Add(-time.Hour),
		}
		token, err = oauth2Config.TokenSource(ctx, t).Token()
	default:
		http.Error(w, fmt.Sprintf("method not implemented: %s", r.Method), http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get token: %v", err), http.StatusInternalServerError)
		return
	}

	parseAndRenderToken(w, r, a, token)
}
