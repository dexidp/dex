package oauth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/log"
)

type oauthConnector struct {
	clientID             string
	clientSecret         string
	redirectURI          string
	tokenURL             string
	authorizationURL     string
	userInfoURL          string
	scopes               []string
	groupsKey            string
	userIDKey            string
	userNameKey          string
	preferredUsernameKey string
	httpClient           *http.Client
	logger               log.Logger
}

type connectorData struct {
	AccessToken string
}

type Config struct {
	ClientID             string   `json:"clientID"`
	ClientSecret         string   `json:"clientSecret"`
	RedirectURI          string   `json:"redirectURI"`
	TokenURL             string   `json:"tokenURL"`
	AuthorizationURL     string   `json:"authorizationURL"`
	UserInfoURL          string   `json:"userInfoURL"`
	Scopes               []string `json:"scopes"`
	GroupsKey            string   `json:"groupsKey"`
	UserIDKey            string   `json:"userIDKey"`
	UserNameKey          string   `json:"userNameKey"`
	PreferredUsernameKey string   `json:"preferredUsernameKey"`
	RootCAs              []string `json:"rootCAs"`
	InsecureSkipVerify   bool     `json:"insecureSkipVerify"`
}

func (c *Config) Open(id string, logger log.Logger) (connector.Connector, error) {
	var err error

	oauthConn := &oauthConnector{
		clientID:         c.ClientID,
		clientSecret:     c.ClientSecret,
		tokenURL:         c.TokenURL,
		authorizationURL: c.AuthorizationURL,
		userInfoURL:      c.UserInfoURL,
		scopes:           c.Scopes,
		groupsKey:        c.GroupsKey,
		userIDKey:        c.UserIDKey,
		userNameKey:      c.UserNameKey,
		redirectURI:      c.RedirectURI,
		logger:           logger,
	}

	oauthConn.httpClient, err = newHTTPClient(c.RootCAs, c.InsecureSkipVerify)
	if err != nil {
		return nil, err
	}

	return oauthConn, err
}

func newHTTPClient(rootCAs []string, insecureSkipVerify bool) (*http.Client, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	tlsConfig := tls.Config{RootCAs: pool, InsecureSkipVerify: insecureSkipVerify}
	for _, rootCA := range rootCAs {
		rootCABytes, err := ioutil.ReadFile(rootCA)
		if err != nil {
			return nil, fmt.Errorf("failed to read root-ca: %v", err)
		}
		if !tlsConfig.RootCAs.AppendCertsFromPEM(rootCABytes) {
			return nil, fmt.Errorf("no certs found in root CA file %q", rootCA)
		}
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tlsConfig,
			Proxy:           http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}, nil
}

func (c *oauthConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, error) {
	if c.redirectURI != callbackURL {
		return "", fmt.Errorf("expected callback URL %q did not match the URL in the config %q", callbackURL, c.redirectURI)
	}

	oauth2Config := &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     oauth2.Endpoint{TokenURL: c.tokenURL, AuthURL: c.authorizationURL},
		RedirectURL:  c.redirectURI,
		Scopes:       c.scopes,
	}

	return oauth2Config.AuthCodeURL(state), nil
}

func (c *oauthConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return identity, errors.New(q.Get("error_description"))
	}

	oauth2Config := &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     oauth2.Endpoint{TokenURL: c.tokenURL, AuthURL: c.authorizationURL},
		RedirectURL:  c.redirectURI,
		Scopes:       c.scopes,
	}

	ctx := context.WithValue(r.Context(), oauth2.HTTPClient, c.httpClient)

	token, err := oauth2Config.Exchange(ctx, q.Get("code"))
	if err != nil {
		return identity, fmt.Errorf("OAuth connector: failed to get token: %v", err)
	}

	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	userInfoResp, err := client.Get(c.userInfoURL)
	if err != nil {
		return identity, fmt.Errorf("OAuth Connector: failed to execute request to userinfo: %v", err)
	}

	if userInfoResp.StatusCode != http.StatusOK {
		return identity, fmt.Errorf("OAuth Connector: failed to execute request to userinfo: status %d", userInfoResp.StatusCode)
	}

	defer userInfoResp.Body.Close()

	var userInfoResult map[string]interface{}
	err = json.NewDecoder(userInfoResp.Body).Decode(&userInfoResult)

	if err != nil {
		return identity, fmt.Errorf("OAuth Connector: failed to parse userinfo: %v", err)
	}

	if c.userIDKey == "" {
		c.userIDKey = "user_id"
	}

	if c.userNameKey == "" {
		c.userNameKey = "user_name"
	}

	if c.groupsKey == "" {
		c.groupsKey = "groups"
	}

	if c.preferredUsernameKey == "" {
		c.preferredUsernameKey = "preferred_username"
	}

	identity.UserID, _ = userInfoResult[c.userIDKey].(string)
	identity.Username, _ = userInfoResult[c.userNameKey].(string)
	identity.PreferredUsername, _ = userInfoResult[c.preferredUsernameKey].(string)
	identity.Email, _ = userInfoResult["email"].(string)
	identity.EmailVerified, _ = userInfoResult["email_verified"].(bool)

	if s.Groups {
		groups := map[string]bool{}

		c.addGroupsFromMap(groups, userInfoResult)
		c.addGroupsFromToken(groups, token.AccessToken)

		for groupName := range groups {
			identity.Groups = append(identity.Groups, groupName)
		}
	}

	if s.OfflineAccess {
		data := connectorData{AccessToken: token.AccessToken}
		connData, err := json.Marshal(data)
		if err != nil {
			return identity, fmt.Errorf("OAuth Connector: failed to parse connector data for offline access: %v", err)
		}
		identity.ConnectorData = connData
	}

	return identity, nil
}

func (c *oauthConnector) addGroupsFromMap(groups map[string]bool, result map[string]interface{}) error {
	groupsClaim, ok := result[c.groupsKey].([]interface{})
	if !ok {
		return errors.New("cant convert to array")
	}

	for _, group := range groupsClaim {
		if groupString, ok := group.(string); ok {
			groups[groupString] = true
		}
	}

	return nil
}

func (c *oauthConnector) addGroupsFromToken(groups map[string]bool, token string) error {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return errors.New("invalid token")
	}

	decoded, err := decode(parts[1])
	if err != nil {
		return err
	}

	var claimsMap map[string]interface{}
	err = json.Unmarshal(decoded, &claimsMap)
	if err != nil {
		return err
	}

	return c.addGroupsFromMap(groups, claimsMap)
}

func decode(seg string) ([]byte, error) {
	if l := len(seg) % 4; l > 0 {
		seg += strings.Repeat("=", 4-l)
	}

	return base64.URLEncoding.DecodeString(seg)
}
