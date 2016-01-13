package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/session"
	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/oauth2"
	"github.com/coreos/go-oidc/oidc"
)

type fakeConnector struct {
	loginURL string
}

func (f *fakeConnector) ID() string {
	return "fake"
}

func (f *fakeConnector) Healthy() error {
	return nil
}

func (f *fakeConnector) LoginURL(sessionKey, prompt string) (string, error) {
	return f.loginURL, nil
}

func (f *fakeConnector) Register(mux *http.ServeMux, errorURL url.URL) {}

func (f *fakeConnector) Sync() chan struct{} {
	return nil
}

func (c *fakeConnector) TrustedEmailProvider() bool {
	return false
}

func TestHandleAuthFuncMethodNotAllowed(t *testing.T) {
	for _, m := range []string{"POST", "PUT", "DELETE"} {
		hdlr := handleAuthFunc(nil, nil, nil, true)
		req, err := http.NewRequest(m, "http://example.com", nil)
		if err != nil {
			t.Errorf("case %s: unable to create HTTP request: %v", m, err)
			continue
		}

		w := httptest.NewRecorder()
		hdlr.ServeHTTP(w, req)

		want := http.StatusMethodNotAllowed
		got := w.Code
		if want != got {
			t.Errorf("case %s: expected HTTP %d, got %d", m, want, got)
		}
	}
}

func TestHandleAuthFuncResponsesSingleRedirectURL(t *testing.T) {
	idpcs := []connector.Connector{
		&fakeConnector{loginURL: "http://fake.example.com"},
	}
	srv := &Server{
		IssuerURL:      url.URL{Scheme: "http", Host: "server.example.com"},
		SessionManager: session.NewSessionManager(session.NewSessionRepo(), session.NewSessionKeyRepo()),
		ClientIdentityRepo: client.NewClientIdentityRepo([]oidc.ClientIdentity{
			oidc.ClientIdentity{
				Credentials: oidc.ClientCredentials{
					ID:     "XXX",
					Secret: "secrete",
				},
				Metadata: oidc.ClientMetadata{
					RedirectURIs: []url.URL{
						url.URL{Scheme: "http", Host: "client.example.com", Path: "/callback"},
					},
				},
			},
		}),
	}

	tests := []struct {
		query        url.Values
		wantCode     int
		wantLocation string
	}{
		// no redirect_uri provided, but client only has one, so it's usable
		{
			query: url.Values{
				"response_type": []string{"code"},
				"client_id":     []string{"XXX"},
				"connector_id":  []string{"fake"},
				"scope":         []string{"openid"},
			},
			wantCode:     http.StatusTemporaryRedirect,
			wantLocation: "http://fake.example.com",
		},

		// provided redirect_uri matches client
		{
			query: url.Values{
				"response_type": []string{"code"},
				"redirect_uri":  []string{"http://client.example.com/callback"},
				"client_id":     []string{"XXX"},
				"connector_id":  []string{"fake"},
				"scope":         []string{"openid"},
			},
			wantCode:     http.StatusTemporaryRedirect,
			wantLocation: "http://fake.example.com",
		},

		// provided redirect_uri does not match client
		{
			query: url.Values{
				"response_type": []string{"code"},
				"redirect_uri":  []string{"http://unrecognized.example.com/callback"},
				"client_id":     []string{"XXX"},
				"connector_id":  []string{"fake"},
				"scope":         []string{"openid"},
			},
			wantCode: http.StatusBadRequest,
		},

		// nonexistant client_id
		{
			query: url.Values{
				"response_type": []string{"code"},
				"redirect_uri":  []string{"http://client.example.com/callback"},
				"client_id":     []string{"YYY"},
				"connector_id":  []string{"fake"},
				"scope":         []string{"openid"},
			},
			wantCode: http.StatusBadRequest,
		},

		// unsupported response type, redirects back to client
		{
			query: url.Values{
				"response_type": []string{"token"},
				"client_id":     []string{"XXX"},
				"connector_id":  []string{"fake"},
				"scope":         []string{"openid"},
			},
			wantCode:     http.StatusTemporaryRedirect,
			wantLocation: "http://client.example.com/callback?error=unsupported_response_type&state=",
		},

		// no 'openid' in scope
		{
			query: url.Values{
				"response_type": []string{"code"},
				"redirect_uri":  []string{"http://client.example.com/callback"},
				"client_id":     []string{"XXX"},
				"connector_id":  []string{"fake"},
			},
			wantCode: http.StatusBadRequest,
		},
	}

	for i, tt := range tests {
		hdlr := handleAuthFunc(srv, idpcs, nil, true)
		w := httptest.NewRecorder()
		u := fmt.Sprintf("http://server.example.com?%s", tt.query.Encode())
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			t.Errorf("case %d: unable to form HTTP request: %v", i, err)
			continue
		}

		hdlr.ServeHTTP(w, req)
		if tt.wantCode != w.Code {
			t.Errorf("case %d: HTTP code mismatch: want=%d got=%d", i, tt.wantCode, w.Code)
			continue
		}

		gotLocation := w.Header().Get("Location")
		if tt.wantLocation != gotLocation {
			t.Errorf("case %d: HTTP Location header mismatch: want=%s got=%s", i, tt.wantLocation, gotLocation)
		}
	}
}

