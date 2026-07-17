package authflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScopesCoveredByConsent(t *testing.T) {
	tests := []struct {
		name      string
		approved  []string
		requested []string
		want      bool
	}{
		{
			name:      "All scopes covered",
			approved:  []string{"email", "profile"},
			requested: []string{"openid", "email", "profile"},
			want:      true,
		},
		{
			name:      "Missing scope",
			approved:  []string{"email"},
			requested: []string{"openid", "email", "groups"},
			want:      false,
		},
		{
			name:      "Only openid scope skipped",
			approved:  []string{},
			requested: []string{"openid"},
			want:      true,
		},
		{
			name:      "offline_access requires consent",
			approved:  []string{},
			requested: []string{"openid", "offline_access"},
			want:      false,
		},
		{
			name:      "offline_access covered by consent",
			approved:  []string{"offline_access"},
			requested: []string{"openid", "offline_access"},
			want:      true,
		},
		{
			name:      "Nil approved",
			approved:  nil,
			requested: []string{"email"},
			want:      false,
		},
		{
			name:      "Empty requested",
			approved:  []string{"email"},
			requested: []string{},
			want:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := scopesCoveredByConsent(tc.approved, tc.requested)
			require.Equal(t, tc.want, got)
		})
	}
}

// TestConsentIsolatedBetweenClients verifies that consent given for
// client-A does not satisfy scope check for client-B.
func TestConsentIsolatedBetweenClients(t *testing.T) {
	approvedForA := map[string][]string{"client-a": {"openid", "email"}}

	// client-b should not have consent.
	require.False(t, scopesCoveredByConsent(approvedForA["client-b"], []string{"openid", "email"}),
		"consent for client-a should not cover client-b")

	// client-a should have consent.
	require.True(t, scopesCoveredByConsent(approvedForA["client-a"], []string{"openid", "email"}),
		"consent for client-a should cover client-a's requested scopes")
}
