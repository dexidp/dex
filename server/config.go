package server

import (
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	texttemplate "text/template"
	"time"

	"github.com/coreos/go-oidc/key"
	"github.com/coreos/pkg/health"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/db"
	"github.com/coreos/dex/email"
	"github.com/coreos/dex/refresh"
	"github.com/coreos/dex/repo"
	"github.com/coreos/dex/session"
	"github.com/coreos/dex/user"
	useremail "github.com/coreos/dex/user/email"
)

type ServerConfig struct {
	IssuerURL          string
	IssuerName         string
	IssuerLogoURL      string
	TemplateDir        string
	EmailTemplateDirs  []string
	EmailFromAddress   string
	EmailerConfigFile  string
	StateConfig        StateConfigurer
	EnableRegistration bool
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

		EnableRegistration: cfg.EnableRegistration,
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

	ks := key.NewPrivateKeySet([]*key.PrivateKey{k}, time.Now().Add(24*time.Hour))
	kRepo := key.NewPrivateKeySetRepo()
	if err = kRepo.Set(ks); err != nil {
		return err
	}

	cf, err := os.Open(cfg.ClientsFile)
	if err != nil {
		return fmt.Errorf("unable to read clients from file %s: %v", cfg.ClientsFile, err)
	}
	defer cf.Close()
	ciRepo, err := client.NewClientIdentityRepoFromReader(cf)
	if err != nil {
		return fmt.Errorf("unable to read client identities from file %s: %v", cfg.ClientsFile, err)
	}

	cfgRepo, err := connector.NewConnectorConfigRepoFromFile(cfg.ConnectorsFile)
	if err != nil {
		return fmt.Errorf("unable to create ConnectorConfigRepo: %v", err)
	}

	sRepo := session.NewSessionRepo()
	skRepo := session.NewSessionKeyRepo()
	sm := session.NewSessionManager(sRepo, skRepo)

	userRepo, err := user.NewUserRepoFromFile(cfg.UsersFile)
	if err != nil {
		return fmt.Errorf("unable to read users from file: %v", err)
	}

	pwiRepo := user.NewPasswordInfoRepo()

	refTokRepo := refresh.NewRefreshTokenRepo()

	txnFactory := repo.InMemTransactionFactory
	userManager := user.NewManager(userRepo, pwiRepo, txnFactory, user.ManagerOptions{})
	srv.ClientIdentityRepo = ciRepo
	srv.KeySetRepo = kRepo
	srv.ConnectorConfigRepo = cfgRepo
	srv.UserRepo = userRepo
	srv.UserManager = userManager
	srv.PasswordInfoRepo = pwiRepo
	srv.SessionManager = sm
	srv.RefreshTokenRepo = refTokRepo
	return nil

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
	userManager := user.NewManager(userRepo, pwiRepo, db.TransactionFactory(dbc), user.ManagerOptions{})
	refreshTokenRepo := db.NewRefreshTokenRepo(dbc)

	sm := session.NewSessionManager(sRepo, skRepo)

	srv.ClientIdentityRepo = ciRepo
	srv.KeySetRepo = kRepo
	srv.ConnectorConfigRepo = cfgRepo
	srv.UserRepo = userRepo
	srv.UserManager = userManager
	srv.PasswordInfoRepo = pwiRepo
	srv.SessionManager = sm
	srv.RefreshTokenRepo = refreshTokenRepo
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
