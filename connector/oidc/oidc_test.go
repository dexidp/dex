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
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2"

	"github.com/dexidp/dex/connector"
)

func TestKnownBrokenAuthHeaderProvider(t *testing.T) {
	tests := []struct {
		issuerURL string
		expect    bool
	}{
		{"https://dev.oktapreview.com", true},
		{"https://dev.okta.com", true},
		{"https://okta.com", true},
		{"https://dev.oktaaccounts.com", false},
		{"https://accounts.google.com", false},
	}

	for _, tc := range tests {
		got := knownBrokenAuthHeaderProvider(tc.issuerURL)
		if got != tc.expect {
			t.Errorf("knownBrokenAuthHeaderProvider(%q), want=%t, got=%t", tc.issuerURL, tc.expect, got)
		}
	}
}

func TestHandleCallback(t *testing.T) {
	t.Helper()

	tests := []struct {
		name                      string
		userIDKey                 string
		userNameKey               string
		preferredUsernameKey      string
		emailKey                  string
		groupsKey                 string
		insecureSkipEmailVerified bool
		scopes                    []string
		expectUserID              string
		expectUserName            string
		expectGroups              []string
		expectPreferredUsername   string
		expectedEmailField        string
		token                     map[string]interface{}
	}{
		{
			name:               "simpleCase",
			userIDKey:          "", // not configured
			userNameKey:        "", // not configured
			expectUserID:       "subvalue",
			expectUserName:     "namevalue",
			expectGroups:       []string{"group1", "group2"},
			expectedEmailField: "emailvalue",
			token: map[string]interface{}{
				"sub":            "subvalue",
				"name":           "namevalue",
				"groups":         []string{"group1", "group2"},
				"email":          "emailvalue",
				"email_verified": true,
			},
		},
		{
			name:               "customEmailClaim",
			userIDKey:          "", // not configured
			userNameKey:        "", // not configured
			emailKey:           "mail",
			expectUserID:       "subvalue",
			expectUserName:     "namevalue",
			expectedEmailField: "emailvalue",
			token: map[string]interface{}{
				"sub":            "subvalue",
				"name":           "namevalue",
				"mail":           "emailvalue",
				"email_verified": true,
			},
		},
		{
			name:                      "email_verified not in claims, configured to be skipped",
			insecureSkipEmailVerified: true,
			expectUserID:              "subvalue",
			expectUserName:            "namevalue",
			expectedEmailField:        "emailvalue",
			token: map[string]interface{}{
				"sub":   "subvalue",
				"name":  "namevalue",
				"email": "emailvalue",
			},
		},
		{
			name:               "withUserIDKey",
			userIDKey:          "name",
			expectUserID:       "namevalue",
			expectUserName:     "namevalue",
			expectedEmailField: "emailvalue",
			token: map[string]interface{}{
				"sub":            "subvalue",
				"name":           "namevalue",
				"email":          "emailvalue",
				"email_verified": true,
			},
		},
		{
			name:               "withUserNameKey",
			userNameKey:        "user_name",
			expectUserID:       "subvalue",
			expectUserName:     "username",
			expectedEmailField: "emailvalue",
			token: map[string]interface{}{
				"sub":            "subvalue",
				"user_name":      "username",
				"email":          "emailvalue",
				"email_verified": true,
			},
		},
		{
			name:                    "withPreferredUsernameKey",
			preferredUsernameKey:    "username_key",
			expectUserID:            "subvalue",
			expectUserName:          "namevalue",
			expectPreferredUsername: "username_value",
			expectedEmailField:      "emailvalue",
			token: map[string]interface{}{
				"sub":            "subvalue",
				"name":           "namevalue",
				"username_key":   "username_value",
				"email":          "emailvalue",
				"email_verified": true,
			},
		},
		{
			name:                    "withoutPreferredUsernameKeyAndBackendReturns",
			expectUserID:            "subvalue",
			expectUserName:          "namevalue",
			expectPreferredUsername: "preferredusernamevalue",
			expectedEmailField:      "emailvalue",
			token: map[string]interface{}{
				"sub":                "subvalue",
				"name":               "namevalue",
				"preferred_username": "preferredusernamevalue",
				"email":              "emailvalue",
				"email_verified":     true,
			},
		},
		{
			name:                    "withoutPreferredUsernameKeyAndBackendNotReturn",
			expectUserID:            "subvalue",
			expectUserName:          "namevalue",
			expectPreferredUsername: "",
			expectedEmailField:      "emailvalue",
			token: map[string]interface{}{
				"sub":            "subvalue",
				"name":           "namevalue",
				"email":          "emailvalue",
				"email_verified": true,
			},
		},
		{
			name:                      "emptyEmailScope",
			expectUserID:              "subvalue",
			expectUserName:            "namevalue",
			expectedEmailField:        "",
			scopes:                    []string{"groups"},
			insecureSkipEmailVerified: true,
			token: map[string]interface{}{
				"sub":       "subvalue",
				"name":      "namevalue",
				"user_name": "username",
			},
		},
		{
			name:                      "emptyEmailScopeButEmailProvided",
			expectUserID:              "subvalue",
			expectUserName:            "namevalue",
			expectedEmailField:        "emailvalue",
			scopes:                    []string{"groups"},
			insecureSkipEmailVerified: true,
			token: map[string]interface{}{
				"sub":       "subvalue",
				"name":      "namevalue",
				"user_name": "username",
				"email":     "emailvalue",
			},
		},
		{
			name:                      "customGroupsKey",
			groupsKey:                 "cognito:groups",
			expectUserID:              "subvalue",
			expectUserName:            "namevalue",
			expectedEmailField:        "emailvalue",
			expectGroups:              []string{"group3", "group4"},
			scopes:                    []string{"groups"},
			insecureSkipEmailVerified: true,
			token: map[string]interface{}{
				"sub":            "subvalue",
				"name":           "namevalue",
				"user_name":      "username",
				"email":          "emailvalue",
				"cognito:groups": []string{"group3", "group4"},
			},
		},
		{
			name:                      "customGroupsKeyButGroupsProvided",
			groupsKey:                 "cognito:groups",
			expectUserID:              "subvalue",
			expectUserName:            "namevalue",
			expectedEmailField:        "emailvalue",
			expectGroups:              []string{"group1", "group2"},
			scopes:                    []string{"groups"},
			insecureSkipEmailVerified: true,
			token: map[string]interface{}{
				"sub":            "subvalue",
				"name":           "namevalue",
				"user_name":      "username",
				"email":          "emailvalue",
				"groups":         []string{"group1", "group2"},
				"cognito:groups": []string{"group3", "group4"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testServer, err := setupServer(tc.token)
			if err != nil {
				t.Fatal("failed to setup test server", err)
			}
			defer testServer.Close()

			var scopes []string
			if len(tc.scopes) > 0 {
				scopes = tc.scopes
			} else {
				scopes = []string{"email", "groups"}
			}
			serverURL := testServer.URL
			basicAuth := true
			config := Config{
				Issuer:                    serverURL,
				ClientID:                  "clientID",
				ClientSecret:              "clientSecret",
				Scopes:                    scopes,
				RedirectURI:               fmt.Sprintf("%s/callback", serverURL),
				UserIDKey:                 tc.userIDKey,
				UserNameKey:               tc.userNameKey,
				InsecureSkipEmailVerified: tc.insecureSkipEmailVerified,
				InsecureEnableGroups:      true,
				BasicAuthUnsupported:      &basicAuth,
			}
			config.ClaimMapping.PreferredUsernameKey = tc.preferredUsernameKey
			config.ClaimMapping.EmailKey = tc.emailKey
			config.ClaimMapping.GroupsKey = tc.groupsKey

			conn, err := newConnector(config)
			if err != nil {
				t.Fatal("failed to create new connector", err)
			}

			req, err := newRequestWithAuthCode(testServer.URL, "someCode")
			if err != nil {
				t.Fatal("failed to create request", err)
			}

			identity, err := conn.HandleCallback(connector.Scopes{Groups: true}, req)
			if err != nil {
				t.Fatal("handle callback failed", err)
			}

			expectEquals(t, identity.UserID, tc.expectUserID)
			expectEquals(t, identity.Username, tc.expectUserName)
			expectEquals(t, identity.PreferredUsername, tc.expectPreferredUsername)
			expectEquals(t, identity.Email, tc.expectedEmailField)
			expectEquals(t, identity.EmailVerified, true)
			expectEquals(t, identity.Groups, tc.expectGroups)
		})
	}
}

func setupServer(tok map[string]interface{}) (*httptest.Server, error) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, fmt.Errorf("failed to generate rsa key: %v", err)
	}

	jwk := jose.JSONWebKey{
		Key:       key,
		KeyID:     "keyId",
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
		tok["iss"] = url
		tok["exp"] = time.Now().Add(time.Hour).Unix()
		tok["aud"] = "clientID"
		token, err := newToken(&jwk, tok)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
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
			"token_endpoint":         fmt.Sprintf("%s/token", url),
			"authorization_endpoint": fmt.Sprintf("%s/authorize", url),
			"userinfo_endpoint":      fmt.Sprintf("%s/userinfo", url),
			"jwks_uri":               fmt.Sprintf("%s/keys", url),
		})
	})

	return httptest.NewServer(mux), nil
}

func newToken(key *jose.JSONWebKey, claims map[string]interface{}) (string, error) {
	signingKey := jose.SigningKey{
		Key:       key,
		Algorithm: jose.RS256,
	}

	signer, err := jose.NewSigner(signingKey, &jose.SignerOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create new signer: %v", err)
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %v", err)
	}

	signature, err := signer.Sign(payload)
	if err != nil {
		return "", fmt.Errorf("failed to sign: %v", err)
	}
	return signature.CompactSerialize()
}

func newConnector(config Config) (*oidcConnector, error) {
	logger := logrus.New()
	conn, err := config.Open("id", logger)
	if err != nil {
		return nil, fmt.Errorf("unable to open: %v", err)
	}

	oidcConn, ok := conn.(*oidcConnector)
	if !ok {
		return nil, errors.New("failed to convert to oidcConnector")
	}

	return oidcConn, nil
}

func newRequestWithAuthCode(serverURL string, code string) (*http.Request, error) {
	req, err := http.NewRequest("GET", serverURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	values := req.URL.Query()
	values.Add("code", code)
	req.URL.RawQuery = values.Encode()

	return req, nil
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

func expectEquals(t *testing.T, a interface{}, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Expected %+v to equal %+v", a, b)
	}
}
