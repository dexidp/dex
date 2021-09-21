package ent

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/conformance"
)

func newSQLiteStorage() storage.Storage {
	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: &logrus.TextFormatter{DisableColors: true},
		Level:     logrus.DebugLevel,
	}

	cfg := SQLite3{File: ":memory:"}
	s, err := cfg.Open(logger)
	if err != nil {
		panic(err)
	}
	return s
}

func TestSQLite3(t *testing.T) {
	conformance.RunTests(t, newSQLiteStorage)
}
