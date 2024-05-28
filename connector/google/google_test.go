package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/option"

	"github.com/dexidp/dex/connector"
)

var (
	//			groups_0
	//		┌───────┤
	//  groups_2 groups_1
	//		│		├────────┐
	//		└──	 user_1	  user_2
	testGroups = map[string][]*admin.Group{
		"user_1@dexidp.com":   {{Email: "groups_2@dexidp.com"}, {Email: "groups_1@dexidp.com"}},
		"user_2@dexidp.com":   {{Email: "groups_1@dexidp.com"}},
		"groups_1@dexidp.com": {{Email: "groups_0@dexidp.com"}},
		"groups_2@dexidp.com": {{Email: "groups_0@dexidp.com"}},
		"groups_0@dexidp.com": {},
	}
	callCounter = make(map[string]int)
)

func testSetup() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/admin/directory/v1/groups/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		userKey := r.URL.Query().Get("userKey")
		if groups, ok := testGroups[userKey]; ok {
			json.NewEncoder(w).Encode(admin.Groups{Groups: groups})
			callCounter[userKey]++
		}
	})

	return httptest.NewServer(mux)
}

func newConnector(config *Config) (*googleConnector, error) {
	log := logrus.New()
	conn, err := config.Open("id", log)
	if err != nil {
		return nil, err
	}

	googleConn, ok := conn.(*googleConnector)
	if !ok {
		return nil, fmt.Errorf("failed to convert to googleConnector")
	}
	return googleConn, nil
}

func tempServiceAccountKey() (string, error) {
	fd, err := os.CreateTemp("", "google_service_account_key")
	if err != nil {
		return "", err
	}
	defer fd.Close()
	err = json.NewEncoder(fd).Encode(map[string]string{
		"type":                 "service_account",
		"project_id":           "sample-project",
		"private_key_id":       "sample-key-id",
		"private_key":          "-----BEGIN PRIVATE KEY-----\nsample-key\n-----END PRIVATE KEY-----\n",
		"client_id":            "sample-client-id",
		"client_x509_cert_url": "localhost",
	})
	return fd.Name(), err
}

func tempWorkloadIdentityFederation() (string, error) {
	fd, err := os.CreateTemp("", "workload_identity_federation")
	if err != nil {
		return "", err
	}
	defer fd.Close()
	err = json.NewEncoder(fd).Encode(map[string]any{
		"type":                              "external_account",
		"audience":                          "//iam.googleapis.com/projects/111111111111/locations/global/workloadIdentityPools/aws-pool/providers/aws-provider",
		"subject_token_type":                "urn:ietf:params:aws:token-type:aws4_request",
		"service_account_impersonation_url": "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/sa@my-google-project.iam.gserviceaccount.com:generateAccessToken",
		"token_url":                         "https://sts.googleapis.com/v1/token",
		"credential_source": map[string]string{
			"environment_id":                 "aws1",
			"region_url":                     "http://169.254.169.254/latest/meta-data/placement/availability-zone",
			"url":                            "http://169.254.169.254/latest/meta-data/iam/security-credentials",
			"regional_cred_verification_url": "https://sts.{region}.amazonaws.com?Action=GetCallerIdentity&Version=2011-06-15",
		},
	})
	return fd.Name(), err
}

func TestOpen(t *testing.T) {
	ts := testSetup()
	defer ts.Close()

	type testCase struct {
		config      *Config
		expectedErr string

		// string to set in GOOGLE_APPLICATION_CREDENTIALS. As local development environments can
		// already contain ADC, test cases will be built upon this setting this env variable
		adc string
	}

	serviceAccountFilePath, err := tempServiceAccountKey()
	assert.Nil(t, err)

	for name, reference := range map[string]testCase{
		"missing_admin_email": {
			config: &Config{
				ClientID:               "testClient",
				ClientSecret:           "testSecret",
				RedirectURI:            ts.URL + "/callback",
				Scopes:                 []string{"openid", "groups"},
				ServiceAccountFilePath: serviceAccountFilePath,
			},
			expectedErr: "requires the domainToAdminEmail",
		},
		"service_account_key_not_found": {
			config: &Config{
				ClientID:               "testClient",
				ClientSecret:           "testSecret",
				RedirectURI:            ts.URL + "/callback",
				Scopes:                 []string{"openid", "groups"},
				DomainToAdminEmail:     map[string]string{"*": "foo@bar.com"},
				ServiceAccountFilePath: "not_found.json",
			},
			expectedErr: "error reading credentials",
		},
		"service_account_key_valid": {
			config: &Config{
				ClientID:               "testClient",
				ClientSecret:           "testSecret",
				RedirectURI:            ts.URL + "/callback",
				Scopes:                 []string{"openid", "groups"},
				DomainToAdminEmail:     map[string]string{"bar.com": "foo@bar.com"},
				ServiceAccountFilePath: serviceAccountFilePath,
			},
			expectedErr: "",
		},
		"adc": {
			config: &Config{
				ClientID:           "testClient",
				ClientSecret:       "testSecret",
				RedirectURI:        ts.URL + "/callback",
				Scopes:             []string{"openid", "groups"},
				DomainToAdminEmail: map[string]string{"*": "foo@bar.com"},
			},
			adc:         serviceAccountFilePath,
			expectedErr: "",
		},
		"adc_priority": {
			config: &Config{
				ClientID:               "testClient",
				ClientSecret:           "testSecret",
				RedirectURI:            ts.URL + "/callback",
				Scopes:                 []string{"openid", "groups"},
				DomainToAdminEmail:     map[string]string{"*": "foo@bar.com"},
				ServiceAccountFilePath: serviceAccountFilePath,
			},
			adc:         "/dev/null",
			expectedErr: "",
		},
	} {
		reference := reference
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", reference.adc)
			conn, err := newConnector(reference.config)

			if reference.expectedErr == "" {
				assert.Nil(err)
				assert.NotNil(conn)
			} else {
				assert.ErrorContains(err, reference.expectedErr)
			}
		})
	}
}

