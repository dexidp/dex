package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCleanHost(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"www.google.com", "www.google.com"},
		{"www.google.com foo", "www.google.com"},
		{"www.google.com/foo", "www.google.com"},
		{" first character is a space", ""},
	}
	for _, tt := range tests {
		got := cleanHost(tt.in)
		if tt.want != got {
			t.Errorf("cleanHost(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestCanonicalHost(t *testing.T) {
	gorilla := "http://www.gorillatoolkit.org"

	rr := httptest.NewRecorder()
	r := newRequest("GET", "http://www.example.com/")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	// Test a re-direct: should return a 302 Found.
	CanonicalHost(gorilla, http.StatusFound)(testHandler).ServeHTTP(rr, r)

	if rr.Code != http.StatusFound {
		t.Fatalf("bad status: got %v want %v", rr.Code, http.StatusFound)
	}

	if rr.Header().Get("Location") != gorilla+r.URL.Path {
		t.Fatalf("bad re-direct: got %q want %q", rr.Header().Get("Location"), gorilla+r.URL.Path)
	}

}

func TestBadDomain(t *testing.T) {
	rr := httptest.NewRecorder()
	r := newRequest("GET", "http://www.example.com/")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	// Test a bad domain - should return 200 OK.
	CanonicalHost("%", http.StatusFound)(testHandler).ServeHTTP(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("bad status: got %v want %v", rr.Code, http.StatusOK)
	}
}

func TestEmptyHost(t *testing.T) {
	rr := httptest.NewRecorder()
	r := newRequest("GET", "http://www.example.com/")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	// Test a domain that returns an empty url.Host from url.Parse.
	CanonicalHost("hello.com", http.StatusFound)(testHandler).ServeHTTP(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("bad status: got %v want %v", rr.Code, http.StatusOK)
	}
}
