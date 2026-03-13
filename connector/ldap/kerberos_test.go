package ldap

import (
	"encoding/base64"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ldaplib "github.com/go-ldap/ldap/v3"

	"github.com/dexidp/dex/connector"
)

type mockKrbValidator struct {
	principal  string
	realm      string
	ok         bool
	err        error
	challenged bool
	contToken  []byte
	step       int // -1 means disabled; 0->continue, then success
}

func (m *mockKrbValidator) ValidateRequest(r *http.Request) (string, string, bool, error) {
	if m.step >= 0 {
		if m.step == 0 {
			return "", "", false, nil
		}
		return m.principal, m.realm, true, nil
	}
	return m.principal, m.realm, m.ok, m.err
}

func (m *mockKrbValidator) Challenge(w http.ResponseWriter) {
	m.challenged = true
	writeNegotiateChallenge(w)
}

func (m *mockKrbValidator) ContinueToken(r *http.Request) ([]byte, bool) {
	if m.step >= 0 {
		if m.step == 0 && len(m.contToken) > 0 {
			m.step++
			return m.contToken, true
		}
		return nil, false
	}
	if len(m.contToken) > 0 {
		return m.contToken, true
	}
	return nil, false
}

func TestKerberos_NoHeader_Returns401Negotiate(t *testing.T) {
	lc := &ldapConnector{logger: slog.Default(), krbEnabled: true, krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "localpart"}}
	mv := &mockKrbValidator{principal: "", realm: "", ok: false, err: nil, step: -1}
	lc.krbValidator = mv
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
	if w.Result().StatusCode != 401 {
		t.Fatalf("expected 401, got %d", w.Result().StatusCode)
	}
	if hdr := w.Header().Get("WWW-Authenticate"); hdr != "Negotiate" {
		t.Fatalf("expected Negotiate challenge")
	}
}

func TestKerberos_ExpectedRealmMismatch_401(t *testing.T) {
	lc := &ldapConnector{logger: slog.Default(), krbEnabled: true, krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "localpart", ExpectedRealm: "EXAMPLE.COM"}}
	mv := &mockKrbValidator{principal: "jdoe@OTHER.COM", realm: "OTHER.COM", ok: true, err: nil, step: -1}
	lc.krbValidator = mv
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
	if w.Result().StatusCode != 401 {
		t.Fatalf("expected 401, got %d", w.Result().StatusCode)
	}
	if hdr := w.Header().Get("WWW-Authenticate"); hdr != "Negotiate" {
		t.Fatalf("expected Negotiate challenge")
	}
}

func TestKerberos_FallbackToPassword_NoHeader_HandledFalse(t *testing.T) {
	lc := &ldapConnector{logger: slog.Default(), krbEnabled: true, krbConf: kerberosConfig{FallbackToPassword: true, UsernameFromPrincipal: "localpart"}}
	mv := &mockKrbValidator{principal: "", realm: "", ok: false, err: nil, step: -1}
	lc.krbValidator = mv
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if bool(handled) {
		t.Fatalf("expected not handled")
	}
	if ident != nil {
		t.Fatalf("expected no identity")
	}
}

func TestKerberos_ContinueNeeded_SendsResponseToken(t *testing.T) {
	lc := &ldapConnector{logger: slog.Default(), krbEnabled: true, krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "localpart"}}
	mv := &mockKrbValidator{principal: "", realm: "", ok: false, err: nil, contToken: []byte{0x01, 0x02, 0x03}}
	lc.krbValidator = mv
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	r.Header.Set("Authorization", "Negotiate Zm9v")
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
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	hdr := w.Header().Get("WWW-Authenticate")
	if !strings.HasPrefix(hdr, "Negotiate ") {
		t.Fatalf("expected Negotiate header, got %q", hdr)
	}
	want := "Negotiate " + base64.StdEncoding.EncodeToString([]byte{0x01, 0x02, 0x03})
	if hdr != want {
		t.Fatalf("unexpected negotiate header, got %q want %q", hdr, want)
	}
}

