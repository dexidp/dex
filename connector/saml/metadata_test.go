package saml

import (
	"encoding/base64"
	"encoding/pem"
	"os"
	"testing"
)

// certPEMBlock returns the base64-DER form of the first PEM block in pemBytes,
// as it appears inside a SAML metadata <ds:X509Certificate> element.
func certPEMBlock(t *testing.T, pemBytes []byte) string {
	t.Helper()
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		t.Fatal("no PEM block found")
	}
	return base64.StdEncoding.EncodeToString(block.Bytes)
}

func metadataXML(t *testing.T, keyDescriptors, ssoServices string) string {
	t.Helper()
	return `<?xml version="1.0" encoding="UTF-8"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" xmlns:ds="http://www.w3.org/2000/09/xmldsig#" entityID="https://idp.example.com">
  <md:IDPSSODescriptor>` + keyDescriptors + ssoServices + `
  </md:IDPSSODescriptor>
</md:EntityDescriptor>`
}

func TestParseMetadata(t *testing.T) {
	cert := certPEMBlock(t, mustReadFile(t, "testdata/ca.crt"))
	xml := metadataXML(t,
		`<md:KeyDescriptor use="signing"><ds:KeyInfo><ds:X509Data><ds:X509Certificate>`+cert+`</ds:X509Certificate></ds:X509Data></ds:KeyInfo></md:KeyDescriptor>`+
			`<md:KeyDescriptor use="encryption"><ds:KeyInfo><ds:X509Data><ds:X509Certificate>`+cert+`</ds:X509Certificate></ds:X509Data></ds:KeyInfo></md:KeyDescriptor>`,
		`<md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://idp.example.com/sso/redirect"/>`+
			`<md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="https://idp.example.com/sso/post"/>`)

	meta, err := parseMetadata([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	if meta.EntityID != "https://idp.example.com" {
		t.Errorf("expected entityID, got %q", meta.EntityID)
	}
	if len(meta.SigningCerts) != 1 {
		t.Fatalf("expected 1 signing cert (encryption key excluded), got %d", len(meta.SigningCerts))
	}
	if len(meta.SSOEndpoints) != 2 {
		t.Fatalf("expected 2 SSO endpoints, got %d", len(meta.SSOEndpoints))
	}
	if meta.SSOEndpoints[0].Binding != "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" {
		t.Errorf("expected first endpoint to be HTTP-Redirect, got %q", meta.SSOEndpoints[0].Binding)
	}
	if url, ok := meta.ssoEndpoint(bindingPOST); !ok || url != "https://idp.example.com/sso/post" {
		t.Errorf("expected ssoEndpoint(POST) to find the POST location, got %q ok=%v", url, ok)
	}
	if url, ok := meta.ssoEndpoint(bindingRedirect); !ok || url != "https://idp.example.com/sso/redirect" {
		t.Errorf("expected ssoEndpoint(Redirect) to find the redirect location, got %q ok=%v", url, ok)
	}
}

func TestParseMetadataKeyDescriptorWithoutUse(t *testing.T) {
	cert := certPEMBlock(t, mustReadFile(t, "testdata/ca.crt"))
	xml := metadataXML(t,
		`<md:KeyDescriptor><ds:KeyInfo><ds:X509Data><ds:X509Certificate>`+cert+`</ds:X509Certificate></ds:X509Data></ds:KeyInfo></md:KeyDescriptor>`,
		`<md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="https://idp.example.com/sso"/>`)

	meta, err := parseMetadata([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	// A KeyDescriptor without a use attribute counts as signing per the spec.
	if len(meta.SigningCerts) != 1 {
		t.Fatalf("expected 1 signing cert, got %d", len(meta.SigningCerts))
	}
}

func TestParseMetadataErrors(t *testing.T) {
	tests := []struct {
		name string
		xml  string
	}{
		{
			name: "malformed XML",
			xml:  `<md:EntityDescriptor`,
		},
		{
			name: "no IDPSSODescriptor",
			xml:  `<?xml version="1.0"?><md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" entityID="x"/>`,
		},
		{
			name: "no signing certs or SSO endpoints",
			xml:  metadataXML(t, "", ""),
		},
		{
			name: "bad base64 cert",
			xml: metadataXML(t,
				`<md:KeyDescriptor use="signing"><ds:KeyInfo><ds:X509Data><ds:X509Certificate>not!base64</ds:X509Certificate></ds:X509Data></ds:KeyInfo></md:KeyDescriptor>`,
				`<md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="https://idp.example.com/sso"/>`),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseMetadata([]byte(tc.xml)); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
