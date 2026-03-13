// Package ldap implements strategies for authenticating using the LDAP protocol.
// This file contains Kerberos/SPNEGO authentication support for LDAP connector.
package ldap

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/jcmturner/gofork/encoding/asn1"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/gssapi"
	"github.com/jcmturner/gokrb5/v8/keytab"
	"github.com/jcmturner/gokrb5/v8/service"
	"github.com/jcmturner/gokrb5/v8/spnego"
	"github.com/jcmturner/gokrb5/v8/types"

	"github.com/dexidp/dex/connector"
	"github.com/go-ldap/ldap/v3"
)

// KerberosValidator abstracts SPNEGO validation for unit-testing.
type KerberosValidator interface {
	// ValidateRequest returns (principal, realm, ok, err). ok=false means header missing/invalid.
	ValidateRequest(r *http.Request) (string, string, bool, error)
	// Challenge writes a 401 Negotiate challenge.
	Challenge(w http.ResponseWriter)
	// ContinueToken tries to advance SPNEGO handshake and returns a response token to include
	// in WWW-Authenticate: Negotiate <token>. Returns (nil, false) if no continuation is needed/possible.
	ContinueToken(r *http.Request) ([]byte, bool)
}

// writeNegotiateChallenge writes a standard 401 Negotiate challenge.
func writeNegotiateChallenge(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Negotiate")
	w.WriteHeader(http.StatusUnauthorized)
}

// mapPrincipal maps a Kerberos principal to LDAP username per configuration.
func mapPrincipal(principal, realm, mapping string) string {
	p := principal
	switch strings.ToLower(mapping) {
	case "localpart", "samaccountname":
		if i := strings.IndexByte(principal, '@'); i >= 0 {
			p = principal[:i]
		}
		return strings.ToLower(p)
	case "userprincipalname":
		return strings.ToLower(principal)
	default:
		if i := strings.IndexByte(principal, '@'); i >= 0 {
			p = principal[:i]
		}
		return strings.ToLower(p)
	}
}

// gokrb5 implementation of KerberosValidator

// context key used by gokrb5 to store credentials in the context
var ctxCredentialsKey interface{} = "github.com/jcmturner/gokrb5/v8/ctxCredentials"

// SPNEGO NegTokenResp (AcceptIncomplete + KRB5 mech) base64 payload used by gokrb5's HTTP server
// to prompt the client to continue the handshake.
const spnegoIncompleteKRB5B64 = "oRQwEqADCgEBoQsGCSqGSIb3EgECAg=="

type gokrb5Validator struct {
	kt     *keytab.Keytab
	logger *slog.Logger
}

