// Package saml contains login methods for SAML.
package saml

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/beevik/etree"
	xrv "github.com/mattermost/xml-roundtrip-validator"
	"github.com/pkg/errors"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/russellhaering/goxmldsig/etreeutils"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/groups"
)

const (
	bindingRedirect = "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
	bindingPOST     = "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"

	nameIDFormatEmailAddress = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
	nameIDFormatUnspecified  = "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified"
	nameIDFormatX509Subject  = "urn:oasis:names:tc:SAML:1.1:nameid-format:X509SubjectName"
	nameIDFormatWindowsDN    = "urn:oasis:names:tc:SAML:1.1:nameid-format:WindowsDomainQualifiedName"
	nameIDFormatEncrypted    = "urn:oasis:names:tc:SAML:2.0:nameid-format:encrypted"
	nameIDFormatEntity       = "urn:oasis:names:tc:SAML:2.0:nameid-format:entity"
	nameIDFormatKerberos     = "urn:oasis:names:tc:SAML:2.0:nameid-format:kerberos"
	nameIDFormatPersistent   = "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"
	nameIDformatTransient    = "urn:oasis:names:tc:SAML:2.0:nameid-format:transient"

	// top level status codes
	statusCodeSuccess = "urn:oasis:names:tc:SAML:2.0:status:Success"

	// subject confirmation methods
	subjectConfirmationMethodBearer = "urn:oasis:names:tc:SAML:2.0:cm:bearer"

	// allowed clock drift for timestamp validation
	allowedClockDrift = time.Duration(30) * time.Second
)

var (
	nameIDFormats = []string{
		nameIDFormatEmailAddress,
		nameIDFormatUnspecified,
		nameIDFormatX509Subject,
		nameIDFormatWindowsDN,
		nameIDFormatEncrypted,
		nameIDFormatEntity,
		nameIDFormatKerberos,
		nameIDFormatPersistent,
		nameIDformatTransient,
	}
	nameIDFormatLookup = make(map[string]string)

	lookupOnce sync.Once
)

// duration is a JSON duration string (e.g. "1h", "30m"). It exists so
// connector configs can express refresh intervals the same way the server
// config does.
type duration time.Duration

// UnmarshalJSON accepts a Go duration string.
func (d *duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("parse duration: %v", err)
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("parse duration %q: %v", s, err)
	}
	*d = duration(v)
	return nil
}

