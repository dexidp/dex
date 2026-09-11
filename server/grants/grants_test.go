package grants

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

func TestValidateScopesAllowedPrefix(t *testing.T) {
	ctx := t.Context()
	h := &Handler{
		Logger: slog.New(slog.DiscardHandler),
	}
	client := storage.Client{ID: "client-1"}
	policy := ScopePolicy{
		Standard:      map[string]bool{tokens.ScopeOpenID: true},
		RequireOpenID: true,
		ErrorType:     oauth2.InvalidScope,
	}
	req := &Request{Scopes: []string{tokens.ScopeOpenID, "custom:read"}}

	// Without a matching allowed prefix, the non-standard scope is unrecognized.
	err := h.validateScopes(ctx, client, req, policy)
	require.NotNil(t, err)
	require.Equal(t, oauth2.InvalidScope, err.Type)

	// A matching allowed prefix lets it through.
	h.AllowedScopePrefixes = []string{"custom:"}
	err = h.validateScopes(ctx, client, req, policy)
	require.Nil(t, err)
}
