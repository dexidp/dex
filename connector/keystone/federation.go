package keystone

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dexidp/dex/connector"
)

// FederationConnector implements the connector interface for Keystone federation authentication
type FederationConnector struct {
	cfg    FederationConfig
	client *http.Client
	logger *slog.Logger
}

var (
	_ connector.CallbackConnector = &FederationConnector{}
	_ connector.RefreshConnector  = &FederationConnector{}
)

// Validate returns error if config is invalid.
func (c *FederationConfig) Validate() error {
	var missing []string

	if c.Domain == "" {
		missing = append(missing, "domain")
	}
	if c.Host == "" {
		missing = append(missing, "host")
	}
	if c.AdminUsername == "" {
		missing = append(missing, "keystoneUsername")
	}
	if c.AdminPassword == "" {
		missing = append(missing, "keystonePassword")
	}
	if c.CustomerName == "" {
		missing = append(missing, "customerName")
	}
	if c.ShibbolethLoginPath == "" {
		missing = append(missing, "shibbolethLoginPath")
	}
	if c.FederationAuthPath == "" {
		missing = append(missing, "federationAuthPath")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required fields in config: %s", strings.Join(missing, ", "))
	}
	return nil
}

// Open returns a connector using the federation configuration
func (c *FederationConfig) Open(id string, logger *slog.Logger) (connector.Connector, error) {
	return NewFederationConnector(*c, logger)
}

func NewFederationConnector(cfg FederationConfig, logger *slog.Logger) (*FederationConnector, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &FederationConnector{
		cfg: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
		},
		logger: logger,
	}, nil
}

