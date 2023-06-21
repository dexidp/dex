package server

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/kylelemons/godebug/pretty"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	jose "gopkg.in/square/go-jose.v2"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/connector/mock"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func mustLoad(s string) *rsa.PrivateKey {
	block, _ := pem.Decode([]byte(s))
	if block == nil {
		panic("no pem data found")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		panic(err)
	}
	return key
}

var testKey = mustLoad(`-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEArmoiX5G36MKPiVGS1sicruEaGRrbhPbIKOf97aGGQRjXVngo
Knwd2L4T9CRyABgQm3tLHHcT5crODoy46wX2g9onTZWViWWuhJ5wxXNmUbCAPWHb
j9SunW53WuLYZ/IJLNZt5XYCAFPjAakWp8uMuuDwWo5EyFaw85X3FSMhVmmaYDd0
cn+1H4+NS/52wX7tWmyvGUNJ8lzjFAnnOtBJByvkyIC7HDphkLQV4j//sMNY1mPX
HbsYgFv2J/LIJtkjdYO2UoDhZG3Gvj16fMy2JE2owA8IX4/s+XAmA2PiTfd0J5b4
drAKEcdDl83G6L3depEkTkfvp0ZLsh9xupAvIwIDAQABAoIBABKGgWonPyKA7+AF
AxS/MC0/CZebC6/+ylnV8lm4K1tkuRKdJp8EmeL4pYPsDxPFepYZLWwzlbB1rxdK
iSWld36fwEb0WXLDkxrQ/Wdrj3Wjyqs6ZqjLTVS5dAH6UEQSKDlT+U5DD4lbX6RA
goCGFUeQNtdXfyTMWHU2+4yKM7NKzUpczFky+0d10Mg0ANj3/4IILdr3hqkmMSI9
1TB9ksWBXJxt3nGxAjzSFihQFUlc231cey/HhYbvAX5fN0xhLxOk88adDcdXE7br
3Ser1q6XaaFQSMj4oi1+h3RAT9MUjJ6johEqjw0PbEZtOqXvA1x5vfFdei6SqgKn
Am3BspkCgYEA2lIiKEkT/Je6ZH4Omhv9atbGoBdETAstL3FnNQjkyVau9f6bxQkl
4/sz985JpaiasORQBiTGY8JDT/hXjROkut91agi2Vafhr29L/mto7KZglfDsT4b2
9z/EZH8wHw7eYhvdoBbMbqNDSI8RrGa4mpLpuN+E0wsFTzSZEL+QMQUCgYEAzIQh
xnreQvDAhNradMqLmxRpayn1ORaPReD4/off+mi7hZRLKtP0iNgEVEWHJ6HEqqi1
r38XAc8ap/lfOVMar2MLyCFOhYspdHZ+TGLZfr8gg/Fzeq9IRGKYadmIKVwjMeyH
REPqg1tyrvMOE0HI5oqkko8JTDJ0OyVC0Vc6+AcCgYAqCzkywugLc/jcU35iZVOH
WLdFq1Vmw5w/D7rNdtoAgCYPj6nV5y4Z2o2mgl6ifXbU7BMRK9Hc8lNeOjg6HfdS
WahV9DmRA1SuIWPkKjE5qczd81i+9AHpmakrpWbSBF4FTNKAewOBpwVVGuBPcDTK
59IE3V7J+cxa9YkotYuCNQKBgCwGla7AbHBEm2z+H+DcaUktD7R+B8gOTzFfyLoi
Tdj+CsAquDO0BQQgXG43uWySql+CifoJhc5h4v8d853HggsXa0XdxaWB256yk2Wm
MePTCRDePVm/ufLetqiyp1kf+IOaw1Oyux0j5oA62mDS3Iikd+EE4Z+BjPvefY/L
E2qpAoGAZo5Wwwk7q8b1n9n/ACh4LpE+QgbFdlJxlfFLJCKstl37atzS8UewOSZj
FDWV28nTP9sqbtsmU8Tem2jzMvZ7C/Q0AuDoKELFUpux8shm8wfIhyaPnXUGZoAZ
Np4vUwMSYV5mopESLWOg3loBxKyLGFtgGKVCjGiQvy6zISQ4fQo=
-----END RSA PRIVATE KEY-----`)

var logger = &logrus.Logger{
	Out:       os.Stderr,
	Formatter: &logrus.TextFormatter{DisableColors: true},
	Level:     logrus.DebugLevel,
}

func newTestServer(ctx context.Context, t *testing.T, updateConfig func(c *Config)) (*httptest.Server, *Server) {
	var server *Server
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.ServeHTTP(w, r)
	}))

	config := Config{
		Issuer:  s.URL,
		Storage: memory.New(logger),
		Web: WebConfig{
			Dir: "../web",
		},
		Logger:             logger,
		PrometheusRegistry: prometheus.NewRegistry(),
		HealthChecker:      gosundheit.New(),
		SkipApprovalScreen: true, // Don't prompt for approval, just immediately redirect with code.
	}
	if updateConfig != nil {
		updateConfig(&config)
	}
	s.URL = config.Issuer

	connector := storage.Connector{
		ID:              "mock",
		Type:            "mockCallback",
		Name:            "Mock",
		ResourceVersion: "1",
	}
	if err := config.Storage.CreateConnector(connector); err != nil {
		t.Fatalf("create connector: %v", err)
	}

	var err error
	if server, err = newServer(ctx, config, staticRotationStrategy(testKey)); err != nil {
		t.Fatal(err)
	}

	// Default rotation policy
	if server.refreshTokenPolicy == nil {
		server.refreshTokenPolicy, err = NewRefreshTokenPolicy(logger, false, "", "", "")
		if err != nil {
			t.Fatalf("failed to prepare rotation policy: %v", err)
		}
		server.refreshTokenPolicy.now = config.Now
	}

	return s, server
}

func newTestServerMultipleConnectors(ctx context.Context, t *testing.T, updateConfig func(c *Config)) (*httptest.Server, *Server) {
	var server *Server
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.ServeHTTP(w, r)
	}))

	config := Config{
		Issuer:  s.URL,
		Storage: memory.New(logger),
		Web: WebConfig{
			Dir: "../web",
		},
		Logger:             logger,
		PrometheusRegistry: prometheus.NewRegistry(),
	}
	if updateConfig != nil {
		updateConfig(&config)
	}
	s.URL = config.Issuer

	connector := storage.Connector{
		ID:              "mock",
		Type:            "mockCallback",
		Name:            "Mock",
		ResourceVersion: "1",
	}
	connector2 := storage.Connector{
		ID:              "mock2",
		Type:            "mockCallback",
		Name:            "Mock",
		ResourceVersion: "1",
	}
	if err := config.Storage.CreateConnector(connector); err != nil {
		t.Fatalf("create connector: %v", err)
	}
	if err := config.Storage.CreateConnector(connector2); err != nil {
		t.Fatalf("create connector: %v", err)
	}

	var err error
	if server, err = newServer(ctx, config, staticRotationStrategy(testKey)); err != nil {
		t.Fatal(err)
	}
	server.skipApproval = true // Don't prompt for approval, just immediately redirect with code.
	return s, server
}

func TestNewTestServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	newTestServer(ctx, t, nil)
}

func TestDiscovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpServer, _ := newTestServer(ctx, t, func(c *Config) {
		c.Issuer += "/non-root-path"
	})
	defer httpServer.Close()

	p, err := oidc.NewProvider(ctx, httpServer.URL)
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}

	var got map[string]*json.RawMessage
	if err := p.Claims(&got); err != nil {
		t.Fatalf("failed to decode claims: %v", err)
	}

	required := []string{
		"issuer",
		"authorization_endpoint",
		"token_endpoint",
		"jwks_uri",
		"userinfo_endpoint",
	}
	for _, field := range required {
		if _, ok := got[field]; !ok {
			t.Errorf("server discovery is missing required field %q", field)
		}
	}
}

type oauth2Tests struct {
	clientID string
	tests    []test
}

type test struct {
	name string
	// If specified these set of scopes will be used during the test case.
	scopes []string
	// handleToken provides the OAuth2 token response for the integration test.
	handleToken func(context.Context, *oidc.Provider, *oauth2.Config, *oauth2.Token, *mock.Callback) error

	// extra parameters to pass when requesting auth_code
	authCodeOptions []oauth2.AuthCodeOption

	// extra parameters to pass when retrieving id token
	retrieveTokenOptions []oauth2.AuthCodeOption

	// define an error response, when the test expects an error on the auth endpoint
	authError *OAuth2ErrorResponse

	// define an error response, when the test expects an error on the token endpoint
	tokenError ErrorResponse
}

// Defines an expected error by HTTP Status Code and
// the OAuth2 error int the response json
type ErrorResponse struct {
	Error      string
	StatusCode int
}

// https://tools.ietf.org/html/rfc6749#section-5.2
type OAuth2ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorURI         string `json:"error_uri"`
}

