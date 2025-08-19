package keystone

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/dexidp/dex/connector"
)

// minimal token structures for responses
type testTokenUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type testToken struct {
	User testTokenUser `json:"user"`
}

type testTokenResponse struct {
	Token testToken `json:"token"`
}

type testUserResponse struct {
	User struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"user"`
}

func newTestFederationConnector(t *testing.T, cfg FederationConfig) *FederationConnector {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(testDiscard{}, nil))
	fc, err := NewFederationConnector(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create FederationConnector: %v", err)
	}
	return fc
}

// testDiscard implements io.Writer but discards output
type testDiscard struct{}

func (testDiscard) Write(p []byte) (int, error) { return len(p), nil }

func TestFederation_LoginURL(t *testing.T) {
	cases := []struct {
		name string
		host string
		path string
	}{
		{"no trailing/leading slash", "https://abc.com/keystone", "shib/login"},
		{"with trailing/leading slash", "https://abc.com/keystone/", "/shib/login"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := FederationConfig{
				Domain:              "default",
				Host:                tc.host,
				AdminUsername:       "admin",
				AdminPassword:       "pass",
				CustomerName:        "cust",
				ShibbolethLoginPath: tc.path,
				FederationAuthPath:  "/fed/auth",
				TimeoutSeconds:      5,
			}
			fc := newTestFederationConnector(t, cfg)
			callback := "https://dex/callback"
			state := "mystate"
			u, err := fc.LoginURL(connector.Scopes{}, callback, state)
			if err != nil {
				t.Fatalf("LoginURL error: %v", err)
			}

			parsed, err := url.Parse(u)
			if err != nil {
				t.Fatalf("parse result: %v", err)
			}
			// Expect path to be shib path at the root (host may have had trailing /keystone stripped)
			if got, want := parsed.Path, "/shib/login"; got != want {
				t.Fatalf("unexpected path: got %q want %q", got, want)
			}
			// target query must include callback and state
			target := parsed.Query().Get("target")
			if target == "" || target[:len(callback)] != callback {
				t.Fatalf("missing/invalid target query: %q", target)
			}
			if target != fmt.Sprintf("%s?state=%s", callback, state) {
				t.Fatalf("unexpected target: %q", target)
			}
		})
	}
}

func TestFederation_getKeystoneTokenFromFederation(t *testing.T) {
	// Test server that returns a token header on federation auth path
	fedPath := "/fed/auth"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fedPath {
			w.Header().Set("X-Subject-Token", "FED_TOKEN")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cfg := FederationConfig{
		Domain:              "default",
		Host:                ts.URL,
		AdminUsername:       "admin",
		AdminPassword:       "pass",
		CustomerName:        "cust",
		ShibbolethLoginPath: "/shib/login",
		FederationAuthPath:  fedPath,
		TimeoutSeconds:      5,
	}
	fc := newTestFederationConnector(t, cfg)

	r, _ := http.NewRequest(http.MethodGet, "https://dex/callback", nil)
	// Add cookies that should be forwarded (prefix is optional for this test)
	r.AddCookie(&http.Cookie{Name: "_shibsession_123", Value: "abc"})

	tok, err := fc.getKeystoneTokenFromFederation(r)
	if err != nil {
		t.Fatalf("getKeystoneTokenFromFederation error: %v", err)
	}
	if tok != "FED_TOKEN" {
		t.Fatalf("unexpected token: got %q want %q", tok, "FED_TOKEN")
	}
}

func TestFederation_HandleCallback_NoGroups(t *testing.T) {
	// Simulate keystone endpoints used in HandleCallback when Groups=false
	fedPath := "/fed/auth"
	userID := "u-123"
	userName := "user1"
	userEmail := "user1@example.com"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == fedPath:
			// federation auth returns subject token header
			w.Header().Set("X-Subject-Token", "FED_TOKEN")
			w.WriteHeader(http.StatusOK)
			return
		case r.Method == http.MethodPost && r.URL.Path == "/v3/auth/tokens":
			// admin token unscoped expects 201 and header
			w.Header().Set("X-Subject-Token", "ADMIN_TOKEN")
			w.WriteHeader(http.StatusCreated)
			return
		case r.Method == http.MethodGet && r.URL.Path == "/v3/auth/tokens":
			// getTokenInfo returns token info JSON
			resp := testTokenResponse{Token: testToken{User: testTokenUser{ID: userID, Name: userName}}}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
			return
		case r.Method == http.MethodGet && r.URL.Path == "/v3/users/"+userID:
			// getUser returns user details with email
			var ur testUserResponse
			ur.User.ID = userID
			ur.User.Name = userName
			ur.User.Email = userEmail
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(ur)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer ts.Close()

	cfg := FederationConfig{
		Domain:              "Default",
		Host:                ts.URL,
		AdminUsername:       "admin",
		AdminPassword:       "pass",
		CustomerName:        "cust",
		ShibbolethLoginPath: "/shib/login",
		FederationAuthPath:  fedPath,
		TimeoutSeconds:      5,
	}
	fc := newTestFederationConnector(t, cfg)

	r, _ := http.NewRequest(http.MethodGet, "https://dex/callback", nil)
	// Add a shibboleth cookie (optional)
	r.AddCookie(&http.Cookie{Name: "_shibsession_123", Value: "abc"})

	scopes := connector.Scopes{Groups: false}
	identity, err := fc.HandleCallback(scopes, r)
	if err != nil {
		t.Fatalf("HandleCallback error: %v", err)
	}

	if identity.UserID != userID {
		t.Fatalf("unexpected userID: got %q want %q", identity.UserID, userID)
	}
	if identity.Username != userName {
		t.Fatalf("unexpected username: got %q want %q", identity.Username, userName)
	}
	if identity.Email != userEmail || !identity.EmailVerified {
		t.Fatalf("unexpected email fields: email=%q verified=%v", identity.Email, identity.EmailVerified)
	}
	if len(identity.Groups) != 0 {
		t.Fatalf("expected no groups when Groups=false, got %v", identity.Groups)
	}
}
