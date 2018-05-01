package oidc

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2"
)

func TestOpen(t *testing.T) {
	testServer := testSetup(t)
	defer testServer.Close()

	conn := newConnector(t, testServer.URL)

	sort.Strings(conn.oauth2Config.Scopes)

	expectEqual(t, conn.oauth2Config.ClientID, "testClient")
	expectEqual(t, conn.oauth2Config.ClientSecret, "testSecret")
	expectEqual(t, conn.oauth2Config.RedirectURL, testServer.URL+"/callback")
	expectEqual(t, conn.oauth2Config.Endpoint.TokenURL, testServer.URL+"/token")
	expectEqual(t, conn.oauth2Config.Endpoint.AuthURL, testServer.URL+"/authorize")
	expectEqual(t, len(conn.oauth2Config.Scopes), 2)
	expectEqual(t, conn.oauth2Config.Scopes[0], "groups")
	expectEqual(t, conn.oauth2Config.Scopes[1], "openid")
}

func TestLoginURL(t *testing.T) {
	testServer := testSetup(t)
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

func TestHandleCallBack(t *testing.T) {
	testServer := testSetup(t)
	defer testServer.Close()

	conn := newConnector(t, testServer.URL)

	req := newRequestWithAuthCode(t, testServer.URL, "some-code")

	identity, err := conn.HandleCallback(connector.Scopes{Groups: true}, req)
	if err != nil {
		t.Fatal("handle callback failed", err)
	}

	expectEqual(t, len(identity.Groups), 1)
	expectEqual(t, identity.Groups[0], "test-group")
	expectEqual(t, identity.Name, "test-name")
	expectEqual(t, identity.Username, "test-username")
	expectEqual(t, identity.Email, "test-email")
	expectEqual(t, identity.EmailVerified, true)
}

func testSetup(t *testing.T) *httptest.Server {

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

	mux.HandleFunc("/keys", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(&map[string]interface{}{
			"keys": []map[string]interface{}{{
				"alg": jwk.Algorithm,
				"kty": jwk.Algorithm,
				"kid": jwk.KeyID,
				"n":   n(&key.PublicKey),
				"e":   e(&key.PublicKey),
			}},
		})
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		url := fmt.Sprintf("http://%s", r.Host)

		token, err := newToken(&jwk, map[string]interface{}{
			"iss":            url,
			"aud":            "testClient",
			"exp":            time.Now().Add(time.Hour).Unix(),
			"groups_key":     []string{"test-group"},
			"name":           "test-name",
			"username":       "test-username",
			"email":          "test-email",
			"email_verified": true,
		})
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

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		url := fmt.Sprintf("http://%s", r.Host)

		json.NewEncoder(w).Encode(&map[string]string{
			"issuer":                 url,
			"token_endpoint":         url + "/token",
			"authorization_endpoint": url + "/authorize",
			"userinfo_endpoint":      url + "/userinfo",
			"jwks_uri":               url + "/keys",
		})
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

func newConnector(t *testing.T, serverURL string) *oidcConnector {
	testConfig := Config{
		Issuer:       serverURL,
		ClientID:     "testClient",
		ClientSecret: "testSecret",
		Scopes:       []string{"groups"},
		RedirectURI:  serverURL + "/callback",
		GroupsKey:    "groups_key",
	}

	log := logrus.New()

	conn, err := testConfig.Open("id", log)
	if err != nil {
		t.Fatal(err)
	}

	oidcConn, ok := conn.(*oidcConnector)
	if !ok {
		t.Fatal(errors.New("failed to convert to oidcConnector"))
	}

	return oidcConn
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

func n(pub *rsa.PublicKey) string {
	return encode(pub.N.Bytes())
}

func e(pub *rsa.PublicKey) string {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, uint64(pub.E))
	return encode(bytes.TrimLeft(data, "\x00"))
}

func encode(payload []byte) string {
	result := base64.URLEncoding.EncodeToString(payload)
	return strings.TrimRight(result, "=")
}
