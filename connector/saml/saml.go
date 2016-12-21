// Package saml contains login methods for SAML.
package saml

import (
	"bytes"
	"compress/flate"
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
		{"issuer", c.Issuer},
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

func (p *provider) POSTData(s connector.Scopes) (action, value string, err error) {

	// NOTE(ericchiang): If we can't follow up with the identity provider, can we
	// support refresh tokens?
	if s.OfflineAccess {
		return "", "", fmt.Errorf("SAML does not support offline access")
	}

	r := &authnRequest{
		ProtocolBinding: bindingPOST,
		ID:              "_" + uuidv4(),
		IssueInstant:    xmlTime(p.now()),
		Destination:     p.ssoURL,
		Issuer: &issuer{
			Issuer: p.issuer,
		},
		NameIDPolicy: &nameIDPolicy{
			AllowCreate: true,
			Format:      p.nameIDPolicyFormat,
		},
	}

	data, err := xml.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("marshal authn request: %v", err)
	}

	buff := new(bytes.Buffer)
	fw, err := flate.NewWriter(buff, flate.DefaultCompression)
	if err != nil {
		return "", "", fmt.Errorf("new flate writer: %v", err)
	}
	if _, err := fw.Write(data); err != nil {
		return "", "", fmt.Errorf("compress message: %v", err)
	}
	if err := fw.Close(); err != nil {
		return "", "", fmt.Errorf("flush message: %v", err)
	}

	return p.ssoURL, base64.StdEncoding.EncodeToString(buff.Bytes()), nil
}

func (p *provider) HandlePOST(s connector.Scopes, samlResponse string) (ident connector.Identity, err error) {
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

	if resp.Destination != "" && resp.Destination != p.redirectURI {
		return ident, fmt.Errorf("expected destination %q got %q", p.redirectURI, resp.Destination)

	}

	assertion := resp.Assertion
	if assertion == nil {
		return ident, fmt.Errorf("response did not contain an assertion")
	}
	subject := assertion.Subject
	if subject == nil {
		return ident, fmt.Errorf("response did not contain a subject")
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
		return ident, fmt.Errorf("no attribute with name %q", p.emailAttr)
	}
	ident.EmailVerified = true

	if ident.Username, _ = attributes.get(p.usernameAttr); ident.Username == "" {
		return ident, fmt.Errorf("no attribute with name %q", p.usernameAttr)
	}

	if s.Groups && p.groupsAttr != "" {
		if p.groupsDelim != "" {
			groupsStr, ok := attributes.get(p.groupsAttr)
			if !ok {
				return ident, fmt.Errorf("no attribute with name %q", p.groupsAttr)
			}
			// TOOD(ericchiang): Do we need to further trim whitespace?
			ident.Groups = strings.Split(groupsStr, p.groupsDelim)
		} else {
			groups, ok := attributes.all(p.groupsAttr)
			if !ok {
				return ident, fmt.Errorf("no attribute with name %q", p.groupsAttr)
			}
			ident.Groups = groups
		}
	}

	return ident, nil
}

// verify checks the signature info of a XML document and returns
// the signed elements.
func verify(validator *dsig.ValidationContext, data []byte) (signed []byte, err error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(data); err != nil {
		return nil, fmt.Errorf("parse document: %v", err)
	}

	result, err := validator.Validate(doc.Root())
	if err != nil {
		return nil, err
	}
	doc.SetRoot(result)
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
