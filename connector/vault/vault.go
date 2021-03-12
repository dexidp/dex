// Package vault implements logging in through Vault userpass or
// OpenID Connect providers behind Vault
package vault

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/log"
)

type Config struct {
	// Parameters used to connect to Vault
	VaultURL           string            `json:"vaultURL"`
	VaultCACert        string            `json:"vault_cacert"`
	InsecureSkipVerify bool              `json:"insecureSkipVerify"`
	MountType          string            `json:"mountType"`
	MountPath          string            `json:"mountPath"`
	ClaimMappings      map[string]string `json:"claimMappings"`

	// UsernamePrompt allows users to override the username attribute (displayed
	// in the username/password prompt). If unset, the handler will use
	// "Username".
	UsernamePrompt string `json:"usernamePrompt"`
}

type vaultConnector struct {
	Config
	logger     log.Logger
	httpClient *http.Client
}

type loginConnector struct {
	vaultConnector
}

type oidcConnector struct {
	vaultConnector
	states sync.Map
}

type vaultLoginRequest struct {
	Password string `json:"password"`
}

type vaultLoginResponse struct {
	Token *vaultToken `json:"auth"`
}

type vaultAuthURLRequest struct {
	Role        string `json:"role"`
	RedirectURI string `json:"redirect_uri"`
}

type vaultAuthURLResponse struct {
	Data struct {
		AuthURL string `json:"auth_url"`
	}
}

type vaultToken struct {
	ClientToken   string            `json:"client_token"`
	Policies      []string          `json:"policies"`
	TokenPolicies []string          `json:"token_policies"`
	Metadata      map[string]string `json:"metadata"`
	LeaseDuration int               `json:"lease_duration"`
	Renewable     bool              `json:"renewable"`
	EntityID      string            `json:"entity_id"`
	TokenType     string            `json:"token_type"`
	Orphan        bool              `json:"orphan"`
}

type vaultEntityResponse struct {
	Data vaultEntity `json:"data"`
}

type vaultEntity struct {
	Aliases           []*vaultEntityAlias `json:"aliases"`
	CreationTime      string              `json:"creation_time"`
	DirectGroupIDs    []string            `json:"direct_group_ids"`
	Disabled          bool                `json:"disabled"`
	GroupIDs          []string            `json:"group_ids"`
	ID                string              `json:"id"`
	InheritedGroupIDs []string            `json:"inherited_group_ids"`
	LastUpdateTime    string              `json:"last_update_time"`
	// MergedEntityIDs []string `json:"merged_entity_ids"`
	Metadata map[string]string `json:"metadata"`
	Name     string            `json:"name"`
	Policies []string          `json:"policies"`
}

type vaultEntityAlias struct {
	CanonicalID    string `json:"canonical_id"`
	CreationTime   string `json:"creation_time"`
	ID             string `json:"id"`
	LastUpdateTime string `json:"last_update_time"`
	// MergedFromCanonicalIDs []string `json:"merged_from_canonical_ids"`
	Metadata      map[string]string `json:"metadata"`
	MountAccessor string            `json:"mount_accessor"`
	MountPath     string            `json:"mount_path"`
	MountType     string            `json:"mount_type"`
	Name          string            `json:"name"`
}

func (c *Config) Open(id string, logger log.Logger) (conn connector.Connector, err error) {
	if c.VaultURL == "" {
		return nil, fmt.Errorf("vaultURL must be set")
	}
	httpClient, err := newHTTPClient(c.InsecureSkipVerify, c.VaultCACert)
	if err != nil {
		return nil, err
	}

	if c.MountPath == "" {
		c.MountPath = c.MountType
	}

	dummy := connector.Identity{}
	for claim := range c.ClaimMappings {
		err := mapClaim(&dummy, claim, "", connector.Scopes{Groups: true})
		if err != nil {
			return nil, fmt.Errorf("claim '%s' cannot be mapped", claim)
		}
	}

	switch c.MountType {
	case "userpass", "ldap", "radius", "okta":
		return &loginConnector{vaultConnector{*c, logger, httpClient}}, nil
	case "oidc":
		return &oidcConnector{vaultConnector{*c, logger, httpClient}, sync.Map{}}, nil
	case "":
		return nil, fmt.Errorf("mountType must be set")
	default:
		return nil, fmt.Errorf("unknown mountType '%s'", c.MountType)
	}
}

