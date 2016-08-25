package server

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ericchiang/oidc"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/coreos/dex/connector/mock"
	"github.com/coreos/dex/storage"
	"github.com/coreos/dex/storage/memory"
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

func newTestServer(updateConfig func(c *Config)) (*httptest.Server, *Server) {
	var server *Server
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.ServeHTTP(w, r)
	}))
	config := Config{
		Issuer:  s.URL,
		Storage: memory.New(),
		Connectors: []Connector{
			{
				ID:          "mock",
				DisplayName: "Mock",
				Connector:   mock.New(),
			},
		},
	}
	if updateConfig != nil {
		updateConfig(&config)
	}
	s.URL = config.Issuer

	var err error
	if server, err = newServer(config, staticRotationStrategy(testKey)); err != nil {
		panic(err)
	}
	server.skipApproval = true // Don't prompt for approval, just immediately redirect with code.
	return s, server
}

func TestNewTestServer(t *testing.T) {
	newTestServer(nil)
}

func TestDiscovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpServer, _ := newTestServer(func(c *Config) {
		c.Issuer = c.Issuer + "/non-root-path"
	})
	defer httpServer.Close()

	p, err := oidc.NewProvider(ctx, httpServer.URL)
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}
	required := []struct {
		name, val string
	}{
		{"issuer", p.Issuer},
		{"authorization_endpoint", p.AuthURL},
		{"token_endpoint", p.TokenURL},
		{"jwks_uri", p.JWKSURL},
	}
	for _, field := range required {
		if field.val == "" {
			t.Errorf("server discovery is missing required field %q", field.name)
		}
	}
}

func TestOAuth2CodeFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpServer, s := newTestServer(func(c *Config) {
		c.Issuer = c.Issuer + "/non-root-path"
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
				idToken, ok := token.Extra("id_token").(string)
				if !ok {
					t.Errorf("no id token found: %v", err)
					return
				}
				// TODO(ericchiang): validate id token
				_ = idToken

				token.Expiry = time.Now().Add(time.Second * -10)
				if token.Valid() {
					t.Errorf("token shouldn't be valid")
				}

				newToken, err := oauth2Config.TokenSource(ctx, token).Token()
				if err != nil {
					t.Errorf("failed to refresh token: %v", err)
					return
				}
				if token.RefreshToken == newToken.RefreshToken {
					t.Errorf("old refresh token was the same as the new token %q", token.RefreshToken)
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
		ID:           "testclient",
		Secret:       "testclientsecret",
		RedirectURIs: []string{redirectURL},
	}
	if err := s.storage.CreateClient(client); err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	oauth2Config = &oauth2.Config{
		ClientID:     client.ID,
		ClientSecret: client.Secret,
		Endpoint:     p.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "offline_access"},
		RedirectURL:  redirectURL,
	}

	resp, err := http.Get(oauth2Server.URL + "/login")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if reqDump, err = httputil.DumpRequest(resp.Request, false); err != nil {
		t.Fatal(err)
	}
	if respDump, err = httputil.DumpResponse(resp, true); err != nil {
		t.Fatal(err)
	}
}

type nonceSource struct {
	nonce string
	once  sync.Once
}

func (n *nonceSource) ClaimNonce(nonce string) error {
	if n.nonce != nonce {
		return errors.New("invalid nonce")
	}
	ok := false
	n.once.Do(func() { ok = true })
	if !ok {
		return errors.New("invalid nonce")
	}
	return nil
}

func TestOAuth2ImplicitFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpServer, s := newTestServer(func(c *Config) {
		// Enable support for the implicit flow.
		c.SupportedResponseTypes = []string{"code", "token"}
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
		u := oauth2Config.AuthCodeURL(state, oauth2.SetAuthURLParam("response_type", "token"), oidc.Nonce(nonce))
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

	src := &nonceSource{nonce: nonce}

	idTokenVerifier := p.NewVerifier(ctx, oidc.VerifyAudience(client.ID), oidc.VerifyNonce(src))

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
		idToken := v.Get("id_token")
		if idToken == "" {
			return errors.New("no id_token in fragment")
		}
		if _, err := idTokenVerifier.Verify(idToken); err != nil {
			return fmt.Errorf("failed to verify id_token: %v", err)
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
	if reqDump, err = httputil.DumpRequest(resp.Request, false); err != nil {
		t.Fatal(err)
	}
	if respDump, err = httputil.DumpResponse(resp, true); err != nil {
		t.Fatal(err)
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

	s := memory.New()

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
