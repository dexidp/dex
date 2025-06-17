package plugin

import (
	"bufio"
	"bytes"
	"net/http"
	"net/rpc"

	"github.com/dexidp/dex/connector"
	"github.com/hashicorp/go-plugin"
)

type CallbackConnectorRPC struct{ client *rpc.Client }

func (g *CallbackConnectorRPC) Config(config map[string]any) error {
	var resp error
	args := ConfigArgs{Config: config}
	err := g.client.Call("Plugin.Config", args, &resp)
	if err != nil {
		panic(err)
	}

	return resp
}

type LoginURLArgs struct {
	Scopes      connector.Scopes
	CallbackURL string
	State       string
}

func (g *CallbackConnectorRPC) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
	var resp string
	args := LoginURLArgs{Scopes: s, CallbackURL: callbackURL, State: state}
	err := g.client.Call("Plugin.LoginURL", args, &resp)
	if err != nil {
		return "", err
	}

	return resp, nil
}

// HandleCallback needs the *http.Request struct to be serialized. This is hard
// to do generally with msgpack, and hard to make secure due to how easy it is
// for clients to forge requests.
//
// Instead, serialize the request into HTTP/1.1 wire format.
type HandleCallbackArgs struct {
	Scopes connector.Scopes
	Req    []byte
}

func (g *CallbackConnectorRPC) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	var b = &bytes.Buffer{}
	if err := r.Write(b); err != nil {
		return connector.Identity{}, err
	}

	var resp connector.Identity
	args := HandleCallbackArgs{Scopes: s, Req: b.Bytes()}
	err = g.client.Call("Plugin.HandleCallback", args, &resp)
	if err != nil {
		return connector.Identity{}, err
	}

	return resp, nil
}

type CallbackConnectorRPCServer struct {
	Impl ConfigurableCallbackConnector
}

func (s *CallbackConnectorRPCServer) Config(args ConfigArgs, resp *error) error {
	*resp = s.Impl.Config(args.Config)
	return nil
}

func (s *CallbackConnectorRPCServer) LoginURL(args LoginURLArgs, resp *string) error {
	url, err := s.Impl.LoginURL(args.Scopes, args.CallbackURL, args.State)
	if err != nil {
		return err
	}
	*resp = url
	return nil
}

func (s *CallbackConnectorRPCServer) HandleCallback(args HandleCallbackArgs, resp *connector.Identity) error {
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(args.Req)))
	if err != nil {
		return err
	}

	identity, err := s.Impl.HandleCallback(args.Scopes, req)
	if err != nil {
		return err
	}

	*resp = identity
	return nil
}

type ConfigurableCallbackConnector interface {
	connector.CallbackConnector
	ConfigurableConnector
}

type CallbackConnectorPlugin struct {
	Impl ConfigurableCallbackConnector
}

func (p *CallbackConnectorPlugin) Server(*plugin.MuxBroker) (any, error) {
	return &CallbackConnectorRPCServer{Impl: p.Impl}, nil
}

func (CallbackConnectorPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (any, error) {
	return &CallbackConnectorRPC{client: c}, nil
}
