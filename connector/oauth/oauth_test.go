package oauth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"testing"

	"github.com/go-jose/go-jose/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/dexidp/dex/connector"
)

func TestOpen(t *testing.T) {
	tokenClaims := map[string]interface{}{}
	userInfoClaims := map[string]interface{}{}

	testServer := testSetup(t, tokenClaims, userInfoClaims)
	defer testServer.Close()

	conn := newConnector(t, testServer.URL)

	sort.Strings(conn.scopes)

	assert.Equal(t, conn.clientID, "testClient")
	assert.Equal(t, conn.clientSecret, "testSecret")
	assert.Equal(t, conn.redirectURI, testServer.URL+"/callback")
	assert.Equal(t, conn.tokenURL, testServer.URL+"/token")
	assert.Equal(t, conn.authorizationURL, testServer.URL+"/authorize")
	assert.Equal(t, conn.userInfoURL, testServer.URL+"/userinfo")
	assert.Equal(t, len(conn.scopes), 2)
	assert.Equal(t, conn.scopes[0], "groups")
	assert.Equal(t, conn.scopes[1], "openid")
}

func TestLoginURL(t *testing.T) {
	tokenClaims := map[string]interface{}{}
	userInfoClaims := map[string]interface{}{}

	testServer := testSetup(t, tokenClaims, userInfoClaims)
	defer testServer.Close()

	conn := newConnector(t, testServer.URL)

	loginURL, err := conn.LoginURL(connector.Scopes{}, conn.redirectURI, "some-state")
	assert.Equal(t, err, nil)

	expectedURL, err := url.Parse(testServer.URL + "/authorize")
	assert.Equal(t, err, nil)

	values := url.Values{}
	values.Add("client_id", "testClient")
	values.Add("redirect_uri", conn.redirectURI)
	values.Add("response_type", "code")
	values.Add("scope", "openid groups")
	values.Add("state", "some-state")
	expectedURL.RawQuery = values.Encode()

	assert.Equal(t, loginURL, expectedURL.String())
}

func TestHandleCallBackForGroupsInUserInfo(t *testing.T) {
	tokenClaims := map[string]interface{}{}

	userInfoClaims := map[string]interface{}{
		"name":               "test-name",
		"user_id_key":        "test-user-id",
		"user_name_key":      "test-username",
		"preferred_username": "test-preferred-username",
		"mail":               "mod_mail",
		"has_verified_email": false,
		"groups_key":         []string{"admin-group", "user-group"},
	}

	testServer := testSetup(t, tokenClaims, userInfoClaims)
	defer testServer.Close()

	conn := newConnector(t, testServer.URL)
	req := newRequestWithAuthCode(t, testServer.URL, "TestHandleCallBackForGroupsInUserInfo")

	identity, err := conn.HandleCallback(connector.Scopes{Groups: true}, req)
	assert.Equal(t, err, nil)

	sort.Strings(identity.Groups)
	assert.Equal(t, len(identity.Groups), 2)
	assert.Equal(t, identity.Groups[0], "admin-group")
	assert.Equal(t, identity.Groups[1], "user-group")
	assert.Equal(t, identity.UserID, "test-user-id")
	assert.Equal(t, identity.Username, "test-username")
	assert.Equal(t, identity.PreferredUsername, "test-preferred-username")
	assert.Equal(t, identity.Email, "mod_mail")
	assert.Equal(t, identity.EmailVerified, false)
}

func TestHandleCallBackForGroupMapsInUserInfo(t *testing.T) {
	tokenClaims := map[string]interface{}{}

	userInfoClaims := map[string]interface{}{
		"name":               "test-name",
		"user_id_key":        "test-user-id",
		"user_name_key":      "test-username",
		"preferred_username": "test-preferred-username",
		"mail":               "mod_mail",
		"has_verified_email": false,
		"groups_key": []interface{}{
			map[string]string{"name": "admin-group", "id": "111"},
			map[string]string{"name": "user-group", "id": "222"},
		},
	}

	testServer := testSetup(t, tokenClaims, userInfoClaims)
	defer testServer.Close()

	conn := newConnector(t, testServer.URL)
	req := newRequestWithAuthCode(t, testServer.URL, "TestHandleCallBackForGroupMapsInUserInfo")

	identity, err := conn.HandleCallback(connector.Scopes{Groups: true}, req)
	assert.Equal(t, err, nil)

	sort.Strings(identity.Groups)
	assert.Equal(t, len(identity.Groups), 2)
	assert.Equal(t, identity.Groups[0], "admin-group")
	assert.Equal(t, identity.Groups[1], "user-group")
	assert.Equal(t, identity.UserID, "test-user-id")
	assert.Equal(t, identity.Username, "test-username")
	assert.Equal(t, identity.PreferredUsername, "test-preferred-username")
	assert.Equal(t, identity.Email, "mod_mail")
	assert.Equal(t, identity.EmailVerified, false)
}

