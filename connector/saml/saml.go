// Package saml contains login methods for SAML.
package saml

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/russellhaering/goxmldsig/etreeutils"

	"github.com/coreos/dex/connector"
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

	Issuer string `json:"issuer"`
	SSOURL string `json:"ssoURL"`

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
	GroupsDelim string `json:"groupsDelim"`

	RedirectURI string `json:"redirectURI"`

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
func (c *Config) Open(logger logrus.FieldLogger) (connector.Connector, error) {
	return c.openConnector(logger)
}

func (c *Config) openConnector(logger logrus.FieldLogger) (interface {
	connector.SAMLConnector
}, error) {
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
		issuer:       c.Issuer,
		ssoURL:       c.SSOURL,
		now:          time.Now,
		usernameAttr: c.UsernameAttr,
		emailAttr:    c.EmailAttr,
		groupsAttr:   c.GroupsAttr,
		groupsDelim:  c.GroupsDelim,
		redirectURI:  c.RedirectURI,
		logger:       logger,

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
	issuer string
	ssoURL string

	now func() time.Time

	// If nil, don't do signature validation.
	validator *dsig.ValidationContext

	// Attribute mappings
	usernameAttr string
	emailAttr    string
	groupsAttr   string
	groupsDelim  string

	redirectURI string

	nameIDPolicyFormat string

	logger logrus.FieldLogger
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
	if p.issuer != "" {
		// Issuer for the request is optional. For example, okta always ignores
		// this value.
		r.Issuer = &issuer{Issuer: p.issuer}
	}

	data, err := xml.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("marshal authn request: %v", err)
	}

	// See: https://docs.oasis-open.org/security/saml/v2.0/saml-bindings-2.0-os.pdf
	// "3.5.4 Message Encoding"
	return p.ssoURL, base64.StdEncoding.EncodeToString(data), nil
}

func (p *provider) HandlePOST(s connector.Scopes, samlResponse, inResponseTo string) (ident connector.Identity, err error) {
	rawResp, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		return ident, fmt.Errorf("decode response: %v", err)
	}
	if p.validator != nil {
		if rawResp, err = verify(p.validator, rawResp); err != nil {
			return ident, fmt.Errorf("verify signature: %v", err)
		}
	}

	var resp response
	if err := xml.Unmarshal(rawResp, &resp); err != nil {
		return ident, fmt.Errorf("unmarshal response: %v", err)
	}

	if p.issuer != "" && resp.Issuer != nil && resp.Issuer.Issuer != p.issuer {
		return ident, fmt.Errorf("expected Issuer value %s, got %s", p.issuer, resp.Issuer.Issuer)
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

	if err = p.validateStatus(&resp); err != nil {
		return ident, err
	}

	assertion := resp.Assertion
	if assertion == nil {
		return ident, fmt.Errorf("response did not contain an assertion")
	}
	subject := assertion.Subject
	if subject == nil {
		return ident, fmt.Errorf("response did not contain a subject")
	}

	if err = p.validateConditions(assertion); err != nil {
		return ident, err
	}
	if err = p.validateSubjectConfirmation(subject); err != nil {
		return ident, err
	}

	switch {
	case subject.NameID != nil:
		if ident.UserID = subject.NameID.Value; ident.UserID == "" {
			return ident, fmt.Errorf("NameID element does not contain a value")
		}
	default:
		return ident, fmt.Errorf("subject does not contain an NameID element")
	}

	attributes := assertion.AttributeStatement
	if attributes == nil {
		return ident, fmt.Errorf("response did not contain a AttributeStatement")
	}

	if ident.Email, _ = attributes.get(p.emailAttr); ident.Email == "" {
		return ident, fmt.Errorf("no attribute with name %q: %s", p.emailAttr, attributes.names())
	}
	ident.EmailVerified = true

	if ident.Username, _ = attributes.get(p.usernameAttr); ident.Username == "" {
		return ident, fmt.Errorf("no attribute with name %q: %s", p.usernameAttr, attributes.names())
	}

	if s.Groups && p.groupsAttr != "" {
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
	}

	return ident, nil
}

