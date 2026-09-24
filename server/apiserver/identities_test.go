package apiserver

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/api/v2"
	"github.com/dexidp/dex/pkg/featureflags"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

// The GDPR purge cascades over several stores with no transaction across them,
// so the one step that fails for a predictable reason has to be settled before
// anything is destroyed. A password that comes from the config file cannot be
// deleted through the API: reaching it halfway through leaves the user signed
// out everywhere and their account still standing, with no way to finish.
func TestPurgeRefusesBeforeTouchingAnythingWhenThePasswordIsConfigBacked(t *testing.T) {
	t.Setenv("DEX_"+strings.ToUpper(featureflags.APISessionsIdentitiesCRUD.Name), "true")

	const userID, connID, sessionID = "u1", "local", "s1"

	tests := []struct {
		name        string
		email       string
		staticEmail string
		wantRefused bool
	}{
		{
			name:        "config-backed password",
			email:       "jane@example.com",
			staticEmail: "jane@example.com",
			wantRefused: true,
		},
		{
			// The same address in a different case is the same account to dex,
			// so it has to be the same answer here.
			name:        "config-backed password, different case",
			email:       "Jane@Example.com",
			staticEmail: "jane@example.com",
			wantRefused: true,
		},
		{
			name:        "password lives in storage",
			email:       "jane@example.com",
			staticEmail: "someone-else@example.com",
			wantRefused: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			logger := newLogger(t)

			// Wrapped the way serve.go wraps it. Nesting is the whole difficulty:
			// the wrappers embed Storage as an interface, so a check that only
			// works when the password wrapper is outermost passes here and does
			// nothing in a real dex.
			var s storage.Storage = memory.New(logger)
			s = storage.WithStaticClients(s, nil)
			s = storage.WithStaticPasswords(s, []storage.Password{{
				Email: tc.staticEmail, UserID: "static-user", Hash: []byte("hash"),
			}}, logger)
			s = storage.WithStaticConnectors(s, nil)

			require.NoError(t, s.CreateUserIdentity(ctx, storage.UserIdentity{
				UserID: userID, ConnectorID: connID,
				Claims:    storage.Claims{UserID: userID, Email: tc.email},
				CreatedAt: time.Now(), LastLogin: time.Now(),
			}))
			require.NoError(t, s.CreateAuthSession(ctx, storage.AuthSession{
				ID: sessionID, Secret: "session-secret",
				UserID: userID, ConnectorID: connID,
				CreatedAt: time.Now(), LastActivity: time.Now(),
			}))

			d := NewAPI(s, logger, "test", nil, nil, nil)
			_, err := d.DeleteUserIdentity(ctx, &api.DeleteUserIdentityReq{
				UserId: userID, ConnectorId: connID,
			})

			if !tc.wantRefused {
				require.NoError(t, err)
				_, err = s.GetAuthSession(ctx, sessionID)
				require.ErrorIs(t, err, storage.ErrNotFound, "a purge that succeeds takes the sessions with it")
				return
			}

			require.Error(t, err)
			message := err.Error()

			// Asserted before the wording, because this is the actual defect:
			// the cascade must never have started.
			_, err = s.GetAuthSession(ctx, sessionID)
			require.NoError(t, err, "the session was ended by a purge that could not finish")
			_, err = s.GetUserIdentity(ctx, userID, connID)
			require.NoError(t, err, "the identity was deleted by a purge that could not finish")

			// And the caller has to be told what to do about it, not handed the
			// storage layer's own words.
			require.Contains(t, message, "configuration file")
			require.Contains(t, message, "Nothing was deleted")
		})
	}
}
