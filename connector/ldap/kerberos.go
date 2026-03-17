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
// Supported mappings:
//   - "localpart" / "samaccountname": extracts username before @ (default)
//   - "userprincipalname": uses full principal as-is
func mapPrincipal(principal, mapping string) string {
	// Extract localpart (before @) from principal
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

// gokrb5 implementation of KerberosValidator

// ctxCredentialsKeyType is the type for gokrb5 context key.
// gokrb5 uses a string constant as context key for storing credentials.
type ctxCredentialsKeyType string

// ctxCredentialsKey is the context key used by gokrb5 to store credentials.
// This must match the exact string used in gokrb5/v8/spnego package.
const ctxCredentialsKey ctxCredentialsKeyType = "github.com/jcmturner/gokrb5/v8/ctxCredentials"

// SPNEGO NegTokenResp (AcceptIncomplete + KRB5 mech) base64 payload used by gokrb5's HTTP server
// to prompt the client to continue the handshake.
const spnegoIncompleteKRB5B64 = "oRQwEqADCgEBoQsGCSqGSIb3EgECAg=="

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

// parseNegotiateHeader extracts and decodes the SPNEGO token from Authorization header.
// Returns (token, ok). If ok is false, the header is missing, malformed, or not Negotiate.
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
		if v.logger != nil {
			v.logger.Info("kerberos: invalid base64 in Authorization", "err", err)
		}
		return nil, false
	}
	var tok spnego.SPNEGOToken
	if err := tok.Unmarshal(data); err != nil {
		// Try raw KRB5 token and wrap it as SPNEGO
		var k5 spnego.KRB5Token
		if k5.Unmarshal(data) != nil {
			if v.logger != nil {
				v.logger.Info("kerberos: failed to unmarshal SPNEGO/KRB5 token", "err", err)
			}
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

// createSPNEGOService creates a SPNEGO service with optional client address binding.
func (v *gokrb5Validator) createSPNEGOService(r *http.Request) *spnego.SPNEGO {
	if ha, err := types.GetHostAddress(r.RemoteAddr); err == nil {
		return spnego.SPNEGOService(v.kt, service.ClientAddress(ha), service.DecodePAC(false))
	}
	if v.logger != nil {
		v.logger.Info("kerberos: cannot parse client address", "remote", r.RemoteAddr)
	}
	return spnego.SPNEGOService(v.kt, service.DecodePAC(false))
}

func (v *gokrb5Validator) ValidateRequest(r *http.Request) (string, string, bool, error) {
	tok, ok := v.parseNegotiateHeader(r)
	if !ok {
		if v.logger != nil {
			v.logger.Info("kerberos: missing or invalid Negotiate header", "path", r.URL.Path)
		}
		return "", "", false, nil
	}

	sp := v.createSPNEGOService(r)
	authed, ctx, status := sp.AcceptSecContext(tok)
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
	tok, ok := v.parseNegotiateHeader(r)
	if !ok {
		// No valid token; return incomplete response to prompt client
		return v.incompleteResponse()
	}

	sp := v.createSPNEGOService(r)
	_, ctx, status := sp.AcceptSecContext(tok)
	if status.Code != gssapi.StatusContinueNeeded || ctx == nil {
		if v.logger != nil {
			v.logger.Info("kerberos: no continuation required", "code", status.Code, "message", status.Message)
		}
		return nil, false
	}
	return v.incompleteResponse()
}

// incompleteResponse returns the standard NegTokenResp to prompt client continuation.
func (v *gokrb5Validator) incompleteResponse() ([]byte, bool) {
	tokb, err := base64.StdEncoding.DecodeString(spnegoIncompleteKRB5B64)
	if err != nil {
		return nil, false
	}
	if v.logger != nil {
		v.logger.Info("kerberos: sending incomplete KRB5 response for continuation")
	}
	return tokb, true
}

// LDAP connector SPNEGO integration

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

	mapped := mapPrincipal(principal, c.krbConf.UsernameFromPrincipal)
	c.logger.Info("kerberos principal mapped", "principal", principal, "realm", realm, "mapped_username", mapped)

	var userEntry ldap.Entry
	// Allow test hook override
	if c.krbLookupUserHook != nil {
		if v, found, herr := c.krbLookupUserHook(c, mapped); found {
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
