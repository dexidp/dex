package kubernetes

import (
	"testing"
	"time"

	"github.com/coreos/poke/storage"
)

func muster(t *testing.T) func(err error) {
	return func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestGCAuthRequests(t *testing.T) {
	cli := loadClient(t)
	must := muster(t)

	now := time.Now()
	cli.now = func() time.Time { return now }

	expiredID := storage.NewNonce()
	goodID := storage.NewNonce()

	must(cli.CreateAuthRequest(storage.AuthRequest{
		ID:     expiredID,
		Expiry: now.Add(-time.Second),
	}))

	must(cli.CreateAuthRequest(storage.AuthRequest{
		ID:     goodID,
		Expiry: now.Add(time.Second),
	}))

	handleErr := func(err error) { t.Error(err.Error()) }
	cli.gcAuthRequests(handleErr)

	if _, err := cli.GetAuthRequest(goodID); err != nil {
		t.Errorf("failed to get good auth ID: %v", err)
	}
	_, err := cli.GetAuthRequest(expiredID)
	switch {
	case err == nil:
		t.Errorf("gc did not remove expired auth request")
	case err == storage.ErrNotFound:
	default:
		t.Errorf("expected storage.ErrNotFound, got %v", err)
	}
}

func TestGCAuthCodes(t *testing.T) {
	cli := loadClient(t)
	must := muster(t)

	now := time.Now()
	cli.now = func() time.Time { return now }

	expiredID := storage.NewNonce()
	goodID := storage.NewNonce()

	must(cli.CreateAuthCode(storage.AuthCode{
		ID:     expiredID,
		Expiry: now.Add(-time.Second),
	}))

	must(cli.CreateAuthCode(storage.AuthCode{
		ID:     goodID,
		Expiry: now.Add(time.Second),
	}))

	handleErr := func(err error) { t.Error(err.Error()) }
	cli.gcAuthCodes(handleErr)

	if _, err := cli.GetAuthCode(goodID); err != nil {
		t.Errorf("failed to get good auth ID: %v", err)
	}
	_, err := cli.GetAuthCode(expiredID)
	switch {
	case err == nil:
		t.Errorf("gc did not remove expired auth request")
	case err == storage.ErrNotFound:
	default:
		t.Errorf("expected storage.ErrNotFound, got %v", err)
	}
}
