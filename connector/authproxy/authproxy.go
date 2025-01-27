// Package authproxy implements a connector which relies on external
// authentication (e.g. mod_auth in Apache2) and returns an identity with the
// HTTP header X-Remote-User as verified email.
package authproxy

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/dexidp/dex/connector"
)

// Config holds the configuration parameters for a connector which returns an
// identity with the HTTP header X-Remote-User as verified email,
// X-Remote-Group and configured staticGroups as user's group.
// Headers retrieved to fetch user's email and group can be configured
// with userHeader and groupHeader.
type Config struct {
	UserIDHeader   string   `json:"userIDHeader"`
	UserHeader     string   `json:"userHeader"`
	UserNameHeader string   `json:"userNameHeader"`
	EmailHeader    string   `json:"emailHeader"`
	GroupHeader    string   `json:"groupHeader"`
	Groups         []string `json:"staticGroups"`
}

// Open returns an authentication strategy which requires no user interaction.
func (c *Config) Open(id string, logger *slog.Logger) (connector.Connector, error) {
	userIDHeader := c.UserIDHeader
	if userIDHeader == "" {
		userIDHeader = "X-Remote-User-Id"
	}
	userHeader := c.UserHeader
	if userHeader == "" {
		userHeader = "X-Remote-User"
	}
	userNameHeader := c.UserNameHeader
	if userNameHeader == "" {
		userNameHeader = "X-Remote-User-Name"
	}
	emailHeader := c.EmailHeader
	if emailHeader == "" {
		emailHeader = "X-Remote-User-Email"
	}
	groupHeader := c.GroupHeader
	if groupHeader == "" {
		groupHeader = "X-Remote-Group"
	}

	return &callback{
		userIDHeader:   userIDHeader,
		userHeader:     userHeader,
		userNameHeader: userNameHeader,
		emailHeader:    emailHeader,
		groupHeader:    groupHeader,
		groups:         c.Groups,
		logger:         logger.With(slog.Group("connector", "type", "authproxy", "id", id)),
		pathSuffix:     "/" + id,
	}, nil
}

// Callback is a connector which returns an identity with the HTTP header
// X-Remote-User as verified email.
type callback struct {
	userIDHeader   string
	userNameHeader string
	userHeader     string
	emailHeader    string
	groupHeader    string
	groups         []string
	logger         *slog.Logger
	pathSuffix     string
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
	remoteUser := r.Header.Get(m.userHeader)
	if remoteUser == "" {
		return connector.Identity{}, fmt.Errorf("required HTTP header %s is not set", m.userHeader)
	}
	remoteUserName := r.Header.Get(m.userNameHeader)
	if remoteUserName == "" {
		remoteUserName = remoteUser
	}
	remoteUserID := r.Header.Get(m.userIDHeader)
	if remoteUserID == "" {
		remoteUserID = remoteUser
	}
	remoteUserEmail := r.Header.Get(m.emailHeader)
	if remoteUserEmail == "" {
		remoteUserEmail = remoteUser
	}
	groups := m.groups
	headerGroup := r.Header.Get(m.groupHeader)
	if headerGroup != "" {
		splitheaderGroup := strings.Split(headerGroup, ",")
		for i, v := range splitheaderGroup {
			splitheaderGroup[i] = strings.TrimSpace(v)
		}
		groups = append(splitheaderGroup, groups...)
	}
	return connector.Identity{
		UserID:            remoteUserID,
		Username:          remoteUser,
		PreferredUsername: remoteUserName,
		Email:             remoteUserEmail,
		EmailVerified:     true,
		Groups:            groups,
	}, nil
}
