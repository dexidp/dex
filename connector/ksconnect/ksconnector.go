// Package ksconnect implements connectors which help test various server components.
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
	keystoneURI string
	domain string
	Logger   logrus.FieldLogger
}

var (
	_ connector.PasswordConnector = &Keystone{}
)

// CallbackConfig holds the configuration parameters for a connector which requires no interaction.
//type CallbackConfig struct{}

type KeystoneConfig struct {
	domain string `json:"domain"`
	keystoneURI string `json:"keystoneURI"`
}

// Open returns an authentication strategy which prompts for a predefined username and password.
func (c *KeystoneConfig) Open(id string, logger logrus.FieldLogger) (connector.Connector, error) {

	return &Keystone{c.keystoneURI,c.domain, logger}, nil
}

func (p Keystone) Close() error { return nil }

/*type KeystoneJson struct {
	Auth struct {
		Identity struct {
			Methods  []string `json:"methods"`
			Password struct {
				User struct {
					Name   string `json:"name"`
					Domain struct {
						ID string `json:"id"`
					} `json:"domain"`
					Password string `json:"password"`
				} `json:"user"`
			} `json:"password"`
		} `json:"identity"`
	} `json:"auth"`
}*/


// declare types
type KeystoneJson struct {
	Auth `json:"auth"`
}

type Auth struct {
	Identity `json:"identity"`
}

type Identity struct {
	Methods  []string `json:"methods"`
	Password

}

type Password struct {
	User `json:"identity"`
}

type User struct {
	Name   string `json:"name"`
	Domain `json:"domain"`
	Password string `json:"password"`
}

type Domain struct {
	ID string `json:"id"`
}
//var jsonData KeystoneJson

func (p Keystone) Login(ctx context.Context, s connector.Scopes, username, password string) (identity connector.Identity, validPassword bool, err error) {
	fmt.Println("Keystone login function called !!!!!!! ")


	// instantiate type
	jsonData := KeystoneJson{
		Auth: Auth{
			Identity: Identity{
				Methods:[]string{"password"},
				Password: Password{
					User: User{
						Name: username,
						Domain: Domain{ID: p.domain},
						Password: password,
					},
				},
			},
		},
	}

	//jsonData.Auth.Identity.Methods = {"password"}
	//jsonData.Auth.Identity.Password.User.Name = username
	//jsonData.Auth.Identity.Password.User.Domain.ID = p.domain
	//jsonData.Auth.Identity.Password.User.Password = password

	jsonValue, _ := json.Marshal(jsonData)
	response, err := http.Post("http://192.168.180.200:5000/v3/auth/tokens", "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		fmt.Println(string(data))
		return connector.Identity{}, false, nil
	}
	//if username == "foo" && password == "bar" {
	//	return connector.Identity{
	//		Username: "Kilgore Trout",
	//		Password: "xyz",
	//	}, true, nil
	//}
	return identity, false, nil
}

func (p Keystone) Prompt() string { return "username" }
