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

	"github.com/go-ldap/ldap/v3"
	"github.com/jcmturner/gofork/encoding/asn1"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/gssapi"
	"github.com/jcmturner/gokrb5/v8/keytab"
	"github.com/jcmturner/gokrb5/v8/service"
	"github.com/jcmturner/gokrb5/v8/spnego"
	"github.com/jcmturner/gokrb5/v8/types"

	"github.com/dexidp/dex/connector"
)

// KerberosResult carries the outcome of a single SPNEGO validation attempt.
//
// Exactly one of the following states holds:
//   - Authenticated:        OK == true, Principal/Realm set.
//   - Continuation needed:  len(ContinueToken) > 0.
//   - No/invalid header:    OK == false, Err == nil, no ContinueToken.
//   - Validation error:     Err != nil.
type KerberosResult struct {
	Principal     string
	Realm         string
	OK            bool
	Err           error
	ContinueToken []byte
}

// KerberosValidator abstracts SPNEGO validation for unit-testing.
//
// Implementations MUST call the underlying GSSAPI AcceptSecContext at most
// once per request to avoid replay detection (KRB_AP_ERR_REPEAT) on valid
// AP-REQ tokens.
type KerberosValidator interface {
	ValidateRequest(r *http.Request) KerberosResult
}

// ctxCredentialsKey is the context key used by gokrb5 to store credentials.
//
// gokrb5/v8/spnego declares this as an untyped string const, which is passed
// into context.WithValue and therefore stored with concrete type `string`.
// To retrieve credentials the key MUST be a plain string (not a typed alias),
// otherwise context.Value returns nil because keys are compared by both type
// and value.
//
//nolint:revive,staticcheck // gokrb5 stores credentials under a plain string key; we must match it.
const ctxCredentialsKey = "github.com/jcmturner/gokrb5/v8/ctxCredentials"

// spnegoIncompleteKRB5B64 is a pre-encoded SPNEGO NegTokenResp with
// AcceptIncomplete status and KRB5 mech, matching what gokrb5 sends when the
// handshake needs to continue. Used as the response body for the second step
// of a multi-leg SPNEGO exchange only (never as the initial challenge).
const spnegoIncompleteKRB5B64 = "oRQwEqADCgEBoQsGCSqGSIb3EgECAg=="

// writeNegotiateChallenge writes a bare RFC 4559 Negotiate challenge.
// This is the correct response when the client has not yet sent an
// Authorization: Negotiate header.
func writeNegotiateChallenge(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Negotiate")
	w.WriteHeader(http.StatusUnauthorized)
}

// writeNegotiateContinuation writes a 401 with a SPNEGO continuation token.
// Used only when the client already initiated SPNEGO and gokrb5 reported
// StatusContinueNeeded.
func writeNegotiateContinuation(w http.ResponseWriter, token []byte) {
	w.Header().Set("WWW-Authenticate", "Negotiate "+base64.StdEncoding.EncodeToString(token))
	w.WriteHeader(http.StatusUnauthorized)
}

// mapPrincipal maps a Kerberos principal to LDAP username per configuration.
// Supported mappings:
//   - "localpart" / "samaccountname": extracts username before @ (default)
//   - "userprincipalname": uses full principal as-is
func mapPrincipal(principal, mapping string) string {
	localpart := principal
	if i := strings.IndexByte(principal, '@'); i >= 0 {
		localpart = principal[:i]
	}

	switch strings.ToLower(mapping) {
	case "userprincipalname":
		return strings.ToLower(principal)
	case "localpart", "samaccountname":
		return strings.ToLower(localpart)
	default:
		return strings.ToLower(localpart)
	}
}

// gokrb5Validator is the default KerberosValidator backed by jcmturner/gokrb5.
type gokrb5Validator struct {
	kt     *keytab.Keytab
	logger *slog.Logger
}

