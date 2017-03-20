package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	oidc "github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
)

// TODO(ericchiang): Generalize this as a library that others can import.

const (
	pathLogout      = "/logout"
	stateCookieName = "oidc-proxy-state-cookie"
	authCookieName  = "oidc-proxy-auth-cookie"
)

func init() {
	gob.Register(&stateCookie{})
	gob.Register(&authCookie{})
}

// stateCookie is used to verify the user coming to an endpoint is the
type stateCookie struct {
	// State passed to the frontend.
	State string
	// Initial path the.
	Path string
}

type authCookie struct {
	// Raw ID Token held by the server.
	IDToken string
}

// logger defines the interface this library uses.
type logger interface {
	Debugf(format string, i ...interface{})
	Infof(format string, i ...interface{})
	Errorf(format string, i ...interface{})
}

type config struct {
	issuerURL       string
	issuerTLSConfig *tls.Config

	// OAuth2 client credentials for the provider. It redirects to the.
	clientID     string
	clientSecret string

	insecureAllowHTTPCookies bool

	sessionSecret *[32]byte

	// Scopes used when authenticating with the OpenID Connect provider.
	//
	// Defaults to "openid profile email"
	scopes      []string
	redirectURI string

	backendURL       string
	backendTLSConfig *tls.Config

	authorizers []authorizer

	websocketSupport bool

	logger logger
}

// server is a HTTP handler used to proxy a host. It authenticates all requests
// using OpenID Connect, requiring the user to login before acccessing the backend.
type server struct {
	// HTTP proxy to the backend.
	proxy *proxy

	// Encrypted cookie store, used to give the frontend an encrypted ID Token
	// payload.
	store *cookieStore

	authorizer authorizer

	// OAuth2 client.
	oauth2Config *oauth2.Config
	// ID Token verifier.
	verifier *oidc.IDTokenVerifier

	logger logger
}

func newServer(c *config) (*server, error) {
	checks := []struct {
		bad bool
		msg string
	}{
		{c.clientID == "", "no client-id provided"},
		{c.clientSecret == "", "no client-secret provided"},
		{c.redirectURI == "", "no redirect-uri provided"},
		{c.issuerURL == "", "no issuer-url provided"},
		{c.backendURL == "", "no backend-url provided"},
	}
	for _, check := range checks {
		if check.bad {
			return nil, errors.New(check.msg)
		}
	}

	backend, err := url.Parse(c.backendURL)
	if err != nil {
		return nil, fmt.Errorf("failed parsing backend-url %s: %v", c.backendURL, err)
	}

	issuerClient := &http.Client{
		Transport: newTransport(c.issuerTLSConfig),
	}
	ctx := oidc.ClientContext(context.Background(), issuerClient)

	provider, err := oidc.NewProvider(ctx, c.issuerURL)
	if err != nil {
		return nil, fmt.Errorf("issuer metadata: %v", err)
	}

	p := &proxyConfig{
		enableWS:  c.websocketSupport,
		tlsConfig: c.backendTLSConfig,
		logger:    c.logger,
	}
	proxy := newProxy(backend, p)

	var a authorizer
	switch len(c.authorizers) {
	case 0:
		a = &allowAll{}
	case 1:
		a = c.authorizers[0]
	case 2:
		a = &unionAuthorizer{authorizers: c.authorizers}
	}

	return &server{
		logger:     c.logger,
		proxy:      proxy,
		authorizer: a,
		store: &cookieStore{
			insecureAllowHTTPCookies: c.insecureAllowHTTPCookies,
			key: c.sessionSecret,
		},
		oauth2Config: &oauth2.Config{
			ClientID:     c.clientID,
			ClientSecret: c.clientSecret,
			Scopes:       c.scopes,
			RedirectURL:  c.redirectURI,
			Endpoint:     provider.Endpoint(),
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: c.clientID}),
	}, nil
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.logger.Debugf("%s %s", r.Method, r.URL.Path)
	if r.Method == "GET" && r.URL.Path == "/logout" {
		s.handleLogout(w, r)
		return
	}

	if r.URL.Path == "/callback" {
		s.handleCallback(w, r)
		return
	}

	var a authCookie
	if err := s.cookie(r, authCookieName, &a); err != nil {
		s.handleLogin(w, r)
		return
	}

	token, err := s.verifyIDToken(r.Context(), a.IDToken)
	if err != nil {
		s.logger.Errorf("verifying id token: %v", err)
		s.clearCookies(w)
		s.handleLogin(w, r)
		return
	}

	if err := s.authorizer.authorized(token); err != nil {
		s.logger.Errorf("unauthorized: %v", err)
		s.clearCookies(w)
		s.errorf(w, http.StatusForbidden, "unauthorized user: %v", err)
		return
	}

	s.proxy.ServeHTTP(w, r)
}

