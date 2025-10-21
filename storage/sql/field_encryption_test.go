package sql

import (
	"encoding/json"
	"log/slog"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/dexidp/dex/connector/atlassiancrowd"
	"github.com/dexidp/dex/connector/bitbucketcloud"
	"github.com/dexidp/dex/connector/gitea"
	"github.com/dexidp/dex/connector/github"
	"github.com/dexidp/dex/connector/gitlab"
	"github.com/dexidp/dex/connector/google"
	"github.com/dexidp/dex/connector/keystone"
	"github.com/dexidp/dex/connector/ldap"
	"github.com/dexidp/dex/connector/linkedin"
	"github.com/dexidp/dex/connector/microsoft"
	"github.com/dexidp/dex/connector/oauth"
	"github.com/dexidp/dex/connector/oidc"
	"github.com/dexidp/dex/connector/openshift"
	"github.com/dexidp/dex/connector/saml"
)

// Test encryption key (base64-encoded 32-byte Fernet key)
const testFernetKey = "cHxZB8z3TcK9mR6vL2nY5qW8sD1fG4hJ7kM0oP3rT6u="

// connectorTestCase represents a test case for a single connector type
type connectorTestCase struct {
	connectorType           string
	config                  any
	expectedSensitiveFields []string
}

func TestFieldEncryption_AllConnectors(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create encryption service with test key
	svc, err := newEncryptionService([]string{testFernetKey}, true, logger)
	if err != nil {
		t.Fatalf("failed to create encryption service: %v", err)
	}

	// Get all connector test cases
	testCases := getAllConnectorTestCases()

	// Run tests for each connector
	for _, tc := range testCases {
		t.Run(tc.connectorType, func(t *testing.T) {
			testConnectorFieldEncryptionAndDecryption(t, svc, tc)
		})
	}
}

func testConnectorFieldEncryptionAndDecryption(t *testing.T, svc *encryptionService, tc connectorTestCase) {
	t.Helper()

	// Verify service discovered the correct sensitive fields
	discoveredFields := svc.sensitiveFields[tc.connectorType]
	assertSensitiveFieldsMatch(t, tc.expectedSensitiveFields, discoveredFields)

	// Marshal config struct to JSON
	configJSON, err := json.Marshal(tc.config)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	// Encrypt the config fields
	encryptedJSON, err := svc.encryptFields(tc.connectorType, configJSON)
	if err != nil {
		t.Fatalf("encryption of connector fields failed: %v", err)
	}

	// Verify sensitive fields are encrypted
	assertFieldsAreEncrypted(t, encryptedJSON, discoveredFields)

	// Decrypt the config fields
	decryptedJSON, err := svc.decryptFields(tc.connectorType, encryptedJSON)
	if err != nil {
		t.Fatalf("decryption of connector fields failed: %v", err)
	}

	// Verify decrypted config matches original
	assertJSONEquals(t, configJSON, decryptedJSON)

	t.Logf("✓ %s: field encryption and decryption successful", tc.connectorType)
}

func getAllConnectorTestCases() []connectorTestCase {
	return []connectorTestCase{
		createAtlassianCrowdTestCase(),
		createBitbucketCloudTestCase(),
		createGiteaTestCase(),
		createGitHubTestCase(),
		createGitLabTestCase(),
		createGoogleTestCase(),
		createKeystoneTestCase(),
		createLDAPTestCase(),
		createLinkedInTestCase(),
		createMicrosoftTestCase(),
		createOAuthTestCase(),
		createOIDCTestCase(),
		createOpenShiftTestCase(),
		createSAMLTestCase(),
	}
}

func createAtlassianCrowdTestCase() connectorTestCase {
	return connectorTestCase{
		connectorType: "atlassian-crowd",
		config: atlassiancrowd.Config{
			ClientID:     "ac-client-id",
			ClientSecret: "crowd-secret-password",
		},
		expectedSensitiveFields: []string{"clientSecret"},
	}
}

func createBitbucketCloudTestCase() connectorTestCase {
	return connectorTestCase{
		connectorType: "bitbucket-cloud",
		config: bitbucketcloud.Config{
			ClientID:     "bitbucket-client-id",
			ClientSecret: "bitbucket-client-secret",
			RedirectURI:  "https://dex.example.com/callback",
		},
		expectedSensitiveFields: []string{"clientSecret"},
	}
}

func createGiteaTestCase() connectorTestCase {
	return connectorTestCase{
		connectorType: "gitea",
		config: gitea.Config{
			BaseURL:      "https://gitea.example.com",
			ClientID:     "gitea-client-id",
			ClientSecret: "gitea-client-secret",
			RedirectURI:  "https://dex.example.com/callback",
		},
		expectedSensitiveFields: []string{"clientSecret"},
	}
}

func createGitHubTestCase() connectorTestCase {
	return connectorTestCase{
		connectorType: "github",
		config: github.Config{
			ClientID:     "github-client-id",
			ClientSecret: "github-client-secret-super-long-value",
			RedirectURI:  "https://dex.example.com/callback",
		},
		expectedSensitiveFields: []string{"clientSecret"},
	}
}

func createGitLabTestCase() connectorTestCase {
	return connectorTestCase{
		connectorType: "gitlab",
		config: gitlab.Config{
			BaseURL:      "https://gitlab.example.com",
			ClientID:     "gitlab-client-id",
			ClientSecret: "gitlab-client-secret",
			RedirectURI:  "https://dex.example.com/callback",
		},
		expectedSensitiveFields: []string{"clientSecret"},
	}
}

