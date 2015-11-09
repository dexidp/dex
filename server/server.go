package server

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"sort"
	"time"

	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oauth2"
	"github.com/coreos/go-oidc/oidc"
	"github.com/coreos/pkg/health"
	"github.com/jonboulle/clockwork"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/refresh"
	"github.com/coreos/dex/session"
	"github.com/coreos/dex/user"
	usersapi "github.com/coreos/dex/user/api"
	useremail "github.com/coreos/dex/user/email"
)

const (
	LoginPageTemplateName              = "login.html"
	RegisterTemplateName               = "register.html"
	VerifyEmailTemplateName            = "verify-email.html"
	SendResetPasswordEmailTemplateName = "send-reset-password.html"
	ResetPasswordTemplateName          = "reset-password.html"

	APIVersion = "v1"
)

type OIDCServer interface {
	ClientMetadata(string) (*oidc.ClientMetadata, error)
	NewSession(connectorID, clientID, clientState string, redirectURL url.URL, nonce string, register bool, scope []string) (string, error)
	Login(oidc.Identity, string) (string, error)
	// CodeToken exchanges a code for an ID token and a refresh token string on success.
	CodeToken(creds oidc.ClientCredentials, sessionKey string) (*jose.JWT, string, error)
	ClientCredsToken(creds oidc.ClientCredentials) (*jose.JWT, error)
	// RefreshToken takes a previously generated refresh token and returns a new ID token
	// if the token is valid.
	RefreshToken(creds oidc.ClientCredentials, token string) (*jose.JWT, error)
	KillSession(string) error
}

type JWTVerifierFactory func(clientID string) oidc.JWTVerifier

type Server struct {
	IssuerURL                      url.URL
	KeyManager                     key.PrivateKeyManager
	KeySetRepo                     key.PrivateKeySetRepo
	SessionManager                 *session.SessionManager
	ClientIdentityRepo             client.ClientIdentityRepo
	ConnectorConfigRepo            connector.ConnectorConfigRepo
	Templates                      *template.Template
	LoginTemplate                  *template.Template
	RegisterTemplate               *template.Template
	VerifyEmailTemplate            *template.Template
	SendResetPasswordEmailTemplate *template.Template
	ResetPasswordTemplate          *template.Template
	HealthChecks                   []health.Checkable
	Connectors                     []connector.Connector
	UserRepo                       user.UserRepo
	UserManager                    *user.Manager
	PasswordInfoRepo               user.PasswordInfoRepo
	RefreshTokenRepo               refresh.RefreshTokenRepo
	UserEmailer                    *useremail.UserEmailer
	EnableRegistration             bool

	localConnectorID string
}

func (s *Server) Run() chan struct{} {
	stop := make(chan struct{})

	chans := []chan struct{}{
		key.NewKeySetSyncer(s.KeySetRepo, s.KeyManager).Run(),
	}

	for _, idpc := range s.Connectors {
		chans = append(chans, idpc.Sync())
	}

	go func() {
		<-stop
		for _, ch := range chans {
			close(ch)
		}
	}()

	return stop
}

func (s *Server) KillSession(sessionKey string) error {
	sessionID, err := s.SessionManager.ExchangeKey(sessionKey)

	if err != nil {
		return err
	}

	_, err = s.SessionManager.Kill(sessionID)
	return err
}

