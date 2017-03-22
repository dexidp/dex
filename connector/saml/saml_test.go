package saml

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"

	dsig "github.com/russellhaering/goxmldsig"

	"github.com/coreos/dex/connector"
)

const (
	defaultIssuer      = "http://www.okta.com/exk91cb99lKkKSYoy0h7"
	defaultRedirectURI = "http://localhost:5556/dex/callback"

	// Response ID embedded in our testdata.
	testDataResponseID = "_fd1b3ef9-ec09-44a7-a66b-0d39c250f6a0"
)

func loadCert(ca string) (*x509.Certificate, error) {
	data, err := ioutil.ReadFile(ca)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("ca file didn't contain any PEM data")
	}
	return x509.ParseCertificate(block.Bytes)
}

func runVerify(t *testing.T, ca string, resp string, shouldSucceed bool) {
	cert, err := loadCert(ca)
	if err != nil {
		t.Fatal(err)
	}
	s := certStore{[]*x509.Certificate{cert}}

	validator := dsig.NewDefaultValidationContext(s)

	data, err := ioutil.ReadFile(resp)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := verify(validator, data); err != nil {
		if shouldSucceed {
			t.Fatal(err)
		}
	} else {
		if !shouldSucceed {
			t.Fatalf("expected an invalid signatrue but verification has been successful")
		}
	}
}

func newProvider(issuer string, redirectURI string) *provider {
	if issuer == "" {
		issuer = defaultIssuer
	}
	if redirectURI == "" {
		redirectURI = defaultRedirectURI
	}
	now, _ := time.Parse(time.RFC3339, "2017-01-24T20:48:41Z")
	timeFunc := func() time.Time { return now }
	return &provider{
		issuer:       issuer,
		ssoURL:       "http://idp.org/saml/sso",
		now:          timeFunc,
		usernameAttr: "user",
		emailAttr:    "email",
		redirectURI:  redirectURI,
		logger:       logrus.New(),
	}
}

func TestVerify(t *testing.T) {
	runVerify(t, "testdata/okta-ca.pem", "testdata/okta-resp.xml", true)
}

func TestVerifyUnsignedMessageAndSignedAssertionWithRootXmlNs(t *testing.T) {
	runVerify(t, "testdata/oam-ca.pem", "testdata/oam-resp.xml", true)
}

func TestVerifySignedMessageAndUnsignedAssertion(t *testing.T) {
	runVerify(t, "testdata/idp-cert.pem", "testdata/idp-resp-signed-message.xml", true)
}

func TestVerifyUnsignedMessageAndSignedAssertion(t *testing.T) {
	runVerify(t, "testdata/idp-cert.pem", "testdata/idp-resp-signed-assertion.xml", true)
}

func TestVerifySignedMessageAndSignedAssertion(t *testing.T) {
	runVerify(t, "testdata/idp-cert.pem", "testdata/idp-resp-signed-message-and-assertion.xml", true)
}

func TestVerifyUnsignedMessageAndUnsignedAssertion(t *testing.T) {
	runVerify(t, "testdata/idp-cert.pem", "testdata/idp-resp.xml", false)
}

func TestHandlePOST(t *testing.T) {
	p := newProvider("", "")
	scopes := connector.Scopes{
		OfflineAccess: false,
		Groups:        true,
	}
	data, err := ioutil.ReadFile("testdata/idp-resp.xml")
	if err != nil {
		t.Fatal(err)
	}
	ident, err := p.HandlePOST(scopes, base64.StdEncoding.EncodeToString(data), testDataResponseID)
	if err != nil {
		t.Fatal(err)
	}
	if ident.UserID != "eric.chiang+okta@coreos.com" {
		t.Fatalf("unexpected UserID %q", ident.UserID)
	}
	if ident.Username != "admin" {
		t.Fatalf("unexpected Username: %q", ident.UserID)
	}
}

func TestValidateStatus(t *testing.T) {
	p := newProvider("", "")
	var err error
	resp := response{}
	// Test missing Status element
	err = p.validateStatus(&resp)
	if err == nil || !strings.HasSuffix(err.Error(), `Status`) {
		t.Fatalf("validation should fail with missing Status")
	}
	// Test missing StatusCode element
	resp.Status = &status{}
	err = p.validateStatus(&resp)
	if err == nil || !strings.HasSuffix(err.Error(), `StatusCode`) {
		t.Fatalf("validation should fail with missing StatusCode")
	}
	// Test failed request without StatusMessage
	resp.Status.StatusCode = &statusCode{
		Value: ":Requester",
	}
	err = p.validateStatus(&resp)
	if err == nil || !strings.HasSuffix(err.Error(), `"Requester"`) {
		t.Fatalf("validation should fail with code %q", "Requester")
	}
	// Test failed request with StatusMessage
	resp.Status.StatusMessage = &statusMessage{
		Value: "Failed",
	}
	err = p.validateStatus(&resp)
	if err == nil || !strings.HasSuffix(err.Error(), `"Requester" -> Failed`) {
		t.Fatalf("validation should fail with code %q and message %q", "Requester", "Failed")
	}
}

