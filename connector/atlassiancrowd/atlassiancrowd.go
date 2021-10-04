// Package atlassiancrowd provides authentication strategies using Atlassian Crowd.
package atlassiancrowd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/groups"
	"github.com/dexidp/dex/pkg/log"
)

// Config holds configuration options for Atlassian Crowd connector.
// Crowd connectors require executing two queries, the first to find
// the user based on the username and password given to the connector.
// The second to use the user entry to search for groups.
//
// An example config:
//
//     type: atlassian-crowd
//     config:
//       baseURL: https://crowd.example.com/context
//       clientID: applogin
//       clientSecret: appP4$$w0rd
//       # users can be restricted by a list of groups
//       groups:
//       - admin
//       # Prompt for username field
//       usernamePrompt: Login
//		 preferredUsernameField: name
//
type Config struct {
	BaseURL      string   `json:"baseURL"`
	ClientID     string   `json:"clientID"`
	ClientSecret string   `json:"clientSecret"`
	Groups       []string `json:"groups"`

	// PreferredUsernameField allows users to set the field to any of the
	// following values: "key", "name" or "email".
	// If unset, the preferred_username field will remain empty.
	PreferredUsernameField string `json:"preferredUsernameField"`

	// UsernamePrompt allows users to override the username attribute (displayed
	// in the username/password prompt). If unset, the handler will use.
	// "Username".
	UsernamePrompt string `json:"usernamePrompt"`
}

type crowdUser struct {
	Key    string
	Name   string
	Active bool
	Email  string
}

type crowdGroups struct {
	Groups []struct {
		Name string
	} `json:"groups"`
}

type crowdAuthentication struct {
	Token string
	User  struct {
		Name string
	} `json:"user"`
	CreatedDate uint64 `json:"created-date"`
	ExpiryDate  uint64 `json:"expiry-date"`
}

type crowdAuthenticationError struct {
	Reason  string
	Message string
}

// Open returns a strategy for logging in through Atlassian Crowd
func (c *Config) Open(_ string, logger log.Logger) (connector.Connector, error) {
	if c.BaseURL == "" {
		return nil, fmt.Errorf("crowd: no baseURL provided for crowd connector")
	}
	return &crowdConnector{Config: *c, logger: logger}, nil
}

type crowdConnector struct {
	Config
	logger log.Logger
}

var (
	_ connector.PasswordConnector = (*crowdConnector)(nil)
	_ connector.RefreshConnector  = (*crowdConnector)(nil)
)

type refreshData struct {
	Username string `json:"username"`
}

func (c *crowdConnector) Login(ctx context.Context, s connector.Scopes, username, password string) (ident connector.Identity, validPass bool, err error) {
	// make this check to avoid empty passwords.
	if password == "" {
		return connector.Identity{}, false, nil
	}

	// We want to return a different error if the user's password is incorrect vs
	// if there was an error.
	var incorrectPass bool
	var user crowdUser

	client := c.crowdAPIClient()

	if incorrectPass, err = c.authenticateWithPassword(ctx, client, username, password); err != nil {
		return connector.Identity{}, false, err
	}

	if incorrectPass {
		return connector.Identity{}, false, nil
	}

	if user, err = c.user(ctx, client, username); err != nil {
		return connector.Identity{}, false, err
	}

	ident = c.identityFromCrowdUser(user)
	if s.Groups {
		userGroups, err := c.getGroups(ctx, client, s.Groups, ident.Username)
		if err != nil {
			return connector.Identity{}, false, fmt.Errorf("crowd: failed to query groups: %v", err)
		}
		ident.Groups = userGroups
	}

	if s.OfflineAccess {
		refresh := refreshData{Username: username}
		// Encode entry for following up requests such as the groups query and refresh attempts.
		if ident.ConnectorData, err = json.Marshal(refresh); err != nil {
			return connector.Identity{}, false, fmt.Errorf("crowd: marshal refresh data: %v", err)
		}
	}

	return ident, true, nil
}

