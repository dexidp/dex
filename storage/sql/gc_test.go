package sql

import (
	"testing"
	"time"

	"github.com/coreos/dex/storage"
)

func TestGC(t *testing.T) {
	// TODO(ericchiang): Add a GarbageCollect method to the storage interface so
	// we can write conformance tests instead of directly testing each implementation.
	s := &SQLite3{":memory:"}
	conn, err := s.open()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	clock := time.Now()
	now := func() time.Time { return clock }

	runGC := (gc{now, conn}).run

	a := storage.AuthRequest{
		ID:     storage.NewID(),
		Expiry: now().Add(time.Second),
	}

	if err := conn.CreateAuthRequest(a); err != nil {
		t.Fatal(err)
	}

	if err := runGC(); err != nil {
		t.Errorf("gc failed: %v", err)
	}

	if _, err := conn.GetAuthRequest(a.ID); err != nil {
		t.Errorf("failed to get auth request after gc: %v", err)
	}

	clock = clock.Add(time.Minute)

	if err := runGC(); err != nil {
		t.Errorf("gc failed: %v", err)
	}

	if _, err := conn.GetAuthRequest(a.ID); err == nil {
		t.Errorf("expected error after gc'ing auth request: %v", err)
	} else if err != storage.ErrNotFound {
		t.Errorf("expected error storage.NotFound got: %v", err)
	}
}
