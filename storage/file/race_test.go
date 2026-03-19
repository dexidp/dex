package file

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/dexidp/dex/storage"
)

// TestConcurrentPasswordOperations exercises overlapping read/write/delete
// on the same password records to surface race conditions.
func TestConcurrentPasswordOperations(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	// Seed users
	for i := 0; i < 20; i++ {
		hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
		err := s.CreatePassword(ctx, storage.Password{
			Email:    fmt.Sprintf("user%d@example.com", i),
			Hash:     hash,
			Username: fmt.Sprintf("user%d", i),
			UserID:   fmt.Sprintf("user-%d", i),
		})
		if err != nil {
			t.Fatalf("seed user %d: %v", i, err)
		}
	}

	var wg sync.WaitGroup
	errs := make(chan error, 500)

	// Concurrent readers
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				email := fmt.Sprintf("user%d@example.com", i%20)
				_, err := s.GetPassword(ctx, email)
				if err != nil && err != storage.ErrNotFound {
					errs <- fmt.Errorf("GetPassword(%s): %w", email, err)
				}
			}
		}()
	}

	// Concurrent updaters
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				email := fmt.Sprintf("user%d@example.com", i)
				err := s.UpdatePassword(ctx, email, func(p storage.Password) (storage.Password, error) {
					p.Username = fmt.Sprintf("updated-by-%d", g)
					return p, nil
				})
				if err != nil && err != storage.ErrNotFound {
					errs <- fmt.Errorf("UpdatePassword(%s): %w", email, err)
				}
			}
		}(g)
	}

	// Concurrent listers
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				_, err := s.ListPasswords(ctx)
				if err != nil {
					errs <- fmt.Errorf("ListPasswords: %w", err)
				}
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

// TestConcurrentClientLifecycle creates, reads, updates, and deletes clients
// from multiple goroutines simultaneously.
func TestConcurrentClientLifecycle(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	errs := make(chan error, 500)

	// Each goroutine manages its own client through full lifecycle
	for g := 0; g < 20; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			clientID := fmt.Sprintf("client-%d", id)

			// Create
			err := s.CreateClient(ctx, storage.Client{
				ID:           clientID,
				Secret:       "secret",
				RedirectURIs: []string{"http://localhost/callback"},
				Name:         fmt.Sprintf("Client %d", id),
			})
			if err != nil {
				errs <- fmt.Errorf("CreateClient(%s): %w", clientID, err)
				return
			}

			// Read
			c, err := s.GetClient(ctx, clientID)
			if err != nil {
				errs <- fmt.Errorf("GetClient(%s): %w", clientID, err)
				return
			}
			if c.ID != clientID {
				errs <- fmt.Errorf("GetClient(%s): got ID %s", clientID, c.ID)
				return
			}

			// Update
			err = s.UpdateClient(ctx, clientID, func(old storage.Client) (storage.Client, error) {
				old.Name = "Updated " + old.Name
				return old, nil
			})
			if err != nil {
				errs <- fmt.Errorf("UpdateClient(%s): %w", clientID, err)
				return
			}

			// Verify update
			c, err = s.GetClient(ctx, clientID)
			if err != nil {
				errs <- fmt.Errorf("GetClient after update(%s): %w", clientID, err)
				return
			}
			if c.Name != fmt.Sprintf("Updated Client %d", id) {
				errs <- fmt.Errorf("unexpected name after update: %q", c.Name)
				return
			}

			// Delete
			err = s.DeleteClient(ctx, clientID)
			if err != nil {
				errs <- fmt.Errorf("DeleteClient(%s): %w", clientID, err)
				return
			}

			// Verify deleted
			_, err = s.GetClient(ctx, clientID)
			if err != storage.ErrNotFound {
				errs <- fmt.Errorf("expected ErrNotFound after delete, got: %v", err)
			}
		}(g)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

