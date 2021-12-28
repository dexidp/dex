package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/dexidp/dex/connector/external/sdk"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func filter(given, required []string) []string {
	groups := []string{}
	groupFilter := make(map[string]struct{})
	for _, group := range required {
		groupFilter[group] = struct{}{}
	}
	for _, group := range given {
		if _, ok := groupFilter[group]; ok {
			groups = append(groups, group)
		}
	}
	return groups
}

const (
	// read operations of the /api/v4/user endpoint
	scopeUser = "read_user"
	// used to retrieve groups from /oauth/userinfo
	// https://docs.gitlab.com/ee/integration/openid_connect_provider.html
	scopeOpenID = "openid"
)

type gitlabUser struct {
	ID       int
	Name     string
	Username string
	State    string
	Email    string
	IsAdmin  bool
}

type connectorData struct {
	// GitLab's OAuth2 tokens never expire. We don't need a refresh token.
	AccessToken string `json:"accessToken"`
}

type gitlabConnector struct {
	sdk.UnimplementedCallbackConnectorServer

	baseURL      string
	redirectURI  string
	groups       []string
	clientID     string
	clientSecret string
	httpClient   *http.Client
	// if set to true will use the user's handle rather than their numeric id as the ID
	useLoginAsID bool
}

func (c *gitlabConnector) oauth2Config(scopes *sdk.Scopes) *oauth2.Config {
	gitlabScopes := []string{scopeUser}
	if c.groupsRequired(scopes.Groups) {
		gitlabScopes = []string{scopeUser, scopeOpenID}
	}

	gitlabEndpoint := oauth2.Endpoint{AuthURL: c.baseURL + "/oauth/authorize", TokenURL: c.baseURL + "/oauth/token"}
	return &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     gitlabEndpoint,
		Scopes:       gitlabScopes,
		RedirectURL:  c.redirectURI,
	}
}

func (c *gitlabConnector) LoginURL(_ context.Context, req *sdk.LoginURLReq) (*sdk.LoginURLResp, error) {
	if c.redirectURI != req.CallbackUrl {
		return nil, fmt.Errorf("expected callback URL %q did not match the URL in the config %q", c.redirectURI, req.CallbackUrl)
	}
	return &sdk.LoginURLResp{Url: c.oauth2Config(req.Scopes).AuthCodeURL(req.State)}, nil
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

func (c *gitlabConnector) HandleCallback(ctx context.Context, req *sdk.CallbackReq) (resp *sdk.CallbackResp, err error) {
	q, _ := url.ParseQuery(req.RawQuery)

	if errType := q.Get("error"); errType != "" {
		return nil, &oauth2Error{errType, q.Get("error_description")}
	}

	oauth2Config := c.oauth2Config(req.Scopes)
	if c.httpClient != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	}

	token, err := oauth2Config.Exchange(ctx, q.Get("code"))
	if err != nil {
		return nil, fmt.Errorf("gitlab: failed to get token: %v", err)
	}

	client := oauth2Config.Client(ctx, token)

	user, err := c.user(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("gitlab: get user: %v", err)
	}

	username := user.Name
	if username == "" {
		username = user.Email
	}

	identity := &sdk.Identity{
		UserId:            strconv.Itoa(user.ID),
		Username:          username,
		PreferredUsername: user.Username,
		Email:             user.Email,
		EmailVerified:     true,
	}
	if c.useLoginAsID {
		identity.UserId = user.Username
	}

	if c.groupsRequired(req.Scopes.Groups) {
		groups, err := c.getGroups(ctx, client, req.Scopes.Groups, user.Username)
		if err != nil {
			return &sdk.CallbackResp{Identity: identity}, fmt.Errorf("gitlab: get groups: %v", err)
		}
		identity.Groups = groups
	}

	if req.Scopes.OfflineAccess {
		data := connectorData{AccessToken: token.AccessToken}
		connData, err := json.Marshal(data)
		if err != nil {
			return &sdk.CallbackResp{Identity: identity}, fmt.Errorf("marshal connector data: %v", err)
		}
		identity.ConnectorData = connData
	}

	return &sdk.CallbackResp{Identity: identity}, nil
}

