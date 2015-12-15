package connector

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/coreos/go-oidc/oidc"
)

func TestLoginURL(t *testing.T) {
	lf := func(ident oidc.Identity, sessionKey string) (redirectURL string, err error) { return }

	tests := []struct {
		cid    string
		redir  string
		state  string
		scope  []string
		prompt string
		v      url.Values
	}{
		// Standard example
		{
			cid:    "fake-client-id",
			redir:  "http://example.com/oauth-redirect",
			state:  "fake-session-id",
			scope:  []string{"openid", "email", "profile"},
			prompt: "",
			v: url.Values{
				"response_type": {"code"},
				"state":         {"fake-session-id"},
				"redirect_uri":  {"http://example.com/oauth-redirect"},
				"scope":         {"openid email profile"},
				"client_id":     {"fake-client-id"},
			},
		},
		// No scope
		{
			cid:    "fake-client-id",
			redir:  "http://example.com/oauth-redirect",
			state:  "fake-session-id",
			scope:  []string{},
			prompt: "",
			v: url.Values{
				"response_type": {"code"},
				"state":         {"fake-session-id"},
				"redirect_uri":  {"http://example.com/oauth-redirect"},
				"scope":         {""},
				"client_id":     {"fake-client-id"},
			},
		},
		// No state
		{
			cid:    "fake-client-id",
			redir:  "http://example.com/oauth-redirect",
			state:  "",
			scope:  []string{},
			prompt: "",
			v: url.Values{
				"response_type": {"code"},
				"state":         {""},
				"redirect_uri":  {"http://example.com/oauth-redirect"},
				"scope":         {""},
				"client_id":     {"fake-client-id"},
			},
		},
		// Force prompt
		{
			cid:    "fake-client-id",
			redir:  "http://example.com/oauth-redirect",
			state:  "fake-session-id",
			scope:  []string{"openid", "email", "profile"},
			prompt: "select_account",
			v: url.Values{
				"response_type": {"code"},
				"prompt":        {"select_account"},
				"state":         {"fake-session-id"},
				"redirect_uri":  {"http://example.com/oauth-redirect"},
				"scope":         {"openid email profile"},
				"client_id":     {"fake-client-id"},
			},
		},
	}

	for i, tt := range tests {
		cfg := oidc.ClientConfig{
			Credentials: oidc.ClientCredentials{ID: tt.cid, Secret: "fake-client-secret"},
			RedirectURL: tt.redir,
			ProviderConfig: oidc.ProviderConfig{
				AuthEndpoint:  "http://example.com/authorize",
				TokenEndpoint: "http://example.com/token",
			},
			Scope: tt.scope,
		}
		cl, err := oidc.NewClient(cfg)
		if err != nil {
			t.Errorf("test: %d. unexpected error: %v", i, err)
		}
		cn := &OIDCConnector{
			loginFunc: lf,
			client:    cl,
		}

		lu, err := cn.LoginURL(tt.state, tt.prompt)
		if err != nil {
			t.Errorf("test: %d. want: no url error, got: error, error: %v", i, err)
		}

		u, err := url.Parse(lu)
		if err != nil {
			t.Errorf("test: %d. want: parsable url, got: unparsable url, error: %v", i, err)
		}

		got := u.Query()
		if !reflect.DeepEqual(tt.v, got) {
			t.Errorf("test: %d.\nwant: %v\ngot:  %v", i, tt.v, got)
		}
	}
}

func TestRedirectError(t *testing.T) {
	eu := url.URL{
		Scheme:   "http",
		Host:     "example.com:9090",
		Path:     "/login",
		RawQuery: "foo=bar",
	}
	q := url.Values{"ping": []string{"pong"}}
	rr := httptest.NewRecorder()
	redirectError(rr, eu, q)

	wantCode := http.StatusSeeOther
	if wantCode != rr.Code {
		t.Errorf("Incorrect code: want=%d got=%d", wantCode, rr.Code)
	}

	wantLoc := "http://example.com:9090/login?foo=bar&ping=pong"
	gotLoc := rr.HeaderMap.Get("Location")
	if wantLoc != gotLoc {
		t.Errorf("Incorrect Location header: want=%s got=%s", wantLoc, gotLoc)
	}
}

func TestParseEmailDomainSuccess(t *testing.T) {
	tests := []struct {
		email string
		want  string
	}{
		{"foo@example.com", "example.com"},
	}

	for i, tt := range tests {
		got, err := parseEmailDomain(tt.email)
		if err != nil {
			t.Errorf("case %d: unexpected error: %v", i, err)
			continue
		}
		if tt.want != got {
			t.Errorf("case %d: want=%v got=%v", i, tt.want, got)
		}
	}
}

func TestParseEmailDomainFailure(t *testing.T) {
	tests := []string{
		"example.com",
		"@example.com",
		"foo@",
	}

	for i, tt := range tests {
		_, err := parseEmailDomain(tt)
		if err == nil {
			t.Errorf("case %d: expected error, got nil", i)
		}
	}
}

func TestOIDCConnectorValidateRemoteIdentity(t *testing.T) {
	oc := &OIDCConnector{domain: "bar.example.com"}

	// valid email address
	ident := oidc.Identity{Email: "foo@bar.example.com"}
	ok, err := oc.validateRemoteIdentity(&ident)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if !ok {
		t.Fatal("expected ok")
	}

	// incorrect domain
	ident = oidc.Identity{Email: "foo@baz.example.com"}
	ok, err = oc.validateRemoteIdentity(&ident)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if ok {
		t.Fatal("expected not ok")
	}

	// invalid email address
	ident = oidc.Identity{Email: "pants"}
	ok, err = oc.validateRemoteIdentity(&ident)
	if err == nil {
		t.Fatal("expected error, got nil")
	} else if ok {
		t.Fatal("expected not ok")
	}
}

func TestOIDCConnectorValidateRemoteIdentityNoFilter(t *testing.T) {
	oc := &OIDCConnector{domain: ""}

	// valid email address
	ident := oidc.Identity{Email: "foo@bar.example.com"}
	ok, err := oc.validateRemoteIdentity(&ident)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if !ok {
		t.Fatal("expected ok")
	}

	// invalid email address
	ident = oidc.Identity{Email: "pants"}
	ok, err = oc.validateRemoteIdentity(&ident)
	if err == nil {
		t.Fatal("expected error, got nil")
	} else if ok {
		t.Fatal("expected not ok")
	}
}
