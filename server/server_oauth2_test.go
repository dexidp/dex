package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"log/slog"
	"net/url"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/memory"
)

func TestSignerKeySet(t *testing.T) {
	logger := newLogger(t)
	s := memory.New(logger)
	if err := s.UpdateKeys(t.Context(), func(keys storage.Keys) (storage.Keys, error) {
		keys.SigningKey = &jose.JSONWebKey{
			Key:       testKey,
			KeyID:     "testkey",
			Algorithm: "RS256",
			Use:       "sig",
		}
		keys.SigningKeyPub = &jose.JSONWebKey{
			Key:       testKey.Public(),
			KeyID:     "testkey",
			Algorithm: "RS256",
			Use:       "sig",
		}
		return keys, nil
	}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		tokenGenerator func() (jwt string, err error)
		wantErr        bool
	}{
		{
			name: "valid token",
			tokenGenerator: func() (string, error) {
				signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: testKey}, nil)
				if err != nil {
					return "", err
				}

				jws, err := signer.Sign([]byte("payload"))
				if err != nil {
					return "", err
				}

				return jws.CompactSerialize()
			},
			wantErr: false,
		},
		{
			name: "token signed by different key",
			tokenGenerator: func() (string, error) {
				key, err := rsa.GenerateKey(rand.Reader, 2048)
				if err != nil {
					return "", err
				}

				signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, nil)
				if err != nil {
					return "", err
				}

				jws, err := signer.Sign([]byte("payload"))
				if err != nil {
					return "", err
				}

				return jws.CompactSerialize()
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			jwt, err := tc.tokenGenerator()
			if err != nil {
				t.Fatal(err)
			}

			// Create a mock signer for testing
			sig, err := signer.NewMockSigner(testKey)
			if err != nil {
				t.Fatal(err)
			}

			keySet := &signer.KeySet{
				Signer: sig,
			}

			_, err = keySet.VerifySignature(t.Context(), jwt)
			if (err != nil && !tc.wantErr) || (err == nil && tc.wantErr) {
				t.Fatalf("wantErr = %v, but got err = %v", tc.wantErr, err)
			}
		})
	}
}

func TestSignerKeySetWithES256LocalSigner(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.DiscardHandler)
	store := memory.New(logger)

	localConfig := signer.LocalConfig{
		KeysRotationPeriod: time.Hour.String(),
		Algorithm:          jose.ES256,
	}
	sig, err := localConfig.Open(ctx, store, time.Hour, time.Now, logger)
	require.NoError(t, err)

	sig.Start(ctx)

	jwt, err := sig.Sign(ctx, []byte("payload"))
	require.NoError(t, err)

	keySet := &signer.KeySet{Signer: sig}
	payload, err := keySet.VerifySignature(ctx, jwt)
	require.NoError(t, err)
	require.Equal(t, []byte("payload"), payload)
}

