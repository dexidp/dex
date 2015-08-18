package net

import (
	"testing"
)

func TestURLEqual(t *testing.T) {
	tests := []struct {
		a     string
		b     string
		equal bool
	}{
		{"https://accounts.example.com", "accounts.example.com", true},
		{"accounts.example.com", "accounts.example.com", true},
		{"accounts.example.com/FOO", "accounts.example.com/foo", true},
		{"https://accounts.example.com", "https://accounts.example.com", true},
		{"https://example.com/path1", "https://example.com/path2", false},
		{"https://example.com/path", "https://example.com/path", true},
		{"https://example.com/path?asdf=123", "example.com/path?foo=bar", true},
		{"foo.com", "bar.com", false},
		{"foo.com/foo", "foo.com/bar", false},
	}

	for i, tt := range tests {
		equal := URLEqual(tt.a, tt.b)
		if tt.equal != equal {
			t.Errorf("case %d: want=%t got=%t", i, tt.equal, equal)
		}
	}
}
