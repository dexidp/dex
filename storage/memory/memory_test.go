package memory

import (
	"testing"

	"github.com/coreos/dex/storage/conformance"
)

func TestStorage(t *testing.T) {
	s := New()
	conformance.RunTestSuite(t, s)
}
