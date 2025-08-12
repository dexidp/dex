// Package keystone provides authentication strategy using Keystone.
package keystone

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/dexidp/dex/connector"
)

type conn struct {
	Domain        domainKeystone
	Host          string
	AdminUsername string
	AdminPassword string
	client        *http.Client
	Logger        *slog.Logger
	CustomerName  string
}

// type group struct {
// 	Name    string `json:"name"`
// 	Replace string `json:"replace"`
// }

type userKeystone struct {
	Domain       domainKeystone `json:"domain"`
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	OSFederation *struct {
		Groups           []keystoneGroup `json:"groups"`
		IdentityProvider struct {
			ID string `json:"id"`
		} `json:"identity_provider"`
		Protocol struct {
			ID string `json:"id"`
		} `json:"protocol"`
	} `json:"OS-FEDERATION"`
}

type domainKeystone struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
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
	Domain             string `json:"domain"`
	Host               string `json:"keystoneHost"`
	AdminUsername      string `json:"keystoneUsername"`
	AdminPassword      string `json:"keystonePassword"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify"`
	CustomerName       string `json:"customerName"`
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
	Name     string         `json:"name"`
	Domain   domainKeystone `json:"domain"`
	Password string         `json:"password"`
}

// type domain struct {
// 	ID string `json:"id"`
// }

type tokenInfo struct {
	User  userKeystone `json:"user"`
	Roles []role       `json:"roles"`
}

type tokenResponse struct {
	Token tokenInfo `json:"token"`
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

type project struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DomainID    string `json:"domain_id"`
	Description string `json:"description"`
}
type identifierContainer struct {
	ID string `json:"id"`
}

type projectScope struct {
	Project identifierContainer `json:"project"`
}

type roleAssignment struct {
	Scope projectScope        `json:"scope"`
	User  identifierContainer `json:"user"`
	Role  identifierContainer `json:"role"`
}

type connectorData struct {
	Token string `json:"token"`
}

var (
	_ connector.PasswordConnector = &conn{}
	_ connector.RefreshConnector  = &conn{}
)

// Open returns an authentication strategy using Keystone.
func (c *Config) Open(id string, logger *slog.Logger) (connector.Connector, error) {
	domain := domainKeystone{
		Name: c.Domain,
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: c.InsecureSkipVerify,
		},
	}
	client := &http.Client{Transport: tr}
	return &conn{
		Domain:        domain,
		Host:          c.Host,
		AdminUsername: c.AdminUsername,
		AdminPassword: c.AdminPassword,
		Logger:        logger.With(slog.Group("connector", "type", "keystone", "id", id)),
		client:        client,
		CustomerName:  c.CustomerName,
	}, nil
}

func (p *conn) Close() error { return nil }

func (p *conn) Login(ctx context.Context, scopes connector.Scopes, username, password string) (identity connector.Identity, validPassword bool, err error) {
	var token string
	var tokenInfo *tokenInfo
	if username == "" || username == "_TOKEN_" {
		token = password
		tokenInfo, err = p.getTokenInfo(ctx, token)
		if err != nil {
			return connector.Identity{}, false, err
		}
	} else {
		token, tokenInfo, err = p.authenticate(ctx, username, password)
		if err != nil || tokenInfo == nil {
			return identity, false, err
		}
	}

	if scopes.Groups {
		p.Logger.Info("groups scope requested, fetching groups")
		var err error
		adminToken, err := p.getAdminTokenUnscoped(ctx)
		if err != nil {
			return identity, false, fmt.Errorf("keystone: failed to obtain admin token: %v", err)
		}
		identity.Groups, err = p.getGroups(ctx, adminToken, tokenInfo)
		if err != nil {
			return connector.Identity{}, false, err
		}
	}
	identity.Username = tokenInfo.User.Name
	identity.UserID = tokenInfo.User.ID

	user, err := p.getUser(ctx, tokenInfo.User.ID, token)
	if err != nil {
		return identity, false, err
	}
	if user.User.Email != "" {
		identity.Email = user.User.Email
		identity.EmailVerified = true
	}

	data := connectorData{Token: token}
	connData, err := json.Marshal(data)
	if err != nil {
		return identity, false, fmt.Errorf("marshal connector data: %v", err)
	}
	identity.ConnectorData = connData

	return identity, true, nil
}

func (p *conn) Prompt() string { return "username" }

