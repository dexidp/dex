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
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/storage/memory"
	dexweb "github.com/dexidp/dex/web"
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

	//nolint:dogsled // only the templates are needed here
	_, _, _, tmpls, err := templates.LoadWebConfig(templates.Config{
		WebFS:     dexweb.FS(),
		IssuerURL: h.IssuerURL.String(),
	})
	require.NoError(t, err)
	h.Templates = tmpls

	serve := func() *httptest.ResponseRecorder {
		t.Helper()
		rr := httptest.NewRecorder()
		h.serveDocument(rr, httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil))
		return rr
	}
	idTokenAlgs := func() []string {
		t.Helper()
		rr := serve()
		require.Equal(t, http.StatusOK, rr.Code)

		var doc Document
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &doc))
		return doc.IDTokenAlgs
	}

	// The failed lookup is an error rather than an RS256 document that clients
	// would cache.
	require.Equal(t, http.StatusInternalServerError, serve().Code)
	// The next request resolves the real algorithm, and that document is cached.
	require.Equal(t, []string{string(jose.ES256)}, idTokenAlgs())
	require.Equal(t, []string{string(jose.ES256)}, idTokenAlgs())
	require.Equal(t, 2, sig.calls)
}

// rotatingSigner reports a fixed next rotation time, as the local signer does
// from its stored keys. With rotateOnRead set, reading the keys moves the next
// rotation that far ahead, as a rotation between the two reads would.
type rotatingSigner struct {
	signer.Signer
	next         time.Time
	err          error
	rotateOnRead time.Duration
}

func (s *rotatingSigner) NextRotation(context.Context) (time.Time, error) {
	return s.next, s.err
}

func (s *rotatingSigner) ValidationKeys(ctx context.Context) ([]*jose.JSONWebKey, error) {
	s.next = s.next.Add(s.rotateOnRead)
	return s.Signer.ValidationKeys(ctx)
}

func TestKeysCacheControl(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		signer func(signer.Signer) signer.Signer
		want   string
	}{
		{
			name:   "rotation time unknown",
			signer: func(s signer.Signer) signer.Signer { return s },
			want:   "max-age=600, must-revalidate",
		},
		{
			name: "cached until the next rotation",
			signer: func(s signer.Signer) signer.Signer {
				return &rotatingSigner{Signer: s, next: now.Add(5*time.Minute + 30*time.Second)}
			},
			want: "max-age=330, must-revalidate",
		},
		{
			name: "next rotation over the cap",
			signer: func(s signer.Signer) signer.Signer {
				return &rotatingSigner{Signer: s, next: now.Add(5*time.Hour + 30*time.Second)}
			},
			want: "max-age=600, must-revalidate",
		},
		{
			name: "next rotation under the floor",
			signer: func(s signer.Signer) signer.Signer {
				return &rotatingSigner{Signer: s, next: now.Add(30 * time.Second)}
			},
			want: "max-age=120, must-revalidate",
		},
		{
			name: "next rotation in the past",
			signer: func(s signer.Signer) signer.Signer {
				return &rotatingSigner{Signer: s, next: now.Add(-time.Hour)}
			},
			want: "max-age=120, must-revalidate",
		},
		{
			// The rotation time is read before the keys, so a rotation in
			// between leaves the earlier, shorter cache time.
			name: "rotation between the two reads",
			signer: func(s signer.Signer) signer.Signer {
				return &rotatingSigner{Signer: s, next: now.Add(30 * time.Second), rotateOnRead: 6 * time.Hour}
			},
			want: "max-age=120, must-revalidate",
		},
		{
			name: "next rotation lookup fails",
			signer: func(s signer.Signer) signer.Signer {
				return &rotatingSigner{Signer: s, err: errors.New("storage unavailable")}
			},
			want: "max-age=600, must-revalidate",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := testHandler(t, false)
			h.Signer = tc.signer(h.Signer)
			h.Now = func() time.Time { return now }

			rr := httptest.NewRecorder()
			h.Keys(rr, httptest.NewRequest(http.MethodGet, "/keys", nil))

			require.Equal(t, http.StatusOK, rr.Code)
			require.Equal(t, tc.want, rr.Header().Get("Cache-Control"))
		})
	}
}

func TestKeysCacheControlLocalSigner(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	logger := slog.New(slog.DiscardHandler)

	config := signer.LocalConfig{KeysRotationPeriod: "6h"}
	sig, err := config.Open(ctx, memory.New(logger), time.Hour, func() time.Time { return now }, logger)
	require.NoError(t, err)
	// Start generates the first signing key, due for rotation in six hours.
	sig.Start(ctx)

	h := testHandler(t, false)
	h.Signer = sig
	// Serve the keys four minutes before the scheduled rotation.
	h.Now = func() time.Time { return now.Add(6*time.Hour - 4*time.Minute) }

	rr := httptest.NewRecorder()
	h.Keys(rr, httptest.NewRequest(http.MethodGet, "/keys", nil))

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "max-age=240, must-revalidate", rr.Header().Get("Cache-Control"))
}
