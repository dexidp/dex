package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	texttemplate "text/template"
	"time"

	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oidc"
	"github.com/coreos/pkg/health"
	"github.com/go-gorp/gorp"

	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/db"
	"github.com/coreos/dex/email"
	sessionmanager "github.com/coreos/dex/session/manager"
	"github.com/coreos/dex/user"
	useremail "github.com/coreos/dex/user/email"
	usermanager "github.com/coreos/dex/user/manager"
)

type ServerConfig struct {
	IssuerURL                string
	IssuerName               string
	IssuerLogoURL            string
	TemplateDir              string
	EmailTemplateDirs        []string
	EmailFromAddress         string
	EmailerConfigFile        string
	StateConfig              StateConfigurer
	EnableRegistration       bool
	EnableClientRegistration bool
}

type StateConfigurer interface {
	Configure(*Server) error
}

type SingleServerConfig struct {
	ClientsFile    string
	ConnectorsFile string
	UsersFile      string
}

type MultiServerConfig struct {
	KeySecrets     [][]byte
	DatabaseConfig db.Config
	UseOldFormat   bool
}

func (cfg *ServerConfig) Server() (*Server, error) {
	iu, err := url.Parse(cfg.IssuerURL)
	if err != nil {
		return nil, err
	}

	tpl, err := getTemplates(cfg.IssuerName, cfg.IssuerLogoURL, cfg.EnableRegistration, cfg.TemplateDir)
	if err != nil {
		return nil, err
	}

	km := key.NewPrivateKeyManager()
	srv := Server{
		IssuerURL:  *iu,
		KeyManager: km,
		Templates:  tpl,

		HealthChecks: []health.Checkable{km},
		Connectors:   []connector.Connector{},

		EnableRegistration:       cfg.EnableRegistration,
		EnableClientRegistration: cfg.EnableClientRegistration,
	}

	err = cfg.StateConfig.Configure(&srv)
	if err != nil {
		return nil, err
	}

	err = setTemplates(&srv, tpl)
	if err != nil {
		return nil, err
	}

	err = setEmailer(&srv, cfg.IssuerName, cfg.EmailFromAddress, cfg.EmailerConfigFile, cfg.EmailTemplateDirs)
	if err != nil {
		return nil, err
	}
	return &srv, nil
}

func (cfg *SingleServerConfig) Configure(srv *Server) error {
	k, err := key.GeneratePrivateKey()
	if err != nil {
		return err
	}

	dbMap := db.NewMemDB()

	ks := key.NewPrivateKeySet([]*key.PrivateKey{k}, time.Now().Add(24*time.Hour))
	kRepo := key.NewPrivateKeySetRepo()
	if err = kRepo.Set(ks); err != nil {
		return err
	}

	clients, err := loadClients(cfg.ClientsFile)
	if err != nil {
		return fmt.Errorf("unable to read clients from file %s: %v", cfg.ClientsFile, err)
	}
	ciRepo, err := db.NewClientIdentityRepoFromClients(dbMap, clients)
	if err != nil {
		return fmt.Errorf("failed to create client identity repo: %v", err)
	}

	f, err := os.Open(cfg.ConnectorsFile)
	if err != nil {
		return fmt.Errorf("opening connectors file: %v", err)
	}
	defer f.Close()
	cfgs, err := connector.ReadConfigs(f)
	if err != nil {
		return fmt.Errorf("decoding connector configs: %v", err)
	}
	cfgRepo := db.NewConnectorConfigRepo(dbMap)
	if err := cfgRepo.Set(cfgs); err != nil {
		return fmt.Errorf("failed to set connectors: %v", err)
	}

	sRepo := db.NewSessionRepo(dbMap)
	skRepo := db.NewSessionKeyRepo(dbMap)
	sm := sessionmanager.NewSessionManager(sRepo, skRepo)

	users, err := loadUsers(cfg.UsersFile)
	if err != nil {
		return fmt.Errorf("unable to read users from file: %v", err)
	}
	userRepo, err := db.NewUserRepoFromUsers(dbMap, users)
	if err != nil {
		return err
	}

	pwiRepo := db.NewPasswordInfoRepo(dbMap)

	refTokRepo := db.NewRefreshTokenRepo(dbMap)

	txnFactory := db.TransactionFactory(dbMap)
	userManager := usermanager.NewUserManager(userRepo, pwiRepo, cfgRepo, txnFactory, usermanager.ManagerOptions{})
	srv.ClientIdentityRepo = ciRepo
	srv.KeySetRepo = kRepo
	srv.ConnectorConfigRepo = cfgRepo
	srv.UserRepo = userRepo
	srv.UserManager = userManager
	srv.PasswordInfoRepo = pwiRepo
	srv.SessionManager = sm
	srv.RefreshTokenRepo = refTokRepo
	srv.HealthChecks = append(srv.HealthChecks, db.NewHealthChecker(dbMap))
	return nil
}