func (c *crowdConnector) Refresh(ctx context.Context, s connector.Scopes, ident connector.Identity) (connector.Identity, error) {
	var data refreshData
	if err := json.Unmarshal(ident.ConnectorData, &data); err != nil {
		return ident, fmt.Errorf("crowd: failed to unmarshal internal data: %v", err)
	}

	var user crowdUser
	client := c.crowdAPIClient()

	user, err := c.user(ctx, client, data.Username)
	if err != nil {
		return ident, fmt.Errorf("crowd: get user %q: %v", data.Username, err)
	}

	newIdent := c.identityFromCrowdUser(user)
	newIdent.ConnectorData = ident.ConnectorData

	// If user exists, authenticate it to prolong sso session.
	err = c.authenticateUser(ctx, client, data.Username)
	if err != nil {
		return ident, fmt.Errorf("crowd: authenticate user: %v", err)
	}

	if s.Groups {
		userGroups, err := c.getGroups(ctx, client, s.Groups, newIdent.Username)
		if err != nil {
			return connector.Identity{}, fmt.Errorf("crowd: failed to query groups: %v", err)
		}
		newIdent.Groups = userGroups
	}
	return newIdent, nil
}

func (c *crowdConnector) Prompt() string {
	return c.UsernamePrompt
}

func (c *crowdConnector) crowdAPIClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

// authenticateWithPassword creates a new session for user and validates a password with Crowd API
func (c *crowdConnector) authenticateWithPassword(ctx context.Context, client *http.Client, username string, password string) (invalidPass bool, err error) {
	req, err := c.crowdUserManagementRequest(ctx,
		"POST",
		"/session",
		struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}{Username: username, Password: password},
	)
	if err != nil {
		return false, fmt.Errorf("crowd: new auth pass api request %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("crowd: api request %v", err)
	}
	defer resp.Body.Close()

	body, err := c.validateCrowdResponse(resp)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != http.StatusCreated {
		var authError crowdAuthenticationError
		if err := json.Unmarshal(body, &authError); err != nil {
			return false, fmt.Errorf("unmarshal auth pass response: %d %v %q", resp.StatusCode, err, string(body))
		}

		if authError.Reason == "INVALID_USER_AUTHENTICATION" {
			return true, nil
		}

		return false, fmt.Errorf("%s: %s", resp.Status, authError.Message)
	}

	var authResponse crowdAuthentication

	if err := json.Unmarshal(body, &authResponse); err != nil {
		return false, fmt.Errorf("decode auth response: %v", err)
	}

	return false, nil
}

// authenticateUser creates a new session for user without password validations with Crowd API
func (c *crowdConnector) authenticateUser(ctx context.Context, client *http.Client, username string) error {
	req, err := c.crowdUserManagementRequest(ctx,
		"POST",
		"/session?validate-password=false",
		struct {
			Username string `json:"username"`
		}{Username: username},
	)
	if err != nil {
		return fmt.Errorf("crowd: new auth api request %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("crowd: api request %v", err)
	}
	defer resp.Body.Close()

	body, err := c.validateCrowdResponse(resp)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("%s: %s", resp.Status, body)
	}

	var authResponse crowdAuthentication

	if err := json.Unmarshal(body, &authResponse); err != nil {
		return fmt.Errorf("decode auth response: %v", err)
	}

	return nil
}

