package ldap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ldaplib "github.com/go-ldap/ldap/v3"
	"github.com/jcmturner/goidentity/v6"
	"github.com/jcmturner/gokrb5/v8/credentials"

	"github.com/dexidp/dex/connector"
)

// fakeAuthenticate returns a krbState whose authenticate middleware immediately
// invokes inner with the given credentials attached to the request context
// under goidentity.CTXKey — mimicking a successful SPNEGO handshake.
func fakeAuthenticate(creds *credentials.Credentials) *krbState {
	return &krbState{
		authenticate: func(inner http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if creds == nil {
					// Emulate a failed SPNEGO exchange: middleware writes 401
					// without invoking the inner handler.
					w.Header().Set("WWW-Authenticate", "Negotiate")
					http.Error(w, "auth failed\n", http.StatusUnauthorized)
					return
				}
				rr := r.WithContext(context.WithValue(r.Context(), goidentity.CTXKey, goidentity.Identity(creds)))
				inner.ServeHTTP(w, rr)
			})
		},
	}
}

// fakeAuthenticateWithResponse returns a krbState whose middleware writes an
// arbitrary response and does not invoke inner. Useful for testing the
// forward-vs-discard branch on failed SPNEGO.
func fakeAuthenticateWithResponse(status int, headers map[string]string, body string) *krbState {
	return &krbState{
		authenticate: func(_ http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				for k, v := range headers {
					w.Header().Set(k, v)
				}
				w.WriteHeader(status)
				_, _ = w.Write([]byte(body))
			})
		},
	}
}

func newCreds(username, realm string) *credentials.Credentials {
	c := credentials.New(username, realm)
	c.SetAuthenticated(true)
	return c
}

func TestKerberos_NoHeader_NoFallback_ForwardsMiddleware401(t *testing.T) {
	lc := &ldapConnector{
		logger:  slog.Default(),
		krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "localpart"},
		krb:     fakeAuthenticate(nil),
	}
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !bool(handled) {
		t.Fatalf("expected handled")
	}
	if ident != nil {
		t.Fatalf("expected no identity")
	}
	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Result().StatusCode)
	}
	if hdr := w.Header().Get("WWW-Authenticate"); hdr != "Negotiate" {
		t.Fatalf("expected bare Negotiate challenge, got %q", hdr)
	}
}

func TestKerberos_NoHeader_Fallback_NoCookie_ChallengesAndSetsCookie(t *testing.T) {
	// First contact under fallback: no Authorization header, no probe cookie.
	// We expect the middleware to run, its 401 Negotiate to be forwarded to
	// the client, and a probe cookie to be set so the *next* round (still
	// without an Authorization header) can short-circuit to the form.
	middlewareRan := false
	lc := &ldapConnector{
		logger:  slog.Default(),
		krbConf: kerberosConfig{FallbackToPassword: true, UsernameFromPrincipal: "localpart"},
		krb: &krbState{authenticate: func(_ http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				middlewareRan = true
				w.Header().Set("WWW-Authenticate", "Negotiate")
				w.WriteHeader(http.StatusUnauthorized)
			})
		}},
	}
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !bool(handled) {
		t.Fatalf("expected handled (probe round forwards 401)")
	}
	if ident != nil {
		t.Fatalf("expected no identity")
	}
	if !middlewareRan {
		t.Fatalf("SPNEGO middleware should run on the probe round")
	}
	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Result().StatusCode)
	}
	if hdr := w.Header().Get("WWW-Authenticate"); hdr != "Negotiate" {
		t.Fatalf("expected Negotiate challenge, got %q", hdr)
	}
	if !setCookieHasName(w, spnegoProbeCookieName) {
		t.Fatalf("expected probe cookie %q to be set, got %q", spnegoProbeCookieName, w.Header().Values("Set-Cookie"))
	}
}

