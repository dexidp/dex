// Package ksconnect implements connectors which help test various server components.
package ksconnect

import (
	"context"

	"github.com/coreos/dex/connector"
	"github.com/sirupsen/logrus"
)

type Keystone struct {
	Identity connector.Identity
	Logger   logrus.FieldLogger
}

var (
	_ connector.PasswordConnector = &Keystone{}
)

// CallbackConfig holds the configuration parameters for a connector which requires no interaction.
//type CallbackConfig struct{}

type KeystoneConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Open returns an authentication strategy which prompts for a predefined username and password.
func (c *KeystoneConfig) Open(id string, logger logrus.FieldLogger) (connector.Connector, error) {

	i := connector.Identity{Username: c.Username, Password:c.Password }
	return &Keystone{i, logger}, nil
}


func (p Keystone) Close() error { return nil }

////////// wip :::: we have identity in keystone struct and not separate username and password

func (p Keystone) Login(ctx context.Context, s connector.Scopes, username, password string) (identity connector.Identity, validPassword bool, err error) {
	if username == "foo" && password == "bar" {
		return connector.Identity{
			Username: "Kilgore Trout",
			Password: "xyz",
		}, true, nil
	}
	return identity, false, nil
}

func (p Keystone) Prompt() string { return "pass!" }