func newGokrb5ValidatorWithLogger(keytabPath string, logger *slog.Logger) (KerberosValidator, error) {
	kt, err := keytab.Load(keytabPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load keytab: %w", err)
	}
	if fi, err := os.Stat(keytabPath); err != nil || fi.IsDir() {
		return nil, fmt.Errorf("invalid keytab path: %s", keytabPath)
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &gokrb5Validator{kt: kt, logger: logger}, nil
}

func (v *gokrb5Validator) ValidateRequest(r *http.Request) (string, string, bool, error) {
	h := r.Header.Get("Authorization")
	if h == "" || !strings.HasPrefix(h, "Negotiate ") {
		if v.logger != nil {
			v.logger.Info("kerberos: missing or non-negotiate Authorization header", "path", r.URL.Path)
		}
		return "", "", false, nil
	}
	b64 := strings.TrimSpace(h[len("Negotiate "):])
	if b64 == "" {
		if v.logger != nil {
			v.logger.Info("kerberos: empty negotiate token", "path", r.URL.Path)
		}
		return "", "", false, nil
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		if v.logger != nil {
			v.logger.Info("kerberos: invalid base64 in Authorization", "err", err)
		}
		return "", "", false, nil
	}
	var tok spnego.SPNEGOToken
	if err := tok.Unmarshal(data); err != nil {
		// Try raw KRB5 token and wrap
		var k5 spnego.KRB5Token
		if k5.Unmarshal(data) != nil {
			if v.logger != nil {
				v.logger.Info("kerberos: failed to unmarshal SPNEGO token and not raw KRB5", "err", err)
			}
			return "", "", false, nil
		}
		tok.Init = true
		tok.NegTokenInit = spnego.NegTokenInit{
			MechTypes:      []asn1.ObjectIdentifier{k5.OID},
			MechTokenBytes: data,
		}
	}

	// Pass client address when available (improves AP-REQ validation with address-bound tickets)
	var sp *spnego.SPNEGO
	if ha, err := types.GetHostAddress(r.RemoteAddr); err == nil {
		sp = spnego.SPNEGOService(v.kt, service.ClientAddress(ha), service.DecodePAC(false))
	} else {
		if v.logger != nil {
			v.logger.Info("kerberos: cannot parse client address", "remote", r.RemoteAddr, "err", err)
		}
		sp = spnego.SPNEGOService(v.kt, service.DecodePAC(false))
	}
	authed, ctx, status := sp.AcceptSecContext(&tok)
	if status.Code != gssapi.StatusComplete {
		if v.logger != nil {
			v.logger.Info("kerberos: AcceptSecContext not complete", "code", status.Code, "message", status.Message)
		}
		return "", "", false, nil
	}
	if !authed || ctx == nil {
		if v.logger != nil {
			v.logger.Info("kerberos: not authenticated or no context")
		}
		return "", "", false, nil
	}
	id, _ := ctx.Value(ctxCredentialsKey).(*credentials.Credentials)
	if id == nil {
		if v.logger != nil {
			v.logger.Info("kerberos: credentials missing in context")
		}
		return "", "", false, fmt.Errorf("no credentials in context")
	}
	return id.UserName(), id.Domain(), true, nil
}

func (v *gokrb5Validator) Challenge(w http.ResponseWriter) { writeNegotiateChallenge(w) }

// ContinueToken attempts to continue the SPNEGO handshake and returns a response token
// (to be placed into WWW-Authenticate: Negotiate <b64>) if available.
func (v *gokrb5Validator) ContinueToken(r *http.Request) ([]byte, bool) {
	h := r.Header.Get("Authorization")
	if h == "" || !strings.HasPrefix(h, "Negotiate ") {
		if v.logger != nil {
			v.logger.Info("kerberos: ContinueToken without negotiate header", "path", r.URL.Path)
		}
		return nil, false
	}
	b64 := strings.TrimSpace(h[len("Negotiate "):])
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		// Malformed header: ask client to continue with KRB5 mech
		if tok, e := base64.StdEncoding.DecodeString(spnegoIncompleteKRB5B64); e == nil {
			if v.logger != nil {
				v.logger.Info("kerberos: malformed negotiate token; sending incomplete KRB5 response")
			}
			return tok, true
		}
		return nil, false
	}
	var tok spnego.SPNEGOToken
	if err := tok.Unmarshal(data); err != nil {
		// Not a full SPNEGO token; still ask client to continue
		if tokb, e := base64.StdEncoding.DecodeString(spnegoIncompleteKRB5B64); e == nil {
			if v.logger != nil {
				v.logger.Info("kerberos: non-SPNEGO token; sending incomplete KRB5 response")
			}
			return tokb, true
		}
		// As a fallback, try wrapping as raw KRB5
		var k5 spnego.KRB5Token
		if k5.Unmarshal(data) != nil {
			if v.logger != nil {
				v.logger.Info("kerberos: not KRB5 token; cannot continue")
			}
			return nil, false
		}
		tok.Init = true
		tok.NegTokenInit = spnego.NegTokenInit{MechTypes: []asn1.ObjectIdentifier{k5.OID}, MechTokenBytes: data}
	}
	// Try continue with same options as in ValidateRequest
	var sp *spnego.SPNEGO
	if ha, err := types.GetHostAddress(r.RemoteAddr); err == nil {
		sp = spnego.SPNEGOService(v.kt, service.ClientAddress(ha), service.DecodePAC(false))
	} else {
		sp = spnego.SPNEGOService(v.kt, service.DecodePAC(false))
	}
	_, ctx, status := sp.AcceptSecContext(&tok)
	if status.Code != gssapi.StatusContinueNeeded || ctx == nil {
		if v.logger != nil {
			v.logger.Info("kerberos: no continuation required", "code", status.Code, "message", status.Message)
		}
		return nil, false
	}
	// Ask client to continue using standard NegTokenResp (KRB5, incomplete)
	if tokb, e := base64.StdEncoding.DecodeString(spnegoIncompleteKRB5B64); e == nil {
		if v.logger != nil {
			v.logger.Info("kerberos: continuation needed; sending incomplete KRB5 response")
		}
		return tokb, true
	}
	return nil, false
}