func (c *FederationConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, error) {
	baseURL := strings.TrimSuffix(c.cfg.Host, "/")
	baseURL = strings.TrimSuffix(baseURL, "/keystone")
	ssoLoginPath := strings.TrimPrefix(c.cfg.ShibbolethLoginPath, "/")

	u, err := url.Parse(fmt.Sprintf("%s/%s", baseURL, ssoLoginPath))
	if err != nil {
		return "", fmt.Errorf("parsing SSO login URL: %w", err)
	}

	// The target will be passed through the entire federation flow.
	// target is nothing but the redirect url that will be used by shibboleth to redirect back to Dex.
	target := fmt.Sprintf("%s?state=%s", callbackURL, state)
	q := u.Query()
	q.Set("target", target)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (c *FederationConnector) HandleCallback(scopes connector.Scopes, r *http.Request) (connector.Identity, error) {
	c.logger.Debug("dex callback received", "method", r.Method)

	var ksToken string
	var err error
	var tokenInfo *tokenInfo
	identity := connector.Identity{}

	ksToken, err = c.getKeystoneTokenFromFederation(r)
	if err != nil {
		c.logger.Error("failed to get token from federation cookies", "error", err)
		return connector.Identity{}, err
	}
	c.logger.Debug("successfully obtained token from federation cookies")

	c.logger.Debug("retrieving user info")
	tokenInfo, err = getTokenInfo(r.Context(), c.client, c.cfg.Host, ksToken, c.logger)
	if err != nil {
		return connector.Identity{}, err
	}
	if scopes.Groups {
		c.logger.Debug("groups scope requested, fetching groups")
		var err error
		adminToken, err := getAdminTokenUnscoped(r.Context(), c.client, c.cfg.Host, c.cfg.AdminUsername, c.cfg.AdminPassword)
		if err != nil {
			c.logger.Error("failed to obtain admin token", "error", err)
			return identity, err
		}
		identity.Groups, err = getAllGroupsForUser(r.Context(), c.client, c.cfg.Host, adminToken, c.cfg.CustomerName, c.cfg.Domain, tokenInfo, c.logger)
		if err != nil {
			return connector.Identity{}, err
		}
	}
	identity.Username = tokenInfo.User.Name
	identity.UserID = tokenInfo.User.ID

	user, err := getUser(r.Context(), c.client, c.cfg.Host, tokenInfo.User.ID, ksToken)
	if err != nil {
		return identity, err
	}
	if user.User.Email != "" {
		identity.Email = user.User.Email
		identity.EmailVerified = true
	}

	data := connectorData{Token: ksToken}
	connData, err := json.Marshal(data)
	if err != nil {
		c.logger.Error("failed to marshal connector data", "error", err)
		return identity, err
	}
	identity.ConnectorData = connData

	return identity, nil
}

// getKeystoneTokenFromFederation gets a Keystone token using an existing federation session.
// This method extracts federation cookies from the request and uses them to authenticate
// with Keystone's federation endpoint.
func (c *FederationConnector) getKeystoneTokenFromFederation(r *http.Request) (string, error) {
	c.logger.Debug("getting keystone token from federation cookies")
	baseURL := strings.TrimSuffix(c.cfg.Host, "/")
	federationAuthPath := strings.TrimPrefix(c.cfg.FederationAuthPath, "/")

	federationAuthURL := fmt.Sprintf("%s/%s", baseURL, federationAuthPath)
	c.logger.Debug("requesting keystone token from federation auth endpoint")

	req, err := http.NewRequest("GET", federationAuthURL, nil)
	if err != nil {
		c.logger.Error("failed to create federation auth request", "error", err)
		return "", err
	}

	shibbolethCookiePrefixes := []string{
		"_shibsession",
		"_shibstate",
	}

	for _, cookie := range r.Cookies() {
		cookieName := strings.ToLower(cookie.Name)
		for _, prefix := range shibbolethCookiePrefixes {
			if strings.HasPrefix(cookieName, prefix) {
				req.AddCookie(cookie)
				break
			}
		}
	}

	if userAgent := r.Header.Get("User-Agent"); userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	if referer := r.Header.Get("Referer"); referer != "" {
		req.Header.Set("Referer", referer)
	}

	clientNoRedirect := &http.Client{
		Timeout: c.client.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := clientNoRedirect.Do(req)
	if err != nil {
		c.logger.Error("failed to execute federation auth request", "error", err)
		return "", err
	}
	defer resp.Body.Close()

	token := resp.Header.Get("X-Subject-Token")
	if token == "" {
		c.logger.Error("No X-Subject-Token found in federation auth response")
		return "", fmt.Errorf("no X-Subject-Token found in federation auth response")
	}

	c.logger.Debug("successfully obtained keystone token from federation")
	return token, nil
}

// Close does nothing since HTTP connections are closed automatically.
func (c *FederationConnector) Close() error {
	return nil
}

// Refresh is used to refresh identity during token refresh.
// It checks if the user still exists and refreshes their group membership.
func (c *FederationConnector) Refresh(
	ctx context.Context, scopes connector.Scopes, identity connector.Identity,
) (connector.Identity, error) {
	c.logger.Info("refresh called", "userID", identity.UserID)

	adminToken, err := getAdminTokenUnscoped(ctx, c.client, c.cfg.Host, c.cfg.AdminUsername, c.cfg.AdminPassword)
	if err != nil {
		c.logger.Error("failed to obtain admin token for refresh", "error", err)
		return identity, err
	}

	// Check if the user still exists
	user, err := getUser(ctx, c.client, c.cfg.Host, identity.UserID, adminToken)
	if err != nil {
		c.logger.Error("failed to get user", "userID", identity.UserID, "error", err)
		return identity, err
	}
	if user == nil {
		c.logger.Error("user does not exist", "userID", identity.UserID)
		return identity, fmt.Errorf("keystone federation: user %q does not exist", identity.UserID)
	}

	tokenInfo := &tokenInfo{
		User: userKeystone{
			Name: identity.Username,
			ID:   identity.UserID,
		},
	}

	// If there is a token associated with this refresh token, use that to get more info
	var data connectorData
	if err := json.Unmarshal(identity.ConnectorData, &data); err != nil {
		c.logger.Error("failed to unmarshal connector data", "error", err)
		return identity, err
	}

	// If we have a stored token, try to use it to get token info
	if len(data.Token) > 0 {
		c.logger.Debug("using stored token to get token info")
		tokenInfoFromStored, err := getTokenInfo(ctx, c.client, c.cfg.Host, data.Token, c.logger)
		if err == nil {
			// Only use the stored token info if we could retrieve it successfully
			tokenInfo = tokenInfoFromStored
		} else {
			c.logger.Warn("could not get token info from stored token", "error", err)
		}
	}

	// If groups scope is requested, refresh the groups
	if scopes.Groups {
		c.logger.Info("refreshing groups", "userID", identity.UserID)
		var err error
		identity.Groups, err = getAllGroupsForUser(ctx, c.client, c.cfg.Host, adminToken, c.cfg.CustomerName, c.cfg.Domain, tokenInfo, c.logger)
		if err != nil {
			c.logger.Error("failed to get groups", "userID", identity.UserID, "error", err)
			return identity, err
		}
	}

	return identity, nil
}
