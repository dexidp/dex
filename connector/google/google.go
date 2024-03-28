// Package google implements logging in through Google's OpenID Connect provider.
package google

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/api/impersonate"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/exp/slices"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/option"

	"github.com/dexidp/dex/connector"
	pkg_groups "github.com/dexidp/dex/pkg/groups"
	"github.com/dexidp/dex/pkg/log"
)

const (
	issuerURL                  = "https://accounts.google.com"
	wildcardDomainToAdminEmail = "*"
)

// Config holds configuration options for Google logins.
type Config struct {
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	RedirectURI  string `json:"redirectURI"`

	Scopes []string `json:"scopes"` // defaults to "profile" and "email"

	// Optional list of whitelisted domains
	// If this field is nonempty, only users from a listed domain will be allowed to log in
	HostedDomains []string `json:"hostedDomains"`

	// Optional list of whitelisted groups
	// If this field is nonempty, only users from a listed group will be allowed to log in
	Groups []string `json:"groups"`

	// Optional path to service account json
	// If nonempty, and groups claim is made, will use authentication from file to
	// check groups with the admin directory api
	ServiceAccountFilePath string `json:"serviceAccountFilePath"`

	// Deprecated: Use DomainToAdminEmail
	AdminEmail string

	// Required if ServiceAccountFilePath
	// The map workspace domain to email of a GSuite super user which the service account will impersonate
	// when listing groups
	DomainToAdminEmail map[string]string

	// Optional email of the service account enabled for domain-wide delegation
	// If nonempty, it is assumed that Workload Identity Federation is to be used. In that case, the
	// specified service account needs to be configured for domain-wide delegation and the service account
	// used for Workload Identity Federation must include "Service Account Token Creator" for the specified
	// service account.
	ServiceAccountToImpersonate string `json:"serviceAccountToImpersonate"`

	// If this field is true, fetch direct group membership and transitive group membership
	FetchTransitiveGroupMembership bool `json:"fetchTransitiveGroupMembership"`
}

func validateConfigAndPrintResult(c *Config, logger log.Logger) error {
	if slices.Contains(c.Scopes, "groups") {
		logger.Warnf("\"scopes\" contain \"groups\" which is not supported by Google. This will result in an error on request.")
	}

	if len(c.Groups) > 0 && len(c.DomainToAdminEmail) == 0 {
		logger.Warnf("\"groups\" is specified in the configuration, but no Google service account has been configured to be used. \"groups\" will be ignored.")
	}

	// We know impersonation is required when using a service account credential
	// TODO: or is it?
	if len(c.DomainToAdminEmail) == 0 && (c.ServiceAccountFilePath != "" || c.ServiceAccountToImpersonate != "") {
		return fmt.Errorf("directory service requires the domainToAdminEmail option to be configured")
	}

	return nil
}

// Open returns a connector which can be used to login users through Google.
func (c *Config) Open(id string, logger log.Logger) (conn connector.Connector, err error) {
	if c.AdminEmail != "" {
		log.Deprecated(logger, `google: use "domainToAdminEmail.*: %s" option instead of "adminEmail: %s".`, c.AdminEmail, c.AdminEmail)
		if c.DomainToAdminEmail == nil {
			c.DomainToAdminEmail = make(map[string]string)
		}

		c.DomainToAdminEmail[wildcardDomainToAdminEmail] = c.AdminEmail
	}
	ctx, cancel := context.WithCancel(context.Background())

	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to get provider: %v", err)
	}

	scopes := []string{oidc.ScopeOpenID}
	if len(c.Scopes) > 0 {
		scopes = append(scopes, c.Scopes...)
	} else {
		scopes = append(scopes, "profile", "email")
	}

	adminSrv := make(map[string]*admin.Service)

	if err := validateConfigAndPrintResult(c, logger); err != nil {
		cancel()
		return nil, err
	}

	// Fixing a regression caused by default config fallback: https://github.com/dexidp/dex/issues/2699
	// TODO: if scopes contain "group", the oauth request to Google will fail with "Some requested scopes were invalid ... invalid=[groups]"
	if ((c.ServiceAccountFilePath != "" || c.ServiceAccountToImpersonate != "") && len(c.DomainToAdminEmail) > 0) || slices.Contains(scopes, "groups") {
		logger.Debug("Directory service will be configured.")

		for domain, adminEmail := range c.DomainToAdminEmail {
			var srv *admin.Service
			var err error

			if c.ServiceAccountToImpersonate == "" {
				logger.Debugf("Using Service Account Key for domain '%s' impersonating '%s'.", domain, adminEmail)
				srv, err = createDirectoryService(c.ServiceAccountFilePath, adminEmail, logger)
			} else {
				logger.Debugf("Using Workload Identity Federation with SA '%s' for domain '%s' impersonating '%s'.", c.ServiceAccountToImpersonate, domain, adminEmail)
				srv, err = createDirectoryServiceForWorkloadIdentityFederation(c.ServiceAccountFilePath, c.ServiceAccountToImpersonate, adminEmail, logger)
			}
			if err != nil {
				cancel()
				return nil, fmt.Errorf("could not create directory service: %v", err)
			}

			adminSrv[domain] = srv
		}
	} else {
		logger.Debug("Directory service will not be configured.")
	}

	clientID := c.ClientID
	return &googleConnector{
		redirectURI: c.RedirectURI,
		oauth2Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: c.ClientSecret,
			Endpoint:     provider.Endpoint(),
			Scopes:       scopes,
			RedirectURL:  c.RedirectURI,
		},
		verifier: provider.Verifier(
			&oidc.Config{ClientID: clientID},
		),
		logger:                         logger,
		cancel:                         cancel,
		hostedDomains:                  c.HostedDomains,
		groups:                         c.Groups,
		serviceAccountFilePath:         c.ServiceAccountFilePath,
		domainToAdminEmail:             c.DomainToAdminEmail,
		serviceAccountToImpersonate:    c.ServiceAccountToImpersonate,
		fetchTransitiveGroupMembership: c.FetchTransitiveGroupMembership,
		adminSrv:                       adminSrv,
	}, nil
}