func (s *server) verifyIDToken(ctx context.Context, idToken string) (*oidc.IDToken, error) {
	// TODO(ericchiang): Once we verify an ID Token is correctly signed
	// by the provider, we can probably cache the results.
	return s.verifier.Verify(ctx, idToken)
}

// errorf formats and serves a user facing error.
func (s *server) errorf(w http.ResponseWriter, status int, format string, v ...interface{}) {
	http.Error(w, fmt.Sprintf(format, v...), status)
	return
}

func (s *server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.clearCookies(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(err)
	}
	state := base64.RawURLEncoding.EncodeToString(b)
	p := r.URL.Path
	if len(r.URL.RawQuery) > 0 {
		p += "?" + r.URL.RawQuery
	}

	s.logger.Debugf("cooking user with state %s", state)
	c := stateCookie{State: state, Path: p}
	if err := s.setCookie(w, stateCookieName, c); err != nil {
		s.logger.Errorf("setting state cookie: %v", err)
		s.errorf(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	u := s.oauth2Config.AuthCodeURL(state)
	s.logger.Debugf("redirecting request to %s", u)
	http.Redirect(w, r, s.oauth2Config.AuthCodeURL(state), http.StatusFound)
}

func (s *server) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Ensure the state matches the value in the session.
	state := r.FormValue("state")

	var c stateCookie
	if err := s.cookie(r, stateCookieName, &c); err != nil {
		s.logger.Errorf("failed to decode state cookie: %v", err)
		s.errorf(w, http.StatusBadRequest, "User finishing the login flow was not the one that started it.")
		return
	} else if c.State != state {
		s.logger.Debugf("expected state %s got %s", c.State, state)
		s.errorf(w, http.StatusBadRequest, "User finishing the login flow was not the one that started it.")
		return
	}

	if errMsg := r.FormValue("error"); errMsg != "" {
		d := r.FormValue("error_description")
		s.logger.Errorf("oauth2 error: %v %v", errMsg, d)
		s.errorf(w, http.StatusBadRequest, "Error from provider %s: %s", errMsg, d)
		return
	}

	code := r.FormValue("code")
	if code == "" {
		http.Error(w, fmt.Sprintf("no code in request: %q", r.Form), http.StatusBadRequest)
		return
	}

	s.logger.Debugf("exchanging code %s with provider", code)
	token, err := s.oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		s.logger.Errorf("failed to exchange code for token: %v", err)
		s.errorf(w, http.StatusInternalServerError, "Failed to exchange code for token.")
		return
	}

	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		s.logger.Errorf("response did not contain an id_token")
		s.errorf(w, http.StatusInternalServerError, "Token response from provider did not contain an id_token.")
		return
	}
	a := authCookie{IDToken: idToken}
	if err := s.setCookie(w, authCookieName, a); err != nil {
		s.logger.Errorf("setting state cookie: %v", err)
		s.errorf(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// http.Redirect is always relative to the request URL. For now, add this hack
	// so the user ends up at the URL they originally requested. In the future, just
	// compose the redirect request ourselves.
	r.URL.Path = "/"

	http.Redirect(w, r, c.Path, http.StatusFound)
}

func (s *server) idToken(ctx context.Context, code string) (*oidc.IDToken, error) {
	token, err := s.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange failed: %v", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("token response did not contain a id_token field")
	}
	idToken, err := s.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("invalid id_token returned by provider: %v", err)
	}
	return idToken, nil
}

func (s *server) clearCookies(w http.ResponseWriter) {
	s.store.deleteCookie(w, authCookieName)
	s.store.deleteCookie(w, stateCookieName)
}

func (s *server) setCookie(w http.ResponseWriter, name string, i interface{}) error {
	b := new(bytes.Buffer)
	if err := gob.NewEncoder(b).Encode(i); err != nil {
		return fmt.Errorf("gob encode: %v", err)
	}
	return s.store.setCookie(w, name, b.Bytes())
}

func (s *server) cookie(r *http.Request, name string, i interface{}) error {
	raw := s.store.cookie(r, name)
	if raw == nil {
		return errors.New("no cookie found")
	}
	reader := bytes.NewReader(raw)
	if err := gob.NewDecoder(reader).Decode(i); err != nil {
		return fmt.Errorf("gob decode: %v", err)
	}
	return nil
}
