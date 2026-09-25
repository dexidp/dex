package health

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// A check that never returns must be reported as a failure.
func TestTimeoutCheckReportsHungCheck(t *testing.T) {
	release := make(chan struct{})
	defer close(release)

	check := NewTimeoutCheckFunc(
		func(ctx context.Context) (interface{}, error) {
			<-release // ignores ctx entirely
			return nil, nil
		},
		50*time.Millisecond)

	done := make(chan error, 1)
	go func() {
		_, err := check(context.Background())
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected a hung check to be reported as a failure")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the wrapper itself hung; a hung check must not block the caller")
	}
}

// Executions are single-flight, so a blocked check cannot accumulate a
// goroutine per execution for the length of an outage.
func TestTimeoutCheckIsSingleFlight(t *testing.T) {
	var starts atomic.Int32
	release := make(chan struct{})
	defer close(release)

	// Closed by the first execution, so the count below is read only once the
	// goroutine has provably run - it is not assumed to have been scheduled
	// within the timeout.
	entered := make(chan struct{})

	check := NewTimeoutCheckFunc(
		func(ctx context.Context) (interface{}, error) {
			if starts.Add(1) == 1 {
				close(entered)
			}
			<-release
			return nil, nil
		},
		20*time.Millisecond)

	for i := 0; i < 5; i++ {
		if _, err := check(context.Background()); !errors.Is(err, errTimeout) &&
			!errors.Is(err, errInFlight) {
			t.Fatalf("execution %d: expected a failure while the check is blocked, got %v", i, err)
		}
	}

	<-entered

	if got := starts.Load(); got != 1 {
		t.Fatalf("expected exactly 1 execution of the wrapped check, got %d", got)
	}
}