var (
	_ connector.CallbackConnector = (*googleConnector)(nil)
	_ connector.RefreshConnector  = (*googleConnector)(nil)
)

type googleConnector struct {
	redirectURI                    string
	oauth2Config                   *oauth2.Config
	verifier                       *oidc.IDTokenVerifier
	cancel                         context.CancelFunc
	logger                         log.Logger
	hostedDomains                  []string
	groups                         []string
	serviceAccountFilePath         string
	domainToAdminEmail             map[string]string
	serviceAccountToImpersonate    string
	fetchTransitiveGroupMembership bool
	adminSrv                       map[string]*admin.Service
}

func (c *googleConnector) Close() error {
	c.cancel()
	return nil
}

func (c *googleConnector) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
	if c.redirectURI != callbackURL {
		return "", fmt.Errorf("expected callback URL %q did not match the URL in the config %q", callbackURL, c.redirectURI)
	}

	var opts []oauth2.AuthCodeOption
	if len(c.hostedDomains) > 0 {
		preferredDomain := c.hostedDomains[0]
		if len(c.hostedDomains) > 1 {
			preferredDomain = "*"
		}
		opts = append(opts, oauth2.SetAuthURLParam("hd", preferredDomain))
	}

	if s.OfflineAccess {
		opts = append(opts, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
	}
	return c.oauth2Config.AuthCodeURL(state, opts...), nil
}

type oauth2Error struct {
	error            string
	errorDescription string
}

func (e *oauth2Error) Error() string {
	if e.errorDescription == "" {
		return e.error
	}
	return e.error + ": " + e.errorDescription
}

func (c *googleConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return identity, &oauth2Error{errType, q.Get("error_description")}
	}
	token, err := c.oauth2Config.Exchange(r.Context(), q.Get("code"))
	if err != nil {
		return identity, fmt.Errorf("google: failed to get token: %v", err)
	}

	return c.createIdentity(r.Context(), identity, token)
}

func (c *googleConnector) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	t := &oauth2.Token{
		RefreshToken: string(identity.ConnectorData),
		Expiry:       time.Now().Add(-time.Hour),
	}
	token, err := c.oauth2Config.TokenSource(ctx, t).Token()
	if err != nil {
		return identity, fmt.Errorf("google: failed to get token: %v", err)
	}

	return c.createIdentity(ctx, identity, token)
}

func (c *googleConnector) createIdentity(ctx context.Context, identity connector.Identity, token *oauth2.Token) (connector.Identity, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return identity, errors.New("google: no id_token in token response")
	}
	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return identity, fmt.Errorf("google: failed to verify ID Token: %v", err)
	}

	var claims struct {
		Username      string `json:"name"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		HostedDomain  string `json:"hd"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return identity, fmt.Errorf("oidc: failed to decode claims: %v", err)
	}

	if len(c.hostedDomains) > 0 {
		found := false
		for _, domain := range c.hostedDomains {
			if claims.HostedDomain == domain {
				found = true
				break
			}
		}

		if !found {
			return identity, fmt.Errorf("oidc: unexpected hd claim %v", claims.HostedDomain)
		}
	}

	var groups []string
	if len(c.adminSrv) > 0 {
		checkedGroups := make(map[string]struct{})
		groups, err = c.getGroups(claims.Email, c.fetchTransitiveGroupMembership, checkedGroups)
		if err != nil {
			return identity, fmt.Errorf("google: could not retrieve groups: %v", err)
		}

		if len(c.groups) > 0 {
			groups = pkg_groups.Filter(groups, c.groups)
			if len(groups) == 0 {
				return identity, fmt.Errorf("google: user %q is not in any of the required groups", claims.Username)
			}
		}
	}

	identity = connector.Identity{
		UserID:        idToken.Subject,
		Username:      claims.Username,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		ConnectorData: []byte(token.RefreshToken),
		Groups:        groups,
	}
	return identity, nil
}

