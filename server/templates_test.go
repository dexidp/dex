package server

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