func TestKerberos_ContinueThenSuccess_ShortCircuitIdentity(t *testing.T) {
	lc := &ldapConnector{logger: slog.Default(), krbEnabled: true, krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "localpart"}}
	lc.Config.UserSearch.IDAttr = "uid"
	lc.Config.UserSearch.EmailAttr = "mail"
	lc.Config.UserSearch.NameAttr = "cn"
	mv := &mockKrbValidator{principal: "jdoe@EXAMPLE.COM", realm: "EXAMPLE.COM", contToken: []byte{0xAA, 0xBB}, step: 0}
	lc.krbValidator = mv

	// First request -> 401 with continue token
	r1 := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w1 := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r1.Context(), connector.Scopes{}, w1, r1)
	if err != nil || !bool(handled) || ident != nil || w1.Code != 401 {
		t.Fatalf("unexpected first step result: ident=%v handled=%v code=%d err=%v", ident, handled, w1.Code, err)
	}
	hdr := w1.Header().Get("WWW-Authenticate")
	if hdr == "" || !strings.HasPrefix(hdr, "Negotiate ") {
		t.Fatalf("expected Negotiate header on first step, got %q", hdr)
	}

	// Second request -> validator now returns ok=true (due to step increment)
	krbLookupUserHook = func(c *ldapConnector, username string) (ldaplib.Entry, bool, error) {
		e := ldaplib.NewEntry("cn=jdoe,dc=example,dc=org", map[string][]string{
			c.UserSearch.IDAttr:    {"uid-jdoe"},
			c.UserSearch.EmailAttr: {"jdoe@example.com"},
			c.UserSearch.NameAttr:  {"John Doe"},
		})
		return *e, true, nil
	}
	defer func() { krbLookupUserHook = nil }()
	r2 := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w2 := httptest.NewRecorder()
	ident2, handled2, err2 := lc.TrySPNEGO(r2.Context(), connector.Scopes{}, w2, r2)
	if err2 != nil {
		t.Fatalf("unexpected err: %v", err2)
	}
	if !bool(handled2) || ident2 == nil {
		t.Fatalf("expected handled with identity on second step")
	}
}

func TestKerberos_ContinueNeeded_FallbackTrue_NotHandled(t *testing.T) {
	lc := &ldapConnector{logger: slog.Default(), krbEnabled: true, krbConf: kerberosConfig{FallbackToPassword: true, UsernameFromPrincipal: "localpart"}}
	mv := &mockKrbValidator{contToken: []byte{0x10}, step: 0}
	lc.krbValidator = mv
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if bool(handled) || ident != nil {
		t.Fatalf("expected not handled with fallback=true when continue is needed")
	}
	if w.Code != 200 && w.Code != 0 {
		t.Fatalf("expected no response written yet, got %d", w.Code)
	}
}

func TestKerberos_mapPrincipal(t *testing.T) {
	cases := []struct{ in, realm, mode, want string }{
		{"JDoe@EXAMPLE.COM", "EXAMPLE.COM", "localpart", "jdoe"},
		{"JDoe@EXAMPLE.COM", "EXAMPLE.COM", "sAMAccountName", "jdoe"},
		{"JDoe@EXAMPLE.COM", "EXAMPLE.COM", "userPrincipalName", "jdoe@example.com"},
	}
	for _, c := range cases {
		got := mapPrincipal(c.in, c.realm, c.mode)
		if got != c.want {
			t.Fatalf("mapPrincipal(%q,%q,%q)=%q; want %q", c.in, c.realm, c.mode, got, c.want)
		}
	}
}

func TestKerberos_UserNotFound_ReturnsError(t *testing.T) {
	lc := &ldapConnector{logger: slog.Default(), krbEnabled: true, krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "localpart"}}
	mv := &mockKrbValidator{principal: "jdoe@EXAMPLE.COM", realm: "EXAMPLE.COM", ok: true, err: nil, step: -1}
	lc.krbValidator = mv
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err == nil {
		t.Fatalf("expected error for user not found")
	}
	if !bool(handled) {
		t.Fatalf("expected handled")
	}
	if ident != nil {
		t.Fatalf("expected no identity")
	}
	if !strings.Contains(err.Error(), "user lookup failed") {
		t.Fatalf("expected 'user lookup failed' error, got: %v", err)
	}
}