func createGoogleTestCase() connectorTestCase {
	return connectorTestCase{
		connectorType: "google",
		config: google.Config{
			ClientID:     "google-client-id.apps.googleusercontent.com",
			ClientSecret: "google-client-secret-GOCSPX",
			RedirectURI:  "https://dex.example.com/callback",
		},
		expectedSensitiveFields: []string{"clientSecret"},
	}
}

func createKeystoneTestCase() connectorTestCase {
	return connectorTestCase{
		connectorType: "keystone",
		config: keystone.Config{
			Domain:        "default",
			AdminUsername: "admin",
			AdminPassword: "keystone-admin-password",
		},
		expectedSensitiveFields: []string{"keystonePassword"},
	}
}

func createLDAPTestCase() connectorTestCase {
	return connectorTestCase{
		connectorType: "ldap",
		config: ldap.Config{
			Host:   "ldap.example.com:636",
			BindDN: "cn=admin,dc=example,dc=com",
			BindPW: "ldap-bind-password-secret",
		},
		expectedSensitiveFields: []string{"bindPW"},
	}
}

func createLinkedInTestCase() connectorTestCase {
	return connectorTestCase{
		connectorType: "linkedin",
		config: linkedin.Config{
			ClientID:     "linkedin-client-id",
			ClientSecret: "linkedin-client-secret",
			RedirectURI:  "https://dex.example.com/callback",
		},
		expectedSensitiveFields: []string{"clientSecret"},
	}
}

func createMicrosoftTestCase() connectorTestCase {
	return connectorTestCase{
		connectorType: "microsoft",
		config: microsoft.Config{
			ClientID:     "microsoft-client-id",
			ClientSecret: "microsoft-client-secret",
			RedirectURI:  "https://dex.example.com/callback",
			Tenant:       "common",
		},
		expectedSensitiveFields: []string{"clientSecret"},
	}
}

func createOAuthTestCase() connectorTestCase {
	return connectorTestCase{
		connectorType: "oauth",
		config: oauth.Config{
			ClientID:         "oauth-client-id",
			ClientSecret:     "oauth-client-secret",
			RedirectURI:      "https://dex.example.com/callback",
			TokenURL:         "https://provider.example.com/token",
			AuthorizationURL: "https://provider.example.com/authorize",
			UserInfoURL:      "https://provider.example.com/userinfo",
		},
		expectedSensitiveFields: []string{"clientSecret"},
	}
}

func createOIDCTestCase() connectorTestCase {
	return connectorTestCase{
		connectorType: "oidc",
		config: oidc.Config{
			Issuer:       "https://accounts.google.com",
			ClientID:     "oidc-client-id",
			ClientSecret: "oidc-client-secret-value-123",
			RedirectURI:  "https://dex.example.com/callback",
		},
		expectedSensitiveFields: []string{"clientSecret"},
	}
}

func createOpenShiftTestCase() connectorTestCase {
	return connectorTestCase{
		connectorType: "openshift",
		config: openshift.Config{
			Issuer:       "https://openshift.example.com",
			ClientID:     "openshift-client-id",
			ClientSecret: "openshift-client-secret",
			RedirectURI:  "https://dex.example.com/callback",
		},
		expectedSensitiveFields: []string{"clientSecret"},
	}
}
func createSAMLTestCase() connectorTestCase {
	return connectorTestCase{
		connectorType: "saml",
		config: saml.Config{
			SSOURL:      "https://saml.example.com/sso",
			RedirectURI: "https://dex.example.com/callback",
		},
		// No sensitive fields marked
	}
}

func sortedCopy(slice []string) []string {
	result := make([]string, len(slice))
	copy(result, slice)
	sort.Strings(result)
	return result
}

func assertSensitiveFieldsMatch(t *testing.T, expected, actual []string) {
	t.Helper()

	sortedExpected := sortedCopy(expected)
	sortedActual := sortedCopy(actual)

	if !reflect.DeepEqual(sortedExpected, sortedActual) {
		t.Errorf("Sensitive fields mismatch:\n  Expected: %v\n  Discovered: %v",
			sortedExpected, sortedActual)
	}
}

func assertFieldsAreEncrypted(t *testing.T, jsonData []byte, fieldNames []string) {
	t.Helper()

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	for _, field := range fieldNames {
		value, exists := data[field]
		if !exists {
			continue
		}

		strVal, ok := value.(string)
		if !ok || strVal == "" {
			continue
		}

		if !isEncrypted(strVal) {
			t.Errorf("field %q should be encrypted but is not: %s", field, strVal)
		}
	}
}

func assertJSONEquals(t *testing.T, expected, actual []byte) {
	t.Helper()

	var expMap, actMap map[string]any

	if err := json.Unmarshal(expected, &expMap); err != nil {
		t.Fatalf("failed to expected config json: %v", err)
	}

	if err := json.Unmarshal(actual, &actMap); err != nil {
		t.Fatalf("failed to actual config json: %v", err)
	}

	if !reflect.DeepEqual(expMap, actMap) {
		t.Errorf("JSON mismatch:\n  Expected: %+v\n  Actual: %+v", expMap, actMap)
	}
}
