package discovery

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/signer"
)

func testHandler(t *testing.T, sessionsEnabled bool) *Handler {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	sig, err := signer.NewMockSigner(key)
	require.NoError(t, err)

	u, err := url.Parse("https://dex.example.com")
	require.NoError(t, err)

	return &Handler{
		IssuerURL:       oauth2.IssuerURL{URL: *u},
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

// flakySigner fails its first Algorithm lookup, as a signer backed by storage or
// Vault does on a transient error, and reports ES256 afterwards.
type flakySigner struct {
	signer.Signer
	calls int
}

func (s *flakySigner) Algorithm(context.Context) (jose.SignatureAlgorithm, error) {
	s.calls++
	if s.calls == 1 {
		return "", errors.New("storage unavailable")
	}
	return jose.ES256, nil
}

func TestServeDocumentRetriesFailedSigningAlgorithm(t *testing.T) {
	h := testHandler(t, false)
	sig := &flakySigner{Signer: h.Signer}
	h.Signer = sig

	idTokenAlgs := func() []string {
		t.Helper()
		rr := httptest.NewRecorder()
		h.serveDocument(rr, httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil))
		require.Equal(t, http.StatusOK, rr.Code)

		var doc Document
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &doc))
		return doc.IDTokenAlgs
	}

	// The failed lookup falls back to RS256 for this response only.
	require.Equal(t, []string{string(jose.RS256)}, idTokenAlgs())
	// The next request resolves the real algorithm, and that document is cached.
	require.Equal(t, []string{string(jose.ES256)}, idTokenAlgs())
	require.Equal(t, []string{string(jose.ES256)}, idTokenAlgs())
	require.Equal(t, 2, sig.calls)
}
