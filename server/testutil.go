package server

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oidc"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/db"
	"github.com/coreos/dex/email"
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
	testIssuerURL = url.URL{Scheme: "http", Host: "server.example.com"}
	testClientID  = "XXX"

	testRedirectURL = url.URL{Scheme: "http", Host: "client.example.com", Path: "/callback"}

	testUsers = []user.UserWithRemoteIdentities{
		{
			User: user.User{
				ID:    "ID-1",
				Email: "Email-1@example.com",
			},
			RemoteIdentities: []user.RemoteIdentity{
				{
					ConnectorID: "IDPC-1",
					ID:          "RID-1",
				},
			},
		},
		{
			User: user.User{
				ID:            "ID-Verified",
				Email:         "Email-Verified@example.com",
				EmailVerified: true,
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

	testPrivKey, _ = key.GeneratePrivateKey()
)

type testFixtures struct {
	srv                *Server
	userRepo           user.UserRepo
	sessionManager     *sessionmanager.SessionManager
	emailer            *email.TemplatizedEmailer
	redirectURL        url.URL
	clientIdentityRepo client.ClientIdentityRepo
}

func sequentialGenerateCodeFunc() sessionmanager.GenerateCodeFunc {
	x := 0
	return func() (string, error) {
		x += 1
		return fmt.Sprintf("code-%d", x), nil
	}
}

func makeTestFixtures() (*testFixtures, error) {
	dbMap := db.NewMemDB()
	userRepo, err := db.NewUserRepoFromUsers(dbMap, testUsers)
	if err != nil {
		return nil, err
	}
	pwRepo, err := db.NewPasswordInfoRepoFromPasswordInfos(dbMap, testPasswordInfos)
	if err != nil {
		return nil, err
	}

	connConfigs := []connector.ConnectorConfig{
		&connector.OIDCConnectorConfig{
			ID:           "oidc",
			IssuerURL:    testIssuerURL.String(),
			ClientID:     "12345",
			ClientSecret: "567789",
		},
		&connector.OIDCConnectorConfig{
			ID:                   "oidc-trusted",
			IssuerURL:            testIssuerURL.String(),
			ClientID:             "12345-trusted",
			ClientSecret:         "567789-trusted",
			TrustedEmailProvider: true,
		},
		&connector.LocalConnectorConfig{
			ID: "local",
		},
	}
	connCfgRepo := connector.NewConnectorConfigRepoFromConfigs(connConfigs)

	manager := usermanager.NewUserManager(userRepo, pwRepo, connCfgRepo, db.TransactionFactory(dbMap), usermanager.ManagerOptions{})

	sessionManager := sessionmanager.NewSessionManager(db.NewSessionRepo(db.NewMemDB()), db.NewSessionKeyRepo(db.NewMemDB()))
	sessionManager.GenerateCode = sequentialGenerateCodeFunc()

	emailer, err := email.NewTemplatizedEmailerFromGlobs(
		emailTemplatesLocation+"/*.txt",
		emailTemplatesLocation+"/*.html",
		&email.FakeEmailer{})
	if err != nil {
		return nil, err
	}

	clientIdentityRepo, err := db.NewClientIdentityRepoFromClients(db.NewMemDB(), []oidc.ClientIdentity{
		oidc.ClientIdentity{
			Credentials: oidc.ClientCredentials{
				ID:     "XXX",
				Secret: base64.URLEncoding.EncodeToString([]byte("secrete")),
			},
			Metadata: oidc.ClientMetadata{
				RedirectURIs: []url.URL{
					testRedirectURL,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	km := key.NewPrivateKeyManager()
	err = km.Set(key.NewPrivateKeySet([]*key.PrivateKey{testPrivKey}, time.Now().Add(time.Minute)))
	if err != nil {
		return nil, err
	}

	tpl, err := getTemplates("dex",
		"https://coreos.com/assets/images/brand/coreos-mark-30px.png",
		true, templatesLocation)
	if err != nil {
		return nil, err
	}

	srv := &Server{
		IssuerURL:          testIssuerURL,
		SessionManager:     sessionManager,
		ClientIdentityRepo: clientIdentityRepo,
		Templates:          tpl,
		UserRepo:           userRepo,
		PasswordInfoRepo:   pwRepo,
		UserManager:        manager,
		KeyManager:         km,
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
		"noreply@example.com",
		srv.absURL(httpPathResetPassword),
		srv.absURL(httpPathEmailVerify),
		srv.absURL(httpPathAcceptInvitation),
	)

	return &testFixtures{
		srv:                srv,
		redirectURL:        testRedirectURL,
		userRepo:           userRepo,
		sessionManager:     sessionManager,
		emailer:            emailer,
		clientIdentityRepo: clientIdentityRepo,
	}, nil
}