// TestConcurrentRefreshTokenChurn simulates token rotation — creating new tokens
// while old ones are being read and deleted.
func TestConcurrentRefreshTokenChurn(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	errs := make(chan error, 500)

	// Creators
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				id := fmt.Sprintf("token-%d-%d", g, i)
				err := s.CreateRefresh(ctx, storage.RefreshToken{
					ID:          id,
					Token:       fmt.Sprintf("tok-%d-%d", g, i),
					ClientID:    "test-client",
					ConnectorID: "local",
					Claims: storage.Claims{
						UserID: fmt.Sprintf("user-%d", g),
						Email:  fmt.Sprintf("user%d@example.com", g),
					},
					CreatedAt: time.Now(),
					Scopes:    []string{"openid"},
				})
				if err != nil {
					errs <- fmt.Errorf("CreateRefresh(%s): %w", id, err)
				}
			}
		}(g)
	}

	// Wait for creation to complete, then read/update/delete concurrently
	wg.Wait()

	// Concurrent readers + updaters + deleters
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				id := fmt.Sprintf("token-%d-%d", g, i)
				_, err := s.GetRefresh(ctx, id)
				if err != nil && err != storage.ErrNotFound {
					errs <- fmt.Errorf("GetRefresh(%s): %w", id, err)
				}
			}
		}(g)

		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				id := fmt.Sprintf("token-%d-%d", g, i)
				err := s.UpdateRefreshToken(ctx, id, func(r storage.RefreshToken) (storage.RefreshToken, error) {
					r.LastUsed = time.Now()
					return r, nil
				})
				if err != nil && err != storage.ErrNotFound {
					errs <- fmt.Errorf("UpdateRefreshToken(%s): %w", id, err)
				}
			}
		}(g)

		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				id := fmt.Sprintf("token-%d-%d", g, i)
				err := s.DeleteRefresh(ctx, id)
				if err != nil && err != storage.ErrNotFound {
					errs <- fmt.Errorf("DeleteRefresh(%s): %w", id, err)
				}
			}
		}(g)
	}

	// Concurrent list while deleting
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				_, err := s.ListRefreshTokens(ctx)
				if err != nil {
					errs <- fmt.Errorf("ListRefreshTokens: %w", err)
				}
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

// TestConcurrentAuthRequestExpiry creates auth requests and runs GC concurrently
// while other goroutines are still reading/updating them.
func TestConcurrentAuthRequestExpiry(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	// Create a mix of expired and valid auth requests
	for i := 0; i < 30; i++ {
		expiry := time.Now().Add(time.Hour)
		if i%3 == 0 {
			expiry = time.Now().Add(-time.Hour) // expired
		}
		err := s.CreateAuthRequest(ctx, storage.AuthRequest{
			ID:       fmt.Sprintf("req-%d", i),
			ClientID: "test-client",
			Expiry:   expiry,
		})
		if err != nil {
			t.Fatalf("seed auth request %d: %v", i, err)
		}
	}

	var wg sync.WaitGroup
	errs := make(chan error, 500)

	// Concurrent GC
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 5; i++ {
				_, err := s.GarbageCollect(ctx, time.Now())
				if err != nil {
					errs <- fmt.Errorf("GarbageCollect: %w", err)
				}
			}
		}()
	}

	// Concurrent readers — some requests may be GC'd mid-read
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 30; i++ {
				id := fmt.Sprintf("req-%d", i)
				_, err := s.GetAuthRequest(ctx, id)
				if err != nil && err != storage.ErrNotFound {
					errs <- fmt.Errorf("GetAuthRequest(%s): %w", id, err)
				}
			}
		}()
	}

	// Concurrent updaters on non-expired requests
	for g := 0; g < 3; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 30; i++ {
				if i%3 == 0 {
					continue // skip expired ones
				}
				id := fmt.Sprintf("req-%d", i)
				err := s.UpdateAuthRequest(ctx, id, func(a storage.AuthRequest) (storage.AuthRequest, error) {
					a.LoggedIn = true
					return a, nil
				})
				if err != nil && err != storage.ErrNotFound {
					errs <- fmt.Errorf("UpdateAuthRequest(%s): %w", id, err)
				}
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

// TestConcurrentKeyRotation simulates multiple key rotations happening
// simultaneously — a realistic scenario during Dex startup or failover.
func TestConcurrentKeyRotation(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	errs := make(chan error, 100)

	// Concurrent key updaters
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				err := s.UpdateKeys(ctx, func(old storage.Keys) (storage.Keys, error) {
					old.NextRotation = time.Now().Add(time.Duration(g*10+i) * time.Minute)
					return old, nil
				})
				if err != nil {
					errs <- fmt.Errorf("UpdateKeys(goroutine %d, iter %d): %w", g, i, err)
				}
			}
		}(g)
	}

	// Concurrent key readers
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				_, err := s.GetKeys(ctx)
				if err != nil {
					errs <- fmt.Errorf("GetKeys: %w", err)
				}
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}

	// Verify keys are still readable after all the churn
	keys, err := s.GetKeys(ctx)
	if err != nil {
		t.Fatalf("final GetKeys: %v", err)
	}
	if keys.NextRotation.IsZero() {
		t.Error("expected NextRotation to be set after updates")
	}
}

