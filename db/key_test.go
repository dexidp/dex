package db

import (
	"testing"
)

func TestNewPrivateKeySetRepoInvalidKey(t *testing.T) {
	_, err := NewPrivateKeySetRepo(nil, false, []byte("sharks"))
	if err == nil {
		t.Fatalf("Expected non-nil error")
	}
}