func TestKerberos_NoHeader_Fallback_WithCookie_RendersForm(t *testing.T) {
	// Follow-up round: probe cookie is already in the request and the client
	// still hasn't sent a Negotiate token. Treat that as "client cannot/will
	// not SPNEGO" and short-circuit to the password form without running the
	// middleware (otherwise w would get tainted with another 401).
	middlewareRan := false
	lc := &ldapConnector{
		logger:  slog.Default(),
		krbConf: kerberosConfig{FallbackToPassword: true, UsernameFromPrincipal: "localpart"},
		krb: &krbState{authenticate: func(_ http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				middlewareRan = true
				w.WriteHeader(http.StatusUnauthorized)
			})
		}},
	}
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	r.AddCookie(&http.Cookie{Name: spnegoProbeCookieName, Value: "1"})
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if bool(handled) {
		t.Fatalf("expected not handled (caller should render form)")
	}
	if ident != nil {
		t.Fatalf("expected no identity")
	}
	if middlewareRan {
		t.Fatalf("SPNEGO middleware should not run on probe-follow-up round")
	}
	if w.Code != 200 && w.Code != 0 {
		t.Fatalf("expected w untouched, got status=%d", w.Code)
	}
}

func TestKerberos_MiddlewareFails_Fallback_DiscardsResponse(t *testing.T) {
	// Authorization header is present (so we enter middleware), middleware
	// rejects the ticket, fallback is on: TrySPNEGO must return (nil, false, nil)
	// and must not forward the middleware's 401 to the real ResponseWriter.
	lc := &ldapConnector{
		logger:  slog.Default(),
		krbConf: kerberosConfig{FallbackToPassword: true, UsernameFromPrincipal: "localpart"},
		krb: fakeAuthenticateWithResponse(
			http.StatusUnauthorized,
			map[string]string{"WWW-Authenticate": "Negotiate oQcwBaADCgEC"},
			"auth failed\n",
		),
	}
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	r.Header.Set("Authorization", "Negotiate deadbeef")
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if bool(handled) {
		t.Fatalf("expected not handled (caller should render form)")
	}
	if ident != nil {
		t.Fatalf("expected no identity")
	}
	if w.Code != 200 && w.Code != 0 {
		t.Fatalf("expected w untouched, got status=%d", w.Code)
	}
	if hdr := w.Header().Get("WWW-Authenticate"); hdr != "" {
		t.Fatalf("expected no WWW-Authenticate on fallback, got %q", hdr)
	}
}

func TestKerberos_MiddlewareFails_NoFallback_ForwardsResponse(t *testing.T) {
	// Middleware writes a reject token; no fallback: forward to client verbatim.
	lc := &ldapConnector{
		logger:  slog.Default(),
		krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "localpart"},
		krb: fakeAuthenticateWithResponse(
			http.StatusUnauthorized,
			map[string]string{"WWW-Authenticate": "Negotiate oQcwBaADCgEC"},
			"auth failed\n",
		),
	}
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	r.Header.Set("Authorization", "Negotiate deadbeef")
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !bool(handled) {
		t.Fatalf("expected handled")
	}
	if ident != nil {
		t.Fatalf("expected no identity")
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 forwarded, got %d", w.Code)
	}
	if hdr := w.Header().Get("WWW-Authenticate"); !strings.HasPrefix(hdr, "Negotiate ") {
		t.Fatalf("expected reject token forwarded, got %q", hdr)
	}
}

func TestKerberos_Success_BuildsIdentity(t *testing.T) {
	lc := &ldapConnector{
		logger:  slog.Default(),
		krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "localpart"},
		krb:     fakeAuthenticate(newCreds("jdoe", "EXAMPLE.COM")),
	}
	lc.Config.UserSearch.IDAttr = "uid"
	lc.Config.UserSearch.EmailAttr = "mail"
	lc.Config.UserSearch.NameAttr = "cn"
	lc.krbLookupUserHook = func(_ context.Context, c *ldapConnector, username string) (ldaplib.Entry, error) {
		if username != "jdoe" {
			t.Fatalf("expected username=jdoe, got %q", username)
		}
		e := ldaplib.NewEntry("cn=jdoe,dc=example,dc=org", map[string][]string{
			c.UserSearch.IDAttr:    {"uid-jdoe"},
			c.UserSearch.EmailAttr: {"jdoe@example.com"},
			c.UserSearch.NameAttr:  {"John Doe"},
		})
		return *e, nil
	}
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !bool(handled) || ident == nil {
		t.Fatalf("expected handled with identity")
	}
	if ident.Username == "" || ident.Email == "" || ident.UserID == "" {
		t.Fatalf("expected populated identity, got %+v", *ident)
	}
	// No probe cookie was on the request, so the success path must not emit
	// a clearing Set-Cookie either (avoids needless header noise).
	if setCookieHasName(w, spnegoProbeCookieName) {
		t.Fatalf("did not expect Set-Cookie %q on success without prior probe cookie, got %q",
			spnegoProbeCookieName, w.Header().Values("Set-Cookie"))
	}
}

