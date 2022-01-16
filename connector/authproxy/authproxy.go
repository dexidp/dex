// Package authproxy implements a connector which relies on external
// authentication (e.g. mod_auth in Apache2) and returns an identity based
// on HTTP Headers
// The UserHeader is mandatory in the request, and defaults to X-Remote-User if
// not configured. Email defaults to the value from the userHeader but a separate
// header can be used by configuring emailHeader.
// Groups are not enabled by default but can be enabled by configuring groupHeader
// and are supported as either a single delimited value, e.g.
//  X-Remote-Groups: group1,group2
// or as a repeated header, e.g.
//  X-Remote-Group: group1
//  X-Remote-Group: group2
// depending on whether groupSeparator is configured.
package authproxy

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/log"
)

// Config holds the configuration parameters for a connector which returns an
// identity based on HTTP headers.
type Config struct {
	UserHeader     string `json:"userHeader"`
	GroupHeader    string `json:"groupHeader"`
	GroupSeparator string `json:"groupSeparator"`
	EmailHeader    string `json:"emailHeader"`
}

// Open returns an authentication strategy which requires no user interaction.
func (c *Config) Open(id string, logger log.Logger) (connector.Connector, error) {
	if c.UserHeader == "" {
		c.UserHeader = "X-Remote-User"
	}

	return &callback{config: *c, logger: logger, pathSuffix: "/" + id}, nil
}

// Callback is a connector which returns an identity based on HTTP headers
type callback struct {
	config     Config
	logger     log.Logger
	pathSuffix string
}

// LoginURL returns the URL to redirect the user to login with.
func (m *callback) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
	u, err := url.Parse(callbackURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse callbackURL %q: %v", callbackURL, err)
	}
	u.Path += m.pathSuffix
	v := u.Query()
	v.Set("state", state)
	u.RawQuery = v.Encode()
	return u.String(), nil
}

// HandleCallback parses the request and returns the user's identity
func (m *callback) HandleCallback(s connector.Scopes, r *http.Request) (connector.Identity, error) {
	remoteUser := r.Header.Get(m.config.UserHeader)
	if remoteUser == "" {
		return connector.Identity{}, fmt.Errorf("required HTTP header %s is not set", m.config.UserHeader)
	}

	groups := []string{}
	if s.Groups && m.config.GroupHeader != "" {
		if m.config.GroupSeparator == "" {
			// No separator provided, check for multiple copies of header, one per group
			groups = append(groups, r.Header.Values(m.config.GroupHeader)...)
		} else {
			// Separator provided, treat header as delimited
			groups = strings.Split(r.Header.Get(m.config.GroupHeader), m.config.GroupSeparator)
		}
	}

	email := remoteUser
	if m.config.EmailHeader != "" && r.Header.Get(m.config.EmailHeader) != "" {
		email = r.Header.Get(m.config.EmailHeader)
	}

	return connector.Identity{
		UserID:        remoteUser, // TODO: figure out if this is a bad ID value.
		Username:      remoteUser,
		Email:         email,
		EmailVerified: true,
		Groups:        groups,
	}, nil
}
