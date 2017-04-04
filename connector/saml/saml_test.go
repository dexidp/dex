package saml

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/kylelemons/godebug/pretty"
	dsig "github.com/russellhaering/goxmldsig"

	"github.com/coreos/dex/connector"
)

// responseTest maps a SAML 2.0 response object to a set of expected values.
//
// Tests are defined in the "testdata" directory and are self-signed using xmlsec1.
//
// To add a new test, define a new, unsigned SAML 2.0 response that exercises some
// case, then sign it using the "testdata/gen.sh" script.
//
//     cp testdata/good-resp.tmpl testdata/( testname ).tmpl
//     vim ( testname ).tmpl # Modify your template for your test case.
//     vim testdata/gen.sh   # Add a xmlsec1 command to the generation script.
//     ./testdata/gen.sh     # Sign your template.
//
// To install xmlsec1 on Fedora run:
//
//     sudo dnf install xmlsec1 xmlsec1-openssl
//
// On mac:
//
//     brew install Libxmlsec1
//
type responseTest struct {
	// CA file and XML file of the response.
	caFile   string
	respFile string

	// Values that should be used to validate the signature.
	now          string
	inResponseTo string
	redirectURI  string

	// Attribute customization.
	usernameAttr string
	emailAttr    string
	groupsAttr   string

	// Expected outcome of the test.
	wantErr   bool
	wantIdent connector.Identity
}

func TestGoodResponse(t *testing.T) {
	test := responseTest{
		caFile:       "testdata/ca.crt",
		respFile:     "testdata/good-resp.xml",
		now:          "2017-04-04T04:34:59.330Z",
		usernameAttr: "Name",
		emailAttr:    "email",
		inResponseTo: "6zmm5mguyebwvajyf2sdwwcw6m",
		redirectURI:  "http://127.0.0.1:5556/dex/callback",
		wantIdent: connector.Identity{
			UserID:        "eric.chiang+okta@coreos.com",
			Username:      "Eric",
			Email:         "eric.chiang+okta@coreos.com",
			EmailVerified: true,
		},
	}
	test.run(t)
}

func TestGroups(t *testing.T) {
	test := responseTest{
		caFile:       "testdata/ca.crt",
		respFile:     "testdata/good-resp.xml",
		now:          "2017-04-04T04:34:59.330Z",
		usernameAttr: "Name",
		emailAttr:    "email",
		groupsAttr:   "groups",
		inResponseTo: "6zmm5mguyebwvajyf2sdwwcw6m",
		redirectURI:  "http://127.0.0.1:5556/dex/callback",
		wantIdent: connector.Identity{
			UserID:        "eric.chiang+okta@coreos.com",
			Username:      "Eric",
			Email:         "eric.chiang+okta@coreos.com",
			EmailVerified: true,
			Groups:        []string{"Admins", "Everyone"},
		},
	}
	test.run(t)
}

// TestOkta tests against an actual response from Okta.
func TestOkta(t *testing.T) {
	test := responseTest{
		caFile:       "testdata/okta-ca.pem",
		respFile:     "testdata/okta-resp.xml",
		now:          "2017-04-04T04:34:59.330Z",
		usernameAttr: "Name",
		emailAttr:    "email",
		inResponseTo: "6zmm5mguyebwvajyf2sdwwcw6m",
		redirectURI:  "http://127.0.0.1:5556/dex/callback",
		wantIdent: connector.Identity{
			UserID:        "eric.chiang+okta@coreos.com",
			Username:      "Eric",
			Email:         "eric.chiang+okta@coreos.com",
			EmailVerified: true,
		},
	}
	test.run(t)
}

func TestBadStatus(t *testing.T) {
	test := responseTest{
		caFile:       "testdata/ca.crt",
		respFile:     "testdata/bad-status.xml",
		now:          "2017-04-04T04:34:59.330Z",
		usernameAttr: "Name",
		emailAttr:    "email",
		inResponseTo: "6zmm5mguyebwvajyf2sdwwcw6m",
		redirectURI:  "http://127.0.0.1:5556/dex/callback",
		wantErr:      true,
	}
	test.run(t)
}

func TestInvalidCA(t *testing.T) {
	test := responseTest{
		caFile:       "testdata/bad-ca.crt", // Not the CA that signed this response.
		respFile:     "testdata/good-resp.xml",
		now:          "2017-04-04T04:34:59.330Z",
		usernameAttr: "Name",
		emailAttr:    "email",
		inResponseTo: "6zmm5mguyebwvajyf2sdwwcw6m",
		redirectURI:  "http://127.0.0.1:5556/dex/callback",
		wantErr:      true,
	}
	test.run(t)
}