func makeOAuth2Tests(clientID string, clientSecret string, now func() time.Time) oauth2Tests {
	requestedScopes := []string{oidc.ScopeOpenID, "email", "profile", "groups", "offline_access"}

	// Used later when configuring test servers to set how long id_tokens will be valid for.
	//
	// The actual value of 30s is completely arbitrary. We just need to set a value
	// so tests can compute the expected "expires_in" field.
	idTokensValidFor := time.Second * 30

	oidcConfig := &oidc.Config{SkipClientIDCheck: true}

	basicIDTokenVerify := func(ctx context.Context, p *oidc.Provider, config *oauth2.Config, token *oauth2.Token, conn *mock.Callback) error {
		idToken, ok := token.Extra("id_token").(string)
		if !ok {
			return fmt.Errorf("no id token found")
		}
		if _, err := p.Verifier(oidcConfig).Verify(ctx, idToken); err != nil {
			return fmt.Errorf("failed to verify id token: %v", err)
		}
		return nil
	}

	return oauth2Tests{
		clientID: clientID,
		tests: []test{
			{
				name: "verify ID Token",
				handleToken: func(ctx context.Context, p *oidc.Provider, config *oauth2.Config, token *oauth2.Token, conn *mock.Callback) error {
					idToken, ok := token.Extra("id_token").(string)
					if !ok {
						return fmt.Errorf("no id token found")
					}
					if _, err := p.Verifier(oidcConfig).Verify(ctx, idToken); err != nil {
						return fmt.Errorf("failed to verify id token: %v", err)
					}
					return nil
				},
			},
			{
				name: "fetch userinfo",
				handleToken: func(ctx context.Context, p *oidc.Provider, config *oauth2.Config, token *oauth2.Token, conn *mock.Callback) error {
					ui, err := p.UserInfo(ctx, config.TokenSource(ctx, token))
					if err != nil {
						return fmt.Errorf("failed to fetch userinfo: %v", err)
					}
					if conn.Identity.Email != ui.Email {
						return fmt.Errorf("expected email to be %v, got %v", conn.Identity.Email, ui.Email)
					}
					return nil
				},
			},
			{
				name: "verify id token and oauth2 token expiry",
				handleToken: func(ctx context.Context, p *oidc.Provider, config *oauth2.Config, token *oauth2.Token, conn *mock.Callback) error {
					expectedExpiry := now().Add(idTokensValidFor)

					timeEq := func(t1, t2 time.Time, within time.Duration) bool {
						return t1.Sub(t2) < within
					}

					// TODO: This is a flaky test. We need something better (eg. clockwork).
					if !timeEq(token.Expiry, expectedExpiry, 2*time.Second) {
						return fmt.Errorf("expected expired_in to be %s, got %s", expectedExpiry, token.Expiry)
					}

					rawIDToken, ok := token.Extra("id_token").(string)
					if !ok {
						return fmt.Errorf("no id token found")
					}
					idToken, err := p.Verifier(oidcConfig).Verify(ctx, rawIDToken)
					if err != nil {
						return fmt.Errorf("failed to verify id token: %v", err)
					}
					if !timeEq(idToken.Expiry, expectedExpiry, time.Second) {
						return fmt.Errorf("expected id token expiry to be %s, got %s", expectedExpiry, token.Expiry)
					}
					return nil
				},
			},
			{
				name: "verify at_hash",
				handleToken: func(ctx context.Context, p *oidc.Provider, config *oauth2.Config, token *oauth2.Token, conn *mock.Callback) error {
					rawIDToken, ok := token.Extra("id_token").(string)
					if !ok {
						return fmt.Errorf("no id token found")
					}
					idToken, err := p.Verifier(oidcConfig).Verify(ctx, rawIDToken)
					if err != nil {
						return fmt.Errorf("failed to verify id token: %v", err)
					}

					var claims struct {
						AtHash string `json:"at_hash"`
					}
					if err := idToken.Claims(&claims); err != nil {
						return fmt.Errorf("failed to decode raw claims: %v", err)
					}
					if claims.AtHash == "" {
						return errors.New("no at_hash value in id_token")
					}
					wantAtHash, err := accessTokenHash(jose.RS256, token.AccessToken)
					if err != nil {
						return fmt.Errorf("computed expected at hash: %v", err)
					}
					if wantAtHash != claims.AtHash {
						return fmt.Errorf("expected at_hash=%q got=%q", wantAtHash, claims.AtHash)
					}

					return nil
				},
			},
			{
				name: "refresh token",
				handleToken: func(ctx context.Context, p *oidc.Provider, config *oauth2.Config, token *oauth2.Token, conn *mock.Callback) error {
					// have to use time.Now because the OAuth2 package uses it.
					token.Expiry = time.Now().Add(time.Second * -10)
					if token.Valid() {
						return errors.New("token shouldn't be valid")
					}

					newToken, err := config.TokenSource(ctx, token).Token()
					if err != nil {
						return fmt.Errorf("failed to refresh token: %v", err)
					}
					if token.RefreshToken == newToken.RefreshToken {
						return fmt.Errorf("old refresh token was the same as the new token %q", token.RefreshToken)
					}

					if _, err := config.TokenSource(ctx, token).Token(); err == nil {
						return errors.New("was able to redeem the same refresh token twice")
					}
					return nil
				},
			},
			{
				name: "refresh with explicit scopes",
				handleToken: func(ctx context.Context, p *oidc.Provider, config *oauth2.Config, token *oauth2.Token, conn *mock.Callback) error {
					v := url.Values{}
					v.Add("client_id", clientID)
					v.Add("client_secret", clientSecret)
					v.Add("grant_type", "refresh_token")
					v.Add("refresh_token", token.RefreshToken)
					v.Add("scope", strings.Join(requestedScopes, " "))
					resp, err := http.PostForm(p.Endpoint().TokenURL, v)
					if err != nil {
						return err
					}
					defer resp.Body.Close()
					if resp.StatusCode != http.StatusOK {
						dump, err := httputil.DumpResponse(resp, true)
						if err != nil {
							panic(err)
						}
						return fmt.Errorf("unexpected response: %s", dump)
					}
					if resp.Header.Get("Cache-Control") != "no-store" {
						return fmt.Errorf("cache-control header doesn't included in token response")
					}
					if resp.Header.Get("Pragma") != "no-cache" {
						return fmt.Errorf("pragma header doesn't included in token response")
					}
					return nil
				},
			},
			{
				name: "refresh with extra spaces",
				handleToken: func(ctx context.Context, p *oidc.Provider, config *oauth2.Config, token *oauth2.Token, conn *mock.Callback) error {
					v := url.Values{}
					v.Add("client_id", clientID)
					v.Add("client_secret", clientSecret)
					v.Add("grant_type", "refresh_token")
					v.Add("refresh_token", token.RefreshToken)

					// go-oidc adds an additional space before scopes when refreshing.
					// Since we support that client we choose to be more relaxed about
					// scope parsing, disregarding extra whitespace.
					v.Add("scope", " "+strings.Join(requestedScopes, " "))
					resp, err := http.PostForm(p.Endpoint().TokenURL, v)
					if err != nil {
						return err
					}
					defer resp.Body.Close()
					if resp.StatusCode != http.StatusOK {
						dump, err := httputil.DumpResponse(resp, true)
						if err != nil {
							panic(err)
						}
						return fmt.Errorf("unexpected response: %s", dump)
					}
					if resp.Header.Get("Cache-Control") != "no-store" {
						return fmt.Errorf("cache-control header doesn't included in token response")
					}
					if resp.Header.Get("Pragma") != "no-cache" {
						return fmt.Errorf("pragma header doesn't included in token response")
					}
					return nil
				},
			},
			{
				name:   "refresh with unauthorized scopes",
				scopes: []string{"openid", "email"},
				handleToken: func(ctx context.Context, p *oidc.Provider, config *oauth2.Config, token *oauth2.Token, conn *mock.Callback) error {
					v := url.Values{}
					v.Add("client_id", clientID)
					v.Add("client_secret", clientSecret)
					v.Add("grant_type", "refresh_token")
					v.Add("refresh_token", token.RefreshToken)
					// Request a scope that wasn't requested initially.
					v.Add("scope", "oidc email profile")
					resp, err := http.PostForm(p.Endpoint().TokenURL, v)
					if err != nil {
						return err
					}
					defer resp.Body.Close()
					if resp.StatusCode == http.StatusOK {
						dump, err := httputil.DumpResponse(resp, true)
						if err != nil {
							panic(err)
						}
						return fmt.Errorf("unexpected response: %s", dump)
					}
					return nil
				},
			},
			{
				name:   "refresh with different client id",
				scopes: []string{"openid", "email"},
				handleToken: func(ctx context.Context, p *oidc.Provider, config *oauth2.Config, token *oauth2.Token, conn *mock.Callback) error {
					v := url.Values{}
					v.Add("client_id", clientID)
					v.Add("client_secret", clientSecret)
					v.Add("grant_type", "refresh_token")
					v.Add("refresh_token", "existedrefrestoken")
					v.Add("scope", "oidc email")
					resp, err := http.PostForm(p.Endpoint().TokenURL, v)
					if err != nil {
						return err
					}

					defer resp.Body.Close()
					if resp.StatusCode != http.StatusBadRequest {
						return fmt.Errorf("expected status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
					}

					var respErr struct {
						Error       string `json:"error"`
						Description string `json:"error_description"`
					}

					if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
						return fmt.Errorf("cannot decode token response: %v", err)
					}

					if respErr.Error != errInvalidGrant {
						return fmt.Errorf("expected error %q, got %q", errInvalidGrant, respErr.Error)
					}

					expectedMsg := "Refresh token is invalid or has already been claimed by another client."
					if respErr.Description != expectedMsg {
						return fmt.Errorf("expected error description %q, got %q", expectedMsg, respErr.Description)
					}

					return nil
				},
			},
			{
				// This test ensures that the connector.RefreshConnector interface is being
				// used when clients request a refresh token.
				name: "refresh with identity changes",
				handleToken: func(ctx context.Context, p *oidc.Provider, config *oauth2.Config, token *oauth2.Token, conn *mock.Callback) error {
					// have to use time.Now because the OAuth2 package uses it.
					token.Expiry = time.Now().Add(time.Second * -10)
					if token.Valid() {
						return errors.New("token shouldn't be valid")
					}

					ident := connector.Identity{
						UserID:        "fooid",
						Username:      "foo",
						Email:         "foo@bar.com",
						EmailVerified: true,
						Groups:        []string{"foo", "bar"},
					}
					conn.Identity = ident

					type claims struct {
						Username      string   `json:"name"`
						Email         string   `json:"email"`
						EmailVerified bool     `json:"email_verified"`
						Groups        []string `json:"groups"`
					}
					want := claims{ident.Username, ident.Email, ident.EmailVerified, ident.Groups}

					newToken, err := config.TokenSource(ctx, token).Token()
					if err != nil {
						return fmt.Errorf("failed to refresh token: %v", err)
					}
					rawIDToken, ok := newToken.Extra("id_token").(string)
					if !ok {
						return fmt.Errorf("no id_token in refreshed token")
					}
					idToken, err := p.Verifier(oidcConfig).Verify(ctx, rawIDToken)
					if err != nil {
						return fmt.Errorf("failed to verify id token: %v", err)
					}
					var got claims
					if err := idToken.Claims(&got); err != nil {
						return fmt.Errorf("failed to unmarshal claims: %v", err)
					}

					if diff := pretty.Compare(want, got); diff != "" {
						return fmt.Errorf("got identity != want identity: %s", diff)
					}
					return nil
				},
			},
			{
				name: "unsupported grant type",
				retrieveTokenOptions: []oauth2.AuthCodeOption{
					oauth2.SetAuthURLParam("grant_type", "unsupported"),
				},
				handleToken: basicIDTokenVerify,
				tokenError: ErrorResponse{
					Error:      errUnsupportedGrantType,
					StatusCode: http.StatusBadRequest,
				},
			},
			{
				// This test ensures that PKCE work in "plain" mode (no code_challenge_method specified)
				name: "PKCE with plain",
				authCodeOptions: []oauth2.AuthCodeOption{
					oauth2.SetAuthURLParam("code_challenge", "challenge123"),
				},
				retrieveTokenOptions: []oauth2.AuthCodeOption{
					oauth2.SetAuthURLParam("code_verifier", "challenge123"),
				},
				handleToken: basicIDTokenVerify,
			},
			{
				// This test ensures that PKCE works in "S256" mode
				name: "PKCE with S256",
				authCodeOptions: []oauth2.AuthCodeOption{
					oauth2.SetAuthURLParam("code_challenge", "lyyl-X4a69qrqgEfUL8wodWic3Be9ZZ5eovBgIKKi-w"),
					oauth2.SetAuthURLParam("code_challenge_method", "S256"),
				},
				retrieveTokenOptions: []oauth2.AuthCodeOption{
					oauth2.SetAuthURLParam("code_verifier", "challenge123"),
				},
				handleToken: basicIDTokenVerify,
			},
			{
				// This test ensures that PKCE does fail with wrong code_verifier in "plain" mode
				name: "PKCE with plain and wrong code_verifier",
				authCodeOptions: []oauth2.AuthCodeOption{
					oauth2.SetAuthURLParam("code_challenge", "challenge123"),
				},
				retrieveTokenOptions: []oauth2.AuthCodeOption{
					oauth2.SetAuthURLParam("code_verifier", "challenge124"),
				},
				handleToken: basicIDTokenVerify,
				tokenError: ErrorResponse{
					Error:      errInvalidGrant,
					StatusCode: http.StatusBadRequest,
				},
			},
			{
				// This test ensures that PKCE fail with wrong code_verifier in "S256" mode
				name: "PKCE with S256 and wrong code_verifier",
				authCodeOptions: []oauth2.AuthCodeOption{
					oauth2.SetAuthURLParam("code_challenge", "lyyl-X4a69qrqgEfUL8wodWic3Be9ZZ5eovBgIKKi-w"),
					oauth2.SetAuthURLParam("code_challenge_method", "S256"),
				},
				retrieveTokenOptions: []oauth2.AuthCodeOption{
					oauth2.SetAuthURLParam("code_verifier", "challenge124"),
				},
				handleToken: basicIDTokenVerify,
				tokenError: ErrorResponse{
					Error:      errInvalidGrant,
					StatusCode: http.StatusBadRequest,
				},
			},
			{
				// Ensure that, when PKCE flow started on /auth
				// we stay in PKCE flow on /token
				name: "PKCE flow expected on /token",
				authCodeOptions: []oauth2.AuthCodeOption{
					oauth2.SetAuthURLParam("code_challenge", "lyyl-X4a69qrqgEfUL8wodWic3Be9ZZ5eovBgIKKi-w"),
					oauth2.SetAuthURLParam("code_challenge_method", "S256"),
				},
				retrieveTokenOptions: []oauth2.AuthCodeOption{
					// No PKCE call on /token
				},
				handleToken: basicIDTokenVerify,
				tokenError: ErrorResponse{
					Error:      errInvalidGrant,
					StatusCode: http.StatusBadRequest,
				},
			},
			{
				// Ensure that when no PKCE flow was started on /auth
				// we cannot switch to PKCE on /token
				name:            "No PKCE flow started on /auth",
				authCodeOptions: []oauth2.AuthCodeOption{
					// No PKCE call on /auth
				},
				retrieveTokenOptions: []oauth2.AuthCodeOption{
					oauth2.SetAuthURLParam("code_verifier", "challenge123"),
				},
				handleToken: basicIDTokenVerify,
				tokenError: ErrorResponse{
					Error:      errInvalidRequest,
					StatusCode: http.StatusBadRequest,
				},
			},
			{
				// Make sure that, when we start with "S256" on /auth, we cannot downgrade to "plain" on /token
				name: "PKCE with S256 and try to downgrade to plain",
				authCodeOptions: []oauth2.AuthCodeOption{
					oauth2.SetAuthURLParam("code_challenge", "lyyl-X4a69qrqgEfUL8wodWic3Be9ZZ5eovBgIKKi-w"),
					oauth2.SetAuthURLParam("code_challenge_method", "S256"),
				},
				retrieveTokenOptions: []oauth2.AuthCodeOption{
					oauth2.SetAuthURLParam("code_verifier", "lyyl-X4a69qrqgEfUL8wodWic3Be9ZZ5eovBgIKKi-w"),
					oauth2.SetAuthURLParam("code_challenge_method", "plain"),
				},
				handleToken: basicIDTokenVerify,
				tokenError: ErrorResponse{
					Error:      errInvalidGrant,
					StatusCode: http.StatusBadRequest,
				},
			},
			{
				name: "Request parameter in authorization query",
				authCodeOptions: []oauth2.AuthCodeOption{
					oauth2.SetAuthURLParam("request", "anything"),
				},
				authError: &OAuth2ErrorResponse{
					Error:            errRequestNotSupported,
					ErrorDescription: "Server does not support request parameter.",
				},
				handleToken: func(ctx context.Context, p *oidc.Provider, config *oauth2.Config, token *oauth2.Token, conn *mock.Callback) error {
					return nil
				},
			},
		},
	}
}

