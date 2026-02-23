// Package microsoft provides authentication strategies using Microsoft.
package microsoft

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/dexidp/dex/connector"
	groups_pkg "github.com/dexidp/dex/pkg/groups"
)

// GroupNameFormat represents the format of the group identifier
// we use type of string instead of int because it's easier to
// marshall/unmarshall
type GroupNameFormat string

// Possible values for GroupNameFormat
const (
	GroupID   GroupNameFormat = "id"
	GroupName GroupNameFormat = "name"
)

const (
	// Microsoft requires this scope to list groups the user is a member of
	// and resolve their ids to groups names.
	scopeGroups = "directory.read.all"
	// Microsoft requires this scope to return a refresh token
	// see https://docs.microsoft.com/en-us/azure/active-directory/develop/v2-permissions-and-consent#offline_access
	scopeOfflineAccess = "offline_access"
)

// Config holds configuration options for microsoft logins.
type Config struct {
	ClientID             string          `json:"clientID"`
	ClientSecret         string          `json:"clientSecret"`
	RedirectURI          string          `json:"redirectURI"`
	Tenant               string          `json:"tenant"`
	OnlySecurityGroups   bool            `json:"onlySecurityGroups"`
	Groups               []string        `json:"groups"`
	GroupNameFormat      GroupNameFormat `json:"groupNameFormat"`
	UseGroupsAsWhitelist bool            `json:"useGroupsAsWhitelist"`
	EmailToLowercase     bool            `json:"emailToLowercase"`

	APIURL   string `json:"apiURL"`
	GraphURL string `json:"graphURL"`

	// PromptType is used for the prompt query parameter.
	// For valid values, see https://docs.microsoft.com/en-us/azure/active-directory/develop/v2-oauth2-auth-code-flow#request-an-authorization-code.
	PromptType string `json:"promptType"`
	DomainHint string `json:"domainHint"`

	Scopes []string `json:"scopes"` // defaults to openid,profile,email
}

// Open returns a strategy for logging in through Microsoft.
func (c *Config) Open(id string, logger *slog.Logger) (connector.Connector, error) {
	m := microsoftConnector{
		apiURL:               strings.TrimSuffix(c.APIURL, "/"),
		graphURL:             strings.TrimSuffix(c.GraphURL, "/"),
		redirectURI:          c.RedirectURI,
		clientID:             c.ClientID,
		clientSecret:         c.ClientSecret,
		tenant:               c.Tenant,
		onlySecurityGroups:   c.OnlySecurityGroups,
		groups:               c.Groups,
		groupNameFormat:      c.GroupNameFormat,
		useGroupsAsWhitelist: c.UseGroupsAsWhitelist,
		logger:               logger.With(slog.Group("connector", "type", "microsoft", "id", id)),
		emailToLowercase:     c.EmailToLowercase,
		promptType:           c.PromptType,
		domainHint:           c.DomainHint,
		scopes:               c.Scopes,
	}

	if m.apiURL == "" {
		m.apiURL = "https://login.microsoftonline.com"
	}

	if m.graphURL == "" {
		m.graphURL = "https://graph.microsoft.com"
	}

	// By default allow logins from both personal and business/school
	// accounts.
	if m.tenant == "" {
		m.tenant = "common"
	}
	m.issuer = m.apiURL + "/{tenantid}/v2.0"

	// By default, use group names
	switch m.groupNameFormat {
	case "":
		m.groupNameFormat = GroupName
	case GroupID, GroupName:
	default:
		return nil, fmt.Errorf("invalid groupNameFormat: %s", m.groupNameFormat)
	}

	issuer := strings.ReplaceAll(m.issuer, "{tenantid}", m.tenant)
	ctx := oidc.InsecureIssuerURLContext(context.Background(), issuer)
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("provider error: %v", err)
	}

	var pclaims map[string]interface{}
	if err := provider.Claims(&pclaims); err != nil {
		return nil, fmt.Errorf("oidc: failed to decode provider claims: %v", err)
	}

	if pissuer, found := pclaims["issuer"]; !found || pissuer != m.issuer {
		return nil, fmt.Errorf("oidc: incorrect prodiver issuer in well known configuration %q", pissuer)
	}

	m.provider = provider

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
	apiURL               string
	graphURL             string
	issuer               string // issuer for discovery of well known configuration
	provider             *oidc.Provider
	redirectURI          string
	clientID             string
	clientSecret         string
	tenant               string
	onlySecurityGroups   bool
	groupNameFormat      GroupNameFormat
	groups               []string
	useGroupsAsWhitelist bool
	logger               *slog.Logger
	emailToLowercase     bool
	promptType           string
	domainHint           string
	scopes               []string
}