func (p *conn) Refresh(
	ctx context.Context, scopes connector.Scopes, identity connector.Identity,
) (connector.Identity, error) {
	token, err := p.getAdminTokenUnscoped(ctx)
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

	tokenInfo := &tokenInfo{
		User: userKeystone{
			Name: identity.Username,
			ID:   identity.UserID,
		},
	}
	var data connectorData
	if err := json.Unmarshal(identity.ConnectorData, &data); err != nil {
		return identity, fmt.Errorf("keystone: unmarshal token info: %v", err)
	}
	// If there is a token associated with this refresh token, use that to look up
	// the token info. This can contain things like SSO groups which are not present elsewhere.
	if len(data.Token) > 0 {
		tokenInfo, err = p.getTokenInfo(ctx, data.Token)
		if err != nil {
			return identity, err
		}
	}

	if scopes.Groups {
		var err error
		identity.Groups, err = p.getGroups(ctx, token, tokenInfo)
		if err != nil {
			return identity, err
		}
	}
	return identity, nil
}

func (p *conn) authenticate(ctx context.Context, username, pass string) (string, *tokenInfo, error) {
	jsonData := loginRequestData{
		auth: auth{
			Identity: identity{
				Methods: []string{"password"},
				Password: password{
					User: user{
						Name:     username,
						Domain:   p.Domain,
						Password: pass,
					},
				},
			},
		},
	}
	jsonValue, err := json.Marshal(jsonData)
	if err != nil {
		return "", nil, err
	}
	// https://developer.openstack.org/api-ref/identity/v3/#password-authentication-with-unscoped-authorization
	authTokenURL := p.Host + "/v3/auth/tokens/"
	req, err := http.NewRequest("POST", authTokenURL, bytes.NewBuffer(jsonValue))
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("keystone: error %v", err)
	}
	if resp.StatusCode/100 != 2 {
		return "", nil, fmt.Errorf("keystone login: error %v", resp.StatusCode)
	}
	if resp.StatusCode != 201 {
		return "", nil, nil
	}
	token := resp.Header.Get("X-Subject-Token")
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	tokenResp := &tokenResponse{}
	err = json.Unmarshal(data, tokenResp)
	if err != nil {
		return "", nil, fmt.Errorf("keystone: invalid token response: %v", err)
	}
	return token, &tokenResp.Token, nil
}

func (p *conn) getAdminTokenUnscoped(ctx context.Context) (string, error) {
	domain := domainKeystone{
		Name: "Default",
	}
	jsonData := loginRequestData{
		auth: auth{
			Identity: identity{
				Methods: []string{"password"},
				Password: password{
					User: user{
						Name:     p.AdminUsername,
						Domain:   domain,
						Password: p.AdminPassword,
					},
				},
			},
		},
	}
	jsonValue, err := json.Marshal(jsonData)
	if err != nil {
		return "", err
	}
	// https://developer.openstack.org/api-ref/identity/v3/#password-authentication-with-unscoped-authorization
	authTokenURL := p.Host + "/v3/auth/tokens/"
	req, err := http.NewRequest("POST", authTokenURL, bytes.NewBuffer(jsonValue))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("keystone: error %v", err)
	}
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("keystone login: error %v", resp.StatusCode)
	}
	if resp.StatusCode != 201 {
		return "", nil
	}
	return resp.Header.Get("X-Subject-Token"), nil
}

func (p *conn) checkIfUserExists(ctx context.Context, userID string, token string) (bool, error) {
	user, err := p.getUser(ctx, userID, token)
	return user != nil, err
}