// Login using username/password to Vault
// https://www.vaultproject.io/api-docs/auth/userpass#login
// https://www.vaultproject.io/api-docs/auth/ldap#login-with-ldap-user
// https://www.vaultproject.io/api-docs/auth/radius#login
// https://www.vaultproject.io/api-docs/auth/okta#login
//
// XXX: have a way to split the password for MFA and set X-Vault-MFA header?
// https://www.vaultproject.io/docs/enterprise/mfa
func (c *loginConnector) Login(ctx context.Context, s connector.Scopes, username, password string) (connector.Identity, bool, error) {
	if password == "" {
		return connector.Identity{}, false, nil
	}

	var rx vaultLoginResponse
	err := c.post(fmt.Sprintf("/v1/auth/%s/login/%s", c.MountPath, username),
		nil, &vaultLoginRequest{password}, &rx)
	// Login failure is indicated by a 400 response code
	if e, ok := err.(*httpError); ok && e.Code == http.StatusBadRequest {
		return connector.Identity{}, false, nil
	}
	if err != nil {
		return connector.Identity{}, false, err
	}

	identity, err := c.vaultIdentity(rx.Token, s)
	if err != nil {
		return connector.Identity{}, false, err
	}
	return identity, true, nil
}

func (c *loginConnector) Prompt() string {
	return c.UsernamePrompt
}

func (c *loginConnector) Close() error {
	return nil
}

// Login using Vault OpenID
// https://www.vaultproject.io/api-docs/auth/jwt#oidc-authorization-url-request
func (c *oidcConnector) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
	var rx vaultAuthURLResponse
	err := c.post(fmt.Sprintf("/v1/auth/%s/oidc/auth_url", c.MountPath),
		nil, &vaultAuthURLRequest{RedirectURI: callbackURL}, &rx)
	if err != nil {
		return "", err
	}
	if rx.Data.AuthURL == "" {
		// If callback URL not permitted, Vault returns 200 status with empty auth_url.
		// https://github.com/hashicorp/vault/issues/11071
		return "", fmt.Errorf("vault returned empty auth_url.  Did you include %s in allowed_redirect_uris?", callbackURL)
	}

	// Vault includes its own "state" in the redirectURI.  However, Dex
	// checks the state before calling HandleCallback.  This means we have
	// to stash the Vault state somewhere and reinsert it later
	//
	// XXX: this ought to go in "storage".  Perhaps connectors should have a
	// way to save request connectorData between LoginURL and HandleCallback?
	redirectURL, err := url.Parse(rx.Data.AuthURL)
	if err != nil {
		return "", fmt.Errorf("unable to parse redirect_uri")
	}
	q := redirectURL.Query()
	vaultState := q.Get("state")
	c.states.Store(state, vaultState)
	q.Set("state", state)
	redirectURL.RawQuery = q.Encode()

	return redirectURL.String(), nil
}

// https://www.vaultproject.io/api-docs/auth/jwt#oidc-callback
func (c *oidcConnector) HandleCallback(s connector.Scopes, r *http.Request) (connector.Identity, error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return connector.Identity{}, fmt.Errorf("oauth error: %s %s", errType, q.Get("error_description"))
	}

	// restore the state to the value Vault expects
	value, loaded := c.states.LoadAndDelete(q.Get("state"))
	if !loaded {
		return connector.Identity{}, fmt.Errorf("vault state not found")
	}
	vaultState, ok := value.(string)
	if !ok {
		return connector.Identity{}, fmt.Errorf("vault state wrong type")
	}
	q.Set("state", vaultState)

	var rx vaultLoginResponse
	err := c.get(fmt.Sprintf("/v1/auth/%s/oidc/callback?%s", c.MountPath, q.Encode()), nil, &rx)
	if err != nil {
		return connector.Identity{}, fmt.Errorf("vault: failed to get identity in callback: %v", err)
	}

	identity, err := c.vaultIdentity(rx.Token, s)
	return identity, err
}

// XXX: Refresh()? If we want to be able to refresh the (Vault) token, then we should exchange it for
// one with fewer policies (for safety) and store it in ConnectorData