// MarshalJSON serializes the duration as a string so config round-trips
// (yaml -> json -> storage -> json) keep a stable representation.
func (d duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// Config represents configuration options for the SAML provider.
type Config struct {
	// TODO(ericchiang): A bunch of these fields could be auto-filled if
	// we supported SAML metadata discovery.
	//
	// https://www.oasis-open.org/committees/download.php/35391/sstc-saml-metadata-errata-2.0-wd-04-diff.pdf

	EntityIssuer string `json:"entityIssuer"`
	SSOIssuer    string `json:"ssoIssuer"`
	SSOURL       string `json:"ssoURL"`

	// X509 CA file or raw data to verify XML signatures.
	CA     string `json:"ca"`
	CAData []byte `json:"caData"`

	InsecureSkipSignatureValidation bool `json:"insecureSkipSignatureValidation"`

	// Assertion attribute names to lookup various claims with.
	UsernameAttr string `json:"usernameAttr"`
	EmailAttr    string `json:"emailAttr"`
	GroupsAttr   string `json:"groupsAttr"`
	// If GroupsDelim is supplied the connector assumes groups are returned as a
	// single string instead of multiple attribute values. This delimiter will be
	// used split the groups string.
	GroupsDelim   string   `json:"groupsDelim"`
	AllowedGroups []string `json:"allowedGroups"`
	FilterGroups  bool     `json:"filterGroups"`
	RedirectURI   string   `json:"redirectURI"`

	// MetadataURL is the URL of the IdP's SAML metadata. When set, dex fetches
	// and polls this URL to auto-discover the IdP's SSO endpoint(s), issuer,
	// and signing certificates. Fields explicitly set in the config (ssoURL,
	// ssoIssuer, entityIssuer, ca/caData) take precedence over discovered
	// values. Discovered values only fill in what is not explicitly set.
	MetadataURL string `json:"metadataURL"`

	// MetadataRefreshInterval is how often dex re-fetches MetadataURL.
	// Defaults to 1 hour. Must be >= 1 minute if set.
	MetadataRefreshInterval duration `json:"metadataRefreshInterval"`

	// Requested format of the NameID. The NameID value is is mapped to the ID Token
	// 'sub' claim.
	//
	// This can be an abbreviated form of the full URI with just the last component. For
	// example, if this value is set to "emailAddress" the format will resolve to:
	//
	//		urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress
	//
	// If no value is specified, this value defaults to:
	//
	//		urn:oasis:names:tc:SAML:2.0:nameid-format:persistent
	//
	NameIDPolicyFormat string `json:"nameIDPolicyFormat"`
}

type certStore struct {
	certs []*x509.Certificate
}

func (c certStore) Certificates() (roots []*x509.Certificate, err error) {
	return c.certs, nil
}

// Open validates the config and returns a connector. It does not actually
// validate connectivity with the provider.
func (c *Config) Open(id string, logger *slog.Logger) (connector.Connector, error) {
	logger = logger.With(slog.Group("connector", "type", "saml", "id", id))
	return c.openConnector(logger)
}

func (c *Config) openConnector(logger *slog.Logger) (*provider, error) {
	requiredFields := []struct {
		name, val string
	}{
		{"usernameAttr", c.UsernameAttr},
		{"emailAttr", c.EmailAttr},
		{"redirectURI", c.RedirectURI},
	}
	if c.MetadataURL == "" {
		requiredFields = append(requiredFields, struct{ name, val string }{"ssoURL", c.SSOURL})
	}
	var missing []string
	for _, f := range requiredFields {
		if f.val == "" {
			missing = append(missing, f.name)
		}
	}
	switch len(missing) {
	case 0:
	case 1:
		return nil, fmt.Errorf("missing required field %q", missing[0])
	default:
		return nil, fmt.Errorf("missing required fields %q", missing)
	}

	p := &provider{
		manualSSOURL:              c.SSOURL,
		manualSSOIssuer:           c.SSOIssuer,
		insecureSkipSigValidation: c.InsecureSkipSignatureValidation,
		entityIssuer:              c.EntityIssuer,
		now:                       time.Now,
		usernameAttr:              c.UsernameAttr,
		emailAttr:                 c.EmailAttr,
		groupsAttr:                c.GroupsAttr,
		groupsDelim:               c.GroupsDelim,
		allowedGroups:             c.AllowedGroups,
		filterGroups:              c.FilterGroups,
		redirectURI:               c.RedirectURI,
		logger:                    logger,

		nameIDPolicyFormat: c.NameIDPolicyFormat,
	}
	p.state.ssoURL = c.SSOURL
	p.state.ssoIssuer = c.SSOIssuer

	if c.MetadataURL != "" {
		p.metadataURL = c.MetadataURL
		p.refreshInterval = value(time.Duration(c.MetadataRefreshInterval), time.Hour)
		if p.refreshInterval < time.Minute {
			return nil, fmt.Errorf("metadataRefreshInterval must be at least 1 minute")
		}
		p.httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	if p.nameIDPolicyFormat == "" {
		p.nameIDPolicyFormat = nameIDFormatPersistent
	} else {
		lookupOnce.Do(func() {
			suffix := func(s, sep string) string {
				if i := strings.LastIndex(s, sep); i > 0 {
					return s[i+1:]
				}
				return s
			}
			for _, format := range nameIDFormats {
				nameIDFormatLookup[suffix(format, ":")] = format
				nameIDFormatLookup[format] = format
			}
		})

		if format, ok := nameIDFormatLookup[p.nameIDPolicyFormat]; ok {
			p.nameIDPolicyFormat = format
		} else {
			return nil, fmt.Errorf("invalid nameIDPolicyFormat: %q", p.nameIDPolicyFormat)
		}
	}

	if !c.InsecureSkipSignatureValidation {
		var caData []byte
		if c.CA != "" {
			data, err := os.ReadFile(c.CA)
			if err != nil {
				return nil, fmt.Errorf("read ca file: %v", err)
			}
			caData = data
		} else if c.CAData != nil {
			caData = c.CAData
		}

		certs, err := parsePEMCerts(caData)
		if err != nil {
			return nil, err
		}
		if c.MetadataURL == "" && len(certs) == 0 {
			return nil, errors.New("must provide either 'ca' or 'caData'")
		}
		p.manualCerts = certs
		if len(certs) > 0 {
			p.state.validator = dsig.NewDefaultValidationContext(certStore{certs})
		}
	}
	return p, nil
}

// parsePEMCerts decodes every PEM certificate in data.
func parsePEMCerts(data []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	var block *pem.Block
	for {
		block, data = pem.Decode(data)
		if block == nil {
			data = bytes.TrimSpace(data)
			if len(data) > 0 { // if there's some left, we've been given bad data
				return nil, fmt.Errorf("parse cert: trailing data: %q", string(data))
			}
			break
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse cert: %v", err)
		}
		certs = append(certs, cert)
	}
	return certs, nil
}

func value(val, defaultValue time.Duration) time.Duration {
	if val == 0 {
		return defaultValue
	}
	return val
}

var (
	_ connector.SAMLConnector    = (*provider)(nil)
	_ connector.RefreshConnector = (*provider)(nil)
)

type provider struct {
	entityIssuer string

	// Manually configured values. These always win over discovered metadata
	// values. ssoURL/issuer may be empty when metadata discovery is enabled.
	manualSSOURL              string
	manualSSOIssuer           string
	manualCerts               []*x509.Certificate
	insecureSkipSigValidation bool

	now func() time.Time

	// Attribute mappings
	usernameAttr  string
	emailAttr     string
	groupsAttr    string
	groupsDelim   string
	allowedGroups []string
	filterGroups  bool

	redirectURI string

	nameIDPolicyFormat string

	// Metadata discovery configuration.
	metadataURL     string
	refreshInterval time.Duration
	httpClient      *http.Client
	cancel          context.CancelFunc

	// mu guards state.
	mu    sync.RWMutex
	state providerState

	logger *slog.Logger
}

// providerState is the set of values the SAML request/response paths read. It is
// swapped atomically when metadata is refreshed.
type providerState struct {
	validator *dsig.ValidationContext
	ssoURL    string
	ssoIssuer string
}

func (p *provider) stateSnapshot() providerState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// cachedIdentity stores the identity from SAML assertion for refresh token support.
// Since SAML has no native refresh mechanism, we cache the identity obtained during
// the initial authentication and return it on subsequent refresh requests.
type cachedIdentity struct {
	UserID            string   `json:"userId"`
	Username          string   `json:"username"`
	PreferredUsername string   `json:"preferredUsername"`
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"emailVerified"`
	Groups            []string `json:"groups,omitempty"`
}

// marshalCachedIdentity serializes the identity into ConnectorData for refresh token support.
func marshalCachedIdentity(ident connector.Identity) (connector.Identity, error) {
	ci := cachedIdentity{
		UserID:            ident.UserID,
		Username:          ident.Username,
		PreferredUsername: ident.PreferredUsername,
		Email:             ident.Email,
		EmailVerified:     ident.EmailVerified,
		Groups:            ident.Groups,
	}
	connectorData, err := json.Marshal(ci)
	if err != nil {
		return ident, fmt.Errorf("saml: failed to marshal cached identity: %v", err)
	}
	ident.ConnectorData = connectorData
	return ident, nil
}

func (p *provider) POSTData(s connector.Scopes, id string) (action, value string, err error) {
	st := p.stateSnapshot()
	r := &authnRequest{
		ProtocolBinding: bindingPOST,
		ID:              id,
		IssueInstant:    xmlTime(p.now()),
		Destination:     st.ssoURL,
		NameIDPolicy: &nameIDPolicy{
			AllowCreate: true,
			Format:      p.nameIDPolicyFormat,
		},
		AssertionConsumerServiceURL: p.redirectURI,
	}
	if p.entityIssuer != "" {
		// Issuer for the request is optional. For example, okta always ignores
		// this value.
		r.Issuer = &issuer{Issuer: p.entityIssuer}
	}

	data, err := xml.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("marshal authn request: %v", err)
	}

	// See: https://docs.oasis-open.org/security/saml/v2.0/saml-bindings-2.0-os.pdf
	// "3.5.4 Message Encoding"
	return st.ssoURL, base64.StdEncoding.EncodeToString(data), nil
}

// HandlePOST interprets a request from a SAML provider attempting to verify a
// user's identity.
//
// The steps taken are:
//
// * Validate XML document does not contain malicious inputs.
// * Verify signature on XML document (or verify sig on assertion elements).
// * Verify various parts of the Assertion element. Conditions, audience, etc.
// * Map the Assertion's attribute elements to user info.
func (p *provider) HandlePOST(s connector.Scopes, samlResponse, inResponseTo string) (ident connector.Identity, err error) {
	st := p.stateSnapshot()
	rawResp, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		return ident, fmt.Errorf("decode response: %v", err)
	}

	byteReader := bytes.NewReader(rawResp)
	if xrvErr := xrv.Validate(byteReader); xrvErr != nil {
		return ident, errors.Wrap(xrvErr, "validating XML response")
	}

	// Root element is allowed to not be signed if the Assertion element is.
	rootElementSigned := true
	if st.validator != nil {
		rawResp, rootElementSigned, err = verifyResponseSig(st.validator, rawResp)
		if err != nil {
			return ident, fmt.Errorf("verify signature: %v", err)
		}
	}

	var resp response
	if err := xml.Unmarshal(rawResp, &resp); err != nil {
		return ident, fmt.Errorf("unmarshal response: %v", err)
	}

	// If the root element isn't signed, there's no reason to inspect these
	// elements. They're not verified.
	if rootElementSigned {
		if st.ssoIssuer != "" && resp.Issuer != nil && resp.Issuer.Issuer != st.ssoIssuer {
			return ident, fmt.Errorf("expected Issuer value %s, got %s", st.ssoIssuer, resp.Issuer.Issuer)
		}

		// Verify InResponseTo value matches the expected ID associated with
		// the RelayState.
		if resp.InResponseTo != inResponseTo {
			return ident, fmt.Errorf("expected InResponseTo value %s, got %s", inResponseTo, resp.InResponseTo)
		}

		// Destination is optional.
		if resp.Destination != "" && resp.Destination != p.redirectURI {
			return ident, fmt.Errorf("expected destination %q got %q", p.redirectURI, resp.Destination)
		}

		// Status is a required element.
		if resp.Status == nil {
			return ident, fmt.Errorf("response did not contain a Status element")
		}

		if err = p.validateStatus(resp.Status); err != nil {
			return ident, err
		}
	}

	assertion := resp.Assertion
	if assertion == nil {
		return ident, fmt.Errorf("response did not contain an assertion")
	}

	// Subject is usually optional, but we need it for the user ID, so complain
	// if it's not present.
	subject := assertion.Subject
	if subject == nil {
		return ident, fmt.Errorf("response did not contain a subject")
	}

	// Validate that the response is to the request we originally sent.
	if err = p.validateSubject(subject, inResponseTo); err != nil {
		return ident, err
	}

	// Conditions element is optional, but must be validated if present.
	if assertion.Conditions != nil {
		// Validate that dex is the intended audience of this response.
		if err = p.validateConditions(assertion.Conditions); err != nil {
			return ident, err
		}
	}

	switch {
	case subject.NameID != nil:
		if ident.UserID = subject.NameID.Value; ident.UserID == "" {
			return ident, fmt.Errorf("element NameID does not contain a value")
		}
	default:
		return ident, fmt.Errorf("subject does not contain an NameID element")
	}

	// After verifying the assertion, map data in the attribute statements to
	// various user info.
	attributes := assertion.AttributeStatement
	if attributes == nil {
		return ident, fmt.Errorf("response did not contain a AttributeStatement")
	}

	// Log the actual attributes we got back from the server. This helps debug
	// configuration errors on the server side, where the SAML server doesn't
	// send us the correct attributes.
	p.logger.Info("parsed and verified saml response attributes", "attributes", attributes)

	// Grab the email.
	if ident.Email, _ = attributes.get(p.emailAttr); ident.Email == "" {
		return ident, fmt.Errorf("no attribute with name %q: %s", p.emailAttr, attributes.names())
	}
	// TODO(ericchiang): Does SAML have an email_verified equivalent?
	ident.EmailVerified = true

	// Grab the username.
	if ident.Username, _ = attributes.get(p.usernameAttr); ident.Username == "" {
		return ident, fmt.Errorf("no attribute with name %q: %s", p.usernameAttr, attributes.names())
	}

	if len(p.allowedGroups) == 0 && (!s.Groups || p.groupsAttr == "") {
		// Groups not requested or not configured. We're done.
		return marshalCachedIdentity(ident)
	}

	if len(p.allowedGroups) > 0 && (!s.Groups || p.groupsAttr == "") {
		// allowedGroups set but no groups or groupsAttr. Disallowing.
		return ident, fmt.Errorf("user not a member of allowed groups")
	}

	// Grab the groups.
	if p.groupsDelim != "" {
		groupsStr, ok := attributes.get(p.groupsAttr)
		if !ok {
			return ident, fmt.Errorf("no attribute with name %q: %s", p.groupsAttr, attributes.names())
		}
		// TODO(ericchiang): Do we need to further trim whitespace?
		ident.Groups = strings.Split(groupsStr, p.groupsDelim)
	} else {
		groups, ok := attributes.all(p.groupsAttr)
		if !ok {
			return ident, fmt.Errorf("no attribute with name %q: %s", p.groupsAttr, attributes.names())
		}
		ident.Groups = groups
	}

	if len(p.allowedGroups) == 0 {
		// No allowed groups set, just return the ident
		return marshalCachedIdentity(ident)
	}

	// Look for membership in one of the allowed groups
	groupMatches := groups.Filter(ident.Groups, p.allowedGroups)

	if len(groupMatches) == 0 {
		// No group membership matches found, disallowing
		return ident, fmt.Errorf("user not a member of allowed groups")
	}

	if p.filterGroups {
		ident.Groups = groupMatches
	}

	// Otherwise, we're good
	return marshalCachedIdentity(ident)
}

// Refresh implements connector.RefreshConnector.
// Since SAML has no native refresh mechanism, this method returns the cached
// identity from the initial SAML assertion stored in ConnectorData.
func (p *provider) Refresh(ctx context.Context, s connector.Scopes, ident connector.Identity) (connector.Identity, error) {
	if len(ident.ConnectorData) == 0 {
		return ident, fmt.Errorf("saml: no connector data available for refresh")
	}

	var ci cachedIdentity
	if err := json.Unmarshal(ident.ConnectorData, &ci); err != nil {
		return ident, fmt.Errorf("saml: failed to unmarshal cached identity: %v", err)
	}

	ident.UserID = ci.UserID
	ident.Username = ci.Username
	ident.PreferredUsername = ci.PreferredUsername
	ident.Email = ci.Email
	ident.EmailVerified = ci.EmailVerified

	// Only populate groups if the client requested the groups scope.
	if s.Groups {
		ident.Groups = ci.Groups
	} else {
		ident.Groups = nil
	}

	return ident, nil
}

// validateStatus verifies that the response has a good status code or
// formats a human readable error based on the bad status.
func (p *provider) validateStatus(status *status) error {
	// StatusCode is mandatory in the Status type
	statusCode := status.StatusCode
	if statusCode == nil {
		return fmt.Errorf("response did not contain a StatusCode")
	}

	if statusCode.Value != statusCodeSuccess {
		parts := strings.Split(statusCode.Value, ":")
		lastPart := parts[len(parts)-1]
		errorMessage := fmt.Sprintf("status code of the Response was not Success, was %q", lastPart)
		statusMessage := status.StatusMessage
		if statusMessage != nil && statusMessage.Value != "" {
			errorMessage += " -> " + statusMessage.Value
		}
		return errors.New(errorMessage)
	}
	return nil
}

// validateSubject ensures the response is to the request we expect.
//
// This is described in the spec "Profiles for the OASIS Security
// Assertion Markup Language" in section 3.3 Bearer.
// see https://www.oasis-open.org/committees/download.php/35389/sstc-saml-profiles-errata-2.0-wd-06-diff.pdf
//
// Some of these fields are optional, but we're going to be strict here since
// we have no other way of guaranteeing that this is actually the response to
// the request we expect.
func (p *provider) validateSubject(subject *subject, inResponseTo string) error {
	// Optional according to the spec, but again, we're going to be strict here.
	if len(subject.SubjectConfirmations) == 0 {
		return fmt.Errorf("subject contained no SubjectConfirmations")
	}

	errs := make([]error, 0, len(subject.SubjectConfirmations))
	// One of these must match our assumptions, not all.
	for _, c := range subject.SubjectConfirmations {
		err := func() error {
			if c.Method != subjectConfirmationMethodBearer {
				return fmt.Errorf("unexpected subject confirmation method: %v", c.Method)
			}

			data := c.SubjectConfirmationData
			if data == nil {
				return fmt.Errorf("no SubjectConfirmationData field found in SubjectConfirmation")
			}
			if data.InResponseTo != inResponseTo {
				return fmt.Errorf("expected SubjectConfirmationData InResponseTo value %q, got %q", inResponseTo, data.InResponseTo)
			}

			notBefore := time.Time(data.NotBefore)
			notOnOrAfter := time.Time(data.NotOnOrAfter)
			now := p.now()
			if !notBefore.IsZero() && before(now, notBefore) {
				return fmt.Errorf("at %s got response that cannot be processed before %s", now, notBefore)
			}
			if !notOnOrAfter.IsZero() && after(now, notOnOrAfter) {
				return fmt.Errorf("at %s got response that cannot be processed because it expired at %s", now, notOnOrAfter)
			}
			if r := data.Recipient; r != "" && r != p.redirectURI {
				return fmt.Errorf("expected Recipient %q got %q", p.redirectURI, r)
			}
			return nil
		}()
		if err == nil {
			// Subject is valid.
			return nil
		}
		errs = append(errs, err)
	}

	if len(errs) == 1 {
		return fmt.Errorf("failed to validate subject confirmation: %v", errs[0])
	}
	return fmt.Errorf("failed to validate subject confirmation: %v", errs)
}

// validateConditions ensures that dex is the intended audience
// for the request, and not another service provider.
//
// See: https://docs.oasis-open.org/security/saml/v2.0/saml-core-2.0-os.pdf
// "2.3.3 Element <Assertion>"
func (p *provider) validateConditions(conditions *conditions) error {
	// Ensure the conditions haven't expired.
	now := p.now()
	notBefore := time.Time(conditions.NotBefore)
	if !notBefore.IsZero() && before(now, notBefore) {
		return fmt.Errorf("at %s got response that cannot be processed before %s", now, notBefore)
	}

	notOnOrAfter := time.Time(conditions.NotOnOrAfter)
	if !notOnOrAfter.IsZero() && after(now, notOnOrAfter) {
		return fmt.Errorf("at %s got response that cannot be processed because it expired at %s", now, notOnOrAfter)
	}

	// Sometimes, dex's issuer string can be different than the redirect URI,
	// but if dex's issuer isn't explicitly provided assume the redirect URI.
	expAud := p.entityIssuer
	if expAud == "" {
		expAud = p.redirectURI
	}

	// AudienceRestriction elements indicate the intended audience(s) of an
	// assertion. If dex isn't in these audiences, reject the assertion.
	//
	// Note that if there are multiple AudienceRestriction elements, each must
	// individually contain dex in their audience list.
	for _, r := range conditions.AudienceRestriction {
		values := make([]string, len(r.Audiences))
		issuerInAudiences := false
		for i, aud := range r.Audiences {
			if aud.Value == expAud {
				issuerInAudiences = true
				break
			}
			values[i] = aud.Value
		}

		if !issuerInAudiences {
			return fmt.Errorf("required audience %s was not in Response audiences %s", expAud, values)
		}
	}
	return nil
}

// verifyResponseSig attempts to verify the signature of a SAML response or
// the assertion.
//
// If the root element is properly signed, this method returns it.
//
// The SAML spec requires supporting responses where the root element is
// unverified, but the sub <Assertion> elements are signed. In these cases,
// this method returns rootVerified=false to indicate that the <Assertion>
// elements should be trusted, but all other elements MUST be ignored.
//
// Note: we still don't support multiple <Assertion> tags. If there are
// multiple present this code will only process the first.
func verifyResponseSig(validator *dsig.ValidationContext, data []byte) (signed []byte, rootVerified bool, err error) {
	doc := etree.NewDocument()
	if err = doc.ReadFromBytes(data); err != nil {
		return nil, false, fmt.Errorf("parse document: %v", err)
	}

	response := doc.Root()
	if response == nil {
		return nil, false, fmt.Errorf("parse document: empty root")
	}
	transformedResponse, err := validator.Validate(response)
	if err == nil {
		// Root element is verified, return it.
		doc.SetRoot(transformedResponse)
		signed, err = doc.WriteToBytes()
		return signed, true, err
	}

	// Ensures xmlns are copied down to the assertion element when they are defined in the root
	//
	// TODO: Only select from child elements of the root.
	assertion, err := etreeutils.NSSelectOne(response, "urn:oasis:names:tc:SAML:2.0:assertion", "Assertion")
	if err != nil || assertion == nil {
		return nil, false, fmt.Errorf("response does not contain an Assertion element")
	}
	transformedAssertion, err := validator.Validate(assertion)
	if err != nil {
		return nil, false, fmt.Errorf("response does not contain a valid signature element: %v", err)
	}

	// Verified an assertion but not the response. Can't trust any child elements,
	// except the assertion. Remove them all.
	for _, el := range response.ChildElements() {
		response.RemoveChild(el)
	}

	// We still return the full <Response> element, even though it's unverified
	// because the <Assertion> element is not a valid XML document on its own.
	// It still requires the root element to define things like namespaces.
	response.AddChild(transformedAssertion)
	signed, err = doc.WriteToBytes()
	return signed, false, err
}

// before determines if a given time is before the current time, with an
// allowed clock drift.
func before(now, notBefore time.Time) bool {
	return now.Add(allowedClockDrift).Before(notBefore)
}

// after determines if a given time is after the current time, with an
// allowed clock drift.
func after(now, notOnOrAfter time.Time) bool {
	return now.After(notOnOrAfter.Add(allowedClockDrift))
}

// applyMetadata merges discovered IdP metadata into the provider state. Manually
// configured values always win; discovered values only fill unset fields. The
// validator uses the union of manual and discovered signing certs.
func (p *provider) applyMetadata(meta *IdPMetadata) error {
	certs := append([]*x509.Certificate(nil), p.manualCerts...)
	certs = append(certs, meta.SigningCerts...)

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.insecureSkipSigValidation {
		if len(certs) == 0 {
			return fmt.Errorf("saml: no signing certificates available from metadata")
		}
		p.state.validator = dsig.NewDefaultValidationContext(certStore{certs})
	}

	if p.manualSSOURL != "" {
		p.state.ssoURL = p.manualSSOURL
	} else if url, ok := meta.ssoEndpoint(bindingPOST); ok && url != "" {
		p.state.ssoURL = url
	} else if url, ok := meta.ssoEndpoint(bindingRedirect); ok && url != "" {
		p.state.ssoURL = url
	} else {
		return fmt.Errorf("saml: no SSO endpoint in metadata and ssoURL is not set")
	}

	if p.manualSSOIssuer != "" {
		p.state.ssoIssuer = p.manualSSOIssuer
	} else if meta.EntityID != "" {
		p.state.ssoIssuer = meta.EntityID
	}

	return nil
}

// fetchMetadata retrieves the IdP metadata document from the metadata URL.
func (p *provider) fetchMetadata() ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, p.metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("saml: build metadata request: %v", err)
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("saml: fetch metadata: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("saml: fetch metadata: unexpected status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("saml: read metadata: %v", err)
	}
	return data, nil
}

// refresh fetches and applies the latest IdP metadata.
func (p *provider) refresh() error {
	data, err := p.fetchMetadata()
	if err != nil {
		return err
	}
	meta, err := parseMetadata(data)
	if err != nil {
		return err
	}
	return p.applyMetadata(meta)
}

// Start implements connector.LifecycleConnector. It performs an immediate
// metadata fetch and then polls on MetadataRefreshInterval until Close is
// called or ctx is cancelled. The first fetch is critical: when it fails and
// no manual certificates are configured, Start returns an error.
func (p *provider) Start(ctx context.Context) error {
	if p.metadataURL == "" {
		return nil
	}

	if err := p.refresh(); err != nil {
		if len(p.manualCerts) > 0 || p.insecureSkipSigValidation {
			p.logger.Warn("failed to fetch SAML metadata; using manual configuration", "err", err)
		} else {
			return fmt.Errorf("saml: failed to fetch metadata: %v", err)
		}
	}

	pollCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	go func() {
		defer cancel()
		ticker := time.NewTicker(p.refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-pollCtx.Done():
				return
			case <-ticker.C:
				if err := p.refresh(); err != nil {
					p.logger.Warn("failed to refresh SAML metadata; keeping last known configuration", "err", err)
					continue
				}
				p.logger.Info("refreshed SAML metadata")
			}
		}
	}()
	return nil
}

// Close stops the metadata polling goroutine, if any.
func (p *provider) Close() {
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
}
