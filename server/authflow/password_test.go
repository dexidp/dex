package authflow

import "testing"

func TestSanitizeBackLink(t *testing.T) {
	tests := map[string]string{
		"":                            "",
		"/auth?prompt=select_account": "/auth?prompt=select_account",
		"/dex/auth?client_id=x":       "/dex/auth?client_id=x",
		"https://evil.example":        "",
		"http://evil.example/auth":    "",
		"//evil.example":              "",
		"/\\evil.example":             "",
		"javascript:alert(1)":         "",
		"relative/path":               "", // not rooted
		"/auth#frag":                  "/auth#frag",
	}
	for in, want := range tests {
		if got := sanitizeBackLink(in); got != want {
			t.Errorf("sanitizeBackLink(%q) = %q, want %q", in, got, want)
		}
	}
}
