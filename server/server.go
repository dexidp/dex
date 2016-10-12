package server

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oauth2"
	"github.com/coreos/go-oidc/oidc"
	"github.com/coreos/pkg/health"
	"github.com/go-gorp/gorp"
	"github.com/jonboulle/clockwork"

	"github.com/coreos/dex/client"
	clientmanager "github.com/coreos/dex/client/manager"
	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/refresh"
	"github.com/coreos/dex/scope"
	"github.com/coreos/dex/session"
	sessionmanager "github.com/coreos/dex/session/manager"
	"github.com/coreos/dex/user"
	usersapi "github.com/coreos/dex/user/api"
	useremail "github.com/coreos/dex/user/email"
	usermanager "github.com/coreos/dex/user/manager"
)

const (
	LoginPageTemplateName              = "login.html"
	RegisterTemplateName               = "register.html"
	CreateAccountTemplateName          = "create-account.html"
	VerifyEmailTemplateName            = "verify-email.html"
	SendResetPasswordEmailTemplateName = "send-reset-password.html"
	ResetPasswordTemplateName          = "reset-password.html"
	OOBTemplateName                    = "oob-template.html"
	EmailConfirmationSentTemplateName  = "email-confirmation-sent.html"
	APIVersion                         = "v1"
)

type OIDCServer interface {
	Client(string) (client.Client, error)
	NewSession(connectorID, clientID, clientState string, redirectURL url.URL, nonce string, register bool, scope []string) (string, error)
	Login(oidc.Identity, string) (string, error)

	// CodeToken exchanges a code for an ID token and a refresh token string on success.
	CodeToken(creds oidc.ClientCredentials, sessionKey string) (*jose.JWT, string, time.Time, error)

	ClientCredsToken(creds oidc.ClientCredentials) (*jose.JWT, time.Time, error)

	// RefreshToken takes a previously generated refresh token and returns a new ID token and new refresh token
	// if the token is valid.
	RefreshToken(creds oidc.ClientCredentials, scopes scope.Scopes, token string) (*jose.JWT, string, time.Time, error)

	KillSession(string) error

	CrossClientAuthAllowed(requestingClientID, authorizingClientID string) (bool, error)
}

type JWTVerifierFactory func(clientID string) oidc.JWTVerifier

