package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"slices"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// generateSessionID creates a random session identifier
func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp if random fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// buildScopes constructs a scope list from base scopes and cross-client IDs
func buildScopes(baseScopes []string, crossClients []string) []string {
	scopes := make([]string, len(baseScopes))
	copy(scopes, baseScopes)

	// Add audience scopes for cross-client authorization
	for _, client := range crossClients {
		if client != "" {
			scopes = append(scopes, "audience:server:client_id:"+client)
		}
	}

	return uniqueStrings(scopes)
}

func (a *app) oauth2Config(scopes []string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     a.clientID,
		ClientSecret: a.clientSecret,
		Endpoint:     a.provider.Endpoint(),
		Scopes:       scopes,
		RedirectURL:  a.redirectURI,
	}
}

func uniqueStrings(values []string) []string {
	slices.Sort(values)
	values = slices.Compact(values)
	return values
}

// return an HTTP client which trusts the provided root CAs.
func httpClientForRootCAs(rootCAs string) (*http.Client, error) {
	tlsConfig := tls.Config{RootCAs: x509.NewCertPool()}
	rootCABytes, err := os.ReadFile(rootCAs)
	if err != nil {
		return nil, fmt.Errorf("failed to read root-ca: %v", err)
	}
	if !tlsConfig.RootCAs.AppendCertsFromPEM(rootCABytes) {
		return nil, fmt.Errorf("no certs found in root CA file %q", rootCAs)
	}
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tlsConfig,
			Proxy:           http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}, nil
}

type debugTransport struct {
	t http.RoundTripper
}

func (d debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		return nil, err
	}
	log.Printf("%s", reqDump)

	resp, err := d.t.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	respDump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}
	log.Printf("%s", respDump)
	return resp, nil
}

func encodeToken(idToken *oidc.IDToken) (string, error) {
	var claims json.RawMessage
	if err := idToken.Claims(&claims); err != nil {
		return "", fmt.Errorf("error decoding ID token claims: %v", err)
	}

	buff := new(bytes.Buffer)
	if err := json.Indent(buff, claims, "", "  "); err != nil {
		return "", fmt.Errorf("error indenting ID token claims: %v", err)
	}
	return buff.String(), nil
}

func parseAndRenderToken(w http.ResponseWriter, r *http.Request, a *app, token *oauth2.Token) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "no id_token in token response", http.StatusInternalServerError)
		return
	}

	idToken, err := a.verifier.Verify(r.Context(), rawIDToken)
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

	buf, err := encodeToken(idToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	renderToken(w, r.Context(), a.provider, a.redirectURI, rawIDToken, accessToken, token.RefreshToken, buf)
}
