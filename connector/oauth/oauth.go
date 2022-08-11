package oauth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
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
	userIDKey            string
	userNameKey          string
	preferredUsernameKey string
	emailKey             string
	emailVerifiedKey     string
	groupsKey            string
	httpClient           *http.Client
	logger               log.Logger
}

type connectorData struct {
	AccessToken string
}

type Config struct {
	ClientID           string   `json:"clientID"`
	ClientSecret       string   `json:"clientSecret"`
	RedirectURI        string   `json:"redirectURI"`
	TokenURL           string   `json:"tokenURL"`
	AuthorizationURL   string   `json:"authorizationURL"`
	UserInfoURL        string   `json:"userInfoURL"`
	Scopes             []string `json:"scopes"`
	RootCAs            []string `json:"rootCAs"`
	InsecureSkipVerify bool     `json:"insecureSkipVerify"`
	UserIDKey          string   `json:"userIDKey"` // defaults to "id"
	ClaimMapping       struct {
		UserNameKey          string `json:"userNameKey"`          // defaults to "user_name"
		PreferredUsernameKey string `json:"preferredUsernameKey"` // defaults to "preferred_username"
		GroupsKey            string `json:"groupsKey"`            // defaults to "groups"
		EmailKey             string `json:"emailKey"`             // defaults to "email"
		EmailVerifiedKey     string `json:"emailVerifiedKey"`     // defaults to "email_verified"
	} `json:"claimMapping"`
}

func (c *Config) Open(id string, logger log.Logger) (connector.Connector, error) {
	var err error

	userIDKey := c.UserIDKey
	if userIDKey == "" {
		userIDKey = "id"
	}

	userNameKey := c.ClaimMapping.UserNameKey
	if userNameKey == "" {
		userNameKey = "user_name"
	}

	preferredUsernameKey := c.ClaimMapping.PreferredUsernameKey
	if preferredUsernameKey == "" {
		preferredUsernameKey = "preferred_username"
	}

	groupsKey := c.ClaimMapping.GroupsKey
	if groupsKey == "" {
		groupsKey = "groups"
	}

	emailKey := c.ClaimMapping.EmailKey
	if emailKey == "" {
		emailKey = "email"
	}

	emailVerifiedKey := c.ClaimMapping.EmailVerifiedKey
	if emailVerifiedKey == "" {
		emailVerifiedKey = "email_verified"
	}

	oauthConn := &oauthConnector{
		clientID:             c.ClientID,
		clientSecret:         c.ClientSecret,
		tokenURL:             c.TokenURL,
		authorizationURL:     c.AuthorizationURL,
		userInfoURL:          c.UserInfoURL,
		scopes:               c.Scopes,
		redirectURI:          c.RedirectURI,
		logger:               logger,
		userIDKey:            userIDKey,
		userNameKey:          userNameKey,
		preferredUsernameKey: preferredUsernameKey,
		groupsKey:            groupsKey,
		emailKey:             emailKey,
		emailVerifiedKey:     emailVerifiedKey,
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
		rootCABytes, err := os.ReadFile(rootCA)
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
	defer userInfoResp.Body.Close()

	if userInfoResp.StatusCode != http.StatusOK {
		return identity, fmt.Errorf("OAuth Connector: failed to execute request to userinfo: status %d", userInfoResp.StatusCode)
	}

	var userInfoResult map[string]interface{}
	err = json.NewDecoder(userInfoResp.Body).Decode(&userInfoResult)
	if err != nil {
		return identity, fmt.Errorf("OAuth Connector: failed to parse userinfo: %v", err)
	}

	userID, found := userInfoResult[c.userIDKey]
	if !found {
		return identity, fmt.Errorf("OAuth Connector: not found %v claim", c.userIDKey)
	}

	switch userID.(type) {
	case float64, int64, string:
		identity.UserID = fmt.Sprintf("%v", userID)
	default:
		return identity, fmt.Errorf("OAuth Connector: %v claim should be string or number, got %T", c.userIDKey, userID)
	}

	identity.Username, _ = userInfoResult[c.userNameKey].(string)
	identity.PreferredUsername, _ = userInfoResult[c.preferredUsernameKey].(string)
	identity.Email, _ = userInfoResult[c.emailKey].(string)
	identity.EmailVerified, _ = userInfoResult[c.emailVerifiedKey].(bool)

	if s.Groups {
		groups := map[string]struct{}{}

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

func (c *oauthConnector) addGroupsFromMap(groups map[string]struct{}, result map[string]interface{}) error {
	groupsClaim, ok := result[c.groupsKey].([]interface{})
	if !ok {
		return errors.New("cannot convert to slice")
	}

	for _, group := range groupsClaim {
		if groupString, ok := group.(string); ok {
			groups[groupString] = struct{}{}
		}
		if groupMap, ok := group.(map[string]interface{}); ok {
			if groupName, ok := groupMap["name"].(string); ok {
				groups[groupName] = struct{}{}
			}
		}
	}

	return nil
}

func (c *oauthConnector) addGroupsFromToken(groups map[string]struct{}, token string) error {
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
