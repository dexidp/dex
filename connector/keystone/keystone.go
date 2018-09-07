// Package keystone provides authentication strategy using Keystone.
package keystone

import (
	"context"
	"fmt"
	"github.com/coreos/dex/connector"
	"github.com/sirupsen/logrus"
	"encoding/json"
	"net/http"
	"bytes"
	"io/ioutil"
	"log"
)

type KeystoneConnector struct {
	domain string
	keystoneURI string
	Logger   logrus.FieldLogger
}

var (
	_ connector.PasswordConnector = &KeystoneConnector{}
)

// Config holds the configuration parameters for Keystone connector.
// An example config:
//	connectors:
//		type: ksconfig
//		id: keystone
//		name: Keystone
//		config:
//			keystoneURI: http://example:5000/v3/auth/tokens
//			domain: default

type Config struct {
	Domain string `json:"domain"`
	KeystoneURI string `json:"keystoneURI"`
}

// Open returns an authentication strategy using Keystone.
func (c *Config) Open(id string, logger logrus.FieldLogger) (connector.Connector, error) {
	return &KeystoneConnector{c.Domain,c.KeystoneURI,logger}, nil
}

func (p KeystoneConnector) Close() error { return nil }

// Declare KeystoneJson struct to get a token
type KeystoneJson struct {
	Auth `json:"auth"`
}

type Auth struct {
	Identity `json:"identity"`
}

type Identity struct {
	Methods  []string `json:"methods"`
	Password `json:"password"`
}

type Password struct {
	User `json:"user"`
}

type User struct {
	Name   string `json:"name"`
	Domain `json:"domain"`
	Password string `json:"password"`
}

type Domain struct {
	ID string `json:"id"`
}

func (p KeystoneConnector) Login(ctx context.Context, s connector.Scopes, username, password string) (identity connector.Identity, validPassword bool, err error) {
	// Instantiate KeystoneJson struct type to get a token
	jsonData := KeystoneJson{
		Auth: Auth{
			Identity: Identity{
				Methods:[]string{"password"},
				Password: Password{
					User: User{
						Name: username,
						Domain: Domain{ID:p.domain},
						Password: password,
					},
				},
			},
		},
	}

	// Marshal jsonData
	jsonValue, _ := json.Marshal(jsonData)

	// Make an http post request to Keystone URI
	response, err := http.Post(p.keystoneURI, "application/json", bytes.NewBuffer(jsonValue))

	// Providing wrong password or wrong keystone URI throws error
	if err == nil && response.StatusCode == 201 {
		data, _ := ioutil.ReadAll(response.Body)
		fmt.Println(string(data))
		identity.Username =	username
		return identity, true, nil

	} else if err != nil {
		log.Fatal(err)
		return identity, false, err

	} else {
		fmt.Printf("The HTTP request failed with error %v\n", response.StatusCode)
		data, _ := ioutil.ReadAll(response.Body)
		fmt.Println(string(data))
		return identity, false, err

	}
	return identity, false, nil
}

func (p KeystoneConnector) Prompt() string { return "username" }