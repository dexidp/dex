// Package saml contains login methods for SAML.
package saml

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/russellhaering/goxmldsig/etreeutils"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/groups"
	"github.com/dexidp/dex/pkg/log"
)

// nolint
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
)

func init() {
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
	RedirectURI   string   `json:"redirectURI"`

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
func (c *Config) Open(id string, logger log.Logger) (connector.Connector, error) {
	return c.openConnector(logger)
}

func (c *Config) openConnector(logger log.Logger) (*provider, error) {
	requiredFields := []struct {
		name, val string
	}{
		{"ssoURL", c.SSOURL},
		{"usernameAttr", c.UsernameAttr},
		{"emailAttr", c.EmailAttr},
		{"redirectURI", c.RedirectURI},
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
		entityIssuer:  c.EntityIssuer,
		ssoIssuer:     c.SSOIssuer,
		ssoURL:        c.SSOURL,
		now:           time.Now,
		usernameAttr:  c.UsernameAttr,
		emailAttr:     c.EmailAttr,
		groupsAttr:    c.GroupsAttr,
		groupsDelim:   c.GroupsDelim,
		allowedGroups: c.AllowedGroups,
		redirectURI:   c.RedirectURI,
		logger:        logger,

		nameIDPolicyFormat: c.NameIDPolicyFormat,
	}

	if p.nameIDPolicyFormat == "" {
		p.nameIDPolicyFormat = nameIDFormatPersistent
	} else {
		if format, ok := nameIDFormatLookup[p.nameIDPolicyFormat]; ok {
			p.nameIDPolicyFormat = format
		} else {
			return nil, fmt.Errorf("invalid nameIDPolicyFormat: %q", p.nameIDPolicyFormat)
		}
	}

	if !c.InsecureSkipSignatureValidation {
		if (c.CA == "") == (c.CAData == nil) {
			return nil, errors.New("must provide either 'ca' or 'caData'")
		}

		var caData []byte
		if c.CA != "" {
			data, err := ioutil.ReadFile(c.CA)
			if err != nil {
				return nil, fmt.Errorf("read ca file: %v", err)
			}
			caData = data
		} else {
			caData = c.CAData
		}

		var (
			certs []*x509.Certificate
			block *pem.Block
		)
		for {
			block, caData = pem.Decode(caData)
			if block == nil {
				caData = bytes.TrimSpace(caData)
				if len(caData) > 0 { // if there's some left, we've been given bad caData
					return nil, fmt.Errorf("parse cert: trailing data: %q", string(caData))
				}
				break
			}
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parse cert: %v", err)
			}
			certs = append(certs, cert)
		}
		if len(certs) == 0 {
			return nil, errors.New("no certificates found in ca data")
		}
		p.validator = dsig.NewDefaultValidationContext(certStore{certs})
	}
	return p, nil
}

type provider struct {
	entityIssuer string
	ssoIssuer    string
	ssoURL       string

	now func() time.Time

	// If nil, don't do signature validation.
	validator *dsig.ValidationContext

	// Attribute mappings
	usernameAttr  string
	emailAttr     string
	groupsAttr    string
	groupsDelim   string
	allowedGroups []string

	redirectURI string

	nameIDPolicyFormat string

	logger log.Logger
}

func (p *provider) POSTData(s connector.Scopes, id string) (action, value string, err error) {
	r := &authnRequest{
		ProtocolBinding: bindingPOST,
		ID:              id,
		IssueInstant:    xmlTime(p.now()),
		Destination:     p.ssoURL,
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
	return p.ssoURL, base64.StdEncoding.EncodeToString(data), nil
}

// HandlePOST interprets a request from a SAML provider attempting to verify a
// user's identity.
//
// The steps taken are:
//
// * Verify signature on XML document (or verify sig on assertion elements).
// * Verify various parts of the Assertion element. Conditions, audience, etc.
// * Map the Assertion's attribute elements to user info.
//
func (p *provider) HandlePOST(s connector.Scopes, samlResponse, inResponseTo string) (ident connector.Identity, err error) {
	rawResp, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		return ident, fmt.Errorf("decode response: %v", err)
	}

	// Root element is allowed to not be signed if the Assertion element is.
	rootElementSigned := true
	if p.validator != nil {
		rawResp, rootElementSigned, err = verifyResponseSig(p.validator, rawResp)
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
		if p.ssoIssuer != "" && resp.Issuer != nil && resp.Issuer.Issuer != p.ssoIssuer {
			return ident, fmt.Errorf("expected Issuer value %s, got %s", p.ssoIssuer, resp.Issuer.Issuer)
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
			return ident, fmt.Errorf("NameID element does not contain a value")
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
	p.logger.Infof("parsed and verified saml response attributes %s", attributes)

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
		return ident, nil
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
		return ident, nil
	}

	// Look for membership in one of the allowed groups
	groupMatches := groups.Filter(ident.Groups, p.allowedGroups)

	if len(groupMatches) == 0 {
		// No group membership matches found, disallowing
		return ident, fmt.Errorf("user not a member of allowed groups")
	}

	// Otherwise, we're good
	return ident, nil
}

// validateStatus verifies that the response has a good status code or
// formats a human readble error based on the bad status.
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
		return fmt.Errorf(errorMessage)
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
// we have no other way of guarenteeing that this is actually the response to
// the request we expect.
func (p *provider) validateSubject(subject *subject, inResponseTo string) error {
	// Optional according to the spec, but again, we're going to be strict here.
	if len(subject.SubjectConfirmations) == 0 {
		return fmt.Errorf("subject contained no SubjectConfirmations")
	}

	var errs []error
	// One of these must match our assumptions, not all.
	for _, c := range subject.SubjectConfirmations {
		err := func() error {
			if c.Method != subjectConfirmationMethodBearer {
				return fmt.Errorf("unexpected subject confirmation method: %v", c.Method)
			}

			data := c.SubjectConfirmationData
			if data == nil {
				return fmt.Errorf("SubjectConfirmation contained no SubjectConfirmationData")
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

// validationConditions ensures that dex is the intended audience
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
	if err != nil {
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
