// // Package ksconnect provides authentication strategies using Keystone.
package ksconnect

import (
	"context"
	"fmt"
	"github.com/coreos/dex/connector"
	"github.com/sirupsen/logrus"
	"encoding/json"
	"net/http"
	"bytes"
	"io/ioutil"
)

type Keystone struct {
	domain string
	keystoneURI string
	Logger   logrus.FieldLogger
}

var (
	_ connector.PasswordConnector = &Keystone{}
)

// Config holds the configuration parameters for Keystone connector.
// An example config:
//	connectors:
//		type: ksconfig
//		id: ksconnect
//		name: Keystone
//		config:
//			keystoneURI: http://192.168.180.200:5000/v3/auth/tokens
//			domain: default

type Config struct {
	Domain string `json:"domain"`
	KeystoneURI string `json:"keystoneURI"`
}


// Open returns an authentication strategy using Keystone.
func (c *Config) Open(id string, logger logrus.FieldLogger) (connector.Connector, error) {

	return &Keystone{c.Domain,c.KeystoneURI,logger}, nil
}

func (p Keystone) Close() error { return nil }

//Declare KeystoneJson struct to get a token with default scope

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


func (p Keystone) Login(ctx context.Context, s connector.Scopes, username, password string) (identity connector.Identity, validPassword bool, err error) {

	// instantiate KeystoneJson struct type to get a token with default scope
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

	//Marshal jsonData
	jsonValue, _ := json.Marshal(jsonData)

	//Make a http post request to Keystone URI
	response, err := http.Post(p.keystoneURI, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		fmt.Println(string(data))
		id :=connector.Identity{
			Username: username,
		}
		return id, true, nil
	}
	return identity, false, nil
}

func (p Keystone) Prompt() string { return "username" }
