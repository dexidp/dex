package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

// handleTools renders the page for looking at tokens you already hold.
func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	s.renderer.RenderToolsPage(w, ToolsPageData{
		LogoURI:      dexLogoDataURI,
		AdminEnabled: s.admin != nil,
	})
}

// handleIntrospect asks the provider what it thinks of a token.
//
// This is the answer the app cannot work out for itself: an access token can
// look perfectly valid and still have been revoked, and only the issuer knows.
func (s *Server) handleIntrospect(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	token := strings.TrimSpace(r.FormValue("token"))
	if token == "" {
		http.Error(w, "token is required", http.StatusBadRequest)
		return
	}

	form := url.Values{"token": {token}}
	if hint := r.FormValue("token_type_hint"); hint != "" {
		form.Set("token_type_hint", hint)
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, s.introspectURL, strings.NewReader(form.Encode()))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(url.QueryEscape(s.clientID), url.QueryEscape(s.clientSecret))

	resp, err := s.client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("introspection request failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if resp.StatusCode != http.StatusOK {
		s.renderResult(w, r, "Introspection", fmt.Sprintf("%s: %s", resp.Status, strings.TrimSpace(string(body))), "")
		return
	}

	var result struct {
		Active bool `json:"active"`
	}
	_ = json.Unmarshal(body, &result)

	verdict := "The provider says this token is active."
	if !result.Active {
		verdict = "The provider says this token is not active — expired, revoked, or never issued by it."
	}

	s.renderResult(w, r, "Introspection", indentJSON(body), verdict)
}

// handleVerify checks a token's signature and claims locally, the way a
// resource server would: fetch the issuer's keys, verify, then look at what the
// token says. It is deliberately separate from introspection — this answers
// "was this signed by the issuer and is it still within its lifetime", not "is
// the issuer still honouring it".
func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	raw := strings.TrimSpace(r.FormValue("token"))
	if raw == "" {
		http.Error(w, "token is required", http.StatusBadRequest)
		return
	}

	ctx := oidc.ClientContext(r.Context(), s.client)

	// Skipping the audience check keeps the tool useful for tokens issued to
	// another client — the claims below still show who the audience is.
	verifier := s.provider.Verifier(&oidc.Config{SkipClientIDCheck: true})
	idToken, err := verifier.Verify(ctx, raw)
	if err != nil {
		claims, _ := decodeJWTClaims(raw)
		s.renderResult(w, r, "Local verification", claims, "Signature or claims rejected: "+err.Error())
		return
	}

	claims, _ := decodeJWTClaims(raw)
	verdict := fmt.Sprintf("Signature valid. Issued by %s for %v, expires %s.",
		idToken.Issuer, idToken.Audience, idToken.Expiry.Format(time.RFC3339))
	if idToken.Audience != nil && !containsScope(idToken.Audience, s.clientID) {
		verdict += " Note: this application is not in the audience."
	}

	s.renderResult(w, r, "Local verification", claims, verdict)
}

// handleUserInfo calls the provider's UserInfo endpoint with an access token.
func (s *Server) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	accessToken := strings.TrimSpace(r.FormValue("access_token"))
	if accessToken == "" {
		http.Error(w, "access_token is required", http.StatusBadRequest)
		return
	}
	if s.userInfoURL == "" {
		http.Error(w, "the provider does not advertise a userinfo endpoint", http.StatusBadRequest)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, s.userInfoURL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("userinfo request failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		s.renderResult(w, r, "UserInfo", fmt.Sprintf("%s: %s", resp.Status, strings.TrimSpace(string(body))), "")
		return
	}

	s.renderResult(w, r, "UserInfo", indentJSON(body), "")
}

// renderResult shows the output of a tool.
func (s *Server) renderResult(w http.ResponseWriter, r *http.Request, title, body, verdict string) {
	s.renderer.RenderResultPage(w, ResultPageData{
		LogoURI:      dexLogoDataURI,
		AdminEnabled: s.admin != nil,
		Title:        title,
		Verdict:      verdict,
		Body:         body,
		LastGrant:    s.session(w, r).LastGrant,
	})
}
