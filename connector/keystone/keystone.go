// Package keystone provides authentication strategy using Keystone.
package keystone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/dexidp/dex/connector"
)

type conn struct {
	Domain        string
	Host          string
	AdminUsername string
	AdminPassword string
	Logger        logrus.FieldLogger
}

type userKeystone struct {
	Domain domainKeystone `json:"domain"`
	ID     string         `json:"id"`
	Name   string         `json:"name"`
}

type domainKeystone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Config holds the configuration parameters for Keystone connector.
// Keystone should expose API v3
// An example config:
//	connectors:
//		type: keystone
//		id: keystone
//		name: Keystone
//		config:
//			keystoneHost: http://example:5000
//			domain: default
//      keystoneUsername: demo
//      keystonePassword: DEMO_PASS
type Config struct {
	Domain        string `json:"domain"`
	Host          string `json:"keystoneHost"`
	AdminUsername string `json:"keystoneUsername"`
	AdminPassword string `json:"keystonePassword"`
}

type loginRequestData struct {
	auth `json:"auth"`
}

type auth struct {
	Identity identity `json:"identity"`
}

type identity struct {
	Methods  []string `json:"methods"`
	Password password `json:"password"`
}

type password struct {
	User user `json:"user"`
}

type user struct {
	Name     string `json:"name"`
	Domain   domain `json:"domain"`
	Password string `json:"password"`
}

type domain struct {
	ID string `json:"id"`
}

type token struct {
	User userKeystone `json:"user"`
}

type tokenResponse struct {
	Token token `json:"token"`
}

type group struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type groupsResponse struct {
	Groups []group `json:"groups"`
}

var (
	_ connector.PasswordConnector = &conn{}
	_ connector.RefreshConnector  = &conn{}
)

// Open returns an authentication strategy using Keystone.
func (c *Config) Open(id string, logger logrus.FieldLogger) (connector.Connector, error) {
	return &conn{
		c.Domain,
		c.Host,
		c.AdminUsername,
		c.AdminPassword,
		logger}, nil
}

func (p *conn) Close() error { return nil }

func (p *conn) Login(ctx context.Context, scopes connector.Scopes, username, password string) (identity connector.Identity, validPassword bool, err error) {
	resp, err := p.getTokenResponse(ctx, username, password)
	if err != nil {
		return identity, false, fmt.Errorf("keystone: error %v", err)
	}
	if resp.StatusCode/100 != 2 {
		return identity, false, fmt.Errorf("keystone login: error %v", resp.StatusCode)
	}
	if resp.StatusCode != 201 {
		return identity, false, nil
	}
	token := resp.Header.Get("X-Subject-Token")
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return identity, false, err
	}
	defer resp.Body.Close()
	var tokenResp = new(tokenResponse)
	err = json.Unmarshal(data, &tokenResp)
	if err != nil {
		return identity, false, fmt.Errorf("keystone: invalid token response: %v", err)
	}
	if scopes.Groups {
		groups, err := p.getUserGroups(ctx, tokenResp.Token.User.ID, token)
		if err != nil {
			return identity, false, err
		}
		identity.Groups = groups
	}
	identity.Username = username
	identity.UserID = tokenResp.Token.User.ID
	return identity, true, nil
}

func (p *conn) Prompt() string { return "username" }

func (p *conn) Refresh(
	ctx context.Context, scopes connector.Scopes, identity connector.Identity) (connector.Identity, error) {

	token, err := p.getAdminToken(ctx)
	if err != nil {
		return identity, fmt.Errorf("keystone: failed to obtain admin token: %v", err)
	}
	ok, err := p.checkIfUserExists(ctx, identity.UserID, token)
	if err != nil {
		return identity, err
	}
	if !ok {
		return identity, fmt.Errorf("keystone: user %q does not exist", identity.UserID)
	}
	if scopes.Groups {
		groups, err := p.getUserGroups(ctx, identity.UserID, token)
		if err != nil {
			return identity, err
		}
		identity.Groups = groups
	}
	return identity, nil
}

func (p *conn) getTokenResponse(ctx context.Context, username, pass string) (response *http.Response, err error) {
	client := &http.Client{}
	jsonData := loginRequestData{
		auth: auth{
			Identity: identity{
				Methods: []string{"password"},
				Password: password{
					User: user{
						Name:     username,
						Domain:   domain{ID: p.Domain},
						Password: pass,
					},
				},
			},
		},
	}
	jsonValue, err := json.Marshal(jsonData)
	if err != nil {
		return nil, err
	}
	// https://developer.openstack.org/api-ref/identity/v3/#password-authentication-with-unscoped-authorization
	authTokenURL := p.Host + "/v3/auth/tokens/"
	req, err := http.NewRequest("POST", authTokenURL, bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	return client.Do(req)
}

func (p *conn) getAdminToken(ctx context.Context) (string, error) {
	resp, err := p.getTokenResponse(ctx, p.AdminUsername, p.AdminPassword)
	if err != nil {
		return "", err
	}
	token := resp.Header.Get("X-Subject-Token")
	return token, nil
}

func (p *conn) checkIfUserExists(ctx context.Context, userID string, token string) (bool, error) {
	// https://developer.openstack.org/api-ref/identity/v3/#show-user-details
	userURL := p.Host + "/v3/users/" + userID
	client := &http.Client{}
	req, err := http.NewRequest("GET", userURL, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("X-Auth-Token", token)
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}

	if resp.StatusCode == 200 {
		return true, nil
	}
	return false, err
}

func (p *conn) getUserGroups(ctx context.Context, userID string, token string) ([]string, error) {
	client := &http.Client{}
	// https://developer.openstack.org/api-ref/identity/v3/#list-groups-to-which-a-user-belongs
	groupsURL := p.Host + "/v3/users/" + userID + "/groups"
	req, err := http.NewRequest("GET", groupsURL, nil)
	req.Header.Set("X-Auth-Token", token)
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		p.Logger.Errorf("keystone: error while fetching user %q groups\n", userID)
		return nil, err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var groupsResp = new(groupsResponse)

	err = json.Unmarshal(data, &groupsResp)
	if err != nil {
		return nil, err
	}

	groups := make([]string, len(groupsResp.Groups))
	for i, group := range groupsResp.Groups {
		groups[i] = group.Name
	}
	return groups, nil
}
