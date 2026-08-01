package server

import (
	"testing"
	"time"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func TestOpenConnectorsInstallsExpiryOverrides(t *testing.T) {
	conn := mockConnector("mock")
	conn.Expiry = &storage.ConnectorExpiry{IDTokens: "5m"}

	_, srv := newTestServerWith(t, []storage.Connector{conn}, nil)
	require.Equal(t, 5*time.Minute, srv.expiryPolicy.IDTokensValidFor("mock"))
}

func TestNewServerRejectsInvalidConnectorExpiry(t *testing.T) {
	ctx := t.Context()
	logger := newLogger(t)

	sig, err := signer.NewMockSigner(testKey)
	require.NoError(t, err)

	store := memory.New(logger)
	bad := mockConnector("bad")
	bad.Expiry = &storage.ConnectorExpiry{IDTokens: "48h"} // above the 24h default
	require.NoError(t, store.CreateConnector(ctx, mockConnector("good")))
	require.NoError(t, store.CreateConnector(ctx, bad))

	config := Config{
		Issuer:             "http://127.0.0.1:5556",
		Storage:            store,
		Web:                WebConfig{Dir: "../web"},
		Logger:             logger,
		PrometheusRegistry: prometheus.NewRegistry(),
		HealthChecker:      gosundheit.New(),
		Signer:             sig,
		RefreshTokenPolicy: tokens.NewRefreshStrategy(true, 0, 0, 0, nil),
	}
	_, err = newServer(ctx, config)
	require.ErrorContains(t, err, "invalid connector expiry")

	// With ContinueOnConnectorFailure the server starts on the valid subset
	// and the rejected override is not installed.
	config.ContinueOnConnectorFailure = true
	srv, err := newServer(ctx, config)
	require.NoError(t, err)
	require.Equal(t, 24*time.Hour, srv.expiryPolicy.IDTokensValidFor("bad"))
}
