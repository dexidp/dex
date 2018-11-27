// Package keystone provides authentication strategy using Keystone.
package keystone

import (
	"context"
	"fmt"
	"github.com/dexidp/dex/connector"
	"github.com/sirupsen/logrus"
	"encoding/json"
	"net/http"
	"bytes"
	"io/ioutil"
)

var (
	_ connector.PasswordConnector = &Connector{}
  	_ connector.RefreshConnector = &Connector{}
)

// Open returns an authentication strategy using Keystone.
func (c *Config) Open(id string, logger logrus.FieldLogger) (connector.Connector, error) {
	return &Connector{c.Domain, c.KeystoneHost,
	c.KeystoneUsername, c.KeystonePassword, logger}, nil
}

func (p Connector) Close() error { return nil }

func (p Connector) Login(ctx context.Context, s connector.Scopes, username, password string) (
		identity connector.Identity, validPassword bool, err error) {
	response, err := p.getTokenResponse(username, password)

	// Providing wrong password or wrong keystone URI throws error
	if err == nil && response.StatusCode == 201 {
    	token := response.Header["X-Subject-Token"][0]
		data, _ := ioutil.ReadAll(response.Body)

    	var tokenResponse = new(TokenResponse)
    	err := json.Unmarshal(data, &tokenResponse)

    	if err != nil {
      		fmt.Printf("keystone: invalid token response: %v", err)
      		return identity, false, err
    	}
    	groups, err := p.getUserGroups(tokenResponse.Token.User.ID, token)

    	if err != nil {
      		return identity, false, err
    	}

		identity.Username =	username
    	identity.UserID = tokenResponse.Token.User.ID
   	 	identity.Groups = groups
		return identity, true, nil

	} else if err != nil {
    	fmt.Printf("keystone: error %v", err)
		return identity, false, err

	} else {
		data, _ := ioutil.ReadAll(response.Body)
		fmt.Println(string(data))
		return identity, false, err
	}
	return identity, false, nil
}

func (p Connector) Prompt() string { return "username" }

func (p Connector) Refresh(
	ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {

  	if len(identity.ConnectorData) == 0 {
  		return identity, nil
	}

	token, err := p.getAdminToken()

  	if err != nil {
    	fmt.Printf("keystone: failed to obtain admin token")
    	return identity, err
  	}

  	ok := p.checkIfUserExists(identity.UserID, token)
  	if !ok {
  		fmt.Printf("keystone: user %q does not exist\n", identity.UserID)
     	return identity, fmt.Errorf("keystone: user %q does not exist", identity.UserID)
  	}

  	groups, err := p.getUserGroups(identity.UserID, token)
  	if err != nil {
    	fmt.Printf("keystone: Failed to fetch user %q groups", identity.UserID)
    	return identity, fmt.Errorf("keystone: failed to fetch user %q groups", identity.UserID)
  	}

  	identity.Groups = groups
  	fmt.Printf("Identity data after use of refresh token: %v", identity)
	return identity, nil
}


func (p Connector) getTokenResponse(username, password string) (response *http.Response, err error) {
	jsonData := LoginRequestData{
		Auth: Auth{
			Identity: Identity{
				Methods:[]string{"password"},
				Password: Password{
					User: User{
						Name: username,
						Domain: Domain{ID:p.Domain},
						Password: password,
					},
				},
			},
		},
	}
	jsonValue, _ := json.Marshal(jsonData)
  	loginURI := p.KeystoneHost + "/v3/auth/tokens"
	return http.Post(loginURI, "application/json", bytes.NewBuffer(jsonValue))
}

func (p Connector) getAdminToken()(string, error) {
  	response, err := p.getTokenResponse(p.KeystoneUsername, p.KeystonePassword)
  	if err!= nil {
    	return "", err
  	}
  	token := response.Header["X-Subject-Token"][0]
  	return token, nil
}

func (p Connector) checkIfUserExists(userID string, token string) (bool) {
  	groupsURI := p.KeystoneHost + "/v3/users/" + userID
  	client := &http.Client{}
  	req, _ := http.NewRequest("GET", groupsURI, nil)
  	req.Header.Set("X-Auth-Token", token)
  	response, err :=  client.Do(req)
  	if err == nil && response.StatusCode == 200 {
    	return true
  	}
  	return false
}

func (p Connector) getUserGroups(userID string, token string) ([]string, error) {
  	groupsURI := p.KeystoneHost + "/v3/users/" + userID + "/groups"
  	client := &http.Client{}
  	req, _ := http.NewRequest("GET", groupsURI, nil)
  	req.Header.Set("X-Auth-Token", token)
  	response, err :=  client.Do(req)

  	if err != nil {
    	fmt.Printf("keystone: error while fetching user %q groups\n", userID)
    	return nil, err
  	}
  	data, _ := ioutil.ReadAll(response.Body)
  	var groupsResponse = new(GroupsResponse)
  	err = json.Unmarshal(data, &groupsResponse)
  	if err != nil {
    	return nil, err
  	}
  	groups := []string{}
  	for _, group := range groupsResponse.Groups {
  		groups = append(groups, group.Name)
  	}
  	return groups, nil
}