// TestKerberos_Success_ClearsProbeCookie verifies that a successful SPNEGO
// login on a request that *did* carry a probe cookie clears it, so a future
// no-Authorization GET starts a fresh negotiation round instead of going
// straight to the password form.
func TestKerberos_Success_ClearsProbeCookie(t *testing.T) {
	lc := &ldapConnector{
		logger:  slog.Default(),
		krbConf: kerberosConfig{FallbackToPassword: true, UsernameFromPrincipal: "localpart"},
		krb:     fakeAuthenticate(newCreds("jdoe", "EXAMPLE.COM")),
	}
	lc.Config.UserSearch.IDAttr = "uid"
	lc.Config.UserSearch.EmailAttr = "mail"
	lc.Config.UserSearch.NameAttr = "cn"
	lc.krbLookupUserHook = func(_ context.Context, c *ldapConnector, _ string) (ldaplib.Entry, error) {
		e := ldaplib.NewEntry("cn=jdoe,dc=example,dc=org", map[string][]string{
			c.UserSearch.IDAttr:    {"uid-jdoe"},
			c.UserSearch.EmailAttr: {"jdoe@example.com"},
			c.UserSearch.NameAttr:  {"John Doe"},
		})
		return *e, nil
	}
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	r.Header.Set("Authorization", "Negotiate deadbeef")
	r.AddCookie(&http.Cookie{Name: spnegoProbeCookieName, Value: "1"})
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !bool(handled) || ident == nil {
		t.Fatalf("expected handled with identity")
	}
	if !setCookieClearsName(w, spnegoProbeCookieName) {
		t.Fatalf("expected probe cookie %q to be cleared, got %q",
			spnegoProbeCookieName, w.Header().Values("Set-Cookie"))
	}
}

// setCookieHasName reports whether any Set-Cookie response header sets a
// cookie with the given name and a non-empty value (i.e. an actual set, not
// a clear). Order-tolerant; useful for assertions that don't care about
// exact attributes.
func setCookieHasName(w *httptest.ResponseRecorder, name string) bool {
	for _, sc := range w.Result().Cookies() {
		if sc.Name == name && sc.Value != "" && sc.MaxAge >= 0 {
			return true
		}
	}
	return false
}

// setCookieClearsName reports whether any Set-Cookie response header clears
// the cookie with the given name (Max-Age<=0 or empty value).
func setCookieClearsName(w *httptest.ResponseRecorder, name string) bool {
	for _, sc := range w.Result().Cookies() {
		if sc.Name == name && (sc.MaxAge < 0 || sc.Value == "") {
			return true
		}
	}
	return false
}

// TestKerberos_UserNotFound_ReturnsError pins the behavior when the LDAP
// directory has no entry for an authenticated Kerberos principal. The hook
// is authoritative, so the test is hermetic (no network call to :636).
func TestKerberos_UserNotFound_ReturnsError(t *testing.T) {
	lc := &ldapConnector{
		logger:  slog.Default(),
		krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "localpart"},
		krb:     fakeAuthenticate(newCreds("jdoe", "EXAMPLE.COM")),
	}
	lc.krbLookupUserHook = func(_ context.Context, _ *ldapConnector, _ string) (ldaplib.Entry, error) {
		return ldaplib.Entry{}, fmt.Errorf("user not found for principal")
	}
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	_, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err == nil {
		t.Fatalf("expected error for user not found")
	}
	if !bool(handled) {
		t.Fatalf("expected handled")
	}
	if !strings.Contains(err.Error(), "user lookup failed") {
		t.Fatalf("expected 'user lookup failed' error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "user not found for principal") {
		t.Fatalf("expected wrapped hook error, got: %v", err)
	}
}