func TestHandleCallBackForGroupsInToken(t *testing.T) {
	tokenClaims := map[string]interface{}{
		"groups_key": []string{"test-group"},
	}

	userInfoClaims := map[string]interface{}{
		"name":               "test-name",
		"user_id_key":        "test-user-id",
		"user_name_key":      "test-username",
		"preferred_username": "test-preferred-username",
		"email":              "test-email",
		"email_verified":     true,
	}

	testServer := testSetup(t, tokenClaims, userInfoClaims)
	defer testServer.Close()

	conn := newConnector(t, testServer.URL)
	req := newRequestWithAuthCode(t, testServer.URL, "TestHandleCallBackForGroupsInToken")

	identity, err := conn.HandleCallback(connector.Scopes{Groups: true}, req)
	assert.Equal(t, err, nil)

	assert.Equal(t, len(identity.Groups), 1)
	assert.Equal(t, identity.Groups[0], "test-group")
	assert.Equal(t, identity.PreferredUsername, "test-preferred-username")
	assert.Equal(t, identity.UserID, "test-user-id")
	assert.Equal(t, identity.Username, "test-username")
	assert.Equal(t, identity.Email, "")
	assert.Equal(t, identity.EmailVerified, false)
}

func TestHandleCallbackForNumericUserID(t *testing.T) {
	tokenClaims := map[string]interface{}{}

	userInfoClaims := map[string]interface{}{
		"name":               "test-name",
		"user_id_key":        1000,
		"user_name_key":      "test-username",
		"preferred_username": "test-preferred-username",
		"mail":               "mod_mail",
		"has_verified_email": false,
	}

	testServer := testSetup(t, tokenClaims, userInfoClaims)
	defer testServer.Close()

	conn := newConnector(t, testServer.URL)
	req := newRequestWithAuthCode(t, testServer.URL, "TestHandleCallbackForNumericUserID")

	identity, err := conn.HandleCallback(connector.Scopes{Groups: true}, req)
	assert.Equal(t, err, nil)

	assert.Equal(t, identity.UserID, "1000")
	assert.Equal(t, identity.Username, "test-username")
	assert.Equal(t, identity.PreferredUsername, "test-preferred-username")
	assert.Equal(t, identity.Email, "mod_mail")
	assert.Equal(t, identity.EmailVerified, false)
}

func testSetup(t *testing.T, tokenClaims map[string]interface{}, userInfoClaims map[string]interface{}) *httptest.Server {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal("Failed to generate rsa key", err)
	}

	jwk := jose.JSONWebKey{
		Key:       key,
		KeyID:     "some-key",
		Algorithm: "RSA",
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		token, err := newToken(&jwk, tokenClaims)
		if err != nil {
			t.Fatal("unable to generate token", err)
		}

		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&map[string]string{
			"access_token": token,
			"id_token":     token,
			"token_type":   "Bearer",
		})
	})

	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userInfoClaims)
	})

	return httptest.NewServer(mux)
}

func newToken(key *jose.JSONWebKey, claims map[string]interface{}) (string, error) {
	signingKey := jose.SigningKey{Key: key, Algorithm: jose.RS256}

	signer, err := jose.NewSigner(signingKey, &jose.SignerOptions{})
	if err != nil {
		return "", fmt.Errorf("new signer: %v", err)
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshaling claims: %v", err)
	}

	signature, err := signer.Sign(payload)
	if err != nil {
		return "", fmt.Errorf("signing payload: %v", err)
	}

	return signature.CompactSerialize()
}

func newConnector(t *testing.T, serverURL string) *oauthConnector {
	testConfig := Config{
		ClientID:         "testClient",
		ClientSecret:     "testSecret",
		RedirectURI:      serverURL + "/callback",
		TokenURL:         serverURL + "/token",
		AuthorizationURL: serverURL + "/authorize",
		UserInfoURL:      serverURL + "/userinfo",
		Scopes:           []string{"openid", "groups"},
		UserIDKey:        "user_id_key",
	}

	testConfig.ClaimMapping.UserNameKey = "user_name_key"
	testConfig.ClaimMapping.GroupsKey = "groups_key"
	testConfig.ClaimMapping.EmailKey = "mail"
	testConfig.ClaimMapping.EmailVerifiedKey = "has_verified_email"

	log := logrus.New()

	conn, err := testConfig.Open("id", log)
	if err != nil {
		t.Fatal(err)
	}

	oauthConn, ok := conn.(*oauthConnector)
	if !ok {
		t.Fatal(errors.New("failed to convert to oauthConnector"))
	}

	return oauthConn
}

func newRequestWithAuthCode(t *testing.T, serverURL string, code string) *http.Request {
	req, err := http.NewRequest("GET", serverURL, nil)
	if err != nil {
		t.Fatal("failed to create request", err)
	}

	values := req.URL.Query()
	values.Add("code", code)
	req.URL.RawQuery = values.Encode()

	return req
}
