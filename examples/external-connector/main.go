package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/dexidp/dex/connector/external/sdk"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var inMemoryUsersStore = map[string]string{
	"jane-doe": "the-most-secure-password",
}

type connector struct {
	sdk.UnimplementedPasswordConnectorServer
}

func (c *connector) Prompt(_ context.Context, _ *sdk.PromptReq) (*sdk.PromptResp, error) {
	return &sdk.PromptResp{Prompt: "Test user"}, nil
}

func (c *connector) Login(_ context.Context, req *sdk.LoginReq) (*sdk.LoginResp, error) {
	resp := &sdk.LoginResp{
		Identity: &sdk.Identity{UserId: req.Username, Username: req.Username},
	}
	if inMemoryUsersStore[req.Username] == req.Password {
		resp.ValidPassword = true
	}
	return resp, nil
}

func (c *connector) Refresh(_ context.Context, req *sdk.RefreshReq) (*sdk.RefreshResp, error) {
	return &sdk.RefreshResp{Identity: req.Identity}, nil
}

func main() {
	var (
		listenAddress string
		tlsCert       string
		tlsKey        string
	)

	flag.StringVar(&listenAddress, "listen-address", "127.0.0.1:5571", "Address to listen on")

	flag.StringVar(&tlsCert, "tls-cert", "examples/external-gitlab/server.crt", "SSL certificate")
	flag.StringVar(&tlsKey, "tls-key", "examples/external-gitlab/server.key", "SSL certificate key")

	flag.Parse()

	grpcListener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatalln(err.Error())
	}

	cert, err := tls.LoadX509KeyPair(tlsCert, tlsKey)
	if err != nil {
		log.Fatalf("invalid config: error parsing gRPC certificate file: %v", err)
	}

	tlsConfig := tls.Config{
		Certificates:             []tls.Certificate{cert},
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
	}

	grpcSrv := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tlsConfig)))
	sdk.RegisterPasswordConnectorServer(grpcSrv, &connector{})

	fmt.Printf("Connector started on: %s", listenAddress)
	if err := grpcSrv.Serve(grpcListener); err != nil {
		log.Fatalln(err.Error())
	}
}
