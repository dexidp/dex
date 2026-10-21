package tokens

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTokenSetResponse(t *testing.T) {
	now := time.Unix(1000, 0)
	ts := TokenSet{
		AccessToken:  "at",
		IDToken:      "id",
		RefreshToken: "rt",
		Expiry:       now.Add(time.Hour),
	}

	resp := ts.Response(now)
	require.Equal(t, "at", resp.AccessToken)
	require.Equal(t, "id", resp.IDToken)
	require.Equal(t, "rt", resp.RefreshToken)
	require.Equal(t, "bearer", resp.TokenType)
	require.Equal(t, 3600, resp.ExpiresIn)
	// Grant-specific fields are left unset.
	require.Empty(t, resp.IssuedTokenType)
	require.Empty(t, resp.Scope)
}

func TestResponseWrite(t *testing.T) {
	resp := Response{AccessToken: "at", TokenType: "bearer", ExpiresIn: 3600, IDToken: "id"}

	rec := httptest.NewRecorder()
	require.NoError(t, resp.Write(rec))

	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	require.Equal(t, "no-cache", rec.Header().Get("Pragma"))

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, "at", got["access_token"])
	require.Equal(t, "bearer", got["token_type"])
	require.Equal(t, float64(3600), got["expires_in"])
	require.Equal(t, "id", got["id_token"])
	// Omitempty fields absent.
	_, hasRefresh := got["refresh_token"]
	require.False(t, hasRefresh)
}