// TestOAuth2CodeFlow runs integration tests against a test server. The tests stand up a server
// which requires no interaction to login, logs in through a test client, then passes the client
// and returned token to the test.
func TestOAuth2CodeFlow(t *testing.T) {
	clientID := "testclient"
	clientSecret := "testclientsecret"
	requestedScopes := []string{oidc.ScopeOpenID, "email", "profile", "groups", "offline_access"}

	t0 := time.Now()

	// Always have the time function used by the server return the same time so
	// we can predict expected values of "expires_in" fields exactly.
	now := func() time.Time { return t0 }

	// Used later when configuring test servers to set how long id_tokens will be valid for.
	//
	// The actual value of 30s is completely arbitrary. We just need to set a value
	// so tests can compute the expected "expires_in" field.
	idTokensValidFor := time.Second * 30

	// Connector used by the tests.
	var conn *mock.Callback

	tests := makeOAuth2Tests(clientID, clientSecret, now)
	for _, tc := range tests.tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Setup a dex server.
			httpServer, s := newTestServer(ctx, t, func(c *Config) {
				c.Issuer += "/non-root-path"
				c.Now = now
				c.IDTokensValidFor = idTokensValidFor
			})
			defer httpServer.Close()

			mockConn := s.connectors["mock"]
			conn = mockConn.Connector.(*mock.Callback)

			// Query server's provider metadata.
			p, err := oidc.NewProvider(ctx, httpServer.URL)
			if err != nil {
				t.Fatalf("failed to get provider: %v", err)
			}

			var (
				// If the OAuth2 client didn't get a response, we need
				// to print the requests the user saw.
				gotCode           bool
				reqDump, respDump []byte // Auth step, not token.
				state             = "a_state"
			)
			defer func() {
				if !gotCode && tc.authError == nil {
					t.Errorf("never got a code in callback\n%s\n%s", reqDump, respDump)
				}
			}()

			// Setup OAuth2 client.
			var oauth2Config *oauth2.Config
			oauth2Client := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/callback" {
					// User is visiting app first time. Redirect to dex.
					http.Redirect(w, r, oauth2Config.AuthCodeURL(state, tc.authCodeOptions...), http.StatusSeeOther)
					return
				}

				// User is at '/callback' so they were just redirected _from_ dex.
				q := r.URL.Query()

				// Did dex return an error?
				if errType := q.Get("error"); errType != "" {
					description := q.Get("error_description")

					if tc.authError == nil {
						if description != "" {
							t.Errorf("got error from server %s: %s", errType, description)
						} else {
							t.Errorf("got error from server %s", errType)
						}
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					require.Equal(t, *tc.authError, OAuth2ErrorResponse{Error: errType, ErrorDescription: description})
					return
				}

				// Grab code, exchange for token.
				if code := q.Get("code"); code != "" {
					gotCode = true
					token, err := oauth2Config.Exchange(ctx, code, tc.retrieveTokenOptions...)
					if tc.tokenError.StatusCode != 0 {
						checkErrorResponse(err, t, tc)
						return
					}

					if err != nil {
						t.Errorf("failed to exchange code for token: %v", err)
						return
					}
					err = tc.handleToken(ctx, p, oauth2Config, token, conn)
					if err != nil {
						t.Errorf("%s: %v", tc.name, err)
					}
					return
				}

				// Ensure state matches.
				if gotState := q.Get("state"); gotState != state {
					t.Errorf("state did not match, want=%q got=%q", state, gotState)
				}
				w.WriteHeader(http.StatusOK)
			}))

			defer oauth2Client.Close()

			// Register the client above with dex.
			redirectURL := oauth2Client.URL + "/callback"
			client := storage.Client{
				ID:           clientID,
				Secret:       clientSecret,
				RedirectURIs: []string{redirectURL},
			}
			if err := s.storage.CreateClient(client); err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			if err := s.storage.CreateRefresh(storage.RefreshToken{
				ID:       "existedrefrestoken",
				ClientID: "unexcistedclientid",
			}); err != nil {
				t.Fatalf("failed to create existed refresh token: %v", err)
			}

			// Create the OAuth2 config.
			oauth2Config = &oauth2.Config{
				ClientID:     client.ID,
				ClientSecret: client.Secret,
				Endpoint:     p.Endpoint(),
				Scopes:       requestedScopes,
				RedirectURL:  redirectURL,
			}
			if len(tc.scopes) != 0 {
				oauth2Config.Scopes = tc.scopes
			}

			// Login!
			//
			//   1. First request to client, redirects to dex.
			//   2. Dex "logs in" the user, redirects to client with "code".
			//   3. Client exchanges "code" for "token" (id_token, refresh_token, etc.).
			//   4. Test is run with OAuth2 token response.
			//
			resp, err := http.Get(oauth2Client.URL + "/login")
			if err != nil {
				t.Fatalf("get failed: %v", err)
			}
			defer resp.Body.Close()

			if reqDump, err = httputil.DumpRequest(resp.Request, false); err != nil {
				t.Fatal(err)
			}
			if respDump, err = httputil.DumpResponse(resp, true); err != nil {
				t.Fatal(err)
			}

			tokens, err := s.storage.ListRefreshTokens()
			if err != nil {
				t.Fatalf("failed to get existed refresh token: %v", err)
			}

			for _, token := range tokens {
				if /* token was updated */ token.ObsoleteToken != "" && token.ConnectorData != nil {
					t.Fatalf("token connectorData with id %q field is not nil: %s", token.ID, token.ConnectorData)
				}
			}
		})
	}
}

