package memory

import (
	"log/slog"
	"testing"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/conformance"
)

func TestStorage(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	newStorage := func() storage.Storage {
		return New(logger)
	}
	conformance.RunTests(t, newStorage)
}