func (c *gitlabConnector) Refresh(ctx context.Context, req *sdk.RefreshReq) (*sdk.RefreshResp, error) {
	if len(req.Identity.ConnectorData) == 0 {
		return &sdk.RefreshResp{Identity: req.Identity}, errors.New("no upstream access token found")
	}

	var data connectorData
	if err := json.Unmarshal(req.Identity.ConnectorData, &data); err != nil {
		return &sdk.RefreshResp{Identity: req.Identity}, fmt.Errorf("gitlab: unmarshal access token: %v", err)
	}

	client := c.oauth2Config(req.Scopes).Client(ctx, &oauth2.Token{AccessToken: data.AccessToken})
	user, err := c.user(ctx, client)
	if err != nil {
		return &sdk.RefreshResp{Identity: req.Identity}, fmt.Errorf("gitlab: get user: %v", err)
	}

	username := user.Name
	if username == "" {
		username = user.Email
	}
	req.Identity.Username = username
	req.Identity.PreferredUsername = user.Username
	req.Identity.Email = user.Email

	if c.groupsRequired(req.Scopes.Groups) {
		groups, err := c.getGroups(ctx, client, req.Scopes.Groups, user.Username)
		if err != nil {
			return &sdk.RefreshResp{Identity: req.Identity}, fmt.Errorf("gitlab: get groups: %v", err)
		}
		req.Identity.Groups = groups
	}
	return &sdk.RefreshResp{Identity: req.Identity}, nil
}

func (c *gitlabConnector) groupsRequired(groupScope bool) bool {
	return len(c.groups) > 0 || groupScope
}

// user queries the GitLab API for profile information using the provided client. The HTTP
// client is expected to be constructed by the golang.org/x/oauth2 package, which inserts
// a bearer token as part of the request.
func (c *gitlabConnector) user(ctx context.Context, client *http.Client) (gitlabUser, error) {
	var u gitlabUser
	req, err := http.NewRequest("GET", c.baseURL+"/api/v4/user", nil)
	if err != nil {
		return u, fmt.Errorf("gitlab: new req: %v", err)
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return u, fmt.Errorf("gitlab: get URL %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return u, fmt.Errorf("gitlab: read body: %v", err)
		}
		return u, fmt.Errorf("%s: %s", resp.Status, body)
	}

	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return u, fmt.Errorf("failed to decode response: %v", err)
	}
	return u, nil
}

type userInfo struct {
	Groups []string
}

// userGroups queries the GitLab API for group membership.
//
// The HTTP passed client is expected to be constructed by the golang.org/x/oauth2 package,
// which inserts a bearer token as part of the request.
func (c *gitlabConnector) userGroups(ctx context.Context, client *http.Client) ([]string, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/oauth/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("gitlab: new req: %v", err)
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitlab: get URL %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gitlab: read body: %v", err)
		}
		return nil, fmt.Errorf("%s: %s", resp.Status, body)
	}
	var u userInfo
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return u.Groups, nil
}

func (c *gitlabConnector) getGroups(ctx context.Context, client *http.Client, groupScope bool, userLogin string) ([]string, error) {
	gitlabGroups, err := c.userGroups(ctx, client)
	if err != nil {
		return nil, err
	}

	if len(c.groups) > 0 {
		filteredGroups := filter(gitlabGroups, c.groups)
		if len(filteredGroups) == 0 {
			return nil, fmt.Errorf("gitlab: user %q is not in any of the required groups", userLogin)
		}
		return filteredGroups, nil
	} else if groupScope {
		return gitlabGroups, nil
	}

	return nil, nil
}

func main() {
	var (
		listenAddress string
		tlsCert       string
		tlsKey        string
	)

	connector := &gitlabConnector{}

	flag.StringVar(&listenAddress, "listen-address", "127.0.0.1:5571", "Address to listen on")
	flag.StringVar(&connector.baseURL, "gitlab.base-url", "https://gitlab.com", "Gitlab URL")
	flag.StringVar(&connector.clientID, "gitlab.client-id", "", "Gitlab application client ID")
	flag.StringVar(&connector.clientSecret, "gitlab.client-secret", "", "Gitlab application client secret")
	flag.StringVar(&connector.redirectURI, "gitlab.redirect-url", "http://127.0.0.1:5556/dex/callback", "Redirect URL for receiving callbacks")

	flag.StringVar(&tlsCert, "tls-cert", "examples/external-gitlab/server.crt", "SSL certificate")
	flag.StringVar(&tlsKey, "tls-key", "examples/external-gitlab/server.key", "SSL certificate key")

	flag.Parse()

	grpcListener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatalln(err.Error())
	}

	cert, err := tls.LoadX509KeyPair(tlsCert, tlsKey)
	if err != nil {
		log.Fatalf("invalid config: error parsing gRPC certificate file: %v", err)
	}

	tlsConfig := tls.Config{
		Certificates:             []tls.Certificate{cert},
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
	}

	grpcSrv := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tlsConfig)))

	fmt.Printf("Connector started on: %s\n", listenAddress)

	sdk.RegisterCallbackConnectorServer(grpcSrv, connector)
	if err := grpcSrv.Serve(grpcListener); err != nil {
		log.Fatalln(err.Error())
	}
}