func TestOAuth2ImplicitFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpServer, s := newTestServer(ctx, t, func(c *Config) {
		// Enable support for the implicit flow.
		c.SupportedResponseTypes = []string{"code", "token", "id_token"}
	})
	defer httpServer.Close()

	p, err := oidc.NewProvider(ctx, httpServer.URL)
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}

	var (
		reqDump, respDump []byte
		gotIDToken        bool
		state             = "a_state"
		nonce             = "a_nonce"
	)
	defer func() {
		if !gotIDToken {
			t.Errorf("never got a id token in fragment\n%s\n%s", reqDump, respDump)
		}
	}()

	var oauth2Config *oauth2.Config
	oauth2Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/callback" {
			q := r.URL.Query()
			if errType := q.Get("error"); errType != "" {
				if desc := q.Get("error_description"); desc != "" {
					t.Errorf("got error from server %s: %s", errType, desc)
				} else {
					t.Errorf("got error from server %s", errType)
				}
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// Fragment is checked by the client since net/http servers don't preserve URL fragments.
			// E.g.
			//
			//    r.URL.Fragment
			//
			// Will always be empty.
			w.WriteHeader(http.StatusOK)
			return
		}
		u := oauth2Config.AuthCodeURL(state, oauth2.SetAuthURLParam("response_type", "id_token token"), oidc.Nonce(nonce))
		http.Redirect(w, r, u, http.StatusSeeOther)
	}))

	defer oauth2Server.Close()

	redirectURL := oauth2Server.URL + "/callback"
	client := storage.Client{
		ID:           "testclient",
		Secret:       "testclientsecret",
		RedirectURIs: []string{redirectURL},
	}
	if err := s.storage.CreateClient(client); err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	idTokenVerifier := p.Verifier(&oidc.Config{
		ClientID: client.ID,
	})

	oauth2Config = &oauth2.Config{
		ClientID:     client.ID,
		ClientSecret: client.Secret,
		Endpoint:     p.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "offline_access"},
		RedirectURL:  redirectURL,
	}

	checkIDToken := func(u *url.URL) error {
		if u.Fragment == "" {
			return fmt.Errorf("url has no fragment: %s", u)
		}
		v, err := url.ParseQuery(u.Fragment)
		if err != nil {
			return fmt.Errorf("failed to parse fragment: %v", err)
		}
		rawIDToken := v.Get("id_token")
		if rawIDToken == "" {
			return errors.New("no id_token in fragment")
		}
		idToken, err := idTokenVerifier.Verify(ctx, rawIDToken)
		if err != nil {
			return fmt.Errorf("failed to verify id_token: %v", err)
		}
		if idToken.Nonce != nonce {
			return fmt.Errorf("failed to verify id_token: nonce was %v, but want %v", idToken.Nonce, nonce)
		}
		return nil
	}

	httpClient := &http.Client{
		// net/http servers don't preserve URL fragments when passing the request to
		// handlers. The only way to get at that values is to check the redirect on
		// the client side.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 10 {
				return errors.New("too many redirects")
			}

			// If we're being redirected back to the client server, inspect the URL fragment
			// for an ID Token.
			u := req.URL.String()
			if strings.HasPrefix(u, oauth2Server.URL) {
				if err := checkIDToken(req.URL); err == nil {
					gotIDToken = true
				} else {
					t.Error(err)
				}
			}
			return nil
		},
	}

	resp, err := httpClient.Get(oauth2Server.URL + "/login")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	defer resp.Body.Close()

	if reqDump, err = httputil.DumpRequest(resp.Request, false); err != nil {
		t.Fatal(err)
	}
	if respDump, err = httputil.DumpResponse(resp, true); err != nil {
		t.Fatal(err)
	}
}

