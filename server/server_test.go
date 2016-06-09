package server

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oauth2"
	"github.com/coreos/go-oidc/oidc"
	"github.com/kylelemons/godebug/pretty"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/db"
	"github.com/coreos/dex/refresh/refreshtest"
	"github.com/coreos/dex/session/manager"
	"github.com/coreos/dex/user"
)

var validRedirURL = url.URL{
	Scheme: "http",
	Host:   "client.example.com",
	Path:   "/callback",
}

type StaticKeyManager struct {
	key.PrivateKeyManager
	expiresAt time.Time
	signer    jose.Signer
	keys      []jose.JWK
}

func (m *StaticKeyManager) ExpiresAt() time.Time {
	return m.expiresAt
}

func (m *StaticKeyManager) Signer() (jose.Signer, error) {
	return m.signer, nil
}

func (m *StaticKeyManager) JWKs() ([]jose.JWK, error) {
	return m.keys, nil
}

type StaticSigner struct {
	sig []byte
	err error
}

func (ss *StaticSigner) ID() string {
	return "static"
}

func (ss *StaticSigner) Alg() string {
	return "static"
}

func (ss *StaticSigner) Verify(sig, data []byte) error {
	if !reflect.DeepEqual(ss.sig, sig) {
		return errors.New("signature mismatch")
	}

	return nil
}

func (ss *StaticSigner) Sign(data []byte) ([]byte, error) {
	return ss.sig, ss.err
}

func (ss *StaticSigner) JWK() jose.JWK {
	return jose.JWK{}
}

func staticGenerateCodeFunc(code string) manager.GenerateCodeFunc {
	return func() (string, error) {
		return code, nil
	}
}

func makeNewUserRepo() (user.UserRepo, error) {
	userRepo := db.NewUserRepo(db.NewMemDB())

	id := "testid-1"
	err := userRepo.Create(nil, user.User{
		ID:    id,
		Email: "testname@example.com",
	})
	if err != nil {
		return nil, err
	}

	err = userRepo.AddRemoteIdentity(nil, id, user.RemoteIdentity{
		ConnectorID: "test_connector_id",
		ID:          "YYY",
	})
	if err != nil {
		return nil, err
	}

	return userRepo, nil
}

