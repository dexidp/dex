package cloudfoundry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/dexidp/dex/connector"
)

type cloudfoundryConnector struct {
	clientID         string
	clientSecret     string
	redirectURI      string
	apiURL           string
	tokenURL         string
	authorizationURL string
	userInfoURL      string
	httpClient       *http.Client
	logger           *slog.Logger
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

type ccResponse struct {
	Pagination pagination `json:"pagination"`
	Resources  []resource `json:"resources"`
}

type pagination struct {
	Next href `json:"next"`
}

type href struct {
	Href string `json:"href"`
}

type resource struct {
	GUID          string        `json:"guid"`
	Name          string        `json:"name,omitempty"`
	Type          string        `json:"type,omitempty"`
	Relationships relationships `json:"relationships"`
}

type relationships struct {
	Organization relOrganization `json:"organization"`
	Space        relSpace        `json:"space"`
}

type relOrganization struct {
	Data data `json:"data"`
}

type relSpace struct {
	Data data `json:"data"`
}

type data struct {
	GUID string `json:"guid"`
}

type space struct {
	Name    string
	GUID    string
	OrgGUID string
	Role    string
}

type org struct {
	Name string
	GUID string
}

type infoResp struct {
	Links links `json:"links"`
}

type links struct {
	Login login `json:"login"`
}

type login struct {
	Href string `json:"href"`
}

func (c *Config) Open(id string, logger *slog.Logger) (connector.Connector, error) {
	var err error

	cloudfoundryConn := &cloudfoundryConnector{
		clientID:     c.ClientID,
		clientSecret: c.ClientSecret,
		apiURL:       c.APIURL,
		redirectURI:  c.RedirectURI,
		logger:       logger,
	}

	cloudfoundryConn.httpClient, err = newHTTPClient(c.RootCAs, c.InsecureSkipVerify)
	if err != nil {
		return nil, err
	}

	apiURL := strings.TrimRight(c.APIURL, "/")
	apiResp, err := cloudfoundryConn.httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed-to-send-request-to-cloud-controller-api: %w", err)
	}

	defer apiResp.Body.Close()

	if apiResp.StatusCode != http.StatusOK {
		err = fmt.Errorf("request failed with status %d", apiResp.StatusCode)
		return nil, fmt.Errorf("failed-get-info-response-from-api: %w", err)
	}

	var apiResult infoResp

	json.NewDecoder(apiResp.Body).Decode(&apiResult)

	uaaURL := strings.TrimRight(apiResult.Links.Login.Href, "/")
	uaaResp, err := cloudfoundryConn.httpClient.Get(fmt.Sprintf("%s/.well-known/openid-configuration", uaaURL))
	if err != nil {
		return nil, fmt.Errorf("failed-to-send-request-to-uaa-api: %w", err)
	}

	if apiResp.StatusCode != http.StatusOK {
		err = fmt.Errorf("request failed with status %d", apiResp.StatusCode)
		return nil, fmt.Errorf("failed-to-get-well-known-config-response-from-api: %w", err)
	}

	defer uaaResp.Body.Close()

	var uaaResult map[string]interface{}
	err = json.NewDecoder(uaaResp.Body).Decode(&uaaResult)

	if err != nil {
		return nil, fmt.Errorf("failed-to-decode-response-from-uaa-api: %w", err)
	}

	cloudfoundryConn.tokenURL, _ = uaaResult["token_endpoint"].(string)
	cloudfoundryConn.authorizationURL, _ = uaaResult["authorization_endpoint"].(string)
	cloudfoundryConn.userInfoURL, _ = uaaResult["userinfo_endpoint"].(string)

	return cloudfoundryConn, err
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

func (c *cloudfoundryConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, error) {
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

func filterUserOrgsSpaces(userOrgsSpaces []resource, orgs []resource, spaces []resource) ([]org, []space) {
	var filteredOrgs []org
	var filteredSpaces []space

	orgMap := make(map[string]org)
	spaceMap := make(map[string]space)

	for _, org_resource := range orgs {
		orgMap[org_resource.GUID] = org{
			Name: org_resource.Name,
			GUID: org_resource.GUID,
		}
	}

	for _, space_resource := range spaces {
		spaceMap[space_resource.GUID] = space{
			Name:    space_resource.Name,
			GUID:    space_resource.GUID,
			OrgGUID: space_resource.Relationships.Organization.Data.GUID,
		}
	}

	for _, userOrgSpace := range userOrgsSpaces {
		if space, ok := spaceMap[userOrgSpace.Relationships.Space.Data.GUID]; ok {
			space.Role = strings.TrimPrefix(userOrgSpace.Type, "space_")
			filteredSpaces = append(filteredSpaces, space)
		}
		if org, ok := orgMap[userOrgSpace.Relationships.Organization.Data.GUID]; ok {
			filteredOrgs = append(filteredOrgs, org)
		}
	}

	return filteredOrgs, filteredSpaces
}

func fetchResources(baseURL, path string, client *http.Client) ([]resource, error) {
	var (
		resources []resource
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

		response := ccResponse{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			return nil, fmt.Errorf("failed to parse response: %v", err)
		}

		resources = append(resources, response.Resources...)

		path = strings.TrimPrefix(response.Pagination.Next.Href, baseURL)
		if path == "" {
			break
		}
	}

	return resources, nil
}

func getGroupsClaims(orgs []org, spaces []space) []string {
	var (
		orgMap       = map[string]string{}
		orgSpaces    = map[string][]space{}
		groupsClaims = map[string]bool{}
	)

	for _, org := range orgs {
		orgMap[org.GUID] = org.Name
		orgSpaces[org.Name] = []space{}
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

	groups := make([]string, 0, len(groupsClaims))
	for group := range groupsClaims {
		groups = append(groups, group)
	}

	sort.Strings(groups)

	return groups
}

func (c *cloudfoundryConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
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
		orgsPath           = "/v3/organizations"
		spacesPath         = "/v3/spaces"
		userOrgsSpacesPath = fmt.Sprintf("/v3/roles?user_guids=%s&types=space_developer,space_manager,space_auditor,organization_user", identity.UserID)
	)

	if s.Groups {
		userOrgsSpaces, err := fetchResources(c.apiURL, userOrgsSpacesPath, client)
		if err != nil {
			return identity, fmt.Errorf("failed to fetch user organizations: %v", err)
		}

		orgs, err := fetchResources(c.apiURL, orgsPath, client)
		if err != nil {
			return identity, fmt.Errorf("failed to fetch organizaitons: %v", err)
		}

		spaces, err := fetchResources(c.apiURL, spacesPath, client)
		if err != nil {
			return identity, fmt.Errorf("failed to fetch spaces: %v", err)
		}

		developerOrgs, developerSpaces := filterUserOrgsSpaces(userOrgsSpaces, orgs, spaces)

		identity.Groups = getGroupsClaims(developerOrgs, developerSpaces)
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
