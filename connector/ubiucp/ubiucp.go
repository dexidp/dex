package ubiucp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

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
	// Encode username and password as a Basic Auth header value
	// auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
	requestBody, err := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})

	// Send HTTP request with Basic Auth header
	req, err := http.NewRequestWithContext(ctx, "POST", u.authURL, bytes.NewReader(requestBody))
	if err != nil {
		return connector.Identity{}, false, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return connector.Identity{}, false, err
	}
	defer resp.Body.Close()

	// Check HTTP response status code
	if resp.StatusCode == http.StatusOK {
		return connector.Identity{
			UserID:   username,
			Username: username,
		}, true, nil
	} else if resp.StatusCode == http.StatusBadRequest {
		return connector.Identity{}, false, nil
	} else {
		return connector.Identity{}, false, fmt.Errorf("unexpected response code: %v", resp.StatusCode)
	}
}