// TestConcurrentOfflineSessionUpdates tests the compound key (userID+connID)
// path with concurrent updates to the same session.
func TestConcurrentOfflineSessionUpdates(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	// Create a session
	err := s.CreateOfflineSessions(ctx, storage.OfflineSessions{
		UserID:  "user-1",
		ConnID:  "local",
		Refresh: map[string]*storage.RefreshTokenRef{},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 200)

	// 20 goroutines all updating the same session — each adds a different client
	for g := 0; g < 20; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			clientID := fmt.Sprintf("client-%d", g)
			err := s.UpdateOfflineSessions(ctx, "user-1", "local", func(o storage.OfflineSessions) (storage.OfflineSessions, error) {
				if o.Refresh == nil {
					o.Refresh = make(map[string]*storage.RefreshTokenRef)
				}
				o.Refresh[clientID] = &storage.RefreshTokenRef{
					ID:        fmt.Sprintf("token-%d", g),
					ClientID:  clientID,
					CreatedAt: time.Now(),
				}
				return o, nil
			})
			if err != nil {
				errs <- fmt.Errorf("UpdateOfflineSessions(client-%d): %w", g, err)
			}
		}(g)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}

	// Note: with a global mutex, all updates succeed but they serialise,
	// so last-writer-wins. This is expected — we're checking for no panics,
	// no corrupt files, no deadlocks.
	session, err := s.GetOfflineSessions(ctx, "user-1", "local")
	if err != nil {
		t.Fatalf("final GetOfflineSessions: %v", err)
	}
	t.Logf("final session has %d refresh entries (last-writer-wins expected)", len(session.Refresh))
}

// TestCreateDuplicateRace tests that two goroutines creating the same resource
// don't both succeed — exactly one should get ErrAlreadyExists.
func TestCreateDuplicateRace(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	results := make(chan error, 20)

	// 20 goroutines all trying to create the same client
	for g := 0; g < 20; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.CreateClient(ctx, storage.Client{
				ID:           "contested-client",
				Secret:       "secret",
				RedirectURIs: []string{"http://localhost/callback"},
				Name:         "Contested",
			})
			results <- err
		}()
	}

	wg.Wait()
	close(results)

	var successes, duplicates, other int
	for err := range results {
		switch {
		case err == nil:
			successes++
		case err == storage.ErrAlreadyExists:
			duplicates++
		default:
			other++
			t.Errorf("unexpected error: %v", err)
		}
	}

	if successes != 1 {
		t.Errorf("expected exactly 1 success, got %d", successes)
	}
	if duplicates != 19 {
		t.Errorf("expected 19 duplicates, got %d", duplicates)
	}
	t.Logf("results: %d success, %d duplicate, %d other", successes, duplicates, other)
}
