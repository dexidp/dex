package authreq

import (
	"crypto/rand"
	"crypto/rsa"
	"log/slog"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

var testKey = func() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return key
}()

const testIssuer = "https://server.example.com/dex"

func testLogger(t *testing.T) *slog.Logger {
	return slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func toResponseTypeSet(types []string) map[string]bool {
	m := make(map[string]bool, len(types))
	for _, t := range types {
		m[t] = true
	}
	return m
}

// newTestParser builds a Parser over a memory storage seeded with the given
// clients, for the parser unit tests.
func newTestParser(t *testing.T, clients []storage.Client, pkce PKCEConfig, responseTypes map[string]bool, sig signer.Signer) *Parser {
	t.Helper()
	logger := testLogger(t)
	store := storage.WithStaticClients(memory.New(logger), clients)
	for _, id := range []string{"mock", "mock2"} {
		require.NoError(t, store.CreateConnector(t.Context(), storage.Connector{
			ID: id, Type: "mockCallback", Name: "Mock", ResourceVersion: "1",
		}))
	}
	if len(pkce.CodeChallengeMethodsSupported) == 0 {
		pkce.CodeChallengeMethodsSupported = []string{"S256", "plain"}
	}
	if responseTypes == nil {
		responseTypes = map[string]bool{"code": true, "id_token": true, "token": true}
	}
	if sig == nil {
		var err error
		sig, err = signer.NewMockSigner(testKey)
		require.NoError(t, err)
	}
	issuerURL, err := url.Parse(testIssuer)
	require.NoError(t, err)
	return New(store, logger, sig, *issuerURL, pkce, responseTypes)
}