type Server struct {
	IssuerURL url.URL

	Templates                      *template.Template
	LoginTemplate                  *template.Template
	RegisterTemplate               *template.Template
	CreateAccountTemplate          *template.Template
	VerifyEmailTemplate            *template.Template
	SendResetPasswordEmailTemplate *template.Template
	ResetPasswordTemplate          *template.Template
	OOBTemplate                    *template.Template
	EmailConfirmationSentTemplate  *template.Template

	HealthChecks []health.Checkable
	// TODO(ericchiang): Make this a map of ID to connector.
	Connectors []connector.Connector

	ClientRepo          client.ClientRepo
	ConnectorConfigRepo connector.ConnectorConfigRepo
	KeySetRepo          key.PrivateKeySetRepo
	RefreshTokenRepo    refresh.RefreshTokenRepo
	UserRepo            user.UserRepo
	PasswordInfoRepo    user.PasswordInfoRepo
	OrganizationRepo    user.OrganizationRepo

	ClientManager  *clientmanager.ClientManager
	KeyManager     key.PrivateKeyManager
	SessionManager *sessionmanager.SessionManager
	UserManager    *usermanager.UserManager

	UserEmailer *useremail.UserEmailer

	EnableRegistration           bool
	EnableClientRegistration     bool
	EnableClientCredentialAccess bool
	RegisterOnFirstLogin         bool

	dbMap            *gorp.DbMap
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
	authEndpoint := s.absURL(httpPathAuth)
	tokenEndpoint := s.absURL(httpPathToken)
	keysEndpoint := s.absURL(httpPathKeys)
	cfg := oidc.ProviderConfig{
		Issuer:        &s.IssuerURL,
		AuthEndpoint:  &authEndpoint,
		TokenEndpoint: &tokenEndpoint,
		KeysEndpoint:  &keysEndpoint,

		GrantTypesSupported:               []string{oauth2.GrantTypeAuthCode, oauth2.GrantTypeClientCreds},
		ResponseTypesSupported:            []string{"code"},
		SubjectTypesSupported:             []string{"public"},
		IDTokenSigningAlgValues:           []string{"RS256"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic"},
	}

	if s.EnableClientRegistration {
		regEndpoint := s.absURL(httpPathClientRegistration)
		cfg.RegistrationEndpoint = &regEndpoint
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

	// Handler methods which register handlers at prefixed paths.
	handle := func(urlPath string, h http.Handler) {
		p := path.Join(s.IssuerURL.Path, urlPath)
		// path.Join always trims trailing slashes (https://play.golang.org/p/GRr0jDd9P7).
		// If path being registered has a trailing slash, add it back on.
		if strings.HasSuffix(urlPath, "/") {
			p = p + "/"
		}
		mux.Handle(p, h)
	}
	handleFunc := func(urlPath string, hf http.HandlerFunc) {
		handle(urlPath, hf)
	}
	handleStripPrefix := func(urlPath string, h http.Handler) {
		if s.IssuerURL.Path != "" {
			handle(urlPath, http.StripPrefix(s.IssuerURL.Path, h))
		} else {
			handle(urlPath, h)
		}
	}

	handleFunc(httpPathDiscovery, handleDiscoveryFunc(s.ProviderConfig()))
	handleFunc(httpPathAuth, handleAuthFunc(s, s.IssuerURL, s.Connectors, s.LoginTemplate, s.EnableRegistration))
	handleFunc(httpPathOOB, handleOOBFunc(s, s.OOBTemplate))
	handleFunc(httpPathToken, handleTokenFunc(s))
	handleFunc(httpPathKeys, handleKeysFunc(s.KeyManager, clock))
	handle(httpPathHealth, makeHealthHandler(checks))

	if s.EnableRegistration {
		handleFunc(httpPathRegister, handleRegisterFunc(s, s.RegisterTemplate))
	}

	handleFunc(httpPathCreateAccount, handleCreateAccountFunc(s, s.CreateAccountTemplate))

	handleFunc(httpPathEmailVerify, handleEmailVerifyFunc(s.VerifyEmailTemplate,
		s.IssuerURL, s.KeyManager.PublicKeys, s.UserManager))

	handle(httpPathVerifyEmailResend, s.NewClientTokenAuthHandler(handleVerifyEmailResendFunc(s.IssuerURL,
		s.KeyManager.PublicKeys,
		s.UserEmailer,
		s.UserRepo,
		s.ClientManager)))

	handle(httpPathSendResetPassword, &SendResetPasswordEmailHandler{
		tpl:     s.SendResetPasswordEmailTemplate,
		emailer: s.UserEmailer,
		sm:      s.SessionManager,
		cm:      s.ClientManager,
	})

	handle(httpPathResetPassword, &ResetPasswordHandler{
		tpl:       s.ResetPasswordTemplate,
		issuerURL: s.IssuerURL,
		um:        s.UserManager,
		keysFunc:  s.KeyManager.PublicKeys,
	})

	handle(httpPathAcceptInvitation, &InvitationHandler{
		passwordResetURL:       s.absURL(httpPathResetPassword),
		issuerURL:              s.IssuerURL,
		um:                     s.UserManager,
		keysFunc:               s.KeyManager.PublicKeys,
		signerFunc:             s.KeyManager.Signer,
		redirectValidityWindow: s.SessionManager.ValidityWindow,
	})

	if s.EnableClientRegistration {
		handleFunc(httpPathClientRegistration, s.handleClientRegistration)
	}

	handleFunc(httpPathDebugVars, health.ExpvarHandler)

	pcfg := s.ProviderConfig()
	for _, idpc := range s.Connectors {
		errorURL, err := url.Parse(fmt.Sprintf("%s?connector_id=%s", pcfg.AuthEndpoint, idpc.ID()))
		if err != nil {
			log.Fatal(err)
		}
		// NOTE(ericchiang): This path MUST end in a "/" in order to indicate a
		// path prefix rather than an absolute path.
		handle(path.Join(httpPathAuth, idpc.ID())+"/", idpc.Handler(*errorURL))
	}

	apiBasePath := path.Join(httpPathAPI, APIVersion)
	registerDiscoveryResource(apiBasePath, mux)

	usersAPI := usersapi.NewUsersAPI(s.UserManager, s.ClientManager, s.RefreshTokenRepo, s.UserEmailer, s.localConnectorID, s.EnableClientCredentialAccess)
	handler := NewUserMgmtServer(usersAPI, s.JWTVerifierFactory(), s.UserManager, s.ClientManager, s.EnableClientCredentialAccess).HTTPHandler()

	handleStripPrefix(apiBasePath+"/", handler)

	return http.Handler(mux)
}

// NewClientTokenAuthHandler returns the given handler wrapped in middleware which requires a Client Bearer token.
func (s *Server) NewClientTokenAuthHandler(handler http.Handler) http.Handler {
	return &clientTokenMiddleware{
		issuerURL: s.IssuerURL.String(),
		ciManager: s.ClientManager,
		keysFunc:  s.KeyManager.PublicKeys,
		next:      handler,
	}
}

func (s *Server) Client(clientID string) (client.Client, error) {
	return s.ClientManager.Get(clientID)
}

func (s *Server) NewSession(ipdcID, clientID, clientState string, redirectURL url.URL, nonce string, register bool, scope []string) (string, error) {
	sessionID, err := s.SessionManager.NewSession(ipdcID, clientID, clientState, redirectURL, nonce, register, scope)
	if err != nil {
		return "", err
	}

	log.Infof("Session %s created: clientID=%s clientState=%s", sessionID, clientID, clientState)
	return s.SessionManager.NewSessionKey(sessionID)
}

func (s *Server) connector(id string) (connector.Connector, bool) {
	for _, c := range s.Connectors {
		if c.ID() == id {
			return c, true
		}
	}
	return nil, false
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

	// Get the connector used to log the user in.
	conn, ok := s.connector(ses.ConnectorID)
	if !ok {
		return "", fmt.Errorf("session contained invalid connector ID (%s)", ses.ConnectorID)
	}

	// If the client has requested access to groups, add them here.
	if ses.Scope.HasScope(scope.ScopeGroups) {
		grouper, ok := conn.(connector.GroupsConnector)
		if !ok {
			return "", fmt.Errorf("scope %q provided but connector does not support groups", scope.ScopeGroups)
		}
		groups, err := grouper.Groups(ident.ID)
		if err != nil {
			return "", fmt.Errorf("failed to retrieve user groups for %q %v", ident.ID, err)
		}

		// Update the session.
		if ses, err = s.SessionManager.AttachGroups(sessionID, groups); err != nil {
			return "", fmt.Errorf("failed save groups")
		}
	}

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

	remoteIdentity := user.RemoteIdentity{ConnectorID: ses.ConnectorID, ID: ses.Identity.ID}

	usr, err := s.UserRepo.GetByRemoteIdentity(nil, remoteIdentity)
	if err == user.ErrorNotFound {
		if ses.Identity.Email == "" {
			// User doesn't have an existing account. Ask them to register.
			u := newLoginURLFromSession(s.IssuerURL, ses, true, []string{ses.ConnectorID}, "register-maybe")
			return u.String(), nil
		}

		// Does the user have an existing account with a different connector?
		if connID, err := getConnectorForUserByEmail(s.UserRepo, ses.Identity.Email); err == nil {
			// Ask user to sign in through existing account.
			u := newLoginURLFromSession(s.IssuerURL, ses, false, []string{connID}, "wrong-connector")
			return u.String(), nil
		}

		// RegisterOnFirstLogin doesn't work for the local connector
		tryToRegister := s.RegisterOnFirstLogin && (ses.ConnectorID != s.localConnectorID)

		if !tryToRegister {
			// User doesn't have an existing account. Ask them to register.
			u := newLoginURLFromSession(s.IssuerURL, ses, true, []string{ses.ConnectorID}, "register-maybe")
			return u.String(), nil
		}

		// First time logging in through a remote connector. Attempt to register.
		emailVerified := conn.TrustedEmailProvider()
		usrID, err := s.UserManager.RegisterWithRemoteIdentity(ses.Identity.Email, emailVerified, remoteIdentity)
		if err != nil {
			return "", fmt.Errorf("failed to register user: %v", err)
		}
		usr, err = s.UserManager.Get(usrID)
		if err != nil {
			return "", fmt.Errorf("getting created user: %v", err)
		}

		if ses.Identity.Name != "" && ses.Identity.Name != usr.DisplayName {
			err = s.UserManager.SetDisplayName(usr, ses.Identity.Name)
			if err != nil {
				return "", fmt.Errorf("couldn't set display name for user: %v", err)
			}
		}
	} else if err != nil {
		return "", fmt.Errorf("getting user: %v", err)
	}

	if usr.Disabled {
		log.Errorf("user %s disabled", ses.Identity.Email)
		return "", user.ErrorNotFound
	}

	ses, err = s.SessionManager.AttachUser(sessionID, usr.ID)
	if err != nil {
		return "", fmt.Errorf("attaching user to session: %v", err)
	}
	log.Infof("Session %s user identified: clientID=%s user=%#v", sessionID, ses.ClientID, usr)

	code, err := s.SessionManager.NewSessionKey(sessionID)
	if err != nil {
		return "", fmt.Errorf("creating new session key: %v", err)
	}

	ru := ses.RedirectURL
	if ru.String() == client.OOBRedirectURI {
		ru = s.absURL(httpPathOOB)
	}
	q := ru.Query()
	q.Set("code", code)
	q.Set("state", ses.ClientState)
	ru.RawQuery = q.Encode()

	return ru.String(), nil
}

func (s *Server) ClientCredsToken(creds oidc.ClientCredentials) (*jose.JWT, time.Time, error) {
	cli, err := s.Client(creds.ID)
	if err != nil {
		return nil, time.Time{}, err
	}

	if cli.Public {
		return nil, time.Time{}, oauth2.NewError(oauth2.ErrorInvalidClient)
	}

	ok, err := s.ClientManager.Authenticate(creds)
	if err != nil {
		log.Errorf("Failed fetching client %s from manager: %v", creds.ID, err)
		return nil, time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
	}
	if !ok {
		return nil, time.Time{}, oauth2.NewError(oauth2.ErrorInvalidClient)
	}

	signer, err := s.KeyManager.Signer()
	if err != nil {
		log.Errorf("Failed to generate ID token: %v", err)
		return nil, time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
	}

	now := time.Now()
	exp := now.Add(s.SessionManager.ValidityWindow)
	claims := oidc.NewClaims(s.IssuerURL.String(), creds.ID, creds.ID, now, exp)
	claims.Add("name", creds.ID)

	jwt, err := jose.NewSignedJWT(claims, signer)
	if err != nil {
		log.Errorf("Failed to generate ID token: %v", err)
		return nil, time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
	}

	log.Infof("Client token sent: clientID=%s", creds.ID)

	return jwt, exp, nil
}

func (s *Server) CodeToken(creds oidc.ClientCredentials, sessionKey string) (*jose.JWT, string, time.Time, error) {
	ok, err := s.ClientManager.Authenticate(creds)
	if err != nil {
		log.Errorf("Failed fetching client %s from repo: %v", creds.ID, err)
		return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
	}
	if !ok {
		log.Errorf("Failed to Authenticate client %s", creds.ID)
		return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorInvalidClient)
	}

	sessionID, err := s.SessionManager.ExchangeKey(sessionKey)
	if err != nil {
		return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorInvalidGrant)
	}

	ses, err := s.SessionManager.Kill(sessionID)
	if err != nil {
		return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorInvalidRequest)
	}

	if ses.ClientID != creds.ID {
		return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorInvalidGrant)
	}

	signer, err := s.KeyManager.Signer()
	if err != nil {
		log.Errorf("Failed to generate ID token: %v", err)
		return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
	}

	user, err := s.UserRepo.Get(nil, ses.UserID)
	if err != nil {
		log.Errorf("Failed to fetch user %q from repo: %v: ", ses.UserID, err)
		return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
	}

	claims := ses.Claims(s.IssuerURL.String())
	user.AddToClaims(claims)

	s.addClaimsFromScope(claims, ses.Scope, ses.ClientID)

	jwt, err := jose.NewSignedJWT(claims, signer)
	if err != nil {
		log.Errorf("Failed to generate ID token: %v", err)
		return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
	}

	// Generate refresh token when 'scope' contains 'offline_access'.
	var refreshToken string

	for _, scope := range ses.Scope {
		if scope == "offline_access" {
			log.Infof("Session %s requests offline access, will generate refresh token", sessionID)

			refreshToken, err = s.RefreshTokenRepo.Create(ses.UserID, creds.ID, ses.ConnectorID, ses.Scope)
			switch err {
			case nil:
				break
			default:
				log.Errorf("Failed to generate refresh token: %v", err)
				return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
			}
			break
		}
	}

	log.Infof("Session %s token sent: clientID=%s", sessionID, creds.ID)
	return jwt, refreshToken, ses.ExpiresAt, nil
}

