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
	domain string
	keystoneURI string
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

/*type Keystone struct {
	Config
	Logger   logrus.FieldLogger
}

var (
	_ connector.PasswordConnector = &Keystone{}
)*/

// Open returns an authentication strategy which prompts for a predefined username and password.
func (c *KeystoneConfig) Open(id string, logger logrus.FieldLogger) (connector.Connector, error) {

	fmt.Println(c.domain)
	fmt.Println(c.keystoneURI)
	//i := Config{keystoneURI:c.keystoneURI,domain:c.domain}
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

//var kjson KeystoneJson

func (p Keystone) Login(ctx context.Context, s connector.Scopes, username, password string) (identity connector.Identity, validPassword bool, err error) {

	fmt.Println("\n Keystone login function called !!!!!!! ")
	fmt.Println("\n domain is !!!!!!! ")
	fmt.Println(p.domain)
	fmt.Println("\n uri is !!!!!!! ")
	fmt.Println(p.keystoneURI)


	// instantiate type
	jsonData := KeystoneJson{
		Auth: Auth{
			Identity: Identity{
				Methods:[]string{"password"},
				Password: Password{
					User: User{
						Name: username,
						Domain: Domain{ID:"default"},
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

	//jsonConf := []byte(`{"Auth":{"Identity":{"Methods":{"password"},"Password":{"User":{"Name": username,"Domain":{"ID": "default"},"Password":password}}}}}`)



		//jsonConfig := []byte(`{"server":{"host":"localhost","port":"8080"},"database":{"host":"localhost","user":"db_user","password":"supersecret","db":"my_db"}}`)

	fmt.Println("\n :::::::::::::before marshaling, jsonData :::::::::::")
	fmt.Print(jsonData)
	fmt.Print("\n")

	//var config Config
	//err := json.Unmarshal(jsonConfig, &config)
	jsonValue, _ := json.Marshal(jsonData)
	//jsonValue, _ := json.Unmarshal(jsonConf, &kjson)

	fmt.Println("\n ::::::::::::::::after Marshaling, jsonValue ::::::::::")
	fmt.Print(jsonValue )
	fmt.Print("\n")

	//jsonstring := String(jsonValue)
	buff := bytes.NewBuffer(jsonValue)
	jsonString := buff.String()

	fmt.Println("\n ::::::::::::::::after converting to string, jsonString ::::::::::")
	fmt.Print(jsonString )
	fmt.Print("\n")
	fmt.Print("\n")

	//buff := bytes.NewBuffer([]byte("abcdefg"))

	//fmt.Println(buff.String())


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
