package openshift

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/groups"
	"github.com/dexidp/dex/pkg/httpclient"
	"github.com/dexidp/dex/storage/kubernetes/k8sapi"
)

const (
	wellKnownURLPath = "/.well-known/oauth-authorization-server"
	usersURLPath     = "/apis/user.openshift.io/v1/users/~"
)

// Config holds configuration options for OpenShift login
type Config struct {
	Issuer           string   `json:"issuer"`
	ClientID         string   `json:"clientID"`
	ClientSecret     string   `json:"clientSecret"`
	ClientSecretFile string   `json:"clientSecretFile"`
	RedirectURI      string   `json:"redirectURI"`
	Groups           []string `json:"groups"`
	InsecureCA       bool     `json:"insecureCA"`
	RootCA           string   `json:"rootCA"`
}

var (
	_ connector.CallbackConnector = (*openshiftConnector)(nil)
	_ connector.RefreshConnector  = (*openshiftConnector)(nil)
)

type openshiftConnector struct {
	apiURL           string
	redirectURI      string
	clientID         string
	clientSecret     string
	clientSecretFile string
	cancel           context.CancelFunc
	logger           *slog.Logger
	httpClient       *http.Client
	oauth2Config     *oauth2.Config
	insecureCA       bool
	rootCA           string
	groups           []string
}

type user struct {
	k8sapi.TypeMeta   `json:",inline"`
	k8sapi.ObjectMeta `json:"metadata,omitempty"`
	Identities        []string `json:"identities" protobuf:"bytes,3,rep,name=identities"`
	FullName          string   `json:"fullName,omitempty" protobuf:"bytes,2,opt,name=fullName"`
	Groups            []string `json:"groups" protobuf:"bytes,4,rep,name=groups"`
}

// Open returns a connector which can be used to login users through an upstream
// OpenShift OAuth2 provider.
func (c *Config) Open(id string, logger *slog.Logger) (conn connector.Connector, err error) {
	var rootCAs []string
	if c.RootCA != "" {
		rootCAs = append(rootCAs, c.RootCA)
	}

	httpClient, err := httpclient.NewHTTPClient(rootCAs, c.InsecureCA)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return c.OpenWithHTTPClient(id, logger, httpClient)
}

// OpenWithHTTPClient returns a connector which can be used to login users through an upstream
// OpenShift OAuth2 provider. It provides the ability to inject a http.Client.
func (c *Config) OpenWithHTTPClient(id string, logger *slog.Logger,
	httpClient *http.Client,
) (conn connector.Connector, err error) {
	if c.ClientSecret != "" && c.ClientSecretFile != "" {
		return nil, fmt.Errorf("invalid config: clientSecret and clientSecretFile are mutually exclusive")
	}
	if c.ClientSecret == "" && c.ClientSecretFile == "" {
		return nil, fmt.Errorf("invalid config: one of clientSecret or clientSecretFile must be specified")
	}

	clientSecret := c.ClientSecret
	if c.ClientSecretFile != "" {
		clientSecretBytes, err := os.ReadFile(c.ClientSecretFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read client secret file %q: %w", c.ClientSecretFile, err)
		}
		clientSecret = strings.TrimSpace(string(clientSecretBytes))
		if clientSecret == "" {
			return nil, fmt.Errorf("client secret file %q contains no valid secret", c.ClientSecretFile)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wellKnownURL := strings.TrimSuffix(c.Issuer, "/") + wellKnownURLPath
	req, err := http.NewRequest(http.MethodGet, wellKnownURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create a request to OpenShift endpoint %w", err)
	}

	openshiftConnector := openshiftConnector{
		apiURL:           c.Issuer,
		cancel:           cancel,
		clientID:         c.ClientID,
		clientSecret:     c.ClientSecret,
		clientSecretFile: c.ClientSecretFile,
		insecureCA:       c.InsecureCA,
		logger:           logger.With(slog.Group("connector", "type", "openshift", "id", id)),
		redirectURI:      c.RedirectURI,
		rootCA:           c.RootCA,
		groups:           c.Groups,
		httpClient:       httpClient,
	}

	var metadata struct {
		Auth  string `json:"authorization_endpoint"`
		Token string `json:"token_endpoint"`
	}

	resp, err := openshiftConnector.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to query OpenShift endpoint %w", err)
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("discovery through endpoint %s failed to decode body: %w",
			wellKnownURL, err)
	}

	openshiftConnector.oauth2Config = &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL: metadata.Auth, TokenURL: metadata.Token,
		},
		Scopes:      []string{"user:info"},
		RedirectURL: c.RedirectURI,
	}
	return &openshiftConnector, nil
}

