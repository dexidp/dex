package onelogin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type (
	apiPagination struct {
		BeforeCursor string `json:"before_cursor"`
		AfterCursor  string `json:"after_cursor"`
		PreviousLink string `json:"previous_link"`
		NextLink     string `json:"next_link"`
	}
	apiResponse struct {
		Status     apiStatus
		Pagination apiPagination
		Data       []interface{}
	}
	apiRole struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	apiRoleResponse struct {
		Status     apiStatus
		Pagination apiPagination
		Data       []apiRole
	}
	apiStatus struct {
		Error   bool   `json:"error"`
		Code    int    `json:"code"`
		Type    string `json:"type"`
		Message string `json:"message"`
	}
	apiToken struct {
		AccessToken  string `json:"access_token"`
		CreatedAt    string `json:"created_at"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		AccountID    int    `json:"account_id"`
	}
	apiTokenResponse struct {
		Status apiStatus
		Data   []apiToken
	}
	apiUserRolesResponse struct {
		Status     apiStatus
		Pagination apiPagination
		Data       [][]int `json:"data"`
	}
)

var (
	roleNames map[int]string
)

func doRequest(req *http.Request) (*[]byte, error) {
	client := http.DefaultClient

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed: %s: %s", resp.Status, body)
	}
	defer resp.Body.Close()

	return &body, nil
}

// Get Onelogin API token
func getAPIAccessToken(auth string) (string, error) {
	var jsonStr = []byte(`{"grant_type":"client_credentials"}`)
	req, err := http.NewRequest("POST", "https://api.us.onelogin.com/auth/oauth2/token", bytes.NewBuffer(jsonStr))
	if err != nil {
		return "", fmt.Errorf("onelogin: failed to setup api token request: %v", err)
	}
	req.Header.Add("Authorization", auth)
	req.Header.Add("Content-Type", "application/json")

	body, err := doRequest(req)
	var token apiTokenResponse
	if err := json.Unmarshal(*body, &token); err != nil {
		return "", err
	}
	fmt.Printf("%+v\n", token)
	return token.Data[0].AccessToken, nil
}

func getAPIUserRoles(rolesPrefix, authToken, userID string) ([]string, error) {
	req, err := http.NewRequest("GET", "https://api.us.onelogin.com/api/1/users/"+userID+"/roles", nil)
	if err != nil {
		return nil, fmt.Errorf("onelogin: failed to setup user roles request: %v", err)
	}
	req.Header.Add("Authorization", "bearer:"+authToken)

	body, err := doRequest(req)
	var userRoleIDs apiUserRolesResponse
	if err := json.Unmarshal(*body, &userRoleIDs); err != nil {
		return nil, err
	}
	fmt.Printf("%+v\n", userRoleIDs)

	var userRoles []string
	for i, g := range userRoleIDs.Data[0] {
		r, ok := roleNames[i]
		if !ok {
			r, err = getAPIRoleName(authToken, g)
			if err != nil {
				return nil, err
			}
			roleNames[i] = r
		}
		if strings.HasPrefix(r, rolesPrefix) {
			userRoles = append(userRoles, strings.TrimPrefix(r, rolesPrefix))
		}
	}
	return userRoles, nil
}

func getAPIRoleName(authToken string, roleID int) (string, error) {
	req, err := http.NewRequest("GET", "https://api.us.onelogin.com/api/1/roles?id="+strconv.Itoa(roleID), nil)
	if err != nil {
		return "", fmt.Errorf("onelogin: failed to setup role request: %v", err)
	}
	req.Header.Add("Authorization", "bearer:"+authToken)

	body, err := doRequest(req)
	var role apiRoleResponse
	if err := json.Unmarshal(*body, &role); err != nil {
		return "", err
	}
	fmt.Printf("%+v\n", role)

	return role.Data[0].Name, nil
}