func TestKerberos_ValidPrincipal_CompletesFlow(t *testing.T) {
	lc := &ldapConnector{logger: slog.Default(), krbEnabled: true, krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "localpart"}}
	lc.Config.UserSearch.IDAttr = "uid"
	lc.Config.UserSearch.EmailAttr = "mail"
	lc.Config.UserSearch.NameAttr = "cn"
	mv := &mockKrbValidator{principal: "jdoe@EXAMPLE.COM", realm: "EXAMPLE.COM", ok: true, err: nil, step: -1}
	lc.krbValidator = mv
	krbLookupUserHook = func(c *ldapConnector, username string) (ldaplib.Entry, bool, error) {
		e := ldaplib.NewEntry("cn=jdoe,dc=example,dc=org", map[string][]string{
			c.UserSearch.IDAttr:    {"uid-jdoe"},
			c.UserSearch.EmailAttr: {"jdoe@example.com"},
			c.UserSearch.NameAttr:  {"John Doe"},
		})
		return *e, true, nil
	}
	defer func() { krbLookupUserHook = nil }()
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !bool(handled) {
		t.Fatalf("expected handled")
	}
	if ident == nil {
		t.Fatalf("expected identity")
	}
	if ident.Username == "" || ident.Email == "" || ident.UserID == "" {
		t.Fatalf("expected populated identity, got %+v", *ident)
	}
}

func TestKerberos_InvalidHeader_Returns401Negotiate(t *testing.T) {
	lc := &ldapConnector{logger: slog.Default(), krbEnabled: true, krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "localpart"}}
	mv := &mockKrbValidator{principal: "", realm: "", ok: false, err: nil, step: -1}
	lc.krbValidator = mv
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	r.Header.Set("Authorization", "Negotiate !!!notbase64!!!")
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
	if w.Result().StatusCode != 401 {
		t.Fatalf("expected 401, got %d", w.Result().StatusCode)
	}
	if hdr := w.Header().Get("WWW-Authenticate"); hdr != "Negotiate" {
		t.Fatalf("expected Negotiate challenge")
	}
}

func TestKerberos_UserPrincipalName_Mapping(t *testing.T) {
	lc := &ldapConnector{logger: slog.Default(), krbEnabled: true, krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "userPrincipalName"}}
	lc.Config.UserSearch.IDAttr = "uid"
	lc.Config.UserSearch.EmailAttr = "mail"
	lc.Config.UserSearch.NameAttr = "cn"
	mv := &mockKrbValidator{principal: "J.Doe@Example.COM", realm: "Example.COM", ok: true, err: nil, step: -1}
	lc.krbValidator = mv
	krbLookupUserHook = func(c *ldapConnector, username string) (ldaplib.Entry, bool, error) {
		if username != "j.doe@example.com" {
			return ldaplib.Entry{}, false, nil
		}
		e := ldaplib.NewEntry("cn=jdoe,dc=example,dc=org", map[string][]string{
			c.UserSearch.IDAttr:    {"uid-jdoe"},
			c.UserSearch.EmailAttr: {"jdoe@example.com"},
			c.UserSearch.NameAttr:  {"John Doe"},
		})
		return *e, true, nil
	}
	defer func() { krbLookupUserHook = nil }()
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !bool(handled) {
		t.Fatalf("expected handled")
	}
	if ident == nil {
		t.Fatalf("expected identity")
	}
}

func TestKerberos_OfflineAccess_SetsConnectorData(t *testing.T) {
	lc := &ldapConnector{logger: slog.Default(), krbEnabled: true, krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "localpart"}}
	lc.Config.UserSearch.IDAttr = "uid"
	lc.Config.UserSearch.EmailAttr = "mail"
	lc.Config.UserSearch.NameAttr = "cn"
	mv := &mockKrbValidator{principal: "jdoe@EXAMPLE.COM", realm: "EXAMPLE.COM", ok: true, err: nil, step: -1}
	lc.krbValidator = mv
	krbLookupUserHook = func(c *ldapConnector, username string) (ldaplib.Entry, bool, error) {
		e := ldaplib.NewEntry("cn=jdoe,dc=example,dc=org", map[string][]string{
			c.UserSearch.IDAttr:    {"uid-jdoe"},
			c.UserSearch.EmailAttr: {"jdoe@example.com"},
			c.UserSearch.NameAttr:  {"John Doe"},
		})
		return *e, true, nil
	}
	defer func() { krbLookupUserHook = nil }()
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	scopes := connector.Scopes{OfflineAccess: true}
	ident, handled, err := lc.TrySPNEGO(r.Context(), scopes, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !bool(handled) {
		t.Fatalf("expected handled")
	}
	if ident == nil {
		t.Fatalf("expected identity")
	}
	if len(ident.ConnectorData) == 0 {
		t.Fatalf("expected connector data for offline access")
	}
}