func loadUsers(filepath string) (users []user.UserWithRemoteIdentities, err error) {
	f, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(&users)
	return
}

func loadClients(filepath string) ([]oidc.ClientIdentity, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var c []struct {
		ID           string   `json:"id"`
		Secret       string   `json:"secret"`
		RedirectURLs []string `json:"redirectURLs"`
	}
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		return nil, err
	}
	clients := make([]oidc.ClientIdentity, len(c))
	for i, client := range c {
		redirectURIs := make([]url.URL, len(client.RedirectURLs))
		for j, u := range client.RedirectURLs {
			uri, err := url.Parse(u)
			if err != nil {
				return nil, err
			}
			redirectURIs[j] = *uri
		}

		clients[i] = oidc.ClientIdentity{
			Credentials: oidc.ClientCredentials{
				ID:     client.ID,
				Secret: client.Secret,
			},
			Metadata: oidc.ClientMetadata{
				RedirectURIs: redirectURIs,
			},
		}
	}
	return clients, nil
}

func (cfg *MultiServerConfig) Configure(srv *Server) error {
	if len(cfg.KeySecrets) == 0 {
		return errors.New("missing key secret")
	}

	if cfg.DatabaseConfig.DSN == "" {
		return errors.New("missing database connection string")
	}

	dbc, err := db.NewConnection(cfg.DatabaseConfig)
	if err != nil {
		return fmt.Errorf("unable to initialize database connection: %v", err)
	}
	if _, ok := dbc.Dialect.(gorp.PostgresDialect); !ok {
		return errors.New("only postgres backend supported for multi server configurations")
	}

	kRepo, err := db.NewPrivateKeySetRepo(dbc, cfg.UseOldFormat, cfg.KeySecrets...)
	if err != nil {
		return fmt.Errorf("unable to create PrivateKeySetRepo: %v", err)
	}

	ciRepo := db.NewClientIdentityRepo(dbc)
	sRepo := db.NewSessionRepo(dbc)
	skRepo := db.NewSessionKeyRepo(dbc)
	cfgRepo := db.NewConnectorConfigRepo(dbc)
	userRepo := db.NewUserRepo(dbc)
	pwiRepo := db.NewPasswordInfoRepo(dbc)
	userManager := usermanager.NewUserManager(userRepo, pwiRepo, cfgRepo, db.TransactionFactory(dbc), usermanager.ManagerOptions{})
	refreshTokenRepo := db.NewRefreshTokenRepo(dbc)

	sm := sessionmanager.NewSessionManager(sRepo, skRepo)

	srv.ClientIdentityRepo = ciRepo
	srv.KeySetRepo = kRepo
	srv.ConnectorConfigRepo = cfgRepo
	srv.UserRepo = userRepo
	srv.UserManager = userManager
	srv.PasswordInfoRepo = pwiRepo
	srv.SessionManager = sm
	srv.RefreshTokenRepo = refreshTokenRepo
	srv.HealthChecks = append(srv.HealthChecks, db.NewHealthChecker(dbc))
	return nil
}

