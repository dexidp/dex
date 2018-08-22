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

type Config struct {
	Domain string `json:"domain"`
	KeystoneURI string `json:"keystoneURI"`
}


// Open returns an authentication strategy which prompts for a predefined username and password.
func (c *Config) Open(id string, logger logrus.FieldLogger) (connector.Connector, error) {

	//fmt.Println(c.Domain)
	//fmt.Println(c.KeystoneURI)

	/*k := Keystone{
		domain:      c.Domain,
		keystoneURI: c.KeystoneURI,
		Logger:      logger,
	}

	fmt.Println("\n Returning keystone struct values ")
	return &Keystone{k.domain,k.keystoneURI,logger}, nil*/

	return &Keystone{c.Domain,c.KeystoneURI,logger}, nil
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


func (p Keystone) Login(ctx context.Context, s connector.Scopes, username, password string) (identity connector.Identity, validPassword bool, err error) {

	/*
	fmt.Println("\n Keystone login function called !!!!!!! ")
	fmt.Println("\n domain is !!!!!!! ")
	fmt.Println(p.domain)
	fmt.Println("\n uri is !!!!!!! ")
	fmt.Println(p.keystoneURI)
	*/


	// instantiate type
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



	//fmt.Println("\n :::::::::::::before marshaling, jsonData :::::::::::")
	//fmt.Print(jsonData)
	//fmt.Print("\n")


	jsonValue, _ := json.Marshal(jsonData)

	//fmt.Println("\n ::::::::::::::::after Marshaling, jsonValue ::::::::::")
	//fmt.Print(jsonValue )
	//fmt.Print("\n")

	//buff := bytes.NewBuffer(jsonValue)
	//jsonString := buff.String()

	//fmt.Println("\n ::::::::::::::::after converting to string, jsonString ::::::::::")
	//fmt.Print(jsonString )
	//fmt.Print("\n")
	//fmt.Print("\n")



	response, err := http.Post(p.keystoneURI, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		fmt.Println(string(data))
		return connector.Identity{}, false, nil
	}
	return identity, false, nil
}

func (p Keystone) Prompt() string { return "username" }