func TestHandleAuthFuncResponsesMultipleRedirectURLs(t *testing.T) {
	idpcs := []connector.Connector{
		&fakeConnector{loginURL: "http://fake.example.com"},
	}
	srv := &Server{
		IssuerURL:      url.URL{Scheme: "http", Host: "server.example.com"},
		SessionManager: session.NewSessionManager(session.NewSessionRepo(), session.NewSessionKeyRepo()),
		ClientIdentityRepo: client.NewClientIdentityRepo([]oidc.ClientIdentity{
			oidc.ClientIdentity{
				Credentials: oidc.ClientCredentials{
					ID:     "XXX",
					Secret: "secrete",
				},
				Metadata: oidc.ClientMetadata{
					RedirectURIs: []url.URL{
						url.URL{Scheme: "http", Host: "foo.example.com", Path: "/callback"},
						url.URL{Scheme: "http", Host: "bar.example.com", Path: "/callback"},
					},
				},
			},
		}),
	}

	tests := []struct {
		query        url.Values
		wantCode     int
		wantLocation string
	}{
		// provided redirect_uri matches client's first
		{
			query: url.Values{
				"response_type": []string{"code"},
				"redirect_uri":  []string{"http://foo.example.com/callback"},
				"client_id":     []string{"XXX"},
				"connector_id":  []string{"fake"},
				"scope":         []string{"openid"},
			},
			wantCode:     http.StatusTemporaryRedirect,
			wantLocation: "http://fake.example.com",
		},

		// provided redirect_uri matches client's second
		{
			query: url.Values{
				"response_type": []string{"code"},
				"redirect_uri":  []string{"http://bar.example.com/callback"},
				"client_id":     []string{"XXX"},
				"connector_id":  []string{"fake"},
				"scope":         []string{"openid"},
			},
			wantCode:     http.StatusTemporaryRedirect,
			wantLocation: "http://fake.example.com",
		},

		// provided redirect_uri does not match either of client's
		{
			query: url.Values{
				"response_type": []string{"code"},
				"redirect_uri":  []string{"http://unrecognized.example.com/callback"},
				"client_id":     []string{"XXX"},
				"connector_id":  []string{"fake"},
				"scope":         []string{"openid"},
			},
			wantCode: http.StatusBadRequest,
		},

		// no redirect_uri provided
		{
			query: url.Values{
				"response_type": []string{"code"},
				"client_id":     []string{"XXX"},
				"connector_id":  []string{"fake"},
				"scope":         []string{"openid"},
			},
			wantCode: http.StatusBadRequest,
		},
	}

	for i, tt := range tests {
		hdlr := handleAuthFunc(srv, idpcs, nil, true)
		w := httptest.NewRecorder()
		u := fmt.Sprintf("http://server.example.com?%s", tt.query.Encode())
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			t.Errorf("case %d: unable to form HTTP request: %v", i, err)
			continue
		}

		hdlr.ServeHTTP(w, req)
		if tt.wantCode != w.Code {
			t.Errorf("case %d: HTTP code mismatch: want=%d got=%d", i, tt.wantCode, w.Code)
			t.Errorf("case %d: BODY: %v", i, w.Body.String())
			t.Errorf("case %d: LOCO: %v", i, w.HeaderMap.Get("Location"))
			continue
		}

		gotLocation := w.Header().Get("Location")
		if tt.wantLocation != gotLocation {
			t.Errorf("case %d: HTTP Location header mismatch: want=%s got=%s", i, tt.wantLocation, gotLocation)
		}
	}
}