func TestServerProviderConfig(t *testing.T) {
	srv := &Server{IssuerURL: url.URL{Scheme: "http", Host: "server.example.com"}}

	want := oidc.ProviderConfig{
		Issuer:        &url.URL{Scheme: "http", Host: "server.example.com"},
		AuthEndpoint:  &url.URL{Scheme: "http", Host: "server.example.com", Path: "/auth"},
		TokenEndpoint: &url.URL{Scheme: "http", Host: "server.example.com", Path: "/token"},
		KeysEndpoint:  &url.URL{Scheme: "http", Host: "server.example.com", Path: "/keys"},

		GrantTypesSupported:               []string{oauth2.GrantTypeAuthCode, oauth2.GrantTypeClientCreds},
		ResponseTypesSupported:            []string{"code"},
		SubjectTypesSupported:             []string{"public"},
		IDTokenSigningAlgValues:           []string{"RS256"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic"},
	}
	got := srv.ProviderConfig()

	if diff := pretty.Compare(want, got); diff != "" {
		t.Fatalf("provider config did not match expected: %s", diff)
	}
}

func TestServerNewSession(t *testing.T) {
	sm := manager.NewSessionManager(db.NewSessionRepo(db.NewMemDB()), db.NewSessionKeyRepo(db.NewMemDB()))
	srv := &Server{
		SessionManager: sm,
	}

	state := "pants"
	nonce := "oncenay"
	ci := client.Client{
		Credentials: oidc.ClientCredentials{
			ID:     testClientID,
			Secret: clientTestSecret,
		},
		Metadata: oidc.ClientMetadata{
			RedirectURIs: []url.URL{
				url.URL{
					Scheme: "http",
					Host:   "client.example.com",
					Path:   "/callback",
				},
			},
		},
	}

	key, err := srv.NewSession("bogus_idpc", ci.Credentials.ID, state, ci.Metadata.RedirectURIs[0], nonce, false, []string{"openid"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	sessionID, err := sm.ExchangeKey(key)
	if err != nil {
		t.Fatalf("Session not retreivable: %v", err)
	}

	ses, err := sm.AttachRemoteIdentity(sessionID, oidc.Identity{})
	if err != nil {
		t.Fatalf("Unable to add Identity to Session: %v", err)
	}

	if !reflect.DeepEqual(ci.Metadata.RedirectURIs[0], ses.RedirectURL) {
		t.Fatalf("Session created with incorrect RedirectURL: want=%#v got=%#v", ci.Metadata.RedirectURIs[0], ses.RedirectURL)
	}

	if ci.Credentials.ID != ses.ClientID {
		t.Fatalf("Session created with incorrect ClientID: want=%q got=%q", ci.Credentials.ID, ses.ClientID)
	}

	if state != ses.ClientState {
		t.Fatalf("Session created with incorrect State: want=%q got=%q", state, ses.ClientState)
	}

	if nonce != ses.Nonce {
		t.Fatalf("Session created with incorrect Nonce: want=%q got=%q", nonce, ses.Nonce)
	}
}

func TestServerLogin(t *testing.T) {
	f, err := makeTestFixtures()
	if err != nil {
		t.Fatalf("error making test fixtures: %v", err)
	}

	sm := f.sessionManager
	sessionID, err := sm.NewSession("IDPC-1", testClientID, "bogus", testRedirectURL, "", false, []string{"openid"})

	ident := oidc.Identity{ID: testUserRemoteID1, Name: "elroy", Email: testUserEmail1}
	key, err := sm.NewSessionKey(sessionID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	redirectURL, err := f.srv.Login(ident, key)
	if err != nil {
		t.Fatalf("Unexpected err from Server.Login: %v", err)
	}

	wantRedirectURL := "http://client.example.com/callback?code=code-3&state=bogus"
	if wantRedirectURL != redirectURL {
		t.Fatalf("Unexpected redirectURL: want=%q, got=%q", wantRedirectURL, redirectURL)
	}
}

func TestServerLoginUnrecognizedSessionKey(t *testing.T) {
	f, err := makeTestFixtures()
	if err != nil {
		t.Fatalf("error making test fixtures: %v", err)
	}

	ident := oidc.Identity{ID: testUserRemoteID1, Name: "elroy", Email: testUserEmail1}
	code, err := f.srv.Login(ident, testClientID)
	if err == nil {
		t.Fatalf("Expected non-nil error")
	}

	if code != "" {
		t.Fatalf("Expected empty code, got=%s", code)
	}
}

func TestServerLoginDisabledUser(t *testing.T) {
	f, err := makeTestFixtures()
	if err != nil {
		t.Fatalf("error making test fixtures: %v", err)
	}

	err = f.userRepo.Create(nil, user.User{
		ID:       "disabled-1",
		Email:    "disabled@example.com",
		Disabled: true,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = f.userRepo.AddRemoteIdentity(nil, "disabled-1", user.RemoteIdentity{
		ConnectorID: "test_connector_id",
		ID:          "disabled-connector-id",
	})

	sessionID, err := f.sessionManager.NewSession("test_connector_id", testClientID, "bogus", testRedirectURL, "", false, []string{"openid"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	ident := oidc.Identity{ID: "disabled-connector-id", Name: "elroy", Email: "elroy@example.com"}
	key, err := f.sessionManager.NewSessionKey(sessionID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	_, err = f.srv.Login(ident, key)
	if err == nil {
		t.Errorf("disabled user was allowed to log in")
	}
}

func TestServerCodeToken(t *testing.T) {
	f, err := makeTestFixtures()
	if err != nil {
		t.Fatalf("Error creating test fixtures: %v", err)
	}
	sm := f.sessionManager

	tests := []struct {
		scope        []string
		refreshToken string
	}{
		// No 'offline_access' in scope, should get empty refresh token.
		{
			scope:        []string{"openid"},
			refreshToken: "",
		},
		// Have 'offline_access' in scope, should get non-empty refresh token.
		{
			// NOTE(ericchiang): This test assumes that the database ID of the
			// first refresh token will be "1".
			scope:        []string{"openid", "offline_access"},
			refreshToken: fmt.Sprintf("1/%s", base64.URLEncoding.EncodeToString([]byte("refresh-1"))),
		},
	}

	for i, tt := range tests {
		sessionID, err := sm.NewSession("bogus_idpc", testClientID, "bogus", url.URL{}, "", false, tt.scope)
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}
		_, err = sm.AttachRemoteIdentity(sessionID, oidc.Identity{})
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}

		_, err = sm.AttachUser(sessionID, testUserID1)
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}

		key, err := sm.NewSessionKey(sessionID)
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}

		jwt, token, err := f.srv.CodeToken(oidc.ClientCredentials{
			ID:     testClientID,
			Secret: clientTestSecret}, key)
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}
		if jwt == nil {
			t.Fatalf("case %d: expect non-nil jwt", i)
		}
		if token != tt.refreshToken {
			t.Fatalf("case %d: expect refresh token %q, got %q", i, tt.refreshToken, token)
		}
	}
}

func TestServerTokenUnrecognizedKey(t *testing.T) {
	f, err := makeTestFixtures()
	if err != nil {
		t.Fatalf("error making test fixtures: %v", err)
	}
	sm := f.sessionManager

	sessionID, err := sm.NewSession("connector_id", testClientID, "bogus", url.URL{}, "", false, []string{"openid", "offline_access"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	_, err = sm.AttachRemoteIdentity(sessionID, oidc.Identity{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	jwt, token, err := f.srv.CodeToken(testClientCredentials, "foo")
	if err == nil {
		t.Fatalf("Expected non-nil error")
	}
	if jwt != nil {
		t.Fatalf("Expected nil jwt")
	}
	if token != "" {
		t.Fatalf("Expected empty refresh token")
	}
}

func TestServerTokenFail(t *testing.T) {
	keyFixture := "goodkey"

	signerFixture := &StaticSigner{sig: []byte("beer"), err: nil}

	tests := []struct {
		signer       jose.Signer
		argCC        oidc.ClientCredentials
		argKey       string
		err          error
		scope        []string
		refreshToken string
	}{
		// control test case to make sure fixtures check out
		{
			// NOTE(ericchiang): This test assumes that the database ID of the first
			// refresh token will be "1".
			signer:       signerFixture,
			argCC:        testClientCredentials,
			argKey:       keyFixture,
			scope:        []string{"openid", "offline_access"},
			refreshToken: fmt.Sprintf("1/%s", base64.URLEncoding.EncodeToString([]byte("refresh-1"))),
		},

		// no 'offline_access' in 'scope', should get empty refresh token
		{
			signer: signerFixture,
			argCC:  testClientCredentials,
			argKey: keyFixture,
			scope:  []string{"openid"},
		},

		// unrecognized key
		{
			signer: signerFixture,
			argCC:  testClientCredentials,
			argKey: "foo",
			err:    oauth2.NewError(oauth2.ErrorInvalidGrant),
			scope:  []string{"openid", "offline_access"},
		},

		// unrecognized client
		{
			signer: signerFixture,
			argCC:  oidc.ClientCredentials{ID: "YYY"},
			argKey: keyFixture,
			err:    oauth2.NewError(oauth2.ErrorInvalidClient),
			scope:  []string{"openid", "offline_access"},
		},

		// signing operation fails
		{
			signer: &StaticSigner{sig: nil, err: errors.New("fail")},
			argCC:  testClientCredentials,
			argKey: keyFixture,
			err:    oauth2.NewError(oauth2.ErrorServerError),
			scope:  []string{"openid", "offline_access"},
		},
	}

	for i, tt := range tests {

		f, err := makeTestFixtures()
		if err != nil {
			t.Fatalf("error making test fixtures: %v", err)
		}
		sm := f.sessionManager
		sm.GenerateCode = func() (string, error) { return keyFixture, nil }
		f.srv.RefreshTokenRepo = refreshtest.NewTestRefreshTokenRepo()
		f.srv.KeyManager = &StaticKeyManager{
			signer: tt.signer,
		}

		sessionID, err := sm.NewSession(testConnectorID1, testClientID, "bogus", url.URL{}, "", false, tt.scope)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		_, err = sm.AttachRemoteIdentity(sessionID, oidc.Identity{})
		if err != nil {
			t.Errorf("case %d: unexpected error: %v", i, err)
			continue
		}
		_, err = sm.AttachUser(sessionID, testUserID1)
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}

		_, err = sm.NewSessionKey(sessionID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		jwt, token, err := f.srv.CodeToken(tt.argCC, tt.argKey)
		if token != tt.refreshToken {
			fmt.Printf("case %d: expect refresh token %q, got %q\n", i, tt.refreshToken, token)
			t.Fatalf("case %d: expect refresh token %q, got %q", i, tt.refreshToken, token)
			panic("")
		}
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("case %d: expect %v, got %v", i, tt.err, err)
		}
		if err == nil && jwt == nil {
			t.Errorf("case %d: got nil JWT", i)
		}
		if err != nil && jwt != nil {
			t.Errorf("case %d: got non-nil JWT %v", i, jwt)
		}
	}
}

func TestServerRefreshToken(t *testing.T) {

	clientB := client.Client{
		Credentials: oidc.ClientCredentials{
			ID:     "example2.com",
			Secret: clientTestSecret,
		},
		Metadata: oidc.ClientMetadata{
			RedirectURIs: []url.URL{
				url.URL{Scheme: "https", Host: "example2.com", Path: "one/two/three"},
			},
		},
	}
	signerFixture := &StaticSigner{sig: []byte("beer"), err: nil}

	// NOTE(ericchiang): These tests assume that the database ID of the first
	// refresh token will be "1".
	tests := []struct {
		token         string
		clientID      string // The client that associates with the token.
		creds         oidc.ClientCredentials
		signer        jose.Signer
		createScopes  []string
		refreshScopes []string
		err           error
	}{
		// Everything is good.
		{
			fmt.Sprintf("1/%s", base64.URLEncoding.EncodeToString([]byte("refresh-1"))),
			testClientID,
			testClientCredentials,
			signerFixture,
			[]string{"openid", "profile"},
			[]string{"openid", "profile"},
			nil,
		},
		// Asking for a scope not originally granted to you.
		{
			fmt.Sprintf("1/%s", base64.URLEncoding.EncodeToString([]byte("refresh-1"))),
			testClientID,
			testClientCredentials,
			signerFixture,
			[]string{"openid", "profile"},
			[]string{"openid", "profile", "extra_scope"},
			oauth2.NewError(oauth2.ErrorInvalidRequest),
		},
		// Invalid refresh token(malformatted).
		{
			"invalid-token",
			testClientID,
			testClientCredentials,
			signerFixture,
			[]string{"openid", "profile"},
			[]string{"openid", "profile"},
			oauth2.NewError(oauth2.ErrorInvalidRequest),
		},
		// Invalid refresh token(invalid payload content).
		{
			fmt.Sprintf("1/%s", base64.URLEncoding.EncodeToString([]byte("refresh-2"))),
			testClientID,
			testClientCredentials,
			signerFixture,
			[]string{"openid", "profile"},
			[]string{"openid", "profile"},
			oauth2.NewError(oauth2.ErrorInvalidRequest),
		},
		// Invalid refresh token(invalid ID content).
		{
			fmt.Sprintf("0/%s", base64.URLEncoding.EncodeToString([]byte("refresh-1"))),
			testClientID,
			testClientCredentials,
			signerFixture,
			[]string{"openid", "profile"},
			[]string{"openid", "profile"},
			oauth2.NewError(oauth2.ErrorInvalidRequest),
		},
		// Invalid client(client is not associated with the token).
		{
			fmt.Sprintf("1/%s", base64.URLEncoding.EncodeToString([]byte("refresh-1"))),
			testClientID,
			clientB.Credentials,
			signerFixture,
			[]string{"openid", "profile"},
			[]string{"openid", "profile"},
			oauth2.NewError(oauth2.ErrorInvalidClient),
		},
		// Invalid client(no client ID).
		{
			fmt.Sprintf("1/%s", base64.URLEncoding.EncodeToString([]byte("refresh-1"))),
			testClientID,
			oidc.ClientCredentials{ID: "", Secret: "aaa"},
			signerFixture,
			[]string{"openid", "profile"},
			[]string{"openid", "profile"},
			oauth2.NewError(oauth2.ErrorInvalidClient),
		},
		// Invalid client(no such client).
		{
			fmt.Sprintf("1/%s", base64.URLEncoding.EncodeToString([]byte("refresh-1"))),
			testClientID,
			oidc.ClientCredentials{ID: "AAA", Secret: "aaa"},
			signerFixture,
			[]string{"openid", "profile"},
			[]string{"openid", "profile"},
			oauth2.NewError(oauth2.ErrorInvalidClient),
		},
		// Invalid client(no secrets).
		{
			fmt.Sprintf("1/%s", base64.URLEncoding.EncodeToString([]byte("refresh-1"))),
			testClientID,
			oidc.ClientCredentials{ID: testClientID},
			signerFixture,
			[]string{"openid", "profile"},
			[]string{"openid", "profile"},
			oauth2.NewError(oauth2.ErrorInvalidClient),
		},
		// Invalid client(invalid secret).
		{
			fmt.Sprintf("1/%s", base64.URLEncoding.EncodeToString([]byte("refresh-1"))),
			testClientID,
			oidc.ClientCredentials{ID: "bad-id", Secret: "bad-secret"},
			signerFixture,
			[]string{"openid", "profile"},
			[]string{"openid", "profile"},
			oauth2.NewError(oauth2.ErrorInvalidClient),
		},
		// Signing operation fails.
		{
			fmt.Sprintf("1/%s", base64.URLEncoding.EncodeToString([]byte("refresh-1"))),
			testClientID,
			testClientCredentials,
			&StaticSigner{sig: nil, err: errors.New("fail")},
			[]string{"openid", "profile"},
			[]string{"openid", "profile"},
			oauth2.NewError(oauth2.ErrorServerError),
		},
	}

	for i, tt := range tests {
		km := &StaticKeyManager{
			signer: tt.signer,
		}
		f, err := makeTestFixtures()
		if err != nil {
			t.Fatalf("error making test fixtures: %v", err)
		}
		f.srv.RefreshTokenRepo = refreshtest.NewTestRefreshTokenRepo()
		f.srv.KeyManager = km
		_, err = f.clientRepo.New(nil, clientB)
		if err != nil {
			t.Errorf("case %d: error creating other client: %v", i, err)
		}

		if _, err := f.srv.RefreshTokenRepo.Create(testUserID1, tt.clientID,
			tt.createScopes); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		jwt, err := f.srv.RefreshToken(tt.creds, tt.refreshScopes, tt.token)
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("Case %d: expect: %v, got: %v", i, tt.err, err)
		}

		if jwt != nil {
			if string(jwt.Signature) != "beer" {
				t.Errorf("Case %d: expect signature: beer, got signature: %v", i, jwt.Signature)
			}
			claims, err := jwt.Claims()
			if err != nil {
				t.Errorf("Case %d: unexpected error: %v", i, err)
			}
			if claims["iss"] != testIssuerURL.String() || claims["sub"] != testUserID1 || claims["aud"] != testClientID {
				t.Errorf("Case %d: invalid claims: %v", i, claims)
			}
		}
	}
}
