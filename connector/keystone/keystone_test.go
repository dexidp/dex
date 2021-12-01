package keystone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/dexidp/dex/connector"
)

const (
	invalidPass = "WRONG_PASS"

	testUser   = "test_user"
	testPass   = "test_pass"
	testEmail  = "test@example.com"
	testGroup  = "test_group"
	testDomain = "default"
)

var (
	keystoneURL      = ""
	keystoneAdminURL = ""
	adminUser        = ""
	adminPass        = ""
	authTokenURL     = ""
	usersURL         = ""
	groupsURL        = ""
)

type groupResponse struct {
	Group struct {
		ID string `json:"id"`
	} `json:"group"`
}

func getAdminToken(t *testing.T, adminName, adminPass string) (token, id string) {
	t.Helper()
	client := &http.Client{}

	jsonData := loginRequestData{
		auth: auth{
			Identity: identity{
				Methods: []string{"password"},
				Password: password{
					User: user{
						Name:     adminName,
						Domain:   domain{ID: testDomain},
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
	resp, err := client.Do(req)
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

func createUser(t *testing.T, token, userName, userEmail, userPass string) string {
	t.Helper()
	client := &http.Client{}

	createUserData := map[string]interface{}{
		"user": map[string]interface{}{
			"name":     userName,
			"email":    userEmail,
			"enabled":  true,
			"password": userPass,
			"roles":    []string{"admin"},
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
	resp, err := client.Do(req)
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
	client := &http.Client{}

	deleteURI := uri + id
	req, err := http.NewRequest("DELETE", deleteURI, nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	defer resp.Body.Close()
}

func createGroup(t *testing.T, token, description, name string) string {
	t.Helper()
	client := &http.Client{}

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
	resp, err := client.Do(req)
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
	client := &http.Client{}
	req, err := http.NewRequest("PUT", uri, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	defer resp.Body.Close()

	return nil
}

func TestIncorrectCredentialsLogin(t *testing.T) {
	setupVariables(t)
	c := conn{
		Host: keystoneURL, Domain: testDomain,
		AdminUsername: adminUser, AdminPassword: adminPass,
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
		username string
		domain   string
		email    string
		password string
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
				username: testUser,
				domain:   testDomain,
				email:    testEmail,
				password: testPass,
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
				username: testUser,
				domain:   testDomain,
				email:    "",
				password: testPass,
			},
			expected: expect{
				username:      testUser,
				email:         "",
				verifiedEmail: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := createUser(t, token, tt.input.username, tt.input.email, tt.input.password)
			defer deleteResource(t, token, userID, usersURL)

			c := conn{
				Host: keystoneURL, Domain: tt.input.domain,
				AdminUsername: adminUser, AdminPassword: adminPass,
			}
			s := connector.Scopes{OfflineAccess: true, Groups: true}
			identity, validPW, err := c.Login(context.Background(), s, tt.input.username, tt.input.password)
			if err != nil {
				t.Fatal(err.Error())
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
		Host: keystoneURL, Domain: testDomain,
		AdminUsername: adminUser, AdminPassword: adminPass,
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

	expectEquals(t, 1, len(identityRefresh.Groups))
	expectEquals(t, testGroup, identityRefresh.Groups[0])
}

func TestUseRefreshTokenUserDeleted(t *testing.T) {
	setupVariables(t)
	token, _ := getAdminToken(t, adminUser, adminPass)
	userID := createUser(t, token, testUser, testEmail, testPass)

	c := conn{
		Host: keystoneURL, Domain: testDomain,
		AdminUsername: adminUser, AdminPassword: adminPass,
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
	userID := createUser(t, token, testUser, testEmail, testPass)
	defer deleteResource(t, token, userID, usersURL)

	c := conn{
		Host: keystoneURL, Domain: testDomain,
		AdminUsername: adminUser, AdminPassword: adminPass,
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

	expectEquals(t, 0, len(identityRefresh.Groups))

	groupID := createGroup(t, token, "Test group", testGroup)
	addUserToGroup(t, token, groupID, userID)
	defer deleteResource(t, token, groupID, groupsURL)

	identityRefresh, err = c.Refresh(context.Background(), s, identityLogin)
	if err != nil {
		t.Fatal(err.Error())
	}

	expectEquals(t, 1, len(identityRefresh.Groups))
}

func TestNoGroupsInScope(t *testing.T) {
	setupVariables(t)
	token, _ := getAdminToken(t, adminUser, adminPass)
	userID := createUser(t, token, testUser, testEmail, testPass)
	defer deleteResource(t, token, userID, usersURL)

	c := conn{
		Host: keystoneURL, Domain: testDomain,
		AdminUsername: adminUser, AdminPassword: adminPass,
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
		t.Skip(fmt.Sprintf("variable %q not set, skipping keystone connector tests\n", keystoneURLEnv))
		return
	}
	keystoneAdminURL = os.Getenv(keystoneAdminURLEnv)
	if keystoneAdminURL == "" {
		t.Skip(fmt.Sprintf("variable %q not set, skipping keystone connector tests\n", keystoneAdminURLEnv))
		return
	}
	adminUser = os.Getenv(keystoneAdminUserEnv)
	if adminUser == "" {
		t.Skip(fmt.Sprintf("variable %q not set, skipping keystone connector tests\n", keystoneAdminUserEnv))
		return
	}
	adminPass = os.Getenv(keystoneAdminPassEnv)
	if adminPass == "" {
		t.Skip(fmt.Sprintf("variable %q not set, skipping keystone connector tests\n", keystoneAdminPassEnv))
		return
	}
	authTokenURL = keystoneURL + "/v3/auth/tokens/"
	usersURL = keystoneAdminURL + "/v3/users/"
	groupsURL = keystoneAdminURL + "/v3/groups/"
}

func expectEquals(t *testing.T, a interface{}, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Expected %v to be equal %v", a, b)
	}
}
