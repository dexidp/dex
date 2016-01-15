package server

import (
	"fmt"
	"html/template"
	"net/url"
	"path/filepath"
	texttemplate "text/template"

	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/db"
	"github.com/coreos/dex/email"
	"github.com/coreos/dex/session"
	useremail "github.com/coreos/dex/user/email"
	"github.com/coreos/dex/user/manager"
	"github.com/coreos/go-oidc/key"
	"github.com/coreos/pkg/health"
)

type ServerConfig struct {
	IssuerURL          string
	IssuerName         string
	IssuerLogoURL      string
	TemplateDir        string
	EmailTemplateDirs  []string
	EmailFromAddress   string
	EmailerConfigFile  string
	EnableRegistration bool
	KeySecrets         [][]byte
	UseOldFormat       bool
	DB                 db.Driver
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

	err = cfg.Configure(&srv)
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

func (cfg *ServerConfig) Configure(srv *Server) error {
	kRepo, err := cfg.DB.NewPrivateKeySetRepo(cfg.UseOldFormat, cfg.KeySecrets...)
	if err != nil {
		return err
	}
	ciRepo := cfg.DB.NewClientIdentityRepo()
	cfgRepo := cfg.DB.NewConnectorConfigRepo()
	sRepo := cfg.DB.NewSessionRepo()
	skRepo := cfg.DB.NewSessionKeyRepo()
	userRepo := cfg.DB.NewUserRepo()
	pwiRepo := cfg.DB.NewPasswordInfoRepo()
	refTokRepo := cfg.DB.NewRefreshTokenRepo()

	sm := session.NewSessionManager(sRepo, skRepo)

	txnFactory := cfg.DB.GetTransactionFactory()
	userManager := manager.NewUserManager(userRepo, pwiRepo, cfgRepo, txnFactory, manager.ManagerOptions{})

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
