package integration

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oauth2"
	"github.com/coreos/go-oidc/oidc"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/db"
	phttp "github.com/coreos/dex/pkg/http"
	"github.com/coreos/dex/refresh/refreshtest"
	"github.com/coreos/dex/server"
	"github.com/coreos/dex/session/manager"
	"github.com/coreos/dex/user"
)

func mockServer(cis []client.LoadableClient) (*server.Server, error) {
	dbMap := db.NewMemDB()
	k, err := key.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("Unable to generate private key: %v", err)
	}

	km := key.NewPrivateKeyManager()
	err = km.Set(key.NewPrivateKeySet([]*key.PrivateKey{k}, time.Now().Add(time.Minute)))
	if err != nil {
		return nil, err
	}

	clientRepo, clientManager, err := makeClientRepoAndManager(dbMap, cis)
	if err != nil {
		return nil, err
	}

	sm := manager.NewSessionManager(db.NewSessionRepo(dbMap), db.NewSessionKeyRepo(dbMap))
	srv := &server.Server{
		IssuerURL:      url.URL{Scheme: "http", Host: "server.example.com"},
		KeyManager:     km,
		ClientRepo:     clientRepo,
		ClientManager:  clientManager,
		SessionManager: sm,
	}

	return srv, nil
}

func mockClient(srv *server.Server, ci client.Client) (*oidc.Client, error) {
	hdlr := srv.HTTPHandler()
	sClient := &phttp.HandlerClient{Handler: hdlr}

	cfg, err := oidc.FetchProviderConfig(sClient, srv.IssuerURL.String())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch provider config: %v", err)
	}

	jwks, err := srv.KeyManager.JWKs()
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWKs: %v", err)
	}

	ks := key.NewPublicKeySet(jwks, time.Now().Add(1*time.Hour))
	ccfg := oidc.ClientConfig{
		HTTPClient:     sClient,
		ProviderConfig: cfg,
		Credentials:    ci.Credentials,
		KeySet:         *ks,
	}

	return oidc.NewClient(ccfg)
}

func verifyUserClaims(claims jose.Claims, ci *client.Client, user *user.User, issuerURL url.URL) error {
	expectedSub, expectedName := ci.Credentials.ID, ci.Credentials.ID
	if user != nil {
		expectedSub, expectedName = user.ID, user.DisplayName
	}

	if aud, ok := claims["aud"].(string); !ok {
		return fmt.Errorf("unexpected claim value for aud, got=nil, want=%v", ci.Credentials.ID)
	} else if aud != ci.Credentials.ID {
		return fmt.Errorf("unexpected claim value for aud, got=%v, want=%v", aud, ci.Credentials.ID)
	}

	if sub, ok := claims["sub"].(string); !ok {
		return fmt.Errorf("unexpected claim value for sub, got=nil, want=%v", expectedSub)
	} else if sub != expectedSub {
		return fmt.Errorf("unexpected claim value for sub, got=%v, want=%v", sub, expectedSub)
	}

	if name, ok := claims["name"].(string); !ok {
		return fmt.Errorf("unexpected claim value for aud, got=nil, want=%v", expectedName)
	} else if name != expectedName {
		return fmt.Errorf("unexpected claim value for name, got=%v, want=%v", name, expectedName)
	}

	wantIss := issuerURL.String()
	if iss := claims["iss"].(string); iss != wantIss {
		return fmt.Errorf("unexpected claim value for iss, got=%v, want=%v", iss, wantIss)
	}

	return nil
}