func (c *openshiftConnector) Close() error {
	c.cancel()
	return nil
}

// LoginURL returns the URL to redirect the user to login with.
func (c *openshiftConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, []byte, error) {
	if c.redirectURI != callbackURL {
		return "", nil, fmt.Errorf("expected callback URL %q did not match the URL in the config %q",
			callbackURL, c.redirectURI)
	}
	return c.oauth2Config.AuthCodeURL(state), nil, nil
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

// HandleCallback parses the request and returns the user's identity
func (c *openshiftConnector) HandleCallback(s connector.Scopes,
	connData []byte,
	r *http.Request,
) (identity connector.Identity, err error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return identity, &oauth2Error{errType, q.Get("error_description")}
	}

	ctx := r.Context()
	if c.httpClient != nil {
		ctx = context.WithValue(r.Context(), oauth2.HTTPClient, c.httpClient)
	}

	oauth2Cfg, err := c.currentOAuth2Config()
	if err != nil {
		return identity, err
	}

	token, err := oauth2Cfg.Exchange(ctx, q.Get("code"))
	if err != nil {
		return identity, fmt.Errorf("oidc: failed to get token: %v", err)
	}

	return c.identity(ctx, s, token)
}

func (c *openshiftConnector) Refresh(ctx context.Context, s connector.Scopes,
	oldID connector.Identity,
) (connector.Identity, error) {
	var token oauth2.Token
	err := json.Unmarshal(oldID.ConnectorData, &token)
	if err != nil {
		return connector.Identity{}, fmt.Errorf("parsing token: %w", err)
	}
	if c.httpClient != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	}
	return c.identity(ctx, s, &token)
}

func (c *openshiftConnector) identity(ctx context.Context, s connector.Scopes,
	token *oauth2.Token,
) (identity connector.Identity, err error) {
	oauth2Cfg, err := c.currentOAuth2Config()
	if err != nil {
		return identity, err
	}
	client := oauth2Cfg.Client(ctx, token)
	user, err := c.user(ctx, client)
	if err != nil {
		return identity, fmt.Errorf("openshift: get user: %v", err)
	}

	if len(c.groups) > 0 {
		validGroups := validateAllowedGroups(user.Groups, c.groups)

		if !validGroups {
			return identity, fmt.Errorf("openshift: user %q is not in any of the required groups", user.Name)
		}
	}

	identity = connector.Identity{
		UserID:            user.UID,
		Username:          user.Name,
		PreferredUsername: user.Name,
		Email:             user.Name,
		Groups:            user.Groups,
	}

	if s.OfflineAccess {
		connData, err := json.Marshal(token)
		if err != nil {
			return identity, fmt.Errorf("marshal connector data: %v", err)
		}
		identity.ConnectorData = connData
	}

	return identity, nil
}

// user function returns the OpenShift user associated with the authenticated user
func (c *openshiftConnector) user(ctx context.Context, client *http.Client) (u user, err error) {
	url := c.apiURL + usersURLPath

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return u, fmt.Errorf("new req: %v", err)
	}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return u, fmt.Errorf("get URL %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return u, fmt.Errorf("read body: %v", err)
		}
		return u, fmt.Errorf("%s: %s", resp.Status, body)
	}

	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return u, fmt.Errorf("JSON decode: %v", err)
	}

	return u, err
}

// currentOAuth2Config returns the oauth2 config with the current client secret.
// When a client secret file is configured, the client secret is re-read from
// disk on each call so that rotated client secret is picked up without a restart.
func (c *openshiftConnector) currentOAuth2Config() (*oauth2.Config, error) {
	if c.clientSecretFile == "" {
		return c.oauth2Config, nil
	}
	clientSecretBytes, err := os.ReadFile(c.clientSecretFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read client secret file %q: %w", c.clientSecretFile, err)
	}
	cfg := *c.oauth2Config
	clientSecret := strings.TrimSpace(string(clientSecretBytes))
	if clientSecret == "" {
		return nil, fmt.Errorf("client secret file %q contains no valid secret", c.clientSecretFile)
	}
	cfg.ClientSecret = clientSecret
	return &cfg, nil
}

func validateAllowedGroups(userGroups, allowedGroups []string) bool {
	matchingGroups := groups.Filter(userGroups, allowedGroups)

	return len(matchingGroups) != 0
}
