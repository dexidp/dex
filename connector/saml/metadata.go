package saml

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"strings"
)

// IdPMetadata is the subset of SAML IdP metadata dex consumes.
type IdPMetadata struct {
	EntityID     string
	SSOEndpoints []SSOEndpoint
	SigningCerts []*x509.Certificate
}

// SSOEndpoint is a SingleSignOnService entry from the IdP metadata.
type SSOEndpoint struct {
	Binding  string
	Location string
}

// ssoEndpoint returns the Location of the first endpoint using the given
// binding, if any.
func (m *IdPMetadata) ssoEndpoint(binding string) (string, bool) {
	for _, e := range m.SSOEndpoints {
		if e.Binding == binding {
			return e.Location, true
		}
	}
	return "", false
}

type entityDescriptor struct {
	XMLName          xml.Name          `xml:"urn:oasis:names:tc:SAML:2.0:metadata EntityDescriptor"`
	EntityID         string            `xml:"entityID,attr"`
	IDPSSODescriptor *idpSSODescriptor `xml:"IDPSSODescriptor"`
}

type idpSSODescriptor struct {
	XMLName             xml.Name              `xml:"urn:oasis:names:tc:SAML:2.0:metadata IDPSSODescriptor"`
	KeyDescriptors      []keyDescriptor       `xml:"KeyDescriptor"`
	SingleSignOnService []singleSignOnService `xml:"SingleSignOnService"`
}

type keyDescriptor struct {
	Use     string   `xml:"use,attr"`
	KeyInfo *keyInfo `xml:"http://www.w3.org/2000/09/xmldsig# KeyInfo"`
}

type keyInfo struct {
	X509Data *x509Data `xml:"http://www.w3.org/2000/09/xmldsig# X509Data"`
}

type x509Data struct {
	X509Certificate []x509Certificate `xml:"http://www.w3.org/2000/09/xmldsig# X509Certificate"`
}

type x509Certificate struct {
	Data string `xml:",chardata"`
}

type singleSignOnService struct {
	Binding  string `xml:"Binding,attr"`
	Location string `xml:"Location,attr"`
}

// parseMetadata parses a SAML IdP metadata document and extracts the entity ID,
// signing certificates, and SingleSignOnService endpoints.
func parseMetadata(data []byte) (*IdPMetadata, error) {
	var ed entityDescriptor
	if err := xml.Unmarshal(data, &ed); err != nil {
		return nil, fmt.Errorf("parse metadata: %v", err)
	}
	if ed.EntityID == "" {
		return nil, fmt.Errorf("parse metadata: missing entityID attribute")
	}
	if ed.IDPSSODescriptor == nil {
		return nil, fmt.Errorf("parse metadata: no IDPSSODescriptor found")
	}

	meta := &IdPMetadata{EntityID: ed.EntityID}
	idp := ed.IDPSSODescriptor

	for _, kd := range idp.KeyDescriptors {
		// Encryption-only keys are irrelevant for signature verification.
		if kd.Use == "encryption" || kd.KeyInfo == nil || kd.KeyInfo.X509Data == nil {
			continue
		}
		for _, cert := range kd.KeyInfo.X509Data.X509Certificate {
			der, err := base64.StdEncoding.DecodeString(strings.TrimSpace(cert.Data))
			if err != nil {
				return nil, fmt.Errorf("parse metadata: decode certificate: %v", err)
			}
			c, err := x509.ParseCertificate(der)
			if err != nil {
				return nil, fmt.Errorf("parse metadata: parse certificate: %v", err)
			}
			meta.SigningCerts = append(meta.SigningCerts, c)
		}
	}

	for _, sso := range idp.SingleSignOnService {
		meta.SSOEndpoints = append(meta.SSOEndpoints, SSOEndpoint{
			Binding:  sso.Binding,
			Location: sso.Location,
		})
	}

	if len(meta.SigningCerts) == 0 && len(meta.SSOEndpoints) == 0 {
		return nil, fmt.Errorf("parse metadata: no signing certificates or SSO endpoints found")
	}
	return meta, nil
}
