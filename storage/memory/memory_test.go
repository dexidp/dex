package memory

import (
	"testing"

	"github.com/coreos/dex/storage/storagetest"
)

func TestStorage(t *testing.T) {
	s := New()
	storagetest.RunTestSuite(t, s)
}
