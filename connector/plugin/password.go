package plugin

import (
	"context"
	"net/rpc"

	"github.com/dexidp/dex/connector"
	"github.com/hashicorp/go-plugin"
)

type ConfigurablePasswordConnector interface {
	connector.PasswordConnector
	ConfigurableConnector
}

type PasswordConnectorRPC struct{ client *rpc.Client }

func (g *PasswordConnectorRPC) Prompt() string {
	var resp string
	err := g.client.Call("Plugin.Prompt", new(any), &resp)
	if err != nil {
		panic(err)
	}

	return resp
}

type ConfigArgs struct {
	Config map[string]any
}

func (g *PasswordConnectorRPC) Config(config map[string]any) error {
	var resp error
	args := ConfigArgs{Config: config}
	err := g.client.Call("Plugin.Config", args, &resp)
	if err != nil {
		panic(err)
	}

	return resp
}

type LoginArgs struct {
	Scopes   connector.Scopes
	Username string
	Password string
}

type LoginResp struct {
	Identity      connector.Identity
	ValidPassword bool
}

func (g *PasswordConnectorRPC) Login(ctx context.Context, s connector.Scopes, username, password string) (identity connector.Identity, validPassword bool, err error) {
	var resp LoginResp
	args := LoginArgs{Scopes: s, Username: username, Password: password}
	err = g.client.Call("Plugin.Login", args, &resp)
	if err != nil {
		return connector.Identity{}, false, err
	}

	return resp.Identity, resp.ValidPassword, nil
}

type PasswordConnectorRPCServer struct {
	Impl ConfigurablePasswordConnector
}

func (s *PasswordConnectorRPCServer) Prompt(args any, resp *string) error {
	*resp = s.Impl.Prompt()
	return nil
}

func (s *PasswordConnectorRPCServer) Config(args ConfigArgs, resp *error) error {
	*resp = s.Impl.Config(args.Config)
	return nil
}

func (s *PasswordConnectorRPCServer) Login(args LoginArgs, resp *LoginResp) error {
	identity, validPassword, err := s.Impl.Login(context.TODO(), args.Scopes, args.Username, args.Password)
	resp.Identity = identity
	resp.ValidPassword = validPassword
	return err
}

type PasswordConnectorPlugin struct {
	Impl ConfigurablePasswordConnector
}

func (p *PasswordConnectorPlugin) Server(*plugin.MuxBroker) (any, error) {
	return &PasswordConnectorRPCServer{Impl: p.Impl}, nil
}

func (PasswordConnectorPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (any, error) {
	return &PasswordConnectorRPC{client: c}, nil
}