func TestUnsignedResponse(t *testing.T) {
	test := responseTest{
		caFile:       "testdata/ca.crt",
		respFile:     "testdata/good-resp.tmpl", // Use the unsigned template, not the signed document.
		now:          "2017-04-04T04:34:59.330Z",
		usernameAttr: "Name",
		emailAttr:    "email",
		inResponseTo: "6zmm5mguyebwvajyf2sdwwcw6m",
		redirectURI:  "http://127.0.0.1:5556/dex/callback",
		wantErr:      true,
	}
	test.run(t)
}

func TestExpiredAssertion(t *testing.T) {
	test := responseTest{
		caFile:       "testdata/ca.crt",
		respFile:     "testdata/assertion-signed.xml",
		now:          "2020-04-04T04:34:59.330Z", // Assertion has expired.
		usernameAttr: "Name",
		emailAttr:    "email",
		inResponseTo: "6zmm5mguyebwvajyf2sdwwcw6m",
		redirectURI:  "http://127.0.0.1:5556/dex/callback",
		wantErr:      true,
	}
	test.run(t)
}

// TestAssertionSignedNotResponse ensures the connector validates SAML 2.0
// responses where the assertion is signed but the root element, the
// response, isn't.
func TestAssertionSignedNotResponse(t *testing.T) {
	test := responseTest{
		caFile:       "testdata/ca.crt",
		respFile:     "testdata/assertion-signed.xml",
		now:          "2017-04-04T04:34:59.330Z",
		usernameAttr: "Name",
		emailAttr:    "email",
		inResponseTo: "6zmm5mguyebwvajyf2sdwwcw6m",
		redirectURI:  "http://127.0.0.1:5556/dex/callback",
		wantIdent: connector.Identity{
			UserID:        "eric.chiang+okta@coreos.com",
			Username:      "Eric",
			Email:         "eric.chiang+okta@coreos.com",
			EmailVerified: true,
		},
	}
	test.run(t)
}

// TestTwoAssertionFirstSigned tries to catch an edge case where an attacker
// provides a second assertion that's not signed.
func TestTwoAssertionFirstSigned(t *testing.T) {
	test := responseTest{
		caFile:       "testdata/ca.crt",
		respFile:     "testdata/two-assertions-first-signed.xml",
		now:          "2017-04-04T04:34:59.330Z",
		usernameAttr: "Name",
		emailAttr:    "email",
		inResponseTo: "6zmm5mguyebwvajyf2sdwwcw6m",
		redirectURI:  "http://127.0.0.1:5556/dex/callback",
		wantIdent: connector.Identity{
			UserID:        "eric.chiang+okta@coreos.com",
			Username:      "Eric",
			Email:         "eric.chiang+okta@coreos.com",
			EmailVerified: true,
		},
	}
	test.run(t)
}

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

func (r responseTest) run(t *testing.T) {
	c := Config{
		CA:           r.caFile,
		UsernameAttr: r.usernameAttr,
		EmailAttr:    r.emailAttr,
		GroupsAttr:   r.groupsAttr,
		RedirectURI:  r.redirectURI,
		// Never logging in, don't need this.
		SSOURL: "http://foo.bar/",
	}
	now, err := time.Parse(timeFormat, r.now)
	if err != nil {
		t.Fatalf("parse test time: %v", err)
	}

	conn, err := c.openConnector(logrus.New())
	if err != nil {
		t.Fatal(err)
	}
	conn.now = func() time.Time { return now }
	resp, err := ioutil.ReadFile(r.respFile)
	if err != nil {
		t.Fatal(err)
	}
	samlResp := base64.StdEncoding.EncodeToString(resp)

	scopes := connector.Scopes{
		OfflineAccess: false,
		Groups:        true,
	}
	ident, err := conn.HandlePOST(scopes, samlResp, r.inResponseTo)
	if err != nil {
		if !r.wantErr {
			t.Fatalf("handle response: %v", err)
		}
		return
	}

	if r.wantErr {
		t.Fatalf("wanted error")
	}
	sort.Strings(ident.Groups)
	sort.Strings(r.wantIdent.Groups)
	if diff := pretty.Compare(ident, r.wantIdent); diff != "" {
		t.Error(diff)
	}
}

const (
	defaultIssuer      = "http://www.okta.com/exk91cb99lKkKSYoy0h7"
	defaultRedirectURI = "http://localhost:5556/dex/callback"

	// Response ID embedded in our testdata.
	testDataResponseID = "_fd1b3ef9-ec09-44a7-a66b-0d39c250f6a0"
)

// Depricated: Use testing framework established above.
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

	if _, _, err := verifyResponseSig(validator, data); err != nil {
		if shouldSucceed {
			t.Fatal(err)
		}
	} else {
		if !shouldSucceed {
			t.Fatalf("expected an invalid signatrue but verification has been successful")
		}
	}
}

// Depricated: Use testing framework established above.
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
