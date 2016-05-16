package session

import (
	"testing"
	"time"

	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/oidc"

	"github.com/kylelemons/godebug/pretty"
)

func TestSessionClaims(t *testing.T) {
	issuerURL := "http://server.example.com"
	now := time.Now()

	tests := []struct {
		ses  Session
		want jose.Claims
	}{
		{
			ses: Session{
				CreatedAt: now,
				ExpiresAt: now.Add(time.Hour),
				ClientID:  "XXX",
				Identity: oidc.Identity{
					ID:    "YYY",
					Name:  "elroy",
					Email: "elroy@example.com",
				},
				UserID: "elroy-id",
			},
			want: jose.Claims{
				"iss": issuerURL,
				"sub": "elroy-id",
				"aud": "XXX",
				"iat": now.Unix(),
				"exp": now.Add(time.Hour).Unix(),
			},
		},

		// Ignore Identity custom ExpiresAt, use from session
		{
			ses: Session{
				CreatedAt: now,
				ExpiresAt: now.Add(time.Hour),
				ClientID:  "XXX",
				Identity: oidc.Identity{
					ID:        "YYY",
					Name:      "elroy",
					Email:     "elroy@example.com",
					ExpiresAt: now.Add(time.Minute),
				},
				UserID: "elroy-id",
			},
			want: jose.Claims{
				"iss": issuerURL,
				"sub": "elroy-id",
				"aud": "XXX",
				"iat": now.Unix(),
				"exp": now.Add(time.Hour).Unix(),
			},
		},
		// Nonce gets propagated.
		{
			ses: Session{
				CreatedAt: now,
				ExpiresAt: now.Add(time.Hour),
				ClientID:  "XXX",
				Identity: oidc.Identity{
					ID:    "YYY",
					Name:  "elroy",
					Email: "elroy@example.com",
				},
				UserID: "elroy-id",
				Nonce:  "oncenay",
			},
			want: jose.Claims{
				"iss":   issuerURL,
				"sub":   "elroy-id",
				"aud":   "XXX",
				"iat":   now.Unix(),
				"exp":   now.Add(time.Hour).Unix(),
				"nonce": "oncenay",
			},
		},
	}

	for i, tt := range tests {
		got := tt.ses.Claims(issuerURL)
		if diff := pretty.Compare(tt.want, got); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i, diff)
		}
	}

}
