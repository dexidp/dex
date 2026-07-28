package templates

import "testing"

func TestRelativeURL(t *testing.T) {
	tests := []struct {
		name       string
		serverPath string
		reqPath    string
		assetPath  string
		expected   string
	}{
		{
			name:       "server-root-req-one-level-asset-two-level",
			serverPath: "/",
			reqPath:    "/auth",
			assetPath:  "/theme/main.css",
			expected:   "theme/main.css",
		},
		{
			name:       "server-one-level-req-one-level-asset-two-level",
			serverPath: "/dex",
			reqPath:    "/dex/auth",
			assetPath:  "/theme/main.css",
			expected:   "theme/main.css",
		},
		{
			name:       "server-root-req-two-level-asset-three-level",
			serverPath: "/dex",
			reqPath:    "/dex/auth/connector",
			assetPath:  "assets/css/main.css",
			expected:   "../assets/css/main.css",
		},
		{
			name:       "server-root-page",
			serverPath: "/dex",
			reqPath:    "/dex",
			assetPath:  "static/main.css",
			expected:   "dex/static/main.css",
		},
		{
			name:       "server-root-page-nested",
			serverPath: "/dex/idp",
			reqPath:    "/dex/idp",
			assetPath:  "theme/styles.css",
			expected:   "dex/idp/theme/styles.css",
		},
		{
			name:       "server-root-no-subpath",
			serverPath: "/",
			reqPath:    "/",
			assetPath:  "static/main.css",
			expected:   "static/main.css",
		},
		{
			name:       "external-url",
			serverPath: "/dex",
			reqPath:    "/dex/auth/connector",
			assetPath:  "https://kubernetes.io/images/favicon.png",
			expected:   "https://kubernetes.io/images/favicon.png",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := relativeURL(test.serverPath, test.reqPath, test.assetPath)
			if actual != test.expected {
				t.Fatalf("Got '%s'. Expected '%s'", actual, test.expected)
			}
		})
	}
}

func TestSortConnectors(t *testing.T) {
	connectors := []ConnectorInfo{
		{ID: "keycloak", Name: "keycloak"},
		{ID: "okta", Name: "Okta"},
		{ID: "azure", Name: "azure ad"},
		{ID: "google", Name: "Google"},
	}

	sortConnectors(connectors)

	// Byte order would have put both lowercase names below both capitalised
	// ones, which is a property of ASCII rather than of anything the operator
	// meant by naming a connector.
	expected := []string{"azure ad", "Google", "keycloak", "Okta"}
	for i, name := range expected {
		if connectors[i].Name != name {
			t.Fatalf("position %d: got %q, expected %q", i, connectors[i].Name, name)
		}
	}
}

func TestSortConnectorsKeepsCaseVariantsStable(t *testing.T) {
	connectors := []ConnectorInfo{
		{ID: "second", Name: "SSO"},
		{ID: "first", Name: "sso"},
	}

	sortConnectors(connectors)

	if connectors[0].ID != "second" || connectors[1].ID != "first" {
		t.Fatalf("names differing only in case were reordered: %q, %q", connectors[0].ID, connectors[1].ID)
	}
}
