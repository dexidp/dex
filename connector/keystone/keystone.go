// Package keystone provides authentication strategy using Keystone.
package keystone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/log"
)

type conn struct {
	Domain           string
	Host             string
	AdminUsername    string
	AdminPassword    string
	Groups           []group
	UseRolesAsGroups bool
	client           *http.Client
	Logger           log.Logger
}

type group struct {
	Name    string `json:"name"`
	Replace string `json:"replace"`
}

type userKeystone struct {
	Domain domainKeystone `json:"domain"`
	ID     string         `json:"id"`
	Name   string         `json:"name"`
}

type domainKeystone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Config holds the configuration parameters for Keystone connector.
// Keystone should expose API v3
// An example config:
//
//	connectors:
//		type: keystone
//		id: keystone
//		name: Keystone
//		config:
//			keystoneHost: http://example:5000
//			domain: default
//			keystoneUsername: demo
//			keystonePassword: DEMO_PASS
//			useRolesAsGroups: true
type Config struct {
	Domain           string  `json:"domain"`
	Host             string  `json:"keystoneHost"`
	AdminUsername    string  `json:"keystoneUsername"`
	AdminPassword    string  `json:"keystonePassword"`
	UseRolesAsGroups bool    `json:"useRolesAsGroups"`
	Groups           []group `json:"groups"`
}

type loginRequestData struct {
	auth `json:"auth"`
}

type auth struct {
	Identity identity `json:"identity"`
}

type identity struct {
	Methods  []string `json:"methods"`
	Password password `json:"password"`
}

type password struct {
	User user `json:"user"`
}

type user struct {
	Name     string `json:"name"`
	Domain   domain `json:"domain"`
	Password string `json:"password"`
}

type domain struct {
	ID string `json:"id"`
}

type token struct {
	User userKeystone `json:"user"`
}

type tokenResponse struct {
	Token token `json:"token"`
}

type keystoneGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type groupsResponse struct {
	Groups []keystoneGroup `json:"groups"`
}

type userResponse struct {
	User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		ID    string `json:"id"`
	} `json:"user"`
}

type role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DomainID    string `json:"domain_id"`
	Description string `json:"description"`
}

type identifierContainer struct {
	ID string `json:"id"`
}

type roleAssignment struct {
	User identifierContainer `json:"user"`
	Role identifierContainer `json:"role"`
}

var (
	_ connector.PasswordConnector = &conn{}
	_ connector.RefreshConnector  = &conn{}
)

// Open returns an authentication strategy using Keystone.
func (c *Config) Open(id string, logger log.Logger) (connector.Connector, error) {
	return &conn{
		Domain:           c.Domain,
		Host:             c.Host,
		AdminUsername:    c.AdminUsername,
		AdminPassword:    c.AdminPassword,
		UseRolesAsGroups: c.UseRolesAsGroups,
		Groups:           c.Groups,
		Logger:           logger,
		client:           http.DefaultClient,
	}, nil
}

func (p *conn) Close() error { return nil }

func (p *conn) Login(ctx context.Context, scopes connector.Scopes, username, password string) (identity connector.Identity, validPassword bool, err error) {
	resp, err := p.getTokenResponse(ctx, username, password)
	if err != nil {
		return identity, false, fmt.Errorf("keystone: error %v", err)
	}
	if resp.StatusCode/100 != 2 {
		return identity, false, fmt.Errorf("keystone login: error %v", resp.StatusCode)
	}
	if resp.StatusCode != 201 {
		return identity, false, nil
	}
	token := resp.Header.Get("X-Subject-Token")
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return identity, false, err
	}
	defer resp.Body.Close()
	tokenResp := new(tokenResponse)
	err = json.Unmarshal(data, &tokenResp)
	if err != nil {
		return identity, false, fmt.Errorf("keystone: invalid token response: %v", err)
	}
	var userGroups []string
	if p.groupsRequired(scopes.Groups) {
		if scopes.Groups {
			userGroups, err = p.getUserGroups(ctx, tokenResp.Token.User.ID, token)
			if err != nil {
				return identity, false, err
			}
		}
		if p.UseRolesAsGroups {
			roleGroups, err := p.getUserRolesAsGroups(ctx, token, tokenResp.Token.User.ID, "")
			if err != nil {
				return connector.Identity{}, false, err
			}
			userGroups = append(userGroups, roleGroups...)
		}
		identity.Groups = p.filterGroups(pruneDuplicates(userGroups))
	}
	identity.Username = username
	identity.UserID = tokenResp.Token.User.ID

	user, err := p.getUser(ctx, tokenResp.Token.User.ID, token)
	if err != nil {
		return identity, false, err
	}
	if user.User.Email != "" {
		identity.Email = user.User.Email
		identity.EmailVerified = true
	}

	return identity, true, nil
}

func (p *conn) Prompt() string { return "username" }

func (p *conn) Refresh(
	ctx context.Context, scopes connector.Scopes, identity connector.Identity,
) (connector.Identity, error) {
	token, err := p.getAdminToken(ctx)
	if err != nil {
		return identity, fmt.Errorf("keystone: failed to obtain admin token: %v", err)
	}
	ok, err := p.checkIfUserExists(ctx, identity.UserID, token)
	if err != nil {
		return identity, err
	}
	if !ok {
		return identity, fmt.Errorf("keystone: user %q does not exist", identity.UserID)
	}
	if scopes.Groups {
		groups, err := p.getUserGroups(ctx, identity.UserID, token)
		if err != nil {
			return identity, err
		}
		identity.Groups = groups
	}
	return identity, nil
}

