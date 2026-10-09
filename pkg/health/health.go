// Package health bounds health checks that perform I/O, so that a check
// blocked on an unresponsive dependency reports a failure rather than nothing.
package health

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// DefaultTimeout bounds a single execution of a health check.
// Generous by design: it bounds a stall, it does not police latency. Keep it
// below the check's execution period.
const DefaultTimeout = 10 * time.Second

var (
	// errTimeout is returned when an execution overruns the timeout.
	errTimeout = errors.New("health check did not return within the timeout")

	// errInFlight is returned when an earlier execution has still
	// not returned, so this one was not started.
	errInFlight = errors.New("a previous health check execution is still running")
)

// NewTimeoutCheckFunc wraps a health check so that an execution which overruns
// is reported as a failure.
//
// A check that performs I/O is only as responsive as the dependency it probes.
// A blocked check reports nothing rather than an error, and schedulers record a
// result only when a check returns, so the last recorded result - a passing one
// - stands for the whole outage while the process reports itself healthy.
//
// Cancellation cannot fix this alone, being cooperative: a check may be blocked
// where it is not observed, such as on a connection pool, in a driver that takes
// no context, in a syscall or in cgo. The call is therefore made on its own
// goroutine, which the caller stops waiting on once it overruns.
//
// Returning early lets the scheduler proceed to the next tick, so executions are
// single-flight: while one is outstanding, later ones report the failure without
// starting another.
//
// An execution that never returns therefore holds the slot for the life of the
// process, and the check keeps failing even if the dependency recovers; recovery
// is then by restart, which the caller is responsible for arranging. One that
// returns late releases the slot and the check recovers by itself.
func NewTimeoutCheckFunc(
	check func(context.Context) (details interface{}, err error),
	timeout time.Duration,
) func(context.Context) (details interface{}, err error) {
	return newTimeoutCheck(check, timeout, time.Now).execute
}

// now is a parameter here, rather than on the exported constructor, because it
// only dates the in-flight message: the timeout itself runs on real time.
func newTimeoutCheck(
	check func(context.Context) (details interface{}, err error),
	timeout time.Duration,
	now func() time.Time,
) *timeoutCheck {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &timeoutCheck{check: check, timeout: timeout, now: now}
}

type timeoutCheck struct {
	check   func(context.Context) (interface{}, error)
	timeout time.Duration
	now     func() time.Time

	mu sync.Mutex
	// nil when no execution is in flight.
	outstanding *execution
}

type execution struct {
	started time.Time
	cancel  context.CancelFunc
	// Buffered, so an abandoned goroutine can still deliver and exit.
	done chan result
}

type result struct {
	details interface{}
	err     error
}

func (c *timeoutCheck) execute(ctx context.Context) (interface{}, error) {
	ex, err := c.start(ctx)
	if err != nil {
		return nil, err
	}

	timer := time.NewTimer(c.timeout)
	defer timer.Stop()

	select {
	case res := <-ex.done:
		ex.cancel()
		return res.details, res.err

	case <-timer.C:
		ex.cancel()
		return nil, fmt.Errorf("%w (%s)", errTimeout, c.timeout)
	case <-ctx.Done():
		ex.cancel()
		return nil, ctx.Err()
	}
}

// start begins an execution, unless a previous one is still outstanding.
func (c *timeoutCheck) start(parent context.Context) (*execution, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// The goroutine clears this itself on the way out, so a non-nil value here
	// means the previous execution genuinely has not returned. Deciding it from
	// the result channel instead would race with the execute that is waiting on
	// it, and report a check that had just succeeded as still running.
	if prev := c.outstanding; prev != nil {
		// Not truncated: collisions are routinely sub-second, and "0s" would
		// tell an operator nothing.
		return nil, fmt.Errorf("%w (started %s ago)",
			errInFlight, c.now().Sub(prev.started))
	}

	// Derived from the caller's context, so shutdown still propagates.
	ctx, cancel := context.WithCancel(parent)
	ex := &execution{
		started: c.now(),
		cancel:  cancel,
		done:    make(chan result, 1),
	}
	c.outstanding = ex

	go func() {
		details, err := c.check(ctx)

		c.mu.Lock()
		if c.outstanding == ex {
			c.outstanding = nil
		}
		c.mu.Unlock()

		ex.done <- result{details: details, err: err}
	}()

	return ex, nil
}
