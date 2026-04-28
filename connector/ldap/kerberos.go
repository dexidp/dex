// Package ldap implements strategies for authenticating using the LDAP protocol.
// This file contains Kerberos/SPNEGO authentication support for LDAP connector.
package ldap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/jcmturner/goidentity/v6"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/keytab"
	"github.com/jcmturner/gokrb5/v8/service"
	"github.com/jcmturner/gokrb5/v8/spnego"

	"github.com/dexidp/dex/connector"
)

// bufferedResponse is a minimal http.ResponseWriter that buffers status,
// headers, and body in memory so the SPNEGO middleware's response can be
// inspected and conditionally forwarded. spnego.SPNEGOKRB5Authenticate is a
// "terminal" middleware (it commits failure responses directly to the
// ResponseWriter), but Dex needs to layer policy on top — FallbackToPassword
// may want to discard a 401 so the password form can render, and
// ExpectedRealm checks happen after a successful auth. Buffering decouples
// the middleware's output from the eventual client response.
type bufferedResponse struct {
	header      http.Header
	body        bytes.Buffer
	code        int
	wroteHeader bool
}

func newBufferedResponse() *bufferedResponse {
	return &bufferedResponse{
		header: make(http.Header),
		code:   http.StatusOK,
	}
}

func (r *bufferedResponse) Header() http.Header {
	return r.header
}

// WriteHeader mirrors net/http: the first call wins, subsequent calls are
// silently ignored (stdlib emits a "superfluous WriteHeader" warning and
// keeps the original status; we just drop the override).
func (r *bufferedResponse) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.code = code
	r.wroteHeader = true
}

// Write mirrors net/http: a Write before any WriteHeader implicitly commits
// status 200, locking the status against later overrides.
func (r *bufferedResponse) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.body.Write(b)
}

// krbState holds Kerberos/SPNEGO configuration bound to a ldapConnector.
//
// The authenticate field is the SPNEGO HTTP middleware factory: it wraps an
// inner http.Handler so that inner runs only after successful SPNEGO auth,
// with an authenticated *credentials.Credentials attached to the request
// context under goidentity.CTXKey. In production it delegates to
// spnego.SPNEGOKRB5Authenticate. Unit tests replace it with a stub that
// injects a fake credential without any real Kerberos exchange.
type krbState struct {
	authenticate func(inner http.Handler) http.Handler
}

// loadKerberosState validates the keytab and returns a krbState wired to
// spnego.SPNEGOKRB5Authenticate with the requested service settings.
func loadKerberosState(cfg kerberosConfig) (*krbState, error) {
	fi, err := os.Stat(cfg.KeytabPath)
	if err != nil {
		return nil, fmt.Errorf("keytab file not found: %w", err)
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("keytab path is a directory: %s", cfg.KeytabPath)
	}
	kt, err := keytab.Load(cfg.KeytabPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load keytab: %w", err)
	}

	settings := []func(*service.Settings){service.DecodePAC(false)}
	if cfg.SPN != "" {
		settings = append(settings, service.SName(cfg.SPN))
	}
	if cfg.KeytabPrincipal != "" {
		settings = append(settings, service.KeytabPrincipal(cfg.KeytabPrincipal))
	}
	if cfg.MaxClockSkew > 0 {
		settings = append(settings, service.MaxClockSkew(time.Duration(cfg.MaxClockSkew)*time.Second))
	}

	return &krbState{
		authenticate: func(inner http.Handler) http.Handler {
			return spnego.SPNEGOKRB5Authenticate(inner, kt, settings...)
		},
	}, nil
}

// mapPrincipal builds the LDAP-side username from the Kerberos credentials
// according to the configured mapping. Inputs come from gokrb5:
// username is credentials.UserName() (bare, no realm); realm is Domain().
//
//   - "userprincipalname":                    "username@realm".
//   - "localpart"/"samaccountname" (default): bare username.
//
// Case of the output is preserved; callers rely on the LDAP server's
// attribute matching rules (typically case-insensitive) for comparisons.
func mapPrincipal(username, realm, mapping string) string {
	// Defensive: if username somehow carries an '@', trim to before it; we
	// always derive the realm from the separate Domain() field.
	if i := strings.IndexByte(username, '@'); i >= 0 {
		username = username[:i]
	}
	if strings.EqualFold(mapping, "userprincipalname") {
		if realm == "" {
			return username
		}
		return username + "@" + realm
	}
	return username
}