func TestGetGroups(t *testing.T) {
	ts := testSetup()
	defer ts.Close()

	serviceAccountFilePath, err := tempServiceAccountKey()
	assert.Nil(t, err)

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", serviceAccountFilePath)
	conn, err := newConnector(&Config{
		ClientID:           "testClient",
		ClientSecret:       "testSecret",
		RedirectURI:        ts.URL + "/callback",
		Scopes:             []string{"openid", "groups"},
		DomainToAdminEmail: map[string]string{"*": "admin@dexidp.com"},
	})
	assert.Nil(t, err)

	conn.adminSrv[wildcardDomainToAdminEmail], err = admin.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(ts.URL))
	assert.Nil(t, err)
	type testCase struct {
		userKey                        string
		fetchTransitiveGroupMembership bool
		shouldErr                      bool
		expectedGroups                 []string
	}

	for name, testCase := range map[string]testCase{
		"user1_non_transitive_lookup": {
			userKey:                        "user_1@dexidp.com",
			fetchTransitiveGroupMembership: false,
			shouldErr:                      false,
			expectedGroups:                 []string{"groups_1@dexidp.com", "groups_2@dexidp.com"},
		},
		"user1_transitive_lookup": {
			userKey:                        "user_1@dexidp.com",
			fetchTransitiveGroupMembership: true,
			shouldErr:                      false,
			expectedGroups:                 []string{"groups_0@dexidp.com", "groups_1@dexidp.com", "groups_2@dexidp.com"},
		},
		"user2_non_transitive_lookup": {
			userKey:                        "user_2@dexidp.com",
			fetchTransitiveGroupMembership: false,
			shouldErr:                      false,
			expectedGroups:                 []string{"groups_1@dexidp.com"},
		},
		"user2_transitive_lookup": {
			userKey:                        "user_2@dexidp.com",
			fetchTransitiveGroupMembership: true,
			shouldErr:                      false,
			expectedGroups:                 []string{"groups_0@dexidp.com", "groups_1@dexidp.com"},
		},
	} {
		testCase := testCase
		callCounter = map[string]int{}
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			lookup := make(map[string]struct{})

			groups, err := conn.getGroups(testCase.userKey, testCase.fetchTransitiveGroupMembership, lookup)
			if testCase.shouldErr {
				assert.NotNil(err)
			} else {
				assert.Nil(err)
			}
			assert.ElementsMatch(testCase.expectedGroups, groups)
			t.Logf("[%s] Amount of API calls per userKey: %+v\n", t.Name(), callCounter)
		})
	}
}

