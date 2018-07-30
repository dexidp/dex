// Package ksconnect implements connectors which help test various server components.
package ksconnect

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	//"github.com/coreos/dex/connector"
	"github.com/sirupsen/logrus"
	//"github.com/knangia/ksconnect"
)

// NewCallbackConnector returns a mock connector which requires no user interaction. It always returns
// the same (fake) identity.
//func NewCallbackConnector(logger logrus.FieldLogger) connector.Connector {
//	return &Callback{
	//	Identity: connector.Identity{
	//		UserID:        "0-385-28089-0",
	//		Username:      "Kilgore Trout",
	//		Email:         "kilgore@kilgore.trout",
	//		EmailVerified: true,
	//		Groups:        []string{"authors"},
	//		ConnectorData: connectorData,
	//	},
	//	Logger: logger,
	//}
//}
//fmt.Println("CONNECTOR FOR KEYSTONE IS IMPORTED")

func NewKeystoneConnector(logger logrus.FieldLogger) Connector {
	return &Keystone{
		Identity: Identity{
			Username: "Kilgore Trout",
			Password: "xyz",
		},
			Logger:    logger,
	}
}

var (
	//_ connector.CallbackConnector = &Callback{}

	_ KeystoneConnector = &Keystone{}
)

// Callback is a connector that requires no user interaction and always returns the same identity.
//type Callback struct {
	// The returned identity.
	//Identity connector.Identity
	//Logger   logrus.FieldLogger
//}

type Keystone struct {
	Identity Identity
	Logger   logrus.FieldLogger
}

// LoginURL returns the URL to redirect the user to login with.
func (m *Keystone) LoginURL(s Scopes, callbackURL, state string) (string, error) {
	u, err := url.Parse(callbackURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse callbackURL %q: %v", callbackURL, err)
	}
	v := u.Query()
	v.Set("state", state)
	u.RawQuery = v.Encode()
	return u.String(), nil
}

//var connectorData = []byte("foobar")

// HandleCallback parses the request and returns the user's identity
func (m *Keystone) HandleCallback(s Scopes, r *http.Request) (Identity, error) {
	return m.Identity, nil
}

// Refresh updates the identity during a refresh token request.
func (m *Keystone) Refresh(ctx context.Context, s Scopes, identity Identity) (Identity, error) {
	return m.Identity, nil
}

// CallbackConfig holds the configuration parameters for a connector which requires no interaction.
//type CallbackConfig struct{}

type KeystoneConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}



// Open returns an authentication strategy which requires no user interaction.
//func (c *KeystoneConfig) Open(id string, logger logrus.FieldLogger) (connector.Connector, error) {
//	return NewKeystoneConnector(logger), nil
//}

// PasswordConfig holds the configuration for a mock connector which prompts for the supplied
// username and password.
//type KeystoneConfig struct {
//	Username string `json:"username"`
//	Password string `json:"password"`
//}


// HandleCallback parses the request and returns the user's identity
//func (m *Callback) HandleCallback(s connector.Scopes, r *http.Request) (connector.Identity, error) {
//	return m.Identity, nil
//}

// Refresh updates the identity during a refresh token request.
//func (m *Callback) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
//	return m.Identity, nil
//}

// CallbackConfig holds the configuration parameters for a connector which requires no interaction.
//type CallbackConfig struct{}

// Open returns an authentication strategy which requires no user interaction.
//func (c *CallbackConfig) Open(id string, logger logrus.FieldLogger) (connector.Connector, error) {
//	return NewCallbackConnector(logger), nil
//}



// Open returns an authentication strategy which prompts for a predefined username and password.
func (c *KeystoneConfig) Open(id string, logger logrus.FieldLogger) (Connector, error) {
	if c.Username == "" {
		return nil, errors.New("no username supplied")
	}
	if c.Password == "" {
		return nil, errors.New("no password supplied")
	}
	i := Identity{c.Username, c.Password}
	return &Keystone{i, logger}, nil
}


func (p Keystone) Close() error { return nil }

////////// wip :::: we have identity in keystone struct and not separate username and password

func (p Keystone) Login(ctx context.Context, s Scopes, username, password string) (identity Identity, validPassword bool, err error) {
	if username == p.Identity.Username && password == p.Identity.Password {
		return Identity{
			Username: "Kilgore Trout",
			Password: "xyz",
		}, true, nil
	}
	return identity, false, nil
}

func (p Keystone) Prompt() string { return "" }
