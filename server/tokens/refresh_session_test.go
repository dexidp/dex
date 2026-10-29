package tokens

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
)

func TestRefreshReferenceSessionID(t *testing.T) {
	offline := storage.OfflineSessions{
		Refresh: map[string]*storage.RefreshTokenRef{
			"bound":   {ID: "r1", ClientID: "bound", SessionID: "sid"},
			"empty":   {ID: "r2", ClientID: "empty", SessionID: ""},
			"nil-ref": nil,
		},
	}

	require.Equal(t, "sid", RefreshReferenceSessionID(offline, "bound"))
	require.Equal(t, "", RefreshReferenceSessionID(offline, "empty"))
	require.Equal(t, "", RefreshReferenceSessionID(offline, "nil-ref"))
	require.Equal(t, "", RefreshReferenceSessionID(offline, "missing"))
}
