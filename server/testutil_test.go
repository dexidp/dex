package server

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oidc"

	"github.com/coreos/dex/client"
	clientmanager "github.com/coreos/dex/client/manager"
	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/db"
	"github.com/coreos/dex/email"
	"github.com/coreos/dex/refresh/refreshtest"
	sessionmanager "github.com/coreos/dex/session/manager"
	"github.com/coreos/dex/user"
	useremail "github.com/coreos/dex/user/email"
	usermanager "github.com/coreos/dex/user/manager"
)

const (
	templatesLocation      = "../static/html"
	emailTemplatesLocation = "../static/email"
)

var (
	testUserID1       = "ID-1"
	testUserEmail1    = "Email-1@example.com"
	testUserRemoteID1 = "RID-1"
	testOrganization  = "OrgID-1"

	testIssuerURL = url.URL{Scheme: "http", Host: "server.example.com"}

	testClientID          = "client.example.com"
	clientTestSecret      = base64.URLEncoding.EncodeToString([]byte("secret"))
	testClientCredentials = oidc.ClientCredentials{
		ID:     testClientID,
		Secret: clientTestSecret,
	}

	testPublicClientID          = "publicclient.example.com"
	publicClientTestSecret      = base64.URLEncoding.EncodeToString([]byte("secret"))
	testPublicClientCredentials = oidc.ClientCredentials{
		ID:     testPublicClientID,
		Secret: publicClientTestSecret,
	}
	testClients = []client.LoadableClient{
		{
			Client: client.Client{
				Credentials: testClientCredentials,
				Metadata: oidc.ClientMetadata{
					RedirectURIs: []url.URL{
						testRedirectURL,
					},
				},
			},
		},
		{
			Client: client.Client{
				Credentials: testPublicClientCredentials,
				Public:      true,
			},
		},
	}

	testConnectorID1 = "IDPC-1"

	testConnectorIDOpenID        = "oidc"
	testConnectorIDOpenIDTrusted = "oidc-trusted"
	testConnectorLocalID         = "local"

	testRedirectURL = url.URL{Scheme: "http", Host: "client.example.com", Path: "/callback"}

	testUsers = []user.UserWithRemoteIdentities{
		{
			User: user.User{
				ID:    testUserID1,
				Email: testUserEmail1,
			},
			RemoteIdentities: []user.RemoteIdentity{
				{
					ConnectorID: testConnectorID1,
					ID:          testUserRemoteID1,
				},
			},
		},
		{
			User: user.User{
				ID:             "ID-Verified",
				Email:          "Email-Verified@example.com",
				EmailVerified:  true,
				OrganizationID: "OrgID-1",
			},
			RemoteIdentities: []user.RemoteIdentity{
				{
					ConnectorID: "IDPC-1",
					ID:          "RID-2",
				},
			},
		},
	}

	testPasswordInfos = []user.PasswordInfo{
		{
			UserID:   "ID-1",
			Password: []byte("password"),
		},
		{
			UserID:   "ID-Verified",
			Password: []byte("password"),
		},
	}

	testOrganizations = []user.Organization{
		{
			OrganizationID: testOrganization,
			Name:           "OrgName-1",
			OwnerID:        "ID-1",
		},
	}

	testPrivKey, _ = key.GeneratePrivateKey()

	testClientCreds = oidc.ClientCredentials{
		ID:     testClientID,
		Secret: base64.URLEncoding.EncodeToString([]byte("secret")),
	}
)

type testFixtures struct {
	srv            *Server
	userRepo       user.UserRepo
	sessionManager *sessionmanager.SessionManager
	emailer        *email.TemplatizedEmailer
	redirectURL    url.URL
	clientRepo     client.ClientRepo
	clientManager  *clientmanager.ClientManager
	clientCreds    map[string]oidc.ClientCredentials
}

type testFixtureOptions struct {
	clients []client.LoadableClient
}

func sequentialGenerateCodeFunc() sessionmanager.GenerateCodeFunc {
	x := 0
	return func() (string, error) {
		x += 1
		return fmt.Sprintf("code-%d", x), nil
	}
}

func makeTestFixtures() (*testFixtures, error) {
	return makeTestFixturesWithOptions(testFixtureOptions{})
}

