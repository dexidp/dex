package external

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/connector/external/sdk"
	"github.com/dexidp/dex/pkg/log"
)

type CallbackConnectorConfig struct {
	gRPCConnectorConfig

	id     string
	logger log.Logger

	client sdk.CallbackConnectorClient
}

func (c *CallbackConnectorConfig) Open(id string, logger log.Logger) (connector.Connector, error) {
	conn, err := grpcConn(c.Port, c.CAPath, c.ClientCrt, c.ClientKey)
	if err != nil {
		return connector.Identity{}, err
	}

	c.id = id
	c.client = sdk.NewCallbackConnectorClient(conn)
	c.logger = logger
	return c, nil
}

func (c *CallbackConnectorConfig) LoginURL(scopes connector.Scopes, callbackURL, state string) (string, error) {
	response, err := c.client.LoginURL(context.Background(), &sdk.LoginURLReq{
		Scopes: toSDKScopes(scopes), CallbackUrl: callbackURL, State: state,
	})
	if err != nil {
		return "", err
	}

	return response.Url, nil
}

func (c *CallbackConnectorConfig) HandleCallback(scopes connector.Scopes, r *http.Request) (connector.Identity, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return connector.Identity{}, fmt.Errorf("external connector %q: read body: %v", c.id, err)
	}

	defer r.Body.Close()

	headers := map[string][]string(r.Header)

	response, err := c.client.HandleCallback(r.Context(), &sdk.CallbackReq{
		Scopes:   toSDKScopes(scopes),
		Body:     body,
		Headers:  convertToListOfStrings(headers),
		RawQuery: r.URL.RawQuery,
	})
	if err != nil {
		return connector.Identity{}, err
	}

	return toConnectorIdentity(response.Identity), nil
}

func (c *CallbackConnectorConfig) Refresh(ctx context.Context, scopes connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	response, err := c.client.Refresh(ctx, &sdk.RefreshReq{
		Scopes: toSDKScopes(scopes), Identity: toSDKIdentity(identity),
	})
	if err != nil {
		return connector.Identity{}, err
	}

	return toConnectorIdentity(response.GetIdentity()), nil
}

func convertToListOfStrings(oldMap map[string][]string) map[string]*sdk.ListOfStrings {
	newMap := make(map[string]*sdk.ListOfStrings, len(oldMap))
	for k, v := range oldMap {
		newMap[k] = &sdk.ListOfStrings{Value: v}
	}

	return newMap
}
