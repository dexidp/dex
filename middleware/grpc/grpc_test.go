package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"testing"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/middleware"
	"github.com/dexidp/dex/middleware/grpc/api"

	"google.golang.org/grpc"

	"github.com/sirupsen/logrus"
)

func TestGPRCMiddlewareAPIVersion(t *testing.T) {
	runTestsWithServer(t, &testServer{APIVersion: 0},
		func(ctx context.Context, mware middleware.Middleware) {
			input := connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
			}

			_, err := mware.Process(ctx, input)
			if err == nil || !IsIncompatibleAPIVersion(err) {
				t.Fatalf("expected incompatible API version error, got %v", err)
			}
		})

	runTestsWithServer(t, &testServer{APIVersion: 1},
		func(ctx context.Context, mware middleware.Middleware) {
			input := connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
			}

			_, err := mware.Process(ctx, input)
			if err != nil {
				t.Fatalf("expected no error")
			}
		})

	runTestsWithServer(t, &testServer{APIVersion: 2},
		func(ctx context.Context, mware middleware.Middleware) {
			input := connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
			}

			_, err := mware.Process(ctx, input)
			if err != nil {
				t.Fatalf("expected no error")
			}
		})
}

func TestGRPCMiddlewareCalls(t *testing.T) {
	mwareServer := &testServer{APIVersion: 1}
	runTestsWithServer(t, mwareServer,
		func(ctx context.Context, mware middleware.Middleware) {
			input := connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
			}

			got, err := mware.Process(ctx, input)
			if err != nil {
				t.Fatalf("middleware failed: %v", err)
			}

			if mwareServer.GetInfoCalls != 1 || mwareServer.ProcessCalls != 1 {
				t.Fatalf("expected one GetInfo() and one Process() call, got %d, %d",
					mwareServer.GetInfoCalls, mwareServer.ProcessCalls)
			}

			if got.CustomClaims == nil {
				t.Fatalf("expected custom claims")
			}

			if b, ok := got.CustomClaims["TestServer"].(bool); !ok || !b {
				t.Fatalf("custom claims incorrect - expected TestServer = true")
			}

			if i, ok := got.CustomClaims["GetInfoCalls"].(float64); !ok || i != 1 {
				t.Fatalf("custom claims incorrect - expected GetInfoCalls = 1, got %f", i)
			}

			if i, ok := got.CustomClaims["ProcessCalls"].(float64); !ok || i != 1 {
				t.Fatalf("custom claims incorrect - expected ProcessCalls = 1, got %f", i)
			}

			got, err = mware.Process(ctx, input)
			if err != nil {
				t.Fatalf("middleware failed: %v", err)
			}

			if mwareServer.GetInfoCalls != 1 || mwareServer.ProcessCalls != 2 {
				t.Fatalf("expected one GetInfo() and two Process() calls, got %d, %d",
					mwareServer.GetInfoCalls, mwareServer.ProcessCalls)
			}

			if got.CustomClaims == nil {
				t.Fatalf("expected custom claims")
			}

			if b, ok := got.CustomClaims["TestServer"].(bool); !ok || !b {
				t.Fatalf("custom claims incorrect - expected TestServer = true")
			}

			if i, ok := got.CustomClaims["GetInfoCalls"].(float64); !ok || i != 1 {
				t.Fatalf("custom claims incorrect - expected GetInfoCalls = 1, got %f", i)
			}

			if i, ok := got.CustomClaims["ProcessCalls"].(float64); !ok || i != 2 {
				t.Fatalf("custom claims incorrect - expected ProcessCalls = 2, got %f", i)
			}
		})
}

func TestGRPCMiddlewarePreservesConnectorData(t *testing.T) {
	mwareServer := &testServer{APIVersion: 1}
	runTestsWithServer(t, mwareServer,
		func(ctx context.Context, mware middleware.Middleware) {
			connectorData := []byte("This should be preserved")
			input := connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				ConnectorData: connectorData,
			}

			got, err := mware.Process(ctx, input)
			if err != nil {
				t.Fatalf("middleware failed: %v", err)
			}

			if string(got.ConnectorData) != string(connectorData) {
				t.Fatalf("connector data was not preserved")
			}
		})
}

type testServer struct {
	api.UnimplementedMiddlewareServer

	APIVersion   int32
	GetInfoCalls int32
	ProcessCalls int32
}

func (t *testServer) GetInfo(ctx context.Context, req *api.ServerInfoReq) (*api.ServerInfo, error) {
	t.GetInfoCalls++
	return &api.ServerInfo{
		Name:    "Test Server",
		Version: "1.0.0",
		Api:     t.APIVersion,
	}, nil
}

func (t *testServer) Process(ctx context.Context, req *api.ProcessReq) (*api.Identity, error) {
	t.ProcessCalls++

	ident := req.GetIdentity()
	if ident == nil {
		return nil, fmt.Errorf("no identity found")
	}

	var claims map[string]interface{}
	err := json.Unmarshal(ident.CustomClaims, &claims)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal custom claims: %v", err)
	}

	if claims == nil {
		claims = map[string]interface{}{}
	}

	claims["TestServer"] = true
	claims["GetInfoCalls"] = t.GetInfoCalls
	claims["ProcessCalls"] = t.ProcessCalls

	ident.CustomClaims, err = json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal custom claims: %v", err)
	}

	return ident, nil
}

func runTestsWithServer(t *testing.T, mwareServer api.MiddlewareServer,
	fn func(ctx context.Context, middleware middleware.Middleware)) {
	listener, err := net.Listen("tcp", ":50511")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	server := grpc.NewServer()
	defer server.GracefulStop()

	api.RegisterMiddlewareServer(server, mwareServer)

	go server.Serve(listener)

	config := &Config{
		Endpoint: "localhost:50511",
		Insecure: true,
	}

	l := &logrus.Logger{Out: ioutil.Discard, Formatter: &logrus.TextFormatter{}}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mware, err := config.Open(l)
	if err != nil {
		t.Fatalf("open middleware: %v", err)
	}

	fn(ctx, mware)
}