func TestHandleTokenFuncMethodNotAllowed(t *testing.T) {
	for _, m := range []string{"GET", "PUT", "DELETE"} {
		hdlr := handleTokenFunc(nil)
		req, err := http.NewRequest(m, "http://example.com", nil)
		if err != nil {
			t.Errorf("case %s: unable to create HTTP request: %v", m, err)
			continue
		}

		w := httptest.NewRecorder()
		hdlr.ServeHTTP(w, req)

		want := http.StatusMethodNotAllowed
		got := w.Code
		if want != got {
			t.Errorf("case %s: expected HTTP %d, got %d", m, want, got)
		}
	}
}

func TestHandleTokenFuncState(t *testing.T) {
	want := "test-state"
	v := url.Values{
		"state": {want},
	}
	hdlr := handleTokenFunc(nil)
	req, err := http.NewRequest("POST", "http://example.com", strings.NewReader(v.Encode()))
	if err != nil {
		t.Errorf("unable to create HTTP request, error=%v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	hdlr.ServeHTTP(w, req)

	// should have errored and returned state in the response body
	var resp map[string]string
	if err = json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("error unmarshaling response, error=%v", err)
	}

	got := resp["state"]
	if want != got {
		t.Errorf("unexpected state, want=%v, got=%v", want, got)
	}
}

func TestHandleDiscoveryFuncMethodNotAllowed(t *testing.T) {
	for _, m := range []string{"POST", "PUT", "DELETE"} {
		hdlr := handleDiscoveryFunc(oidc.ProviderConfig{})
		req, err := http.NewRequest(m, "http://example.com", nil)
		if err != nil {
			t.Errorf("case %s: unable to create HTTP request: %v", m, err)
			continue
		}

		w := httptest.NewRecorder()
		hdlr.ServeHTTP(w, req)

		want := http.StatusMethodNotAllowed
		got := w.Code
		if want != got {
			t.Errorf("case %s: expected HTTP %d, got %d", m, want, got)
		}
	}
}

func TestHandleDiscoveryFunc(t *testing.T) {
	u := url.URL{Scheme: "http", Host: "server.example.com"}
	pathURL := func(path string) *url.URL {
		ucopy := u
		ucopy.Path = path
		return &ucopy
	}
	cfg := oidc.ProviderConfig{
		Issuer:        &u,
		AuthEndpoint:  pathURL(httpPathAuth),
		TokenEndpoint: pathURL(httpPathToken),
		KeysEndpoint:  pathURL(httpPathKeys),

		GrantTypesSupported:               []string{oauth2.GrantTypeAuthCode},
		ResponseTypesSupported:            []string{"code"},
		SubjectTypesSupported:             []string{"public"},
		IDTokenSigningAlgValues:           []string{"RS256"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic"},
	}

	req, err := http.NewRequest("GET", "http://server.example.com", nil)
	if err != nil {
		t.Fatalf("Failed creating HTTP request: err=%v", err)
	}

	w := httptest.NewRecorder()
	hdlr := handleDiscoveryFunc(cfg)
	hdlr.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Incorrect status code: want=200 got=%d", w.Code)
	}

	h := w.Header()

	if ct := h.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Incorrect Content-Type: want=application/json, got %s", ct)
	}

	gotCC := h.Get("Cache-Control")
	wantCC := "public, max-age=86400"
	if wantCC != gotCC {
		t.Fatalf("Incorrect Cache-Control header: want=%q, got=%q", wantCC, gotCC)
	}

	wantBody := `{"issuer":"http://server.example.com","authorization_endpoint":"http://server.example.com/auth","token_endpoint":"http://server.example.com/token","jwks_uri":"http://server.example.com/keys","response_types_supported":["code"],"grant_types_supported":["authorization_code"],"subject_types_supported":["public"],"id_token_signing_alg_values_supported":["RS256"],"token_endpoint_auth_methods_supported":["client_secret_basic"]}`
	gotBody := w.Body.String()
	if wantBody != gotBody {
		t.Fatalf("Incorrect body: want=%s got=%s", wantBody, gotBody)
	}
}