func (c *microsoftConnector) isOrgTenant() bool {
	return c.tenant != "common" && c.tenant != "consumers" && c.tenant != "organizations"
}

func (c *microsoftConnector) groupsRequired(groupScope bool) bool {
	return (len(c.groups) > 0 || groupScope) && c.isOrgTenant()
}

func (c *microsoftConnector) oauth2Config(scopes connector.Scopes) *oauth2.Config {
	var microsoftScopes []string
	if len(c.scopes) > 0 {
		microsoftScopes = c.scopes
	} else {
		microsoftScopes = append(microsoftScopes, oidc.ScopeOpenID, "profile", "email")
	}
	if c.groupsRequired(scopes.Groups) {
		microsoftScopes = append(microsoftScopes, scopeGroups)
	}

	if scopes.OfflineAccess {
		microsoftScopes = append(microsoftScopes, scopeOfflineAccess)
	}

	return &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  c.apiURL + "/" + c.tenant + "/oauth2/v2.0/authorize",
			TokenURL: c.apiURL + "/" + c.tenant + "/oauth2/v2.0/token",
		},
		Scopes:      microsoftScopes,
		RedirectURL: c.redirectURI,
	}
}

func (c *microsoftConnector) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, []byte, error) {
	if c.redirectURI != callbackURL {
		return "", nil, fmt.Errorf("expected callback URL %q did not match the URL in the config %q", callbackURL, c.redirectURI)
	}

	var options []oauth2.AuthCodeOption
	if c.promptType != "" {
		options = append(options, oauth2.SetAuthURLParam("prompt", c.promptType))
	}
	if c.domainHint != "" {
		options = append(options, oauth2.SetAuthURLParam("domain_hint", c.domainHint))
	}

	return c.oauth2Config(scopes).AuthCodeURL(state, options...), nil, nil
}

func (c *microsoftConnector) HandleCallback(s connector.Scopes, connData []byte, r *http.Request) (identity connector.Identity, err error) {
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

	user, err := c.userFromToken(ctx, token)
	if err != nil {
		return identity, fmt.Errorf("microsoft: get user: %v", err)
	}

	if c.emailToLowercase {
		user.Email = strings.ToLower(user.Email)
	}

	identity = connector.Identity{
		UserID:        user.ID,
		Username:      user.Name,
		Email:         user.Email,
		EmailVerified: true,
	}

	if c.groupsRequired(s.Groups) {
		groups, err := c.getGroups(ctx, client, user.ID)
		if err != nil {
			return identity, fmt.Errorf("microsoft: get groups: %w", err)
		}
		identity.Groups = groups
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

// notifyRefreshTokenSource is essentially `oauth2.ReuseTokenSource` with `TokenNotifyFunc` added.
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
	user, err := c.userFromToken(ctx, tok)
	if err != nil {
		return identity, fmt.Errorf("microsoft: get user: %v", err)
	}

	identity.Username = user.Name
	identity.Email = user.Email

	if c.groupsRequired(s.Groups) {
		groups, err := c.getGroups(ctx, client, user.ID)
		if err != nil {
			return identity, fmt.Errorf("microsoft: get groups: %w", err)
		}
		identity.Groups = groups
	}

	return identity, nil
}

type user struct {
	ID    string `json:"id"`
	Name  string `json:"displayName"`
	Email string `json:"userPrincipalName"`
}

func (c *microsoftConnector) userFromToken(ctx context.Context, token *oauth2.Token) (u user, err error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return u, errors.New("oidc: no id_token in token response")
	}

	// NOTE: issuer is verified below manually
	verifier := c.provider.Verifier(
		&oidc.Config{ClientID: c.clientID, SkipIssuerCheck: true},
	)

	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return u, fmt.Errorf("oidc: failed to verify ID Token: %v", err)
	}

	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		return u, fmt.Errorf("oidc: failed to decode claims: %v", err)
	}

	// https://docs.microsoft.com/en-us/azure/active-directory/develop/id-tokens
	tid, found := claims["tid"].(string)
	if !found {
		return u, errors.New("missing 'tid' claim")
	}

	iss, found := claims["iss"].(string)
	if !found {
		return u, errors.New("missing 'iss' claim")
	}

	objectID, found := claims["oid"].(string)
	if !found {
		return u, errors.New("missing 'oid' claim")
	}

	name, found := claims["name"].(string)
	if !found {
		return u, errors.New("missing 'name' claim")
	}

	email, found := claims["email"].(string)
	if !found {
		return u, errors.New("missing 'email' claim")
	}

	if iss != strings.ReplaceAll(c.issuer, "{tenantid}", tid) {
		return u, fmt.Errorf("incorrect token issuer:", iss)
	}

	return user{
		ID:    objectID,
		Name:  name,
		Email: email,
	}, nil
}