func TestValidateSubjectConfirmation(t *testing.T) {
	p := newProvider("", "")
	var err error
	var notAfter time.Time
	subj := &subject{}
	// Subject without any SubjectConfirmation
	err = p.validateSubjectConfirmation(subj)
	if err == nil {
		t.Fatalf("validation of %q should fail", "Subject without any SubjectConfirmation")
	}
	// SubjectConfirmation without Method and SubjectConfirmationData
	subj.SubjectConfirmations = []subjectConfirmation{subjectConfirmation{}}
	err = p.validateSubjectConfirmation(subj)
	if err == nil {
		t.Fatalf("validation of %q should fail", "SubjectConfirmation without Method and SubjectConfirmationData")
	}
	// SubjectConfirmation with invalid Method and no SubjectConfirmationData
	subj.SubjectConfirmations = []subjectConfirmation{subjectConfirmation{
		Method: "invalid",
	}}
	err = p.validateSubjectConfirmation(subj)
	if err == nil {
		t.Fatalf("validation of %q should fail", "SubjectConfirmation with invalid Method and no SubjectConfirmationData")
	}
	// SubjectConfirmation with valid Method and empty SubjectConfirmationData
	subjConfirmationData := subjectConfirmationData{}
	subj.SubjectConfirmations = []subjectConfirmation{subjectConfirmation{
		Method:                  "urn:oasis:names:tc:SAML:2.0:cm:bearer",
		SubjectConfirmationData: &subjConfirmationData,
	}}
	err = p.validateSubjectConfirmation(subj)
	if err != nil {
		t.Fatalf("validation of %q should succeed", "SubjectConfirmation with valid Method and empty SubjectConfirmationData")
	}
	// SubjectConfirmationData with invalid Recipient
	subjConfirmationData.Recipient = "invalid"
	err = p.validateSubjectConfirmation(subj)
	if err == nil {
		t.Fatalf("validation of %q should fail", "SubjectConfirmationData with invalid Recipient")
	}
	// expired SubjectConfirmationData
	notAfter = p.now().Add(-time.Duration(60) * time.Second)
	subjConfirmationData.NotOnOrAfter = xmlTime(notAfter)
	subjConfirmationData.Recipient = defaultRedirectURI
	err = p.validateSubjectConfirmation(subj)
	if err == nil {
		t.Fatalf("validation of %q should fail", " expired SubjectConfirmationData")
	}
	// valid SubjectConfirmationData
	notAfter = p.now().Add(+time.Duration(60) * time.Second)
	subjConfirmationData.NotOnOrAfter = xmlTime(notAfter)
	subjConfirmationData.Recipient = defaultRedirectURI
	err = p.validateSubjectConfirmation(subj)
	if err != nil {
		t.Fatalf("validation of %q should succed", "valid SubjectConfirmationData")
	}
}

func TestValidateConditions(t *testing.T) {
	p := newProvider("", "")
	var err error
	var notAfter, notBefore time.Time
	cond := conditions{
		AudienceRestriction: &audienceRestriction{},
	}
	assert := &assertion{}
	// Assertion without Conditions
	err = p.validateConditions(assert)
	if err != nil {
		t.Fatalf("validation of %q should succeed", "Assertion without Conditions")
	}
	// Assertion with empty Conditions
	assert.Conditions = &cond
	err = p.validateConditions(assert)
	if err != nil {
		t.Fatalf("validation of %q should succeed", "Assertion with empty Conditions")
	}
	// Conditions with valid timestamps
	notBefore = p.now().Add(-time.Duration(60) * time.Second)
	notAfter = p.now().Add(+time.Duration(60) * time.Second)
	cond.NotBefore = xmlTime(notBefore)
	cond.NotOnOrAfter = xmlTime(notAfter)
	err = p.validateConditions(assert)
	if err != nil {
		t.Fatalf("validation of %q should succeed", "Conditions with valid timestamps")
	}
	// Conditions where notBefore is 45 seconds after now
	notBefore = p.now().Add(+time.Duration(45) * time.Second)
	cond.NotBefore = xmlTime(notBefore)
	err = p.validateConditions(assert)
	if err == nil {
		t.Fatalf("validation of %q should fail", "Conditions where notBefore is 45 seconds after now")
	}
	// Conditions where notBefore is 15 seconds after now
	notBefore = p.now().Add(+time.Duration(15) * time.Second)
	cond.NotBefore = xmlTime(notBefore)
	err = p.validateConditions(assert)
	if err != nil {
		t.Fatalf("validation of %q should succeed", "Conditions where notBefore is 15 seconds after now")
	}
	// Audiences contains the redirectURI
	validAudience := audience{Value: p.redirectURI}
	cond.AudienceRestriction.Audiences = []audience{validAudience}
	err = p.validateConditions(assert)
	if err != nil {
		t.Fatalf("validation of %q should succeed: %v", "Audiences contains the redirectURI", err)
	}
	// Audiences is not empty and not contains the issuer
	invalidAudience := audience{Value: "invalid"}
	cond.AudienceRestriction.Audiences = []audience{invalidAudience}
	err = p.validateConditions(assert)
	if err == nil {
		t.Fatalf("validation of %q should succeed", "Audiences is not empty and not contains the issuer")
	}
}
