package server

import "testing"

// The subject the app derives has to be byte for byte what dex puts in a token,
// or the refresh listing it feeds asks about a user who does not exist. These
// are subs taken from tokens dex issued.
func TestIDTokenSubject(t *testing.T) {
	tests := []struct {
		userID      string
		connectorID string
		want        string
	}{
		{"0-385-28089-0", "mock", "Cg0wLTM4NS0yODA4OS0wEgRtb2Nr"},
		{"08a8684b-db88-4b73-90a9-3cd1661f5466", "local", "CiQwOGE4Njg0Yi1kYjg4LTRiNzMtOTBhOS0zY2QxNjYxZjU0NjYSBWxvY2Fs"},
	}

	for _, tc := range tests {
		if got := idTokenSubject(tc.userID, tc.connectorID); got != tc.want {
			t.Errorf("idTokenSubject(%q, %q) = %q, want %q", tc.userID, tc.connectorID, got, tc.want)
		}
	}
}
