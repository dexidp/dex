package server

import (
	"testing"
)

func TestEvaluateIDJAGPolicy(t *testing.T) {
	tests := []struct {
		name     string
		policies []TokenExchangePolicy
		clientID string
		audience string
		scopes   []string
		wantErr  bool
	}{
		{
			name:     "no policies: allow all",
			policies: nil,
			clientID: "any-client",
			audience: "https://resource.example.com",
			wantErr:  false,
		},
		{
			name: "exact match allowed",
			policies: []TokenExchangePolicy{
				{ClientID: "client-a", AllowedAudiences: []string{"https://resource.example.com"}},
			},
			clientID: "client-a",
			audience: "https://resource.example.com",
			wantErr:  false,
		},
		{
			name: "audience not allowed",
			policies: []TokenExchangePolicy{
				{ClientID: "client-a", AllowedAudiences: []string{"https://other.example.com"}},
			},
			clientID: "client-a",
			audience: "https://resource.example.com",
			wantErr:  true,
		},
		{
			name: "client not found: denied",
			policies: []TokenExchangePolicy{
				{ClientID: "client-a", AllowedAudiences: []string{"https://resource.example.com"}},
			},
			clientID: "unknown-client",
			audience: "https://resource.example.com",
			wantErr:  true,
		},
		{
			name: "wildcard client matches",
			policies: []TokenExchangePolicy{
				{ClientID: "*", AllowedAudiences: []string{"https://resource.example.com"}},
			},
			clientID: "any-client",
			audience: "https://resource.example.com",
			wantErr:  false,
		},
		{
			name: "exact match takes priority over wildcard",
			policies: []TokenExchangePolicy{
				{ClientID: "*", AllowedAudiences: []string{"https://other.example.com"}},
				{ClientID: "client-a", AllowedAudiences: []string{"https://resource.example.com"}},
			},
			clientID: "client-a",
			audience: "https://resource.example.com",
			wantErr:  false,
		},
		{
			name: "scope denied by policy",
			policies: []TokenExchangePolicy{
				{ClientID: "client-a", AllowedAudiences: []string{"https://resource.example.com"}, AllowedScopes: []string{"read"}},
			},
			clientID: "client-a",
			audience: "https://resource.example.com",
			scopes:   []string{"admin"},
			wantErr:  true,
		},
		{
			name: "allowed scope passes",
			policies: []TokenExchangePolicy{
				{ClientID: "client-a", AllowedAudiences: []string{"https://resource.example.com"}, AllowedScopes: []string{"read", "write"}},
			},
			clientID: "client-a",
			audience: "https://resource.example.com",
			scopes:   []string{"read"},
			wantErr:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := evaluateIDJAGPolicy(tc.policies, tc.clientID, tc.audience, tc.scopes)
			if tc.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