func makeTestFixturesWithOptions(options testFixtureOptions) (*testFixtures, error) {
	dbMap := db.NewMemDB()
	userRepo, err := db.NewUserRepoFromUsers(dbMap, testUsers)
	if err != nil {
		return nil, err
	}
	pwRepo, err := db.NewPasswordInfoRepoFromPasswordInfos(dbMap, testPasswordInfos)
	if err != nil {
		return nil, err
	}
	orgRepo, err := db.NewOrganizationRepoFromOrganizations(dbMap, testOrganizations)
	if err != nil {
		return nil, err
	}

	connConfigs := []connector.ConnectorConfig{
		&connector.OIDCConnectorConfig{
			ID:           testConnectorIDOpenID,
			IssuerURL:    testIssuerURL.String(),
			ClientID:     "12345",
			ClientSecret: "567789",
		},
		&connector.OIDCConnectorConfig{
			ID:                   testConnectorIDOpenIDTrusted,
			IssuerURL:            testIssuerURL.String(),
			ClientID:             "12345-trusted",
			ClientSecret:         "567789-trusted",
			TrustedEmailProvider: true,
		},
		&connector.OIDCConnectorConfig{
			ID:                   testConnectorID1,
			IssuerURL:            testIssuerURL.String(),
			ClientID:             testConnectorID1 + "_client_id",
			ClientSecret:         testConnectorID1 + "_client_secret",
			TrustedEmailProvider: true,
		},
		&connector.LocalConnectorConfig{
			ID: testConnectorLocalID,
		},
	}
	connCfgRepo := db.NewConnectorConfigRepo(dbMap)
	if err := connCfgRepo.Set(connConfigs); err != nil {
		return nil, err
	}

	userManager := usermanager.NewUserManager(userRepo, pwRepo, orgRepo, connCfgRepo, db.TransactionFactory(dbMap), usermanager.ManagerOptions{})

	sessionManager := sessionmanager.NewSessionManager(db.NewSessionRepo(db.NewMemDB()), db.NewSessionKeyRepo(db.NewMemDB()))
	sessionManager.GenerateCode = sequentialGenerateCodeFunc()

	refreshTokenRepo := refreshtest.NewTestRefreshTokenRepo()

	emailer, err := email.NewTemplatizedEmailerFromGlobs(
		emailTemplatesLocation+"/*.txt",
		emailTemplatesLocation+"/*.html",
		&email.FakeEmailer{},
		"admin@example.com")
	if err != nil {
		return nil, err
	}

	var clients []client.LoadableClient
	if options.clients == nil {
		clients = testClients
	} else {
		clients = options.clients
	}

	clientIDGenerator := func(hostport string) (string, error) {
		return hostport, nil
	}
	secGen := func() ([]byte, error) {
		return []byte("secret"), nil
	}
	clientRepo, err := db.NewClientRepoFromClients(dbMap, clients)
	if err != nil {
		return nil, err
	}

	clientManager := clientmanager.NewClientManager(clientRepo, db.TransactionFactory(dbMap), clientmanager.ManagerOptions{ClientIDGenerator: clientIDGenerator, SecretGenerator: secGen})

	km := key.NewPrivateKeyManager()
	err = km.Set(key.NewPrivateKeySet([]*key.PrivateKey{testPrivKey}, time.Now().Add(time.Minute)))
	if err != nil {
		return nil, err
	}

	tpl, err := getTemplates("dex", "https://coreos.com",
		"https://coreos.com/assets/images/brand/coreos-mark-30px.png",
		true, templatesLocation)
	if err != nil {
		return nil, err
	}

	srv := &Server{
		IssuerURL:        testIssuerURL,
		SessionManager:   sessionManager,
		ClientRepo:       clientRepo,
		Templates:        tpl,
		UserRepo:         userRepo,
		PasswordInfoRepo: pwRepo,
		UserManager:      userManager,
		ClientManager:    clientManager,
		KeyManager:       km,
		RefreshTokenRepo: refreshTokenRepo,
	}

	err = setTemplates(srv, tpl)
	if err != nil {
		return nil, err
	}

	for _, config := range connConfigs {
		if err := srv.AddConnector(config); err != nil {
			return nil, err
		}
	}

	srv.UserEmailer = useremail.NewUserEmailer(srv.UserRepo,
		srv.PasswordInfoRepo,
		srv.KeyManager.Signer,
		srv.SessionManager.ValidityWindow,
		srv.IssuerURL,
		emailer,
		srv.absURL(httpPathResetPassword),
		srv.absURL(httpPathEmailVerify),
		srv.absURL(httpPathAcceptInvitation),
	)

	clientCreds := map[string]oidc.ClientCredentials{}
	for _, c := range clients {
		clientCreds[c.Client.Credentials.ID] = c.Client.Credentials
	}
	return &testFixtures{
		srv:            srv,
		redirectURL:    testRedirectURL,
		userRepo:       userRepo,
		sessionManager: sessionManager,
		emailer:        emailer,
		clientRepo:     clientRepo,
		clientManager:  clientManager,
		clientCreds:    clientCreds,
	}, nil
}

func clientsToLoadableClients(cs []client.Client) []client.LoadableClient {
	lcs := make([]client.LoadableClient, len(cs), len(cs))
	for i, c := range cs {
		lcs[i] = client.LoadableClient{
			Client: c,
		}
	}
	return lcs
}