func (p *conn) getGroups(ctx context.Context, token string, tokenInfo *tokenInfo) ([]string, error) {
	var userGroups []string   //nolint:prealloc
	var userGroupIDs []string //nolint:prealloc

	allGroups, err := p.getAllGroups(ctx, token)
	if err != nil {
		return nil, err
	}

	// For SSO users, groups are passed down through the federation API.
	if tokenInfo.User.OSFederation != nil {
		for _, osGroup := range tokenInfo.User.OSFederation.Groups {
			// If grouop name is empty, try to find the group by ID
			if len(osGroup.Name) == 0 {
				var ok bool
				osGroup, ok = findGroupByID(allGroups, osGroup.ID)
				if !ok {
					p.Logger.Warn("GroupID attached to user could not be found. Skipping.", "group_id", osGroup.ID, "user_id", tokenInfo.User.ID)
					continue
				}
			}
			userGroups = append(userGroups, osGroup.Name)
			userGroupIDs = append(userGroupIDs, osGroup.ID)
		}
	}

	// For local users, fetch the groups stored in Keystone.
	localGroups, err := p.getUserGroups(ctx, tokenInfo.User.ID, token)
	if err != nil {
		return nil, err
	}

	for _, localGroup := range localGroups {
		// If group name is empty, try to find the group by ID
		if len(localGroup.Name) == 0 {
			var ok bool
			localGroup, ok = findGroupByID(allGroups, localGroup.ID)
			if !ok {
				p.Logger.Warn("Group with ID attached to user could not be found. Skipping.", "group_id", localGroup.ID, "user_id", tokenInfo.User.ID)
				continue
			}
		}
		userGroups = append(userGroups, localGroup.Name)
		userGroupIDs = append(userGroupIDs, localGroup.ID)
	}

	// Get user-related role assignments
	roleAssignments := []roleAssignment{}
	localUserRoleAssignments, err := p.getRoleAssignments(ctx, token, getRoleAssignmentsOptions{
		userID: tokenInfo.User.ID,
	})
	if err != nil {
		p.Logger.Error("failed to fetch role assignments for userID", "userID", tokenInfo.User.ID, "error", err)
		return userGroups, err
	}
	roleAssignments = append(roleAssignments, localUserRoleAssignments...)

	// Get group-related role assignments
	for _, groupID := range userGroupIDs {
		groupRoleAssignments, err := p.getRoleAssignments(ctx, token, getRoleAssignmentsOptions{
			groupID: groupID,
		})
		if err != nil {
			p.Logger.Error("failed to fetch role assignments for groupID", "groupID", groupID, "error", err)
			return userGroups, err
		}
		roleAssignments = append(roleAssignments, groupRoleAssignments...)
	}

	if len(roleAssignments) == 0 {
		p.Logger.Warn("Warning: no role assignments found.")
		return userGroups, nil
	}

	roles, err := p.getRoles(ctx, token)
	if err != nil {
		return userGroups, err
	}
	roleMap := map[string]role{}
	for _, role := range roles {
		roleMap[role.ID] = role
	}

	projects, err := p.getProjects(ctx, token)
	if err != nil {
		return userGroups, err
	}
	projectMap := map[string]project{}
	for _, project := range projects {
		projectMap[project.ID] = project
	}

	//  Now create groups based on the role assignments
	roleGroups := make([]string, 0, len(roleAssignments))

	// get the customer name to be prefixed in the group name
	customerName := p.CustomerName
	// if customerName is not provided in the keystone config get it from keystone host url.
	if customerName == "" {
		customerName, err = p.getHostname()
		if err != nil {
			return userGroups, err
		}
	}
	for _, roleAssignment := range roleAssignments {
		role, ok := roleMap[roleAssignment.Role.ID]
		if !ok {
			// Ignore role assignments to non-existent roles (shouldn't happen)
			continue
		}
		project, ok := projectMap[roleAssignment.Scope.Project.ID]
		if !ok {
			// Ignore role assignments to non-existent projects (shouldn't happen)
			continue
		}
		groupName := p.generateGroupName(project, role, customerName)
		roleGroups = append(roleGroups, groupName)
	}

	// combine user-groups and role-groups
	userGroups = append(userGroups, roleGroups...)
	return pruneDuplicates(userGroups), nil
}

func (p *conn) getHostname() (string, error) {
	keystoneURL := p.Host
	parsedURL, err := url.Parse(keystoneURL)
	if err != nil {
		return "", fmt.Errorf("error parsing URL: %v", err)
	}
	customerFqdn := parsedURL.Hostname()
	// get customer name and not the full fqdn
	parts := strings.Split(customerFqdn, ".")
	hostName := parts[0]

	return hostName, nil
}

func (p *conn) generateGroupName(project project, role role, customerName string) string {
	roleName := role.Name
	if roleName == "_member_" {
		roleName = "member"
	}
	domainName := strings.ToLower(strings.ReplaceAll(p.Domain.Name, "_", "-"))
	projectName := strings.ToLower(strings.ReplaceAll(project.Name, "_", "-"))
	return customerName + "-" + domainName + "-" + projectName + "-" + roleName
}

