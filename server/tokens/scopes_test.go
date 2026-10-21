package tokens

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCrossClientScope(t *testing.T) {
	peer, ok := ParseCrossClientScope("audience:server:client_id:peer-1")
	require.True(t, ok)
	require.Equal(t, "peer-1", peer)

	// Empty peer is still a cross-client scope.
	peer, ok = ParseCrossClientScope("audience:server:client_id:")
	require.True(t, ok)
	require.Equal(t, "", peer)

	for _, scope := range []string{"openid", "email", "audience:other", ""} {
		peer, ok = ParseCrossClientScope(scope)
		require.False(t, ok, "scope %q", scope)
		require.Equal(t, "", peer)
	}
}