// TrySPNEGO attempts SPNEGO authentication against the LDAP connector's keytab.
//
// Behavior:
//   - Kerberos disabled: returns (nil, false, nil) so the caller renders the
//     password form.
//   - FallbackToPassword=true with no Authorization header: a cookie probe
//     gives the browser exactly one round to negotiate. The first such
//     request sets a short-lived "tried" cookie and forwards the middleware's
//     401 Negotiate challenge so a Kerberos-aware client can respond. If the
//     follow-up request still has no Authorization header (cookie present),
//     we treat the client as unable to SPNEGO and render the password form.
//   - FallbackToPassword=true with an Authorization header that the
//     middleware rejects: render the password form (the client tried and
//     failed; do not loop 401s).
//   - FallbackToPassword=false: forward the middleware's response verbatim.
//   - On success, the authenticated principal is resolved in LDAP and a
//     connector.Identity is returned. Any prior probe cookie is cleared.
func (c *ldapConnector) TrySPNEGO(ctx context.Context, s connector.Scopes, w http.ResponseWriter, r *http.Request) (*connector.Identity, connector.Handled, error) {
	if c.krb == nil {
		return nil, false, nil
	}

	hasNegotiate := strings.HasPrefix(r.Header.Get("Authorization"), "Negotiate ")

	// Cookie probe: see the package-level doc on spnegoProbeCookieName. Only
	// applies when fallback is enabled and the client has not (yet) sent a
	// SPNEGO token; "Authorization present" paths bypass the probe entirely.
	if c.krbConf.FallbackToPassword && !hasNegotiate {
		if hasSPNEGOProbeCookie(r) {
			c.logger.Info("kerberos: SPNEGO probe cookie present and no Negotiate header; falling back to password form")
			return nil, false, nil
		}
		setSPNEGOProbeCookie(w, r)
	}

	// Run the SPNEGO middleware with a capturing recorder so we can decide
	// whether to forward its response or fall back to the password form.
	var (
		id       *credentials.Credentials
		innerErr error
	)
	inner := http.HandlerFunc(func(_ http.ResponseWriter, rr *http.Request) {
		ident := goidentity.FromHTTPRequestContext(rr)
		if ident == nil {
			innerErr = fmt.Errorf("kerberos: no identity in request context after SPNEGO")
			return
		}
		creds, ok := ident.(*credentials.Credentials)
		if !ok {
			innerErr = fmt.Errorf("kerberos: unexpected identity type %T", ident)
			return
		}
		id = creds
	})

	rec := newBufferedResponse()
	c.krb.authenticate(inner).ServeHTTP(rec, r)

	if innerErr != nil {
		c.logger.Error("kerberos: SPNEGO middleware completed with unusable credentials", "err", innerErr)
		return nil, true, innerErr
	}

	if id == nil {
		// Client offered a token and the middleware rejected it: under
		// fallback semantics this is "tried and failed" — render the form
		// rather than looping 401s.
		if c.krbConf.FallbackToPassword && hasNegotiate {
			c.logger.Info("kerberos: SPNEGO rejected client token; falling back to password form")
			return nil, false, nil
		}
		// Otherwise (fallback=false, OR fallback=true probe round): forward
		// the middleware-authored response (challenge token, error payload,
		// etc.) verbatim — the protocol decided to reject and we preserve
		// its wire details.
		c.logger.Info("kerberos: SPNEGO did not authenticate; forwarding middleware response", "status", rec.code)
		copyBuffered(rec, w)
		return nil, true, nil
	}

	if c.krbConf.ExpectedRealm != "" && !strings.EqualFold(c.krbConf.ExpectedRealm, id.Domain()) {
		c.logger.Info("kerberos: realm mismatch", "expected", c.krbConf.ExpectedRealm, "actual", id.Domain())
		if c.krbConf.FallbackToPassword {
			// Intentionally do NOT clear the probe cookie here: the
			// client's TGT is for a realm we will never accept. Clearing
			// would re-arm a probe round on the next no-Authorization GET,
			// re-issue 401 Negotiate, the browser would resend the same
			// wrong-realm token, and we'd land back here — an infinite
			// flap. Keeping the cookie pins the client to the password
			// form until the cookie's Max-Age elapses.
			return nil, false, nil
		}
		// SPNEGO authenticated successfully, but our ExpectedRealm policy
		// rejects it. The middleware's buffered response is irrelevant here
		// (it represents the post-success path through the inner handler);
		// emit our own bare Negotiate challenge so the client knows to
		// retry with a different realm.
		writeBareNegotiateChallenge(w)
		return nil, true, nil
	}

	mapped := mapPrincipal(id.UserName(), id.Domain(), c.krbConf.UsernameFromPrincipal)
	c.logger.Info("kerberos: principal authenticated",
		"principal", id.UserName(),
		"realm", id.Domain(),
		"auth_time", id.AuthTime(),
		"mapped_username", mapped,
	)

	userEntry, err := c.lookupKerberosUser(ctx, mapped)
	if err != nil {
		c.logger.Error("kerberos: LDAP user lookup failed",
			"principal", id.UserName(), "mapped", mapped, "err", err)
		return nil, true, fmt.Errorf("ldap: user lookup failed for kerberos principal %q: %w", id.UserName(), err)
	}
	c.logger.Info("kerberos: LDAP user found", "dn", userEntry.DN)

	ident, err := c.identityFromEntry(userEntry)
	if err != nil {
		c.logger.Error("kerberos: failed to build identity from LDAP entry", "err", err)
		return nil, true, err
	}
	if s.Groups {
		groups, err := c.groups(ctx, userEntry)
		if err != nil {
			c.logger.Error("kerberos: failed to query groups", "err", err)
			return nil, true, fmt.Errorf("ldap: failed to query groups: %w", err)
		}
		ident.Groups = groups
	}

	// No user-bind has happened; only materialize ConnectorData when refresh
	// needs the LDAP entry later (OfflineAccess). Mirror (*ldapConnector).Login:
	// a marshal failure here would silently break a subsequent Refresh call
	// (Unmarshal on nil), so fail the login instead of letting that happen.
	if s.OfflineAccess {
		refresh := refreshData{Username: mapped, Entry: userEntry}
		data, mErr := json.Marshal(refresh)
		if mErr != nil {
			c.logger.Error("kerberos: failed to marshal refresh data", "err", mErr)
			return nil, true, fmt.Errorf("ldap: marshal refresh data: %w", mErr)
		}
		ident.ConnectorData = data
	}

	// If the client carried a stale probe cookie from a prior fallback
	// round, clear it so a future logout/re-login starts negotiation fresh.
	if hasSPNEGOProbeCookie(r) {
		clearSPNEGOProbeCookie(w, r)
	}

	c.logger.Info("kerberos: SPNEGO login succeeded",
		"username", ident.Username, "email", ident.Email, "groups_count", len(ident.Groups))
	return &ident, true, nil
}