func TestNewIDTokenUsesStoredAlgorithmUntilNextRotation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.DiscardHandler)
	store := memory.New(logger)

	now := time.Now().UTC()
	err := store.UpdateKeys(ctx, func(keys storage.Keys) (storage.Keys, error) {
		keys.SigningKey = &jose.JSONWebKey{
			Key:       testKey,
			KeyID:     "legacy-rs256",
			Algorithm: string(jose.RS256),
			Use:       "sig",
		}
		keys.SigningKeyPub = &jose.JSONWebKey{
			Key:       testKey.Public(),
			KeyID:     "legacy-rs256",
			Algorithm: string(jose.RS256),
			Use:       "sig",
		}
		keys.NextRotation = now.Add(time.Hour)
		return keys, nil
	})
	require.NoError(t, err)

	localConfig := signer.LocalConfig{
		KeysRotationPeriod: time.Hour.String(),
		Algorithm:          jose.ES256,
	}
	sig, err := localConfig.Open(ctx, store, time.Hour, func() time.Time { return now }, logger)
	require.NoError(t, err)

	sig.Start(ctx)

	alg, err := sig.Algorithm(ctx)
	require.NoError(t, err)
	require.Equal(t, jose.RS256, alg)

	issuerURL, err := url.Parse("https://issuer.example.com")
	require.NoError(t, err)

	s := &Server{
		signer:           sig,
		issuerURL:        oauth2.IssuerURL{URL: *issuerURL},
		logger:           logger,
		now:              func() time.Time { return now },
		idTokensValidFor: time.Hour,
	}

	s.issuer = tokens.NewIssuer(store, s.signer, s.issuerURL.URL, s.idTokensValidFor, s.now, s.logger)

	accessToken := "test-access-token"
	code := "test-auth-code"
	idToken, _, err := s.issuer.SignIDToken(ctx, tokens.Authorization{
		Client:      storage.Client{ID: "test-client"},
		Claims:      storage.Claims{UserID: "1", Username: "jane"},
		Scopes:      []string{"openid"},
		Nonce:       "nonce",
		ConnectorID: "test",
	}, accessToken, code)
	require.NoError(t, err)

	keys, err := sig.ValidationKeys(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, keys)

	jws, err := jose.ParseSigned(idToken, []jose.SignatureAlgorithm{jose.RS256})
	require.NoError(t, err)
	require.Len(t, jws.Signatures, 1)
	require.Equal(t, string(jose.RS256), jws.Signatures[0].Protected.Algorithm)

	payload, err := jws.Verify(keys[0])
	require.NoError(t, err)

	var claims struct {
		AccessTokenHash string `json:"at_hash"`
		CodeHash        string `json:"c_hash"`
	}
	err = json.Unmarshal(payload, &claims)
	require.NoError(t, err)

	wantAtHash, err := tokens.AccessTokenHash(jose.RS256, accessToken)
	require.NoError(t, err)
	require.Equal(t, wantAtHash, claims.AccessTokenHash)

	wantCodeHash, err := tokens.AccessTokenHash(jose.RS256, code)
	require.NoError(t, err)
	require.Equal(t, wantCodeHash, claims.CodeHash)
}

func TestNewIDTokenContainsJTI(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
		issuerURL:        oauth2.IssuerURL{URL: *issuerURL},
		logger:           logger,
		now:              func() time.Time { return now },
		idTokensValidFor: time.Hour,
	}

	s.issuer = tokens.NewIssuer(store, s.signer, s.issuerURL.URL, s.idTokensValidFor, s.now, s.logger)

	keys, err := sig.ValidationKeys(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, keys)

	extractJTI := func(t *testing.T, idToken string) string {
		t.Helper()
		jws, err := jose.ParseSigned(idToken, []jose.SignatureAlgorithm{jose.RS256})
		require.NoError(t, err)
		payload, err := jws.Verify(keys[0])
		require.NoError(t, err)
		var claims struct {
			JTI string `json:"jti"`
		}
		err = json.Unmarshal(payload, &claims)
		require.NoError(t, err)
		return claims.JTI
	}

	mint := func(nonce string) string {
		t.Helper()
		token, _, err := s.issuer.SignIDToken(ctx, tokens.Authorization{
			Client:      storage.Client{ID: "client"},
			Claims:      storage.Claims{UserID: "1", Username: "alice"},
			Scopes:      []string{"openid"},
			Nonce:       nonce,
			ConnectorID: "mock",
		}, "", "")
		require.NoError(t, err)
		return token
	}
	token1 := mint("n1")
	token2 := mint("n2")

	jti1 := extractJTI(t, token1)
	jti2 := extractJTI(t, token2)

	assert.NotEmpty(t, jti1, "jti claim must be present and non-empty")
	assert.NotEmpty(t, jti2, "jti claim must be present and non-empty")
	assert.NotEqual(t, jti1, jti2, "each token must have a unique jti")
}
