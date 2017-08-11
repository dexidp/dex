package saml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"time"
)

const timeFormat = "2006-01-02T15:04:05Z"

type xmlTime time.Time

func (t xmlTime) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{
		Name:  name,
		Value: time.Time(t).UTC().Format(timeFormat),
	}, nil
}

func (t *xmlTime) UnmarshalXMLAttr(attr xml.Attr) error {
	got, err := time.Parse(timeFormat, attr.Value)
	if err != nil {
		return err
	}
	*t = xmlTime(got)
	return nil
}

type samlVersion struct{}

func (s samlVersion) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{
		Name:  name,
		Value: "2.0",
	}, nil
}

func (s *samlVersion) UnmarshalXMLAttr(attr xml.Attr) error {
	if attr.Value != "2.0" {
		return fmt.Errorf(`saml version expected "2.0" got %q`, attr.Value)
	}
	return nil
}

type authnRequest struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:protocol AuthnRequest"`

	ID      string      `xml:"ID,attr"`
	Version samlVersion `xml:"Version,attr"`

	ProviderName string  `xml:"ProviderName,attr,omitempty"`
	IssueInstant xmlTime `xml:"IssueInstant,attr,omitempty"`
	Consent      bool    `xml:"Consent,attr,omitempty"`
	Destination  string  `xml:"Destination,attr,omitempty"`

	ForceAuthn      bool   `xml:"ForceAuthn,attr,omitempty"`
	IsPassive       bool   `xml:"IsPassive,attr,omitempty"`
	ProtocolBinding string `xml:"ProtocolBinding,attr,omitempty"`

	AssertionConsumerServiceURL string `xml:"AssertionConsumerServiceURL,attr,omitempty"`

	Subject      *subject      `xml:"Subject,omitempty"`
	Issuer       *issuer       `xml:"Issuer,omitempty"`
	NameIDPolicy *nameIDPolicy `xml:"NameIDPolicy,omitempty"`

	// TODO(ericchiang): Make this configurable and determine appropriate default values.
	RequestAuthnContext *requestAuthnContext `xml:"RequestAuthnContext,omitempty"`
}

type subject struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion Subject"`

	NameID               *nameID               `xml:"NameID,omitempty"`
	SubjectConfirmations []subjectConfirmation `xml:"SubjectConfirmation"`

	// TODO(ericchiang): Do we need to deal with baseID?
}

type nameID struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion NameID"`

	Format string `xml:"Format,omitempty"`
	Value  string `xml:",chardata"`
}

type subjectConfirmationData struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion SubjectConfirmationData"`

	NotBefore    xmlTime `xml:"NotBefore,attr,omitempty"`
	NotOnOrAfter xmlTime `xml:"NotOnOrAfter,attr,omitempty"`
	Recipient    string  `xml:"Recipient,attr,omitempty"`
	InResponseTo string  `xml:"InResponseTo,attr,omitempty"`
}

type subjectConfirmation struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion SubjectConfirmation"`

	Method                  string                   `xml:"Method,attr,omitempty"`
	SubjectConfirmationData *subjectConfirmationData `xml:"SubjectConfirmationData,omitempty"`
}

type audience struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion Audience"`
	Value   string   `xml:",chardata"`
}

type audienceRestriction struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion AudienceRestriction"`

	Audiences []audience `xml:"Audience"`
}

type conditions struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion Conditions"`

	NotBefore    xmlTime `xml:"NotBefore,attr,omitempty"`
	NotOnOrAfter xmlTime `xml:"NotOnOrAfter,attr,omitempty"`

	AudienceRestriction []audienceRestriction `xml:"AudienceRestriction,omitempty"`
}

type statusCode struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:protocol StatusCode"`

	Value string `xml:"Value,attr,omitempty"`
}

type statusMessage struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:protocol StatusMessage"`

	Value string `xml:",chardata"`
}

type status struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:protocol Status"`

	StatusCode    *statusCode    `xml:"StatusCode"`
	StatusMessage *statusMessage `xml:"StatusMessage,omitempty"`
}

type issuer struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion Issuer"`
	Issuer  string   `xml:",chardata"`
}

type nameIDPolicy struct {
	XMLName     xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:protocol NameIDPolicy"`
	AllowCreate bool     `xml:"AllowCreate,attr,omitempty"`
	Format      string   `xml:"Format,attr,omitempty"`
}

type requestAuthnContext struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:protocol RequestAuthnContext"`

	AuthnContextClassRefs []authnContextClassRef
}

type authnContextClassRef struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:protocol AuthnContextClassRef"`
	Value   string   `xml:",chardata"`
}

type response struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:protocol Response"`

	ID           string      `xml:"ID,attr"`
	InResponseTo string      `xml:"InResponseTo,attr"`
	Version      samlVersion `xml:"Version,attr"`

	Destination string `xml:"Destination,attr,omitempty"`

	Issuer *issuer `xml:"Issuer,omitempty"`

	Status *status `xml:"Status"`

	// TODO(ericchiang): How do deal with multiple assertions?
	Assertion *assertion `xml:"Assertion,omitempty"`
}

type assertion struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion Assertion"`

	Version       samlVersion `xml:"Version,attr"`
	ID            string      `xml:"ID,attr"`
	IssueInstance xmlTime     `xml:"IssueInstance,attr"`

	Issuer issuer `xml:"Issuer"`

	Subject *subject `xml:"Subject,omitempty"`

	Conditions *conditions `xml:"Conditions"`

	AttributeStatement *attributeStatement `xml:"AttributeStatement,omitempty"`
}

type attributeStatement struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion AttributeStatement"`

	Attributes []attribute `xml:"Attribute"`
}

func (a *attributeStatement) get(name string) (s string, ok bool) {
	for _, attr := range a.Attributes {
		if attr.Name == name {
			ok = true
			if len(attr.AttributeValues) > 0 {
				return attr.AttributeValues[0].Value, true
			}
		}
	}
	return
}

func (a *attributeStatement) all(name string) (s []string, ok bool) {
	for _, attr := range a.Attributes {
		if attr.Name == name {
			ok = true
			for _, val := range attr.AttributeValues {
				s = append(s, val.Value)
			}
		}
	}
	return
}

// names list the names of all attributes in the attribute statement.
func (a *attributeStatement) names() []string {
	s := make([]string, len(a.Attributes))

	for i, attr := range a.Attributes {
		s[i] = attr.Name
	}
	return s
}

// String is a formatter for logging an attribute statement's sub statements.
func (a *attributeStatement) String() string {
	buff := new(bytes.Buffer)
	for i, attr := range a.Attributes {
		if i != 0 {
			buff.WriteString(", ")
		}
		buff.WriteString(attr.String())
	}
	return buff.String()
}

type attribute struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion Attribute"`

	Name string `xml:"Name,attr"`

	NameFormat   string `xml:"NameFormat,attr,omitempty"`
	FriendlyName string `xml:"FriendlyName,attr,omitempty"`

	AttributeValues []attributeValue `xml:"AttributeValue,omitempty"`
}

type attributeValue struct {
	XMLName xml.Name `xml:"AttributeValue"`
	Value   string   `xml:",chardata"`
}

func (a attribute) String() string {
	if len(a.AttributeValues) == 1 {
		// "email" = "jane.doe@coreos.com"
		return fmt.Sprintf("%q = %q", a.Name, a.AttributeValues[0].Value)
	}
	values := make([]string, len(a.AttributeValues))
	for i, av := range a.AttributeValues {
		values[i] = av.Value
	}

	// "groups" = ["engineering", "docs"]
	return fmt.Sprintf("%q = %q", a.Name, values)
}
