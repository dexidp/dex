package ubiucp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/dexidp/dex/pkg/log"

	"net/http"

	"github.com/dexidp/dex/connector"
)

var (
	_ connector.PasswordConnector = ubiucpConnector{}
)

type UbiucpConfig struct {
	AuthURL string `json:"authURL"`
}

// Open returns an authentication strategy which prompts for a predefined username and password.
func (u *UbiucpConfig) Open(id string, logger log.Logger) (connector.Connector, error) {
	return &ubiucpConnector{logger, u.AuthURL}, nil
}

type ubiucpConnector struct {
	logger  log.Logger
	authURL string
}

// Prompt implements PasswordConnector interface
func (u ubiucpConnector) Prompt() string {
	return ""
}

func (u ubiucpConnector) Close() error {
	return nil
}

func (u ubiucpConnector) Refresh(_ context.Context, _ connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	return identity, nil
}

// Login implements PasswordConnector interface
func (u ubiucpConnector) Login(ctx context.Context, s connector.Scopes, username, password string) (identity connector.Identity, validPassword bool, err error) {
	// Read the session cookie from the request
	cookie, err := getSessionCookie("user", "pass")
	if err != nil {
		return connector.Identity{}, false, err
	}

	// Call the auth endpoint to validate the session cookie
	req, err := http.NewRequest("GET", "http://47.100.113.76:8080/auth", nil)

	if err != nil {
		fmt.Printf("err is %v", err)
		return
	}
	req.AddCookie(cookie)
	fmt.Printf("add cookie %s\n", cookie)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("err is %v", err)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("err is %v", err)
		return
	}
	// Return user info
	userInfo := string(body)
	fmt.Printf("userInfo is %s\n", userInfo)

	defer resp.Body.Close()

	if err != nil {
		return connector.Identity{}, false, err
	}
	defer resp.Body.Close()

	// If the auth endpoint returns a non-200 status code, the authentication failed
	if resp.StatusCode != http.StatusOK {
		return connector.Identity{}, false, fmt.Errorf("authentication failed: %v", resp.Status)
	}

	// If the authentication succeeded, create an identity object and return it
	identity = connector.Identity{
		UserID:        "user",
		Username:      "user@example.com",
		Email:         "user@example.com",
		ConnectorData: []byte(cookie.Value),
	}
	return identity, true, nil
}

func getSessionCookie(username string, password string) (*http.Cookie, error) {
	requestBody, err := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "http://localhost:8080/login", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login failed with status code: %d", resp.StatusCode)
	}

	cookies := resp.Cookies()
	fmt.Println(cookies)
	if len(cookies) == 0 {
		return nil, errors.New("no cookie found")
	}

	return cookies[0], nil
}
