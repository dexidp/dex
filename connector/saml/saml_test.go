package saml

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
	dsig "github.com/russellhaering/goxmldsig"

	"github.com/dexidp/dex/connector"
)

// responseTest maps a SAML 2.0 response object to a set of expected values.
//
// Tests are defined in the "testdata" directory and are self-signed using xmlsec1.
//
// To add a new test, define a new, unsigned SAML 2.0 response that exercises some
// case, then sign it using the "testdata/gen.sh" script.
//
//	cp testdata/good-resp.tmpl testdata/( testname ).tmpl
//	vim ( testname ).tmpl # Modify your template for your test case.
//	vim testdata/gen.sh   # Add a xmlsec1 command to the generation script.
//	./testdata/gen.sh     # Sign your template.
//
// To install xmlsec1 on Fedora run:
//
//	sudo dnf install xmlsec1 xmlsec1-openssl
//
// On mac:
//
//	brew install Libxmlsec1
type responseTest struct {
	// CA file and XML file of the response.
	caFile   string
	respFile string

	// Values that should be used to validate the signature.
	now          string
	inResponseTo string
	redirectURI  string
	entityIssuer string

	// Attribute customization.
	usernameAttr  string
	emailAttr     string
	groupsAttr    string
	allowedGroups []string
	filterGroups  bool

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

func TestGroupsWhitelist(t *testing.T) {
	test := responseTest{
		caFile:        "testdata/ca.crt",
		respFile:      "testdata/good-resp.xml",
		now:           "2017-04-04T04:34:59.330Z",
		usernameAttr:  "Name",
		emailAttr:     "email",
		groupsAttr:    "groups",
		allowedGroups: []string{"Admins"},
		inResponseTo:  "6zmm5mguyebwvajyf2sdwwcw6m",
		redirectURI:   "http://127.0.0.1:5556/dex/callback",
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

func TestGroupsWhitelistWithFiltering(t *testing.T) {
	test := responseTest{
		caFile:        "testdata/ca.crt",
		respFile:      "testdata/good-resp.xml",
		now:           "2017-04-04T04:34:59.330Z",
		usernameAttr:  "Name",
		emailAttr:     "email",
		groupsAttr:    "groups",
		allowedGroups: []string{"Admins"},
		filterGroups:  true,
		inResponseTo:  "6zmm5mguyebwvajyf2sdwwcw6m",
		redirectURI:   "http://127.0.0.1:5556/dex/callback",
		wantIdent: connector.Identity{
			UserID:        "eric.chiang+okta@coreos.com",
			Username:      "Eric",
			Email:         "eric.chiang+okta@coreos.com",
			EmailVerified: true,
			Groups:        []string{"Admins"}, // "Everyone" is filtered
		},
	}
	test.run(t)
}

func TestGroupsWhitelistEmpty(t *testing.T) {
	test := responseTest{
		caFile:        "testdata/ca.crt",
		respFile:      "testdata/good-resp.xml",
		now:           "2017-04-04T04:34:59.330Z",
		usernameAttr:  "Name",
		emailAttr:     "email",
		groupsAttr:    "groups",
		allowedGroups: []string{},
		inResponseTo:  "6zmm5mguyebwvajyf2sdwwcw6m",
		redirectURI:   "http://127.0.0.1:5556/dex/callback",
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

func TestGroupsWhitelistDisallowed(t *testing.T) {
	test := responseTest{
		wantErr:       true,
		caFile:        "testdata/ca.crt",
		respFile:      "testdata/good-resp.xml",
		now:           "2017-04-04T04:34:59.330Z",
		usernameAttr:  "Name",
		emailAttr:     "email",
		groupsAttr:    "groups",
		allowedGroups: []string{"Nope"},
		inResponseTo:  "6zmm5mguyebwvajyf2sdwwcw6m",
		redirectURI:   "http://127.0.0.1:5556/dex/callback",
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

func TestGroupsWhitelistDisallowedNoGroupsOnIdent(t *testing.T) {
	test := responseTest{
		wantErr:       true,
		caFile:        "testdata/ca.crt",
		respFile:      "testdata/good-resp.xml",
		now:           "2017-04-04T04:34:59.330Z",
		usernameAttr:  "Name",
		emailAttr:     "email",
		groupsAttr:    "groups",
		allowedGroups: []string{"Nope"},
		inResponseTo:  "6zmm5mguyebwvajyf2sdwwcw6m",
		redirectURI:   "http://127.0.0.1:5556/dex/callback",
		wantIdent: connector.Identity{
			UserID:        "eric.chiang+okta@coreos.com",
			Username:      "Eric",
			Email:         "eric.chiang+okta@coreos.com",
			EmailVerified: true,
			Groups:        []string{},
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

func TestInvalidSubjectInResponseTo(t *testing.T) {
	test := responseTest{
		caFile:       "testdata/ca.crt",
		respFile:     "testdata/assertion-signed.xml",
		now:          "2017-04-04T04:34:59.330Z",
		usernameAttr: "Name",
		emailAttr:    "email",
		inResponseTo: "invalid-id", // Bad InResponseTo value.
		redirectURI:  "http://127.0.0.1:5556/dex/callback",
		wantErr:      true,
	}
	test.run(t)
}

func TestInvalidSubjectRecipient(t *testing.T) {
	test := responseTest{
		caFile:       "testdata/ca.crt",
		respFile:     "testdata/assertion-signed.xml",
		now:          "2017-04-04T04:34:59.330Z",
		usernameAttr: "Name",
		emailAttr:    "email",
		inResponseTo: "6zmm5mguyebwvajyf2sdwwcw6m",
		redirectURI:  "http://bad.com/dex/callback", // Doesn't match Recipient value.
		wantErr:      true,
	}
	test.run(t)
}

func TestInvalidAssertionAudience(t *testing.T) {
	test := responseTest{
		caFile:       "testdata/ca.crt",
		respFile:     "testdata/assertion-signed.xml",
		now:          "2017-04-04T04:34:59.330Z",
		usernameAttr: "Name",
		emailAttr:    "email",
		inResponseTo: "6zmm5mguyebwvajyf2sdwwcw6m",
		redirectURI:  "http://127.0.0.1:5556/dex/callback",
		// EntityIssuer overrides RedirectURI when determining the expected
		// audience. In this case, ensure the audience is invalid.
		entityIssuer: "http://localhost:5556/dex/callback",
		wantErr:      true,
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

func TestTamperedResponseNameID(t *testing.T) {
	test := responseTest{
		caFile:       "testdata/ca.crt",
		respFile:     "testdata/tampered-resp.xml",
		now:          "2017-04-04T04:34:59.330Z",
		usernameAttr: "Name",
		emailAttr:    "email",
		inResponseTo: "6zmm5mguyebwvajyf2sdwwcw6m",
		redirectURI:  "http://127.0.0.1:5556/dex/callback",
		wantErr:      true,
	}
	test.run(t)
}

func loadCert(ca string) (*x509.Certificate, error) {
	data, err := os.ReadFile(ca)
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
		CA:            r.caFile,
		UsernameAttr:  r.usernameAttr,
		EmailAttr:     r.emailAttr,
		GroupsAttr:    r.groupsAttr,
		RedirectURI:   r.redirectURI,
		EntityIssuer:  r.entityIssuer,
		AllowedGroups: r.allowedGroups,
		FilterGroups:  r.filterGroups,
		// Never logging in, don't need this.
		SSOURL: "http://foo.bar/",
	}
	now, err := time.Parse(timeFormat, r.now)
	if err != nil {
		t.Fatalf("parse test time: %v", err)
	}

	conn, err := c.openConnector(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	conn.now = func() time.Time { return now }
	resp, err := os.ReadFile(r.respFile)
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

	// Verify ConnectorData contains valid cached identity, then clear it
	// for the main identity comparison (ConnectorData is an implementation
	// detail of refresh token support).
	if len(ident.ConnectorData) > 0 {
		var ci cachedIdentity
		if err := json.Unmarshal(ident.ConnectorData, &ci); err != nil {
			t.Fatalf("failed to unmarshal ConnectorData: %v", err)
		}
		if ci.UserID != ident.UserID {
			t.Errorf("cached identity UserID mismatch: got %q, want %q", ci.UserID, ident.UserID)
		}
		if ci.Email != ident.Email {
			t.Errorf("cached identity Email mismatch: got %q, want %q", ci.Email, ident.Email)
		}
	}
	ident.ConnectorData = nil

	if diff := pretty.Compare(ident, r.wantIdent); diff != "" {
		t.Error(diff)
	}
}

func TestConfigCAData(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	validPEM, err := os.ReadFile("testdata/ca.crt")
	if err != nil {
		t.Fatal(err)
	}
	valid2ndPEM, err := os.ReadFile("testdata/okta-ca.pem")
	if err != nil {
		t.Fatal(err)
	}

	// copy helper, avoid messing with the byte slice among different cases
	c := func(bs []byte) []byte {
		return append([]byte(nil), bs...)
	}

	tests := []struct {
		name    string
		caData  []byte
		wantErr bool
	}{
		{
			name:   "one valid PEM entry",
			caData: c(validPEM),
		},
		{
			name:   "one valid PEM entry with trailing newline",
			caData: append(c(validPEM), []byte("\n")...),
		},
		{
			name:   "one valid PEM entry with trailing spaces",
			caData: append(c(validPEM), []byte("   ")...),
		},
		{
			name:   "one valid PEM entry with two trailing newlines",
			caData: append(c(validPEM), []byte("\n\n")...),
		},
		{
			name:   "two valid PEM entries",
			caData: append(c(validPEM), c(valid2ndPEM)...),
		},
		{
			name:   "two valid PEM entries with newline in between",
			caData: append(append(c(validPEM), []byte("\n")...), c(valid2ndPEM)...),
		},
		{
			name:   "two valid PEM entries with trailing newline",
			caData: append(c(valid2ndPEM), append(c(validPEM), []byte("\n")...)...),
		},
		{
			name:    "empty",
			caData:  []byte{},
			wantErr: true,
		},
		{
			name:    "one valid PEM entry with trailing data",
			caData:  append(c(validPEM), []byte("yaddayadda")...),
			wantErr: true,
		},
		{
			name:    "one valid PEM entry with bad data before",
			caData:  append([]byte("yaddayadda"), c(validPEM)...),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := Config{
				CAData:       tc.caData,
				UsernameAttr: "user",
				EmailAttr:    "email",
				RedirectURI:  "http://127.0.0.1:5556/dex/callback",
				SSOURL:       "http://foo.bar/",
			}
			_, err := (&c).Open("samltest", logger)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

// Deprecated: Use testing framework established above.
func runVerify(t *testing.T, ca string, resp string, shouldSucceed bool) {
	cert, err := loadCert(ca)
	if err != nil {
		t.Fatal(err)
	}
	s := certStore{[]*x509.Certificate{cert}}

	validator := dsig.NewDefaultValidationContext(s)

	data, err := os.ReadFile(resp)
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := verifyResponseSig(validator, data); err != nil {
		if shouldSucceed {
			t.Fatal(err)
		}
	} else {
		if !shouldSucceed {
			t.Fatalf("expected an invalid signature but verification has been successful")
		}
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

func TestSAMLRefresh(t *testing.T) {
	// Create a provider using the same pattern as existing tests.
	c := Config{
		CA:           "testdata/ca.crt",
		UsernameAttr: "Name",
		EmailAttr:    "email",
		GroupsAttr:   "groups",
		RedirectURI:  "http://127.0.0.1:5556/dex/callback",
		SSOURL:       "http://foo.bar/",
	}

	conn, err := c.openConnector(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("SuccessfulRefresh", func(t *testing.T) {
		ci := cachedIdentity{
			UserID:            "test-user-id",
			Username:          "testuser",
			PreferredUsername: "testuser",
			Email:             "test@example.com",
			EmailVerified:     true,
			Groups:            []string{"group1", "group2"},
		}
		connectorData, err := json.Marshal(ci)
		if err != nil {
			t.Fatal(err)
		}

		ident := connector.Identity{
			UserID:        "old-id",
			Username:      "old-name",
			ConnectorData: connectorData,
		}

		refreshed, err := conn.Refresh(context.Background(), connector.Scopes{Groups: true}, ident)
		if err != nil {
			t.Fatalf("Refresh failed: %v", err)
		}

		if refreshed.UserID != "test-user-id" {
			t.Errorf("expected UserID %q, got %q", "test-user-id", refreshed.UserID)
		}
		if refreshed.Username != "testuser" {
			t.Errorf("expected Username %q, got %q", "testuser", refreshed.Username)
		}
		if refreshed.PreferredUsername != "testuser" {
			t.Errorf("expected PreferredUsername %q, got %q", "testuser", refreshed.PreferredUsername)
		}
		if refreshed.Email != "test@example.com" {
			t.Errorf("expected Email %q, got %q", "test@example.com", refreshed.Email)
		}
		if !refreshed.EmailVerified {
			t.Error("expected EmailVerified to be true")
		}
		if len(refreshed.Groups) != 2 || refreshed.Groups[0] != "group1" || refreshed.Groups[1] != "group2" {
			t.Errorf("expected groups [group1, group2], got %v", refreshed.Groups)
		}
		// ConnectorData should be preserved through refresh
		if len(refreshed.ConnectorData) == 0 {
			t.Error("expected ConnectorData to be preserved")
		}
	})

	t.Run("RefreshPreservesConnectorData", func(t *testing.T) {
		ci := cachedIdentity{
			UserID:        "user-123",
			Username:      "alice",
			Email:         "alice@example.com",
			EmailVerified: true,
		}
		connectorData, err := json.Marshal(ci)
		if err != nil {
			t.Fatal(err)
		}

		ident := connector.Identity{
			UserID:        "old-id",
			ConnectorData: connectorData,
		}

		refreshed, err := conn.Refresh(context.Background(), connector.Scopes{}, ident)
		if err != nil {
			t.Fatalf("Refresh failed: %v", err)
		}

		// Verify the refreshed identity can be refreshed again (round-trip)
		var roundTrip cachedIdentity
		if err := json.Unmarshal(refreshed.ConnectorData, &roundTrip); err != nil {
			t.Fatalf("failed to unmarshal ConnectorData after refresh: %v", err)
		}
		if roundTrip.UserID != "user-123" {
			t.Errorf("round-trip UserID mismatch: got %q, want %q", roundTrip.UserID, "user-123")
		}
	})

	t.Run("EmptyConnectorData", func(t *testing.T) {
		ident := connector.Identity{
			UserID:        "test-id",
			ConnectorData: nil,
		}
		_, err := conn.Refresh(context.Background(), connector.Scopes{}, ident)
		if err == nil {
			t.Error("expected error for empty ConnectorData")
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		ident := connector.Identity{
			UserID:        "test-id",
			ConnectorData: []byte("not-json"),
		}
		_, err := conn.Refresh(context.Background(), connector.Scopes{}, ident)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("HandlePOSTThenRefresh", func(t *testing.T) {
		// Full integration: HandlePOST → get ConnectorData → Refresh → verify identity
		now, err := time.Parse(timeFormat, "2017-04-04T04:34:59.330Z")
		if err != nil {
			t.Fatal(err)
		}
		conn.now = func() time.Time { return now }

		resp, err := os.ReadFile("testdata/good-resp.xml")
		if err != nil {
			t.Fatal(err)
		}
		samlResp := base64.StdEncoding.EncodeToString(resp)

		scopes := connector.Scopes{
			OfflineAccess: true,
			Groups:        true,
		}
		ident, err := conn.HandlePOST(scopes, samlResp, "6zmm5mguyebwvajyf2sdwwcw6m")
		if err != nil {
			t.Fatalf("HandlePOST failed: %v", err)
		}

		if len(ident.ConnectorData) == 0 {
			t.Fatal("expected ConnectorData to be set after HandlePOST")
		}

		// Now refresh using the ConnectorData from HandlePOST
		refreshed, err := conn.Refresh(context.Background(), scopes, ident)
		if err != nil {
			t.Fatalf("Refresh failed: %v", err)
		}

		if refreshed.UserID != ident.UserID {
			t.Errorf("UserID mismatch: got %q, want %q", refreshed.UserID, ident.UserID)
		}
		if refreshed.Username != ident.Username {
			t.Errorf("Username mismatch: got %q, want %q", refreshed.Username, ident.Username)
		}
		if refreshed.Email != ident.Email {
			t.Errorf("Email mismatch: got %q, want %q", refreshed.Email, ident.Email)
		}
		if refreshed.EmailVerified != ident.EmailVerified {
			t.Errorf("EmailVerified mismatch: got %v, want %v", refreshed.EmailVerified, ident.EmailVerified)
		}
		sort.Strings(refreshed.Groups)
		sort.Strings(ident.Groups)
		if len(refreshed.Groups) != len(ident.Groups) {
			t.Errorf("Groups length mismatch: got %d, want %d", len(refreshed.Groups), len(ident.Groups))
		}
		for i := range ident.Groups {
			if i < len(refreshed.Groups) && refreshed.Groups[i] != ident.Groups[i] {
				t.Errorf("Groups[%d] mismatch: got %q, want %q", i, refreshed.Groups[i], ident.Groups[i])
			}
		}
	})

	t.Run("HandlePOSTThenDoubleRefresh", func(t *testing.T) {
		// Verify that refresh tokens can be chained: HandlePOST → Refresh → Refresh
		now, err := time.Parse(timeFormat, "2017-04-04T04:34:59.330Z")
		if err != nil {
			t.Fatal(err)
		}
		conn.now = func() time.Time { return now }

		resp, err := os.ReadFile("testdata/good-resp.xml")
		if err != nil {
			t.Fatal(err)
		}
		samlResp := base64.StdEncoding.EncodeToString(resp)

		scopes := connector.Scopes{OfflineAccess: true, Groups: true}
		ident, err := conn.HandlePOST(scopes, samlResp, "6zmm5mguyebwvajyf2sdwwcw6m")
		if err != nil {
			t.Fatalf("HandlePOST failed: %v", err)
		}

		// First refresh
		refreshed1, err := conn.Refresh(context.Background(), scopes, ident)
		if err != nil {
			t.Fatalf("first Refresh failed: %v", err)
		}
		if len(refreshed1.ConnectorData) == 0 {
			t.Fatal("expected ConnectorData after first refresh")
		}

		// Second refresh using output of first refresh
		refreshed2, err := conn.Refresh(context.Background(), scopes, refreshed1)
		if err != nil {
			t.Fatalf("second Refresh failed: %v", err)
		}

		// All fields should match original
		if refreshed2.UserID != ident.UserID {
			t.Errorf("UserID mismatch after double refresh: got %q, want %q", refreshed2.UserID, ident.UserID)
		}
		if refreshed2.Email != ident.Email {
			t.Errorf("Email mismatch after double refresh: got %q, want %q", refreshed2.Email, ident.Email)
		}
		if refreshed2.Username != ident.Username {
			t.Errorf("Username mismatch after double refresh: got %q, want %q", refreshed2.Username, ident.Username)
		}
	})

	t.Run("HandlePOSTWithAssertionSignedThenRefresh", func(t *testing.T) {
		// Test with assertion-signed.xml (signature on assertion, not response)
		now, err := time.Parse(timeFormat, "2017-04-04T04:34:59.330Z")
		if err != nil {
			t.Fatal(err)
		}
		conn.now = func() time.Time { return now }

		resp, err := os.ReadFile("testdata/assertion-signed.xml")
		if err != nil {
			t.Fatal(err)
		}
		samlResp := base64.StdEncoding.EncodeToString(resp)

		scopes := connector.Scopes{OfflineAccess: true, Groups: true}
		ident, err := conn.HandlePOST(scopes, samlResp, "6zmm5mguyebwvajyf2sdwwcw6m")
		if err != nil {
			t.Fatalf("HandlePOST with assertion-signed failed: %v", err)
		}

		if len(ident.ConnectorData) == 0 {
			t.Fatal("expected ConnectorData after HandlePOST with assertion-signed")
		}

		refreshed, err := conn.Refresh(context.Background(), scopes, ident)
		if err != nil {
			t.Fatalf("Refresh after assertion-signed HandlePOST failed: %v", err)
		}

		if refreshed.Email != ident.Email {
			t.Errorf("Email mismatch: got %q, want %q", refreshed.Email, ident.Email)
		}
		if refreshed.Username != ident.Username {
			t.Errorf("Username mismatch: got %q, want %q", refreshed.Username, ident.Username)
		}
	})

	t.Run("HandlePOSTRefreshWithoutGroupsScope", func(t *testing.T) {
		// Verify that groups are NOT returned when groups scope is not requested during refresh
		now, err := time.Parse(timeFormat, "2017-04-04T04:34:59.330Z")
		if err != nil {
			t.Fatal(err)
		}
		conn.now = func() time.Time { return now }

		resp, err := os.ReadFile("testdata/good-resp.xml")
		if err != nil {
			t.Fatal(err)
		}
		samlResp := base64.StdEncoding.EncodeToString(resp)

		// Initial auth WITH groups
		scopesWithGroups := connector.Scopes{OfflineAccess: true, Groups: true}
		ident, err := conn.HandlePOST(scopesWithGroups, samlResp, "6zmm5mguyebwvajyf2sdwwcw6m")
		if err != nil {
			t.Fatalf("HandlePOST failed: %v", err)
		}
		if len(ident.Groups) == 0 {
			t.Fatal("expected groups in initial identity")
		}

		// Refresh WITHOUT groups scope
		scopesNoGroups := connector.Scopes{OfflineAccess: true, Groups: false}
		refreshed, err := conn.Refresh(context.Background(), scopesNoGroups, ident)
		if err != nil {
			t.Fatalf("Refresh failed: %v", err)
		}

		if len(refreshed.Groups) != 0 {
			t.Errorf("expected no groups when groups scope not requested, got %v", refreshed.Groups)
		}

		// Refresh WITH groups scope — groups should be back
		refreshedWithGroups, err := conn.Refresh(context.Background(), scopesWithGroups, ident)
		if err != nil {
			t.Fatalf("Refresh with groups failed: %v", err)
		}

		if len(refreshedWithGroups.Groups) == 0 {
			t.Error("expected groups when groups scope is requested")
		}
	})
}

func TestSAMLHandleSLO(t *testing.T) {
	c := Config{
		CA:                                 "testdata/ca.crt",
		UsernameAttr:                       "Name",
		EmailAttr:                          "email",
		RedirectURI:                        "http://127.0.0.1:5556/dex/callback",
		SSOURL:                             "http://foo.bar/",
		InsecureSkipSLOSignatureValidation: true,
	}

	conn, err := c.openConnector(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}

	// Helper to create a LogoutRequest XML
	makeLogoutRequest := func(nameID string) string {
		return fmt.Sprintf(`<samlp:LogoutRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_test123" Version="2.0" IssueInstant="2024-01-01T00:00:00Z">
	<saml:Issuer>https://idp.example.com</saml:Issuer>
	<saml:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">%s</saml:NameID>
</samlp:LogoutRequest>`, nameID)
	}

	t.Run("ValidLogoutRequest", func(t *testing.T) {
		logoutXML := makeLogoutRequest("user@example.com")
		encoded := base64.StdEncoding.EncodeToString([]byte(logoutXML))

		form := url.Values{}
		form.Set("SAMLRequest", encoded)

		req := httptest.NewRequest(http.MethodPost, "/saml/slo/test", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		nameID, err := conn.HandleSLO(w, req)
		if err != nil {
			t.Fatalf("HandleSLO failed: %v", err)
		}
		if nameID != "user@example.com" {
			t.Errorf("expected nameID %q, got %q", "user@example.com", nameID)
		}
	})

	t.Run("MissingSAMLRequest", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/saml/slo/test", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		_, err := conn.HandleSLO(w, req)
		if err == nil {
			t.Error("expected error for missing SAMLRequest")
		}
	})

	t.Run("InvalidBase64", func(t *testing.T) {
		form := url.Values{}
		form.Set("SAMLRequest", "not-valid-base64!!!")

		req := httptest.NewRequest(http.MethodPost, "/saml/slo/test", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		_, err := conn.HandleSLO(w, req)
		if err == nil {
			t.Error("expected error for invalid base64")
		}
	})

	t.Run("InvalidXML", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("not xml at all"))
		form := url.Values{}
		form.Set("SAMLRequest", encoded)

		req := httptest.NewRequest(http.MethodPost, "/saml/slo/test", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		_, err := conn.HandleSLO(w, req)
		if err == nil {
			t.Error("expected error for invalid XML")
		}
	})

	t.Run("MissingNameID", func(t *testing.T) {
		logoutXML := `<samlp:LogoutRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_test123" Version="2.0" IssueInstant="2024-01-01T00:00:00Z">
	<saml:Issuer>https://idp.example.com</saml:Issuer>
	<saml:NameID></saml:NameID>
</samlp:LogoutRequest>`
		encoded := base64.StdEncoding.EncodeToString([]byte(logoutXML))

		form := url.Values{}
		form.Set("SAMLRequest", encoded)

		req := httptest.NewRequest(http.MethodPost, "/saml/slo/test", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		_, err := conn.HandleSLO(w, req)
		if err == nil {
			t.Error("expected error for missing NameID")
		}
	})

	t.Run("WrongHTTPMethod", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/saml/slo/test", nil)
		w := httptest.NewRecorder()

		_, err := conn.HandleSLO(w, req)
		if err == nil {
			t.Error("expected error for GET method")
		}
	})

	t.Run("DifferentNameIDValues", func(t *testing.T) {
		testCases := []struct {
			name       string
			nameIDVal  string
			wantNameID string
		}{
			{"email format", "admin@corp.example.com", "admin@corp.example.com"},
			{"persistent ID", "AQIC5w...", "AQIC5w..."},
			{"transient ID", "_ce3d2948b4cf20146dee0a0b3dd6f69b6cf86f62d7", "_ce3d2948b4cf20146dee0a0b3dd6f69b6cf86f62d7"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				logoutXML := makeLogoutRequest(tc.nameIDVal)
				encoded := base64.StdEncoding.EncodeToString([]byte(logoutXML))

				form := url.Values{}
				form.Set("SAMLRequest", encoded)

				req := httptest.NewRequest(http.MethodPost, "/saml/slo/test", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				w := httptest.NewRecorder()

				nameID, err := conn.HandleSLO(w, req)
				if err != nil {
					t.Fatalf("HandleSLO failed: %v", err)
				}
				if nameID != tc.wantNameID {
					t.Errorf("expected nameID %q, got %q", tc.wantNameID, nameID)
				}
			})
		}
	})
}