// The check recovers once the outstanding execution returns.
func TestTimeoutCheckRecovers(t *testing.T) {
	release := make(chan struct{})
	var starts atomic.Int32

	check := NewTimeoutCheckFunc(
		func(ctx context.Context) (interface{}, error) {
			if starts.Add(1) == 1 {
				<-release
			}
			return nil, nil
		},
		20*time.Millisecond)

	if _, err := check(context.Background()); err == nil {
		t.Fatal("expected the blocked execution to fail")
	}

	close(release)

	// The blocked execution returns, its result is discarded, and a fresh
	// execution runs. Poll: the goroutine needs a moment to deliver.
	deadline := time.Now().Add(5 * time.Second)
	for {
		_, err := check(context.Background())
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("check did not recover after the blocked execution returned: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// A check that observes cancellation is canceled on overrun, freeing the slot
// without waiting for the check's goodwill.
func TestTimeoutCheckCancelsContext(t *testing.T) {
	canceled := make(chan struct{}, 1)

	check := NewTimeoutCheckFunc(
		func(ctx context.Context) (interface{}, error) {
			<-ctx.Done()
			canceled <- struct{}{}
			return nil, ctx.Err()
		},
		20*time.Millisecond)

	if _, err := check(context.Background()); err == nil {
		t.Fatal("expected a failure on overrun")
	}

	select {
	case <-canceled:
	case <-time.After(5 * time.Second):
		t.Fatal("the wrapped check was not canceled when it overran")
	}
}

// Details and errors pass through untouched.
func TestTimeoutCheckPassesThrough(t *testing.T) {
	sentinel := errors.New("dependency unavailable")

	ok := NewTimeoutCheckFunc(
		func(ctx context.Context) (interface{}, error) { return "details", nil },
		time.Second)

	details, err := ok(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details != "details" {
		t.Fatalf("details not propagated, got %v", details)
	}

	failing := NewTimeoutCheckFunc(
		func(ctx context.Context) (interface{}, error) { return nil, sentinel },
		time.Second)

	if _, err := failing(context.Background()); !errors.Is(err, sentinel) {
		t.Fatalf("error not propagated, got %v", err)
	}
}

// A completed execution leaves nothing behind that would block the next.
func TestTimeoutCheckRepeatedRuns(t *testing.T) {
	var starts atomic.Int32
	check := NewTimeoutCheckFunc(
		func(ctx context.Context) (interface{}, error) {
			starts.Add(1)
			return nil, nil
		},
		time.Second)

	for i := 0; i < 20; i++ {
		if _, err := check(context.Background()); err != nil {
			t.Fatalf("execution %d failed: %v", i, err)
		}
	}
	if got := starts.Load(); got != 20 {
		t.Fatalf("expected 20 executions, got %d", got)
	}
}

// A completed execution must never be reported as still running. The
// bookkeeping is shared between the waiting caller and the goroutine it
// started, so hammer both paths together under -race.
func TestTimeoutCheckConcurrentExecutions(t *testing.T) {
	check := NewTimeoutCheckFunc(
		func(ctx context.Context) (interface{}, error) { return nil, nil },
		time.Second)

	var wg sync.WaitGroup
	errs := make(chan error, 200)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if _, err := check(context.Background()); err != nil {
					errs <- err
				}
			}
		}()
	}
	wg.Wait()
	close(errs)

	// Concurrent callers may legitimately collide with an execution that is
	// genuinely in flight, so failures are permitted - but only that one.
	for err := range errs {
		if !errors.Is(err, errInFlight) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

// Canceling the caller's context releases the caller, so a scheduler shutting
// down is not held up by a check that has stopped returning.
func TestTimeoutCheckHonorsCallerCancellation(t *testing.T) {
	release := make(chan struct{})
	defer close(release)

	check := NewTimeoutCheckFunc(
		func(ctx context.Context) (interface{}, error) {
			<-release
			return nil, nil
		},
		time.Hour) // long enough that only cancellation can end it

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := check(ctx)
		done <- err
	}()

	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("caller was not released when its context was canceled")
	}
}

// A result produced by an execution the caller already gave up on must never be
// handed to a later caller: health is a statement about now.
func TestTimeoutCheckDiscardsStaleResult(t *testing.T) {
	release := make(chan struct{})
	var n atomic.Int32

	check := NewTimeoutCheckFunc(
		func(ctx context.Context) (interface{}, error) {
			if n.Add(1) == 1 {
				<-release
				return "stale", nil
			}
			return "fresh", nil
		},
		20*time.Millisecond)

	if _, err := check(context.Background()); err == nil {
		t.Fatal("expected the first execution to overrun")
	}

	close(release)

	deadline := time.Now().Add(5 * time.Second)
	for {
		details, err := check(context.Background())
		if err == nil {
			if details != "fresh" {
				t.Fatalf("stale result leaked to a later caller: %v", details)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("check did not recover: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// A non-positive timeout falls back to the default. Asserted on the value
// rather than on the absence of a panic: a timeout of zero that stayed zero
// would fire on every execution.
func TestTimeoutCheckDefaults(t *testing.T) {
	nop := func(ctx context.Context) (interface{}, error) { return nil, nil }

	for _, timeout := range []time.Duration{0, -time.Second} {
		c := newTimeoutCheck(nop, timeout, time.Now)
		if c.timeout != DefaultTimeout {
			t.Fatalf("timeout %s: got %s, want %s", timeout, c.timeout, DefaultTimeout)
		}
		if _, err := c.execute(context.Background()); err != nil {
			t.Fatalf("timeout %s: unexpected error: %v", timeout, err)
		}
	}
}

// The goroutine must release the slot itself, rather than leaving it to
// whichever execute is waiting on the result.
//
// Deciding it from the result channel instead - a non-blocking receive in start
// - races with the execute waiting on that same channel: whichever arrives
// first consumes the value, and the other concludes the execution is still
// running, reporting a check that had just succeeded as in flight.
//
// The slot is observed with no execute consuming the result, which sequential
// calls to execute cannot do: they release it on the way out either way. This
// does not pin the ordering of the release against the send on done, which is
// buffered and so cannot be observed from outside.
func TestTimeoutCheckGoroutineReleasesSlot(t *testing.T) {
	ran := make(chan struct{})
	c := newTimeoutCheck(
		func(ctx context.Context) (interface{}, error) {
			close(ran)
			return nil, nil
		},
		time.Hour, time.Now)

	if _, err := c.start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	<-ran

	deadline := time.Now().Add(5 * time.Second)
	for {
		c.mu.Lock()
		outstanding := c.outstanding
		c.mu.Unlock()
		if outstanding == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the goroutine did not release the slot; only a waiting " +
				"execute does, which reports a completed check as still running")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// An execution that never returns holds the slot for the life of the process,
// so the check keeps failing from then on. This is deliberate - see
// NewTimeoutCheckFunc - and is pinned here because it is a behavior an
// operator will meet: recovery is by restart, not by the dependency coming
// back.
func TestTimeoutCheckStaysFailedWhileExecutionIsStuck(t *testing.T) {
	release := make(chan struct{})
	defer close(release)

	var starts atomic.Int32
	entered := make(chan struct{})

	check := NewTimeoutCheckFunc(
		func(ctx context.Context) (interface{}, error) {
			if starts.Add(1) == 1 {
				close(entered)
			}
			<-release // never observes cancellation
			return nil, nil
		},
		20*time.Millisecond)

	if _, err := check(context.Background()); !errors.Is(err, errTimeout) {
		t.Fatalf("expected a timeout, got %v", err)
	}
	<-entered

	// Every later execution keeps failing, and none of them starts the check
	// again, for as long as the stuck execution holds the slot.
	for i := 0; i < 10; i++ {
		_, err := check(context.Background())
		if !errors.Is(err, errInFlight) {
			t.Fatalf("execution %d: expected errInFlight, got %v", i, err)
		}
		time.Sleep(5 * time.Millisecond)
	}

	if got := starts.Load(); got != 1 {
		t.Fatalf("the stuck execution was restarted: %d executions", got)
	}
}