func TestCrossClientScopes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpServer, s := newTestServer(ctx, t, func(c *Config) {
		c.Issuer += "/non-root-path"
	})
	defer httpServer.Close()

	p, err := oidc.NewProvider(ctx, httpServer.URL)
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}

	var (
		reqDump, respDump []byte
		gotCode           bool
		state             = "a_state"
	)
	defer func() {
		if !gotCode {
			t.Errorf("never got a code in callback\n%s\n%s", reqDump, respDump)
		}
	}()

	testClientID := "testclient"
	peerID := "peer"

	var oauth2Config *oauth2.Config
	oauth2Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/callback" {
			q := r.URL.Query()
			if errType := q.Get("error"); errType != "" {
				if desc := q.Get("error_description"); desc != "" {
					t.Errorf("got error from server %s: %s", errType, desc)
				} else {
					t.Errorf("got error from server %s", errType)
				}
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if code := q.Get("code"); code != "" {
				gotCode = true
				token, err := oauth2Config.Exchange(ctx, code)
				if err != nil {
					t.Errorf("failed to exchange code for token: %v", err)
					return
				}
				rawIDToken, ok := token.Extra("id_token").(string)
				if !ok {
					t.Errorf("no id token found: %v", err)
					return
				}
				idToken, err := p.Verifier(&oidc.Config{ClientID: testClientID}).Verify(ctx, rawIDToken)
				if err != nil {
					t.Errorf("failed to parse ID Token: %v", err)
					return
				}

				sort.Strings(idToken.Audience)
				expAudience := []string{peerID, testClientID}
				if !reflect.DeepEqual(idToken.Audience, expAudience) {
					t.Errorf("expected audience %q, got %q", expAudience, idToken.Audience)
				}
			}
			if gotState := q.Get("state"); gotState != state {
				t.Errorf("state did not match, want=%q got=%q", state, gotState)
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, oauth2Config.AuthCodeURL(state), http.StatusSeeOther)
	}))

	defer oauth2Server.Close()

	redirectURL := oauth2Server.URL + "/callback"
	client := storage.Client{
		ID:           testClientID,
		Secret:       "testclientsecret",
		RedirectURIs: []string{redirectURL},
	}
	if err := s.storage.CreateClient(client); err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	peer := storage.Client{
		ID:           peerID,
		Secret:       "foobar",
		TrustedPeers: []string{"testclient"},
	}

	if err := s.storage.CreateClient(peer); err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	oauth2Config = &oauth2.Config{
		ClientID:     client.ID,
		ClientSecret: client.Secret,
		Endpoint:     p.Endpoint(),
		Scopes: []string{
			oidc.ScopeOpenID, "profile", "email",
			"audience:server:client_id:" + client.ID,
			"audience:server:client_id:" + peer.ID,
		},
		RedirectURL: redirectURL,
	}

	resp, err := http.Get(oauth2Server.URL + "/login")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	defer resp.Body.Close()

	if reqDump, err = httputil.DumpRequest(resp.Request, false); err != nil {
		t.Fatal(err)
	}
	if respDump, err = httputil.DumpResponse(resp, true); err != nil {
		t.Fatal(err)
	}
}

func TestCrossClientScopesWithAzpInAudienceByDefault(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpServer, s := newTestServer(ctx, t, func(c *Config) {
		c.Issuer += "/non-root-path"
	})
	defer httpServer.Close()

	p, err := oidc.NewProvider(ctx, httpServer.URL)
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}

	var (
		reqDump, respDump []byte
		gotCode           bool
		state             = "a_state"
	)
	defer func() {
		if !gotCode {
			t.Errorf("never got a code in callback\n%s\n%s", reqDump, respDump)
		}
	}()

	testClientID := "testclient"
	peerID := "peer"

	var oauth2Config *oauth2.Config
	oauth2Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/callback" {
			q := r.URL.Query()
			if errType := q.Get("error"); errType != "" {
				if desc := q.Get("error_description"); desc != "" {
					t.Errorf("got error from server %s: %s", errType, desc)
				} else {
					t.Errorf("got error from server %s", errType)
				}
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if code := q.Get("code"); code != "" {
				gotCode = true
				token, err := oauth2Config.Exchange(ctx, code)
				if err != nil {
					t.Errorf("failed to exchange code for token: %v", err)
					return
				}
				rawIDToken, ok := token.Extra("id_token").(string)
				if !ok {
					t.Errorf("no id token found: %v", err)
					return
				}
				idToken, err := p.Verifier(&oidc.Config{ClientID: testClientID}).Verify(ctx, rawIDToken)
				if err != nil {
					t.Errorf("failed to parse ID Token: %v", err)
					return
				}

				sort.Strings(idToken.Audience)
				expAudience := []string{peerID, testClientID}
				if !reflect.DeepEqual(idToken.Audience, expAudience) {
					t.Errorf("expected audience %q, got %q", expAudience, idToken.Audience)
				}
			}
			if gotState := q.Get("state"); gotState != state {
				t.Errorf("state did not match, want=%q got=%q", state, gotState)
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, oauth2Config.AuthCodeURL(state), http.StatusSeeOther)
	}))

	defer oauth2Server.Close()

	redirectURL := oauth2Server.URL + "/callback"
	client := storage.Client{
		ID:           testClientID,
		Secret:       "testclientsecret",
		RedirectURIs: []string{redirectURL},
	}
	if err := s.storage.CreateClient(client); err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	peer := storage.Client{
		ID:           peerID,
		Secret:       "foobar",
		TrustedPeers: []string{"testclient"},
	}

	if err := s.storage.CreateClient(peer); err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	oauth2Config = &oauth2.Config{
		ClientID:     client.ID,
		ClientSecret: client.Secret,
		Endpoint:     p.Endpoint(),
		Scopes: []string{
			oidc.ScopeOpenID, "profile", "email",
			"audience:server:client_id:" + peer.ID,
		},
		RedirectURL: redirectURL,
	}

	resp, err := http.Get(oauth2Server.URL + "/login")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	defer resp.Body.Close()

	if reqDump, err = httputil.DumpRequest(resp.Request, false); err != nil {
		t.Fatal(err)
	}
	if respDump, err = httputil.DumpResponse(resp, true); err != nil {
		t.Fatal(err)
	}
}

func TestPasswordDB(t *testing.T) {
	s := memory.New(logger)
	conn := newPasswordDB(s)

	pw := "hi"

	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	s.CreatePassword(storage.Password{
		Email:    "jane@example.com",
		Username: "jane",
		UserID:   "foobar",
		Hash:     h,
	})

	tests := []struct {
		name         string
		username     string
		password     string
		wantIdentity connector.Identity
		wantInvalid  bool
		wantErr      bool
	}{
		{
			name:     "valid password",
			username: "jane@example.com",
			password: pw,
			wantIdentity: connector.Identity{
				Email:         "jane@example.com",
				Username:      "jane",
				UserID:        "foobar",
				EmailVerified: true,
			},
		},
		{
			name:        "unknown user",
			username:    "john@example.com",
			password:    pw,
			wantInvalid: true,
		},
		{
			name:        "invalid password",
			username:    "jane@example.com",
			password:    "not the correct password",
			wantInvalid: true,
		},
	}

	for _, tc := range tests {
		ident, valid, err := conn.Login(context.Background(), connector.Scopes{}, tc.username, tc.password)
		if err != nil {
			if !tc.wantErr {
				t.Errorf("%s: %v", tc.name, err)
			}
			continue
		}

		if tc.wantErr {
			t.Errorf("%s: expected error", tc.name)
			continue
		}

		if !valid {
			if !tc.wantInvalid {
				t.Errorf("%s: expected valid password", tc.name)
			}
			continue
		}

		if tc.wantInvalid {
			t.Errorf("%s: expected invalid password", tc.name)
			continue
		}

		if diff := pretty.Compare(tc.wantIdentity, ident); diff != "" {
			t.Errorf("%s: %s", tc.name, diff)
		}
	}
}