// Exchange Vault token for connector.Identity
func (c *vaultConnector) vaultIdentity(token *vaultToken, s connector.Scopes) (connector.Identity, error) {
	if token == nil || token.ClientToken == "" {
		return connector.Identity{}, fmt.Errorf("vault response contained no vault token")
	}

	var data vaultEntityResponse
	err := c.get("/v1/identity/entity/id/"+token.EntityID, map[string]string{
		"X-Vault-Token": token.ClientToken,
	}, &data)
	if err != nil {
		return connector.Identity{}, err
	}
	entity := data.Data
	if entity.ID == "" {
		return connector.Identity{}, fmt.Errorf("no entity id")
	}

	groups := []string{}
	if s.Groups {
		groups = entity.GroupIDs
		// XXX: mapping group IDs to names? /v1/identity/group/id/<id> is not accessible to regular users without a policy
		// XXX: mapping policies to groups?
	}
	identity := connector.Identity{
		UserID:   entity.ID,
		Username: entity.Name,
		Groups:   groups,
	}

	for claim, meta := range c.ClaimMappings {
		err := mapClaim(&identity, claim, entity.Metadata[meta], s)
		if err != nil {
			return connector.Identity{}, err
		}
	}

	return identity, nil
}

func mapClaim(identity *connector.Identity, claim, value string, s connector.Scopes) error {
	switch claim {
	case "":
		return fmt.Errorf("claim name must not be empty")
	case "email":
		identity.Email = value
	case "preferred_username":
		identity.PreferredUsername = value
	case "email_verified":
		switch value {
		case "yes", "true":
			identity.EmailVerified = true
		default:
			identity.EmailVerified = false
		}
	default:
		// In the absence of custom claims, have a way of mapping entity metadata to groups
		// "+foo": "meta_bar"     -- add group "foo{{meta_bar}}" if meta_bar is non-empty
		// "?foo": "meta_bar"     -- add group "foo" if meta_bar is non-empty
		// "?foo:baz": "meta_bar" -- add group "foo" if meta_bar equals "baz"
		switch claim[0] {
		case '+':
			if s.Groups && value != "" {
				group := claim[1:] + value
				identity.Groups = append(identity.Groups, group)
			}
		case '?':
			if s.Groups {
				pieces := strings.SplitN(claim[1:], ":", 2)
				if len(pieces) == 1 && value != "" {
					identity.Groups = append(identity.Groups, pieces[0])
				} else if len(pieces) == 2 && value == pieces[1] {
					identity.Groups = append(identity.Groups, pieces[0])
				}
			}
		default:
			// XXX: custom claims: https://github.com/dexidp/dex/issues/1182
			return fmt.Errorf("claim '%s' is not supported by dex", claim)
		}
	}
	return nil
}

// Utilities for HTTP interaction
func (c *vaultConnector) raw(method, path string, headers map[string]string, tx io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.VaultURL+path, tx)
	if err != nil {
		return nil, fmt.Errorf("%s %s: NewRequest: %w", method, path, err)
	}
	if method != "GET" {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: httpClient.Do: %w", method, path, err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return resp, nil
	case http.StatusForbidden, http.StatusUnauthorized:
		resp.Body.Close()
		return nil, &httpError{
			Code:        resp.StatusCode,
			Description: fmt.Sprintf("%s %s: permission denied by Vault", method, path),
		}
	default:
		// Try parsing it as vault {"errors":[...]}
		var errors struct {
			Errors []string `json:"errors"`
		}
		dec := json.NewDecoder(resp.Body)
		err := dec.Decode(&errors)
		resp.Body.Close()

		var msg string
		if err == nil && len(errors.Errors) > 0 {
			msg = strings.Join(errors.Errors, "; ")
		} else {
			msg = fmt.Sprintf("%s %s: Unexpected status: %s", method, path, resp.Status)
		}

		return nil, &httpError{
			Code:        resp.StatusCode,
			Description: msg,
		}
	}
}

func (c *vaultConnector) get(path string, headers map[string]string, response interface{}) error {
	r, err := c.raw("GET", path, headers, nil)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	err = dec.Decode(response)
	if err != nil {
		return fmt.Errorf("json error: get %s: %w", path, err)
	}
	return nil
}

func (c *vaultConnector) post(path string, headers map[string]string, request interface{}, response interface{}) error {
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	r, err := c.raw("POST", path, headers, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	err = dec.Decode(response)
	if err != nil {
		return err
	}
	return nil
}

type httpError struct {
	Code        int
	Description string
}

func (e httpError) Error() string {
	return e.Description
}

// newHTTPClient returns a new HTTP client
func newHTTPClient(insecureCA bool, rootCA string) (*http.Client, error) {
	tlsConfig := tls.Config{}

	if insecureCA {
		tlsConfig = tls.Config{InsecureSkipVerify: true}
	} else if rootCA != "" {
		tlsConfig = tls.Config{RootCAs: x509.NewCertPool()}
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
