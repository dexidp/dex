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
		tokenInfo, err = getTokenInfo(ctx, p.client, p.Host, token, p.Logger)
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
		p.Logger.Debug("groups scope requested, fetching groups")
		var err error
		adminToken, err := getAdminTokenUnscoped(ctx, p.client, p.Host, p.AdminUsername, p.AdminPassword)
		if err != nil {
			p.Logger.Error("failed to obtain admin token", "error", err)
			return identity, false, err
		}
		identity.Groups, err = getAllGroupsForUser(ctx, p.client, p.Host, adminToken, p.CustomerName, p.Domain.Name, tokenInfo, p.Logger)
		if err != nil {
			return connector.Identity{}, false, err
		}
	}
	identity.Username = tokenInfo.User.Name
	identity.UserID = tokenInfo.User.ID

	user, err := getUser(ctx, p.client, p.Host, tokenInfo.User.ID, token)
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
		p.Logger.Error("failed to marshal connector data", "error", err)
		return identity, false, err
	}
	identity.ConnectorData = connData

	return identity, true, nil
}

func (p *conn) Prompt() string { return "username" }

func (p *conn) Refresh(
	ctx context.Context, scopes connector.Scopes, identity connector.Identity,
) (connector.Identity, error) {
	token, err := getAdminTokenUnscoped(ctx, p.client, p.Host, p.AdminUsername, p.AdminPassword)
	if err != nil {
		p.Logger.Error("failed to obtain admin token", "error", err)
		return identity, err
	}

	ok, err := p.checkIfUserExists(ctx, identity.UserID, token)
	if err != nil {
		return identity, err
	}
	if !ok {
		p.Logger.Error("user does not exist", "userID", identity.UserID)
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
		p.Logger.Error("failed to unmarshal token info", "error", err)
		return identity, err
	}
	// If there is a token associated with this refresh token, use that to look up
	// the token info. This can contain things like SSO groups which are not present elsewhere.
	if len(data.Token) > 0 {
		tokenInfo, err = getTokenInfo(ctx, p.client, p.Host, data.Token, p.Logger)
		if err != nil {
			return identity, err
		}
	}

	if scopes.Groups {
		var err error
		identity.Groups, err = getAllGroupsForUser(ctx, p.client, p.Host, token, p.CustomerName, p.Domain.Name, tokenInfo, p.Logger)
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

	// Build auth token URL preserving any base path in p.Host (e.g., /keystone)
	authTokenURL, err := url.JoinPath(p.Host, "v3", "auth", "tokens")
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequest("POST", authTokenURL, bytes.NewBuffer(jsonValue))
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	resp, err := p.client.Do(req)
	if err != nil {
		p.Logger.Error("keystone authentication request failed", "error", err)
		return "", nil, err
	}
	if resp.StatusCode/100 != 2 {
		p.Logger.Error("keystone login failed", "statusCode", resp.StatusCode)
		return "", nil, fmt.Errorf("keystone login: URL %s error %v", authTokenURL, resp.StatusCode)
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
		p.Logger.Error("invalid token response", "error", err)
		return "", nil, err
	}
	return token, &tokenResp.Token, nil
}

func getAdminTokenUnscoped(ctx context.Context, client *http.Client, baseURL, adminUsername, adminPassword string) (string, error) {
	domain := domainKeystone{
		Name: "Default",
	}
	jsonData := loginRequestData{
		auth: auth{
			Identity: identity{
				Methods: []string{"password"},
				Password: password{
					User: user{
						Name:     adminUsername,
						Domain:   domain,
						Password: adminPassword,
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
	authTokenURL, err := url.JoinPath(baseURL, "v3", "auth", "tokens")
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", authTokenURL, bytes.NewBuffer(jsonValue))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
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
	user, err := getUser(ctx, p.client, p.Host, userID, token)
	return user != nil, err
}

// getAllKeystoneGroups returns all groups in keystone
func getAllKeystoneGroups(ctx context.Context, client *http.Client, baseURL, token string) ([]keystoneGroup, error) {
	// https://docs.openstack.org/api-ref/identity/v3/?expanded=list-groups-detail#list-groups
	groupsURL, err := url.JoinPath(baseURL, "v3", "groups")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, groupsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
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

// getUserLocalGroups returns local groups for a user
func getUserLocalGroups(ctx context.Context, client *http.Client, baseURL, userID, token string) ([]keystoneGroup, error) {
	// https://developer.openstack.org/api-ref/identity/v3/#list-groups-to-which-a-user-belongs
	groupsURL, err := url.JoinPath(baseURL, "v3", "users", userID, "groups")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", groupsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
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

// getRoleAssignments returns role assignments for a user or group
func getRoleAssignments(ctx context.Context, client *http.Client, baseURL, token string, opts getRoleAssignmentsOptions, logger *slog.Logger) ([]roleAssignment, error) {
	endpoint, err := url.JoinPath(baseURL, "v3", "role_assignments")
	if err != nil {
		return nil, err
	}
	if len(opts.userID) > 0 {
		endpoint = fmt.Sprintf("%s?effective&user.id=%s", endpoint, opts.userID)
	} else if len(opts.groupID) > 0 {
		endpoint = fmt.Sprintf("%s?group.id=%s", endpoint, opts.groupID)
	}

	// https://docs.openstack.org/api-ref/identity/v3/?expanded=validate-and-show-information-for-token-detail,list-role-assignments-detail#list-role-assignments
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("failed to fetch role assignments", "error", err)
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

// getRoles returns all roles in keystone
func getRoles(ctx context.Context, client *http.Client, baseURL, token string, logger *slog.Logger) ([]role, error) {
	// https://docs.openstack.org/api-ref/identity/v3/?expanded=validate-and-show-information-for-token-detail,list-role-assignments-detail,list-roles-detail#list-roles
	rolesURL, err := url.JoinPath(baseURL, "v3", "roles")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, rolesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("failed to fetch keystone roles", "error", err)
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

// getProjects returns all projects in keystone
func getProjects(ctx context.Context, client *http.Client, baseURL, token string, logger *slog.Logger) ([]project, error) {
	// https://docs.openstack.org/api-ref/identity/v3/?expanded=validate-and-show-information-for-token-detail,list-role-assignments-detail,list-roles-detail#list-roles
	projectsURL, err := url.JoinPath(baseURL, "v3", "projects")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, projectsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("failed to fetch keystone projects", "error", err)
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

func getUser(ctx context.Context, client *http.Client, baseURL, userID, token string) (*userResponse, error) {
	// https://developer.openstack.org/api-ref/identity/v3/#show-user-details
	userURL, err := url.JoinPath(baseURL, "v3", "users", userID)
	if err != nil {
		return nil, err
	}
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

// getAllGroupsForUser returns all groups for a user (local groups + SSO groups + role groups)
func getAllGroupsForUser(ctx context.Context, client *http.Client, baseURL, token, customerName, domainID string, tokenInfo *tokenInfo, logger *slog.Logger) ([]string, error) {
	var userGroups []string   //nolint:prealloc
	var userGroupIDs []string //nolint:prealloc

	allGroups, err := getAllKeystoneGroups(ctx, client, baseURL, token)
	if err != nil {
		return nil, err
	}

	// 1. Get SSO groups
	// For SSO users, groups are passed down through the federation API.
	if tokenInfo.User.OSFederation != nil {
		for _, osGroup := range tokenInfo.User.OSFederation.Groups {
			// If group name is empty, try to find the group by ID
			if len(osGroup.Name) == 0 {
				var ok bool
				osGroup, ok = findGroupByID(allGroups, osGroup.ID)
				if !ok {
					logger.Warn("SSO group not found, skipping", "groupID", osGroup.ID, "userID", tokenInfo.User.ID)
					continue
				}
			}
			userGroups = append(userGroups, osGroup.Name)
			userGroupIDs = append(userGroupIDs, osGroup.ID)
		}
	}

	// 2. Get local groups
	// For local users, fetch the groups stored in Keystone.
	localGroups, err := getUserLocalGroups(ctx, client, baseURL, tokenInfo.User.ID, token)
	if err != nil {
		return nil, err
	}

	for _, localGroup := range localGroups {
		// If group name is empty, try to find the group by ID
		if len(localGroup.Name) == 0 {
			var ok bool
			localGroup, ok = findGroupByID(allGroups, localGroup.ID)
			if !ok {
				logger.Warn("local group not found, skipping", "groupID", localGroup.ID, "userID", tokenInfo.User.ID)
				continue
			}
		}
		userGroups = append(userGroups, localGroup.Name)
		userGroupIDs = append(userGroupIDs, localGroup.ID)
	}

	// Get user-related role assignments
	roleAssignments := []roleAssignment{}
	localUserRoleAssignments, err := getRoleAssignments(ctx, client, baseURL, token, getRoleAssignmentsOptions{
		userID: tokenInfo.User.ID,
	}, logger)
	if err != nil {
		logger.Error("failed to fetch role assignments for user", "userID", tokenInfo.User.ID, "error", err)
		return userGroups, err
	}
	roleAssignments = append(roleAssignments, localUserRoleAssignments...)

	// Get group-related role assignments
	for _, groupID := range userGroupIDs {
		groupRoleAssignments, err := getRoleAssignments(ctx, client, baseURL, token, getRoleAssignmentsOptions{
			groupID: groupID,
		}, logger)
		if err != nil {
			logger.Error("failed to fetch role assignments for group", "groupID", groupID, "error", err)
			return userGroups, err
		}
		roleAssignments = append(roleAssignments, groupRoleAssignments...)
	}

	if len(roleAssignments) == 0 {
		logger.Warn("no role assignments found")
		return userGroups, nil
	}

	roles, err := getRoles(ctx, client, baseURL, token, logger)
	if err != nil {
		return userGroups, err
	}
	roleMap := map[string]role{}
	for _, role := range roles {
		roleMap[role.ID] = role
	}

	projects, err := getProjects(ctx, client, baseURL, token, logger)
	if err != nil {
		return userGroups, err
	}
	projectMap := map[string]project{}
	for _, project := range projects {
		projectMap[project.ID] = project
	}

	// 3. Now create groups based on the role assignments
	roleGroups := make([]string, 0, len(roleAssignments))

	// get the customer name to be prefixed in the group name
	// if customerName is not provided in the keystone config get it from keystone host url.
	if customerName == "" {
		customerName, err = getHostname(baseURL)
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
		groupName := generateGroupName(project, role, customerName, domainID)
		roleGroups = append(roleGroups, groupName)
	}

	// combine local groups + sso groups + role groups
	userGroups = append(userGroups, roleGroups...)
	return pruneDuplicates(userGroups), nil
}

func getTokenInfo(ctx context.Context, client *http.Client, baseURL, token string, logger *slog.Logger) (*tokenInfo, error) {
	// https://developer.openstack.org/api-ref/identity/v3/#password-authentication-with-unscoped-authorization
	authTokenURL, err := url.JoinPath(baseURL, "v3", "auth", "tokens")
	if err != nil {
		return nil, err
	}
	logger.Debug("fetching keystone token info", "url", authTokenURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, authTokenURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("X-Subject-Token", token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		logger.Error("failed to get token info", "statusCode", resp.StatusCode, "response", strings.ReplaceAll(string(data), "\n", ""))
		return nil, fmt.Errorf("keystone: get token info: error status code %d", resp.StatusCode)
	}

	tokenResp := &tokenResponse{}
	err = json.Unmarshal(data, tokenResp)
	if err != nil {
		return nil, err
	}

	return &tokenResp.Token, nil
}

func pruneDuplicates(ss []string) []string {
	set := make(map[string]struct{}, len(ss))
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

// generateGroupName generates a group name based on project, role, customer name, and domain ID
func generateGroupName(project project, role role, customerName, domainID string) string {
	roleName := role.Name
	if roleName == "_member_" {
		roleName = "member"
	}
	domainName := strings.ToLower(strings.ReplaceAll(domainID, "_", "-"))
	projectName := strings.ToLower(strings.ReplaceAll(project.Name, "_", "-"))
	return customerName + "-" + domainName + "-" + projectName + "-" + roleName
}

func findGroupByID(groups []keystoneGroup, groupID string) (group keystoneGroup, ok bool) {
	for _, group := range groups {
		if group.ID == groupID {
			return group, true
		}
	}
	return group, false
}

// getHostname returns the hostname from the base URL
func getHostname(baseURL string) (string, error) {
	keystoneURL := baseURL
	parsedURL, err := url.Parse(keystoneURL)
	if err != nil {
		return "", err
	}
	customerFqdn := parsedURL.Hostname()
	// get customer name and not the full fqdn
	parts := strings.Split(customerFqdn, ".")
	hostName := parts[0]

	return hostName, nil
}
