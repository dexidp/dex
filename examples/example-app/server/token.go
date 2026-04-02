package server

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/dexidp/dex/examples/example-app/session"
)

// handleTokenRefresh redeems a refresh token for a new token set.
func (s *Server) handleTokenRefresh(w http.ResponseWriter, r *http.Request) {
	refresh := r.FormValue("refresh_token")
	if refresh == "" {
		http.Error(w, fmt.Sprintf("no refresh_token in request: %q", r.Form), http.StatusBadRequest)
		return
	}

	ctx := oidc.ClientContext(r.Context(), s.client)

	t := &oauth2.Token{
		RefreshToken: refresh,
		Expiry:       time.Now().Add(-time.Hour),
	}
	token, err := s.oauth2Config(nil).TokenSource(ctx, t).Token()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get token: %v", err), http.StatusInternalServerError)
		return
	}

	s.renderTokenResult(w, r, token)
}

// renderTokenResult verifies an ID token, extracts claims, and renders the token page.
func (s *Server) renderTokenResult(w http.ResponseWriter, r *http.Request, token *oauth2.Token) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "no id_token in token response", http.StatusInternalServerError)
		return
	}

	idToken, err := s.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to verify ID token: %v", err), http.StatusInternalServerError)
		return
	}

	accessToken, ok := token.Extra("access_token").(string)
	if !ok {
		accessToken = token.AccessToken
		if accessToken == "" {
			http.Error(w, "no access_token in token response", http.StatusInternalServerError)
			return
		}
	}

	// Persist claims for session-aware index page and logout.
	var uc session.UserClaims
	_ = idToken.Claims(&uc)
	s.auth.Set(&uc, rawIDToken)

	claims, err := encodeIDTokenClaims(idToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.renderer.RenderTokenPage(w, TokenPageData{
		IDToken:            rawIDToken,
		IDTokenJWTLink:     jwtIOLink(rawIDToken),
		AccessToken:        accessToken,
		AccessTokenJWTLink: jwtIOLink(accessToken),
		RefreshToken:       token.RefreshToken,
		RedirectURL:        s.redirectURI,
		Claims:             claims,
		PublicKeyPEM:       s.fetchPublicKeyPEM(),
	})
}

// encodeIDTokenClaims extracts and pretty-prints the claims from an ID token.
func encodeIDTokenClaims(idToken *oidc.IDToken) (string, error) {
	var claims json.RawMessage
	if err := idToken.Claims(&claims); err != nil {
		return "", fmt.Errorf("error decoding ID token claims: %v", err)
	}

	buf := new(bytes.Buffer)
	if err := json.Indent(buf, claims, "", "  "); err != nil {
		return "", fmt.Errorf("error indenting ID token claims: %v", err)
	}
	return buf.String(), nil
}

// jwtIOLink creates a jwt.io debugger URL for the given token.
func jwtIOLink(token string) string {
	return "https://jwt.io/#debugger-io?token=" + url.QueryEscape(token)
}

// fetchPublicKeyPEM fetches the provider's JWKS and returns the first RSA public key as PEM.
func (s *Server) fetchPublicKeyPEM() string {
	if s.jwksURL == "" {
		return ""
	}

	resp, err := s.client.Get(s.jwksURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var jwks struct {
		Keys []json.RawMessage `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil || len(jwks.Keys) == 0 {
		return ""
	}

	var key struct {
		N   string `json:"n"`
		E   string `json:"e"`
		Kty string `json:"kty"`
	}
	if err := json.Unmarshal(jwks.Keys[0], &key); err != nil || key.Kty != "RSA" {
		return ""
	}

	nBytes, err1 := base64.RawURLEncoding.DecodeString(key.N)
	eBytes, err2 := base64.RawURLEncoding.DecodeString(key.E)
	if err1 != nil || err2 != nil {
		return ""
	}

	var eInt int
	for _, b := range eBytes {
		eInt = eInt<<8 | int(b)
	}

	pubKey := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: eInt,
	}

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return ""
	}

	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	}))
}