func newGokrb5ValidatorWithLogger(keytabPath string, logger *slog.Logger) (KerberosValidator, error) {
	fi, err := os.Stat(keytabPath)
	if err != nil {
		return nil, fmt.Errorf("keytab file not found: %w", err)
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("keytab path is a directory: %s", keytabPath)
	}
	kt, err := keytab.Load(keytabPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load keytab: %w", err)
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &gokrb5Validator{kt: kt, logger: logger}, nil
}

// parseNegotiateHeader extracts and decodes the SPNEGO token from the
// Authorization header. Returns (token, true) on success, (nil, false) when
// the header is absent, malformed or not a valid SPNEGO/KRB5 token.
func (v *gokrb5Validator) parseNegotiateHeader(r *http.Request) (*spnego.SPNEGOToken, bool) {
	h := r.Header.Get("Authorization")
	if h == "" || !strings.HasPrefix(h, "Negotiate ") {
		return nil, false
	}
	b64 := strings.TrimSpace(h[len("Negotiate "):])
	if b64 == "" {
		return nil, false
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		v.logger.Info("kerberos: invalid base64 in Authorization", "err", err)
		return nil, false
	}
	var tok spnego.SPNEGOToken
	if err := tok.Unmarshal(data); err != nil {
		// Fall back to raw KRB5 token, wrap as SPNEGO NegTokenInit.
		var k5 spnego.KRB5Token
		if k5.Unmarshal(data) != nil {
			v.logger.Info("kerberos: failed to unmarshal SPNEGO/KRB5 token", "err", err)
			return nil, false
		}
		tok.Init = true
		tok.NegTokenInit = spnego.NegTokenInit{
			MechTypes:      []asn1.ObjectIdentifier{k5.OID},
			MechTokenBytes: data,
		}
	}
	return &tok, true
}

// createSPNEGOService creates a SPNEGO service, binding the client address
// when it can be parsed from RemoteAddr (required for address-bound tickets).
func (v *gokrb5Validator) createSPNEGOService(r *http.Request) *spnego.SPNEGO {
	if ha, err := types.GetHostAddress(r.RemoteAddr); err == nil {
		return spnego.SPNEGOService(v.kt, service.ClientAddress(ha), service.DecodePAC(false))
	}
	v.logger.Info("kerberos: cannot parse client address", "remote", r.RemoteAddr)
	return spnego.SPNEGOService(v.kt, service.DecodePAC(false))
}

// ValidateRequest performs a single AcceptSecContext call and translates the
// GSSAPI status into a KerberosResult. It never retries the same AP-REQ.
func (v *gokrb5Validator) ValidateRequest(r *http.Request) KerberosResult {
	tok, ok := v.parseNegotiateHeader(r)
	if !ok {
		v.logger.Info("kerberos: missing or invalid Negotiate header", "path", r.URL.Path)
		return KerberosResult{}
	}

	sp := v.createSPNEGOService(r)
	authed, ctx, status := sp.AcceptSecContext(tok)

	switch status.Code {
	case gssapi.StatusComplete:
		// Handled below.
	case gssapi.StatusContinueNeeded:
		v.logger.Info("kerberos: continuation needed", "message", status.Message)
		tokb, err := base64.StdEncoding.DecodeString(spnegoIncompleteKRB5B64)
		if err != nil {
			return KerberosResult{Err: fmt.Errorf("decode continuation token: %w", err)}
		}
		return KerberosResult{ContinueToken: tokb}
	default:
		v.logger.Info("kerberos: AcceptSecContext rejected", "code", status.Code, "message", status.Message)
		return KerberosResult{}
	}

	if !authed || ctx == nil {
		v.logger.Info("kerberos: not authenticated or no context")
		return KerberosResult{}
	}

	id, _ := ctx.Value(ctxCredentialsKey).(*credentials.Credentials)
	if id == nil {
		v.logger.Info("kerberos: credentials missing in context")
		return KerberosResult{Err: fmt.Errorf("no credentials in context")}
	}
	return KerberosResult{Principal: id.UserName(), Realm: id.Domain(), OK: true}
}

// LDAP connector SPNEGO integration

// TrySPNEGO attempts Kerberos authentication on the current request. Returns
// handled=true when the response has been fully written (either a challenge,
// a continuation token or a terminal error) and the caller MUST NOT render
// the password form. handled=false means the caller should continue with the
// default login flow.
func (c *ldapConnector) TrySPNEGO(ctx context.Context, s connector.Scopes, w http.ResponseWriter, r *http.Request) (*connector.Identity, connector.Handled, error) {
	if !c.krbEnabled || c.krbValidator == nil {
		return nil, false, nil
	}

	res := c.krbValidator.ValidateRequest(r)

	if !res.OK {
		if c.krbConf.FallbackToPassword {
			c.logger.Info("kerberos SPNEGO not completed; falling back to password form")
			return nil, false, nil
		}

		switch {
		case len(res.ContinueToken) > 0:
			c.logger.Info("kerberos SPNEGO continuation required; sending response token")
			writeNegotiateContinuation(w, res.ContinueToken)
		case res.Err != nil:
			c.logger.Info("kerberos SPNEGO validation error; sending Negotiate challenge", "err", res.Err)
			writeNegotiateChallenge(w)
		default:
			c.logger.Info("kerberos SPNEGO header missing or invalid; sending Negotiate challenge")
			writeNegotiateChallenge(w)
		}
		return nil, true, nil
	}

	if c.krbConf.ExpectedRealm != "" && !strings.EqualFold(c.krbConf.ExpectedRealm, res.Realm) {
		c.logger.Info("kerberos realm mismatch", "expected", c.krbConf.ExpectedRealm, "actual", res.Realm)
		if c.krbConf.FallbackToPassword {
			c.logger.Info("kerberos realm mismatch but fallback enabled; rendering login form")
			return nil, false, nil
		}
		writeNegotiateChallenge(w)
		return nil, true, nil
	}

	mapped := mapPrincipal(res.Principal, c.krbConf.UsernameFromPrincipal)
	c.logger.Info("kerberos principal mapped",
		"principal", res.Principal, "realm", res.Realm, "mapped_username", mapped)

	userEntry, err := c.lookupKerberosUser(ctx, mapped)
	if err != nil {
		c.logger.Error("kerberos user lookup failed", "principal", res.Principal, "mapped", mapped, "err", err)
		return nil, true, fmt.Errorf("ldap: user lookup failed for kerberos principal %q: %v", res.Principal, err)
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

	// No password means no user bind happened; only populate ConnectorData when
	// OfflineAccess was requested so refresh can query groups later.
	if s.OfflineAccess {
		refresh := refreshData{Username: mapped, Entry: userEntry}
		if data, mErr := json.Marshal(refresh); mErr == nil {
			ident.ConnectorData = data
		}
	}

	c.logger.Info("kerberos SPNEGO authentication succeeded",
		"username", ident.Username, "email", ident.Email, "groups_count", len(ident.Groups))
	return &ident, true, nil
}

// lookupKerberosUser resolves an LDAP user entry by username. Tests may
// override the lookup via krbLookupUserHook.
func (c *ldapConnector) lookupKerberosUser(ctx context.Context, username string) (ldap.Entry, error) {
	if c.krbLookupUserHook != nil {
		if entry, found, err := c.krbLookupUserHook(c, username); found {
			return entry, err
		}
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