func TestHTTPExchangeTokenRefreshToken(t *testing.T) {
	password, err := user.NewPasswordFromPlaintext("woof")
	if err != nil {
		t.Fatalf("unexpectd error: %q", err)
	}

	passwordInfo := user.PasswordInfo{
		UserID:   "elroy77",
		Password: password,
	}

	cfg := &connector.LocalConnectorConfig{
		ID: "local",
	}

	validRedirURL := url.URL{
		Scheme: "http",
		Host:   "client.example.com",
		Path:   "/callback",
	}
	ci := client.Client{
		Credentials: oidc.ClientCredentials{
			ID:     validRedirURL.Host,
			Secret: base64.URLEncoding.EncodeToString([]byte("secret")),
		},
		Metadata: oidc.ClientMetadata{
			RedirectURIs: []url.URL{
				validRedirURL,
			},
		},
	}

	dbMap := db.NewMemDB()
	clientRepo, clientManager, err := makeClientRepoAndManager(dbMap,
		[]client.LoadableClient{{
			Client: ci,
		}})
	if err != nil {
		t.Fatalf("Failed to create client identity manager: " + err.Error())
	}

	passwordInfoRepo, err := db.NewPasswordInfoRepoFromPasswordInfos(db.NewMemDB(), []user.PasswordInfo{passwordInfo})
	if err != nil {
		t.Fatalf("Failed to create password info repo: %v", err)
	}

	issuerURL := url.URL{Scheme: "http", Host: "server.example.com"}
	sm := manager.NewSessionManager(db.NewSessionRepo(dbMap), db.NewSessionKeyRepo(dbMap))

	k, err := key.GeneratePrivateKey()
	if err != nil {
		t.Fatalf("Unable to generate RSA key: %v", err)
	}

	km := key.NewPrivateKeyManager()
	err = km.Set(key.NewPrivateKeySet([]*key.PrivateKey{k}, time.Now().Add(time.Minute)))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	usr := user.User{
		ID:          "ID-test",
		Email:       "testemail@example.com",
		DisplayName: "displayname",
	}
	userRepo := db.NewUserRepo(db.NewMemDB())
	if err := userRepo.Create(nil, usr); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	refreshTokenRepo := refreshtest.NewTestRefreshTokenRepo()

	srv := &server.Server{
		IssuerURL:        issuerURL,
		KeyManager:       km,
		SessionManager:   sm,
		ClientRepo:       clientRepo,
		ClientManager:    clientManager,
		Templates:        template.New(connector.LoginPageTemplateName),
		Connectors:       []connector.Connector{},
		UserRepo:         userRepo,
		PasswordInfoRepo: passwordInfoRepo,
		RefreshTokenRepo: refreshTokenRepo,
	}

	if err = srv.AddConnector(cfg); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	sClient := &phttp.HandlerClient{Handler: srv.HTTPHandler()}
	pcfg, err := oidc.FetchProviderConfig(sClient, issuerURL.String())
	if err != nil {
		t.Fatalf("Failed to fetch provider config: %v", err)
	}

	ks := key.NewPublicKeySet([]jose.JWK{k.JWK()}, time.Now().Add(1*time.Hour))

	ccfg := oidc.ClientConfig{
		HTTPClient:     sClient,
		ProviderConfig: pcfg,
		Credentials:    ci.Credentials,
		RedirectURL:    validRedirURL.String(),
		KeySet:         *ks,
	}

	cl, err := oidc.NewClient(ccfg)
	if err != nil {
		t.Fatalf("Failed creating oidc.Client: %v", err)
	}

	m := http.NewServeMux()

	var claims jose.Claims
	var refresh string

	m.HandleFunc("/callback", handleCallbackFunc(cl, &claims, &refresh))
	cClient := &phttp.HandlerClient{Handler: m}

	// this will actually happen due to some interaction between the
	// end-user and a remote identity provider
	sessionID, err := sm.NewSession("bogus_idpc", ci.Credentials.ID, "bogus", url.URL{}, "", false, []string{"openid", "offline_access", "email", "profile"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if _, err = sm.AttachRemoteIdentity(sessionID, passwordInfo.Identity()); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if _, err = sm.AttachUser(sessionID, usr.ID); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	key, err := sm.NewSessionKey(sessionID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("http://client.example.com/callback?code=%s", key), nil)
	if err != nil {
		t.Fatalf("Failed creating HTTP request: %v", err)
	}

	resp, err := cClient.Do(req)
	if err != nil {
		t.Fatalf("Failed resolving HTTP requests against /callback: %v", err)
	}

	if err := verifyUserClaims(claims, &ci, &usr, issuerURL); err != nil {
		t.Fatalf("Failed to verify claims: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Received status code %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if refresh == "" {
		t.Fatalf("No refresh token")
	}

	// Use refresh token to get a new ID token.
	token, err := cl.RefreshToken(refresh)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	claims, err = token.Claims()
	if err != nil {
		t.Fatalf("Failed parsing claims from client token: %v", err)
	}

	if err := verifyUserClaims(claims, &ci, &usr, issuerURL); err != nil {
		t.Fatalf("Failed to verify claims: %v", err)
	}
}

func TestHTTPClientCredsToken(t *testing.T) {
	validRedirURL := url.URL{
		Scheme: "http",
		Host:   "client.example.com",
		Path:   "/callback",
	}
	ci := client.Client{
		Credentials: oidc.ClientCredentials{
			ID:     validRedirURL.Host,
			Secret: base64.URLEncoding.EncodeToString([]byte("secret")),
		},
		Metadata: oidc.ClientMetadata{
			RedirectURIs: []url.URL{
				validRedirURL,
			},
		},
	}
	cis := []client.LoadableClient{{Client: ci}}

	srv, err := mockServer(cis)
	if err != nil {
		t.Fatalf("Unexpected error setting up server: %v", err)
	}

	cl, err := mockClient(srv, ci)
	if err != nil {
		t.Fatalf("Unexpected error setting up OIDC client: %v", err)
	}

	tok, err := cl.ClientCredsToken([]string{"openid"})
	if err != nil {
		t.Fatalf("Failed getting client token: %v", err)
	}

	claims, err := tok.Claims()
	if err != nil {
		t.Fatalf("Failed parsing claims from client token: %v", err)
	}

	if err := verifyUserClaims(claims, &ci, nil, srv.IssuerURL); err != nil {
		t.Fatalf("Failed to verify claims: %v", err)
	}
}

func handleCallbackFunc(c *oidc.Client, claims *jose.Claims, refresh *string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			phttp.WriteError(w, http.StatusBadRequest, "code query param must be set")
			return
		}

		oac, err := c.OAuthClient()
		if err != nil {
			phttp.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("unable to create oauth client: %v", err))
			return
		}

		t, err := oac.RequestToken(oauth2.GrantTypeAuthCode, code)
		if err != nil {
			phttp.WriteError(w, http.StatusBadRequest, fmt.Sprintf("unable to verify auth code with issuer: %v", err))
			return
		}

		// Get id token and claims.
		tok, err := jose.ParseJWT(t.IDToken)
		if err != nil {
			phttp.WriteError(w, http.StatusBadRequest, fmt.Sprintf("unable to parse id_token: %v", err))
			return
		}

		if err := c.VerifyJWT(tok); err != nil {
			phttp.WriteError(w, http.StatusBadRequest, fmt.Sprintf("unable to verify the JWT: %v", err))
			return
		}

		if *claims, err = tok.Claims(); err != nil {
			phttp.WriteError(w, http.StatusBadRequest, fmt.Sprintf("unable to construct claims: %v", err))
			return
		}

		// Get refresh token.
		*refresh = t.RefreshToken

		w.WriteHeader(http.StatusOK)
	}
}
