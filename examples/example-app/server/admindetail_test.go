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
		{"/admin/client/example-app", "/admin/client/example-app?notice=done"},
		{"/admin?section=clients", "/admin?section=clients&notice=done"},
		{"", "/admin?notice=done"},
		{"https://example.com/phish", "/admin?notice=done"},
		{"//example.com/phish", "/admin?notice=done"},
		// A browser turns the backslash into a slash, so these read as a host too.
		{`/\example.com/phish`, "/admin?notice=done"},
		{`/\/example.com/phish`, "/admin?notice=done"},
		{"http://127.0.0.1:5599/", "/admin?notice=done"},
		{"javascript:alert(1)", "/admin?notice=done"},
	}

	s := &Server{}
	for _, tc := range tests {
		r := httptest.NewRequest(http.MethodPost, "/admin/consent/revoke",
			strings.NewReader(url.Values{"back": {tc.back}}.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		s.detailRedirect(w, r, "done", "")

		if got := w.Header().Get("Location"); got != tc.want {
			t.Errorf("back=%q: redirected to %q, want %q", tc.back, got, tc.want)
		}
	}
}
