package memory

import (
	"testing"

	"github.com/coreos/poke/storage/storagetest"
)

func TestStorage(t *testing.T) {
	s := New()
	storagetest.RunTestSuite(t, s)
}
