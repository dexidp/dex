package repo

import (
	"os"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/kylelemons/godebug/pretty"

	"github.com/coreos/dex/db"
	"github.com/coreos/dex/session"
)

func newSessionRepo(t *testing.T) (session.SessionRepo, clockwork.FakeClock) {
	clock := clockwork.NewFakeClock()
	if os.Getenv("DEX_TEST_DSN") == "" {
		return session.NewSessionRepoWithClock(clock), clock
	}
	dbMap := connect(t)
	return db.NewSessionRepoWithClock(dbMap, clock), clock
}

func newSessionKeyRepo(t *testing.T) (session.SessionKeyRepo, clockwork.FakeClock) {
	clock := clockwork.NewFakeClock()
	if os.Getenv("DEX_TEST_DSN") == "" {
		return session.NewSessionKeyRepoWithClock(clock), clock
	}
	dbMap := connect(t)
	return db.NewSessionKeyRepoWithClock(dbMap, clock), clock
}

func TestSessionKeyRepoPopNoExist(t *testing.T) {
	r, _ := newSessionKeyRepo(t)

	_, err := r.Pop("123")
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestSessionKeyRepoPushPop(t *testing.T) {
	r, _ := newSessionKeyRepo(t)

	key := "123"
	sessionID := "456"

	r.Push(session.SessionKey{Key: key, SessionID: sessionID}, time.Second)

	got, err := r.Pop(key)
	if err != nil {
		t.Fatalf("Expected nil error: %v", err)
	}

	if got != sessionID {
		t.Fatalf("Incorrect sessionID: want=%s got=%s", sessionID, got)
	}
}

func TestSessionKeyRepoExpired(t *testing.T) {
	r, fc := newSessionKeyRepo(t)

	key := "123"
	sessionID := "456"

	r.Push(session.SessionKey{Key: key, SessionID: sessionID}, time.Second)

	fc.Advance(2 * time.Second)

	_, err := r.Pop(key)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestSessionRepoGetNoExist(t *testing.T) {
	r, _ := newSessionRepo(t)

	ses, err := r.Get("123")
	if ses != nil {
		t.Fatalf("Expected nil, got %#v", ses)
	}
	if err == nil {
		t.Fatalf("Expected non-nil error")
	}
}

func TestSessionRepoCreateGet(t *testing.T) {
	tests := []session.Session{
		session.Session{
			ID:          "123",
			ClientState: "blargh",
			ExpiresAt:   time.Unix(123, 0).UTC(),
		},
		session.Session{
			ID:          "456",
			ClientState: "argh",
			ExpiresAt:   time.Unix(456, 0).UTC(),
			Register:    true,
		},
		session.Session{
			ID:          "789",
			ClientState: "blargh",
			ExpiresAt:   time.Unix(789, 0).UTC(),
			Nonce:       "oncenay",
		},
	}

	for i, tt := range tests {
		r, _ := newSessionRepo(t)

		r.Create(tt)

		ses, _ := r.Get(tt.ID)
		if ses == nil {
			t.Fatalf("case %d: Expected non-nil Session", i)
		}

		if diff := pretty.Compare(tt, ses); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i, diff)
		}

	}
}

func TestSessionRepoCreateUpdate(t *testing.T) {
	tests := []struct {
		initial session.Session
		update  session.Session
	}{
		{
			initial: session.Session{
				ID:          "123",
				ClientState: "blargh",
				ExpiresAt:   time.Unix(123, 0).UTC(),
			},
			update: session.Session{
				ID:          "123",
				ClientState: "boom",
				ExpiresAt:   time.Unix(123, 0).UTC(),
				Register:    true,
			},
		},
	}

	for i, tt := range tests {
		r, _ := newSessionRepo(t)
		r.Create(tt.initial)

		ses, _ := r.Get(tt.initial.ID)
		if diff := pretty.Compare(tt.initial, ses); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i, diff)
		}

		r.Update(tt.update)
		ses, _ = r.Get(tt.initial.ID)
		if ses == nil {
			t.Fatalf("Expected non-nil Session")
		}
		if diff := pretty.Compare(tt.update, ses); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i, diff)
		}
	}
}

func TestSessionRepoUpdateNoExist(t *testing.T) {
	r, _ := newSessionRepo(t)

	err := r.Update(session.Session{ID: "123", ClientState: "boom"})
	if err == nil {
		t.Fatalf("Expected non-nil error")
	}
}
