// Package ldap implements strategies for authenticating using the LDAP protocol.
// This file contains Kerberos/SPNEGO authentication support for LDAP connector.
package ldap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
//   - FallbackToPassword=true and the request carries no Negotiate header:
//     short-circuits to the password form without invoking the SPNEGO
//     middleware. This avoids writing an unwanted 401 when a password is
//     explicitly permitted.
//   - Otherwise, runs the SPNEGO middleware into a response recorder. If the
//     middleware does not authenticate, its buffered response is either
//     forwarded to the client (FallbackToPassword=false) or discarded
//     (FallbackToPassword=true, caller renders form).
//   - On success, the authenticated principal is resolved in LDAP and a
//     connector.Identity is returned.
func (c *ldapConnector) TrySPNEGO(ctx context.Context, s connector.Scopes, w http.ResponseWriter, r *http.Request) (*connector.Identity, connector.Handled, error) {
	if c.krb == nil {
		return nil, false, nil
	}

	if c.krbConf.FallbackToPassword &&
		!strings.HasPrefix(r.Header.Get("Authorization"), "Negotiate ") {
		return nil, false, nil
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

	rec := httptest.NewRecorder()
	c.krb.authenticate(inner).ServeHTTP(rec, r)

	if innerErr != nil {
		c.logger.Error("kerberos: SPNEGO middleware completed with unusable credentials", "err", innerErr)
		return nil, true, innerErr
	}

	if id == nil {
		if c.krbConf.FallbackToPassword {
			c.logger.Info("kerberos: SPNEGO did not authenticate; falling back to password form")
			return nil, false, nil
		}
		c.logger.Info("kerberos: SPNEGO did not authenticate; forwarding middleware response", "status", rec.Code)
		copyRecorded(rec, w)
		return nil, true, nil
	}

	if c.krbConf.ExpectedRealm != "" && !strings.EqualFold(c.krbConf.ExpectedRealm, id.Domain()) {
		c.logger.Info("kerberos: realm mismatch", "expected", c.krbConf.ExpectedRealm, "actual", id.Domain())
		if c.krbConf.FallbackToPassword {
			return nil, false, nil
		}
		w.Header().Set("WWW-Authenticate", "Negotiate")
		w.WriteHeader(http.StatusUnauthorized)
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

	c.logger.Info("kerberos: SPNEGO login succeeded",
		"username", ident.Username, "email", ident.Email, "groups_count", len(ident.Groups))
	return &ident, true, nil
}

// copyRecorded forwards the buffered middleware response to the client.
func copyRecorded(rec *httptest.ResponseRecorder, w http.ResponseWriter) {
	for k, vv := range rec.Header() {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(rec.Code)
	_, _ = w.Write(rec.Body.Bytes())
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
