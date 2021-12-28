package external

import (
	"context"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/connector/external/sdk"
	"github.com/dexidp/dex/pkg/log"
)

type PasswordConnectorConfig struct {
	gRPCConnectorConfig

	id     string
	logger log.Logger

	client sdk.PasswordConnectorClient
}

func (c *PasswordConnectorConfig) Open(id string, logger log.Logger) (connector.Connector, error) {
	conn, err := grpcConn(c.Port, c.CAPath, c.ClientCrt, c.ClientKey)
	if err != nil {
		return connector.Identity{}, err
	}

	c.id = id
	c.client = sdk.NewPasswordConnectorClient(conn)
	c.logger = logger
	return c, nil
}

func (c *PasswordConnectorConfig) Prompt() string {
	prompt := "username"

	response, err := c.client.Prompt(context.Background(), &sdk.PromptReq{})
	if err != nil {
		c.logger.Errorf(err.Error())
		return prompt
	}

	if response.Prompt != "" {
		prompt = response.Prompt
	}

	return prompt
}

func (c *PasswordConnectorConfig) Login(ctx context.Context, scopes connector.Scopes, username, password string) (identity connector.Identity, validPassword bool, err error) {
	response, err := c.client.Login(ctx, &sdk.LoginReq{
		Scopes: toSDKScopes(scopes), Username: username, Password: password,
	})
	if err != nil {
		return connector.Identity{}, false, err
	}

	return toConnectorIdentity(response.GetIdentity()), response.GetValidPassword(), nil
}

func (c *PasswordConnectorConfig) Refresh(ctx context.Context, scopes connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	response, err := c.client.Refresh(ctx, &sdk.RefreshReq{
		Scopes: toSDKScopes(scopes), Identity: toSDKIdentity(identity),
	})
	if err != nil {
		return connector.Identity{}, err
	}

	return toConnectorIdentity(response.GetIdentity()), nil
}
