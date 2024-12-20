// Package cas provides authentication strategies using CAS.
package cas

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/dexidp/dex/connector"
	"github.com/pkg/errors"
	"gopkg.in/cas.v2"
)

// Config holds configuration options for CAS logins.
type Config struct {
	Portal  string            `json:"portal"`
	Mapping map[string]string `json:"mapping"`
}

// Open returns a strategy for logging in through CAS.
func (c *Config) Open(id string, logger *slog.Logger) (connector.Connector, error) {
	casURL, err := url.Parse(c.Portal)
	if err != nil {
		return "", fmt.Errorf("failed to parse casURL %q: %v", c.Portal, err)
	}
	return &casConnector{
		client:     http.DefaultClient,
		portal:     casURL,
		mapping:    c.Mapping,
		logger:     logger.With(slog.Group("connector", "type", "cas", "id", id)),
		pathSuffix: "/" + id,
	}, nil
}

var _ connector.CallbackConnector = (*casConnector)(nil)

type casConnector struct {
	client     *http.Client
	portal     *url.URL
	mapping    map[string]string
	logger     *slog.Logger
	pathSuffix string
}

// LoginURL returns the URL to redirect the user to login with.
func (m *casConnector) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
	u, err := url.Parse(callbackURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse callbackURL %q: %v", callbackURL, err)
	}
	u.Path += m.pathSuffix
	// context = $callbackURL + $m.pathSuffix
	v := u.Query()
	v.Set("context", u.String()) // without query params
	v.Set("state", state)
	u.RawQuery = v.Encode()

	loginURL := *m.portal
	loginURL.Path += "/login"
	// encode service url to context, which used in `HandleCallback`
	// service = $callbackURL + $m.pathSuffix ? state=$state & context=$callbackURL + $m.pathSuffix
	q := loginURL.Query()
	q.Set("service", u.String()) // service = ...?state=...&context=...
	loginURL.RawQuery = q.Encode()
	return loginURL.String(), nil
}

// HandleCallback parses the request and returns the user's identity
func (m *casConnector) HandleCallback(s connector.Scopes, r *http.Request) (connector.Identity, error) {
	state := r.URL.Query().Get("state")
	ticket := r.URL.Query().Get("ticket")
	// service=context = $callbackURL + $m.pathSuffix
	serviceURL, err := url.Parse(r.URL.Query().Get("context"))
	if err != nil {
		return connector.Identity{}, fmt.Errorf("failed to parse serviceURL %q: %v", r.URL.Query().Get("context"), err)
	}
	// service = $callbackURL + $m.pathSuffix ? state=$state & context=$callbackURL + $m.pathSuffix
	q := serviceURL.Query()
	q.Set("context", serviceURL.String())
	q.Set("state", state)
	serviceURL.RawQuery = q.Encode()

	user, err := m.getCasUserByTicket(ticket, serviceURL)
	if err != nil {
		return connector.Identity{}, err
	}
	m.logger.Info("cas user", "user", user)
	return user, nil
}

func (m *casConnector) getCasUserByTicket(ticket string, serviceURL *url.URL) (connector.Identity, error) {
	id := connector.Identity{}
	// validate ticket
	validator := cas.NewServiceTicketValidator(m.client, m.portal)
	resp, err := validator.ValidateTicket(serviceURL, ticket)
	if err != nil {
		return id, errors.Wrapf(err, "failed to validate ticket via %q with ticket %q", serviceURL, ticket)
	}
	// fill identity
	id.UserID = resp.User
	id.Groups = resp.MemberOf
	if len(m.mapping) == 0 {
		return id, nil
	}
	if username, ok := m.mapping["username"]; ok {
		id.Username = resp.Attributes.Get(username)
		if id.Username == "" && username == "userid" {
			id.Username = resp.User
		}
	}
	if preferredUsername, ok := m.mapping["preferred_username"]; ok {
		id.PreferredUsername = resp.Attributes.Get(preferredUsername)
		if id.PreferredUsername == "" && preferredUsername == "userid" {
			id.PreferredUsername = resp.User
		}
	}
	if email, ok := m.mapping["email"]; ok {
		id.Email = resp.Attributes.Get(email)
		if id.Email != "" {
			id.EmailVerified = true
		}
	}
	// override memberOf
	if groups, ok := m.mapping["groups"]; ok {
		id.Groups = resp.Attributes[groups]
	}
	return id, nil
}