// TestKerberos_LookupHookError_Wrapped verifies hook errors propagate through
// TrySPNEGO with %w wrapping so callers can errors.Is against the original.
func TestKerberos_LookupHookError_Wrapped(t *testing.T) {
	sentinel := errors.New("boom: LDAP server tantrum")
	lc := &ldapConnector{
		logger:  slog.Default(),
		krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "localpart"},
		krb:     fakeAuthenticate(newCreds("jdoe", "EXAMPLE.COM")),
	}
	lc.krbLookupUserHook = func(_ context.Context, _ *ldapConnector, _ string) (ldaplib.Entry, error) {
		return ldaplib.Entry{}, sentinel
	}
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	_, _, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected errors.Is(err, sentinel), got: %v", err)
	}
}

func TestKerberos_ExpectedRealmMismatch_NoFallback_401(t *testing.T) {
	lc := &ldapConnector{
		logger: slog.Default(),
		krbConf: kerberosConfig{
			FallbackToPassword:    false,
			UsernameFromPrincipal: "localpart",
			ExpectedRealm:         "EXAMPLE.COM",
		},
		krb: fakeAuthenticate(newCreds("jdoe", "OTHER.COM")),
	}
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !bool(handled) || ident != nil {
		t.Fatalf("expected handled with no identity")
	}
	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Result().StatusCode)
	}
	if hdr := w.Header().Get("WWW-Authenticate"); hdr != "Negotiate" {
		t.Fatalf("expected bare Negotiate, got %q", hdr)
	}
}

func TestKerberos_ExpectedRealmMismatch_Fallback_RendersForm(t *testing.T) {
	lc := &ldapConnector{
		logger: slog.Default(),
		krbConf: kerberosConfig{
			FallbackToPassword:    true,
			UsernameFromPrincipal: "localpart",
			ExpectedRealm:         "EXAMPLE.COM",
		},
		krb: fakeAuthenticate(newCreds("jdoe", "OTHER.COM")),
	}
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	r.Header.Set("Authorization", "Negotiate deadbeef")
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if bool(handled) || ident != nil {
		t.Fatalf("expected not handled, got handled=%v ident=%v", handled, ident)
	}
}

func TestKerberos_ExpectedRealm_CaseInsensitive(t *testing.T) {
	lc := &ldapConnector{
		logger: slog.Default(),
		krbConf: kerberosConfig{
			FallbackToPassword:    false,
			UsernameFromPrincipal: "localpart",
			ExpectedRealm:         "ExAmPlE.CoM",
		},
		krb: fakeAuthenticate(newCreds("user", "EXAMPLE.COM")),
	}
	lc.Config.UserSearch.IDAttr = "uid"
	lc.Config.UserSearch.EmailAttr = "mail"
	lc.Config.UserSearch.NameAttr = "cn"
	lc.krbLookupUserHook = func(_ context.Context, c *ldapConnector, _ string) (ldaplib.Entry, error) {
		e := ldaplib.NewEntry("cn=user,dc=example,dc=com", map[string][]string{
			c.UserSearch.IDAttr:    {"uid-user"},
			c.UserSearch.EmailAttr: {"user@example.com"},
			c.UserSearch.NameAttr:  {"User"},
		})
		return *e, nil
	}
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !bool(handled) || ident == nil {
		t.Fatalf("expected handled with identity")
	}
}

func TestKerberos_UserPrincipalName_Mapping(t *testing.T) {
	lc := &ldapConnector{
		logger:  slog.Default(),
		krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "userPrincipalName"},
		krb:     fakeAuthenticate(newCreds("J.Doe", "Example.COM")),
	}
	lc.Config.UserSearch.IDAttr = "uid"
	lc.Config.UserSearch.EmailAttr = "mail"
	lc.Config.UserSearch.NameAttr = "cn"
	lc.krbLookupUserHook = func(_ context.Context, c *ldapConnector, username string) (ldaplib.Entry, error) {
		// userPrincipalName mapping reconstructs "username@realm" from the
		// gokrb5 credentials; LDAP server handles case according to the
		// attribute's matching rule.
		if username != "J.Doe@Example.COM" {
			t.Fatalf("expected reconstructed UPN, got %q", username)
		}
		e := ldaplib.NewEntry("cn=jdoe,dc=example,dc=org", map[string][]string{
			c.UserSearch.IDAttr:    {"uid-jdoe"},
			c.UserSearch.EmailAttr: {"jdoe@example.com"},
			c.UserSearch.NameAttr:  {"John Doe"},
		})
		return *e, nil
	}
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !bool(handled) || ident == nil {
		t.Fatalf("expected handled with identity")
	}
}

