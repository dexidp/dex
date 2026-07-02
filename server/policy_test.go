package server

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvaluateIDJAGPolicy(t *testing.T) {
	tests := []struct {
		name              string
		policies          []TokenExchangePolicy
		clientID          string
		audience          string
		scopes            []string
		wantDenied        bool
		wantDenialReason  PolicyDenialReason
		wantGrantedScopes []string
	}{
		{
			name:             "no policies: default-deny",
			policies:         nil,
			clientID:         "any-client",
			audience:         "https://resource.example.com",
			wantDenied:       true,
			wantDenialReason: PolicyDenialClientHasNoPolicy,
		},
		{
			name: "exact match allowed",
			policies: []TokenExchangePolicy{
				{ClientID: "client-a", AllowedAudiences: []string{"https://resource.example.com"}},
			},
			clientID: "client-a",
			audience: "https://resource.example.com",
		},
		{
			name: "audience not allowed",
			policies: []TokenExchangePolicy{
				{ClientID: "client-a", AllowedAudiences: []string{"https://other.example.com"}},
			},
			clientID:         "client-a",
			audience:         "https://resource.example.com",
			wantDenied:       true,
			wantDenialReason: PolicyDenialAudienceNotAllowed,
		},
		{
			name: "client not found: denied",
			policies: []TokenExchangePolicy{
				{ClientID: "client-a", AllowedAudiences: []string{"https://resource.example.com"}},
			},
			clientID:         "unknown-client",
			audience:         "https://resource.example.com",
			wantDenied:       true,
			wantDenialReason: PolicyDenialClientHasNoPolicy,
		},
		{
			name: "wildcard client matches",
			policies: []TokenExchangePolicy{
				{ClientID: "*", AllowedAudiences: []string{"https://resource.example.com"}},
			},
			clientID: "any-client",
			audience: "https://resource.example.com",
		},
		{
			name: "exact match takes priority over wildcard",
			policies: []TokenExchangePolicy{
				{ClientID: "*", AllowedAudiences: []string{"https://other.example.com"}},
				{ClientID: "client-a", AllowedAudiences: []string{"https://resource.example.com"}},
			},
			clientID: "client-a",
			audience: "https://resource.example.com",
		},
		{
			name: "scope filtered by policy",
			policies: []TokenExchangePolicy{
				{ClientID: "client-a", AllowedAudiences: []string{"https://resource.example.com"}, AllowedScopes: []string{"read"}},
			},
			clientID:          "client-a",
			audience:          "https://resource.example.com",
			scopes:            []string{"read", "admin"},
			wantGrantedScopes: []string{"read"},
		},
		{
			name: "allowed scope passes",
			policies: []TokenExchangePolicy{
				{ClientID: "client-a", AllowedAudiences: []string{"https://resource.example.com"}, AllowedScopes: []string{"read", "write"}},
			},
			clientID:          "client-a",
			audience:          "https://resource.example.com",
			scopes:            []string{"read"},
			wantGrantedScopes: []string{"read"},
		},
		{
			name: "no scope restriction: all scopes granted",
			policies: []TokenExchangePolicy{
				{ClientID: "client-a", AllowedAudiences: []string{"https://resource.example.com"}},
			},
			clientID:          "client-a",
			audience:          "https://resource.example.com",
			scopes:            []string{"anything"},
			wantGrantedScopes: []string{"anything"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := evaluateIDJAGPolicy(tc.policies, tc.clientID, tc.audience, tc.scopes)
			if tc.wantDenied {
				require.True(t, result.Denied)
				require.Equal(t, tc.wantDenialReason, result.DenialReason)
			} else {
				require.False(t, result.Denied)
				if tc.wantGrantedScopes != nil {
					require.Equal(t, tc.wantGrantedScopes, result.GrantedScopes)
				}
			}
		})
	}
}