func (s *Server) ProviderConfig() oidc.ProviderConfig {
	iss := s.IssuerURL.String()
	cfg := oidc.ProviderConfig{
		Issuer: iss,

		AuthEndpoint:  iss + httpPathAuth,
		TokenEndpoint: iss + httpPathToken,
		KeysEndpoint:  iss + httpPathKeys,

		GrantTypesSupported:               []string{oauth2.GrantTypeAuthCode, oauth2.GrantTypeClientCreds},
		ResponseTypesSupported:            []string{"code"},
		SubjectTypesSupported:             []string{"public"},
		IDTokenAlgValuesSupported:         []string{"RS256"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic"},
	}

	return cfg
}

func (s *Server) absURL(paths ...string) url.URL {
	url := s.IssuerURL
	paths = append([]string{url.Path}, paths...)
	url.Path = path.Join(paths...)
	return url
}

func (s *Server) AddConnector(cfg connector.ConnectorConfig) error {
	connectorID := cfg.ConnectorID()
	ns := s.IssuerURL
	ns.Path = path.Join(ns.Path, httpPathAuth, connectorID)

	idpc, err := cfg.Connector(ns, s.Login, s.Templates)
	if err != nil {
		return err
	}

	s.Connectors = append(s.Connectors, idpc)

	sortable := sortableIDPCs(s.Connectors)
	sort.Sort(sortable)

	// We handle the LocalConnector specially because it needs access to the
	// UserRepo and the PasswordInfoRepo; if it turns out that other connectors
	// need access to these resources we'll figure out how to provide it in a
	// cleaner manner.
	localConn, ok := idpc.(*connector.LocalConnector)
	if ok {
		s.localConnectorID = connectorID
		if s.UserRepo == nil {
			return errors.New("UserRepo cannot be nil")
		}

		if s.PasswordInfoRepo == nil {
			return errors.New("PasswordInfoRepo cannot be nil")
		}

		localConn.SetLocalIdentityProvider(&connector.LocalIdentityProvider{
			UserRepo:         s.UserRepo,
			PasswordInfoRepo: s.PasswordInfoRepo,
		})

		localCfg, ok := cfg.(*connector.LocalConnectorConfig)
		if !ok {
			return errors.New("config for LocalConnector not a LocalConnectorConfig?")
		}

		if len(localCfg.PasswordInfos) > 0 {
			err := user.LoadPasswordInfos(s.PasswordInfoRepo,
				localCfg.PasswordInfos)
			if err != nil {
				return err
			}
		}
	}

	log.Infof("Loaded IdP connector: id=%s type=%s", connectorID, cfg.ConnectorType())
	return nil
}

func (s *Server) HTTPHandler() http.Handler {
	checks := make([]health.Checkable, len(s.HealthChecks))
	copy(checks, s.HealthChecks)
	for _, idpc := range s.Connectors {
		idpc := idpc
		checks = append(checks, idpc)
	}

	clock := clockwork.NewRealClock()
	mux := http.NewServeMux()
	mux.HandleFunc(httpPathDiscovery, handleDiscoveryFunc(s.ProviderConfig()))
	mux.HandleFunc(httpPathAuth, handleAuthFunc(s, s.Connectors, s.LoginTemplate, s.EnableRegistration))
	mux.HandleFunc(httpPathToken, handleTokenFunc(s))
	mux.HandleFunc(httpPathKeys, handleKeysFunc(s.KeyManager, clock))
	mux.Handle(httpPathHealth, makeHealthHandler(checks))

	if s.EnableRegistration {
		mux.HandleFunc(httpPathRegister, handleRegisterFunc(s))
	}

	mux.HandleFunc(httpPathEmailVerify, handleEmailVerifyFunc(s.VerifyEmailTemplate,
		s.IssuerURL, s.KeyManager.PublicKeys, s.UserManager))

	mux.Handle(httpPathVerifyEmailResend, s.NewClientTokenAuthHandler(handleVerifyEmailResendFunc(s.IssuerURL,
		s.KeyManager.PublicKeys,
		s.UserEmailer,
		s.UserRepo,
		s.ClientIdentityRepo)))

	mux.Handle(httpPathSendResetPassword, &SendResetPasswordEmailHandler{
		tpl:     s.SendResetPasswordEmailTemplate,
		emailer: s.UserEmailer,
		sm:      s.SessionManager,
		cr:      s.ClientIdentityRepo,
	})

	mux.Handle(httpPathResetPassword, &ResetPasswordHandler{
		tpl:       s.ResetPasswordTemplate,
		issuerURL: s.IssuerURL,
		um:        s.UserManager,
		keysFunc:  s.KeyManager.PublicKeys,
	})

	mux.Handle(httpPathAcceptInvitation, &InvitationHandler{
		passwordResetURL:       s.absURL(httpPathResetPassword),
		issuerURL:              s.IssuerURL,
		um:                     s.UserManager,
		keysFunc:               s.KeyManager.PublicKeys,
		signerFunc:             s.KeyManager.Signer,
		redirectValidityWindow: s.SessionManager.ValidityWindow,
	})

	mux.HandleFunc(httpPathDebugVars, health.ExpvarHandler)

	pcfg := s.ProviderConfig()
	for _, idpc := range s.Connectors {
		errorURL, err := url.Parse(fmt.Sprintf("%s?connector_id=%s", pcfg.AuthEndpoint, idpc.ID()))
		if err != nil {
			log.Fatal(err)
		}
		idpc.Register(mux, *errorURL)
	}

	apiBasePath := path.Join(httpPathAPI, APIVersion)
	registerDiscoveryResource(apiBasePath, mux)

	clientPath, clientHandler := registerClientResource(apiBasePath, s.ClientIdentityRepo)
	mux.Handle(path.Join(apiBasePath, clientPath), s.NewClientTokenAuthHandler(clientHandler))

	usersAPI := usersapi.NewUsersAPI(s.UserManager, s.ClientIdentityRepo, s.UserEmailer, s.localConnectorID)
	handler := NewUserMgmtServer(usersAPI, s.JWTVerifierFactory(), s.UserManager, s.ClientIdentityRepo).HTTPHandler()
	path := path.Join(apiBasePath, UsersSubTree)

	mux.Handle(path, handler)
	mux.Handle(path+"/", handler)

	return http.Handler(mux)
}

// NewClientTokenAuthHandler returns the given handler wrapped in middleware which requires a Client Bearer token.
func (s *Server) NewClientTokenAuthHandler(handler http.Handler) http.Handler {
	return &clientTokenMiddleware{
		issuerURL: s.IssuerURL.String(),
		ciRepo:    s.ClientIdentityRepo,
		keysFunc:  s.KeyManager.PublicKeys,
		next:      handler,
	}
}

func (s *Server) ClientMetadata(clientID string) (*oidc.ClientMetadata, error) {
	return s.ClientIdentityRepo.Metadata(clientID)
}

func (s *Server) NewSession(ipdcID, clientID, clientState string, redirectURL url.URL, nonce string, register bool, scope []string) (string, error) {
	sessionID, err := s.SessionManager.NewSession(ipdcID, clientID, clientState, redirectURL, nonce, register, scope)
	if err != nil {
		return "", err
	}

	log.Infof("Session %s created: clientID=%s clientState=%s", sessionID, clientID, clientState)
	return s.SessionManager.NewSessionKey(sessionID)
}

func (s *Server) Login(ident oidc.Identity, key string) (string, error) {
	sessionID, err := s.SessionManager.ExchangeKey(key)
	if err != nil {
		return "", err
	}

	ses, err := s.SessionManager.AttachRemoteIdentity(sessionID, ident)
	if err != nil {
		return "", err
	}
	log.Infof("Session %s remote identity attached: clientID=%s identity=%#v", sessionID, ses.ClientID, ident)

	if ses.Register {
		code, err := s.SessionManager.NewSessionKey(sessionID)
		if err != nil {
			return "", err
		}

		ru := s.absURL(httpPathRegister)
		q := ru.Query()
		q.Set("code", code)
		q.Set("state", ses.ClientState)
		ru.RawQuery = q.Encode()
		return ru.String(), nil
	}

	usr, err := s.UserRepo.GetByRemoteIdentity(nil, user.RemoteIdentity{
		ConnectorID: ses.ConnectorID,
		ID:          ses.Identity.ID,
	})
	if err != nil {
		if err == user.ErrorNotFound {
			// If has authenticated via a connector, but no local identity there
			// are a couple of possibilities:

			// * Maybe they are using the wrong connector:
			if ses.Identity.Email != "" {
				if connID, err := getConnectorForUserByEmail(s.UserRepo,
					ses.Identity.Email); err == nil {

					return newLoginURLFromSession(s.IssuerURL,
						ses, true, []string{connID},
						"wrong-connector").String(), nil
				}
			}

			// * User needs to register
			return newLoginURLFromSession(s.IssuerURL,
				ses, true, []string{ses.ConnectorID}, "register-maybe").String(), nil
		}
		return "", err
	}

	if usr.Disabled {
		return "", user.ErrorNotFound
	}

	ses, err = s.SessionManager.AttachUser(sessionID, usr.ID)
	if err != nil {
		return "", err
	}
	log.Infof("Session %s user identified: clientID=%s user=%#v", sessionID, ses.ClientID, usr)

	code, err := s.SessionManager.NewSessionKey(sessionID)
	if err != nil {
		return "", err
	}

	ru := ses.RedirectURL
	q := ru.Query()
	q.Set("code", code)
	q.Set("state", ses.ClientState)
	ru.RawQuery = q.Encode()

	return ru.String(), nil
}

func (s *Server) ClientCredsToken(creds oidc.ClientCredentials) (*jose.JWT, error) {
	ok, err := s.ClientIdentityRepo.Authenticate(creds)
	if err != nil {
		log.Errorf("Failed fetching client %s from repo: %v", creds.ID, err)
		return nil, oauth2.NewError(oauth2.ErrorServerError)
	}
	if !ok {
		return nil, oauth2.NewError(oauth2.ErrorInvalidClient)
	}

	signer, err := s.KeyManager.Signer()
	if err != nil {
		log.Errorf("Failed to generate ID token: %v", err)
		return nil, oauth2.NewError(oauth2.ErrorServerError)
	}

	now := time.Now()
	exp := now.Add(s.SessionManager.ValidityWindow)
	claims := oidc.NewClaims(s.IssuerURL.String(), creds.ID, creds.ID, now, exp)
	claims.Add("name", creds.ID)

	jwt, err := jose.NewSignedJWT(claims, signer)
	if err != nil {
		log.Errorf("Failed to generate ID token: %v", err)
		return nil, oauth2.NewError(oauth2.ErrorServerError)
	}

	log.Infof("Client token sent: clientID=%s", creds.ID)

	return jwt, nil
}

func (s *Server) CodeToken(creds oidc.ClientCredentials, sessionKey string) (*jose.JWT, string, error) {
	ok, err := s.ClientIdentityRepo.Authenticate(creds)
	if err != nil {
		log.Errorf("Failed fetching client %s from repo: %v", creds.ID, err)
		return nil, "", oauth2.NewError(oauth2.ErrorServerError)
	}
	if !ok {
		log.Errorf("Failed to Authenticate client %s", creds.ID)
		return nil, "", oauth2.NewError(oauth2.ErrorInvalidClient)
	}

	sessionID, err := s.SessionManager.ExchangeKey(sessionKey)
	if err != nil {
		return nil, "", oauth2.NewError(oauth2.ErrorInvalidGrant)
	}

	ses, err := s.SessionManager.Kill(sessionID)
	if err != nil {
		return nil, "", oauth2.NewError(oauth2.ErrorInvalidRequest)
	}

	if ses.ClientID != creds.ID {
		return nil, "", oauth2.NewError(oauth2.ErrorInvalidGrant)
	}

	signer, err := s.KeyManager.Signer()
	if err != nil {
		log.Errorf("Failed to generate ID token: %v", err)
		return nil, "", oauth2.NewError(oauth2.ErrorServerError)
	}

	user, err := s.UserRepo.Get(nil, ses.UserID)
	if err != nil {
		log.Errorf("Failed to fetch user %q from repo: %v: ", ses.UserID, err)
		return nil, "", oauth2.NewError(oauth2.ErrorServerError)
	}

	claims := ses.Claims(s.IssuerURL.String())
	user.AddToClaims(claims)

	jwt, err := jose.NewSignedJWT(claims, signer)
	if err != nil {
		log.Errorf("Failed to generate ID token: %v", err)
		return nil, "", oauth2.NewError(oauth2.ErrorServerError)
	}

	// Generate refresh token when 'scope' contains 'offline_access'.
	var refreshToken string

	for _, scope := range ses.Scope {
		if scope == "offline_access" {
			log.Infof("Session %s requests offline access, will generate refresh token", sessionID)

			refreshToken, err = s.RefreshTokenRepo.Create(ses.UserID, creds.ID)
			switch err {
			case nil:
				break
			default:
				log.Errorf("Failed to generate refresh token: %v", err)
				return nil, "", oauth2.NewError(oauth2.ErrorServerError)
			}
			break
		}
	}

	log.Infof("Session %s token sent: clientID=%s", sessionID, creds.ID)
	return jwt, refreshToken, nil
}

func (s *Server) RefreshToken(creds oidc.ClientCredentials, token string) (*jose.JWT, error) {
	ok, err := s.ClientIdentityRepo.Authenticate(creds)
	if err != nil {
		log.Errorf("Failed fetching client %s from repo: %v", creds.ID, err)
		return nil, oauth2.NewError(oauth2.ErrorServerError)
	}
	if !ok {
		log.Errorf("Failed to Authenticate client %s", creds.ID)
		return nil, oauth2.NewError(oauth2.ErrorInvalidClient)
	}

	userID, err := s.RefreshTokenRepo.Verify(creds.ID, token)
	switch err {
	case nil:
		break
	case refresh.ErrorInvalidToken:
		return nil, oauth2.NewError(oauth2.ErrorInvalidRequest)
	case refresh.ErrorInvalidClientID:
		return nil, oauth2.NewError(oauth2.ErrorInvalidClient)
	default:
		return nil, oauth2.NewError(oauth2.ErrorServerError)
	}

	user, err := s.UserRepo.Get(nil, userID)
	if err != nil {
		// The error can be user.ErrorNotFound, but we are not deleting
		// user at this moment, so this shouldn't happen.
		log.Errorf("Failed to fetch user %q from repo: %v: ", userID, err)
		return nil, oauth2.NewError(oauth2.ErrorServerError)
	}

	signer, err := s.KeyManager.Signer()
	if err != nil {
		log.Errorf("Failed to refresh ID token: %v", err)
		return nil, oauth2.NewError(oauth2.ErrorServerError)
	}

	now := time.Now()
	expireAt := now.Add(session.DefaultSessionValidityWindow)

	claims := oidc.NewClaims(s.IssuerURL.String(), user.ID, creds.ID, now, expireAt)
	user.AddToClaims(claims)

	jwt, err := jose.NewSignedJWT(claims, signer)
	if err != nil {
		log.Errorf("Failed to generate ID token: %v", err)
		return nil, oauth2.NewError(oauth2.ErrorServerError)
	}

	log.Infof("New token sent: clientID=%s", creds.ID)

	return jwt, nil
}

func (s *Server) JWTVerifierFactory() JWTVerifierFactory {
	noop := func() error { return nil }

	keyFunc := func() []key.PublicKey {
		keys, err := s.KeyManager.PublicKeys()
		if err != nil {
			log.Errorf("error getting public keys from manager: %v", err)
			return []key.PublicKey{}
		}
		return keys
	}
	return func(clientID string) oidc.JWTVerifier {

		return oidc.NewJWTVerifier(s.IssuerURL.String(), clientID, noop, keyFunc)
	}
}

type sortableIDPCs []connector.Connector

func (s sortableIDPCs) Len() int {
	return len([]connector.Connector(s))
}

func (s sortableIDPCs) Less(i, j int) bool {
	idpcs := []connector.Connector(s)
	return idpcs[i].ID() < idpcs[j].ID()
}

func (s sortableIDPCs) Swap(i, j int) {
	idpcs := []connector.Connector(s)
	idpcs[i], idpcs[j] = idpcs[j], idpcs[i]
}