// LDAP connector SPNEGO integration

// krbLookupUserHook allows tests to inject a user entry without LDAP queries.
var krbLookupUserHook func(c *ldapConnector, username string) (ldap.Entry, bool, error)

// TrySPNEGO attempts Kerberos auth and builds identity on success.
func (c *ldapConnector) TrySPNEGO(ctx context.Context, s connector.Scopes, w http.ResponseWriter, r *http.Request) (*connector.Identity, connector.Handled, error) {
	if !c.krbEnabled || c.krbValidator == nil {
		return nil, false, nil
	}

	principal, realm, ok, err := c.krbValidator.ValidateRequest(r)
	if err != nil || !ok {
		if !c.krbConf.FallbackToPassword {
			// Try to get a continuation token to advance SPNEGO handshake
			if tok, ok2 := c.krbValidator.ContinueToken(r); ok2 && len(tok) > 0 {
				c.logger.Info("kerberos SPNEGO continuation required; sending response token")
				w.Header().Set("WWW-Authenticate", "Negotiate "+base64.StdEncoding.EncodeToString(tok))
				w.WriteHeader(http.StatusUnauthorized)
				return nil, true, nil
			}
			if err != nil {
				c.logger.Info("kerberos SPNEGO validation error; sending Negotiate challenge", "err", err)
			} else {
				c.logger.Info("kerberos SPNEGO not completed or header missing; sending Negotiate challenge")
			}
			c.krbValidator.Challenge(w)
			return nil, true, nil
		}
		c.logger.Info("kerberos SPNEGO fallback to password enabled; rendering login form")
		return nil, false, nil
	}

	if c.krbConf.ExpectedRealm != "" && !strings.EqualFold(c.krbConf.ExpectedRealm, realm) {
		c.logger.Info("kerberos realm mismatch", "expected", c.krbConf.ExpectedRealm, "actual", realm)
		if !c.krbConf.FallbackToPassword {
			c.krbValidator.Challenge(w)
			return nil, true, nil
		}
		c.logger.Info("kerberos realm mismatch but fallback enabled; rendering login form")
		return nil, false, nil
	}

	mapped := mapPrincipal(principal, realm, c.krbConf.UsernameFromPrincipal)
	c.logger.Info("kerberos principal mapped", "principal", principal, "realm", realm, "mapped_username", mapped)

	var userEntry ldap.Entry
	// Allow test hook override
	if krbLookupUserHook != nil {
		if v, found, herr := krbLookupUserHook(c, mapped); found {
			if herr != nil {
				return nil, true, herr
			}
			userEntry = v
		}
	}

	if userEntry.DN == "" {
		// Reuse existing search logic via do() and userEntry
		err = c.do(ctx, func(conn *ldap.Conn) error {
			entry, found, err := c.userEntry(conn, mapped)
			if err != nil {
				return err
			}
			if !found {
				return fmt.Errorf("user not found for principal")
			}
			userEntry = entry
			return nil
		})
	}
	if err != nil {
		c.logger.Error("kerberos user lookup failed", "principal", principal, "mapped", mapped, "err", err)
		return nil, true, fmt.Errorf("ldap: user lookup failed for kerberos principal %q: %v", principal, err)
	}
	c.logger.Info("kerberos user lookup succeeded", "dn", userEntry.DN)

	ident, err := c.identityFromEntry(userEntry)
	if err != nil {
		c.logger.Info("failed to build identity from LDAP entry after kerberos SPNEGO", "err", err)
		return nil, true, err
	}
	if s.Groups {
		groups, err := c.groups(ctx, userEntry)
		if err != nil {
			c.logger.Info("failed to query groups after kerberos SPNEGO", "err", err)
			return nil, true, fmt.Errorf("ldap: failed to query groups: %v", err)
		}
		ident.Groups = groups
	}

	// No password -> no user bind; do not set ConnectorData unless OfflineAccess requested
	if s.OfflineAccess {
		refresh := refreshData{Username: mapped, Entry: userEntry}
		if data, mErr := json.Marshal(refresh); mErr == nil {
			ident.ConnectorData = data
		}
	}

	c.logger.Info("kerberos SPNEGO authentication succeeded", "username", ident.Username, "email", ident.Email, "groups_count", len(ident.Groups))
	return &ident, true, nil
}
