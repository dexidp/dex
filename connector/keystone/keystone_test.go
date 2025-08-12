package keystone

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/dexidp/dex/connector"
)

const (
	invalidPass = "WRONG_PASS"

	testUser          = "test_user"
	testPass          = "test_pass"
	testEmail         = "test@example.com"
	testGroup         = "test_group"
	testDomainAltName = "altdomain"
	testDomainID      = "default"
	testDomainName    = "Default"
)

var (
	keystoneURL      = ""
	keystoneAdminURL = ""
	adminUser        = ""
	adminPass        = ""
	authTokenURL     = ""
	usersURL         = ""
	groupsURL        = ""
	domainsURL       = ""
)

type userReq struct {
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Enabled  bool     `json:"enabled"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
	DomainID string   `json:"domain_id,omitempty"`
}

type domainResponse struct {
	Domain domainKeystone `json:"domain"`
}

type domainsResponse struct {
	Domains []domainKeystone `json:"domains"`
}

type groupResponse struct {
	Group struct {
		ID string `json:"id"`
	} `json:"group"`
}

func getAdminToken(t *testing.T, adminName, adminPass string) (token, id string) {
	t.Helper()
	jsonData := loginRequestData{
		auth: auth{
			Identity: identity{
				Methods: []string{"password"},
				Password: password{
					User: user{
						Name:     adminName,
						Domain:   domainKeystone{ID: testDomainID},
						Password: adminPass,
					},
				},
			},
		},
	}

	body, err := json.Marshal(jsonData)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", authTokenURL, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("keystone: failed to obtain admin token: %v\n", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	token = resp.Header.Get("X-Subject-Token")

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	tokenResp := new(tokenResponse)
	err = json.Unmarshal(data, &tokenResp)
	if err != nil {
		t.Fatal(err)
	}
	return token, tokenResp.Token.User.ID
}

func getOrCreateDomain(t *testing.T, token, domainName string) string {
	t.Helper()

	domainSearchURL := domainsURL + "?name=" + domainName
	reqGet, err := http.NewRequest("GET", domainSearchURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	reqGet.Header.Set("X-Auth-Token", token)
	reqGet.Header.Add("Content-Type", "application/json")
	respGet, err := http.DefaultClient.Do(reqGet)
	if err != nil {
		t.Fatal(err)
	}

	dataGet, err := io.ReadAll(respGet.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer respGet.Body.Close()

	domainsResp := new(domainsResponse)
	err = json.Unmarshal(dataGet, &domainsResp)
	if err != nil {
		t.Fatal(err)
	}

	if len(domainsResp.Domains) >= 1 {
		return domainsResp.Domains[0].ID
	}

	createDomainData := map[string]interface{}{
		"domain": map[string]interface{}{
			"name":    domainName,
			"enabled": true,
		},
	}

	body, err := json.Marshal(createDomainData)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", domainsURL, bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Auth-Token", token)
	req.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 201 {
		t.Fatalf("failed to create domain %s", domainName)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	domainResp := new(domainResponse)
	err = json.Unmarshal(data, &domainResp)
	if err != nil {
		t.Fatal(err)
	}

	return domainResp.Domain.ID
}

func createUser(t *testing.T, token, domainID, userName, userEmail, userPass string) string {
	t.Helper()

	createUserData := map[string]interface{}{
		"user": userReq{
			DomainID: domainID,
			Name:     userName,
			Email:    userEmail,
			Enabled:  true,
			Password: userPass,
			Roles:    []string{"admin"},
		},
	}

	body, err := json.Marshal(createUserData)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", usersURL, bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Auth-Token", token)
	req.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	userResp := new(userResponse)
	err = json.Unmarshal(data, &userResp)
	if err != nil {
		t.Fatal(err)
	}

	return userResp.User.ID
}

// delete group or user
func deleteResource(t *testing.T, token, id, uri string) {
	t.Helper()

	deleteURI := uri + id
	req, err := http.NewRequest("DELETE", deleteURI, nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	defer resp.Body.Close()
}

func createGroup(t *testing.T, token, description, name string) string {
	t.Helper()

	createGroupData := map[string]interface{}{
		"group": map[string]interface{}{
			"name":        name,
			"description": description,
		},
	}

	body, err := json.Marshal(createGroupData)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", groupsURL, bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Auth-Token", token)
	req.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	groupResp := new(groupResponse)
	err = json.Unmarshal(data, &groupResp)
	if err != nil {
		t.Fatal(err)
	}

	return groupResp.Group.ID
}

func addUserToGroup(t *testing.T, token, groupID, userID string) error {
	t.Helper()
	uri := groupsURL + groupID + "/users/" + userID
	req, err := http.NewRequest("PUT", uri, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	defer resp.Body.Close()

	return nil
}

func TestIncorrectCredentialsLogin(t *testing.T) {
	setupVariables(t)
	c := conn{
		client: http.DefaultClient,
		Host:   keystoneAdminURL, Domain: domainKeystone{ID: testDomainID},
		AdminUsername: adminUser, AdminPassword: adminPass,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	s := connector.Scopes{OfflineAccess: true, Groups: true}
	_, validPW, err := c.Login(context.Background(), s, adminUser, invalidPass)

	if validPW {
		t.Fatal("Incorrect password check")
	}

	if err == nil {
		t.Fatal("Error should be returned when invalid password is provided")
	}

	if !strings.Contains(err.Error(), "401") {
		t.Fatal("Unrecognized error, expecting 401")
	}
}

func TestValidUserLogin(t *testing.T) {
	setupVariables(t)
	token, _ := getAdminToken(t, adminUser, adminPass)

	type tUser struct {
		createDomain bool
		domain       domainKeystone
		username     string
		email        string
		password     string
	}

	type expect struct {
		username      string
		email         string
		verifiedEmail bool
	}

	tests := []struct {
		name     string
		input    tUser
		expected expect
	}{
		{
			name: "test with email address",
			input: tUser{
				createDomain: false,
				domain:       domainKeystone{ID: testDomainID},
				username:     testUser,
				email:        testEmail,
				password:     testPass,
			},
			expected: expect{
				username:      testUser,
				email:         testEmail,
				verifiedEmail: true,
			},
		},
		{
			name: "test without email address",
			input: tUser{
				createDomain: false,
				domain:       domainKeystone{ID: testDomainID},
				username:     testUser,
				email:        "",
				password:     testPass,
			},
			expected: expect{
				username:      testUser,
				email:         "",
				verifiedEmail: false,
			},
		},
		{
			name: "test with default domain Name",
			input: tUser{
				createDomain: false,
				domain:       domainKeystone{Name: testDomainName},
				username:     testUser,
				email:        testEmail,
				password:     testPass,
			},
			expected: expect{
				username:      testUser,
				email:         testEmail,
				verifiedEmail: true,
			},
		},
		{
			name: "test with custom domain Name",
			input: tUser{
				createDomain: true,
				domain:       domainKeystone{Name: testDomainAltName},
				username:     testUser,
				email:        testEmail,
				password:     testPass,
			},
			expected: expect{
				username:      testUser,
				email:         testEmail,
				verifiedEmail: true,
			},
		},
		{
			name: "test with custom domain ID",
			input: tUser{
				createDomain: true,
				domain:       domainKeystone{},
				username:     testUser,
				email:        testEmail,
				password:     testPass,
			},
			expected: expect{
				username:      testUser,
				email:         testEmail,
				verifiedEmail: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domainID := ""
			if tt.input.createDomain == true {
				domainID = getOrCreateDomain(t, token, testDomainAltName)
				t.Logf("getOrCreateDomain ID: %s\n", domainID)

				// if there was nothing set then use the dynamically generated domain ID
				if tt.input.domain.ID == "" && tt.input.domain.Name == "" {
					tt.input.domain.ID = domainID
				}
			}
			userID := createUser(t, token, domainID, tt.input.username, tt.input.email, tt.input.password)
			defer deleteResource(t, token, userID, usersURL)

			c := conn{
				client: http.DefaultClient,
				Host:   keystoneAdminURL, Domain: tt.input.domain,
				AdminUsername: adminUser, AdminPassword: adminPass,
				Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			}
			s := connector.Scopes{OfflineAccess: true, Groups: true}
			identity, validPW, err := c.Login(context.Background(), s, tt.input.username, tt.input.password)
			if err != nil {
				t.Fatalf("Login failed for user %s: %v", tt.input.username, err.Error())
			}
			t.Log(identity)
			if identity.Username != tt.expected.username {
				t.Fatalf("Invalid user. Got: %v. Wanted: %v", identity.Username, tt.expected.username)
			}
			if identity.UserID == "" {
				t.Fatalf("Didn't get any UserID back")
			}
			if identity.Email != tt.expected.email {
				t.Fatalf("Invalid email. Got: %v. Wanted: %v", identity.Email, tt.expected.email)
			}
			if identity.EmailVerified != tt.expected.verifiedEmail {
				t.Fatalf("Invalid verifiedEmail. Got: %v. Wanted: %v", identity.EmailVerified, tt.expected.verifiedEmail)
			}

			if !validPW {
				t.Fatal("Valid password was not accepted")
			}
		})
	}
}

func TestUseRefreshToken(t *testing.T) {
	setupVariables(t)
	token, adminID := getAdminToken(t, adminUser, adminPass)
	groupID := createGroup(t, token, "Test group description", testGroup)
	addUserToGroup(t, token, groupID, adminID)
	defer deleteResource(t, token, groupID, groupsURL)

	c := conn{
		client: http.DefaultClient,
		Host:   keystoneAdminURL, Domain: domainKeystone{ID: testDomainID},
		AdminUsername: adminUser, AdminPassword: adminPass,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	s := connector.Scopes{OfflineAccess: true, Groups: true}

	identityLogin, _, err := c.Login(context.Background(), s, adminUser, adminPass)
	if err != nil {
		t.Fatal(err.Error())
	}

	identityRefresh, err := c.Refresh(context.Background(), s, identityLogin)
	if err != nil {
		t.Fatal(err.Error())
	}

	// Custom connector may return additional role-derived groups; ensure our test group is present.
	found := false
	for _, g := range identityRefresh.Groups {
		if g == testGroup {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected group %q to be present in %v", testGroup, identityRefresh.Groups)
	}
}

func TestUseRefreshTokenUserDeleted(t *testing.T) {
	setupVariables(t)
	token, _ := getAdminToken(t, adminUser, adminPass)
	userID := createUser(t, token, "", testUser, testEmail, testPass)

	c := conn{
		client: http.DefaultClient,
		Host:   keystoneAdminURL, Domain: domainKeystone{ID: testDomainID},
		AdminUsername: adminUser, AdminPassword: adminPass,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	s := connector.Scopes{OfflineAccess: true, Groups: true}

	identityLogin, _, err := c.Login(context.Background(), s, testUser, testPass)
	if err != nil {
		t.Fatal(err.Error())
	}

	_, err = c.Refresh(context.Background(), s, identityLogin)
	if err != nil {
		t.Fatal(err.Error())
	}

	deleteResource(t, token, userID, usersURL)
	_, err = c.Refresh(context.Background(), s, identityLogin)

	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("unexpected error: %s", err.Error())
	}
}

func TestUseRefreshTokenGroupsChanged(t *testing.T) {
	setupVariables(t)
	token, _ := getAdminToken(t, adminUser, adminPass)
	userID := createUser(t, token, "", testUser, testEmail, testPass)
	defer deleteResource(t, token, userID, usersURL)

	c := conn{
		client: http.DefaultClient,
		Host:   keystoneAdminURL, Domain: domainKeystone{ID: testDomainID},
		AdminUsername: adminUser, AdminPassword: adminPass,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	s := connector.Scopes{OfflineAccess: true, Groups: true}

	identityLogin, _, err := c.Login(context.Background(), s, testUser, testPass)
	if err != nil {
		t.Fatal(err.Error())
	}

	identityRefresh, err := c.Refresh(context.Background(), s, identityLogin)
	if err != nil {
		t.Fatal(err.Error())
	}

	// With custom connector, initial groups may not be empty due to role-derived groups.
	// Assert that our test group is not present initially.
	for _, g := range identityRefresh.Groups {
		if g == testGroup {
			t.Fatalf("did not expect group %q to be present initially: %v", testGroup, identityRefresh.Groups)
		}
	}

	groupID := createGroup(t, token, "Test group", testGroup)
	addUserToGroup(t, token, groupID, userID)
	defer deleteResource(t, token, groupID, groupsURL)

	identityRefresh, err = c.Refresh(context.Background(), s, identityLogin)
	if err != nil {
		t.Fatal(err.Error())
	}

	// Ensure our test group is present after adding it.
	found := false
	for _, g := range identityRefresh.Groups {
		if g == testGroup {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected group %q to be present after change in %v", testGroup, identityRefresh.Groups)
	}
}

func TestNoGroupsInScope(t *testing.T) {
	setupVariables(t)
	token, _ := getAdminToken(t, adminUser, adminPass)
	userID := createUser(t, token, "", testUser, testEmail, testPass)
	defer deleteResource(t, token, userID, usersURL)

	c := conn{
		client: http.DefaultClient,
		Host:   keystoneAdminURL, Domain: domainKeystone{ID: testDomainID},
		AdminUsername: adminUser, AdminPassword: adminPass,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	s := connector.Scopes{OfflineAccess: true, Groups: false}

	groupID := createGroup(t, token, "Test group", testGroup)
	addUserToGroup(t, token, groupID, userID)
	defer deleteResource(t, token, groupID, groupsURL)

	identityLogin, _, err := c.Login(context.Background(), s, testUser, testPass)
	if err != nil {
		t.Fatal(err.Error())
	}
	expectEquals(t, 0, len(identityLogin.Groups))

	identityRefresh, err := c.Refresh(context.Background(), s, identityLogin)
	if err != nil {
		t.Fatal(err.Error())
	}
	expectEquals(t, 0, len(identityRefresh.Groups))
}

func setupVariables(t *testing.T) {
	keystoneURLEnv := "DEX_KEYSTONE_URL"
	keystoneAdminURLEnv := "DEX_KEYSTONE_ADMIN_URL"
	keystoneAdminUserEnv := "DEX_KEYSTONE_ADMIN_USER"
	keystoneAdminPassEnv := "DEX_KEYSTONE_ADMIN_PASS"
	keystoneURL = os.Getenv(keystoneURLEnv)
	if keystoneURL == "" {
		t.Skipf("variable %q not set, skipping keystone connector tests\n", keystoneURLEnv)
		return
	}
	keystoneAdminURL = os.Getenv(keystoneAdminURLEnv)
	if keystoneAdminURL == "" {
		t.Skipf("variable %q not set, skipping keystone connector tests\n", keystoneAdminURLEnv)
		return
	}
	adminUser = os.Getenv(keystoneAdminUserEnv)
	if adminUser == "" {
		t.Skipf("variable %q not set, skipping keystone connector tests\n", keystoneAdminUserEnv)
		return
	}
	adminPass = os.Getenv(keystoneAdminPassEnv)
	if adminPass == "" {
		t.Skipf("variable %q not set, skipping keystone connector tests\n", keystoneAdminPassEnv)
		return
	}
	authTokenURL = keystoneURL + "/v3/auth/tokens/"
	usersURL = keystoneAdminURL + "/v3/users/"
	groupsURL = keystoneAdminURL + "/v3/groups/"
	domainsURL = keystoneAdminURL + "/v3/domains/"
}

func expectEquals(t *testing.T, a interface{}, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Expected %v to be equal %v", a, b)
	}
}