// copyBuffered forwards a buffered handler response to the real client.
// Used when we want the middleware-authored bytes to reach the user agent
// unchanged (e.g. SPNEGO continuation/reject tokens).
func copyBuffered(rec *bufferedResponse, w http.ResponseWriter) {
	for k, vv := range rec.header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(rec.code)
	_, _ = w.Write(rec.body.Bytes())
}

// writeBareNegotiateChallenge emits an unsolicited "WWW-Authenticate:
// Negotiate" 401 directly to the client. This is reserved for cases where
// Dex itself rejects an otherwise-successful SPNEGO exchange (currently
// only ExpectedRealm mismatch); in those cases the middleware's buffered
// output does not represent the answer we want to send.
func writeBareNegotiateChallenge(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Negotiate")
	w.WriteHeader(http.StatusUnauthorized)
}

// spnegoProbeCookieName names the short-lived cookie that records "we
// already issued a 401 Negotiate challenge to this client". It implements
// the fallback-to-password semantics: the very first GET without an
// Authorization header gets a real Negotiate challenge so a Kerberos-aware
// browser can SSO; if the client comes back without a token we treat that
// as "client cannot/will not negotiate" and render the password form
// instead of looping 401s. The cookie is bounded by a small Max-Age so a
// later visit gets a fresh chance to negotiate.
const (
	spnegoProbeCookieName = "dex_spnego_tried"
	spnegoProbeMaxAge     = 60 // seconds; long enough for one challenge round-trip
)

func hasSPNEGOProbeCookie(r *http.Request) bool {
	_, err := r.Cookie(spnegoProbeCookieName)
	return err == nil
}

// newSPNEGOProbeCookie returns a probe-cookie carrier with the attributes
// shared between "set" and "clear" forms (Path, HttpOnly, Secure, SameSite),
// so the two callers cannot accidentally drift apart.
func newSPNEGOProbeCookie(r *http.Request, value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:  spnegoProbeCookieName,
		Value: value,
		// Scope to "/" so the cookie reaches every Dex auth endpoint
		// regardless of issuer prefix; the cookie carries no secret and
		// expires within spnegoProbeMaxAge seconds, so the broad path is
		// not a privacy or security boundary concern.
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
	}
}

func setSPNEGOProbeCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, newSPNEGOProbeCookie(r, "1", spnegoProbeMaxAge))
}

func clearSPNEGOProbeCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, newSPNEGOProbeCookie(r, "", -1))
}

// isSecureRequest reports whether the original client connection is HTTPS.
// It honors X-Forwarded-Proto so the cookie's Secure attribute is correct
// when Dex sits behind a TLS-terminating proxy. The header is treated as
// advisory: misbehaving/spoofed values can only flip Secure to true on a
// plain-HTTP setup (which makes browsers refuse to echo the cookie back —
// a graceful no-op for the probe), never to false on a real HTTPS link.
func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// lookupKerberosUser resolves an LDAP user entry by username. When
// krbLookupUserHook is non-nil it is used exclusively and the real LDAP
// search is skipped entirely — this keeps unit tests hermetic.
func (c *ldapConnector) lookupKerberosUser(ctx context.Context, username string) (ldap.Entry, error) {
	if c.krbLookupUserHook != nil {
		return c.krbLookupUserHook(ctx, c, username)
	}

	var userEntry ldap.Entry
	err := c.do(ctx, func(conn *ldap.Conn) error {
		entry, found, err := c.userEntry(conn, username)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("user not found for principal")
		}
		userEntry = entry
		return nil
	})
	return userEntry, err
}