// Validate that the StatusCode of the Response is success.
// Otherwise return a human readable message to the end user
func (p *provider) validateStatus(resp *response) error {
	// Status is mandatory in the Response type
	status := resp.Status
	if status == nil {
		return fmt.Errorf("response did not contain a Status")
	}
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

// Multiple subject SubjectConfirmation can be in the assertion
// and at least one SubjectConfirmation must be valid.
// This is described in the spec "Profiles for the OASIS Security
// Assertion Markup Language" in section 3.3 Bearer.
// see https://www.oasis-open.org/committees/download.php/35389/sstc-saml-profiles-errata-2.0-wd-06-diff.pdf
func (p *provider) validateSubjectConfirmation(subject *subject) error {
	validSubjectConfirmation := false
	subjectConfirmations := subject.SubjectConfirmations
	if subjectConfirmations != nil && len(subjectConfirmations) > 0 {
		for _, subjectConfirmation := range subjectConfirmations {
			// skip if method is wrong
			method := subjectConfirmation.Method
			if method != "" && method != subjectConfirmationMethodBearer {
				continue
			}
			subjectConfirmationData := subjectConfirmation.SubjectConfirmationData
			if subjectConfirmationData == nil {
				continue
			}
			inResponseTo := subjectConfirmationData.InResponseTo
			if inResponseTo != "" {
				// TODO also validate InResponseTo if present
			}
			// only validate that subjectConfirmationData is not expired
			now := p.now()
			notOnOrAfter := time.Time(subjectConfirmationData.NotOnOrAfter)
			if !notOnOrAfter.IsZero() {
				if now.After(notOnOrAfter) {
					continue
				}
			}
			// validate recipient if present
			recipient := subjectConfirmationData.Recipient
			if recipient != "" && recipient != p.redirectURI {
				continue
			}
			validSubjectConfirmation = true
		}
	}
	if !validSubjectConfirmation {
		return fmt.Errorf("no valid SubjectConfirmation was found on this Response")
	}
	return nil
}

// Validates the Conditions element and all of it's content
//
// See: https://docs.oasis-open.org/security/saml/v2.0/saml-core-2.0-os.pdf
// "2.3.3 Element <Assertion>"
func (p *provider) validateConditions(assertion *assertion) error {
	// Checks if a Conditions element exists
	conditions := assertion.Conditions
	if conditions == nil {
		return nil
	}
	// Validates Assertion timestamps
	now := p.now()
	notBefore := time.Time(conditions.NotBefore)
	if !notBefore.IsZero() {
		if now.Add(allowedClockDrift).Before(notBefore) {
			return fmt.Errorf("at %s got response that cannot be processed before %s", now, notBefore)
		}
	}
	notOnOrAfter := time.Time(conditions.NotOnOrAfter)
	if !notOnOrAfter.IsZero() {
		if now.After(notOnOrAfter.Add(allowedClockDrift)) {
			return fmt.Errorf("at %s got response that cannot be processed because it expired at %s", now, notOnOrAfter)
		}
	}
	// Validates audience
	audienceRestriction := conditions.AudienceRestriction
	if audienceRestriction != nil {
		audiences := audienceRestriction.Audiences
		if audiences != nil && len(audiences) > 0 {
			values := make([]string, len(audiences))
			issuerInAudiences := false
			for i, audience := range audiences {
				if audience.Value == p.redirectURI {
					issuerInAudiences = true
					break
				}
				values[i] = audience.Value
			}
			if !issuerInAudiences {
				return fmt.Errorf("required audience %s was not in Response audiences %s", p.redirectURI, values)
			}
		}
	}
	return nil
}

// verify checks the signature info of a XML document and returns
// the signed elements.
// The Validate function of the goxmldsig library only looks for
// signatures on the root element level. But a saml Response is valid
// if the complete message is signed, or only the Assertion is signed,
// or but elements are signed. Therefore we first check a possible
// signature of the Response than of the Assertion. If one of these
// is successful the Response is considered as valid.
func verify(validator *dsig.ValidationContext, data []byte) (signed []byte, err error) {
	doc := etree.NewDocument()
	if err = doc.ReadFromBytes(data); err != nil {
		return nil, fmt.Errorf("parse document: %v", err)
	}
	verified := false
	response := doc.Root()
	transformedResponse, err := validator.Validate(response)
	if err == nil {
		verified = true
		doc.SetRoot(transformedResponse)
	}
	// Ensures xmlns are copied down to the assertion element when they are defined in the root
	assertion, err := etreeutils.NSSelectOne(response, "urn:oasis:names:tc:SAML:2.0:assertion", "Assertion")
	if err != nil {
		return nil, fmt.Errorf("response does not contain an Assertion element")
	}
	transformedAssertion, err := validator.Validate(assertion)
	if err == nil {
		verified = true
		response.RemoveChild(assertion)
		response.AddChild(transformedAssertion)
	}
	if verified != true {
		return nil, fmt.Errorf("response does not contain a valid Signature element")
	}
	return doc.WriteToBytes()
}

func uuidv4() string {
	u := make([]byte, 16)
	if _, err := rand.Read(u); err != nil {
		panic(err)
	}
	u[6] = (u[6] | 0x40) & 0x4F
	u[8] = (u[8] | 0x80) & 0xBF

	r := make([]byte, 36)
	r[8] = '-'
	r[13] = '-'
	r[18] = '-'
	r[23] = '-'
	hex.Encode(r, u[0:4])
	hex.Encode(r[9:], u[4:6])
	hex.Encode(r[14:], u[6:8])
	hex.Encode(r[19:], u[8:10])
	hex.Encode(r[24:], u[10:])

	return string(r)
}