// user retrieves user info from Crowd API
func (c *crowdConnector) user(ctx context.Context, client *http.Client, username string) (crowdUser, error) {
	var user crowdUser

	req, err := c.crowdUserManagementRequest(ctx,
		"GET",
		fmt.Sprintf("/user?username=%s", username),
		nil,
	)
	if err != nil {
		return user, fmt.Errorf("crowd: new user api request %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return user, fmt.Errorf("crowd: api request %v", err)
	}
	defer resp.Body.Close()

	body, err := c.validateCrowdResponse(resp)
	if err != nil {
		return user, err
	}

	if resp.StatusCode != http.StatusOK {
		return user, fmt.Errorf("%s: %s", resp.Status, body)
	}

	if err := json.Unmarshal(body, &user); err != nil {
		return user, fmt.Errorf("failed to decode response: %v", err)
	}

	return user, nil
}

// groups retrieves groups from Crowd API
func (c *crowdConnector) groups(ctx context.Context, client *http.Client, username string) (userGroups []string, err error) {
	var crowdGroups crowdGroups

	req, err := c.crowdUserManagementRequest(ctx,
		"GET",
		fmt.Sprintf("/user/group/nested?username=%s", username),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("crowd: new groups api request %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("crowd: api request %v", err)
	}
	defer resp.Body.Close()

	body, err := c.validateCrowdResponse(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", resp.Status, body)
	}

	if err := json.Unmarshal(body, &crowdGroups); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	for _, group := range crowdGroups.Groups {
		userGroups = append(userGroups, group.Name)
	}

	return userGroups, nil
}

// identityFromCrowdUser converts crowdUser to Identity
func (c *crowdConnector) identityFromCrowdUser(user crowdUser) connector.Identity {
	identity := connector.Identity{
		Username:      user.Name,
		UserID:        user.Key,
		Email:         user.Email,
		EmailVerified: true,
	}

	switch c.PreferredUsernameField {
	case "key":
		identity.PreferredUsername = user.Key
	case "name":
		identity.PreferredUsername = user.Name
	case "email":
		identity.PreferredUsername = user.Email
	default:
		if c.PreferredUsernameField != "" {
			c.logger.Warnf("preferred_username left empty. Invalid crowd field mapped to preferred_username: %s", c.PreferredUsernameField)
		}
	}

	return identity
}

// getGroups retrieves a list of user's groups and filters it
func (c *crowdConnector) getGroups(ctx context.Context, client *http.Client, groupScope bool, userLogin string) ([]string, error) {
	crowdGroups, err := c.groups(ctx, client, userLogin)
	if err != nil {
		return nil, err
	}

	if len(c.Groups) > 0 {
		filteredGroups := groups.Filter(crowdGroups, c.Groups)
		if len(filteredGroups) == 0 {
			return nil, fmt.Errorf("crowd: user %q is not in any of the required groups", userLogin)
		}
		return filteredGroups, nil
	} else if groupScope {
		return crowdGroups, nil
	}

	return nil, nil
}

// crowdUserManagementRequest create a http.Request with basic auth, json payload and Accept header
func (c *crowdConnector) crowdUserManagementRequest(ctx context.Context, method string, apiURL string, jsonPayload interface{}) (*http.Request, error) {
	var body io.Reader
	if jsonPayload != nil {
		jsonData, err := json.Marshal(jsonPayload)
		if err != nil {
			return nil, fmt.Errorf("crowd: marshal API json payload: %v", err)
		}
		body = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s/rest/usermanagement/1%s", c.BaseURL, apiURL), body)
	if err != nil {
		return nil, fmt.Errorf("new API req: %v", err)
	}
	req = req.WithContext(ctx)

	// Crowd API requires a basic auth
	req.SetBasicAuth(c.ClientID, c.ClientSecret)
	req.Header.Set("Accept", "application/json")
	if jsonPayload != nil {
		req.Header.Set("Content-type", "application/json")
	}
	return req, nil
}

// validateCrowdResponse validates unique not JSON responses from API
func (c *crowdConnector) validateCrowdResponse(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("crowd: read user body: %v", err)
	}

	if resp.StatusCode == http.StatusForbidden && strings.Contains(string(body), "The server understood the request but refuses to authorize it.") {
		c.logger.Debugf("crowd response validation failed: %s", string(body))
		return nil, fmt.Errorf("dex is forbidden from making requests to the Atlassian Crowd application by URL %q", c.BaseURL)
	}

	if resp.StatusCode == http.StatusUnauthorized && string(body) == "Application failed to authenticate" {
		c.logger.Debugf("crowd response validation failed: %s", string(body))
		return nil, fmt.Errorf("dex failed to authenticate Crowd Application with ID %q", c.ClientID)
	}
	return body, nil
}