func TestKerberos_FallbackTrue_InvalidHeader_NotHandled(t *testing.T) {
	lc := &ldapConnector{logger: slog.Default(), krbEnabled: true, krbConf: kerberosConfig{FallbackToPassword: true, UsernameFromPrincipal: "localpart"}}
	mv := &mockKrbValidator{principal: "", realm: "", ok: false, err: nil, step: -1}
	lc.krbValidator = mv
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	r.Header.Set("Authorization", "Negotiate !!!")
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if bool(handled) {
		t.Fatalf("expected not handled for fallback path")
	}
	if ident != nil {
		t.Fatalf("expected no identity")
	}
	if w.Code != 200 && w.Code != 0 {
		t.Fatalf("expected no response written yet, got %d", w.Code)
	}
}

func TestKerberos_sAMAccountName_EqualsLocalpart(t *testing.T) {
	lc := &ldapConnector{logger: slog.Default(), krbEnabled: true, krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "sAMAccountName"}}
	lc.Config.UserSearch.IDAttr = "uid"
	lc.Config.UserSearch.EmailAttr = "mail"
	lc.Config.UserSearch.NameAttr = "cn"
	mv := &mockKrbValidator{principal: "Admin@REALM.LOCAL", realm: "REALM.LOCAL", ok: true, err: nil, step: -1}
	lc.krbValidator = mv
	krbLookupUserHook = func(c *ldapConnector, username string) (ldaplib.Entry, bool, error) {
		if username != "admin" {
			return ldaplib.Entry{}, false, nil
		}
		e := ldaplib.NewEntry("cn=admin,dc=local", map[string][]string{
			c.UserSearch.IDAttr:    {"uid-admin"},
			c.UserSearch.EmailAttr: {"admin@local"},
			c.UserSearch.NameAttr:  {"Admin"},
		})
		return *e, true, nil
	}
	defer func() { krbLookupUserHook = nil }()
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !bool(handled) {
		t.Fatalf("expected handled")
	}
	if ident == nil {
		t.Fatalf("expected identity")
	}
}

func TestKerberos_ExpectedRealm_CaseInsensitive(t *testing.T) {
	lc := &ldapConnector{logger: slog.Default(), krbEnabled: true, krbConf: kerberosConfig{FallbackToPassword: false, UsernameFromPrincipal: "localpart", ExpectedRealm: "ExAmPlE.CoM"}}
	lc.Config.UserSearch.IDAttr = "uid"
	lc.Config.UserSearch.EmailAttr = "mail"
	lc.Config.UserSearch.NameAttr = "cn"
	mv := &mockKrbValidator{principal: "user@EXAMPLE.COM", realm: "EXAMPLE.COM", ok: true, err: nil, step: -1}
	lc.krbValidator = mv
	krbLookupUserHook = func(c *ldapConnector, username string) (ldaplib.Entry, bool, error) {
		e := ldaplib.NewEntry("cn=user,dc=example,dc=com", map[string][]string{
			c.UserSearch.IDAttr:    {"uid-user"},
			c.UserSearch.EmailAttr: {"user@example.com"},
			c.UserSearch.NameAttr:  {"User"},
		})
		return *e, true, nil
	}
	defer func() { krbLookupUserHook = nil }()
	r := httptest.NewRequest("GET", "/auth/ldap/login?state=abc", nil)
	w := httptest.NewRecorder()
	ident, handled, err := lc.TrySPNEGO(r.Context(), connector.Scopes{}, w, r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !bool(handled) {
		t.Fatalf("expected handled")
	}
	if ident == nil {
		t.Fatalf("expected identity")
	}
}