func (s *Server) RefreshToken(creds oidc.ClientCredentials, scopes scope.Scopes, token string) (*jose.JWT, string, time.Time, error) {
	ok, err := s.ClientManager.Authenticate(creds)
	if err != nil {
		log.Errorf("Failed fetching client %s from repo: %v", creds.ID, err)
		return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
	}
	if !ok {
		log.Errorf("Failed to Authenticate client %s", creds.ID)
		return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorInvalidClient)
	}

	userID, connectorID, rtScopes, err := s.RefreshTokenRepo.Verify(creds.ID, token)
	switch err {
	case nil:
		break
	case refresh.ErrorInvalidToken:
		return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorInvalidRequest)
	case refresh.ErrorInvalidClientID:
		return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorInvalidClient)
	default:
		return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
	}

	if len(scopes) == 0 {
		scopes = rtScopes
	} else {
		if !rtScopes.Contains(scopes) {
			return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorInvalidRequest)
		}
	}

	usr, err := s.UserRepo.Get(nil, userID)
	if err != nil {
		// The error can be user.ErrorNotFound, but we are not deleting
		// user at this moment, so this shouldn't happen.
		log.Errorf("Failed to fetch user %q from repo: %v: ", userID, err)
		return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
	}

	var groups []string
	if rtScopes.HasScope(scope.ScopeGroups) {
		conn, ok := s.connector(connectorID)
		if !ok {
			log.Errorf("refresh token contained invalid connector ID (%s)", connectorID)
			return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
		}

		grouper, ok := conn.(connector.GroupsConnector)
		if !ok {
			log.Errorf("refresh token requested groups for connector (%s) that doesn't support groups", connectorID)
			return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
		}

		remoteIdentities, err := s.UserRepo.GetRemoteIdentities(nil, userID)
		if err != nil {
			log.Errorf("failed to get remote identities: %v", err)
			return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
		}
		remoteIdentity, ok := func() (user.RemoteIdentity, bool) {
			for _, ri := range remoteIdentities {
				if ri.ConnectorID == connectorID {
					return ri, true
				}
			}
			return user.RemoteIdentity{}, false
		}()
		if !ok {
			log.Errorf("failed to get remote identity for connector %s", connectorID)
			return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
		}
		if groups, err = grouper.Groups(remoteIdentity.ID); err != nil {
			log.Errorf("failed to get groups for refresh token: %v", connectorID)
			return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
		}
	}

	signer, err := s.KeyManager.Signer()
	if err != nil {
		log.Errorf("Failed to refresh ID token: %v", err)
		return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
	}

	now := time.Now()
	expiresAt := now.Add(session.DefaultSessionValidityWindow)

	claims := oidc.NewClaims(s.IssuerURL.String(), usr.ID, creds.ID, now, expiresAt)
	usr.AddToClaims(claims)
	if rtScopes.HasScope(scope.ScopeGroups) {
		if groups == nil {
			groups = []string{}
		}
		claims["groups"] = groups
	}

	s.addClaimsFromScope(claims, scope.Scopes(scopes), creds.ID)

	jwt, err := jose.NewSignedJWT(claims, signer)
	if err != nil {
		log.Errorf("Failed to generate ID token: %v", err)
		return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
	}

	refreshToken, err := s.RefreshTokenRepo.RenewRefreshToken(creds.ID, userID, token)
	if err != nil {
		log.Errorf("Failed to generate new refresh token: %v", err)
		return nil, "", time.Time{}, oauth2.NewError(oauth2.ErrorServerError)
	}

	log.Infof("New token sent: clientID=%s", creds.ID)

	return jwt, refreshToken, expiresAt, nil
}

