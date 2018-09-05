package custom

import (
	"fmt"
	"net/http"

	"github.com/coreos/dex/connector"
	"github.com/sirupsen/logrus"
)

// Config contains a sample object
type Config struct {
	Name   string `json:"name"`
	Email  string `json:"Email"`
	UserID string `json:"userid"`
}

// Open validates the config and returns a connector. It does not actually
// validate connectivity with the provider.
func (c *Config) Open(id string, logger logrus.FieldLogger) (connector.Connector, error) {
	return &customConnector{c}, nil
}

type customConnector struct {
	config *Config
}

func (c *customConnector) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
	return fmt.Sprintf("%s?state=%s", callbackURL, state), nil
}

func (c *customConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	identity = connector.Identity{
		Username:      c.config.Name,
		UserID:        c.config.UserID,
		Email:         c.config.Email,
		EmailVerified: true,
	}
	return identity, nil
}
