package server

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/storage"
)

func TestFilterConnectors(t *testing.T) {
	connectors := []storage.Connector{
		{ID: "github", Type: "github", Name: "GitHub"},
		{ID: "google", Type: "oidc", Name: "Google"},
		{ID: "ldap", Type: "ldap", Name: "LDAP"},
	}

	tests := []struct {
		name              string
		allowedConnectors []string
		wantIDs           []string
	}{
		{
			name:              "No filter - all connectors returned",
			allowedConnectors: nil,
			wantIDs:           []string{"github", "google", "ldap"},
		},
		{
			name:              "Empty filter - all connectors returned",
			allowedConnectors: []string{},
			wantIDs:           []string{"github", "google", "ldap"},
		},
		{
			name:              "Filter to one connector",
			allowedConnectors: []string{"github"},
			wantIDs:           []string{"github"},
		},
		{
			name:              "Filter to two connectors",
			allowedConnectors: []string{"github", "ldap"},
			wantIDs:           []string{"github", "ldap"},
		},
		{
			name:              "Filter with non-existent connector ID",
			allowedConnectors: []string{"nonexistent"},
			wantIDs:           []string{},
		},
		{
			name:              "Filter with mix of valid and invalid IDs",
			allowedConnectors: []string{"google", "nonexistent"},
			wantIDs:           []string{"google"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := filterConnectors(connectors, tc.allowedConnectors)
			gotIDs := make([]string, len(result))
			for i, c := range result {
				gotIDs[i] = c.ID
			}
			require.Equal(t, tc.wantIDs, gotIDs)
		})
	}
}

func TestIsConnectorAllowed(t *testing.T) {
	tests := []struct {
		name              string
		allowedConnectors []string
		connectorID       string
		want              bool
	}{
		{
			name:              "No restrictions - all allowed",
			allowedConnectors: nil,
			connectorID:       "any",
			want:              true,
		},
		{
			name:              "Empty list - all allowed",
			allowedConnectors: []string{},
			connectorID:       "any",
			want:              true,
		},
		{
			name:              "Connector in allowed list",
			allowedConnectors: []string{"github", "google"},
			connectorID:       "github",
			want:              true,
		},
		{
			name:              "Connector not in allowed list",
			allowedConnectors: []string{"github", "google"},
			connectorID:       "ldap",
			want:              false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := connectors.ConnectorAllowed(tc.allowedConnectors, tc.connectorID)
			require.Equal(t, tc.want, got)
		})
	}
}
