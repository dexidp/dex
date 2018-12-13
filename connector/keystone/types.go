package keystone

import (
	"github.com/sirupsen/logrus"
)

type keystoneConnector struct {
	Domain           string
	KeystoneHost     string
	KeystoneUsername string
	KeystonePassword string
	Logger           logrus.FieldLogger
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
	Domain           string `json:"domain"`
	KeystoneHost     string `json:"keystoneHost"`
	KeystoneUsername string `json:"keystoneUsername"`
	KeystonePassword string `json:"keystonePassword"`
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
