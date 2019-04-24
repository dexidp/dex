package redis

import (
	"os"
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/conformance"
	"github.com/sirupsen/logrus"
)

func TestStorage(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: &logrus.TextFormatter{DisableColors: true},
		Level:     logrus.DebugLevel,
	}
	config := Config{
		Addr: s.Addr(),
	}

	newStorage := func() storage.Storage {
		conn, err := config.Open(logger)
		if err != nil {
			t.Fatal(err)
		}
		return conn
	}
	conformance.RunTests(t, newStorage)
}
