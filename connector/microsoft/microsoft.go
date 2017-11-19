// Package microsoft provides authentication strategies using Microsoft.
package microsoft

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/coreos/dex/connector"
	"github.com/sirupsen/logrus"
)

const (
	apiURL = "https://graph.microsoft.com"
	// Microsoft requires this scope to access user's profile
	scopeUser = "user.read"
)

// Config holds configuration options for microsoft logins.
type Config struct {
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	RedirectURI  string `json:"redirectURI"`
	Tenant       string `json:"tenant"`
}

// Open returns a strategy for logging in through Microsoft.
func (c *Config) Open(id string, logger logrus.FieldLogger) (connector.Connector, error) {
	m := microsoftConnector{
		redirectURI:  c.RedirectURI,
		clientID:     c.ClientID,
		clientSecret: c.ClientSecret,
		tenant:       c.Tenant,
		logger:       logger,
	}
	// By default allow logins from both personal and business/school
	// accounts.
	if m.tenant == "" {
		m.tenant = "common"
	}

	return &m, nil
}

type connectorData struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	Expiry       time.Time `json:"expiry"`
}

var (
	_ connector.CallbackConnector = (*microsoftConnector)(nil)
	_ connector.RefreshConnector  = (*microsoftConnector)(nil)
)

type microsoftConnector struct {
	redirectURI  string
	clientID     string
	clientSecret string
	tenant       string
	logger       logrus.FieldLogger
}

func (c *microsoftConnector) oauth2Config(scopes connector.Scopes) *oauth2.Config {
	microsoftScopes := []string{scopeUser}

	return &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://login.microsoftonline.com/" + c.tenant + "/oauth2/v2.0/authorize",
			TokenURL: "https://login.microsoftonline.com/" + c.tenant + "/oauth2/v2.0/token",
		},
		Scopes:      microsoftScopes,
		RedirectURL: c.redirectURI,
	}
}

func (c *microsoftConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, error) {
	if c.redirectURI != callbackURL {
		return "", fmt.Errorf("expected callback URL %q did not match the URL in the config %q", callbackURL, c.redirectURI)
	}

	return c.oauth2Config(scopes).AuthCodeURL(state), nil
}

func (c *microsoftConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return identity, &oauth2Error{errType, q.Get("error_description")}
	}

	oauth2Config := c.oauth2Config(s)

	ctx := r.Context()

	token, err := oauth2Config.Exchange(ctx, q.Get("code"))
	if err != nil {
		return identity, fmt.Errorf("microsoft: failed to get token: %v", err)
	}

	client := oauth2Config.Client(ctx, token)

	user, err := c.user(ctx, client)
	if err != nil {
		return identity, fmt.Errorf("microsoft: get user: %v", err)
	}

	identity = connector.Identity{
		UserID:        user.ID,
		Username:      user.Name,
		Email:         user.Email,
		EmailVerified: true,
	}

	if s.OfflineAccess {
		data := connectorData{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			Expiry:       token.Expiry,
		}
		connData, err := json.Marshal(data)
		if err != nil {
			return identity, fmt.Errorf("microsoft: marshal connector data: %v", err)
		}
		identity.ConnectorData = connData
	}

	return identity, nil
}

type tokenNotifyFunc func(*oauth2.Token) error

// notifyRefreshTokenSource is essentially `oauth2.ResuseTokenSource` with `TokenNotifyFunc` added.
type notifyRefreshTokenSource struct {
	new oauth2.TokenSource
	mu  sync.Mutex // guards t
	t   *oauth2.Token
	f   tokenNotifyFunc // called when token refreshed so new refresh token can be persisted
}

// Token returns the current token if it's still valid, else will
// refresh the current token (using r.Context for HTTP client
// information) and return the new one.
func (s *notifyRefreshTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.t.Valid() {
		return s.t, nil
	}
	t, err := s.new.Token()
	if err != nil {
		return nil, err
	}
	s.t = t
	return t, s.f(t)
}

func (c *microsoftConnector) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	if len(identity.ConnectorData) == 0 {
		return identity, errors.New("microsoft: no upstream access token found")
	}

	var data connectorData
	if err := json.Unmarshal(identity.ConnectorData, &data); err != nil {
		return identity, fmt.Errorf("microsoft: unmarshal access token: %v", err)
	}
	tok := &oauth2.Token{
		AccessToken:  data.AccessToken,
		RefreshToken: data.RefreshToken,
		Expiry:       data.Expiry,
	}

	client := oauth2.NewClient(ctx, &notifyRefreshTokenSource{
		new: c.oauth2Config(s).TokenSource(ctx, tok),
		t:   tok,
		f: func(tok *oauth2.Token) error {
			data := connectorData{
				AccessToken:  tok.AccessToken,
				RefreshToken: tok.RefreshToken,
				Expiry:       tok.Expiry,
			}
			connData, err := json.Marshal(data)
			if err != nil {
				return fmt.Errorf("microsoft: marshal connector data: %v", err)
			}
			identity.ConnectorData = connData
			return nil
		},
	})
	user, err := c.user(ctx, client)
	if err != nil {
		return identity, fmt.Errorf("microsoft: get user: %v", err)
	}

	identity.Username = user.Name
	identity.Email = user.Email

	return identity, nil
}

// https://developer.microsoft.com/en-us/graph/docs/api-reference/v1.0/resources/user
// id                - The unique identifier for the user. Inherited from
//                     directoryObject. Key. Not nullable. Read-only.
// displayName       - The name displayed in the address book for the user.
//                     This is usually the combination of the user's first name,
//                     middle initial and last name. This property is required
//                     when a user is created and it cannot be cleared during
//                     updates. Supports $filter and $orderby.
// userPrincipalName - The user principal name (UPN) of the user.
//                     The UPN is an Internet-style login name for the user
//                     based on the Internet standard RFC 822. By convention,
//                     this should map to the user's email name. The general
//                     format is alias@domain, where domain must be present in
//                     the tenant’s collection of verified domains. This
//                     property is required when a user is created. The
//                     verified domains for the tenant can be accessed from the
//                     verifiedDomains property of organization. Supports
//                     $filter and $orderby.
type user struct {
	ID    string `json:"id"`
	Name  string `json:"displayName"`
	Email string `json:"userPrincipalName"`
}

func (c *microsoftConnector) user(ctx context.Context, client *http.Client) (u user, err error) {
	// https://developer.microsoft.com/en-us/graph/docs/api-reference/v1.0/api/user_get
	req, err := http.NewRequest("GET", apiURL+"/v1.0/me?$select=id,displayName,userPrincipalName", nil)
	if err != nil {
		return u, fmt.Errorf("new req: %v", err)
	}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return u, fmt.Errorf("get URL %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// https://developer.microsoft.com/en-us/graph/docs/concepts/errors
		var ge graphError
		if err := json.NewDecoder(resp.Body).Decode(&struct {
			Error *graphError `json:"error"`
		}{&ge}); err != nil {
			return u, fmt.Errorf("JSON error decode: %v", err)
		}
		return u, &ge
	}

	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return u, fmt.Errorf("JSON decode: %v", err)
	}

	return u, err
}

type graphError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *graphError) Error() string {
	return e.Code + ": " + e.Message
}

type oauth2Error struct {
	error            string
	errorDescription string
}

func (e *oauth2Error) Error() string {
	if e.errorDescription == "" {
		return e.error
	}
	return e.error + ": " + e.errorDescription
}