func TestPasswordDBUsernamePrompt(t *testing.T) {
	s := memory.New(logger)
	conn := newPasswordDB(s)

	expected := "Email Address"
	if actual := conn.Prompt(); actual != expected {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

type storageWithKeysTrigger struct {
	storage.Storage
	f func()
}

func (s storageWithKeysTrigger) GetKeys() (storage.Keys, error) {
	s.f()
	return s.Storage.GetKeys()
}

func TestKeyCacher(t *testing.T) {
	tNow := time.Now()
	now := func() time.Time { return tNow }

	s := memory.New(logger)

	tests := []struct {
		before            func()
		wantCallToStorage bool
	}{
		{
			before:            func() {},
			wantCallToStorage: true,
		},
		{
			before: func() {
				s.UpdateKeys(func(old storage.Keys) (storage.Keys, error) {
					old.NextRotation = tNow.Add(time.Minute)
					return old, nil
				})
			},
			wantCallToStorage: true,
		},
		{
			before:            func() {},
			wantCallToStorage: false,
		},
		{
			before: func() {
				tNow = tNow.Add(time.Hour)
			},
			wantCallToStorage: true,
		},
		{
			before: func() {
				tNow = tNow.Add(time.Hour)
				s.UpdateKeys(func(old storage.Keys) (storage.Keys, error) {
					old.NextRotation = tNow.Add(time.Minute)
					return old, nil
				})
			},
			wantCallToStorage: true,
		},
		{
			before:            func() {},
			wantCallToStorage: false,
		},
	}

	gotCall := false
	s = newKeyCacher(storageWithKeysTrigger{s, func() { gotCall = true }}, now)
	for i, tc := range tests {
		gotCall = false
		tc.before()
		s.GetKeys()
		if gotCall != tc.wantCallToStorage {
			t.Errorf("case %d: expected call to storage=%t got call to storage=%t", i, tc.wantCallToStorage, gotCall)
		}
	}
}

func checkErrorResponse(err error, t *testing.T, tc test) {
	if err == nil {
		t.Errorf("%s: DANGEROUS! got a token when we should not get one!", tc.name)
		return
	}
	if rErr, ok := err.(*oauth2.RetrieveError); ok {
		if rErr.Response.StatusCode != tc.tokenError.StatusCode {
			t.Errorf("%s: got wrong StatusCode from server %d. expected %d",
				tc.name, rErr.Response.StatusCode, tc.tokenError.StatusCode)
		}
		details := new(OAuth2ErrorResponse)
		if err := json.Unmarshal(rErr.Body, details); err != nil {
			t.Errorf("%s: could not parse return json: %s", tc.name, err)
			return
		}
		if tc.tokenError.Error != "" && details.Error != tc.tokenError.Error {
			t.Errorf("%s: got wrong Error in response: %s (%s). expected %s",
				tc.name, details.Error, details.ErrorDescription, tc.tokenError.Error)
		}
	} else {
		t.Errorf("%s: unexpected error type: %s. expected *oauth2.RetrieveError", tc.name, reflect.TypeOf(err))
	}
}

type oauth2Client struct {
	config *oauth2.Config
	token  *oauth2.Token
	server *httptest.Server
}

// TestRefreshTokenFlow tests the refresh token code flow for oauth2. The test verifies
// that only valid refresh tokens can be used to refresh an expired token.
func TestRefreshTokenFlow(t *testing.T) {
	state := "state"
	now := time.Now
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpServer, s := newTestServer(ctx, t, func(c *Config) {
		c.Now = now
	})
	defer httpServer.Close()

	p, err := oidc.NewProvider(ctx, httpServer.URL)
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}

	var oauth2Client oauth2Client

	oauth2Client.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/callback" {
			// User is visiting app first time. Redirect to dex.
			http.Redirect(w, r, oauth2Client.config.AuthCodeURL(state), http.StatusSeeOther)
			return
		}

		// User is at '/callback' so they were just redirected _from_ dex.
		q := r.URL.Query()

		if errType := q.Get("error"); errType != "" {
			if desc := q.Get("error_description"); desc != "" {
				t.Errorf("got error from server %s: %s", errType, desc)
			} else {
				t.Errorf("got error from server %s", errType)
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Grab code, exchange for token.
		if code := q.Get("code"); code != "" {
			token, err := oauth2Client.config.Exchange(ctx, code)
			if err != nil {
				t.Errorf("failed to exchange code for token: %v", err)
				return
			}
			oauth2Client.token = token
		}

		// Ensure state matches.
		if gotState := q.Get("state"); gotState != state {
			t.Errorf("state did not match, want=%q got=%q", state, gotState)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer oauth2Client.server.Close()

	// Register the client above with dex.
	redirectURL := oauth2Client.server.URL + "/callback"
	client := storage.Client{
		ID:           "testclient",
		Secret:       "testclientsecret",
		RedirectURIs: []string{redirectURL},
	}
	if err := s.storage.CreateClient(client); err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	oauth2Client.config = &oauth2.Config{
		ClientID:     client.ID,
		ClientSecret: client.Secret,
		Endpoint:     p.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "email", "offline_access"},
		RedirectURL:  redirectURL,
	}

	resp, err := http.Get(oauth2Client.server.URL + "/login")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	defer resp.Body.Close()

	tok := &oauth2.Token{
		RefreshToken: oauth2Client.token.RefreshToken,
		Expiry:       time.Now().Add(-time.Hour),
	}

	// Login in again to receive a new token.
	resp, err = http.Get(oauth2Client.server.URL + "/login")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	defer resp.Body.Close()

	// try to refresh expired token with old refresh token.
	if _, err := oauth2Client.config.TokenSource(ctx, tok).Token(); err == nil {
		t.Errorf("Token refreshed with invalid refresh token, error expected.")
	}
}

// TestOAuth2DeviceFlow runs device flow integration tests against a test server
func TestOAuth2DeviceFlow(t *testing.T) {
	clientID := "testclient"
	clientSecret := ""
	requestedScopes := []string{oidc.ScopeOpenID, "email", "profile", "groups", "offline_access"}

	t0 := time.Now()

	// Always have the time function used by the server return the same time so
	// we can predict expected values of "expires_in" fields exactly.
	now := func() time.Time { return t0 }

	// Connector used by the tests.
	var conn *mock.Callback
	idTokensValidFor := time.Second * 30

	tests := makeOAuth2Tests(clientID, clientSecret, now)
	testCases := []struct {
		name          string
		tokenEndpoint string
		oauth2Tests   oauth2Tests
	}{
		{
			name:          "Actual token endpoint for devices",
			tokenEndpoint: "/token",
			oauth2Tests:   tests,
		},
		// TODO(nabokihms): delete temporary tests after removing the deprecated token endpoint support
		{
			name:          "Deprecated token endpoint for devices",
			tokenEndpoint: "/device/token",
			oauth2Tests:   tests,
		},
	}

	for _, testCase := range testCases {
		for _, tc := range testCase.oauth2Tests.tests {
			t.Run(tc.name, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// Setup a dex server.
				httpServer, s := newTestServer(ctx, t, func(c *Config) {
					c.Issuer += "/non-root-path"
					c.Now = now
					c.IDTokensValidFor = idTokensValidFor
				})
				defer httpServer.Close()

				mockConn := s.connectors["mock"]
				conn = mockConn.Connector.(*mock.Callback)

				p, err := oidc.NewProvider(ctx, httpServer.URL)
				if err != nil {
					t.Fatalf("failed to get provider: %v", err)
				}

				// Add the Clients to the test server
				client := storage.Client{
					ID:           clientID,
					RedirectURIs: []string{deviceCallbackURI},
					Public:       true,
				}
				if err := s.storage.CreateClient(client); err != nil {
					t.Fatalf("failed to create client: %v", err)
				}

				if err := s.storage.CreateRefresh(storage.RefreshToken{
					ID:       "existedrefrestoken",
					ClientID: "unexcistedclientid",
				}); err != nil {
					t.Fatalf("failed to create existed refresh token: %v", err)
				}

				// Grab the issuer that we'll reuse for the different endpoints to hit
				issuer, err := url.Parse(s.issuerURL.String())
				if err != nil {
					t.Errorf("Could not parse issuer URL %v", err)
				}

				// Send a new Device Request
				codeURL, _ := url.Parse(issuer.String())
				codeURL.Path = path.Join(codeURL.Path, "device/code")

				data := url.Values{}
				data.Set("client_id", clientID)
				data.Add("scope", strings.Join(requestedScopes, " "))
				resp, err := http.PostForm(codeURL.String(), data)
				if err != nil {
					t.Errorf("Could not request device code: %v", err)
				}
				defer resp.Body.Close()
				responseBody, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Errorf("Could read device code response %v", err)
				}
				if resp.StatusCode != http.StatusOK {
					t.Errorf("%v - Unexpected Response Type.  Expected 200 got  %v.  Response: %v", tc.name, resp.StatusCode, string(responseBody))
				}
				if resp.Header.Get("Cache-Control") != "no-store" {
					t.Errorf("Cache-Control header doesn't exist in Device Code Response")
				}

				// Parse the code response
				var deviceCode deviceCodeResponse
				if err := json.Unmarshal(responseBody, &deviceCode); err != nil {
					t.Errorf("Unexpected Device Code Response Format %v", string(responseBody))
				}

				// Mock the user hitting the verification URI and posting the form
				verifyURL, _ := url.Parse(issuer.String())
				verifyURL.Path = path.Join(verifyURL.Path, "/device/auth/verify_code")
				urlData := url.Values{}
				urlData.Set("user_code", deviceCode.UserCode)
				resp, err = http.PostForm(verifyURL.String(), urlData)
				if err != nil {
					t.Errorf("Error Posting Form: %v", err)
				}
				defer resp.Body.Close()
				responseBody, err = io.ReadAll(resp.Body)
				if err != nil {
					t.Errorf("Could read verification response %v", err)
				}
				if resp.StatusCode != http.StatusOK {
					t.Errorf("%v - Unexpected Response Type.  Expected 200 got  %v.  Response: %v", tc.name, resp.StatusCode, string(responseBody))
				}

				// Hit the Token Endpoint, and try and get an access token
				tokenURL, _ := url.Parse(issuer.String())
				tokenURL.Path = path.Join(tokenURL.Path, testCase.tokenEndpoint)
				v := url.Values{}
				v.Add("grant_type", grantTypeDeviceCode)
				v.Add("device_code", deviceCode.DeviceCode)
				resp, err = http.PostForm(tokenURL.String(), v)
				if err != nil {
					t.Errorf("Could not request device token: %v", err)
				}
				defer resp.Body.Close()
				responseBody, err = io.ReadAll(resp.Body)
				if err != nil {
					t.Errorf("Could read device token response %v", err)
				}
				if resp.StatusCode != http.StatusOK {
					t.Errorf("%v - Unexpected Token Response Type.  Expected 200 got  %v.  Response: %v", tc.name, resp.StatusCode, string(responseBody))
				}

				// Parse the response
				var tokenRes accessTokenResponse
				if err := json.Unmarshal(responseBody, &tokenRes); err != nil {
					t.Errorf("Unexpected Device Access Token Response Format %v", string(responseBody))
				}

				token := &oauth2.Token{
					AccessToken:  tokenRes.AccessToken,
					TokenType:    tokenRes.TokenType,
					RefreshToken: tokenRes.RefreshToken,
				}
				raw := make(map[string]interface{})
				json.Unmarshal(responseBody, &raw) // no error checks for optional fields
				token = token.WithExtra(raw)
				if secs := tokenRes.ExpiresIn; secs > 0 {
					token.Expiry = time.Now().Add(time.Duration(secs) * time.Second)
				}

				// Run token tests to validate info is correct
				// Create the OAuth2 config.
				oauth2Config := &oauth2.Config{
					ClientID:     client.ID,
					ClientSecret: client.Secret,
					Endpoint:     p.Endpoint(),
					Scopes:       requestedScopes,
					RedirectURL:  deviceCallbackURI,
				}
				if len(tc.scopes) != 0 {
					oauth2Config.Scopes = tc.scopes
				}
				err = tc.handleToken(ctx, p, oauth2Config, token, conn)
				if err != nil {
					t.Errorf("%s: %v", tc.name, err)
				}
			})
		}
	}
}

