//go:build cgo

package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	dexsql "github.com/dexidp/dex/storage/sql"
)

// TestRefreshGrantStorageBackedPasswordDBSQLite repeats the password-store
// refresh on a real SQLite backend, the deployment shape the deadlock was
// reported on: the pre-fix nested read waits on the single-connection pool
// until the bounded request context expires (#5017).
func TestRefreshGrantStorageBackedPasswordDBSQLite(t *testing.T) {
	sqlite, err := (&dexsql.SQLite3{File: t.TempDir() + "/dex.db"}).Open(newLogger(t))
	require.NoError(t, err)

	httpServer, s := newTestServer(t, func(c *Config) {
		c.PasswordConnector = "local"
		c.Storage = sqlite
	})
	defer httpServer.Close()
	seedLocalPasswordDB(t, s)

	first := doPasswordGrant(t, s)

	rr := doRefreshGrant(t, s, first, 10*time.Second)
	require.Equal(t, http.StatusOK, rr.Code, "refresh grant failed: %s", rr.Body.String())
	var second tokenResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &second))
	require.NotEqual(t, first, second.RefreshToken, "refresh token was not rotated")
}
