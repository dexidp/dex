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
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// renderToken shows the result of a grant that produced a token through the
// oauth2 library.
func (s *Server) renderToken(w http.ResponseWriter, r *http.Request, grant string, token *oauth2.Token) {
	data := TokenPageData{
		LogoURI:      dexLogoDataURI,
		AdminEnabled: s.admin != nil,
		Grant:        grant,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		RedirectURL:  s.redirectURI,
		PublicKeyPEM: s.fetchPublicKeyPEM(),
	}

	if rawIDToken, ok := token.Extra("id_token").(string); ok {
		data.IDToken = rawIDToken
		data.IDTokenJWTLink = jwtIOLink(rawIDToken)
		data.Claims, _ = decodeJWTClaims(rawIDToken)
	}
	if data.AccessToken != "" {
		data.AccessTokenJWTLink = jwtIOLink(data.AccessToken)
		if data.Claims == "" {
			// A grant without an ID token — client credentials, token exchange
			// — still says who the token is for, in the access token itself.
			data.Claims, _ = decodeJWTClaims(data.AccessToken)
		}
	}
	if !token.Expiry.IsZero() {
		data.ExpiresIn = time.Until(token.Expiry).Round(time.Second).String()
	}

	s.renderer.RenderTokenPage(w, data)
}

// renderRawToken shows the result of a grant whose response the app reads
// directly, because it does not have the shape the oauth2 library expects.
func (s *Server) renderRawToken(w http.ResponseWriter, r *http.Request, grant string, body []byte) {
	var resp struct {
		AccessToken     string `json:"access_token"`
		IDToken         string `json:"id_token"`
		RefreshToken    string `json:"refresh_token"`
		IssuedTokenType string `json:"issued_token_type"`
		ExpiresIn       int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		http.Error(w, fmt.Sprintf("token response is not JSON: %v", err), http.StatusInternalServerError)
		return
	}

	data := TokenPageData{
		LogoURI:         dexLogoDataURI,
		AdminEnabled:    s.admin != nil,
		Grant:           grant,
		AccessToken:     resp.AccessToken,
		IDToken:         resp.IDToken,
		RefreshToken:    resp.RefreshToken,
		IssuedTokenType: resp.IssuedTokenType,
		RedirectURL:     s.redirectURI,
		PublicKeyPEM:    s.fetchPublicKeyPEM(),
		RawResponse:     indentJSON(body),
	}
	if resp.ExpiresIn > 0 {
		data.ExpiresIn = (time.Duration(resp.ExpiresIn) * time.Second).String()
	}
	if data.IDToken != "" {
		data.IDTokenJWTLink = jwtIOLink(data.IDToken)
		data.Claims, _ = decodeJWTClaims(data.IDToken)
	}
	if data.AccessToken != "" {
		data.AccessTokenJWTLink = jwtIOLink(data.AccessToken)
		if data.Claims == "" {
			data.Claims, _ = decodeJWTClaims(data.AccessToken)
		}
	}

	s.renderer.RenderTokenPage(w, data)
}

// decodeJWTClaims pretty-prints a JWT payload without verifying it: this is for
// looking at a token, and the verify tool is what says whether to believe it.
func decodeJWTClaims(token string) (string, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false
	}
	return indentJSON(payload), true
}

func indentJSON(raw []byte) string {
	buf := new(bytes.Buffer)
	if err := json.Indent(buf, raw, "", "  "); err != nil {
		return string(raw)
	}
	return buf.String()
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
