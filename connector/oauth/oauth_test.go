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
	"reflect"
	"sort"
	"testing"

	"github.com/dexidp/dex/connector"
	"github.com/sirupsen/logrus"
	jose "gopkg.in/square/go-jose.v2"
)

func TestOpen(t *testing.T) {
	tokenClaims := map[string]interface{}{}
	userInfoClaims := map[string]interface{}{}

	testServer := testSetup(t, tokenClaims, userInfoClaims)
	defer testServer.Close()

	conn := newConnector(t, testServer.URL)

	sort.Strings(conn.scopes)

	expectEqual(t, conn.clientID, "testClient")
	expectEqual(t, conn.clientSecret, "testSecret")
	expectEqual(t, conn.redirectURI, testServer.URL+"/callback")
	expectEqual(t, conn.tokenURL, testServer.URL+"/token")
	expectEqual(t, conn.authorizationURL, testServer.URL+"/authorize")
	expectEqual(t, conn.userInfoURL, testServer.URL+"/userinfo")
	expectEqual(t, len(conn.scopes), 2)
	expectEqual(t, conn.scopes[0], "groups")
	expectEqual(t, conn.scopes[1], "openid")
}

func TestLoginURL(t *testing.T) {
	tokenClaims := map[string]interface{}{}
	userInfoClaims := map[string]interface{}{}

	testServer := testSetup(t, tokenClaims, userInfoClaims)
	defer testServer.Close()

	conn := newConnector(t, testServer.URL)

	loginURL, err := conn.LoginURL(connector.Scopes{}, conn.redirectURI, "some-state")
	expectEqual(t, err, nil)

	expectedURL, err := url.Parse(testServer.URL + "/authorize")
	expectEqual(t, err, nil)

	values := url.Values{}
	values.Add("client_id", "testClient")
	values.Add("redirect_uri", conn.redirectURI)
	values.Add("response_type", "code")
	values.Add("scope", "openid groups")
	values.Add("state", "some-state")
	expectedURL.RawQuery = values.Encode()

	expectEqual(t, loginURL, expectedURL.String())
}

func TestHandleCallBackForGroupsInUserInfo(t *testing.T) {

	tokenClaims := map[string]interface{}{}

	userInfoClaims := map[string]interface{}{
		"name":           "test-name",
		"user_name":      "test-username",
		"user_id":        "test-user-id",
		"email":          "test-email",
		"email_verified": true,
		"groups_key":     []string{"admin-group", "user-group"},
	}

	testServer := testSetup(t, tokenClaims, userInfoClaims)
	defer testServer.Close()

	conn := newConnector(t, testServer.URL)
	req := newRequestWithAuthCode(t, testServer.URL, "some-code")

	identity, err := conn.HandleCallback(connector.Scopes{Groups: true}, req)
	expectEqual(t, err, nil)

	sort.Strings(identity.Groups)
	expectEqual(t, len(identity.Groups), 2)
	expectEqual(t, identity.Groups[0], "admin-group")
	expectEqual(t, identity.Groups[1], "user-group")
	expectEqual(t, identity.Name, "test-name")
	expectEqual(t, identity.Username, "test-username")
	expectEqual(t, identity.Email, "test-email")
	expectEqual(t, identity.EmailVerified, true)
}

func TestHandleCallBackForGroupsInToken(t *testing.T) {

	tokenClaims := map[string]interface{}{
		"groups_key": []string{"test-group"},
	}

	userInfoClaims := map[string]interface{}{
		"name":           "test-name",
		"user_name":      "test-username",
		"user_id":        "test-user-id",
		"email":          "test-email",
		"email_verified": true,
	}

	testServer := testSetup(t, tokenClaims, userInfoClaims)
	defer testServer.Close()

	conn := newConnector(t, testServer.URL)
	req := newRequestWithAuthCode(t, testServer.URL, "some-code")

	identity, err := conn.HandleCallback(connector.Scopes{Groups: true}, req)
	expectEqual(t, err, nil)

	expectEqual(t, len(identity.Groups), 1)
	expectEqual(t, identity.Groups[0], "test-group")
	expectEqual(t, identity.Name, "test-name")
	expectEqual(t, identity.Username, "test-username")
	expectEqual(t, identity.Email, "test-email")
	expectEqual(t, identity.EmailVerified, true)
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
		GroupsKey:        "groups_key",
	}

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

func expectEqual(t *testing.T, a interface{}, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("Expected %+v to equal %+v", a, b)
	}
}
