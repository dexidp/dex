package server

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dexidp/dex/api/v2"
	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/connector/mock"
	"github.com/dexidp/dex/storage/memory"
)

func TestConnectorCacheInvalidation(t *testing.T) {
	t.Setenv("DEX_API_CONNECTORS_CRUD", "true")

	logger := newLogger(t)
	s := memory.New(logger)

	serv := &Server{
		storage:    s,
		logger:     logger,
		connectors: make(map[string]Connector),
	}

	apiServer := NewAPI(s, logger, "test", serv)
	ctx := context.Background()

	connID := "mock-conn"

	// 1. Create a connector via API
	config1 := mock.PasswordConfig{
		Username: "user",
		Password: "first-password",
	}
	config1Bytes, _ := json.Marshal(config1)

	_, err := apiServer.CreateConnector(ctx, &api.CreateConnectorReq{
		Connector: &api.Connector{
			Id:     connID,
			Type:   "mockPassword",
			Name:   "Mock",
			Config: config1Bytes,
		},
	})
	if err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	// 2. Load it into server cache
	c1, err := serv.getConnector(ctx, connID)
	if err != nil {
		t.Fatalf("failed to get connector: %v", err)
	}

	pc1 := c1.Connector.(connector.PasswordConnector)
	_, valid, err := pc1.Login(ctx, connector.Scopes{}, "user", "first-password")
	if err != nil || !valid {
		t.Fatalf("failed to login with first password: %v", err)
	}

	// 3. Delete it via API
	_, err = apiServer.DeleteConnector(ctx, &api.DeleteConnectorReq{Id: connID})
	if err != nil {
		t.Fatalf("failed to delete connector: %v", err)
	}

	// 4. Create it again with different password
	config2 := mock.PasswordConfig{
		Username: "user",
		Password: "second-password",
	}
	config2Bytes, _ := json.Marshal(config2)

	_, err = apiServer.CreateConnector(ctx, &api.CreateConnectorReq{
		Connector: &api.Connector{
			Id:     connID,
			Type:   "mockPassword",
			Name:   "Mock",
			Config: config2Bytes,
		},
	})
	if err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	// 5. Load it again
	c2, err := serv.getConnector(ctx, connID)
	if err != nil {
		t.Fatalf("failed to get connector second time: %v", err)
	}

	pc2 := c2.Connector.(connector.PasswordConnector)

	// If the fix works, it should now use the second password.
	_, valid2, err := pc2.Login(ctx, connector.Scopes{}, "user", "second-password")
	if err != nil || !valid2 {
		t.Errorf("failed to login with second password, cache might still be stale")
	}

	_, valid1, _ := pc2.Login(ctx, connector.Scopes{}, "user", "first-password")
	if valid1 {
		t.Errorf("unexpectedly logged in with first password, cache is definitely stale")
	}

	// 6. Update it via API with a third password
	config3 := mock.PasswordConfig{
		Username: "user",
		Password: "third-password",
	}
	config3Bytes, _ := json.Marshal(config3)

	_, err = apiServer.UpdateConnector(ctx, &api.UpdateConnectorReq{
		Id:        connID,
		NewConfig: config3Bytes,
	})
	if err != nil {
		t.Fatalf("failed to update connector: %v", err)
	}

	// 7. Load it again
	c3, err := serv.getConnector(ctx, connID)
	if err != nil {
		t.Fatalf("failed to get connector third time: %v", err)
	}

	pc3 := c3.Connector.(connector.PasswordConnector)

	_, valid3, err := pc3.Login(ctx, connector.Scopes{}, "user", "third-password")
	if err != nil || !valid3 {
		t.Errorf("failed to login with third password, UpdateConnector might be missing cache invalidation")
	}
}
