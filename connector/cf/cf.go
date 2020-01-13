package cf

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/log"
)

type cfConnector struct {
	clientID         string
	clientSecret     string
	redirectURI      string
	apiURL           string
	tokenURL         string
	authorizationURL string
	userInfoURL      string
	httpClient       *http.Client
	logger           log.Logger
}

type connectorData struct {
	AccessToken string
}

type Config struct {
	ClientID           string   `json:"clientID"`
	ClientSecret       string   `json:"clientSecret"`
	RedirectURI        string   `json:"redirectURI"`
	APIURL             string   `json:"apiURL"`
	RootCAs            []string `json:"rootCAs"`
	InsecureSkipVerify bool     `json:"insecureSkipVerify"`
}

type CCResponse struct {
	NextURL      string     `json:"next_url"`
	Resources    []Resource `json:"resources"`
	TotalResults int        `json:"total_results"`
}

type Resource struct {
	Metadata Metadata `json:"metadata"`
	Entity   Entity   `json:"entity"`
}

type Metadata struct {
	GUID string `json:"guid"`
}

type Entity struct {
	Name             string `json:"name"`
	OrganizationGUID string `json:"organization_guid"`
}

type Space struct {
	Name    string
	GUID    string
	OrgGUID string
	Role    string
}

type Org struct {
	Name string
	GUID string
}

func (c *Config) Open(id string, logger log.Logger) (connector.Connector, error) {
	var err error

	cfConn := &cfConnector{
		clientID:     c.ClientID,
		clientSecret: c.ClientSecret,
		apiURL:       c.APIURL,
		redirectURI:  c.RedirectURI,
		logger:       logger,
	}

	cfConn.httpClient, err = newHTTPClient(c.RootCAs, c.InsecureSkipVerify)
	if err != nil {
		return nil, err
	}

	apiURL := strings.TrimRight(c.APIURL, "/")
	apiResp, err := cfConn.httpClient.Get(fmt.Sprintf("%s/v2/info", apiURL))
	if err != nil {
		logger.Errorf("failed-to-send-request-to-cloud-controller-api", err)
		return nil, err
	}

	defer apiResp.Body.Close()

	if apiResp.StatusCode != http.StatusOK {
		err = fmt.Errorf("request failed with status %d", apiResp.StatusCode)
		logger.Errorf("failed-get-info-response-from-api", err)
		return nil, err
	}

	var apiResult map[string]interface{}
	json.NewDecoder(apiResp.Body).Decode(&apiResult)

	uaaURL := strings.TrimRight(apiResult["authorization_endpoint"].(string), "/")
	uaaResp, err := cfConn.httpClient.Get(fmt.Sprintf("%s/.well-known/openid-configuration", uaaURL))
	if err != nil {
		logger.Errorf("failed-to-send-request-to-uaa-api", err)
		return nil, err
	}

	if apiResp.StatusCode != http.StatusOK {
		err = fmt.Errorf("request failed with status %d", apiResp.StatusCode)
		logger.Errorf("failed-to-get-well-known-config-response-from-api", err)
		return nil, err
	}

	defer uaaResp.Body.Close()

	var uaaResult map[string]interface{}
	err = json.NewDecoder(uaaResp.Body).Decode(&uaaResult)

	if err != nil {
		logger.Errorf("failed-to-decode-response-from-uaa-api", err)
		return nil, err
	}

	cfConn.tokenURL, _ = uaaResult["token_endpoint"].(string)
	cfConn.authorizationURL, _ = uaaResult["authorization_endpoint"].(string)
	cfConn.userInfoURL, _ = uaaResult["userinfo_endpoint"].(string)

	return cfConn, err
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

func (c *cfConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, error) {
	if c.redirectURI != callbackURL {
		return "", fmt.Errorf("expected callback URL %q did not match the URL in the config %q", callbackURL, c.redirectURI)
	}

	oauth2Config := &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     oauth2.Endpoint{TokenURL: c.tokenURL, AuthURL: c.authorizationURL},
		RedirectURL:  c.redirectURI,
		Scopes:       []string{"openid", "cloud_controller.read"},
	}

	return oauth2Config.AuthCodeURL(state), nil
}

func fetchRoleSpaces(baseURL, path, role string, client *http.Client) ([]Space, error) {
	var spaces []Space

	resources, err := fetchResources(baseURL, path, client)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch resources: %v", err)
	}

	for _, resource := range resources {
		spaces = append(spaces, Space{
			Name:    resource.Entity.Name,
			GUID:    resource.Metadata.GUID,
			OrgGUID: resource.Entity.OrganizationGUID,
			Role:    role,
		})
	}

	return spaces, nil
}

func fetchOrgs(baseURL, path string, client *http.Client) ([]Org, error) {
	var orgs []Org

	resources, err := fetchResources(baseURL, path, client)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch resources: %v", err)
	}

	for _, resource := range resources {
		orgs = append(orgs, Org{
			Name: resource.Entity.Name,
			GUID: resource.Metadata.GUID,
		})
	}

	return orgs, nil
}