// https://developer.microsoft.com/en-us/graph/docs/api-reference/v1.0/resources/group
// displayName - The display name for the group. This property is required when
//
//	a group is created and it cannot be cleared during updates.
//	Supports $filter and $orderby.
type group struct {
	Name string `json:"displayName"`
}

func (c *microsoftConnector) getGroups(ctx context.Context, client *http.Client, userID string) ([]string, error) {
	userGroups, err := c.getGroupIDs(ctx, client)
	if err != nil {
		return nil, err
	}

	if c.groupNameFormat == GroupName {
		userGroups, err = c.getGroupNames(ctx, client, userGroups)
		if err != nil {
			return nil, err
		}
	}

	// ensure that the user is in at least one required group
	filteredGroups := groups_pkg.Filter(userGroups, c.groups)
	if len(c.groups) > 0 && len(filteredGroups) == 0 {
		return nil, &connector.UserNotInRequiredGroupsError{UserID: userID, Groups: c.groups}
	} else if c.useGroupsAsWhitelist {
		return filteredGroups, nil
	}

	return userGroups, nil
}

func (c *microsoftConnector) getGroupIDs(ctx context.Context, client *http.Client) (ids []string, err error) {
	// https://developer.microsoft.com/en-us/graph/docs/api-reference/v1.0/api/user_getmembergroups
	in := &struct {
		SecurityEnabledOnly bool `json:"securityEnabledOnly"`
	}{c.onlySecurityGroups}
	reqURL := c.graphURL + "/v1.0/me/getMemberGroups"
	for {
		var out []string
		var next string

		next, err = c.post(ctx, client, reqURL, in, &out)
		if err != nil {
			return ids, err
		}

		ids = append(ids, out...)
		if next == "" {
			return
		}
		reqURL = next
	}
}

func (c *microsoftConnector) getGroupNames(ctx context.Context, client *http.Client, ids []string) (groups []string, err error) {
	if len(ids) == 0 {
		return
	}

	// https://developer.microsoft.com/en-us/graph/docs/api-reference/v1.0/api/directoryobject_getbyids
	in := &struct {
		IDs   []string `json:"ids"`
		Types []string `json:"types"`
	}{ids, []string{"group"}}
	reqURL := c.graphURL + "/v1.0/directoryObjects/getByIds"
	for {
		var out []group
		var next string

		next, err = c.post(ctx, client, reqURL, in, &out)
		if err != nil {
			return groups, err
		}

		for _, g := range out {
			groups = append(groups, g.Name)
		}
		if next == "" {
			return
		}
		reqURL = next
	}
}

func (c *microsoftConnector) post(ctx context.Context, client *http.Client, reqURL string, in interface{}, out interface{}) (string, error) {
	var payload bytes.Buffer

	err := json.NewEncoder(&payload).Encode(in)
	if err != nil {
		return "", fmt.Errorf("microsoft: JSON encode: %v", err)
	}

	req, err := http.NewRequest("POST", reqURL, &payload)
	if err != nil {
		return "", fmt.Errorf("new req: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("post URL %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", newGraphError(resp.Body)
	}

	var next string
	if err = json.NewDecoder(resp.Body).Decode(&struct {
		NextLink *string     `json:"@odata.nextLink"`
		Value    interface{} `json:"value"`
	}{&next, out}); err != nil {
		return "", fmt.Errorf("JSON decode: %v", err)
	}

	return next, nil
}

type graphError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *graphError) Error() string {
	return e.Code + ": " + e.Message
}

func newGraphError(r io.Reader) error {
	// https://developer.microsoft.com/en-us/graph/docs/concepts/errors
	var ge graphError
	if err := json.NewDecoder(r).Decode(&struct {
		Error *graphError `json:"error"`
	}{&ge}); err != nil {
		return fmt.Errorf("JSON error decode: %v", err)
	}
	return &ge
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