func (p *conn) getTokenResponse(ctx context.Context, username, pass string) (response *http.Response, err error) {
	client := &http.Client{}
	jsonData := loginRequestData{
		auth: auth{
			Identity: identity{
				Methods: []string{"password"},
				Password: password{
					User: user{
						Name:     username,
						Domain:   domain{ID: p.Domain},
						Password: pass,
					},
				},
			},
		},
	}
	jsonValue, err := json.Marshal(jsonData)
	if err != nil {
		return nil, err
	}
	// https://developer.openstack.org/api-ref/identity/v3/#password-authentication-with-unscoped-authorization
	authTokenURL := p.Host + "/v3/auth/tokens/"
	req, err := http.NewRequest("POST", authTokenURL, bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	return client.Do(req)
}

func (p *conn) getAdminToken(ctx context.Context) (string, error) {
	resp, err := p.getTokenResponse(ctx, p.AdminUsername, p.AdminPassword)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	token := resp.Header.Get("X-Subject-Token")
	return token, nil
}

func (p *conn) checkIfUserExists(ctx context.Context, userID string, token string) (bool, error) {
	user, err := p.getUser(ctx, userID, token)
	return user != nil, err
}

func (p *conn) getUser(ctx context.Context, userID string, token string) (*userResponse, error) {
	// https://developer.openstack.org/api-ref/identity/v3/#show-user-details
	userURL := p.Host + "/v3/users/" + userID
	client := &http.Client{}
	req, err := http.NewRequest("GET", userURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Auth-Token", token)
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	user := userResponse{}
	err = json.Unmarshal(data, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (p *conn) getUserGroups(ctx context.Context, userID string, token string) ([]string, error) {
	client := &http.Client{}
	// https://developer.openstack.org/api-ref/identity/v3/#list-groups-to-which-a-user-belongs
	groupsURL := p.Host + "/v3/users/" + userID + "/groups"
	req, err := http.NewRequest("GET", groupsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		p.Logger.Errorf("keystone: error while fetching user %q groups\n", userID)
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	groupsResp := new(groupsResponse)

	err = json.Unmarshal(data, &groupsResp)
	if err != nil {
		return nil, err
	}

	groups := make([]string, len(groupsResp.Groups))
	for i, group := range groupsResp.Groups {
		groups[i] = group.Name
	}
	return groups, nil
}

func (p *conn) groupsRequired(groupScope bool) bool {
	return len(p.Groups) > 0 || groupScope
}

// If project ID is left empty, all roles will be fetched
func (p *conn) getUserRolesAsGroups(ctx context.Context, token string, userID string, projectID string) ([]string, error) {
	roleAssignments, err := p.getRoleAssignments(ctx, token, userID, projectID)
	if err != nil {
		return nil, err
	}
	roles, err := p.getRoles(ctx, token)
	if err != nil {
		return nil, err
	}
	roleMap := map[string]role{}
	for _, role := range roles {
		roleMap[role.ID] = role
	}
	var groups []string
	for _, roleAssignment := range roleAssignments {
		role, ok := roleMap[roleAssignment.Role.ID]
		if !ok {
			// Ignore role assignments to non-existent roles (shouldn't happen)
			continue
		}
		groups = append(groups, role.Name)
	}
	return groups, nil
}

func (p *conn) getRoleAssignments(ctx context.Context, token string, userID string, projectID string) ([]roleAssignment, error) {
	// https://docs.openstack.org/api-ref/identity/v3/?expanded=validate-and-show-information-for-token-detail,list-role-assignments-detail#list-role-assignments
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v3/role_assignments?effective&user.id=%s&scope.project.id=%s", p.Host, userID, projectID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)
	req = req.WithContext(ctx)
	resp, err := p.client.Do(req)
	if err != nil {
		p.Logger.Errorf("keystone: error while fetching user %q groups\n", userID)
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	roleAssignmentResp := struct {
		RoleAssignments []roleAssignment `json:"role_assignments"`
	}{}

	err = json.Unmarshal(data, &roleAssignmentResp)
	if err != nil {
		return nil, err
	}
	return roleAssignmentResp.RoleAssignments, nil
}

func (p *conn) getRoles(ctx context.Context, token string) ([]role, error) {
	// https://docs.openstack.org/api-ref/identity/v3/?expanded=validate-and-show-information-for-token-detail,list-role-assignments-detail,list-roles-detail#list-roles
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v3/roles", p.Host), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)
	req = req.WithContext(ctx)
	resp, err := p.client.Do(req)
	if err != nil {
		p.Logger.Errorf("keystone: error while fetching keystone roles\n")
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rolesResp := struct {
		Roles []role `json:"roles"`
	}{}

	err = json.Unmarshal(data, &rolesResp)
	if err != nil {
		return nil, err
	}

	return rolesResp.Roles, nil
}

func (p *conn) filterGroups(groups []string) []string {
	if len(p.Groups) == 0 {
		return groups
	}
	var matches []string
	for _, group := range groups {
		for _, filter := range p.Groups {
			// Future: support regexp?
			if group != filter.Name {
				continue
			}
			if len(filter.Replace) > 0 {
				group = filter.Replace
			}
			matches = append(matches, group)
		}
	}
	return matches
}

func pruneDuplicates(ss []string) []string {
	set := map[string]struct{}{}
	var ns []string
	for _, s := range ss {
		if _, ok := set[s]; ok {
			continue
		}
		set[s] = struct{}{}
		ns = append(ns, s)
	}
	return ns
}
