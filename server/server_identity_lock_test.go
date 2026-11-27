package server

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/dexidp/dex/api/v2"
	"github.com/dexidp/dex/server/apiserver"
)

// TestIdentityLockThroughAdminAPI locks an account through the gRPC API and
// checks the password grant refuses it, then unlocks it and checks the same
// credentials work again. Before LockUserIdentity existed, the only way to set
// BlockedUntil — the field RecordLogin and the refresh grant already check —
// was to edit storage directly, so an operator had no API to lock an account
// (#5176).
func TestIdentityLockThroughAdminAPI(t *testing.T) {
	t.Setenv("DEX_API_SESSIONS_IDENTITIES_CRUD", "true")

	httpServer, s := newTestServer(t, func(c *Config) {
		c.PasswordConnector = "local"
		c.SessionConfig = testSessionConfig()
	})
	defer httpServer.Close()
	seedLocalPasswordDB(t, s)

	// Serve the admin API in front of the same storage the server uses.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	grpcServer := grpc.NewServer()
	api.RegisterDexServer(grpcServer, apiserver.NewAPI(s.storage, s.logger, "test", nil, nil, nil))
	go grpcServer.Serve(l)
	defer grpcServer.Stop()
	defer l.Close()

	conn, err := grpc.NewClient(l.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	apiClient := api.NewDexClient(conn)

	u, err := url.Parse(s.issuerURL.String())
	require.NoError(t, err)
	u.Path = path.Join(u.Path, "/token")

	passwordLogin := func() *httptest.ResponseRecorder {
		v := url.Values{}
		v.Add("scope", "openid email offline_access")
		v.Add("grant_type", "password")
		v.Add("username", "user@example.com")
		v.Add("password", "secret")

		req, _ := http.NewRequest("POST", u.String(), bytes.NewBufferString(v.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("test", "barfoo")

		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)
		return rr
	}

	// The first login creates the identity the lock will apply to: there is
	// nothing to lock before the user has signed in once.
	rr := passwordLogin()
	require.Equal(t, http.StatusOK, rr.Code, "first login should succeed: %s", rr.Body.String())

	ctx := t.Context()
	blockedUntil := time.Now().Add(time.Hour)
	_, err = apiClient.LockUserIdentity(ctx, &api.LockUserIdentityReq{
		UserId: "user-id", ConnectorId: "local", BlockedUntil: blockedUntil.Unix(),
	})
	require.NoError(t, err)

	rr = passwordLogin()
	require.Equal(t, http.StatusBadRequest, rr.Code, "locked account must not get tokens: %s", rr.Body.String())
	require.Contains(t, rr.Body.String(), "invalid_grant")

	_, err = apiClient.UnlockUserIdentity(ctx, &api.UnlockUserIdentityReq{UserId: "user-id", ConnectorId: "local"})
	require.NoError(t, err)

	rr = passwordLogin()
	require.Equal(t, http.StatusOK, rr.Code, "unlocked account should get tokens: %s", rr.Body.String())
}
