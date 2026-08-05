package apiserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/api/v2"
	"github.com/dexidp/dex/pkg/featureflags"
	"github.com/dexidp/dex/server/backchannel"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

// TestTerminateSessionNotifiesRelyingParties: an operator ending a session leaves the
// relying parties as the only thing keeping the user signed in, so they are told.
func TestTerminateSessionNotifiesRelyingParties(t *testing.T) {
	t.Setenv("DEX_"+strings.ToUpper(featureflags.APISessionsIdentitiesCRUD.Name), "true")

	const userID, connectorID, clientID = "u1", "mock", "web"

	tests := []struct {
		name string
		call func(context.Context, api.DexServer) error
	}{
		{
			name: "delete one session",
			call: func(ctx context.Context, d api.DexServer) error {
				_, err := d.DeleteAuthSession(ctx, &api.DeleteAuthSessionReq{
					UserId: userID, ConnectorId: connectorID,
				})
				return err
			},
		},
		{
			name: "terminate every session of a user",
			call: func(ctx context.Context, d api.DexServer) error {
				_, err := d.TerminateSessionsByUser(ctx, &api.TerminateSessionsByUserReq{
					UserId: userID,
				})
				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			logger := newLogger(t)
			s := memory.New(logger)

			var mu sync.Mutex
			var tokens []string
			rp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.NoError(t, r.ParseForm())
				mu.Lock()
				tokens = append(tokens, r.PostForm.Get("logout_token"))
				mu.Unlock()
			}))
			defer rp.Close()

			require.NoError(t, s.CreateClient(ctx, storage.Client{
				ID: clientID, Secret: "secret",
				RedirectURIs:         []string{"https://example.com/cb"},
				BackchannelLogoutURI: rp.URL,
			}))
			require.NoError(t, s.CreateAuthSession(ctx, storage.AuthSession{
				UserID: userID, ConnectorID: connectorID, Nonce: "nonce",
				CreatedAt: time.Now(), LastActivity: time.Now(),
				ClientStates: map[string]*storage.ClientAuthState{clientID: {Active: true}},
			}))

			sign, err := (&signer.MockConfig{}).Open(ctx)
			require.NoError(t, err)

			d := NewAPI(s, logger, "test", nil, nil, &backchannel.Notifier{
				Storage: s, Signer: sign, IssuerURL: issuerURL(t), Logger: logger,
			}, nil)

			require.NoError(t, tc.call(ctx, d))

			// Delivery is fire-and-forget, so the call returns before the POST lands.
			require.Eventually(t, func() bool {
				mu.Lock()
				defer mu.Unlock()
				return len(tokens) == 1
			}, 2*time.Second, 10*time.Millisecond)

			_, err = s.GetAuthSession(ctx, userID, connectorID)
			require.ErrorIs(t, err, storage.ErrNotFound)
		})
	}
}

func issuerURL(t *testing.T) oauth2.IssuerURL {
	t.Helper()
	u, err := url.Parse("https://dex.example.com")
	require.NoError(t, err)
	return oauth2.IssuerURL{URL: *u}
}
