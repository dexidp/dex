package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// The redirect target comes from a form field, so it has to stay inside the
// app: an endpoint that bounces a browser to whatever a form says is an open
// redirect, whatever the app is for.
func TestDetailRedirectStaysLocal(t *testing.T) {
	tests := []struct {
		back string
		want string
	}{
		{"/admin/client/example-app", "/admin/client/example-app"},
		{"/admin?section=clients", "/admin?section=clients&notice=done"},
		{"", "/admin"},
		{"https://example.com/phish", "/admin"},
		{"//example.com/phish", "/admin"},
		{"http://127.0.0.1:5599/", "/admin"},
		{"javascript:alert(1)", "/admin"},
	}

	s := &Server{}
	for _, tc := range tests {
		r := httptest.NewRequest(http.MethodPost, "/admin/consent/revoke",
			strings.NewReader(url.Values{"back": {tc.back}}.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		s.detailRedirect(w, r, "done", "")

		got := w.Header().Get("Location")
		if !strings.HasPrefix(got, "/") || strings.HasPrefix(got, "//") {
			t.Fatalf("back=%q redirected off the app: %q", tc.back, got)
		}
		if tc.back == "" || !strings.HasPrefix(tc.back, "/") || strings.HasPrefix(tc.back, "//") {
			if !strings.HasPrefix(got, "/admin") {
				t.Errorf("back=%q: got %q, expected the admin page", tc.back, got)
			}
		}
	}
}
