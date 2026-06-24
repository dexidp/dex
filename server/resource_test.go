package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func TestParseResourceParams(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    []string
		wantErr bool
	}{
		{name: "absent", input: nil, want: nil},
		{name: "empty values", input: []string{"", ""}, want: nil},
		{
			name:  "single absolute URI",
			input: []string{"https://mcp.example.com/"},
			want:  []string{"https://mcp.example.com/"},
		},
		{
			name:  "multiple values",
			input: []string{"https://a.example.com/", "https://b.example.com/api"},
			want:  []string{"https://a.example.com/", "https://b.example.com/api"},
		},
		{
			name:    "relative URI rejected",
			input:   []string{"/not-absolute"},
			wantErr: true,
		},
		{
			name:    "not a URI rejected",
			input:   []string{"this is not a uri"},
			wantErr: true,
		},
		{
			name:    "fragment rejected",
			input:   []string{"https://mcp.example.com/#section"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseResourceParams(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// newResourceTestServer builds a minimal Server with a working signer, so that
// newAccessToken / newIDToken can be exercised directly.
func newResourceTestServer(t *testing.T) (*Server, []*jose.JSONWebKey) {
	t.Helper()
	ctx := context.Background()

	logger := slog.New(slog.DiscardHandler)
	store := memory.New(logger)

	now := time.Now().UTC()
	err := store.UpdateKeys(ctx, func(keys storage.Keys) (storage.Keys, error) {
		keys.SigningKey = &jose.JSONWebKey{
			Key:       testKey,
			KeyID:     "test-rs256",
			Algorithm: string(jose.RS256),
			Use:       "sig",
		}
		keys.SigningKeyPub = &jose.JSONWebKey{
			Key:       testKey.Public(),
			KeyID:     "test-rs256",
			Algorithm: string(jose.RS256),
			Use:       "sig",
		}
		keys.NextRotation = now.Add(time.Hour)
		return keys, nil
	})
	require.NoError(t, err)

	localConfig := signer.LocalConfig{
		KeysRotationPeriod: time.Hour.String(),
		Algorithm:          jose.RS256,
	}
	sig, err := localConfig.Open(ctx, store, time.Hour, func() time.Time { return now }, logger)
	require.NoError(t, err)
	sig.Start(ctx)

	issuerURL, err := url.Parse("https://issuer.example.com")
	require.NoError(t, err)

	s := &Server{
		signer:           sig,
		issuerURL:        *issuerURL,
		logger:           logger,
		now:              func() time.Time { return now },
		idTokensValidFor: time.Hour,
	}

	keys, err := sig.ValidationKeys(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, keys)
	return s, keys
}

func decodeClaims(t *testing.T, token string, key *jose.JSONWebKey) (aud []string, azp string) {
	t.Helper()
	jws, err := jose.ParseSigned(token, []jose.SignatureAlgorithm{jose.RS256})
	require.NoError(t, err)
	payload, err := jws.Verify(key)
	require.NoError(t, err)
	// The "aud" claim is marshaled as a bare string when there is a single
	// audience (see audience.MarshalJSON) and as an array otherwise, so decode
	// it flexibly.
	var claims struct {
		Audience json.RawMessage `json:"aud"`
		AZP      string          `json:"azp"`
	}
	require.NoError(t, json.Unmarshal(payload, &claims))

	var single string
	if err := json.Unmarshal(claims.Audience, &single); err == nil {
		return []string{single}, claims.AZP
	}
	require.NoError(t, json.Unmarshal(claims.Audience, &aud))
	return aud, claims.AZP
}

// TestResourceAccessTokenAudience asserts the core RFC 8707 behavior: when a
// resource is requested, the access token's aud is bound to the resource (with
// azp = client) while the ID token's aud stays equal to the client.
func TestResourceAccessTokenAudience(t *testing.T) {
	s, keys := newResourceTestServer(t)
	ctx := context.Background()

	claims := storage.Claims{UserID: "1", Username: "jane"}
	resources := []string{"https://mcp.example.com/"}

	accessToken, _, err := s.newAccessToken(ctx, "test-client", claims, []string{"openid"}, "nonce", "conn", time.Time{}, resources)
	require.NoError(t, err)

	idToken, _, err := s.newIDToken(ctx, "test-client", claims, []string{"openid"}, "nonce", accessToken, "", "conn", time.Time{}, nil)
	require.NoError(t, err)

	atAud, atAZP := decodeClaims(t, accessToken, keys[0])
	assert.Equal(t, []string{"https://mcp.example.com/"}, atAud, "access token aud must be the resource")
	assert.Equal(t, "test-client", atAZP, "access token azp must be the client")

	idAud, _ := decodeClaims(t, idToken, keys[0])
	assert.Equal(t, []string{"test-client"}, idAud, "id token aud must remain the client (OIDC)")
}

func TestResourceAccessTokenMultipleResources(t *testing.T) {
	s, keys := newResourceTestServer(t)
	ctx := context.Background()

	resources := []string{"https://a.example.com/", "https://b.example.com/"}
	accessToken, _, err := s.newAccessToken(ctx, "test-client", storage.Claims{UserID: "1"}, []string{"openid"}, "", "conn", time.Time{}, resources)
	require.NoError(t, err)

	aud, _ := decodeClaims(t, accessToken, keys[0])
	assert.Equal(t, resources, aud)
}

// TestNoResourceAudienceUnchanged is the regression guard: with no resource,
// both the access token and the ID token keep aud == client_id.
func TestNoResourceAudienceUnchanged(t *testing.T) {
	s, keys := newResourceTestServer(t)
	ctx := context.Background()

	claims := storage.Claims{UserID: "1", Username: "jane"}

	accessToken, _, err := s.newAccessToken(ctx, "test-client", claims, []string{"openid"}, "nonce", "conn", time.Time{}, nil)
	require.NoError(t, err)

	idToken, _, err := s.newIDToken(ctx, "test-client", claims, []string{"openid"}, "nonce", accessToken, "", "conn", time.Time{}, nil)
	require.NoError(t, err)

	atAud, _ := decodeClaims(t, accessToken, keys[0])
	idAud, _ := decodeClaims(t, idToken, keys[0])
	assert.Equal(t, []string{"test-client"}, atAud)
	assert.Equal(t, []string{"test-client"}, idAud)
}