func (p *conn) getUser(ctx context.Context, userID string, token string) (*userResponse, error) {
	// https://developer.openstack.org/api-ref/identity/v3/#show-user-details
	userURL := p.Host + "/v3/users/" + userID
	req, err := http.NewRequest("GET", userURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Auth-Token", token)
	req = req.WithContext(ctx)
	resp, err := p.client.Do(req)
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

func (p *conn) getTokenInfo(ctx context.Context, token string) (*tokenInfo, error) {
	// https://developer.openstack.org/api-ref/identity/v3/#password-authentication-with-unscoped-authorization
	authTokenURL := p.Host + "/v3/auth/tokens"
	p.Logger.Info("Fetching Keystone token info", "url", authTokenURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, authTokenURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("X-Subject-Token", token)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		p.Logger.Error("keystone: failed to get token info", "error_status_code", resp.StatusCode, "response", strings.ReplaceAll(string(data), "\n", ""))
		return nil, fmt.Errorf("keystone: get token info: error status code %d", resp.StatusCode)
	}

	tokenResp := &tokenResponse{}
	err = json.Unmarshal(data, tokenResp)
	if err != nil {
		return nil, err
	}

	return &tokenResp.Token, nil
}

func (p *conn) getAllGroups(ctx context.Context, token string) ([]keystoneGroup, error) {
	// https://docs.openstack.org/api-ref/identity/v3/?expanded=list-groups-detail#list-groups
	groupsURL := p.Host + "/v3/groups"
	req, err := http.NewRequest(http.MethodGet, groupsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)
	req = req.WithContext(ctx)
	resp, err := p.client.Do(req)
	if err != nil {
		p.Logger.Error("keystone: error while fetching groups")
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
	return groupsResp.Groups, nil
}

func (p *conn) getUserGroups(ctx context.Context, userID string, token string) ([]keystoneGroup, error) {
	// https://developer.openstack.org/api-ref/identity/v3/#list-groups-to-which-a-user-belongs
	groupsURL := p.Host + "/v3/users/" + userID + "/groups"
	req, err := http.NewRequest("GET", groupsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)
	req = req.WithContext(ctx)
	resp, err := p.client.Do(req)
	if err != nil {
		p.Logger.Error("error while fetching user groups", "user_id", userID, "err", err)
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
	return groupsResp.Groups, nil
}

type getRoleAssignmentsOptions struct {
	userID  string
	groupID string
}

func (p *conn) getRoleAssignments(ctx context.Context, token string, opts getRoleAssignmentsOptions) ([]roleAssignment, error) {
	endpoint := fmt.Sprintf("%s/v3/role_assignments?", p.Host)
	// note: group and user filters are mutually exclusive
	if len(opts.userID) > 0 {
		endpoint = fmt.Sprintf("%seffective&user.id=%s", endpoint, opts.userID)
	} else if len(opts.groupID) > 0 {
		endpoint = fmt.Sprintf("%sgroup.id=%s", endpoint, opts.groupID)
	}

	// https://docs.openstack.org/api-ref/identity/v3/?expanded=validate-and-show-information-for-token-detail,list-role-assignments-detail#list-role-assignments
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)
	req = req.WithContext(ctx)
	resp, err := p.client.Do(req)
	if err != nil {
		p.Logger.Error("keystone: error while fetching role assignments", "error", err)
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
		p.Logger.Error("keystone: error while fetching keystone roles", "error", err)
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

func (p *conn) getProjects(ctx context.Context, token string) ([]project, error) {
	// https://docs.openstack.org/api-ref/identity/v3/?expanded=validate-and-show-information-for-token-detail,list-role-assignments-detail,list-roles-detail#list-roles
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v3/projects", p.Host), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)
	req = req.WithContext(ctx)
	resp, err := p.client.Do(req)
	if err != nil {
		p.Logger.Error("keystone: error while fetching keystone projects", "error", err)
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	projectsResp := struct {
		Projects []project `json:"projects"`
	}{}

	err = json.Unmarshal(data, &projectsResp)
	if err != nil {
		return nil, err
	}

	return projectsResp.Projects, nil
}

func pruneDuplicates(ss []string) []string {
	set := map[string]struct{}{}
	ns := make([]string, 0, len(ss))
	for _, s := range ss {
		if _, ok := set[s]; ok {
			continue
		}
		set[s] = struct{}{}
		ns = append(ns, s)
	}
	return ns
}

func findGroupByID(groups []keystoneGroup, groupID string) (group keystoneGroup, ok bool) {
	for _, group := range groups {
		if group.ID == groupID {
			return group, true
		}
	}
	return group, false
}
