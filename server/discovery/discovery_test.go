package discovery

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"log/slog"
	"strings"
	"testing"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/signer"
)

func testHandler(t *testing.T, sessionsEnabled bool) *Handler {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	sig, err := signer.NewMockSigner(key)
	require.NoError(t, err)

	return &Handler{
		Issuer:          "https://dex.example.com",
		AbsURL:          func(p ...string) string { return "https://dex.example.com" + strings.Join(p, "") },
		Signer:          sig,
		Logger:          slog.New(slog.DiscardHandler),
		ResponseTypes:   map[string]bool{"id_token": true, "code": true},
		GrantTypes:      []string{"authorization_code", "refresh_token"},
		PKCEMethods:     []string{"S256", "plain"},
		SessionsEnabled: sessionsEnabled,
	}
}

func TestConstruct(t *testing.T) {
	doc := testHandler(t, true).Construct(context.Background())

	require.Equal(t, "https://dex.example.com", doc.Issuer)
	require.Equal(t, "https://dex.example.com/auth", doc.Auth)
	require.Equal(t, "https://dex.example.com/token", doc.Token)
	require.Equal(t, "https://dex.example.com/keys", doc.Keys)
	require.Equal(t, "https://dex.example.com/token/introspect", doc.Introspect)
	// Response types are sorted.
	require.Equal(t, []string{"code", "id_token"}, doc.ResponseTypes)
	require.Equal(t, []string{"authorization_code", "refresh_token"}, doc.GrantTypes)
	require.Equal(t, []string{"S256", "plain"}, doc.CodeChallengeAlgs)
	require.Equal(t, []string{string(jose.RS256)}, doc.IDTokenAlgs)
	// end_session_endpoint is present only when sessions are enabled.
	require.Equal(t, "https://dex.example.com/logout", doc.EndSession)
}

func TestConstructNoSessions(t *testing.T) {
	doc := testHandler(t, false).Construct(context.Background())
	require.Empty(t, doc.EndSession)
}