func TestKerberos_sAMAccountName_EqualsLocalpart(t *testing.T) {
	lc := &ldapConnector{
		logger:  slog.Default(),
		krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "sAMAccountName"},
		krb:     fakeAuthenticate(newCreds("Admin", "REALM.LOCAL")),
	}
	lc.Config.UserSearch.IDAttr = "uid"
	lc.Config.UserSearch.EmailAttr = "mail"
	lc.Config.UserSearch.NameAttr = "cn"
	lc.krbLookupUserHook = func(_ context.Context, c *ldapConnector, username string) (ldaplib.Entry, error) {
		if username != "Admin" {
			t.Fatalf("expected localpart-derived username Admin, got %q", username)
		}
		e := ldaplib.NewEntry("cn=admin,dc=local", map[string][]string{
			c.UserSearch.IDAttr:    {"uid-admin"},
			c.UserSearch.EmailAttr: {"admin@local"},
			c.UserSearch.NameAttr:  {"Admin"},
		})
		return *e, nil
	}
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !bool(handled) || ident == nil {
		t.Fatalf("expected handled with identity")
	}
}

func TestKerberos_OfflineAccess_SetsConnectorData(t *testing.T) {
	lc := &ldapConnector{
		logger:  slog.Default(),
		krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "localpart"},
		krb:     fakeAuthenticate(newCreds("jdoe", "EXAMPLE.COM")),
	}
	lc.Config.UserSearch.IDAttr = "uid"
	lc.Config.UserSearch.EmailAttr = "mail"
	lc.Config.UserSearch.NameAttr = "cn"
	lc.krbLookupUserHook = func(_ context.Context, c *ldapConnector, _ string) (ldaplib.Entry, error) {
		e := ldaplib.NewEntry("cn=jdoe,dc=example,dc=org", map[string][]string{
			c.UserSearch.IDAttr:    {"uid-jdoe"},
			c.UserSearch.EmailAttr: {"jdoe@example.com"},
			c.UserSearch.NameAttr:  {"John Doe"},
		})
		return *e, nil
	}
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{OfflineAccess: true}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !bool(handled) || ident == nil {
		t.Fatalf("expected handled with identity")
	}
	if len(ident.ConnectorData) == 0 {
		t.Fatalf("expected connector data for offline access")
	}
}

func TestKerberos_Disabled_PassThrough(t *testing.T) {
	// krb==nil: TrySPNEGO must leave w untouched and return (nil, false, nil).
	lc := &ldapConnector{logger: slog.Default()}
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if bool(handled) || ident != nil {
		t.Fatalf("expected not handled with no identity")
	}
}

func TestKerberos_mapPrincipal(t *testing.T) {
	cases := []struct {
		username, realm, mode, want string
	}{
		{"JDoe", "EXAMPLE.COM", "localpart", "JDoe"},
		{"JDoe", "EXAMPLE.COM", "sAMAccountName", "JDoe"},
		{"JDoe", "EXAMPLE.COM", "userPrincipalName", "JDoe@EXAMPLE.COM"},
		{"JDoe", "EXAMPLE.COM", "", "JDoe"},
		// Defensive cases: username accidentally contains '@'.
		{"JDoe@EXAMPLE.COM", "EXAMPLE.COM", "localpart", "JDoe"},
		{"JDoe@EXAMPLE.COM", "EXAMPLE.COM", "userPrincipalName", "JDoe@EXAMPLE.COM"},
		// Empty realm for userPrincipalName degrades to bare username.
		{"jdoe", "", "userPrincipalName", "jdoe"},
		{"", "EXAMPLE.COM", "localpart", ""},
	}
	for _, c := range cases {
		got := mapPrincipal(c.username, c.realm, c.mode)
		if got != c.want {
			t.Fatalf("mapPrincipal(%q,%q,%q)=%q; want %q", c.username, c.realm, c.mode, got, c.want)
		}
	}
}