func fetchResources(baseURL, path string, client *http.Client) ([]Resource, error) {
	var (
		resources []Resource
		url       string
	)

	for {
		url = fmt.Sprintf("%s%s", baseURL, path)

		resp, err := client.Get(url)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unsuccessful status code %d", resp.StatusCode)
		}

		response := CCResponse{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			return nil, fmt.Errorf("failed to parse spaces: %v", err)
		}

		resources = append(resources, response.Resources...)

		path = response.NextURL
		if path == "" {
			break
		}
	}

	return resources, nil
}

func getGroupsClaims(orgs []Org, spaces []Space) []string {
	var (
		orgMap       = map[string]string{}
		orgSpaces    = map[string][]Space{}
		groupsClaims = map[string]bool{}
	)

	for _, org := range orgs {
		orgMap[org.GUID] = org.Name
		orgSpaces[org.Name] = []Space{}
		groupsClaims[org.GUID] = true
		groupsClaims[org.Name] = true
	}

	for _, space := range spaces {
		orgName := orgMap[space.OrgGUID]
		orgSpaces[orgName] = append(orgSpaces[orgName], space)
		groupsClaims[space.GUID] = true
		groupsClaims[fmt.Sprintf("%s:%s", space.GUID, space.Role)] = true
	}

	for orgName, spaces := range orgSpaces {
		for _, space := range spaces {
			groupsClaims[fmt.Sprintf("%s:%s", orgName, space.Name)] = true
			groupsClaims[fmt.Sprintf("%s:%s:%s", orgName, space.Name, space.Role)] = true
		}
	}

	var groups []string
	for group := range groupsClaims {
		groups = append(groups, group)
	}

	sort.Strings(groups)

	return groups
}

func (c *cfConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return identity, errors.New(q.Get("error_description"))
	}

	oauth2Config := &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     oauth2.Endpoint{TokenURL: c.tokenURL, AuthURL: c.authorizationURL},
		RedirectURL:  c.redirectURI,
		Scopes:       []string{"openid", "cloud_controller.read"},
	}

	ctx := context.WithValue(r.Context(), oauth2.HTTPClient, c.httpClient)

	token, err := oauth2Config.Exchange(ctx, q.Get("code"))
	if err != nil {
		return identity, fmt.Errorf("CF connector: failed to get token: %v", err)
	}

	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	userInfoResp, err := client.Get(c.userInfoURL)
	if err != nil {
		return identity, fmt.Errorf("CF Connector: failed to execute request to userinfo: %v", err)
	}

	if userInfoResp.StatusCode != http.StatusOK {
		return identity, fmt.Errorf("CF Connector: failed to execute request to userinfo: status %d", userInfoResp.StatusCode)
	}

	defer userInfoResp.Body.Close()

	var userInfoResult map[string]interface{}
	err = json.NewDecoder(userInfoResp.Body).Decode(&userInfoResult)

	if err != nil {
		return identity, fmt.Errorf("CF Connector: failed to parse userinfo: %v", err)
	}

	identity.UserID, _ = userInfoResult["user_id"].(string)
	identity.Username, _ = userInfoResult["user_name"].(string)
	identity.PreferredUsername, _ = userInfoResult["user_name"].(string)
	identity.Email, _ = userInfoResult["email"].(string)
	identity.EmailVerified, _ = userInfoResult["email_verified"].(bool)

	var (
		devPath     = fmt.Sprintf("/v2/users/%s/spaces", identity.UserID)
		auditorPath = fmt.Sprintf("/v2/users/%s/audited_spaces", identity.UserID)
		managerPath = fmt.Sprintf("/v2/users/%s/managed_spaces", identity.UserID)
		orgsPath    = fmt.Sprintf("/v2/users/%s/organizations", identity.UserID)
	)

	if s.Groups {
		orgs, err := fetchOrgs(c.apiURL, orgsPath, client)
		if err != nil {
			return identity, fmt.Errorf("failed to fetch organizaitons: %v", err)
		}

		developerSpaces, err := fetchRoleSpaces(c.apiURL, devPath, "developer", client)
		if err != nil {
			return identity, fmt.Errorf("failed to fetch spaces for developer roles: %v", err)
		}

		auditorSpaces, err := fetchRoleSpaces(c.apiURL, auditorPath, "auditor", client)
		if err != nil {
			return identity, fmt.Errorf("failed to fetch spaces for developer roles: %v", err)
		}

		managerSpaces, err := fetchRoleSpaces(c.apiURL, managerPath, "manager", client)
		if err != nil {
			return identity, fmt.Errorf("failed to fetch spaces for developer roles: %v", err)
		}

		spaces := append(developerSpaces, append(auditorSpaces, managerSpaces...)...)

		identity.Groups = getGroupsClaims(orgs, spaces)
	}

	if s.OfflineAccess {
		data := connectorData{AccessToken: token.AccessToken}
		connData, err := json.Marshal(data)
		if err != nil {
			return identity, fmt.Errorf("CF Connector: failed to parse connector data for offline access: %v", err)
		}
		identity.ConnectorData = connData
	}

	return identity, nil
}