func (s *Server) CrossClientAuthAllowed(requestingClientID, authorizingClientID string) (bool, error) {
	alloweds, err := s.ClientRepo.GetTrustedPeers(nil, authorizingClientID)
	if err != nil {
		return false, err
	}
	for _, allowed := range alloweds {
		if requestingClientID == allowed {
			return true, nil
		}
	}
	return false, nil
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

// addClaimsFromScope adds claims that are based on the scopes that the client requested.
// Currently, these include cross-client claims (aud, azp).
func (s *Server) addClaimsFromScope(claims jose.Claims, scopes scope.Scopes, clientID string) error {
	crossClientIDs := scopes.CrossClientIDs()
	if len(crossClientIDs) > 0 {
		var aud []string
		for _, id := range crossClientIDs {
			if clientID == id {
				aud = append(aud, id)
				continue
			}
			allowed, err := s.CrossClientAuthAllowed(clientID, id)
			if err != nil {
				log.Errorf("Failed to check cross client auth. reqClientID %v; authClient:ID %v; err: %v", clientID, id, err)
				return oauth2.NewError(oauth2.ErrorServerError)
			}
			if !allowed {
				err := oauth2.NewError(oauth2.ErrorInvalidRequest)
				err.Description = fmt.Sprintf(
					"%q is not authorized to perform cross-client requests for %q",
					clientID, id)
				return err
			}
			aud = append(aud, id)
		}
		if len(aud) == 1 {
			claims.Add("aud", aud[0])
		} else {
			claims.Add("aud", aud)
		}
		claims.Add("azp", clientID)
	}
	return nil
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