func getTemplates(issuerName, issuerLogoURL string,
	enableRegister bool, dir string) (*template.Template, error) {
	tpl := template.New("").Funcs(map[string]interface{}{
		"issuerName": func() string {
			return issuerName
		},
		"issuerLogoURL": func() string {
			return issuerLogoURL
		},
		"enableRegister": func() bool {
			return enableRegister
		},
	})

	return tpl.ParseGlob(dir + "/*.html")
}

func setTemplates(srv *Server, tpls *template.Template) error {
	ltpl, err := findTemplate(LoginPageTemplateName, tpls)
	if err != nil {
		return err
	}
	srv.LoginTemplate = ltpl

	rtpl, err := findTemplate(RegisterTemplateName, tpls)
	if err != nil {
		return err
	}
	srv.RegisterTemplate = rtpl

	vtpl, err := findTemplate(VerifyEmailTemplateName, tpls)
	if err != nil {
		return err
	}
	srv.VerifyEmailTemplate = vtpl

	srtpl, err := findTemplate(SendResetPasswordEmailTemplateName, tpls)
	if err != nil {
		return err
	}
	srv.SendResetPasswordEmailTemplate = srtpl

	rpwtpl, err := findTemplate(ResetPasswordTemplateName, tpls)
	if err != nil {
		return err
	}
	srv.ResetPasswordTemplate = rpwtpl

	return nil
}

func setEmailer(srv *Server, issuerName, fromAddress, emailerConfigFile string, emailTemplateDirs []string) error {

	cfg, err := email.NewEmailerConfigFromFile(emailerConfigFile)
	if err != nil {
		return err
	}

	emailer, err := cfg.Emailer()
	if err != nil {
		return err
	}

	getFileNames := func(dir, ext string) ([]string, error) {
		fns, err := filepath.Glob(dir + "/*." + ext)
		if err != nil {
			return nil, err
		}
		return fns, nil
	}
	getTextFiles := func(dir string) ([]string, error) {
		return getFileNames(dir, "txt")
	}
	getHTMLFiles := func(dir string) ([]string, error) {
		return getFileNames(dir, "html")
	}

	textTemplates := texttemplate.New("textTemplates")
	htmlTemplates := template.New("htmlTemplates")
	for _, dir := range emailTemplateDirs {
		textFileNames, err := getTextFiles(dir)
		if err != nil {
			return err
		}
		if len(textFileNames) != 0 {
			textTemplates, err = textTemplates.ParseFiles(textFileNames...)
		}
		if err != nil {
			return err
		}

		htmlFileNames, err := getHTMLFiles(dir)
		if err != nil {
			return err
		}
		if len(htmlFileNames) != 0 {
			htmlTemplates, err = htmlTemplates.ParseFiles(htmlFileNames...)
		}
		if err != nil {
			return err
		}
	}
	tMailer := email.NewTemplatizedEmailerFromTemplates(textTemplates, htmlTemplates, emailer)
	tMailer.SetGlobalContext(map[string]interface{}{
		"issuer_name": issuerName,
	})

	ue := useremail.NewUserEmailer(srv.UserRepo,
		srv.PasswordInfoRepo,
		srv.KeyManager.Signer,
		srv.SessionManager.ValidityWindow,
		srv.IssuerURL,
		tMailer,
		fromAddress,
		srv.absURL(httpPathResetPassword),
		srv.absURL(httpPathEmailVerify),
		srv.absURL(httpPathAcceptInvitation),
	)

	srv.UserEmailer = ue
	return nil
}

func findTemplate(name string, tpls *template.Template) (*template.Template, error) {
	tpl := tpls.Lookup(name)
	if tpl == nil {
		return nil, fmt.Errorf("unable to find template: %q", name)
	}
	return tpl, nil
}