// getGroups creates a connection to the admin directory service and lists
// all groups the user is a member of
func (c *googleConnector) getGroups(email string, fetchTransitiveGroupMembership bool, checkedGroups map[string]struct{}) ([]string, error) {
	var userGroups []string
	var err error
	groupsList := &admin.Groups{}
	domain := c.extractDomainFromEmail(email)
	adminSrv, err := c.findAdminService(domain)
	if err != nil {
		return nil, err
	}

	for {
		groupsList, err = adminSrv.Groups.List().
			UserKey(email).PageToken(groupsList.NextPageToken).Do()
		if err != nil {
			return nil, fmt.Errorf("could not list groups: %v", err)
		}

		for _, group := range groupsList.Groups {
			if _, exists := checkedGroups[group.Email]; exists {
				continue
			}

			checkedGroups[group.Email] = struct{}{}
			// TODO (joelspeed): Make desired group key configurable
			userGroups = append(userGroups, group.Email)

			if !fetchTransitiveGroupMembership {
				continue
			}

			// getGroups takes a user's email/alias as well as a group's email/alias
			transitiveGroups, err := c.getGroups(group.Email, fetchTransitiveGroupMembership, checkedGroups)
			if err != nil {
				return nil, fmt.Errorf("could not list transitive groups: %v", err)
			}

			userGroups = append(userGroups, transitiveGroups...)
		}

		if groupsList.NextPageToken == "" {
			break
		}
	}

	return userGroups, nil
}

func (c *googleConnector) findAdminService(domain string) (*admin.Service, error) {
	adminSrv, ok := c.adminSrv[domain]
	if !ok {
		adminSrv, ok = c.adminSrv[wildcardDomainToAdminEmail]
		c.logger.Debugf("using wildcard (%s) admin email to fetch groups", c.domainToAdminEmail[wildcardDomainToAdminEmail])
	}

	if !ok {
		return nil, fmt.Errorf("unable to find super admin email, domainToAdminEmail for domain: %s not set, %s is also empty", domain, wildcardDomainToAdminEmail)
	}

	return adminSrv, nil
}

// extracts the domain name from an email input. If the email is valid, it returns the domain name after the "@" symbol.
// However, in the case of a broken or invalid email, it returns a wildcard symbol.
func (c *googleConnector) extractDomainFromEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at >= 0 {
		_, domain := email[:at], email[at+1:]

		return domain
	}

	return wildcardDomainToAdminEmail
}

// createDirectoryService sets up super user impersonation and creates an admin client for calling
// the google admin api. If no serviceAccountFilePath is defined, the application default credential
// is used.
func createDirectoryService(serviceAccountFilePath, email string, logger log.Logger) (*admin.Service, error) {
	var jsonCredentials []byte
	var err error

	ctx := context.Background()
	if serviceAccountFilePath == "" {
		logger.Warn("the application default credential is used since the service account file path is not used")
		credential, err := google.FindDefaultCredentials(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch application default credentials: %w", err)
		}
		jsonCredentials = credential.JSON
	} else {
		jsonCredentials, err = os.ReadFile(serviceAccountFilePath)
		if err != nil {
			return nil, fmt.Errorf("error reading credentials from file: %v", err)
		}
	}
	config, err := google.JWTConfigFromJSON(jsonCredentials, admin.AdminDirectoryGroupReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	// Only attempt impersonation when there is a user configured
	if email != "" {
		config.Subject = email
	}

	return admin.NewService(ctx, option.WithHTTPClient(config.Client(ctx)))
}

func createDirectoryServiceForWorkloadIdentityFederation(serviceAccountFilePath, serviceAccountWithDWD, email string, logger log.Logger) (*admin.Service, error) {
	var err error
	var ts oauth2.TokenSource

	ctx := context.Background()
	var config = impersonate.CredentialsConfig{
		Subject:         email,
		Scopes:          []string{admin.AdminDirectoryGroupReadonlyScope},
		TargetPrincipal: serviceAccountWithDWD,
	}

	if serviceAccountFilePath == "" {
		logger.Debug("Using application default credentials.")
		ts, err = impersonate.CredentialsTokenSource(ctx, config)
	} else {
		logger.Debugf("Using credentials at '%s'.", serviceAccountFilePath)
		var jsonCredentials []byte
		jsonCredentials, err = os.ReadFile(serviceAccountFilePath)
		if err != nil {
			return nil, fmt.Errorf("error reading credentials from file: %v", err)
		}
		ts, err = impersonate.CredentialsTokenSource(ctx, config, option.WithCredentialsJSON(jsonCredentials))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create : %v", err)
	}

	return admin.NewService(ctx, option.WithTokenSource(ts))
}