func TestHandleKeysFuncMethodNotAllowed(t *testing.T) {
	for _, m := range []string{"POST", "PUT", "DELETE"} {
		hdlr := handleKeysFunc(nil, clockwork.NewRealClock())
		req, err := http.NewRequest(m, "http://example.com", nil)
		if err != nil {
			t.Errorf("case %s: unable to create HTTP request: %v", m, err)
			continue
		}

		w := httptest.NewRecorder()
		hdlr.ServeHTTP(w, req)

		want := http.StatusMethodNotAllowed
		got := w.Code
		if want != got {
			t.Errorf("case %s: expected HTTP %d, got %d", m, want, got)
		}
	}
}

func TestHandleKeysFunc(t *testing.T) {
	fc := clockwork.NewFakeClock()
	exp := fc.Now().Add(13 * time.Second)
	km := &StaticKeyManager{
		expiresAt: exp,
		keys: []jose.JWK{
			jose.JWK{
				ID:       "1234",
				Type:     "RSA",
				Alg:      "RS256",
				Use:      "sig",
				Exponent: 65537,
				Modulus:  big.NewInt(int64(5716758339926702)),
			},
			jose.JWK{
				ID:       "5678",
				Type:     "RSA",
				Alg:      "RS256",
				Use:      "sig",
				Exponent: 65537,
				Modulus:  big.NewInt(int64(1234294715519622)),
			},
		},
	}

	req, err := http.NewRequest("GET", "http://server.example.com", nil)
	if err != nil {
		t.Fatalf("Failed creating HTTP request: err=%v", err)
	}

	w := httptest.NewRecorder()
	hdlr := handleKeysFunc(km, fc)
	hdlr.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Incorrect status code: want=200 got=%d", w.Code)
	}

	wantHeader := http.Header{
		"Content-Type":  []string{"application/json"},
		"Cache-Control": []string{"public, max-age=13"},
		"Expires":       []string{exp.Format(time.RFC1123)},
	}
	gotHeader := w.Header()
	if !reflect.DeepEqual(wantHeader, gotHeader) {
		t.Fatalf("Incorrect headers: want=%#v got=%#v", wantHeader, gotHeader)
	}

	wantBody := `{"keys":[{"kid":"1234","kty":"RSA","alg":"RS256","use":"sig","e":"AQAB","n":"FE9chh46rg=="},{"kid":"5678","kty":"RSA","alg":"RS256","use":"sig","e":"AQAB","n":"BGKVohEShg=="}]}`
	gotBody := w.Body.String()
	if wantBody != gotBody {
		t.Fatalf("Incorrect body: want=%s got=%s", wantBody, gotBody)
	}
}

func TestShouldReprompt(t *testing.T) {
	tests := []struct {
		c *http.Cookie
		v bool
	}{
		// No cookie
		{
			c: nil,
			v: false,
		},
		// different cookie
		{
			c: &http.Cookie{
				Name: "rando-cookie",
			},
			v: false,
		},
		// actual cookie we care about
		{
			c: &http.Cookie{
				Name: "LastSeen",
			},
			v: true,
		},
	}

	for i, tt := range tests {
		r := &http.Request{Header: make(http.Header)}
		if tt.c != nil {
			r.AddCookie(tt.c)
		}
		want := tt.v
		got := shouldReprompt(r)
		if want != got {
			t.Errorf("case %d: want=%t, got=%t", i, want, got)
		}
	}
}

type checkable struct {
	healthy bool
}

func (c checkable) Healthy() (err error) {
	if !c.healthy {
		err = errors.New("im unhealthy")
	}
	return
}