func TestGetGroupsWithWorkloadIdentityFederation(t *testing.T) {
	ts := testSetup()
	defer ts.Close()

	workloadIdentityFederationFilePath, err := tempWorkloadIdentityFederation()
	assert.Nil(t, err)

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", workloadIdentityFederationFilePath)
	conn, err := newConnector(&Config{
		ClientID:                    "testClient",
		ClientSecret:                "testSecret",
		RedirectURI:                 ts.URL + "/callback",
		Scopes:                      []string{"openid", "groups"},
		DomainToAdminEmail:          map[string]string{"*": "admin@dexidp.com"},
		ServiceAccountToImpersonate: "sa-dwd@my-google-project.iam.gserviceaccount.com",
	})
	assert.Nil(t, err)

	conn.adminSrv[wildcardDomainToAdminEmail], err = admin.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(ts.URL))
	assert.Nil(t, err)
	type testCase struct {
		userKey                        string
		fetchTransitiveGroupMembership bool
		shouldErr                      bool
		expectedGroups                 []string
	}

	for name, testCase := range map[string]testCase{
		"user1_non_transitive_lookup": {
			userKey:                        "user_1@dexidp.com",
			fetchTransitiveGroupMembership: false,
			shouldErr:                      false,
			expectedGroups:                 []string{"groups_1@dexidp.com", "groups_2@dexidp.com"},
		},
		"user1_transitive_lookup": {
			userKey:                        "user_1@dexidp.com",
			fetchTransitiveGroupMembership: true,
			shouldErr:                      false,
			expectedGroups:                 []string{"groups_0@dexidp.com", "groups_1@dexidp.com", "groups_2@dexidp.com"},
		},
		"user2_non_transitive_lookup": {
			userKey:                        "user_2@dexidp.com",
			fetchTransitiveGroupMembership: false,
			shouldErr:                      false,
			expectedGroups:                 []string{"groups_1@dexidp.com"},
		},
		"user2_transitive_lookup": {
			userKey:                        "user_2@dexidp.com",
			fetchTransitiveGroupMembership: true,
			shouldErr:                      false,
			expectedGroups:                 []string{"groups_0@dexidp.com", "groups_1@dexidp.com"},
		},
	} {
		testCase := testCase
		callCounter = map[string]int{}
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			lookup := make(map[string]struct{})

			groups, err := conn.getGroups(testCase.userKey, testCase.fetchTransitiveGroupMembership, lookup)
			if testCase.shouldErr {
				assert.NotNil(err)
			} else {
				assert.Nil(err)
			}
			assert.ElementsMatch(testCase.expectedGroups, groups)
			t.Logf("[%s] Amount of API calls per userKey: %+v\n", t.Name(), callCounter)
		})
	}
}

func TestDomainToAdminEmailConfig(t *testing.T) {
	ts := testSetup()
	defer ts.Close()

	serviceAccountFilePath, err := tempServiceAccountKey()
	assert.Nil(t, err)

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", serviceAccountFilePath)
	conn, err := newConnector(&Config{
		ClientID:           "testClient",
		ClientSecret:       "testSecret",
		RedirectURI:        ts.URL + "/callback",
		Scopes:             []string{"openid", "groups"},
		DomainToAdminEmail: map[string]string{"dexidp.com": "admin@dexidp.com"},
	})
	assert.Nil(t, err)

	conn.adminSrv["dexidp.com"], err = admin.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(ts.URL))
	assert.Nil(t, err)
	type testCase struct {
		userKey     string
		expectedErr string
	}

	for name, testCase := range map[string]testCase{
		"correct_user_request": {
			userKey:     "user_1@dexidp.com",
			expectedErr: "",
		},
		"wrong_user_request": {
			userKey:     "user_1@foo.bar",
			expectedErr: "unable to find super admin email",
		},
		"wrong_connector_response": {
			userKey:     "user_1_foo.bar",
			expectedErr: "unable to find super admin email",
		},
	} {
		testCase := testCase
		callCounter = map[string]int{}
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			lookup := make(map[string]struct{})

			_, err := conn.getGroups(testCase.userKey, true, lookup)
			if testCase.expectedErr != "" {
				assert.ErrorContains(err, testCase.expectedErr)
			} else {
				assert.Nil(err)
			}
			t.Logf("[%s] Amount of API calls per userKey: %+v\n", t.Name(), callCounter)
		})
	}
}

func TestPromptTypeConfig(t *testing.T) {
	promptTypeLogin := "login"
	cases := []struct {
		name                    string
		promptType              *string
		expectedPromptTypeValue string
	}{
		{
			name:                    "prompt type is nil",
			promptType:              nil,
			expectedPromptTypeValue: "consent",
		},
		{
			name:                    "prompt type is empty",
			promptType:              new(string),
			expectedPromptTypeValue: "",
		},
		{
			name:                    "prompt type is set",
			promptType:              &promptTypeLogin,
			expectedPromptTypeValue: "login",
		},
	}

	ts := testSetup()
	defer ts.Close()

	serviceAccountFilePath, err := tempServiceAccountKey()
	assert.Nil(t, err)

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", serviceAccountFilePath)

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			conn, err := newConnector(&Config{
				ClientID:           "testClient",
				ClientSecret:       "testSecret",
				RedirectURI:        ts.URL + "/callback",
				Scopes:             []string{"openid", "groups", "offline_access"},
				DomainToAdminEmail: map[string]string{"dexidp.com": "admin@dexidp.com"},
				PromptType:         test.promptType,
			})

			assert.Nil(t, err)
			assert.Equal(t, test.expectedPromptTypeValue, conn.promptType)

			loginURL, err := conn.LoginURL(connector.Scopes{OfflineAccess: true}, ts.URL+"/callback", "state")
			assert.Nil(t, err)

			urlp, err := url.Parse(loginURL)
			assert.Nil(t, err)

			assert.Equal(t, test.expectedPromptTypeValue, urlp.Query().Get("prompt"))
		})
	}
}