func TestServerSupportedGrants(t *testing.T) {
	tests := []struct {
		name      string
		config    func(c *Config)
		resGrants []string
	}{
		{
			name:      "Simple",
			config:    func(c *Config) {},
			resGrants: []string{grantTypeAuthorizationCode, grantTypeRefreshToken, grantTypeDeviceCode},
		},
		{
			name:      "With password connector",
			config:    func(c *Config) { c.PasswordConnector = "local" },
			resGrants: []string{grantTypeAuthorizationCode, grantTypePassword, grantTypeRefreshToken, grantTypeDeviceCode},
		},
		{
			name:      "With token response",
			config:    func(c *Config) { c.SupportedResponseTypes = append(c.SupportedResponseTypes, responseTypeToken) },
			resGrants: []string{grantTypeAuthorizationCode, grantTypeImplicit, grantTypeRefreshToken, grantTypeDeviceCode},
		},
		{
			name: "All",
			config: func(c *Config) {
				c.PasswordConnector = "local"
				c.SupportedResponseTypes = append(c.SupportedResponseTypes, responseTypeToken)
			},
			resGrants: []string{grantTypeAuthorizationCode, grantTypeImplicit, grantTypePassword, grantTypeRefreshToken, grantTypeDeviceCode},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, srv := newTestServer(context.TODO(), t, tc.config)
			require.Equal(t, srv.supportedGrantTypes, tc.resGrants)
		})
	}
}
